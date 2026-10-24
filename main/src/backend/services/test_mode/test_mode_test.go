package test_mode_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

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

func (f *fakeUsers) FindByID(_ context.Context, id string) (*models.User, error) {
	for _, user := range f.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, repositories.ErrUserNotFound
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

func newServiceWithTTL(enabled bool, ttl time.Duration) (test_mode.Service, *fakeSessions, *fakeUsers, *fakeTrainers) {
	sessions := newFakeSessions()
	users := newFakeUsers()
	trainers := newFakeTrainers()
	return test_mode.NewService(enabled, sessions, users, trainers, fakeHasher{}, test_mode.WithSessionTTL(ttl)), sessions, users, trainers
}

type failingSessionCreator struct {
	*fakeSessions
	fail bool
}

func (f *failingSessionCreator) Create(ctx context.Context, session *models.TestSession) error {
	if f.fail {
		f.fail = false
		return errors.New("session persistence failed")
	}
	return f.fakeSessions.Create(ctx, session)
}

// racingUsers simulates a concurrent writer that creates an unowned account
// with the persona address between the initial lookup and the insert.
type racingUsers struct {
	*fakeUsers
	findCalls int
}

func (f *racingUsers) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	f.findCalls++
	if f.findCalls == 1 {
		return nil, repositories.ErrUserNotFound
	}
	return f.fakeUsers.FindByEmail(ctx, email)
}

