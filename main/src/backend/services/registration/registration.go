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

	hash, err := s.hasher.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:        strings.TrimSpace(input.Email),
		PasswordHash: hash,
		FirstName:    strings.TrimSpace(input.FirstName),
		LastName:     strings.TrimSpace(input.LastName),
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
