// Package account_recovery implements self-service password recovery, email
// verification and the account's language preference.
//
// Tokens are 32 random bytes, sent only inside an email link fragment
// ("#token=..."), so they never reach server or proxy logs; only their SHA-256
// is stored. A password reset revokes every session. Requests never reveal
// whether an email address has an account.
package account_recovery

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"ryze/backend/email"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/password"
)

const (
	resetTokenTTL        = 60 * time.Minute
	verificationTokenTTL = 48 * time.Hour
	// A new link is not issued while a recent one is still fresh, and at most
	// a few per hour, so the feature cannot be used to flood an inbox.
	minIssueInterval = 2 * time.Minute
	maxIssuedPerHour = 5
)

var (
	// ErrInvalidInput covers a malformed email, language or token.
	ErrInvalidInput = errors.New("invalid input")
	// ErrInvalidToken covers unknown, used and expired links alike.
	ErrInvalidToken = errors.New("invalid or expired link")
	// ErrAlreadyVerified is returned when a verified account asks for a link.
	ErrAlreadyVerified = errors.New("email already verified")
	// ErrUserNotFound is returned for authenticated calls whose account vanished.
	ErrUserNotFound = errors.New("user not found")
)

// Service is the account recovery contract.
type Service interface {
	// RequestPasswordReset issues a reset link when the address belongs to an
	// active account. It reports success either way.
	RequestPasswordReset(ctx context.Context, address, locale string) error
	// ResetPassword sets a new password from a valid reset token.
	ResetPassword(ctx context.Context, token, newPassword string) error
	// SendVerification issues an email verification link for the account.
	SendVerification(ctx context.Context, userID string) error
	// VerifyEmail consumes a verification token.
	VerifyEmail(ctx context.Context, token string) error
	// SetLanguage stores the language RYZE uses for the account's emails.
	SetLanguage(ctx context.Context, userID, language string) error
}

// Users is the subset of the user repository the service needs.
type Users interface {
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	SetPreferredLanguage(ctx context.Context, id, language string) error
}

type service struct {
	users  Users
	tokens repositories.UserTokenRepository
	mailer email.Mailer
	now    func() time.Time
}

func New(users Users, tokens repositories.UserTokenRepository, mailer email.Mailer) Service {
	return &service{users: users, tokens: tokens, mailer: mailer, now: func() time.Time { return time.Now().UTC() }}
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func newToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// validTokenShape rejects obviously malformed tokens before any lookup.
func validTokenShape(token string) bool {
	if len(token) != 43 {
		return false
	}
	_, err := base64.RawURLEncoding.DecodeString(token)
	return err == nil
}

func localeOf(user *models.User, fallback string) string {
	if user.PreferredLanguage != nil {
		return *user.PreferredLanguage
	}
	return email.NormalizeLocale(fallback)
}

// issue creates a token for the purpose unless the account is being throttled.
// It reports whether a token was issued.
func (s *service) issue(ctx context.Context, userID, purpose string, ttl time.Duration) (string, *models.UserToken, error) {
	now := s.now()
	fresh, err := s.tokens.CountRecent(ctx, userID, purpose, now.Add(-minIssueInterval))
	if err != nil {
		return "", nil, err
	}
	hourly, err := s.tokens.CountRecent(ctx, userID, purpose, now.Add(-time.Hour))
	if err != nil {
		return "", nil, err
	}
	if fresh > 0 || hourly >= maxIssuedPerHour {
		return "", nil, nil
	}
	raw, err := newToken()
	if err != nil {
		return "", nil, err
	}
	token := &models.UserToken{
		UserID:    userID,
		Purpose:   purpose,
		TokenHash: hashToken(raw),
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}
	if err := s.tokens.Create(ctx, token); err != nil {
		return "", nil, err
	}
	return raw, token, nil
}

func (s *service) RequestPasswordReset(ctx context.Context, address, locale string) error {
	address = strings.TrimSpace(address)
	if parsed, err := mail.ParseAddress(address); err != nil || parsed.Address != address || len(address) > 255 {
		return ErrInvalidInput
	}
	user, err := s.users.FindByEmail(ctx, address)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil
		}
		return err
	}
	raw, token, err := s.issue(ctx, user.ID, models.UserTokenPurposePasswordReset, resetTokenTTL)
	if err != nil || token == nil {
		return err
	}
	return s.mailer.Enqueue(ctx, email.Request{
		UserID:    user.ID,
		To:        user.Email,
		Template:  email.TemplatePasswordReset,
		Locale:    localeOf(user, locale),
		Name:      email.GreetingName(user),
		Path:      "/reset-password#token=" + raw,
		DedupeKey: "password_reset:" + token.ID,
		ValidFor:  resetTokenTTL,
	})
}

func (s *service) ResetPassword(ctx context.Context, token, newPassword string) error {
	if !validTokenShape(token) {
		return ErrInvalidToken
	}
	if err := password.ValidatePolicy(newPassword); err != nil {
		return err
	}
	hash, err := password.HashPassword(newPassword)
	if err != nil {
		return err
	}
	user, err := s.tokens.ConsumePasswordReset(ctx, hashToken(token), hash, s.now())
	if err != nil {
		if errors.Is(err, repositories.ErrUserTokenInvalid) {
			return ErrInvalidToken
		}
		return err
	}
	if err := s.mailer.Enqueue(ctx, email.Request{
		UserID:    user.ID,
		To:        user.Email,
		Template:  email.TemplatePasswordChanged,
		Locale:    localeOf(user, email.LocaleEnglish),
		Name:      email.GreetingName(user),
		DedupeKey: "password_changed:" + hashToken(token)[:32],
	}); err != nil {
		// The password is already changed; a missing notice must not undo it.
		slog.Error("password change notice not recorded", slog.String("user_id", user.ID), slog.String("error", err.Error()))
	}
	return nil
}

func (s *service) SendVerification(ctx context.Context, userID string) error {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if user.EmailVerifiedAt != nil {
		return ErrAlreadyVerified
	}
	raw, token, err := s.issue(ctx, user.ID, models.UserTokenPurposeEmailVerification, verificationTokenTTL)
	if err != nil || token == nil {
		return err
	}
	return s.mailer.Enqueue(ctx, email.Request{
		UserID:    user.ID,
		To:        user.Email,
		Template:  email.TemplateAccountVerification,
		Locale:    localeOf(user, email.LocaleEnglish),
		Name:      email.GreetingName(user),
		Path:      "/verify-email#token=" + raw,
		DedupeKey: "email_verification:" + token.ID,
		ValidFor:  verificationTokenTTL,
	})
}

func (s *service) VerifyEmail(ctx context.Context, token string) error {
	if !validTokenShape(token) {
		return ErrInvalidToken
	}
	if _, err := s.tokens.ConsumeEmailVerification(ctx, hashToken(token), s.now()); err != nil {
		if errors.Is(err, repositories.ErrUserTokenInvalid) {
			return ErrInvalidToken
		}
		return err
	}
	return nil
}

func (s *service) SetLanguage(ctx context.Context, userID, language string) error {
	if language != email.LocaleEnglish && language != email.LocalePortuguese {
		return ErrInvalidInput
	}
	if err := s.users.SetPreferredLanguage(ctx, userID, language); err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	return nil
}
