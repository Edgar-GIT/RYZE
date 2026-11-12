// Package support implements the support chat between an account and RYZE
// support, answered by the Management Administrator (ADMIN_2).
//
// Rules:
//   - Trainers, and clients with an active entitlement or Free activation, may
//     use support. Eligibility is checked on every account operation, never
//     only in the interface.
//   - A conversation is always between one account and RYZE: it never
//     connects a client with a trainer and creates no coaching relationship.
//   - Messages are append-only. A client-chosen message id makes a resend
//     after a dropped connection idempotent.
//   - Staff identities are never shown to accounts; replies come from "RYZE
//     support".
package support

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"ryze/backend/email"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/notifications"
)

const (
	maxOpenConversations = 5
	maxBodyRunes         = 4000
	minSubjectRunes      = 3
	maxSubjectRunes      = 120
	messagePageSize      = 50
	streamBatchSize      = 100
	staffPageLimitMax    = 50
)

var (
	ErrNotEligible  = errors.New("support is available to accounts with an active programme and to trainers")
	ErrNotFound     = errors.New("conversation not found")
	ErrClosed       = errors.New("conversation is closed")
	ErrInvalidInput = errors.New("invalid input")
	ErrTooManyOpen  = errors.New("too many open conversations")
)

var clientMessageIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)

// Eligibility tells whether the account may use support, and as what.
type Eligibility struct {
	Eligible bool
	Role     string
}

// Thread is one conversation with a page of its messages (oldest first).
type Thread struct {
	Conversation models.SupportConversation
	Messages     []models.SupportMessage
	// HasOlder reports whether messages before the first one exist.
	HasOlder bool
	// Requester is filled for staff reads only.
	Requester *repositories.SupportConversationSummary
}

// StaffList is a page of conversations for support staff.
type StaffList struct {
	Conversations []repositories.SupportConversationSummary
	Total         int64
	Unread        int64
}

// Service is the support contract.
type Service interface {
	Eligibility(ctx context.Context, userID string) (Eligibility, error)
	ListMine(ctx context.Context, userID string) ([]repositories.SupportConversationSummary, error)
	Open(ctx context.Context, userID, subject, body, clientMessageID string) (*Thread, error)
	GetMine(ctx context.Context, userID, conversationID string, beforeSeq int64) (*Thread, error)
	SendMine(ctx context.Context, userID, conversationID, body, clientMessageID string) (*models.SupportMessage, error)
	MarkMineRead(ctx context.Context, userID, conversationID string, seq int64) error
	// NewMessagesForUser returns the account's messages after a sequence.
	NewMessagesForUser(ctx context.Context, userID string, afterSeq int64) ([]models.SupportMessage, error)

	ListStaff(ctx context.Context, status string, page, limit int) (StaffList, error)
	GetStaff(ctx context.Context, conversationID string, beforeSeq int64) (*Thread, error)
	Reply(ctx context.Context, adminID, conversationID, body, clientMessageID string) (*models.SupportMessage, error)
	MarkStaffRead(ctx context.Context, conversationID string, seq int64) error
	SetStatus(ctx context.Context, adminID, conversationID, status string) (*models.SupportConversation, error)
	NewMessagesForStaff(ctx context.Context, afterSeq int64) ([]models.SupportMessage, error)

	CurrentSeq(ctx context.Context) (int64, error)
	Hub() *Hub
}

type service struct {
	repo     repositories.SupportRepository
	notifier notifications.Notifier
	hub      *Hub
	now      func() time.Time
}

// New builds the service. notifier may be nil.
func New(repo repositories.SupportRepository, notifier notifications.Notifier) Service {
	return &service{repo: repo, notifier: notifier, hub: NewHub(), now: func() time.Time { return time.Now().UTC() }}
}

func (s *service) Hub() *Hub { return s.hub }

// cleanText removes control characters (keeping line breaks and tabs),
// normalizes line endings and trims the text.
func cleanText(value string, keepNewlines bool) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.Map(func(r rune) rune {
		switch {
		case r == '\n' && keepNewlines, r == '\t':
			return r
		case r == '\n', r == '\r':
			return ' '
		case unicode.IsControl(r), r == ' ', r == ' ':
			return -1
		default:
			return r
		}
	}, value)
	return strings.TrimSpace(value)
}

func validBody(body string) (string, error) {
	body = cleanText(body, true)
	if body == "" || utf8.RuneCountInString(body) > maxBodyRunes {
		return "", ErrInvalidInput
	}
	return body, nil
}

func optionalClientID(value string) (*string, error) {
	if value == "" {
		return nil, nil
	}
	if !clientMessageIDPattern.MatchString(value) {
		return nil, ErrInvalidInput
	}
	return &value, nil
}

