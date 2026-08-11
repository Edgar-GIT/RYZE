package admin_users

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/password"
	"ryze/backend/services/registration"
)

var (
	// ErrInvalidInput indicates the admin user-management input was malformed
	// or incomplete.
	ErrInvalidInput = errors.New("invalid admin user management input")
	// ErrUserNotFound indicates the requested user does not exist or is not in
	// the state required by the operation (for example, an active-only
	// operation targeting a soft-deleted user).
	ErrUserNotFound = errors.New("user not found")
	// ErrDuplicateEmail indicates the email is already owned by an existing
	// active identity.
	ErrDuplicateEmail = errors.New("email already in use")
	// ErrAlreadyActive indicates a reactivation was attempted on an account
	// that is already active.
	ErrAlreadyActive = errors.New("user is already active")
)

// MaxPageSize caps the number of users returned in a single page. Larger
// limits are clamped to this value by the service.
const MaxPageSize = 100

// UserRepository is the data-access surface required by the admin user
// management service.
type UserRepository interface {
	ListActive(ctx context.Context, page, limit int) ([]models.User, int64, error)
	ListDeleted(ctx context.Context, page, limit int) ([]models.User, int64, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	FindByIDIncludingDeleted(ctx context.Context, id string) (*models.User, error)
	FindByEmailIncludingDeleted(ctx context.Context, email string) (*models.User, error)
	DeleteAccount(ctx context.Context, id string) error
	Update(ctx context.Context, user *models.User) error
	ChangePassword(ctx context.Context, id string, newHash string) error
	ClearDeletedAt(ctx context.Context, id string) error
}

// RegistrationService is the creation dependency required by the admin
// create-user operation. Reusing the public registration flow guarantees that
// admin-created users and re-registration of a soft-deleted email behave
// exactly like the public registration/reactivation lifecycle.
type RegistrationService interface {
	Register(ctx context.Context, input registration.RegisterInput) (*models.User, error)
}

// PasswordHasher is the hashing dependency required by the administrative
// password reset.
type PasswordHasher interface {
	HashPassword(password string) (string, error)
}

// ListUsersResult carries one page of users plus the pagination metadata needed
// to render the list.
type ListUsersResult struct {
	Users []models.User
	Total int64
	Page  int
	Limit int
}

// CreateUserInput carries the admin-provided data for creating a new user. The
// plaintext password is never logged, stored or returned, and the caller can
// never influence id, deleted_at or session_version.
type CreateUserInput struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
}

// UpdateUserInput carries the admin-provided fields for updating a user. Only
// explicitly whitelisted safe fields are accepted; nil means "leave unchanged".
type UpdateUserInput struct {
	Email     *string
	FirstName *string
	LastName  *string
}

// AdminUserService manages regular RYZE users on behalf of administrators.
// Authorization (which administrator may perform each operation) is enforced by
// the route middleware; this service only implements the domain rules.
type AdminUserService interface {
	ListUsers(ctx context.Context, page, limit int) (ListUsersResult, error)
	ListDeletedUsers(ctx context.Context, page, limit int) (ListUsersResult, error)
	GetUser(ctx context.Context, id string) (*models.User, error)
	CreateUser(ctx context.Context, input CreateUserInput) (*models.User, error)
	UpdateUser(ctx context.Context, id string, input UpdateUserInput) (*models.User, error)
	SoftDeleteUser(ctx context.Context, id string) error
	ReactivateUser(ctx context.Context, id string) (*models.User, error)
	ResetUserPassword(ctx context.Context, id, newPassword string) error
}

type adminUserService struct {
	repo      UserRepository
	registrar RegistrationService
	hasher    PasswordHasher
}

func NewAdminUserService(repo UserRepository, registrar RegistrationService, hasher PasswordHasher) AdminUserService {
	return &adminUserService{repo: repo, registrar: registrar, hasher: hasher}
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

// ListDeletedUsers returns one page of soft-deleted users (a clearly separated
// view from the normal active-user listing, used for lifecycle management) plus
// the total count of soft-deleted users.
func (s *adminUserService) ListDeletedUsers(ctx context.Context, page, limit int) (ListUsersResult, error) {
	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return ListUsersResult{}, err
	}

	users, total, err := s.repo.ListDeleted(ctx, page, limit)
	if err != nil {
		return ListUsersResult{}, fmt.Errorf("failed to list deleted users: %w", err)
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

// CreateUser creates a new active user. If the email belongs to a soft-deleted
// account, the existing identity is reactivated instead of creating a second
// row: the same UUID and created_at are preserved, deleted_at is cleared and
// the password is replaced with the supplied one. If the email belongs to an
// active account, ErrDuplicateEmail is returned. The returned user never
// carries a password hash.
func (s *adminUserService) CreateUser(ctx context.Context, input CreateUserInput) (*models.User, error) {
	user, err := s.registrar.Register(ctx, registration.RegisterInput{
		Email:     input.Email,
		Password:  input.Password,
		FirstName: input.FirstName,
		LastName:  input.LastName,
	})
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrDuplicateEmail):
			return nil, ErrDuplicateEmail
		case errors.Is(err, registration.ErrInvalidInput), errors.Is(err, password.ErrEmptyPassword):
			return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
		default:
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}
	return user, nil
}

