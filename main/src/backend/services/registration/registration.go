package registration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/password"
	"ryze/backend/services/profile"
	"ryze/backend/services/test_mode"
)

var (
	ErrInvalidInput    = errors.New("invalid registration input")
	ErrInvalidUsername = errors.New("invalid username")
	// ErrInvalidDisplayName: the optional nickname is too long or has
	// control characters.
	ErrInvalidDisplayName = errors.New("invalid display name")
	ErrUsernameTaken      = errors.New("username taken")
)

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
	// Username is the chosen @username, required (owner decision,
	// 2026-11-21: the reference is created by the person, never derived from
	// the email).
	Username string
	// DisplayName is the optional nickname.
	DisplayName string
	// GenerateUsername is for accounts an administrator creates (they have no
	// @username field): with no Username, the database generates a
	// placeholder and the person chooses theirs on the first visit.
	GenerateUsername bool
}

// usernameChooser is implemented by the user repository.
type usernameChooser interface {
	ChooseUsername(ctx context.Context, id, username string) error
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

	// The fixed Test Mode personas are platform identities. Report the address
	// as already registered so a public caller cannot claim it, reactivate it,
	// or learn whether a persona account currently exists.
	if test_mode.IsReservedEmail(input.Email) {
		return nil, repositories.ErrDuplicateEmail
	}

	username := ""
	if !input.GenerateUsername || strings.TrimSpace(input.Username) != "" {
		normalized, ok := profile.NormalizeUsername(input.Username)
		if !ok || profile.IsReserved(normalized) {
			return nil, ErrInvalidUsername
		}
		username = normalized
	}

	nickname, ok := profile.NormalizeDisplayName(input.DisplayName)
	if !ok {
		return nil, ErrInvalidDisplayName
	}
	var displayName *string
	if nickname != "" {
		displayName = &nickname
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
		// The restored account takes the @username chosen now. If another
		// account holds it, the account keeps its earlier one and the person
		// is asked to choose on the next visit.
		if chooser, ok := s.repo.(usernameChooser); ok && username != "" {
			if err := chooser.ChooseUsername(ctx, existing.ID, username); err == nil {
				existing.Username = username
				existing.UsernameChosen = true
			} else if !errors.Is(err, repositories.ErrUsernameTaken) {
				return nil, fmt.Errorf("failed to set username: %w", err)
			}
		}
		existing.PasswordHash = ""
		return existing, nil
	}

	hash, err := s.hasher.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:          email,
		PasswordHash:   hash,
		FirstName:      firstName,
		LastName:       lastName,
		Username:       username,
		UsernameChosen: username != "",
		DisplayName:    displayName,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		if errors.Is(err, repositories.ErrDuplicateEmail) {
			return nil, err
		}
		if errors.Is(err, repositories.ErrUsernameTaken) {
			return nil, ErrUsernameTaken
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
	if err := password.ValidatePolicy(input.Password); err != nil {
		return err
	}
	return nil
}