func (s *service) Eligibility(ctx context.Context, userID string) (Eligibility, error) {
	result, err := s.repo.Eligibility(ctx, userID, s.now())
	if err != nil {
		return Eligibility{}, err
	}
	switch {
	case result.Trainer:
		return Eligibility{Eligible: true, Role: models.SupportRoleTrainer}, nil
	case result.Client:
		return Eligibility{Eligible: true, Role: models.SupportRoleClient}, nil
	default:
		return Eligibility{}, nil
	}
}

func (s *service) requireEligible(ctx context.Context, userID string) (Eligibility, error) {
	eligibility, err := s.Eligibility(ctx, userID)
	if err != nil {
		return Eligibility{}, err
	}
	if !eligibility.Eligible {
		return Eligibility{}, ErrNotEligible
	}
	return eligibility, nil
}

// owned loads a conversation only if the account owns it; anything else is
// reported as not found so ids of other accounts' conversations leak nothing.
func (s *service) owned(ctx context.Context, userID, conversationID string) (*models.SupportConversation, error) {
	conversation, err := s.repo.FindConversation(ctx, conversationID)
	if err != nil {
		if errors.Is(err, repositories.ErrSupportConversationNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if conversation.UserID != userID {
		return nil, ErrNotFound
	}
	return conversation, nil
}

func (s *service) ListMine(ctx context.Context, userID string) ([]repositories.SupportConversationSummary, error) {
	if _, err := s.requireEligible(ctx, userID); err != nil {
		return nil, err
	}
	return s.repo.ListForUser(ctx, userID)
}

func (s *service) Open(ctx context.Context, userID, subject, body, clientMessageID string) (*Thread, error) {
	eligibility, err := s.requireEligible(ctx, userID)
	if err != nil {
		return nil, err
	}
	subject = cleanText(subject, false)
	if count := utf8.RuneCountInString(subject); count < minSubjectRunes || count > maxSubjectRunes {
		return nil, ErrInvalidInput
	}
	cleanBody, err := validBody(body)
	if err != nil {
		return nil, err
	}
	clientID, err := optionalClientID(clientMessageID)
	if err != nil {
		return nil, err
	}
	// A retried "open" (same client message id) returns the conversation the
	// first attempt created instead of opening a second one.
	if clientID != nil {
		existing, err := s.repo.FindOpenedByClientID(ctx, userID, *clientID)
		if err == nil {
			return s.thread(ctx, existing, 0)
		}
		if !errors.Is(err, repositories.ErrSupportConversationNotFound) {
			return nil, err
		}
	}
	conversation := &models.SupportConversation{
		UserID:        userID,
		RequesterRole: eligibility.Role,
		Subject:       subject,
		Status:        models.SupportStatusOpen,
		LastMessageAt: s.now(),
	}
	author := userID
	first := &models.SupportMessage{AuthorKind: models.SupportAuthorUser, AuthorUserID: &author, Body: cleanBody, ClientMessageID: clientID}
	if err := s.repo.CreateConversation(ctx, conversation, first, maxOpenConversations); err != nil {
		return nil, mapRepoError(err)
	}
	s.hub.Publish(userID)
	return &Thread{Conversation: *conversation, Messages: []models.SupportMessage{*first}}, nil
}

func (s *service) thread(ctx context.Context, conversation *models.SupportConversation, beforeSeq int64) (*Thread, error) {
	messages, err := s.repo.Messages(ctx, conversation.ID, beforeSeq, messagePageSize+1)
	if err != nil {
		return nil, err
	}
	hasOlder := len(messages) > messagePageSize
	if hasOlder {
		messages = messages[1:]
	}
	return &Thread{Conversation: *conversation, Messages: messages, HasOlder: hasOlder}, nil
}

func (s *service) GetMine(ctx context.Context, userID, conversationID string, beforeSeq int64) (*Thread, error) {
	if _, err := s.requireEligible(ctx, userID); err != nil {
		return nil, err
	}
	conversation, err := s.owned(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}
	return s.thread(ctx, conversation, beforeSeq)
}

func (s *service) SendMine(ctx context.Context, userID, conversationID, body, clientMessageID string) (*models.SupportMessage, error) {
	if _, err := s.requireEligible(ctx, userID); err != nil {
		return nil, err
	}
	if _, err := s.owned(ctx, userID, conversationID); err != nil {
		return nil, err
	}
	cleanBody, err := validBody(body)
	if err != nil {
		return nil, err
	}
	clientID, err := optionalClientID(clientMessageID)
	if err != nil {
		return nil, err
	}
	author := userID
	message := &models.SupportMessage{ConversationID: conversationID, AuthorKind: models.SupportAuthorUser, AuthorUserID: &author, Body: cleanBody, ClientMessageID: clientID}
	created, err := s.repo.AddMessage(ctx, message)
	if err != nil {
		return nil, mapRepoError(err)
	}
	if created {
		s.hub.Publish(userID)
	}
	return message, nil
}

func mapRepoError(err error) error {
	switch {
	case errors.Is(err, repositories.ErrSupportConversationNotFound):
		return ErrNotFound
	case errors.Is(err, repositories.ErrSupportConversationClosed):
		return ErrClosed
	case errors.Is(err, repositories.ErrSupportTooManyOpen):
		return ErrTooManyOpen
	case errors.Is(err, repositories.ErrSupportClientIDTaken):
		return ErrInvalidInput
	default:
		return err
	}
}

func (s *service) MarkMineRead(ctx context.Context, userID, conversationID string, seq int64) error {
	if _, err := s.requireEligible(ctx, userID); err != nil {
		return err
	}
	if _, err := s.owned(ctx, userID, conversationID); err != nil {
		return err
	}
	return mapRepoError(s.repo.MarkRead(ctx, conversationID, false, seq))
}

func (s *service) NewMessagesForUser(ctx context.Context, userID string, afterSeq int64) ([]models.SupportMessage, error) {
	return s.repo.MessagesSince(ctx, userID, afterSeq, streamBatchSize)
}

func (s *service) ListStaff(ctx context.Context, status string, page, limit int) (StaffList, error) {
	if status != "" && status != models.SupportStatusOpen && status != models.SupportStatusClosed {
		return StaffList{}, ErrInvalidInput
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > staffPageLimitMax {
		limit = 20
	}
	rows, total, err := s.repo.ListForStaff(ctx, status, page, limit)
	if err != nil {
		return StaffList{}, err
	}
	unread, err := s.repo.UnreadForStaff(ctx)
	if err != nil {
		return StaffList{}, err
	}
	return StaffList{Conversations: rows, Total: total, Unread: unread}, nil
}

func (s *service) GetStaff(ctx context.Context, conversationID string, beforeSeq int64) (*Thread, error) {
	conversation, err := s.repo.FindConversation(ctx, conversationID)
	if err != nil {
		return nil, mapRepoError(err)
	}
	thread, err := s.thread(ctx, conversation, beforeSeq)
	if err != nil {
		return nil, err
	}
	// The requester's name and email come from the list projection.
	rows, err := s.repo.ListForUser(ctx, conversation.UserID)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if rows[i].ID == conversation.ID {
			thread.Requester = &rows[i]
			break
		}
	}
	return thread, nil
}

func (s *service) Reply(ctx context.Context, adminID, conversationID, body, clientMessageID string) (*models.SupportMessage, error) {
	if adminID == "" {
		return nil, ErrInvalidInput
	}
	conversation, err := s.repo.FindConversation(ctx, conversationID)
	if err != nil {
		return nil, mapRepoError(err)
	}
	cleanBody, err := validBody(body)
	if err != nil {
		return nil, err
	}
	clientID, err := optionalClientID(clientMessageID)
	if err != nil {
		return nil, err
	}
	// Email only the first reply the account has not read yet, so a quick
	// back-and-forth does not flood the inbox.
	unreadBefore, err := s.repo.UnreadStaffMessages(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	author := adminID
	message := &models.SupportMessage{ConversationID: conversationID, AuthorKind: models.SupportAuthorStaff, AuthorAdminID: &author, Body: cleanBody, ClientMessageID: clientID}
	created, err := s.repo.AddMessage(ctx, message)
	if err != nil {
		return nil, mapRepoError(err)
	}
	if !created {
		return message, nil
	}
	s.hub.Publish(conversation.UserID)
	if s.notifier != nil {
		event := notifications.Event{
			UserID:    conversation.UserID,
			Kind:      models.NotificationSupportReply,
			Link:      "/support?conversation=" + conversation.ID,
			DedupeKey: "support_reply:" + message.ID,
		}
		if unreadBefore == 0 {
			event.EmailTemplate = email.TemplateSupportReply
		}
		s.notifier.Notify(ctx, event)
	}
	return message, nil
}

func (s *service) MarkStaffRead(ctx context.Context, conversationID string, seq int64) error {
	return mapRepoError(s.repo.MarkRead(ctx, conversationID, true, seq))
}

func (s *service) SetStatus(ctx context.Context, adminID, conversationID, status string) (*models.SupportConversation, error) {
	if status != models.SupportStatusOpen && status != models.SupportStatusClosed {
		return nil, ErrInvalidInput
	}
	conversation, err := s.repo.SetStatus(ctx, conversationID, status, adminID, s.now())
	if err != nil {
		return nil, mapRepoError(err)
	}
	return conversation, nil
}

func (s *service) NewMessagesForStaff(ctx context.Context, afterSeq int64) ([]models.SupportMessage, error) {
	return s.repo.MessagesSince(ctx, "", afterSeq, streamBatchSize)
}

func (s *service) CurrentSeq(ctx context.Context) (int64, error) {
	return s.repo.MaxSeq(ctx)
}