// UpdateUser updates the whitelisted safe fields of an active user (email,
// first_name and last_name). At least one field must be provided. id,
// password_hash, deleted_at and session_version can never be modified through
// this operation; password changes use ResetUserPassword. The email must not
// be owned by any other existing identity (active or soft-deleted).
func (s *adminUserService) UpdateUser(ctx context.Context, id string, input UpdateUserInput) (*models.User, error) {
	if err := validateUserID(id); err != nil {
		return nil, err
	}
	if input.Email == nil && input.FirstName == nil && input.LastName == nil {
		return nil, fmt.Errorf("%w: at least one field is required", ErrInvalidInput)
	}
	if err := validateUpdateFields(input); err != nil {
		return nil, err
	}

	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if input.Email != nil && *input.Email != user.Email {
		other, err := s.repo.FindByEmailIncludingDeleted(ctx, *input.Email)
		switch {
		case err == nil && other.ID != id:
			return nil, ErrDuplicateEmail
		case err != nil && !errors.Is(err, repositories.ErrUserNotFound):
			return nil, fmt.Errorf("failed to check email: %w", err)
		}
	}

	if input.Email != nil {
		user.Email = *input.Email
	}
	if input.FirstName != nil {
		user.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		user.LastName = *input.LastName
	}

	if err := s.repo.Update(ctx, user); err != nil {
		if errors.Is(err, repositories.ErrDuplicateEmail) {
			return nil, ErrDuplicateEmail
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	updated, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to reload user: %w", err)
	}
	return updated, nil
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

// ReactivateUser restores a soft-deleted user: the same UUID and created_at are
// preserved, deleted_at is cleared, the account becomes active again and can
// authenticate again. The password is not changed. Because the account was
// soft-deleted with a session-version increment, every pre-deletion access
// token remains invalid. Active users and nonexistent users are rejected.
func (s *adminUserService) ReactivateUser(ctx context.Context, id string) (*models.User, error) {
	if err := validateUserID(id); err != nil {
		return nil, err
	}

	user, err := s.repo.FindByIDIncludingDeleted(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if !user.DeletedAt.Valid {
		return nil, ErrAlreadyActive
	}

	if err := s.repo.ClearDeletedAt(ctx, id); err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to reactivate user: %w", err)
	}

	active, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to reload user: %w", err)
	}
	return active, nil
}

// ResetUserPassword replaces an active user's password hash and invalidates
// every previously issued access token (the repository increments the session
// version in the same statement). It does NOT require the current password and
// never returns the plaintext password, the hash, a token or the session
// version.
func (s *adminUserService) ResetUserPassword(ctx context.Context, id, newPassword string) error {
	if err := validateUserID(id); err != nil {
		return err
	}
	if newPassword == "" {
		return password.ErrEmptyPassword
	}

	hash, err := s.hasher.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.repo.ChangePassword(ctx, id, hash); err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to reset password: %w", err)
	}
	return nil
}

func validateUpdateFields(input UpdateUserInput) error {
	if input.Email != nil {
		email := strings.TrimSpace(*input.Email)
		if email == "" || !strings.Contains(email, "@") {
			return fmt.Errorf("%w: invalid email", ErrInvalidInput)
		}
		*input.Email = email
	}
	if input.FirstName != nil {
		firstName := strings.TrimSpace(*input.FirstName)
		if firstName == "" {
			return fmt.Errorf("%w: first_name must not be empty", ErrInvalidInput)
		}
		*input.FirstName = firstName
	}
	if input.LastName != nil {
		lastName := strings.TrimSpace(*input.LastName)
		if lastName == "" {
			return fmt.Errorf("%w: last_name must not be empty", ErrInvalidInput)
		}
		*input.LastName = lastName
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
