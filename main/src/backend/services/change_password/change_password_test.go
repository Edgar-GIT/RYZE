package change_password_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/change_password"
	"ryze/backend/services/login"
	"ryze/backend/services/password"
)

func TestChangePasswordSuccess(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	ctx := context.Background()
	email := fmt.Sprintf("change-success-%d@ryze.local", time.Now().UnixNano())
	oldPassword := "OldPassword123!"
	newPassword := "NewPassword456!"
	user := seedUser(t, repo, email, oldPassword)

	svc := change_password.NewChangePasswordService(repo, password.Verifier{}, password.Hasher{})
	result, err := svc.ChangePassword(ctx, change_password.Input{
		UserID:          user.ID,
		CurrentPassword: oldPassword,
		NewPassword:     newPassword,
	})
	if err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
	if result.ID != user.ID {
		t.Fatalf("ChangePassword: expected id %q, got %q", user.ID, result.ID)
	}
	if result.PasswordHash != "" {
		t.Fatal("ChangePassword: returned user must not expose password_hash")
	}

	// The stored hash now verifies the new password and not the old one.
	stored, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	ok, err := password.VerifyPassword(newPassword, stored.PasswordHash)
	if err != nil {
		t.Fatalf("VerifyPassword (new): %v", err)
	}
	if !ok {
		t.Fatal("new password must verify after the change")
	}
	ok, err = password.VerifyPassword(oldPassword, stored.PasswordHash)
	if err != nil {
		t.Fatalf("VerifyPassword (old): %v", err)
	}
	if ok {
		t.Fatal("old password must no longer verify after the change")
	}

	// Login with the old password fails and the new one succeeds.
	loginSvc := login.NewLoginService(repo, password.Verifier{})
	if _, err := loginSvc.Login(ctx, login.LoginInput{Email: email, Password: oldPassword}); !errors.Is(err, login.ErrInvalidCredentials) {
		t.Fatalf("Login with old password: expected ErrInvalidCredentials, got %v", err)
	}
	if _, err := loginSvc.Login(ctx, login.LoginInput{Email: email, Password: newPassword}); err != nil {
		t.Fatalf("Login with new password: %v", err)
	}
}

func TestChangePasswordBumpsSessionVersion(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	ctx := context.Background()
	user := seedUser(t, repo, fmt.Sprintf("change-version-%d@ryze.local", time.Now().UnixNano()), "OldPassword123!")

	if v, err := repo.GetSessionVersion(ctx, user.ID); err != nil {
		t.Fatalf("GetSessionVersion (initial): %v", err)
	} else if v != 0 {
		t.Fatalf("expected initial session version 0, got %d", v)
	}

	svc := change_password.NewChangePasswordService(repo, password.Verifier{}, password.Hasher{})
	if _, err := svc.ChangePassword(ctx, change_password.Input{
		UserID:          user.ID,
		CurrentPassword: "OldPassword123!",
		NewPassword:     "NewPassword456!",
	}); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}

	if v, err := repo.GetSessionVersion(ctx, user.ID); err != nil {
		t.Fatalf("GetSessionVersion (after): %v", err)
	} else if v != 1 {
		t.Fatalf("expected session version 1 after the change, got %d", v)
	}
}

func TestChangePasswordWrongCurrentPassword(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	user := seedUser(t, repo, fmt.Sprintf("change-wrong-%d@ryze.local", time.Now().UnixNano()), "OldPassword123!")

	svc := change_password.NewChangePasswordService(repo, password.Verifier{}, password.Hasher{})
	_, err := svc.ChangePassword(context.Background(), change_password.Input{
		UserID:          user.ID,
		CurrentPassword: "WrongPassword123!",
		NewPassword:     "NewPassword456!",
	})
	if !errors.Is(err, change_password.ErrInvalidCredentials) {
		t.Fatalf("ChangePassword: expected ErrInvalidCredentials, got %v", err)
	}
}

func TestChangePasswordUnknownUser(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	svc := change_password.NewChangePasswordService(repo, password.Verifier{}, password.Hasher{})
	_, err := svc.ChangePassword(context.Background(), change_password.Input{
		UserID:          "00000000-0000-0000-0000-000000000000",
		CurrentPassword: "OldPassword123!",
		NewPassword:     "NewPassword456!",
	})
	if !errors.Is(err, change_password.ErrInvalidCredentials) {
		t.Fatalf("ChangePassword: expected ErrInvalidCredentials, got %v", err)
	}
}

