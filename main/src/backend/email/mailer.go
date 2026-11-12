package email

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"
	"unicode"

	"ryze/backend/config"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

// Request describes one email a business event wants to send.
type Request struct {
	// UserID is the recipient account, when the email concerns one.
	UserID string
	// To is the recipient address.
	To string
	// Template is one of the Template* identifiers.
	Template string
	// Locale is the recipient's preferred language ("en"/"pt").
	Locale string
	// Name is the recipient's display name.
	Name string
	// Program is the product name the email refers to, when any.
	Program string
	// Path is the app path (and optional fragment) the email links to, for
	// example "/services/my-programs". It is joined to PUBLIC_APP_URL.
	Path string
	// DedupeKey makes the email unique per business event. Empty means no
	// deduplication.
	DedupeKey string
	// ValidFor bounds how long the email may still be delivered (for example
	// a reset link that expires). Zero means no bound.
	ValidFor time.Duration
}

// Mailer records transactional emails in the outbox. It never sends inline,
// so a slow relay cannot block or fail the request that triggered the email.
type Mailer interface {
	Enqueue(ctx context.Context, request Request) error
	Enabled() bool
}

type mailer struct {
	config config.MailConfig
	outbox repositories.EmailOutboxRepository
}

func NewMailer(cfg config.MailConfig, outbox repositories.EmailOutboxRepository) Mailer {
	return &mailer{config: cfg, outbox: outbox}
}

func (m *mailer) Enabled() bool {
	return m.config.Enabled()
}

func (m *mailer) Enqueue(ctx context.Context, request Request) error {
	if _, ok := copies[request.Template]; !ok {
		return fmt.Errorf("unknown email template %q", request.Template)
	}
	data := Data{Name: plainText(request.Name, 120), Program: plainText(request.Program, 160)}
	if request.Path != "" && m.config.PublicAppURL != "" {
		link, err := joinAppLink(m.config.PublicAppURL, request.Path)
		if err != nil {
			return err
		}
		data.Link = link
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("encode email payload: %w", err)
	}

	now := time.Now().UTC()
	record := &models.EmailOutbox{
		Recipient:     strings.TrimSpace(request.To),
		Template:      request.Template,
		Locale:        NormalizeLocale(request.Locale),
		Payload:       payload,
		Status:        models.EmailStatusPending,
		NextAttemptAt: now,
	}
	if request.UserID != "" {
		userID := request.UserID
		record.UserID = &userID
	}
	if request.DedupeKey != "" {
		key := request.DedupeKey
		record.DedupeKey = &key
	}
	if request.ValidFor > 0 {
		expires := now.Add(request.ValidFor)
		record.ExpiresAt = &expires
	}
	if !m.config.Enabled() {
		// Recorded as skipped (with an empty payload) so the event stays
		// observable without keeping a link nobody will deliver.
		record.Status = models.EmailStatusSkipped
		record.Payload = json.RawMessage(`{}`)
	}

	created, err := m.outbox.Enqueue(ctx, record)
	if err != nil {
		return err
	}
	if !created {
		slog.Info("email already recorded for this event", slog.String("template", request.Template))
	}
	return nil
}

// joinAppLink resolves an app path against the public origin. Only relative
// paths are accepted, so an email can never link outside the RYZE app.
func joinAppLink(origin, path string) (string, error) {
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return "", fmt.Errorf("email link must be an app path, got %q", path)
	}
	base, err := url.Parse(origin)
	if err != nil {
		return "", fmt.Errorf("invalid public app url")
	}
	relative, err := url.Parse(path)
	if err != nil || relative.Scheme != "" || relative.Host != "" {
		return "", fmt.Errorf("invalid email link path")
	}
	return base.ResolveReference(relative).String(), nil
}

// plainText removes control characters and bounds the length of a value that
// is placed in an email subject or body.
func plainText(value string, max int) string {
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if len([]rune(cleaned)) > max {
		cleaned = string([]rune(cleaned)[:max])
	}
	return cleaned
}