func (f *racingUsers) Create(_ context.Context, user *models.User) error {
	f.users[user.Email] = &models.User{ID: "squatter-user", Email: user.Email}
	return repositories.ErrDuplicateEmail
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

func TestEnterOwnsNewPersonaAccount(t *testing.T) {
	svc, _, users, _ := newService(true)

	if _, err := svc.Enter(context.Background(), "ADMIN_1", test_mode.PersonaClient, ""); err != nil {
		t.Fatalf("Enter: %v", err)
	}

	user := users.users[test_mode.PersonaClientEmail]
	if user.TestPersona == nil || *user.TestPersona != test_mode.PersonaClient {
		t.Fatal("a new persona account must carry the platform ownership marker")
	}
}

func TestEnterRefusesUnmarkedReservedAccount(t *testing.T) {
	sessions := newFakeSessions()
	users := newFakeUsers()
	users.users[test_mode.PersonaClientEmail] = &models.User{
		ID:    "squatter-user",
		Email: test_mode.PersonaClientEmail,
	}
	svc := test_mode.NewService(true, sessions, users, newFakeTrainers(), fakeHasher{})

	_, err := svc.Enter(context.Background(), "ADMIN_1", test_mode.PersonaClient, "")
	if !errors.Is(err, test_mode.ErrPersonaConflict) {
		t.Fatalf("expected ErrPersonaConflict for an unowned reserved account, got %v", err)
	}
	if users.users[test_mode.PersonaClientEmail].ID != "squatter-user" {
		t.Fatal("an unowned account must never be adopted or replaced")
	}
	if len(sessions.byHash) != 0 {
		t.Fatal("a refused persona must never produce a session")
	}
}

func TestEnterRefusesPersonaOwnedByAnotherPersona(t *testing.T) {
	other := test_mode.PersonaTrainer
	users := newFakeUsers()
	users.users[test_mode.PersonaClientEmail] = &models.User{
		ID:          "wrong-persona",
		Email:       test_mode.PersonaClientEmail,
		TestPersona: &other,
	}
	svc := test_mode.NewService(true, newFakeSessions(), users, newFakeTrainers(), fakeHasher{})

	_, err := svc.Enter(context.Background(), "ADMIN_1", test_mode.PersonaClient, "")
	if !errors.Is(err, test_mode.ErrPersonaConflict) {
		t.Fatalf("expected ErrPersonaConflict for a mismatched marker, got %v", err)
	}
}

func TestEnterRefusesConcurrentUnmarkedAccount(t *testing.T) {
	base := newFakeUsers()
	users := &racingUsers{fakeUsers: base}
	sessions := newFakeSessions()
	svc := test_mode.NewService(true, sessions, users, newFakeTrainers(), fakeHasher{})

	_, err := svc.Enter(context.Background(), "ADMIN_1", test_mode.PersonaClient, "")
	if !errors.Is(err, test_mode.ErrPersonaConflict) {
		t.Fatalf("expected ErrPersonaConflict when a concurrent writer owns the address, got %v", err)
	}
	if len(sessions.byHash) != 0 {
		t.Fatal("a concurrent ownership conflict must never produce a session")
	}
}

func TestEnterRetryAfterSessionFailure(t *testing.T) {
	sessions := &failingSessionCreator{fakeSessions: newFakeSessions(), fail: true}
	users := newFakeUsers()
	svc := test_mode.NewService(true, sessions, users, newFakeTrainers(), fakeHasher{})

	if _, err := svc.Enter(context.Background(), "ADMIN_1", test_mode.PersonaClient, ""); err == nil {
		t.Fatal("expected the session persistence failure to be reported")
	}
	if user := users.users[test_mode.PersonaClientEmail]; user == nil || user.TestPersona == nil {
		t.Fatal("the owned persona must remain provisioned after a session failure")
	}
	if len(sessions.byHash) != 0 {
		t.Fatal("a failed enter must not leave an active session")
	}

	if _, err := svc.Enter(context.Background(), "ADMIN_1", test_mode.PersonaClient, ""); err != nil {
		t.Fatalf("retry after a session failure must succeed: %v", err)
	}
}

func TestActivePersonaRejectsSessionWithoutOwnershipMarker(t *testing.T) {
	ctx := context.Background()
	sessions := newFakeSessions()
	users := newFakeUsers()
	users.users["unowned@ryze.local"] = &models.User{ID: "unowned-user", Email: "unowned@ryze.local"}
	sum := sha256.Sum256([]byte("raw-token"))
	if err := sessions.Create(ctx, &models.TestSession{
		AdminIdentity: "ADMIN_1",
		Persona:       test_mode.PersonaClient,
		PersonaUserID: "unowned-user",
		TokenHash:     hex.EncodeToString(sum[:]),
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	svc := test_mode.NewService(true, sessions, users, newFakeTrainers(), fakeHasher{})
	if _, err := svc.ActivePersona(ctx, "raw-token"); !errors.Is(err, test_mode.ErrNoActiveSession) {
		t.Fatalf("expected a session without ownership to be inactive, got %v", err)
	}
}

func TestIsReservedEmail(t *testing.T) {
	reserved := []string{
		test_mode.PersonaClientEmail,
		test_mode.PersonaTrainerEmail,
		"  " + test_mode.PersonaClientEmail + "  ",
		"TEST.CLIENT.PERSONA@RYZE.LOCAL",
	}
	for _, email := range reserved {
		if !test_mode.IsReservedEmail(email) {
			t.Fatalf("expected %q to be reserved", email)
		}
	}

	notReserved := []string{"", "client@ryze.local", "test.client@ryze.local", "persona@ryze.local.example"}
	for _, email := range notReserved {
		if test_mode.IsReservedEmail(email) {
			t.Fatalf("expected %q not to be reserved", email)
		}
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

func TestActivePersonaRejectsExpiredSession(t *testing.T) {
	svc, sessions, _, _ := newServiceWithTTL(true, time.Hour)
	ctx := context.Background()

	entered, err := svc.Enter(ctx, "ADMIN_1", test_mode.PersonaClient, "/admin")
	if err != nil {
		t.Fatalf("Enter: %v", err)
	}
	for _, session := range sessions.byHash {
		session.CreatedAt = time.Now()
	}

	if _, err := svc.ActivePersona(ctx, entered.Token); err != nil {
		t.Fatalf("an in-lifetime session must resolve: %v", err)
	}

	for _, session := range sessions.byHash {
		session.CreatedAt = time.Now().Add(-2 * time.Hour)
	}
	if _, err := svc.ActivePersona(ctx, entered.Token); !errors.Is(err, test_mode.ErrNoActiveSession) {
		t.Fatalf("expected ErrNoActiveSession for an expired session, got %v", err)
	}
}

func TestActivePersonaZeroTTLMeansUnlimited(t *testing.T) {
	svc, sessions, _, _ := newServiceWithTTL(true, 0)
	ctx := context.Background()

	entered, err := svc.Enter(ctx, "ADMIN_1", test_mode.PersonaClient, "")
	if err != nil {
		t.Fatalf("Enter: %v", err)
	}
	for _, session := range sessions.byHash {
		session.CreatedAt = time.Now().Add(-240 * time.Hour)
	}
	if _, err := svc.ActivePersona(ctx, entered.Token); err != nil {
		t.Fatalf("a zero TTL must impose no server-side limit: %v", err)
	}
}

func TestActivePersonaRejectsWhenDisabled(t *testing.T) {
	ctx := context.Background()
	sessions := newFakeSessions()
	sum := sha256.Sum256([]byte("raw-token"))
	if err := sessions.Create(ctx, &models.TestSession{
		AdminIdentity: "ADMIN_1",
		Persona:       test_mode.PersonaClient,
		PersonaUserID: "persona-user",
		TokenHash:     hex.EncodeToString(sum[:]),
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	svc := test_mode.NewService(false, sessions, newFakeUsers(), newFakeTrainers(), fakeHasher{})
	if _, err := svc.ActivePersona(ctx, "raw-token"); !errors.Is(err, test_mode.ErrDisabled) {
		t.Fatalf("expected ErrDisabled for a disabled service, got %v", err)
	}
}

func TestExitRejectsWhenDisabled(t *testing.T) {
	ctx := context.Background()
	sessions := newFakeSessions()
	sum := sha256.Sum256([]byte("raw-token"))
	if err := sessions.Create(ctx, &models.TestSession{
		AdminIdentity: "ADMIN_1",
		Persona:       test_mode.PersonaClient,
		PersonaUserID: "persona-user",
		TokenHash:     hex.EncodeToString(sum[:]),
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	// The kill switch must also stop exit: a stored session must never be able
	// to mint an admin session once the feature is disabled.
	svc := test_mode.NewService(false, sessions, newFakeUsers(), newFakeTrainers(), fakeHasher{})
	if _, err := svc.Exit(ctx, "raw-token"); !errors.Is(err, test_mode.ErrDisabled) {
		t.Fatalf("expected ErrDisabled for a disabled service, got %v", err)
	}
}

func TestExitRestoresAdminEvenAfterExpiry(t *testing.T) {
	svc, sessions, _, _ := newServiceWithTTL(true, time.Hour)
	ctx := context.Background()

	entered, err := svc.Enter(ctx, "ADMIN_1", test_mode.PersonaClient, "/admin/test-mode")
	if err != nil {
		t.Fatalf("Enter: %v", err)
	}
	for _, session := range sessions.byHash {
		session.CreatedAt = time.Now().Add(-2 * time.Hour)
	}

	// Ending a session must never be blocked by its own expiry, otherwise an
	// admin could be stranded inside a persona with no way back.
	exited, err := svc.Exit(ctx, entered.Token)
	if err != nil {
		t.Fatalf("Exit after expiry must restore the admin: %v", err)
	}
	if exited.AdminIdentity != "ADMIN_1" {
		t.Fatalf("expected admin identity ADMIN_1, got %q", exited.AdminIdentity)
	}
	if exited.ReturnPath != "/admin/test-mode" {
		t.Fatalf("expected the recorded return path, got %q", exited.ReturnPath)
	}
}
