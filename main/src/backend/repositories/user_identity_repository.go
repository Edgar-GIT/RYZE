package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ryze/backend/models"
)

var (
	// ErrUserIdentityNotFound indicates no identity matches the lookup.
	ErrUserIdentityNotFound = errors.New("user identity not found")
	// ErrIdentityInUse indicates the provider account is already linked to a
	// RYZE account.
	ErrIdentityInUse = errors.New("identity already linked to an account")
	// ErrProviderAlreadyLinked indicates the RYZE account already links a
	// different account of the same provider.
	ErrProviderAlreadyLinked = errors.New("account already linked to this provider")
	// ErrPasswordRequired indicates an identity cannot be removed because it is
	// the account's only way to sign in.
	ErrPasswordRequired = errors.New("account has no password")
)

// UserIdentityRepository stores external sign-in identities and the account
// changes that go with them.
type UserIdentityRepository interface {
	FindByProviderSubject(ctx context.Context, provider, subject string) (*models.UserIdentity, error)
	ListByUser(ctx context.Context, userID string) ([]models.UserIdentity, error)
	// Link adds an identity to an existing, active account.
	Link(ctx context.Context, identity *models.UserIdentity) error
	// Touch records a sign-in with the identity and the provider's current
	// email address.
	Touch(ctx context.Context, id, email string, now time.Time) error
	// CreateUserWithIdentity creates an account without a known password
	// (password_set false), with a verified email, and links the identity, in
	// one transaction. It returns ErrDuplicateEmail when the address is taken.
	CreateUserWithIdentity(ctx context.Context, user *models.User, identity *models.UserIdentity, now time.Time) error
	// ReactivateWithIdentity restores a soft-deleted account for its provider
	// identity: names and an unusable password hash replace the old ones
	// (password_set false), the email is marked verified, and the account's
	// link for the provider becomes this identity, in one transaction.
	ReactivateWithIdentity(ctx context.Context, user *models.User, identity *models.UserIdentity, now time.Time) error
	// Unlink removes the account's identity for the provider, only while the
	// account has a password (ErrPasswordRequired otherwise).
	Unlink(ctx context.Context, userID, provider string) error
}

type userIdentityRepository struct {
	db *gorm.DB
}

func NewUserIdentityRepository(db *gorm.DB) UserIdentityRepository {
	return &userIdentityRepository{db: db}
}

func (r *userIdentityRepository) FindByProviderSubject(ctx context.Context, provider, subject string) (*models.UserIdentity, error) {
	var identity models.UserIdentity
	err := r.db.WithContext(ctx).Where("provider = ? AND subject = ?", provider, subject).Take(&identity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserIdentityNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user identity: %w", err)
	}
	return &identity, nil
}

func (r *userIdentityRepository) ListByUser(ctx context.Context, userID string) ([]models.UserIdentity, error) {
	var identities []models.UserIdentity
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("provider").Find(&identities).Error; err != nil {
		return nil, fmt.Errorf("failed to list user identities: %w", err)
	}
	return identities, nil
}

// identityInsertError maps a unique violation on user_identities to the
// matching domain error.
func identityInsertError(err error) error {
	if isDuplicateEntry(err) {
		switch violatedConstraint(err) {
		case "uq_user_identities_user_provider":
			return ErrProviderAlreadyLinked
		default:
			return ErrIdentityInUse
		}
	}
	return fmt.Errorf("failed to link user identity: %w", err)
}

func (r *userIdentityRepository) Link(ctx context.Context, identity *models.UserIdentity) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the account row so a concurrent deletion cannot leave an
		// identity pointing at an account removed in the meantime.
		var user models.User
		err := tx.Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).Select("id").Take(&user, "id = ?", identity.UserID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		if err != nil {
			return fmt.Errorf("failed to lock user: %w", err)
		}
		if err := tx.Create(identity).Error; err != nil {
			return identityInsertError(err)
		}
		return nil
	})
}

func (r *userIdentityRepository) Touch(ctx context.Context, id, email string, now time.Time) error {
	if err := r.db.WithContext(ctx).Model(&models.UserIdentity{}).Where("id = ?", id).
		Updates(map[string]any{"email": email, "last_used_at": now}).Error; err != nil {
		return fmt.Errorf("failed to update user identity: %w", err)
	}
	return nil
}

func (r *userIdentityRepository) CreateUserWithIdentity(ctx context.Context, user *models.User, identity *models.UserIdentity, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			if isDuplicateEntry(err) {
				return ErrDuplicateEmail
			}
			return fmt.Errorf("failed to create user: %w", err)
		}
		// password_set has a database default of TRUE, which GORM applies for
		// the zero value on insert; set the real values explicitly.
		if err := tx.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{
			"password_set":      false,
			"email_verified_at": now,
		}).Error; err != nil {
			return fmt.Errorf("failed to finish user: %w", err)
		}
		user.PasswordSet = false
		user.EmailVerifiedAt = &now
		identity.UserID = user.ID
		identity.LastUsedAt = &now
		if err := tx.Create(identity).Error; err != nil {
			return identityInsertError(err)
		}
		return nil
	})
}

func (r *userIdentityRepository) ReactivateWithIdentity(ctx context.Context, user *models.User, identity *models.UserIdentity, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Unscoped().Model(&models.User{}).
			Where("id = ? AND deleted_at IS NOT NULL", user.ID).
			Updates(map[string]any{
				"first_name":        user.FirstName,
				"last_name":         user.LastName,
				"password_hash":     user.PasswordHash,
				"password_set":      false,
				"email_verified_at": gorm.Expr("COALESCE(email_verified_at, ?)", now),
				"session_version":   gorm.Expr("session_version + 1"),
				"deleted_at":        nil,
				"updated_at":        now,
			})
		if result.Error != nil {
			if isDuplicateEntry(result.Error) {
				return ErrDuplicateEmail
			}
			return fmt.Errorf("failed to reactivate user: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrUserNotFound
		}
		// The restored account links exactly this provider account: drop a link
		// to a different account of the same provider left from before.
		if err := tx.Where("user_id = ? AND provider = ? AND subject <> ?", user.ID, identity.Provider, identity.Subject).
			Delete(&models.UserIdentity{}).Error; err != nil {
			return fmt.Errorf("failed to replace user identity: %w", err)
		}
		var existing models.UserIdentity
		err := tx.Where("provider = ? AND subject = ?", identity.Provider, identity.Subject).Take(&existing).Error
		switch {
		case err == nil:
			if existing.UserID != user.ID {
				return ErrIdentityInUse
			}
			*identity = existing
			return tx.Model(&models.UserIdentity{}).Where("id = ?", existing.ID).
				Updates(map[string]any{"email": user.Email, "last_used_at": now}).Error
		case errors.Is(err, gorm.ErrRecordNotFound):
			identity.UserID = user.ID
			identity.LastUsedAt = &now
			if err := tx.Create(identity).Error; err != nil {
				return identityInsertError(err)
			}
			return nil
		default:
			return fmt.Errorf("failed to find user identity: %w", err)
		}
	})
}

func (r *userIdentityRepository) Unlink(ctx context.Context, userID, provider string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user models.User
		err := tx.Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).Select("id", "password_set").Take(&user, "id = ?", userID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		if err != nil {
			return fmt.Errorf("failed to lock user: %w", err)
		}
		if !user.PasswordSet {
			return ErrPasswordRequired
		}
		result := tx.Where("user_id = ? AND provider = ?", userID, provider).Delete(&models.UserIdentity{})
		if result.Error != nil {
			return fmt.Errorf("failed to unlink user identity: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrUserIdentityNotFound
		}
		return nil
	})
}
