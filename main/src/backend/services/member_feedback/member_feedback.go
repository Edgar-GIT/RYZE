// Package member_feedback implements member feedback: a star rating and a
// short message from members who hold (or held) a plan, published only after
// an administrator approves it.
package member_feedback

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

const (
	MinRating = 1
	MaxRating = 5
	// MinMessageLength and MaxMessageLength bound the trimmed message, counted
	// in characters.
	MinMessageLength = 10
	MaxMessageLength = 1000
	// DefaultPageSize and MaxPageSize bound every list.
	DefaultPageSize = 12
	MaxPageSize     = 50
)

var (
	// ErrInvalidInput indicates malformed or out-of-range input.
	ErrInvalidInput = errors.New("invalid member feedback input")
	// ErrUserNotFound indicates the caller's account does not exist or is not active.
	ErrUserNotFound = errors.New("user not found")
	// ErrNotEligible indicates the member has never held a plan.
	ErrNotEligible = errors.New("member has no plan")
	// ErrAlreadyPending indicates the member already has a submission awaiting review.
	ErrAlreadyPending = errors.New("member already has pending feedback")
	// ErrNotFound indicates the requested feedback does not exist.
	ErrNotFound = errors.New("member feedback not found")
	// ErrStateConflict indicates the feedback was already reviewed.
	ErrStateConflict = errors.New("member feedback is not pending")
)

// Repository is the data-access surface required by the service.
type Repository interface {
	Create(ctx context.Context, feedback *models.MemberFeedback) error
	FindPendingByUserID(ctx context.Context, userID string) (*models.MemberFeedback, error)
	HasHeldPlan(ctx context.Context, userID string) (bool, error)
	ListPublished(ctx context.Context, page, limit int) ([]models.MemberFeedback, int64, error)
	ListByStatus(ctx context.Context, status string, page, limit int) ([]models.MemberFeedback, int64, error)
	Review(ctx context.Context, id, status, reviewer string, at time.Time) (*models.MemberFeedback, error)
}

// UserRepository resolves the caller's account (soft-deleted accounts are not found).
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*models.User, error)
}

// Eligibility tells the caller whether they can leave feedback now.
type Eligibility struct {
	// Eligible is true when the member holds or held a plan.
	Eligible bool
	// Pending is the member's submission awaiting review, if any. A member with
	// pending feedback cannot submit another until it is reviewed.
	Pending *models.MemberFeedback
}

// SubmitInput is the member's rating and message.
type SubmitInput struct {
	Rating  int
	Message string
}

// ListResult carries one page of feedback and its pagination metadata.
type ListResult struct {
	Feedback []models.MemberFeedback
	Total    int64
	Page     int
	Limit    int
}

// Decision is the administrator's review outcome.
type Decision string

const (
	DecisionApprove Decision = "approve"
	DecisionReject  Decision = "reject"
)

// Service implements member feedback. Authentication and administrator
// authorization are enforced by the route middleware; the service derives
// identity only from the IDs it is given by the handlers, which read them from
// the authenticated context.
type Service interface {
	GetEligibility(ctx context.Context, userID string) (Eligibility, error)
	Submit(ctx context.Context, userID string, input SubmitInput) (*models.MemberFeedback, error)
	ListPublished(ctx context.Context, page, limit int) (ListResult, error)
	ListForReview(ctx context.Context, status string, page, limit int) (ListResult, error)
	Review(ctx context.Context, id, adminID string, decision Decision) (*models.MemberFeedback, error)
}

type service struct {
	feedback Repository
	users    UserRepository
	now      func() time.Time
}

func NewService(feedback Repository, users UserRepository) Service {
	return &service{feedback: feedback, users: users, now: func() time.Time { return time.Now().UTC() }}
}

func (s *service) activeUser(ctx context.Context, userID string) (*models.User, error) {
	if _, err := uuid.Parse(userID); err != nil {
		return nil, ErrInvalidInput
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to load user: %w", err)
	}
	return user, nil
}

func (s *service) GetEligibility(ctx context.Context, userID string) (Eligibility, error) {
	if _, err := s.activeUser(ctx, userID); err != nil {
		return Eligibility{}, err
	}
	held, err := s.feedback.HasHeldPlan(ctx, userID)
	if err != nil {
		return Eligibility{}, fmt.Errorf("failed to check plans: %w", err)
	}
	result := Eligibility{Eligible: held}
	pending, err := s.feedback.FindPendingByUserID(ctx, userID)
	switch {
	case err == nil:
		result.Pending = pending
	case errors.Is(err, repositories.ErrMemberFeedbackNotFound):
	default:
		return Eligibility{}, fmt.Errorf("failed to check pending feedback: %w", err)
	}
	return result, nil
}

