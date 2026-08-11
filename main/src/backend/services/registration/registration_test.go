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

	"gorm.io/gorm"
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

func TestRegisterReactivateSoftDeletedUser(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	svc := registration.NewRegistrationService(repo, password.Hasher{})
	ctx := context.Background()
	email := fmt.Sprintf("register-reactivate-%d@ryze.local", time.Now().UnixNano())
	oldPassword := "OldPassword123!"
	newPassword := "NewPassword456!"

	first, err := svc.Register(ctx, registration.RegisterInput{
		Email:     email,
		Password:  oldPassword,
		FirstName: "First",
		LastName:  "User",
	})
	if err != nil {
		t.Fatalf("first Register: %v", err)
	}

	if err := repo.SoftDelete(ctx, first.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	reactivated, err := svc.Register(ctx, registration.RegisterInput{
		Email:     email,
		Password:  newPassword,
		FirstName: "Second",
		LastName:  "Name",
	})
	if err != nil {
		t.Fatalf("Register after soft delete: %v", err)
	}
	if reactivated.ID != first.ID {
		t.Fatalf("reactivation must keep the same UUID: got %q, want %q", reactivated.ID, first.ID)
	}
	if reactivated.PasswordHash != "" {
		t.Fatal("reactivated user must not expose password_hash")
	}
	if reactivated.DeletedAt.Valid {
		t.Fatal("reactivated user must not expose deleted_at")
	}

	// No second row is created: the same row is restored and active again.
	stored, err := repo.FindByID(ctx, first.ID)
	if err != nil {
		t.Fatalf("FindByID after reactivation: %v", err)
	}
	if stored.FirstName != "Second" || stored.LastName != "Name" {
		t.Fatalf("reactivation must update names, got %s %s", stored.FirstName, stored.LastName)
	}
	if stored.Email != email {
		t.Fatalf("reactivation must keep the email, got %q", stored.Email)
	}
	if !stored.CreatedAt.Equal(first.CreatedAt) {
		t.Fatal("reactivation must preserve created_at")
	}

	// The stored hash was replaced with a fresh Argon2id hash: only the new
	// password verifies, the old password no longer works.
	ok, err := password.VerifyPassword(newPassword, stored.PasswordHash)
	if err != nil {
		t.Fatalf("VerifyPassword (new): %v", err)
	}
	if !ok {
		t.Fatal("new password must verify after reactivation")
	}
	ok, err = password.VerifyPassword(oldPassword, stored.PasswordHash)
	if err != nil {
		t.Fatalf("VerifyPassword (old): %v", err)
	}
	if ok {
		t.Fatal("old password must no longer verify after reactivation")
	}
}

func TestRegisterReactivationFailure(t *testing.T) {
	svc := registration.NewRegistrationService(reactivationFailRepo{}, password.Hasher{})

	_, err := svc.Register(context.Background(), registration.RegisterInput{
		Email:     "reactivate-fail@ryze.local",
		Password:  "Password123!",
		FirstName: "First",
		LastName:  "Last",
	})
	if !errors.Is(err, errRepositoryFailure) {
		t.Fatalf("Register: expected reactivation repository error to propagate, got %v", err)
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
func (failingRepo) FindByEmailIncludingDeleted(_ context.Context, _ string) (*models.User, error) {
	return nil, errRepositoryFailure
}
func (failingRepo) GetSessionVersion(_ context.Context, _ string) (int, error) {
	return 0, errRepositoryFailure
}
func (failingRepo) Update(_ context.Context, _ *models.User) error {
	return errRepositoryFailure
}
func (failingRepo) ChangePassword(_ context.Context, _ string, _ string) error {
	return errRepositoryFailure
}
func (failingRepo) Reactivate(_ context.Context, _ *models.User) error {
	return errRepositoryFailure
}
func (failingRepo) SoftDelete(_ context.Context, _ string) error {
	return errRepositoryFailure
}
func (failingRepo) DeleteAccount(_ context.Context, _ string) error {
	return errRepositoryFailure
}

// reactivationFailRepo returns a soft-deleted user on lookup but fails when the
// reactivation update is performed.
type reactivationFailRepo struct{}

func (reactivationFailRepo) Create(_ context.Context, _ *models.User) error {
	return errRepositoryFailure
}
func (reactivationFailRepo) FindByID(_ context.Context, _ string) (*models.User, error) {
	return nil, errRepositoryFailure
}
func (reactivationFailRepo) FindByEmail(_ context.Context, _ string) (*models.User, error) {
	return nil, errRepositoryFailure
}
func (reactivationFailRepo) FindByEmailIncludingDeleted(_ context.Context, _ string) (*models.User, error) {
	return &models.User{DeletedAt: gorm.DeletedAt{Valid: true}}, nil
}
func (reactivationFailRepo) GetSessionVersion(_ context.Context, _ string) (int, error) {
	return 0, errRepositoryFailure
}
func (reactivationFailRepo) Update(_ context.Context, _ *models.User) error {
	return errRepositoryFailure
}
func (reactivationFailRepo) ChangePassword(_ context.Context, _ string, _ string) error {
	return errRepositoryFailure
}
func (reactivationFailRepo) Reactivate(_ context.Context, _ *models.User) error {
	return errRepositoryFailure
}
func (reactivationFailRepo) SoftDelete(_ context.Context, _ string) error {
	return errRepositoryFailure
}
func (reactivationFailRepo) DeleteAccount(_ context.Context, _ string) error {
	return errRepositoryFailure
}
