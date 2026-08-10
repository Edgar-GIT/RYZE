package repositories

import (
	"context"
	"errors"
	"fmt"

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
	Update(ctx context.Context, user *models.User) error
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
