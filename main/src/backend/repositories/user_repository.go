package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	gomysql "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"ryze/backend/models"
)

var (
	ErrUserNotFound   = errors.New("user not found")
	ErrDuplicateEmail = errors.New("email already in use")
)

const mysqlDuplicateEntry = 1062

// UserRepository defines the data-access operations for users.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByID(ctx context.Context, id string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByEmailIncludingDeleted(ctx context.Context, email string) (*models.User, error)
	GetSessionVersion(ctx context.Context, id string) (int, error)
	Update(ctx context.Context, user *models.User) error
	ChangePassword(ctx context.Context, id string, newHash string) error
	Reactivate(ctx context.Context, user *models.User) error
	SoftDelete(ctx context.Context, id string) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		if isDuplicateEntry(err) {
			return ErrDuplicateEmail
		}
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}
	return &user, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	return &user, nil
}

// FindByEmailIncludingDeleted looks up a user by email without excluding
// soft-deleted rows. It is used to detect whether a previous account owns the
// email so it can be reactivated instead of creating a duplicate.
func (r *userRepository) FindByEmailIncludingDeleted(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Unscoped().Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	return &user, nil
}

// GetSessionVersion returns the current session version of the user. It is
// used by the authentication middleware to reject access tokens that were
// issued for a revoked session (e.g. after a password change).
func (r *userRepository) GetSessionVersion(ctx context.Context, id string) (int, error) {
	var user models.User
	if err := r.db.WithContext(ctx).
		Select("session_version").
		First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrUserNotFound
		}
		return 0, fmt.Errorf("failed to get session version: %w", err)
	}
	return user.SessionVersion, nil
}

func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).
		Model(user).
		Select("first_name", "last_name", "email").
		Updates(user).Error; err != nil {
		if isDuplicateEntry(err) {
			return ErrDuplicateEmail
		}
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// ChangePassword replaces the user's password hash and invalidates every
// previously issued access token by incrementing the session version. Both
// changes happen in the same statement so the new hash can never be stored
// with a session version that leaves old tokens valid.
func (r *userRepository) ChangePassword(ctx context.Context, id string, newHash string) error {
	result := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"password_hash":   newHash,
			"session_version": gorm.Expr("session_version + 1"),
			"updated_at":      time.Now(),
		})
	if result.Error != nil {
		return fmt.Errorf("failed to change password: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// Reactivate restores a soft-deleted user row, clearing deleted_at and
// replacing the account data (names and password hash) with the provided
// values. The UUID, created_at and email are preserved.
func (r *userRepository) Reactivate(ctx context.Context, user *models.User) error {
	result := r.db.WithContext(ctx).
		Unscoped().
		Model(user).
		Select("first_name", "last_name", "password_hash", "deleted_at").
		Updates(map[string]any{
			"first_name":    user.FirstName,
			"last_name":     user.LastName,
			"password_hash": user.PasswordHash,
			"deleted_at":    nil,
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
	return nil
}

func (r *userRepository) SoftDelete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.User{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to soft delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func isDuplicateEntry(err error) bool {
	var mysqlErr *gomysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlDuplicateEntry
}
