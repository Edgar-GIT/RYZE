package login

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/password"
)

var (
	// ErrInvalidInput indicates the login input was malformed or incomplete.
	ErrInvalidInput = errors.New("invalid login input")
	// ErrInvalidCredentials is the single authentication error returned for
	// unknown emails, wrong passwords, soft-deleted users and unverifiable
	// stored hashes. Callers cannot distinguish which case occurred.
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// PasswordVerifier is the verification dependency required by the login flow.
type PasswordVerifier interface {
	VerifyPassword(password, hash string) (bool, error)
}

// dummyPasswordHash is a valid Argon2id hash (same parameters as stored hashes)
// of a random, never-revealed value. When the supplied email does not resolve
// to an account the login flow verifies the supplied password against this
// hash, so the work performed is comparable to a real verification. This keeps
// the response time for an unknown email indistinguishable from a wrong
// password and removes account enumeration through timing. It is never a valid
// credential for any account.
const dummyPasswordHash = "$argon2id$v=19$m=65536,t=3,p=4$P6hWtEAgL9s9sDpyNlv8ZA$kF2Fa/A54s9S2yTQyz/+0e22Qiwhwqq8tJxfNp3M1UQ"

// LoginService authenticates users against the stored credentials.
type LoginService interface {
	Login(ctx context.Context, input LoginInput) (*models.User, error)
}

// LoginInput carries the supplied credentials. The plaintext password is never
// logged, stored, returned or embedded in an error.
type LoginInput struct {
	Email    string
	Password string
}

type loginService struct {
	repo     repositories.UserRepository
	verifier PasswordVerifier
}

func NewLoginService(repo repositories.UserRepository, verifier PasswordVerifier) LoginService {
	return &loginService{repo: repo, verifier: verifier}
}

func (s *loginService) Login(ctx context.Context, input LoginInput) (*models.User, error) {
	if err := validateInput(input); err != nil {
		return nil, err
	}

	user, err := s.repo.FindByEmail(ctx, strings.TrimSpace(input.Email))
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			// Unknown email: verify against the dummy hash so the response time
			// matches a wrong-password attempt. The result is deliberately
			// ignored; the outcome is always the same generic error.
			_, _ = s.verifier.VerifyPassword(input.Password, dummyPasswordHash)
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to authenticate user: %w", err)
	}

	ok, err := s.verifier.VerifyPassword(input.Password, user.PasswordHash)
	if err != nil || !ok {
		return nil, ErrInvalidCredentials
	}

	user.PasswordHash = ""
	return user, nil
}

func validateInput(input LoginInput) error {
	if input.Email == "" || !strings.Contains(input.Email, "@") {
		return fmt.Errorf("%w: email is required", ErrInvalidInput)
	}
	if input.Password == "" {
		return password.ErrEmptyPassword
	}
	return nil
}
