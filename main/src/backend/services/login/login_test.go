package login_test

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
	"ryze/backend/services/login"
	"ryze/backend/services/password"
)

func TestLoginSuccess(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	svc := login.NewLoginService(repo, password.Verifier{})
	ctx := context.Background()
	email := fmt.Sprintf("login-test-%d@ryze.local", time.Now().UnixNano())
	plaintext := "SuperSecret123!"
	seedUser(t, repo, email, plaintext)

	user, err := svc.Login(ctx, login.LoginInput{Email: email, Password: plaintext})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if user.ID == "" {
		t.Fatal("Login: expected the authenticated user id")
	}
	if user.Email != email {
		t.Fatalf("Login: expected email %q, got %q", email, user.Email)
	}
	if user.PasswordHash != "" {
		t.Fatal("Login: result must not expose password_hash")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	svc := login.NewLoginService(repo, password.Verifier{})
	ctx := context.Background()
	email := fmt.Sprintf("login-wrong-%d@ryze.local", time.Now().UnixNano())
	seedUser(t, repo, email, "CorrectPassword1!")

	_, err := svc.Login(ctx, login.LoginInput{Email: email, Password: "WrongPassword1!"})
	if !errors.Is(err, login.ErrInvalidCredentials) {
		t.Fatalf("Login: expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginNonExistentEmail(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	svc := login.NewLoginService(repo, password.Verifier{})

	_, err := svc.Login(context.Background(), login.LoginInput{
		Email:    fmt.Sprintf("missing-%d@ryze.local", time.Now().UnixNano()),
		Password: "Whatever123!",
	})
	if !errors.Is(err, login.ErrInvalidCredentials) {
		t.Fatalf("Login: expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginUnknownEmailAndWrongPasswordAreIndistinguishable(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	svc := login.NewLoginService(repo, password.Verifier{})
	ctx := context.Background()
	email := fmt.Sprintf("login-same-%d@ryze.local", time.Now().UnixNano())
	seedUser(t, repo, email, "CorrectPassword1!")

	_, errWrongPassword := svc.Login(ctx, login.LoginInput{Email: email, Password: "WrongPassword1!"})
	_, errUnknownEmail := svc.Login(ctx, login.LoginInput{
		Email:    fmt.Sprintf("nobody-%d@ryze.local", time.Now().UnixNano()),
		Password: "WrongPassword1!",
	})
	if !errors.Is(errWrongPassword, errUnknownEmail) {
		t.Fatal("Login: wrong password and unknown email must produce the exact same error")
	}
	if errWrongPassword == nil || errUnknownEmail == nil {
		t.Fatal("Login: both attempts must fail")
	}
}

func TestLoginSoftDeletedUser(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	svc := login.NewLoginService(repo, password.Verifier{})
	ctx := context.Background()
	email := fmt.Sprintf("login-deleted-%d@ryze.local", time.Now().UnixNano())
	user := seedUser(t, repo, email, "SuperSecret123!")

	if err := repo.SoftDelete(ctx, user.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	_, err := svc.Login(ctx, login.LoginInput{Email: email, Password: "SuperSecret123!"})
	if !errors.Is(err, login.ErrInvalidCredentials) {
		t.Fatalf("Login: soft-deleted user must fail with ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginEmptyInput(t *testing.T) {
	repo, close := newTestRepository(t)
	defer close()

	svc := login.NewLoginService(repo, password.Verifier{})

	_, err := svc.Login(context.Background(), login.LoginInput{Email: "", Password: "Password123!"})
	if !errors.Is(err, login.ErrInvalidInput) {
		t.Fatalf("Login: empty email must return ErrInvalidInput, got %v", err)
	}

	_, err = svc.Login(context.Background(), login.LoginInput{Email: "a@b.local", Password: ""})
	if !errors.Is(err, password.ErrEmptyPassword) {
		t.Fatalf("Login: empty password must return ErrEmptyPassword, got %v", err)
	}
}

func TestLoginRepositoryFailure(t *testing.T) {
	svc := login.NewLoginService(failingRepo{}, password.Verifier{})

	_, err := svc.Login(context.Background(), login.LoginInput{Email: "repo@ryze.local", Password: "Password123!"})
	if !errors.Is(err, errRepositoryFailure) {
		t.Fatalf("Login: expected repository failure to propagate, got %v", err)
	}
	if errors.Is(err, login.ErrInvalidCredentials) {
		t.Fatal("Login: repository failure must not be reported as ErrInvalidCredentials")
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
func (failingRepo) Update(_ context.Context, _ *models.User) error {
	return errRepositoryFailure
}
func (failingRepo) SoftDelete(_ context.Context, _ string) error {
	return errRepositoryFailure
}
