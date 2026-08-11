package delete_account

import (
	"context"
	"errors"
	"fmt"

	"ryze/backend/repositories"
	"ryze/backend/services/password"
)

var (
	// ErrInvalidInput indicates the delete-account input was malformed or
	// incomplete.
	ErrInvalidInput = errors.New("invalid delete account input")
	// ErrInvalidCredentials is the single authentication error returned for
	// an unknown or already-deleted user, a wrong password and an unverifiable
	// stored hash. Callers cannot distinguish which case occurred.
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// PasswordVerifier is the verification dependency required by the
// delete-account flow.
type PasswordVerifier interface {
	VerifyPassword(password, hash string) (bool, error)
}

// DeleteAccountService soft-deletes the authenticated user's account after
// verifying the current password.
type DeleteAccountService interface {
	DeleteAccount(ctx context.Context, input Input) error
}

// Input carries the confirmation required to delete an account. The user ID is
// resolved server-side from the authenticated session; the plaintext password
// is never logged, stored or returned.
type Input struct {
	UserID   string
	Password string
}

type deleteAccountService struct {
	repo     repositories.UserRepository
	verifier PasswordVerifier
}

func NewDeleteAccountService(repo repositories.UserRepository, verifier PasswordVerifier) DeleteAccountService {
	return &deleteAccountService{repo: repo, verifier: verifier}
}

func (s *deleteAccountService) DeleteAccount(ctx context.Context, input Input) error {
	if err := validateInput(input); err != nil {
		return err
	}

	user, err := s.repo.FindByID(ctx, input.UserID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return ErrInvalidCredentials
		}
		return fmt.Errorf("failed to load user: %w", err)
	}

	ok, err := s.verifier.VerifyPassword(input.Password, user.PasswordHash)
	if err != nil || !ok {
		return ErrInvalidCredentials
	}

	if err := s.repo.DeleteAccount(ctx, input.UserID); err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return ErrInvalidCredentials
		}
		return fmt.Errorf("failed to delete account: %w", err)
	}

	return nil
}

func validateInput(input Input) error {
	if input.UserID == "" {
		return fmt.Errorf("%w: user is required", ErrInvalidInput)
	}
	if input.Password == "" {
		return password.ErrEmptyPassword
	}
	return nil
}
