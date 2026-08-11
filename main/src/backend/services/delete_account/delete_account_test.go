package delete_account_test

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
	"ryze/backend/services/delete_account"
	"ryze/backend/services/login"
	"ryze/backend/services/password"
	"ryze/backend/services/registration"
)

func TestDeleteAccountSuccess(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	ctx := context.Background()
	email := fmt.Sprintf("delete-success-%d@ryze.local", time.Now().UnixNano())
	user := seedUser(t, repo, email, "Password123!")
	originalHash := user.PasswordHash
	if originalHash == "" {
		t.Fatal("seed user must carry a password hash")
	}

	svc := delete_account.NewDeleteAccountService(repo, password.Verifier{})
	if err := svc.DeleteAccount(ctx, delete_account.Input{UserID: user.ID, Password: "Password123!"}); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}

	// The account is gone from normal lookups but the row still exists.
	if _, err := repo.FindByID(ctx, user.ID); !errors.Is(err, repositories.ErrUserNotFound) {
		t.Fatalf("FindByID: expected ErrUserNotFound after deletion, got %v", err)
	}
	stored, err := repo.FindByEmailIncludingDeleted(ctx, email)
	if err != nil {
		t.Fatalf("FindByEmailIncludingDeleted: %v", err)
	}
	if !stored.DeletedAt.Valid {
		t.Fatal("deleted_at must be set after account deletion")
	}

	// id, email, created_at and password_hash are preserved.
	if stored.ID != user.ID {
		t.Fatalf("id must be preserved: expected %q, got %q", user.ID, stored.ID)
	}
	if stored.Email != email {
		t.Fatalf("email must be preserved: expected %q, got %q", email, stored.Email)
	}
	if !stored.CreatedAt.Equal(user.CreatedAt) {
		t.Fatalf("created_at must be preserved: expected %v, got %v", user.CreatedAt, stored.CreatedAt)
	}
	if stored.PasswordHash != originalHash {
		t.Fatal("password_hash must be unchanged by deletion")
	}
	if stored.SessionVersion != 1 {
		t.Fatalf("expected session version 1 after deletion, got %d", stored.SessionVersion)
	}

	// The deleted account cannot log in anymore.
	loginSvc := login.NewLoginService(repo, password.Verifier{})
	if _, err := loginSvc.Login(ctx, login.LoginInput{Email: email, Password: "Password123!"}); !errors.Is(err, login.ErrInvalidCredentials) {
		t.Fatalf("Login: expected ErrInvalidCredentials for a deleted account, got %v", err)
	}
}

func TestDeleteAccountAtomicSessionVersionBump(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	ctx := context.Background()
	user := seedUser(t, repo, fmt.Sprintf("delete-version-%d@ryze.local", time.Now().UnixNano()), "Password123!")
	if v, err := repo.GetSessionVersion(ctx, user.ID); err != nil {
		t.Fatalf("GetSessionVersion (initial): %v", err)
	} else if v != 0 {
		t.Fatalf("expected initial session version 0, got %d", v)
	}

	svc := delete_account.NewDeleteAccountService(repo, password.Verifier{})
	if err := svc.DeleteAccount(ctx, delete_account.Input{UserID: user.ID, Password: "Password123!"}); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}

	stored, err := repo.FindByEmailIncludingDeleted(ctx, user.Email)
	if err != nil {
		t.Fatalf("FindByEmailIncludingDeleted: %v", err)
	}
	if stored.SessionVersion != 1 {
		t.Fatalf("expected session version 1 after deletion, got %d", stored.SessionVersion)
	}
}

func TestDeleteAccountWrongPassword(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	ctx := context.Background()
	user := seedUser(t, repo, fmt.Sprintf("delete-wrong-%d@ryze.local", time.Now().UnixNano()), "Password123!")

	svc := delete_account.NewDeleteAccountService(repo, password.Verifier{})
	err := svc.DeleteAccount(ctx, delete_account.Input{UserID: user.ID, Password: "WrongPassword123!"})
	if !errors.Is(err, delete_account.ErrInvalidCredentials) {
		t.Fatalf("DeleteAccount: expected ErrInvalidCredentials, got %v", err)
	}

	// Nothing changed: the account still exists and can log in.
	if _, err := repo.FindByID(ctx, user.ID); err != nil {
		t.Fatalf("account must remain active after a wrong-password attempt: %v", err)
	}
	loginSvc := login.NewLoginService(repo, password.Verifier{})
	if _, err := loginSvc.Login(ctx, login.LoginInput{Email: user.Email, Password: "Password123!"}); err != nil {
		t.Fatalf("login must still work after a wrong-password attempt: %v", err)
	}
}

func TestDeleteAccountUnknownUser(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	svc := delete_account.NewDeleteAccountService(repo, password.Verifier{})
	err := svc.DeleteAccount(context.Background(), delete_account.Input{
		UserID:   "00000000-0000-0000-0000-000000000000",
		Password: "Password123!",
	})
	if !errors.Is(err, delete_account.ErrInvalidCredentials) {
		t.Fatalf("DeleteAccount: expected ErrInvalidCredentials, got %v", err)
	}
}