func (s *service) Submit(ctx context.Context, userID string, input SubmitInput) (*models.MemberFeedback, error) {
	message, err := normalizeMessage(input.Message)
	if err != nil {
		return nil, err
	}
	if input.Rating < MinRating || input.Rating > MaxRating {
		return nil, ErrInvalidInput
	}
	user, err := s.activeUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	held, err := s.feedback.HasHeldPlan(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check plans: %w", err)
	}
	if !held {
		return nil, ErrNotEligible
	}

	feedback := &models.MemberFeedback{
		UserID:      userID,
		Rating:      input.Rating,
		Message:     message,
		DisplayName: PublicName(user.FirstName, user.LastName),
		Status:      models.MemberFeedbackStatusPending,
	}
	if err := s.feedback.Create(ctx, feedback); err != nil {
		if errors.Is(err, repositories.ErrMemberFeedbackAlreadyPending) {
			return nil, ErrAlreadyPending
		}
		return nil, fmt.Errorf("failed to save feedback: %w", err)
	}
	return feedback, nil
}

func (s *service) ListPublished(ctx context.Context, page, limit int) (ListResult, error) {
	page, limit, err := pagination(page, limit)
	if err != nil {
		return ListResult{}, err
	}
	rows, total, err := s.feedback.ListPublished(ctx, page, limit)
	if err != nil {
		return ListResult{}, fmt.Errorf("failed to list feedback: %w", err)
	}
	return ListResult{Feedback: rows, Total: total, Page: page, Limit: limit}, nil
}

func (s *service) ListForReview(ctx context.Context, status string, page, limit int) (ListResult, error) {
	switch status {
	case "", models.MemberFeedbackStatusPending, models.MemberFeedbackStatusApproved, models.MemberFeedbackStatusRejected:
	default:
		return ListResult{}, ErrInvalidInput
	}
	page, limit, err := pagination(page, limit)
	if err != nil {
		return ListResult{}, err
	}
	rows, total, err := s.feedback.ListByStatus(ctx, status, page, limit)
	if err != nil {
		return ListResult{}, fmt.Errorf("failed to list feedback: %w", err)
	}
	return ListResult{Feedback: rows, Total: total, Page: page, Limit: limit}, nil
}

func (s *service) Review(ctx context.Context, id, adminID string, decision Decision) (*models.MemberFeedback, error) {
	if _, err := uuid.Parse(id); err != nil || strings.TrimSpace(adminID) == "" {
		return nil, ErrInvalidInput
	}
	var status string
	switch decision {
	case DecisionApprove:
		status = models.MemberFeedbackStatusApproved
	case DecisionReject:
		status = models.MemberFeedbackStatusRejected
	default:
		return nil, ErrInvalidInput
	}
	reviewed, err := s.feedback.Review(ctx, id, status, adminID, s.now())
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrMemberFeedbackNotFound):
			return nil, ErrNotFound
		case errors.Is(err, repositories.ErrMemberFeedbackStateConflict):
			return nil, ErrStateConflict
		default:
			return nil, fmt.Errorf("failed to review feedback: %w", err)
		}
	}
	return reviewed, nil
}

// PublicName is the only name shown publicly: the first name and the initial
// of the last name ("Alex D."), never the email or the full surname.
func PublicName(firstName, lastName string) string {
	first := strings.TrimSpace(firstName)
	last := strings.TrimSpace(lastName)
	switch {
	case first == "" && last == "":
		return "RYZE member"
	case last == "":
		return first
	case first == "":
		initial, _ := utf8.DecodeRuneInString(last)
		return string(unicode.ToUpper(initial)) + "."
	default:
		initial, _ := utf8.DecodeRuneInString(last)
		return first + " " + string(unicode.ToUpper(initial)) + "."
	}
}

// normalizeMessage trims the message, removes control characters other than
// line breaks and enforces the length bounds.
func normalizeMessage(raw string) (string, error) {
	if !utf8.ValidString(raw) {
		return "", ErrInvalidInput
	}
	cleaned := strings.Map(func(r rune) rune {
		if r == '\n' {
			return r
		}
		if r == '\r' || r == '\t' {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, raw)
	cleaned = strings.TrimSpace(cleaned)
	length := utf8.RuneCountInString(cleaned)
	if length < MinMessageLength || length > MaxMessageLength {
		return "", ErrInvalidInput
	}
	return cleaned, nil
}

func pagination(page, limit int) (int, int, error) {
	if page < 1 || limit < 0 {
		return 0, 0, ErrInvalidInput
	}
	if limit == 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	return page, limit, nil
}
