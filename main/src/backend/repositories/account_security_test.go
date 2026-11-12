package repositories_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

// accountSecurityTx runs against the isolated test database inside a
// transaction that is rolled back when the test ends.
func accountSecurityTx(t *testing.T) *gorm.DB {
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
	return tx
}

func seedSecurityUser(t *testing.T, tx *gorm.DB, label string) *models.User {
	t.Helper()
	user := &models.User{Email: fmt.Sprintf("acct-sec-%s-%d@ryze.local", label, time.Now().UnixNano()), PasswordHash: "hash", FirstName: "Ana", LastName: label}
	if err := repositories.NewUserRepository(tx).Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func linkGoogle(t *testing.T, tx *gorm.DB, user *models.User) {
	t.Helper()
	identity := &models.UserIdentity{UserID: user.ID, Provider: models.IdentityProviderGoogle, Subject: "sub-" + user.ID, Email: "x@gmail.com"}
	if err := tx.Create(identity).Error; err != nil {
		t.Fatalf("link identity: %v", err)
	}
}

func identityCount(t *testing.T, tx *gorm.DB, userID string) int64 {
	t.Helper()
	var count int64
	tx.Model(&models.UserIdentity{}).Where("user_id = ?", userID).Count(&count)
	return count
}

func issueToken(t *testing.T, tx *gorm.DB, userID, purpose, raw string) string {
	t.Helper()
	sum := sha256.Sum256([]byte(raw))
	hash := hex.EncodeToString(sum[:])
	token := &models.UserToken{UserID: userID, Purpose: purpose, TokenHash: hash, ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now()}
	if err := repositories.NewUserTokenRepository(tx).Create(context.Background(), token); err != nil {
		t.Fatalf("create token: %v", err)
	}
	return hash
}

func TestPasswordResetRemovesExternalSignInLinks(t *testing.T) {
	tx := accountSecurityTx(t)
	ctx := context.Background()
	tokens := repositories.NewUserTokenRepository(tx)

	// An account with a password: someone who knew it may have linked their
	// own Google account, so recovery removes the link.
	user := seedSecurityUser(t, tx, "reset")
	linkGoogle(t, tx, user)
	hash := issueToken(t, tx, user.ID, models.UserTokenPurposePasswordReset, "reset-"+user.ID)
	if _, err := tokens.ConsumePasswordReset(ctx, hash, "new-hash", time.Now()); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if identityCount(t, tx, user.ID) != 0 {
		t.Fatal("a Google link survived the password reset")
	}

	// An account created through Google keeps the Google account that
	// created it when its owner sets a first password.
	googleOnly := seedSecurityUser(t, tx, "google-only")
	if err := tx.Model(&models.User{}).Where("id = ?", googleOnly.ID).Update("password_set", false).Error; err != nil {
		t.Fatal(err)
	}
	linkGoogle(t, tx, googleOnly)
	hash = issueToken(t, tx, googleOnly.ID, models.UserTokenPurposePasswordReset, "reset-"+googleOnly.ID)
	if _, err := tokens.ConsumePasswordReset(ctx, hash, "new-hash", time.Now()); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if identityCount(t, tx, googleOnly.ID) != 1 {
		t.Fatal("the creating Google account was removed from a Google-only account")
	}
}

func TestReRegistrationRemovesOldSignInLinks(t *testing.T) {
	tx := accountSecurityTx(t)
	ctx := context.Background()
	users := repositories.NewUserRepository(tx)
	user := seedSecurityUser(t, tx, "rereg")
	linkGoogle(t, tx, user)
	if err := users.DeleteAccount(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	deleted, err := users.FindByIDIncludingDeleted(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	deleted.PasswordHash = "new-owner-hash"
	if err := users.Reactivate(ctx, deleted); err != nil {
		t.Fatalf("reactivate: %v", err)
	}
	if identityCount(t, tx, user.ID) != 0 {
		t.Fatal("a Google link from before the deletion survived re-registration")
	}
}

func TestEmailChangeInvalidatesProofAndLinks(t *testing.T) {
	tx := accountSecurityTx(t)
	ctx := context.Background()
	users := repositories.NewUserRepository(tx)
	tokens := repositories.NewUserTokenRepository(tx)
	user := seedSecurityUser(t, tx, "email")
	if err := tx.Model(&models.User{}).Where("id = ?", user.ID).Update("email_verified_at", time.Now()).Error; err != nil {
		t.Fatal(err)
	}
	resetHash := issueToken(t, tx, user.ID, models.UserTokenPurposePasswordReset, "reset-"+user.ID)
	verifyHash := issueToken(t, tx, user.ID, models.UserTokenPurposeEmailVerification, "verify-"+user.ID)

	// A name-only update keeps the proof.
	user.FirstName = "Renamed"
	if err := users.Update(ctx, user); err != nil {
		t.Fatal(err)
	}
	if fresh, _ := users.FindByID(ctx, user.ID); fresh.EmailVerifiedAt == nil {
		t.Fatal("a name change cleared the email verification")
	}

	user.Email = fmt.Sprintf("acct-sec-moved-%d@ryze.local", time.Now().UnixNano())
	if err := users.Update(ctx, user); err != nil {
		t.Fatal(err)
	}
	fresh, err := users.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.EmailVerifiedAt != nil {
		t.Fatal("the new address is marked verified")
	}
	if _, err := tokens.ConsumePasswordReset(ctx, resetHash, "x", time.Now()); !errors.Is(err, repositories.ErrUserTokenInvalid) {
		t.Fatalf("a reset link sent to the old address still works: %v", err)
	}
	if _, err := tokens.ConsumeEmailVerification(ctx, verifyHash, time.Now()); !errors.Is(err, repositories.ErrUserTokenInvalid) {
		t.Fatalf("a verification link sent to the old address still works: %v", err)
	}
}

func TestPasswordChangeRetiresResetLinks(t *testing.T) {
	tx := accountSecurityTx(t)
	ctx := context.Background()
	user := seedSecurityUser(t, tx, "change")
	hash := issueToken(t, tx, user.ID, models.UserTokenPurposePasswordReset, "reset-"+user.ID)
	if err := repositories.NewUserRepository(tx).ChangePassword(ctx, user.ID, "changed-hash"); err != nil {
		t.Fatal(err)
	}
	if _, err := repositories.NewUserTokenRepository(tx).ConsumePasswordReset(ctx, hash, "x", time.Now()); !errors.Is(err, repositories.ErrUserTokenInvalid) {
		t.Fatalf("a reset link issued before the change still works: %v", err)
	}
}
