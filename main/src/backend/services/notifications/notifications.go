// Package notifications records temporary in-app notifications (visible for
// 24 hours) and, for the events that warrant one, the matching email.
//
// Notifications are created only at real state transitions (a purchase
// completing, an intake being submitted, a trainer being assigned, a service
// being delivered, a support reply). Each event carries a dedupe key, so a
// repeated delivery of the same event (a webhook redelivery, a retried
// request) notifies once; the email outbox applies the same key.
//
// Notify never fails its caller: the business operation it reports on is
// already committed, so a failure here is logged and nothing more.
package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"ryze/backend/email"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

const listLimit = 50

// ErrNotFound covers unknown, foreign and expired notifications alike.
var ErrNotFound = errors.New("notification not found")

// Event is one notification to record.
type Event struct {
	UserID string
	Kind   string
	// Program is a display name shown in the text (optional).
	Program string
	// Link is a relative path inside the web app (optional).
	Link string
	// DedupeKey identifies the business event; required.
	DedupeKey string
	// EmailTemplate also sends this email when set (never to Test Mode
	// personas, whose addresses are not real mailboxes).
	EmailTemplate string
}

// Notifier records notifications.
type Notifier interface {
	Notify(ctx context.Context, event Event)
}

// Users resolves the recipient of an email.
type Users interface {
	FindByID(ctx context.Context, id string) (*models.User, error)
}

// Programs resolves programme names.
type Programs interface {
	FindByID(ctx context.Context, programID string) (*models.Program, error)
}

// List is the account's current notifications.
type List struct {
	Notifications []models.Notification
	Unread        int64
}

// Service is the notifications contract.
type Service interface {
	Notifier
	List(ctx context.Context, userID string) (List, error)
	MarkRead(ctx context.Context, userID, id string) error
	MarkAllRead(ctx context.Context, userID string) error
	// Purge applies the retention policy (see migration 000039).
	Purge(ctx context.Context) error
	// FulfillPurchase notifies the buyer that access is ready. It is a
	// purchase fulfilment hook: it runs on every delivery of a completed
	// purchase and notifies once per purchase.
	FulfillPurchase(ctx context.Context, userID, programID string) error
}

type service struct {
	repo     repositories.NotificationRepository
	users    Users
	programs Programs
	mailer   email.Mailer
	now      func() time.Time
}

// New builds the service. mailer may be nil (in-app only).
func New(repo repositories.NotificationRepository, users Users, programs Programs, mailer email.Mailer) Service {
	return &service{repo: repo, users: users, programs: programs, mailer: mailer, now: func() time.Time { return time.Now().UTC() }}
}

// safeLink keeps only an in-app absolute path.
func safeLink(link string) string {
	if !strings.HasPrefix(link, "/") || strings.HasPrefix(link, "//") || len(link) > 255 || strings.ContainsAny(link, "\\\r\n") {
		return ""
	}
	return link
}

func truncate(value string, limit int) string {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	return string([]rune(value)[:limit])
}

func (s *service) Notify(ctx context.Context, event Event) {
	if event.UserID == "" || event.Kind == "" || event.DedupeKey == "" || len(event.DedupeKey) > 160 {
		slog.Error("notification rejected: incomplete event", slog.String("kind", event.Kind))
		return
	}
	params := map[string]string{}
	if program := truncate(event.Program, models.NotificationMaxParamValueLength); program != "" {
		params["program"] = program
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		slog.Error("notification params not encoded", slog.String("kind", event.Kind))
		return
	}
	now := s.now()
	notification := &models.Notification{
		UserID:    event.UserID,
		Kind:      event.Kind,
		Params:    encoded,
		DedupeKey: &event.DedupeKey,
		CreatedAt: now,
		ExpiresAt: now.Add(models.NotificationLifetime),
	}
	if link := safeLink(event.Link); link != "" {
		notification.Link = &link
	}
	inserted, err := s.repo.Create(ctx, notification)
	if err != nil {
		slog.Error("notification not recorded", slog.String("kind", event.Kind), slog.String("user_id", event.UserID), slog.String("error", err.Error()))
		return
	}
	if !inserted || event.EmailTemplate == "" || s.mailer == nil {
		return
	}
	user, err := s.users.FindByID(ctx, event.UserID)
	if err != nil {
		if !errors.Is(err, repositories.ErrUserNotFound) {
			slog.Error("notification email recipient not loaded", slog.String("user_id", event.UserID), slog.String("error", err.Error()))
		}
		return
	}
	if user.TestPersona != nil {
		return
	}
	locale := email.LocaleEnglish
	if user.PreferredLanguage != nil {
		locale = *user.PreferredLanguage
	}
	if err := s.mailer.Enqueue(ctx, email.Request{
		UserID:    user.ID,
		To:        user.Email,
		Template:  event.EmailTemplate,
		Locale:    locale,
		Name:      email.GreetingName(user),
		Program:   truncate(event.Program, models.NotificationMaxParamValueLength),
		Path:      safeLink(event.Link),
		DedupeKey: "notification:" + event.DedupeKey,
	}); err != nil {
		slog.Error("notification email not recorded", slog.String("kind", event.Kind), slog.String("user_id", user.ID), slog.String("error", err.Error()))
	}
}

func (s *service) List(ctx context.Context, userID string) (List, error) {
	notifications, unread, err := s.repo.ListVisible(ctx, userID, s.now(), listLimit)
	if err != nil {
		return List{}, err
	}
	return List{Notifications: notifications, Unread: unread}, nil
}

func (s *service) MarkRead(ctx context.Context, userID, id string) error {
	if err := s.repo.MarkRead(ctx, userID, id, s.now()); err != nil {
		if errors.Is(err, repositories.ErrNotificationNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *service) MarkAllRead(ctx context.Context, userID string) error {
	_, err := s.repo.MarkAllRead(ctx, userID, s.now())
	return err
}

func (s *service) Purge(ctx context.Context) error {
	now := s.now()
	_, _, err := s.repo.Purge(ctx, now, now.Add(-models.NotificationTombstoneRetention))
	return err
}

func (s *service) FulfillPurchase(ctx context.Context, userID, programID string) error {
	purchase, err := s.repo.LatestCompletedPurchase(ctx, userID, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrPurchaseNotFound) {
			return nil
		}
		return err
	}
	program, err := s.programs.FindByID(ctx, programID)
	if err != nil {
		return err
	}
	link := "/services/my-programs"
	if program.ProductType == models.ProgramProductTypePremiumLevel2 {
		// The buyer's next step is the intake.
		link = "/services/premium-level-2"
	}
	event := Event{
		UserID:        userID,
		Kind:          models.NotificationAccessGranted,
		Program:       program.Name,
		Link:          link,
		DedupeKey:     "access_granted:" + purchase.ID,
		EmailTemplate: email.TemplateAccessGranted,
	}
	if purchase.Test {
		event.EmailTemplate = ""
	}
	s.Notify(ctx, event)
	return nil
}
