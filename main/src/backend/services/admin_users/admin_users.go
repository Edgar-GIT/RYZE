package admin_users

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

var (
	// ErrInvalidInput indicates the admin user-management input was malformed
	// or incomplete.
	ErrInvalidInput = errors.New("invalid admin user management input")
	// ErrUserNotFound indicates the requested user does not exist or is not
	// active (soft-deleted). Soft-deleted users are never returned by the
	// normal endpoints and cannot be deleted again.
	ErrUserNotFound = errors.New("user not found")
)

// MaxPageSize caps the number of users returned in a single page. Larger
// limits are clamped to this value by the service.
const MaxPageSize = 100

// UserRepository is the data-access surface required by the admin user
// management service.
type UserRepository interface {
	ListActive(ctx context.Context, page, limit int) ([]models.User, int64, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	DeleteAccount(ctx context.Context, id string) error
}

// ListUsersResult carries one page of users plus the pagination metadata needed
// to render the list.
type ListUsersResult struct {
	Users []models.User
	Total int64
	Page  int
	Limit int
}

// AdminUserService manages regular RYZE users on behalf of administrators.
type AdminUserService interface {
	ListUsers(ctx context.Context, page, limit int) (ListUsersResult, error)
	GetUser(ctx context.Context, id string) (*models.User, error)
	SoftDeleteUser(ctx context.Context, id string) error
}

type adminUserService struct {
	repo UserRepository
}

func NewAdminUserService(repo UserRepository) AdminUserService {
	return &adminUserService{repo: repo}
}

// ListUsers returns one page of active users (soft-deleted users are never
// listed) plus the total count of active users. The page must be at least 1
// and the limit must be at least 1; limits above MaxPageSize are clamped.
func (s *adminUserService) ListUsers(ctx context.Context, page, limit int) (ListUsersResult, error) {
	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return ListUsersResult{}, err
	}

	users, total, err := s.repo.ListActive(ctx, page, limit)
	if err != nil {
		return ListUsersResult{}, fmt.Errorf("failed to list users: %w", err)
	}
	return ListUsersResult{Users: users, Total: total, Page: page, Limit: limit}, nil
}

// GetUser returns one active user by its UUID. Soft-deleted users are never
// returned.
func (s *adminUserService) GetUser(ctx context.Context, id string) (*models.User, error) {
	if err := validateUserID(id); err != nil {
		return nil, err
	}

	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// SoftDeleteUser soft-deletes an active user by its UUID. The row is never
// physically removed: id, email and created_at are preserved. The operation is
// atomic: it sets deleted_at and increments session_version in the same
// statement, so every previously issued access token is revoked immediately.
// Soft-deleted and nonexistent users are reported as not found.
func (s *adminUserService) SoftDeleteUser(ctx context.Context, id string) error {
	if err := validateUserID(id); err != nil {
		return err
	}

	if err := s.repo.DeleteAccount(ctx, id); err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to soft delete user: %w", err)
	}
	return nil
}

// normalizePagination validates the pagination parameters and clamps oversized
// limits to MaxPageSize.
func normalizePagination(page, limit int) (int, int, error) {
	if page < 1 {
		return 0, 0, fmt.Errorf("%w: page must be at least 1", ErrInvalidInput)
	}
	if limit < 1 {
		return 0, 0, fmt.Errorf("%w: limit must be at least 1", ErrInvalidInput)
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	return page, limit, nil
}

// validateUserID rejects empty and malformed identifiers before any database
// access.
func validateUserID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid user id", ErrInvalidInput)
	}
	return nil
}
