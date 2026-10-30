package contact

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/mail"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

var (
	// ErrInvalidInput indicates a contact submission that failed validation.
	ErrInvalidInput = errors.New("invalid contact input")
	// ErrUnavailable indicates that no delivery channel is configured or that
	// the configured channel refused the message. The caller must not report a
	// successful submission in this case.
	ErrUnavailable = errors.New("contact delivery unavailable")
)

const (
	// MaxAttachmentBytes bounds the optional attachment independently of the
	// request body limit. The route allows multipart overhead on top of it.
	MaxAttachmentBytes = 1 << 20
	MaxNameLength      = 100
	MaxEmailLength     = 254
	MaxSubjectLength   = 200
	MaxMessageLength   = 5000
	MaxFilenameLength  = 255
)

var allowedAttachmentTypes = map[string]bool{
	"image/png":       true,
	"image/jpeg":      true,
	"image/webp":      true,
	"application/pdf": true,
}

// Attachment is an optional file supplied with a contact message. It is kept
// in memory only for the delivery attempt and is never persisted by RYZE.
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// Message is a validated, normalized contact submission.
type Message struct {
	Name       string
	Email      string
	Subject    string
	Body       string
	Attachment *Attachment
}

// Notifier delivers a validated contact message. Implementations must return
// an error when delivery did not happen.
type Notifier interface {
	Send(ctx context.Context, message Message) error
}

// Service validates and delivers contact submissions.
type Service interface {
	Submit(ctx context.Context, message Message) error
}

type service struct {
	notifier Notifier
}

// NewService builds a contact service. A nil notifier leaves delivery
// unavailable rather than silently accepting messages.
func NewService(notifier Notifier) Service {
	return &service{notifier: notifier}
}

// Submit validates the message and delivers it. Validation errors are
// distinguishable so the HTTP layer can return a helpful 400; delivery errors
// map to ErrUnavailable and never claim success.
func (s *service) Submit(ctx context.Context, message Message) error {
	normalized, err := normalize(message)
	if err != nil {
		return err
	}
	if s.notifier == nil {
		return ErrUnavailable
	}
	if err := s.notifier.Send(ctx, normalized); err != nil {
		return fmt.Errorf("%w: delivery failed", ErrUnavailable)
	}
	return nil
}

func normalize(message Message) (Message, error) {
	name := strings.TrimSpace(message.Name)
	if name == "" || utf8.RuneCountInString(name) > MaxNameLength || hasHeaderControl(name) {
		return Message{}, fmt.Errorf("%w: invalid name", ErrInvalidInput)
	}

	email := strings.TrimSpace(message.Email)
	if len(email) > MaxEmailLength || hasHeaderControl(email) {
		return Message{}, fmt.Errorf("%w: invalid email", ErrInvalidInput)
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return Message{}, fmt.Errorf("%w: invalid email", ErrInvalidInput)
	}

	subject := strings.TrimSpace(message.Subject)
	if subject == "" || utf8.RuneCountInString(subject) > MaxSubjectLength || hasHeaderControl(subject) {
		return Message{}, fmt.Errorf("%w: invalid subject", ErrInvalidInput)
	}

	body := strings.TrimSpace(message.Body)
	if body == "" || utf8.RuneCountInString(body) > MaxMessageLength {
		return Message{}, fmt.Errorf("%w: invalid message", ErrInvalidInput)
	}

	var attachment *Attachment
	if message.Attachment != nil {
		if len(message.Attachment.Data) == 0 || len(message.Attachment.Data) > MaxAttachmentBytes {
			return Message{}, fmt.Errorf("%w: invalid attachment size", ErrInvalidInput)
		}
		filename := filepath.Base(strings.TrimSpace(message.Attachment.Filename))
		if filename == "" || filename == "." || filename == ".." || utf8.RuneCountInString(filename) > MaxFilenameLength || hasHeaderControl(filename) {
			return Message{}, fmt.Errorf("%w: invalid attachment name", ErrInvalidInput)
		}
		detected := http.DetectContentType(message.Attachment.Data)
		if !allowedAttachmentTypes[detected] {
			return Message{}, fmt.Errorf("%w: unsupported attachment type", ErrInvalidInput)
		}
		attachment = &Attachment{
			Filename:    filename,
			ContentType: detected,
			Data:        message.Attachment.Data,
		}
	}

	return Message{
		Name:       name,
		Email:      email,
		Subject:    subject,
		Body:       body,
		Attachment: attachment,
	}, nil
}

func hasHeaderControl(value string) bool {
	return strings.ContainsAny(value, "\r\n\x00")
}

// ParseContentType is kept narrow so callers cannot accidentally accept a
// client-supplied MIME type instead of the detected one.
func ParseContentType(value string) string {
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return ""
	}
	return mediaType
}
