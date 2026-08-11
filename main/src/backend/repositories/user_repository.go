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
	FindByIDIncludingDeleted(ctx context.Context, id string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByEmailIncludingDeleted(ctx context.Context, email string) (*models.User, error)
	ListActive(ctx context.Context, page, limit int) ([]models.User, int64, error)
	ListDeleted(ctx context.Context, page, limit int) ([]models.User, int64, error)
	GetSessionVersion(ctx context.Context, id string) (int, error)
	Update(ctx context.Context, user *models.User) error
	ChangePassword(ctx context.Context, id string, newHash string) error
	Reactivate(ctx context.Context, user *models.User) error
	ClearDeletedAt(ctx context.Context, id string) error
	SoftDelete(ctx context.Context, id string) error
	DeleteAccount(ctx context.Context, id string) error
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

// FindByIDIncludingDeleted looks up a user by id without excluding soft-deleted
// rows. It is used by the admin lifecycle operations (reactivation) to inspect
// accounts that are no longer active.
func (r *userRepository) FindByIDIncludingDeleted(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Unscoped().First(&user, "id = ?", id).Error; err != nil {
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

// ListActive returns one page of active users (soft-deleted users are excluded
// by GORM's default scope) ordered by creation time, plus the total number of
// active users. The caller guarantees page >= 1 and limit >= 1.
func (r *userRepository) ListActive(ctx context.Context, page, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Order("created_at DESC, id ASC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	return users, total, nil
}

// ListDeleted returns one page of soft-deleted users (rows with a populated
// deleted_at) ordered by deletion time, plus the total number of soft-deleted
// users. It is used by the admin lifecycle management as a clearly separated
// view from the normal active-user listing. The caller guarantees page >= 1
// and limit >= 1.
func (r *userRepository) ListDeleted(ctx context.Context, page, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	if err := r.db.WithContext(ctx).
		Unscoped().
		Model(&models.User{}).
		Where("deleted_at IS NOT NULL").
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count deleted users: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Unscoped().
		Where("deleted_at IS NOT NULL").
		Order("deleted_at DESC, id ASC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list deleted users: %w", err)
	}
	return users, total, nil
}

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

// ClearDeletedAt reactivates a soft-deleted user without touching any account
// data: the UUID, email, created_at, password hash and session version are all
// preserved, only deleted_at is cleared (and updated_at refreshed). The
// `deleted_at IS NOT NULL` guard makes reactivation of an already-active user
// impossible. Because the account was soft-deleted with a session-version
// increment, every pre-deletion access token remains invalid afterwards.
func (r *userRepository) ClearDeletedAt(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Unscoped().
		Model(&models.User{}).
		Where("id = ? AND deleted_at IS NOT NULL", id).
		Updates(map[string]any{
			"deleted_at": nil,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
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

// DeleteAccount soft-deletes the user's account and invalidates every
// previously issued access token in the same statement. The row is never
// physically removed: id, email and created_at are preserved so the account
// remains available for the soft-delete/reactivation lifecycle. The
// `deleted_at IS NULL` guard makes a second deletion impossible.
func (r *userRepository) DeleteAccount(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{
			"deleted_at":      time.Now(),
			"session_version": gorm.Expr("session_version + 1"),
			"updated_at":      time.Now(),
		})
	if result.Error != nil {
		return fmt.Errorf("failed to delete account: %w", result.Error)
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
