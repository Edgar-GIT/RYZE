package account_recovery_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/email"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/account_recovery"
	"ryze/backend/services/password"
)

func setup(t *testing.T) (*gorm.DB, account_recovery.Service, repositories.UserRepository) {
	t.Helper()
	config.LoadEnvFile()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	tx := db.Begin()
	t.Cleanup(func() {
		tx.Rollback()
		// Each test opens its own pool; close it so repeated runs never
		// exhaust the server's connections.
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	users := repositories.NewUserRepository(tx)
	mailCfg := config.MailConfig{SMTPHost: "smtp.example", SMTPPort: "587", From: "RYZE <no-reply@ryze.example>", PublicAppURL: "https://ryze.example"}
	mailer := email.NewMailer(mailCfg, repositories.NewEmailOutboxRepository(tx))
	return tx, account_recovery.New(users, repositories.NewUserTokenRepository(tx), mailer), users
}

func seedUser(t *testing.T, users repositories.UserRepository) *models.User {
	t.Helper()
	hash, err := password.HashPassword("Original-Passw0rd!")
	if err != nil {
		t.Fatal(err)
	}
	user := &models.User{
		Email:        fmt.Sprintf("Recovery-%d@Ryze.local", time.Now().UnixNano()),
		PasswordHash: hash,
		FirstName:    "Ana",
		LastName:     "Silva",
	}
	if err := users.Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

// tokenFromOutbox extracts the raw token from the most recent email link.
func tokenFromOutbox(t *testing.T, tx *gorm.DB, userID, template string) (string, models.EmailOutbox) {
	t.Helper()
	var record models.EmailOutbox
	if err := tx.Where("user_id = ? AND template = ?", userID, template).Order("created_at DESC").First(&record).Error; err != nil {
		t.Fatalf("expected a %s email: %v", template, err)
	}
	var data email.Data
	if err := json.Unmarshal(record.Payload, &data); err != nil {
		t.Fatal(err)
	}
	_, token, found := strings.Cut(data.Link, "#token=")
	if !found {
		t.Fatalf("link %q carries no fragment token", data.Link)
	}
	return token, record
}

func TestPasswordRecoveryFlow(t *testing.T) {
	tx, svc, users := setup(t)
	ctx := context.Background()
	user := seedUser(t, users)
	versionBefore, _ := users.GetSessionVersion(ctx, user.ID)

	// Unknown, malformed and case-different addresses.
	if err := svc.RequestPasswordReset(ctx, "nobody-"+user.ID+"@ryze.local", "en"); err != nil {
		t.Fatalf("an unknown address must answer like a known one, got %v", err)
	}
	if err := svc.RequestPasswordReset(ctx, "not an email", "en"); !errors.Is(err, account_recovery.ErrInvalidInput) {
		t.Fatalf("a malformed address must be rejected, got %v", err)
	}
	if err := svc.RequestPasswordReset(ctx, strings.ToLower(user.Email), "pt"); err != nil {
		t.Fatalf("request reset: %v", err)
	}
	token, record := tokenFromOutbox(t, tx, user.ID, email.TemplatePasswordReset)
	if record.Locale != "pt" || record.ExpiresAt == nil || record.Status != models.EmailStatusPending {
		t.Fatalf("unexpected reset email %+v", record)
	}
	var stored models.UserToken
	if err := tx.Where("user_id = ? AND purpose = ?", user.ID, models.UserTokenPurposePasswordReset).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.TokenHash == token || len(stored.TokenHash) != 64 {
		t.Fatal("only the token hash may be stored")
	}

	// A second request right away is throttled: no new token, no new email.
	if err := svc.RequestPasswordReset(ctx, user.Email, "en"); err != nil {
		t.Fatal(err)
	}
	var count int64
	tx.Model(&models.UserToken{}).Where("user_id = ? AND purpose = ?", user.ID, models.UserTokenPurposePasswordReset).Count(&count)
	if count != 1 {
		t.Fatalf("expected one reset token while throttled, got %d", count)
	}

	// Policy and token checks.
	if err := svc.ResetPassword(ctx, token, "short"); !password.IsPolicyError(err) {
		t.Fatalf("a weak password must be rejected, got %v", err)
	}
	// Change the first character to a different one (the token is random, so
	// it may itself start with any letter).
	wrong := "x" + token[1:]
	if token[0] == 'x' {
		wrong = "y" + token[1:]
	}
	if err := svc.ResetPassword(ctx, wrong, "Brand-New-Passw0rd!"); !errors.Is(err, account_recovery.ErrInvalidToken) {
		t.Fatalf("a wrong token must be rejected, got %v", err)
	}
	if err := svc.ResetPassword(ctx, token, "Brand-New-Passw0rd!"); err != nil {
		t.Fatalf("reset password: %v", err)
	}
	if err := svc.ResetPassword(ctx, token, "Another-New-Passw0rd!"); !errors.Is(err, account_recovery.ErrInvalidToken) {
		t.Fatalf("a token must work once, got %v", err)
	}

	updated, err := users.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := password.VerifyPassword("Brand-New-Passw0rd!", updated.PasswordHash); !ok {
		t.Fatal("the new password must be stored")
	}
	if updated.SessionVersion != versionBefore+1 {
		t.Fatalf("a reset must revoke every session (version %d -> %d)", versionBefore, updated.SessionVersion)
	}
	var notice int64
	tx.Model(&models.EmailOutbox{}).Where("user_id = ? AND template = ?", user.ID, email.TemplatePasswordChanged).Count(&notice)
	if notice != 1 {
		t.Fatalf("expected one password-changed notice, got %d", notice)
	}
}

func TestExpiredResetTokenIsRejected(t *testing.T) {
	tx, svc, users := setup(t)
	ctx := context.Background()
	user := seedUser(t, users)
	if err := svc.RequestPasswordReset(ctx, user.Email, "en"); err != nil {
		t.Fatal(err)
	}
	token, _ := tokenFromOutbox(t, tx, user.ID, email.TemplatePasswordReset)
	if err := tx.Model(&models.UserToken{}).Where("user_id = ?", user.ID).
		Updates(map[string]any{"created_at": time.Now().Add(-3 * time.Hour), "expires_at": time.Now().Add(-2 * time.Hour)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.ResetPassword(ctx, token, "Brand-New-Passw0rd!"); !errors.Is(err, account_recovery.ErrInvalidToken) {
		t.Fatalf("an expired token must be rejected, got %v", err)
	}
}

func TestEmailVerificationAndLanguage(t *testing.T) {
	tx, svc, users := setup(t)
	ctx := context.Background()
	user := seedUser(t, users)

	if err := svc.SetLanguage(ctx, user.ID, "pt"); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetLanguage(ctx, user.ID, "de"); !errors.Is(err, account_recovery.ErrInvalidInput) {
		t.Fatalf("an unsupported language must be rejected, got %v", err)
	}
	if err := svc.SendVerification(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	token, record := tokenFromOutbox(t, tx, user.ID, email.TemplateAccountVerification)
	if record.Locale != "pt" {
		t.Fatalf("the stored language must drive the email, got %q", record.Locale)
	}
	if err := svc.VerifyEmail(ctx, token); err != nil {
		t.Fatalf("verify: %v", err)
	}
	verified, _ := users.FindByID(ctx, user.ID)
	if verified.EmailVerifiedAt == nil {
		t.Fatal("the email must be marked verified")
	}
	if err := svc.VerifyEmail(ctx, token); !errors.Is(err, account_recovery.ErrInvalidToken) {
		t.Fatalf("a verification link must work once, got %v", err)
	}
	if err := svc.SendVerification(ctx, user.ID); !errors.Is(err, account_recovery.ErrAlreadyVerified) {
		t.Fatalf("a verified account needs no link, got %v", err)
	}
}
