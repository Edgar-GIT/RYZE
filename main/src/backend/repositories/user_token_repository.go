package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ryze/backend/models"
)

// ErrUserTokenInvalid indicates the token is unknown, already used, expired or
// belongs to a removed account. The cases are deliberately indistinguishable.
var ErrUserTokenInvalid = errors.New("user token invalid")

// UserTokenRepository stores single-use account tokens (hashed) and applies
// their effects atomically.
type UserTokenRepository interface {
	Create(ctx context.Context, token *models.UserToken) error
	// CountRecent counts tokens issued for the purpose since the given time.
	CountRecent(ctx context.Context, userID, purpose string, since time.Time) (int64, error)
	// ConsumePasswordReset consumes a reset token, stores the new password hash,
	// revokes every session (session_version + 1) and retires the user's other
	// outstanding reset tokens, all in one transaction. It returns the user.
	ConsumePasswordReset(ctx context.Context, tokenHash, newPasswordHash string, now time.Time) (*models.User, error)
	// ConsumeEmailVerification consumes a verification token and marks the
	// account's email as verified. It returns the user.
	ConsumeEmailVerification(ctx context.Context, tokenHash string, now time.Time) (*models.User, error)
}

type userTokenRepository struct {
	db *gorm.DB
}

func NewUserTokenRepository(db *gorm.DB) UserTokenRepository {
	return &userTokenRepository{db: db}
}

func (r *userTokenRepository) Create(ctx context.Context, token *models.UserToken) error {
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		return fmt.Errorf("failed to create user token: %w", err)
	}
	return nil
}

func (r *userTokenRepository) CountRecent(ctx context.Context, userID, purpose string, since time.Time) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.UserToken{}).
		Where("user_id = ? AND purpose = ? AND created_at >= ?", userID, purpose, since).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count user tokens: %w", err)
	}
	return count, nil
}

// consume marks one valid token used and returns it, or ErrUserTokenInvalid.
// The conditional update makes a token single-use under concurrency.
func consume(tx *gorm.DB, tokenHash, purpose string, now time.Time) (*models.UserToken, error) {
	var token models.UserToken
	result := tx.Raw(`
		UPDATE user_tokens SET used_at = ?
		WHERE token_hash = ? AND purpose = ? AND used_at IS NULL AND expires_at > ?
		RETURNING *`, now, tokenHash, purpose, now).Scan(&token)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to consume user token: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrUserTokenInvalid
	}
	return &token, nil
}

func (r *userTokenRepository) ConsumePasswordReset(ctx context.Context, tokenHash, newPasswordHash string, now time.Time) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		token, err := consume(tx, tokenHash, models.UserTokenPurposePasswordReset, now)
		if err != nil {
			return err
		}
		// Recovery shuts every other door: when the account already had a
		// password, someone who knew it may have connected their own Google
		// account, so external sign-in links are removed with the sessions.
		// An account created through Google (no password yet) keeps the
		// Google account that created it.
		var current models.User
		if err := tx.Select("id", "password_set").Take(&current, "id = ?", token.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrUserTokenInvalid
			}
			return fmt.Errorf("failed to load user: %w", err)
		}
		if current.PasswordSet {
			if err := tx.Where("user_id = ?", token.UserID).Delete(&models.UserIdentity{}).Error; err != nil {
				return fmt.Errorf("failed to remove sign-in links: %w", err)
			}
		}
		update := tx.Model(&models.User{}).
			Where("id = ?", token.UserID).
			Updates(map[string]any{
				"password_hash":   newPasswordHash,
				"password_set":    true,
				"session_version": gorm.Expr("session_version + 1"),
				"updated_at":      now,
			})
		if update.Error != nil {
			return fmt.Errorf("failed to reset password: %w", update.Error)
		}
		if update.RowsAffected == 0 {
			// The account was removed after the link was sent.
			return ErrUserTokenInvalid
		}
		if err := tx.Model(&models.UserToken{}).
			Where("user_id = ? AND purpose = ? AND used_at IS NULL", token.UserID, models.UserTokenPurposePasswordReset).
			Update("used_at", now).Error; err != nil {
			return fmt.Errorf("failed to retire reset tokens: %w", err)
		}
		return tx.First(&user, "id = ?", token.UserID).Error
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userTokenRepository) ConsumeEmailVerification(ctx context.Context, tokenHash string, now time.Time) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		token, err := consume(tx, tokenHash, models.UserTokenPurposeEmailVerification, now)
		if err != nil {
			return err
		}
		update := tx.Model(&models.User{}).
			Where("id = ?", token.UserID).
			Update("email_verified_at", gorm.Expr("COALESCE(email_verified_at, ?)", now))
		if update.Error != nil {
			return fmt.Errorf("failed to verify email: %w", update.Error)
		}
		if update.RowsAffected == 0 {
			return ErrUserTokenInvalid
		}
		return tx.First(&user, "id = ?", token.UserID).Error
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}
