package test_mode_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/test_mode"
)

// --- fakes ---

type fakeSessions struct {
	byHash map[string]*models.TestSession
	byID   map[string]*models.TestSession
}

func newFakeSessions() *fakeSessions {
	return &fakeSessions{byHash: map[string]*models.TestSession{}, byID: map[string]*models.TestSession{}}
}

func (f *fakeSessions) Create(_ context.Context, session *models.TestSession) error {
	if session.ID == "" {
		session.ID = "session-id"
	}
	f.byHash[session.TokenHash] = session
	f.byID[session.ID] = session
	return nil
}

func (f *fakeSessions) FindActiveByTokenHash(_ context.Context, tokenHash string) (*models.TestSession, error) {
	session, ok := f.byHash[tokenHash]
	if !ok {
		return nil, repositories.ErrTestSessionNotFound
	}
	return session, nil
}

func (f *fakeSessions) SoftDelete(_ context.Context, id string) error {
	session, ok := f.byID[id]
	if !ok {
		return repositories.ErrTestSessionNotFound
	}
	delete(f.byHash, session.TokenHash)
	delete(f.byID, id)
	return nil
}

type fakeUsers struct {
	users map[string]*models.User
	calls int
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{users: map[string]*models.User{}}
}

func (f *fakeUsers) FindByEmail(_ context.Context, email string) (*models.User, error) {
	user, ok := f.users[email]
	if !ok {
		return nil, repositories.ErrUserNotFound
	}
	return user, nil
}

func (f *fakeUsers) Create(_ context.Context, user *models.User) error {
	f.calls++
	if _, ok := f.users[user.Email]; ok {
		return repositories.ErrDuplicateEmail
	}
	user.ID = "user-" + user.Email
	f.users[user.Email] = user
	return nil
}

type fakeTrainers struct {
	trainers map[string]*models.Trainer
}

func newFakeTrainers() *fakeTrainers {
	return &fakeTrainers{trainers: map[string]*models.Trainer{}}
}

func (f *fakeTrainers) FindByUserID(_ context.Context, userID string) (*models.Trainer, error) {
	trainer, ok := f.trainers[userID]
	if !ok {
		return nil, repositories.ErrTrainerNotFound
	}
	return trainer, nil
}

func (f *fakeTrainers) Create(_ context.Context, trainer *models.Trainer) error {
	trainer.ID = "trainer-id"
	f.trainers[trainer.UserID] = trainer
	return nil
}

type fakeHasher struct{}

func (fakeHasher) HashPassword(password string) (string, error) {
	return "hashed:" + password, nil
}

func newService(enabled bool) (test_mode.Service, *fakeSessions, *fakeUsers, *fakeTrainers) {
	sessions := newFakeSessions()
	users := newFakeUsers()
	trainers := newFakeTrainers()
	return test_mode.NewService(enabled, sessions, users, trainers, fakeHasher{}), sessions, users, trainers
}

// --- tests ---

func TestEnterRejectsWhenDisabled(t *testing.T) {
	svc, _, _, _ := newService(false)

	_, err := svc.Enter(context.Background(), "ADMIN_1", test_mode.PersonaClient, "/admin/test-mode")
	if !errors.Is(err, test_mode.ErrDisabled) {
		t.Fatalf("expected ErrDisabled, got %v", err)
	}
}

func TestEnterRejectsInvalidAdminIdentity(t *testing.T) {
	svc, _, _, _ := newService(true)

	for _, identity := range []string{"", "not-an-admin", "ADMIN_3"} {
		if _, err := svc.Enter(context.Background(), identity, test_mode.PersonaClient, ""); !errors.Is(err, test_mode.ErrInvalidInput) {
			t.Fatalf("identity %q: expected ErrInvalidInput, got %v", identity, err)
		}
	}
}

