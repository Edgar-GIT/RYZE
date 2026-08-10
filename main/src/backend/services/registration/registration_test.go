package registration_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/password"
	"ryze/backend/services/registration"
)

func TestRegisterSuccess(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	svc := registration.NewRegistrationService(repo, password.Hasher{})
	ctx := context.Background()
	email := fmt.Sprintf("register-test-%d@ryze.local", time.Now().UnixNano())
	plaintext := "SuperSecret123!"

	user, err := svc.Register(ctx, registration.RegisterInput{
		Email:     email,
		Password:  plaintext,
		FirstName: "Alice",
		LastName:  "Example",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if user.ID == "" {
		t.Fatal("Register: expected generated UUID id")
	}
	if user.PasswordHash != "" {
		t.Fatal("Register: returned user must not expose password_hash")
	}

	stored, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if stored.PasswordHash == "" {
		t.Fatal("Register: password_hash must be persisted")
	}
	if stored.PasswordHash == plaintext {
		t.Fatal("Register: password must be stored only as a hash")
	}
	if contains(stored.PasswordHash, plaintext) {
		t.Fatal("Register: stored hash must not contain the plaintext password")
	}

	ok, err := password.VerifyPassword(plaintext, stored.PasswordHash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatal("Register: stored hash must verify against the correct password")
	}

	ok, err = password.VerifyPassword("wrong-password", stored.PasswordHash)
	if err != nil {
		t.Fatalf("VerifyPassword (wrong): %v", err)
	}
	if ok {
		t.Fatal("Register: stored hash must not verify against a wrong password")
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	svc := registration.NewRegistrationService(repo, password.Hasher{})
	ctx := context.Background()
	email := fmt.Sprintf("register-dup-%d@ryze.local", time.Now().UnixNano())

	if _, err := svc.Register(ctx, registration.RegisterInput{
		Email:     email,
		Password:  "Password123!",
		FirstName: "First",
		LastName:  "Last",
	}); err != nil {
		t.Fatalf("first Register: %v", err)
	}

	_, err := svc.Register(ctx, registration.RegisterInput{
		Email:     email,
		Password:  "Password123!",
		FirstName: "First",
		LastName:  "Last",
	})
	if !errors.Is(err, repositories.ErrDuplicateEmail) {
		t.Fatalf("second Register: expected ErrDuplicateEmail, got %v", err)
	}
}

func TestRegisterEmptyPassword(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	svc := registration.NewRegistrationService(repo, password.Hasher{})

	_, err := svc.Register(context.Background(), registration.RegisterInput{
		Email:     "empty-password@ryze.local",
		Password:  "",
		FirstName: "First",
		LastName:  "Last",
	})
	if !errors.Is(err, password.ErrEmptyPassword) {
		t.Fatalf("Register: expected ErrEmptyPassword, got %v", err)
	}
}

func TestRegisterInvalidInput(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	svc := registration.NewRegistrationService(repo, password.Hasher{})

	cases := []registration.RegisterInput{
		{Email: "", Password: "Password123!", FirstName: "First", LastName: "Last"},
		{Email: "not-an-email", Password: "Password123!", FirstName: "First", LastName: "Last"},
		{Email: "a@b.local", Password: "Password123!", FirstName: "", LastName: "Last"},
		{Email: "a@b.local", Password: "Password123!", FirstName: "First", LastName: ""},
	}

	for i, input := range cases {
		if _, err := svc.Register(context.Background(), input); !errors.Is(err, registration.ErrInvalidInput) {
			t.Errorf("case %d: expected ErrInvalidInput, got %v", i, err)
		}
	}
}

func TestRegisterRepositoryErrorPropagation(t *testing.T) {
	svc := registration.NewRegistrationService(failingRepo{}, password.Hasher{})

	_, err := svc.Register(context.Background(), registration.RegisterInput{
		Email:     "repo-error@ryze.local",
		Password:  "Password123!",
		FirstName: "First",
		LastName:  "Last",
	})
	if !errors.Is(err, errRepositoryFailure) {
		t.Fatalf("Register: expected repository error to propagate, got %v", err)
	}
	if errors.Is(err, repositories.ErrDuplicateEmail) {
		t.Fatal("Register: repository failure must not be reported as ErrDuplicateEmail")
	}
}

func newTestRepository(t *testing.T) (repositories.UserRepository, func()) {
	t.Helper()

	config.LoadEnvFile()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	tx := db.Begin()
	return repositories.NewUserRepository(tx), func() { _ = tx.Rollback() }
}

func contains(hash, plaintext string) bool {
	return strings.Contains(hash, plaintext)
}

type failingRepo struct{}

var errRepositoryFailure = errors.New("repository failure")

func (failingRepo) Create(_ context.Context, _ *models.User) error {
	return errRepositoryFailure
}
func (failingRepo) FindByID(_ context.Context, _ string) (*models.User, error) {
	return nil, errRepositoryFailure
}
func (failingRepo) FindByEmail(_ context.Context, _ string) (*models.User, error) {
	return nil, errRepositoryFailure
}
func (failingRepo) Update(_ context.Context, _ *models.User) error {
	return errRepositoryFailure
}
func (failingRepo) SoftDelete(_ context.Context, _ string) error {
	return errRepositoryFailure
}
