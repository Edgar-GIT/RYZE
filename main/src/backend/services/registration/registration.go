package registration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/password"
)

var ErrInvalidInput = errors.New("invalid registration input")

// PasswordHasher is the hashing dependency required by the registration flow.
type PasswordHasher interface {
	HashPassword(password string) (string, error)
}

// RegistrationService registers new RYZE users.
type RegistrationService interface {
	Register(ctx context.Context, input RegisterInput) (*models.User, error)
}

// RegisterInput carries the user-provided registration data. The plaintext
// password is never logged, stored or returned.
type RegisterInput struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
}

type registrationService struct {
	repo   repositories.UserRepository
	hasher PasswordHasher
}

func NewRegistrationService(repo repositories.UserRepository, hasher PasswordHasher) RegistrationService {
	return &registrationService{repo: repo, hasher: hasher}
}

func (s *registrationService) Register(ctx context.Context, input RegisterInput) (*models.User, error) {
	if err := validateInput(input); err != nil {
		return nil, err
	}

	email := strings.TrimSpace(input.Email)
	firstName := strings.TrimSpace(input.FirstName)
	lastName := strings.TrimSpace(input.LastName)

	existing, err := s.repo.FindByEmailIncludingDeleted(ctx, email)
	switch {
	case errors.Is(err, repositories.ErrUserNotFound):
		// No account (active or soft-deleted) owns the email: register normally.
	case err != nil:
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	default:
		if !existing.DeletedAt.Valid {
			return nil, repositories.ErrDuplicateEmail
		}
		// A soft-deleted account re-registers with the same email: restore the
		// original row (same UUID) instead of creating a duplicate.
		hash, err := s.hasher.HashPassword(input.Password)
		if err != nil {
			return nil, err
		}
		existing.FirstName = firstName
		existing.LastName = lastName
		existing.PasswordHash = hash
		if err := s.repo.Reactivate(ctx, existing); err != nil {
			return nil, fmt.Errorf("failed to reactivate user: %w", err)
		}
		existing.PasswordHash = ""
		return existing, nil
	}

	hash, err := s.hasher.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:        email,
		PasswordHash: hash,
		FirstName:    firstName,
		LastName:     lastName,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		if errors.Is(err, repositories.ErrDuplicateEmail) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to register user: %w", err)
	}

	user.PasswordHash = ""
	return user, nil
}

func validateInput(input RegisterInput) error {
	if input.Email == "" || !strings.Contains(input.Email, "@") {
		return fmt.Errorf("%w: email is required", ErrInvalidInput)
	}
	if input.FirstName == "" {
		return fmt.Errorf("%w: first_name is required", ErrInvalidInput)
	}
	if input.LastName == "" {
		return fmt.Errorf("%w: last_name is required", ErrInvalidInput)
	}
	if input.Password == "" {
		return password.ErrEmptyPassword
	}
	return nil
}