func TestDeleteAccountAlreadyDeleted(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	ctx := context.Background()
	user := seedUser(t, repo, fmt.Sprintf("delete-twice-%d@ryze.local", time.Now().UnixNano()), "Password123!")
	if err := repo.SoftDelete(ctx, user.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	svc := delete_account.NewDeleteAccountService(repo, password.Verifier{})
	err := svc.DeleteAccount(ctx, delete_account.Input{UserID: user.ID, Password: "Password123!"})
	if !errors.Is(err, delete_account.ErrInvalidCredentials) {
		t.Fatalf("DeleteAccount: expected ErrInvalidCredentials for a deleted account, got %v", err)
	}
}

func TestDeleteAccountUnverifiableHash(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	user := &models.User{
		Email:        fmt.Sprintf("delete-badhash-%d@ryze.local", time.Now().UnixNano()),
		PasswordHash: "not-a-valid-argon2-hash",
		FirstName:    "Bad",
		LastName:     "Hash",
	}
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	svc := delete_account.NewDeleteAccountService(repo, password.Verifier{})
	err := svc.DeleteAccount(context.Background(), delete_account.Input{UserID: user.ID, Password: "Password123!"})
	if !errors.Is(err, delete_account.ErrInvalidCredentials) {
		t.Fatalf("DeleteAccount: expected ErrInvalidCredentials, got %v", err)
	}
}

func TestDeleteAccountWrongPasswordAndUnknownAreIndistinguishable(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	user := seedUser(t, repo, fmt.Sprintf("delete-same-%d@ryze.local", time.Now().UnixNano()), "Password123!")
	svc := delete_account.NewDeleteAccountService(repo, password.Verifier{})
	ctx := context.Background()

	errWrongPassword := svc.DeleteAccount(ctx, delete_account.Input{UserID: user.ID, Password: "WrongPassword123!"})
	errUnknown := svc.DeleteAccount(ctx, delete_account.Input{
		UserID:   "00000000-0000-0000-0000-000000000000",
		Password: "Password123!",
	})
	if !errors.Is(errWrongPassword, errUnknown) {
		t.Fatal("wrong password and unknown user must produce the exact same error")
	}
	if errWrongPassword == nil || errUnknown == nil {
		t.Fatal("both attempts must fail")
	}
}

func TestDeleteAccountEmptyInput(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	user := seedUser(t, repo, fmt.Sprintf("delete-empty-%d@ryze.local", time.Now().UnixNano()), "Password123!")
	svc := delete_account.NewDeleteAccountService(repo, password.Verifier{})
	ctx := context.Background()

	err := svc.DeleteAccount(ctx, delete_account.Input{UserID: user.ID, Password: ""})
	if !errors.Is(err, password.ErrEmptyPassword) {
		t.Fatalf("empty password: expected ErrEmptyPassword, got %v", err)
	}

	err = svc.DeleteAccount(ctx, delete_account.Input{UserID: "", Password: "Password123!"})
	if !errors.Is(err, delete_account.ErrInvalidInput) {
		t.Fatalf("empty user id: expected ErrInvalidInput, got %v", err)
	}
}

func TestDeleteAccountRepositoryFailure(t *testing.T) {
	svc := delete_account.NewDeleteAccountService(failingRepo{}, password.Verifier{})

	err := svc.DeleteAccount(context.Background(), delete_account.Input{
		UserID:   "00000000-0000-0000-0000-000000000000",
		Password: "Password123!",
	})
	if !errors.Is(err, errRepositoryFailure) {
		t.Fatalf("DeleteAccount: expected repository failure to propagate, got %v", err)
	}
	if errors.Is(err, delete_account.ErrInvalidCredentials) {
		t.Fatal("repository failure must not be reported as ErrInvalidCredentials")
	}
}

func TestDeleteAccountPreservesRowForReactivation(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	ctx := context.Background()
	email := fmt.Sprintf("delete-react-%d@ryze.local", time.Now().UnixNano())
	user := seedUser(t, repo, email, "Password123!")

	svc := delete_account.NewDeleteAccountService(repo, password.Verifier{})
	if err := svc.DeleteAccount(ctx, delete_account.Input{UserID: user.ID, Password: "Password123!"}); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}

	// Re-registering with the same email restores the original row (same UUID).
	regSvc := registration.NewRegistrationService(repo, password.Hasher{})
	reactivated, err := regSvc.Register(ctx, registration.RegisterInput{
		Email:     email,
		Password:  "NewPassword456!",
		FirstName: "New",
		LastName:  "Name",
	})
	if err != nil {
		t.Fatalf("Register (reactivation): %v", err)
	}
	if reactivated.ID != user.ID {
		t.Fatalf("reactivation must preserve the UUID: expected %q, got %q", user.ID, reactivated.ID)
	}

	stored, err := repo.FindByEmailIncludingDeleted(ctx, email)
	if err != nil {
		t.Fatalf("FindByEmailIncludingDeleted: %v", err)
	}
	if stored.DeletedAt.Valid {
		t.Fatal("reactivated account must have a cleared deleted_at in the database")
	}

	loginSvc := login.NewLoginService(repo, password.Verifier{})
	if _, err := loginSvc.Login(ctx, login.LoginInput{Email: email, Password: "NewPassword456!"}); err != nil {
		t.Fatalf("reactivated account must log in: %v", err)
	}
	if _, err := loginSvc.Login(ctx, login.LoginInput{Email: email, Password: "Password123!"}); !errors.Is(err, login.ErrInvalidCredentials) {
		t.Fatalf("old password must not work after reactivation, got %v", err)
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