func TestEnterRejectsUnknownPersona(t *testing.T) {
	svc, _, _, _ := newService(true)

	for _, persona := range []string{"", "admin", "guest", "ADMIN_1"} {
		if _, err := svc.Enter(context.Background(), "ADMIN_1", persona, ""); !errors.Is(err, test_mode.ErrUnknownPersona) {
			t.Fatalf("persona %q: expected ErrUnknownPersona, got %v", persona, err)
		}
	}
}

func TestEnterSanitizesReturnPath(t *testing.T) {
	svc, _, _, _ := newService(true)

	invalid := []string{
		"https://evil.example.com/phish",
		"http://evil.example.com",
		"//evil.example.com/phish",
		"/admin\\..\\..",
		"/admin\x00",
		"/admin/test-mode\nX-Injected: 1",
		strings.Repeat("/", 600),
		"evil.com",
	}
	for _, path := range invalid {
		if _, err := svc.Enter(context.Background(), "ADMIN_1", test_mode.PersonaClient, path); !errors.Is(err, test_mode.ErrInvalidReturnPath) {
			t.Fatalf("path %q: expected ErrInvalidReturnPath, got %v", path, err)
		}
	}

	valid := []string{"", "/admin/test-mode", "/admin", "/services/generic-program"}
	for _, path := range valid {
		result, err := svc.Enter(context.Background(), "ADMIN_1", test_mode.PersonaClient, path)
		if err != nil {
			t.Fatalf("path %q: expected valid, got %v", path, err)
		}
		if result.ReturnPath != path && !(path == "" && result.ReturnPath == "") {
			t.Fatalf("path %q: expected return path to be preserved, got %q", path, result.ReturnPath)
		}
	}
}

func TestEnterStoresOnlyHashedTokenAndCreatesPersona(t *testing.T) {
	svc, sessions, users, trainers := newService(true)
	ctx := context.Background()

	result, err := svc.Enter(ctx, "ADMIN_1", test_mode.PersonaTrainer, "/admin/test-mode")
	if err != nil {
		t.Fatalf("Enter: %v", err)
	}

	if result.Token == "" {
		t.Fatal("expected a session token")
	}
	if len(result.Token) != 64 {
		t.Fatalf("expected a 64 character token, got %d", len(result.Token))
	}

	sum := sha256.Sum256([]byte(result.Token))
	expectedHash := hex.EncodeToString(sum[:])
	stored, ok := sessions.byHash[expectedHash]
	if !ok {
		t.Fatal("expected the session to be stored under the SHA-256 of the token")
	}
	for _, session := range sessions.byHash {
		if session.TokenHash == result.Token {
			t.Fatal("raw token must never be stored")
		}
	}

	if result.PersonaUserID == "" || result.PersonaUserID != stored.PersonaUserID {
		t.Fatal("expected the session to be bound to the persona user")
	}
	if stored.AdminIdentity != "ADMIN_1" {
		t.Fatalf("expected the admin identity to be recorded, got %q", stored.AdminIdentity)
	}

	// The persona account is created on demand with a hashed random password.
	user, ok := users.users[test_mode.PersonaTrainerEmail]
	if !ok {
		t.Fatal("expected the trainer persona account to exist")
	}
	if user.PasswordHash == "" || strings.HasPrefix(user.PasswordHash, "hashed:hashed") {
		t.Fatal("expected a single hashed password")
	}
	// The trainer persona also owns a trainer profile row.
	if _, ok := trainers.trainers[user.ID]; !ok {
		t.Fatal("expected the trainer persona to own a trainer profile")
	}
}

func TestEnterReusesPersonaAccount(t *testing.T) {
	svc, _, users, _ := newService(true)
	ctx := context.Background()

	if _, err := svc.Enter(ctx, "ADMIN_1", test_mode.PersonaClient, ""); err != nil {
		t.Fatalf("first Enter: %v", err)
	}
	firstID := users.users[test_mode.PersonaClientEmail].ID

	if _, err := svc.Enter(ctx, "ADMIN_1", test_mode.PersonaClient, ""); err != nil {
		t.Fatalf("second Enter: %v", err)
	}
	if users.users[test_mode.PersonaClientEmail].ID != firstID {
		t.Fatal("expected the same persona account to be reused across sessions")
	}
	if users.calls != 1 {
		t.Fatalf("expected the persona account to be created once, got %d creations", users.calls)
	}
}

