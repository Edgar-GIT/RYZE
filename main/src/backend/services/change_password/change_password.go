package change_password

import (
	"context"
	"errors"
	"fmt"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/password"
)

var (
	// ErrInvalidInput indicates the change-password input was malformed or
	// incomplete.
	ErrInvalidInput = errors.New("invalid change password input")
	// ErrInvalidCredentials is the single authentication error returned for
	// an unknown or soft-deleted user, a wrong current password and an
	// unverifiable stored hash. Callers cannot distinguish which case occurred.
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// PasswordVerifier is the verification dependency required by the
// change-password flow.
type PasswordVerifier interface {
	VerifyPassword(password, hash string) (bool, error)
}

// PasswordHasher is the hashing dependency required by the change-password
// flow.
type PasswordHasher interface {
	HashPassword(password string) (string, error)
}

// ChangePasswordService replaces the authenticated user's password and revokes
// all previously issued sessions.
type ChangePasswordService interface {
	ChangePassword(ctx context.Context, input Input) (*models.User, error)
}

// Input carries the credentials required to change a password. The user ID is
// resolved server-side from the authenticated session; the plaintext passwords
// are never logged, stored or returned.
type Input struct {
	UserID          string
	CurrentPassword string
	NewPassword     string
}

type changePasswordService struct {
	repo     repositories.UserRepository
	verifier PasswordVerifier
	hasher   PasswordHasher
}

func NewChangePasswordService(repo repositories.UserRepository, verifier PasswordVerifier, hasher PasswordHasher) ChangePasswordService {
	return &changePasswordService{repo: repo, verifier: verifier, hasher: hasher}
}

func (s *changePasswordService) ChangePassword(ctx context.Context, input Input) (*models.User, error) {
	if err := validateInput(input); err != nil {
		return nil, err
	}

	user, err := s.repo.FindByID(ctx, input.UserID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to load user: %w", err)
	}

	ok, err := s.verifier.VerifyPassword(input.CurrentPassword, user.PasswordHash)
	if err != nil || !ok {
		return nil, ErrInvalidCredentials
	}

	newHash, err := s.hasher.HashPassword(input.NewPassword)
	if err != nil {
		return nil, err
	}

	if err := s.repo.ChangePassword(ctx, input.UserID, newHash); err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to change password: %w", err)
	}

	user.PasswordHash = ""
	return user, nil
}

func validateInput(input Input) error {
	if input.UserID == "" {
		return fmt.Errorf("%w: user is required", ErrInvalidInput)
	}
	if input.CurrentPassword == "" {
		return password.ErrEmptyPassword
	}
	if input.NewPassword == "" {
		return password.ErrEmptyPassword
	}
	return nil
}