func TestChangePasswordSoftDeletedUser(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	ctx := context.Background()
	user := seedUser(t, repo, fmt.Sprintf("change-deleted-%d@ryze.local", time.Now().UnixNano()), "OldPassword123!")
	if err := repo.SoftDelete(ctx, user.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	svc := change_password.NewChangePasswordService(repo, password.Verifier{}, password.Hasher{})
	_, err := svc.ChangePassword(ctx, change_password.Input{
		UserID:          user.ID,
		CurrentPassword: "OldPassword123!",
		NewPassword:     "NewPassword456!",
	})
	if !errors.Is(err, change_password.ErrInvalidCredentials) {
		t.Fatalf("ChangePassword: expected ErrInvalidCredentials, got %v", err)
	}
}

func TestChangePasswordUnverifiableHash(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	// A corrupted stored hash must behave like invalid credentials and never
	// surface the corruption cause.
	user := &models.User{
		Email:        fmt.Sprintf("change-badhash-%d@ryze.local", time.Now().UnixNano()),
		PasswordHash: "not-a-valid-argon2-hash",
		FirstName:    "Bad",
		LastName:     "Hash",
	}
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	svc := change_password.NewChangePasswordService(repo, password.Verifier{}, password.Hasher{})
	_, err := svc.ChangePassword(context.Background(), change_password.Input{
		UserID:          user.ID,
		CurrentPassword: "Whatever123!",
		NewPassword:     "NewPassword456!",
	})
	if !errors.Is(err, change_password.ErrInvalidCredentials) {
		t.Fatalf("ChangePassword: expected ErrInvalidCredentials, got %v", err)
	}
}

func TestChangePasswordWrongCurrentAndUnknownAreIndistinguishable(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	user := seedUser(t, repo, fmt.Sprintf("change-same-%d@ryze.local", time.Now().UnixNano()), "OldPassword123!")
	svc := change_password.NewChangePasswordService(repo, password.Verifier{}, password.Hasher{})
	ctx := context.Background()

	_, errWrongCurrent := svc.ChangePassword(ctx, change_password.Input{
		UserID:          user.ID,
		CurrentPassword: "WrongPassword123!",
		NewPassword:     "NewPassword456!",
	})
	_, errUnknown := svc.ChangePassword(ctx, change_password.Input{
		UserID:          "00000000-0000-0000-0000-000000000000",
		CurrentPassword: "OldPassword123!",
		NewPassword:     "NewPassword456!",
	})
	if !errors.Is(errWrongCurrent, errUnknown) {
		t.Fatal("wrong current password and unknown user must produce the exact same error")
	}
	if errWrongCurrent == nil || errUnknown == nil {
		t.Fatal("both attempts must fail")
	}
}

func TestChangePasswordEmptyInput(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	user := seedUser(t, repo, fmt.Sprintf("change-empty-%d@ryze.local", time.Now().UnixNano()), "OldPassword123!")
	svc := change_password.NewChangePasswordService(repo, password.Verifier{}, password.Hasher{})

	_, err := svc.ChangePassword(context.Background(), change_password.Input{
		UserID:          user.ID,
		CurrentPassword: "",
		NewPassword:     "NewPassword456!",
	})
	if !errors.Is(err, password.ErrEmptyPassword) {
		t.Fatalf("empty current password: expected ErrEmptyPassword, got %v", err)
	}

	_, err = svc.ChangePassword(context.Background(), change_password.Input{
		UserID:          user.ID,
		CurrentPassword: "OldPassword123!",
		NewPassword:     "",
	})
	if !errors.Is(err, password.ErrEmptyPassword) {
		t.Fatalf("empty new password: expected ErrEmptyPassword, got %v", err)
	}

	_, err = svc.ChangePassword(context.Background(), change_password.Input{
		UserID:          "",
		CurrentPassword: "OldPassword123!",
		NewPassword:     "NewPassword456!",
	})
	if !errors.Is(err, change_password.ErrInvalidInput) {
		t.Fatalf("empty user id: expected ErrInvalidInput, got %v", err)
	}
}

func TestChangePasswordRepositoryFailure(t *testing.T) {
	svc := change_password.NewChangePasswordService(failingRepo{}, password.Verifier{}, password.Hasher{})

	_, err := svc.ChangePassword(context.Background(), change_password.Input{
		UserID:          "00000000-0000-0000-0000-000000000000",
		CurrentPassword: "OldPassword123!",
		NewPassword:     "NewPassword456!",
	})
	if !errors.Is(err, errRepositoryFailure) {
		t.Fatalf("ChangePassword: expected repository failure to propagate, got %v", err)
	}
	if errors.Is(err, change_password.ErrInvalidCredentials) {
		t.Fatal("repository failure must not be reported as ErrInvalidCredentials")
	}
}

func seedUser(t *testing.T, repo repositories.UserRepository, email, plaintext string) *models.User {
	t.Helper()

	hash, err := password.HashPassword(plaintext)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	user := &models.User{
		Email:        email,
		PasswordHash: hash,
		FirstName:    "Test",
		LastName:     "User",
	}
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user
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
func (failingRepo) ListActive(_ context.Context, _ int, _ int) ([]models.User, int64, error) {
	return nil, 0, errRepositoryFailure
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