func TestExitRestoresAdminContext(t *testing.T) {
	svc, _, _, _ := newService(true)
	ctx := context.Background()

	entered, err := svc.Enter(ctx, "ADMIN_1", test_mode.PersonaClient, "/admin/test-mode")
	if err != nil {
		t.Fatalf("Enter: %v", err)
	}

	exited, err := svc.Exit(ctx, entered.Token)
	if err != nil {
		t.Fatalf("Exit: %v", err)
	}
	if exited.AdminIdentity != "ADMIN_1" {
		t.Fatalf("expected the original admin identity, got %q", exited.AdminIdentity)
	}
	if exited.ReturnPath != "/admin/test-mode" {
		t.Fatalf("expected the recorded return path, got %q", exited.ReturnPath)
	}

	if _, err := svc.ActivePersona(ctx, entered.Token); !errors.Is(err, test_mode.ErrNoActiveSession) {
		t.Fatalf("expected ErrNoActiveSession after exit, got %v", err)
	}
	if _, err := svc.Exit(ctx, entered.Token); !errors.Is(err, test_mode.ErrNoActiveSession) {
		t.Fatalf("expected a second exit to fail with ErrNoActiveSession, got %v", err)
	}
}

func TestExitAndStatusRejectUnknownToken(t *testing.T) {
	svc, _, _, _ := newService(true)
	ctx := context.Background()

	if _, err := svc.Exit(ctx, ""); !errors.Is(err, test_mode.ErrNoActiveSession) {
		t.Fatalf("expected ErrNoActiveSession for an empty token, got %v", err)
	}
	if _, err := svc.Exit(ctx, strings.Repeat("a", 64)); !errors.Is(err, test_mode.ErrNoActiveSession) {
		t.Fatalf("expected ErrNoActiveSession for an unknown token, got %v", err)
	}
	if _, err := svc.ActivePersona(ctx, strings.Repeat("a", 64)); !errors.Is(err, test_mode.ErrNoActiveSession) {
		t.Fatalf("expected ErrNoActiveSession for an unknown token, got %v", err)
	}
}

func TestActivePersonaReturnsSessionData(t *testing.T) {
	svc, _, _, _ := newService(true)
	ctx := context.Background()

	entered, err := svc.Enter(ctx, "ADMIN_1", test_mode.PersonaTrainer, "/admin/test-mode")
	if err != nil {
		t.Fatalf("Enter: %v", err)
	}

	session, err := svc.ActivePersona(ctx, entered.Token)
	if err != nil {
		t.Fatalf("ActivePersona: %v", err)
	}
	if session.AdminIdentity != "ADMIN_1" {
		t.Fatalf("expected admin identity ADMIN_1, got %q", session.AdminIdentity)
	}
	if session.Persona != test_mode.PersonaTrainer {
		t.Fatalf("expected trainer persona, got %q", session.Persona)
	}
	if session.PersonaUserID != entered.PersonaUserID {
		t.Fatalf("expected persona user %q, got %q", entered.PersonaUserID, session.PersonaUserID)
	}
	if session.ReturnPath != "/admin/test-mode" {
		t.Fatalf("expected return path, got %q", session.ReturnPath)
	}
}

func TestIsValidPersonaWhitelist(t *testing.T) {
	allowed := []string{test_mode.PersonaClient, test_mode.PersonaTrainer}
	for _, persona := range allowed {
		if !test_mode.IsValidPersona(persona) {
			t.Fatalf("expected %q to be allowed", persona)
		}
	}
	for _, persona := range []string{"", "user", "ADMIN_1", "trainer2"} {
		if test_mode.IsValidPersona(persona) {
			t.Fatalf("expected %q to be rejected", persona)
		}
	}
}
