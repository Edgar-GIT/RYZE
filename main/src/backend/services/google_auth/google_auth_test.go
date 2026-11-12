package google_auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/login"
	"ryze/backend/services/password"
	"ryze/backend/services/test_mode"
)

type harness struct {
	tx         *gorm.DB
	svc        Service
	fake       *fakeGoogle
	users      repositories.UserRepository
	identities repositories.UserIdentityRepository
}

// newHarness runs against the isolated test database inside a transaction
// that is rolled back when the test ends.
func newHarness(t *testing.T) *harness {
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
	t.Cleanup(func() {
		tx.Rollback()
		// Each test opens its own pool; close it so repeated runs never
		// exhaust the server's connections.
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	fake := newFakeGoogle(t)
	users := repositories.NewUserRepository(tx)
	identities := repositories.NewUserIdentityRepository(tx)
	svc := New(Config{
		ClientID:     testClientID,
		ClientSecret: testSecret,
		RedirectURL:  testRedirectURL,
		StateSecret:  []byte("state-secret-for-tests"),
		Endpoints:    fake.endpoints(),
		HTTPClient:   fake.server.Client(),
	}, users, identities, password.Hasher{})
	return &harness{tx: tx, svc: svc, fake: fake, users: users, identities: identities}
}

// signIn runs one full flow and returns the outcome.
func (h *harness) run(t *testing.T, mode Mode, userID, returnPath string, claims jwt.MapClaims, sessionUserID string) Outcome {
	t.Helper()
	start, err := h.svc.Begin(mode, userID, returnPath, "pt")
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	state, code := h.fake.authorize(start.URL, claims)
	return h.svc.Complete(context.Background(), start.StateCookie, Callback{State: state, Code: code}, sessionUserID)
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%d@example.com", prefix, time.Now().UnixNano())
}

func uniqueSubject() string {
	return fmt.Sprintf("sub-%d", time.Now().UnixNano())
}

func (h *harness) seedPasswordUser(t *testing.T, address string) *models.User {
	t.Helper()
	hash, err := password.HashPassword("Original-Passw0rd!")
	if err != nil {
		t.Fatal(err)
	}
	user := &models.User{Email: address, PasswordHash: hash, FirstName: "Ana", LastName: "Silva"}
	if err := h.users.Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

// verify marks the account's email as confirmed (linking requires it).
func (h *harness) verify(t *testing.T, user *models.User) {
	t.Helper()
	if err := h.tx.Model(&models.User{}).Where("id = ?", user.ID).Update("email_verified_at", time.Now()).Error; err != nil {
		t.Fatal(err)
	}
}

func TestGoogleCreatesAnAccountAndSignsBackIn(t *testing.T) {
	h := newHarness(t)
	address := uniqueEmail("google-new")
	subject := uniqueSubject()

	outcome := h.run(t, ModeLogin, "", "/programs", idClaimsFor(subject, address), "")
	if outcome.User == nil || outcome.Path != "/programs" {
		t.Fatalf("expected a session and /programs, got %+v", outcome)
	}

	stored, err := h.users.FindByID(context.Background(), outcome.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.PasswordSet || stored.EmailVerifiedAt == nil || stored.PreferredLanguage == nil || *stored.PreferredLanguage != "pt" {
		t.Fatalf("unexpected new account state: password_set=%v verified=%v language=%v", stored.PasswordSet, stored.EmailVerifiedAt, stored.PreferredLanguage)
	}
	if stored.FirstName != "Rita" || stored.LastName != "Costa" {
		t.Fatalf("unexpected names %q %q", stored.FirstName, stored.LastName)
	}

	// Password sign-in is impossible until a password is set.
	loginSvc := login.NewLoginService(h.users, password.Verifier{})
	if _, err := loginSvc.Login(context.Background(), login.LoginInput{Email: address, Password: "anything-at-all-123"}); !errors.Is(err, login.ErrInvalidCredentials) {
		t.Fatalf("expected password sign-in to fail, got %v", err)
	}

	// Signing in again reaches the same account; the default destination is
	// the profile.
	again := h.run(t, ModeLogin, "", "", idClaimsFor(subject, address), "")
	if again.User == nil || again.User.ID != outcome.User.ID || again.Path != "/profile" {
		t.Fatalf("expected the same account and /profile, got %+v", again)
	}
	identity, err := h.identities.FindByProviderSubject(context.Background(), models.IdentityProviderGoogle, subject)
	if err != nil || identity.LastUsedAt == nil {
		t.Fatalf("expected a used identity, got %+v (%v)", identity, err)
	}
}

func TestGoogleNeverLinksAnExistingAccountAutomatically(t *testing.T) {
	h := newHarness(t)
	address := uniqueEmail("google-existing")
	h.seedPasswordUser(t, address)

	// Same address in another letter case: CITEXT matches it.
	outcome := h.run(t, ModeLogin, "", "", idClaimsFor(uniqueSubject(), strings.ToUpper(address)), "")
	if outcome.User != nil || outcome.Path != "/login?oauth=link_required" {
		t.Fatalf("expected link_required without a session, got %+v", outcome)
	}
	var count int64
	h.tx.Model(&models.UserIdentity{}).Joins("JOIN users ON users.id = user_identities.user_id").Where("users.email = ?", address).Count(&count)
	if count != 0 {
		t.Fatalf("an identity was linked automatically")
	}
}

func TestLinkingFromTheProfile(t *testing.T) {
	h := newHarness(t)
	owner := h.seedPasswordUser(t, uniqueEmail("link-owner"))
	other := h.seedPasswordUser(t, uniqueEmail("link-other"))
	subject := uniqueSubject()

	// An unconfirmed email cannot receive a Google link.
	if outcome := h.run(t, ModeLink, owner.ID, "", idClaimsFor(subject, "rita@gmail.com"), owner.ID); outcome.Path != "/profile?google=verify_email" {
		t.Fatalf("expected verify_email for an unconfirmed account, got %+v", outcome)
	}
	h.verify(t, owner)
	h.verify(t, other)

	// The session must belong to the account that started the flow.
	if outcome := h.run(t, ModeLink, owner.ID, "", idClaimsFor(subject, "rita@gmail.com"), other.ID); outcome.Path != "/login?oauth=expired" {
		t.Fatalf("expected expired for a different session, got %+v", outcome)
	}
	if outcome := h.run(t, ModeLink, owner.ID, "", idClaimsFor(subject, "rita@gmail.com"), ""); outcome.Path != "/login?oauth=expired" {
		t.Fatalf("expected expired without a session, got %+v", outcome)
	}

	outcome := h.run(t, ModeLink, owner.ID, "", idClaimsFor(subject, "rita@gmail.com"), owner.ID)
	if outcome.User != nil || outcome.Path != "/profile?google=linked" {
		t.Fatalf("expected linked, got %+v", outcome)
	}
	// Linking the same Google account again is harmless.
	if outcome := h.run(t, ModeLink, owner.ID, "", idClaimsFor(subject, "rita@gmail.com"), owner.ID); outcome.Path != "/profile?google=linked" {
		t.Fatalf("expected linked again, got %+v", outcome)
	}
	// A second Google account cannot be added to the same RYZE account.
	if outcome := h.run(t, ModeLink, owner.ID, "", idClaimsFor(uniqueSubject(), "rita2@gmail.com"), owner.ID); outcome.Path != "/profile?google=already_linked" {
		t.Fatalf("expected already_linked, got %+v", outcome)
	}
	// The linked Google account cannot be added to another RYZE account.
	if outcome := h.run(t, ModeLink, other.ID, "", idClaimsFor(subject, "rita@gmail.com"), other.ID); outcome.Path != "/profile?google=in_use" {
		t.Fatalf("expected in_use, got %+v", outcome)
	}

	// Google now signs in to the owner's account.
	signIn := h.run(t, ModeLogin, "", "", idClaimsFor(subject, "rita@gmail.com"), "")
	if signIn.User == nil || signIn.User.ID != owner.ID {
		t.Fatalf("expected a session for the owner, got %+v", signIn)
	}

	summary, err := h.svc.Summary(context.Background(), owner.ID)
	if err != nil || !summary.PasswordSet || len(summary.Identities) != 1 || summary.Identities[0].Email != "rita@gmail.com" {
		t.Fatalf("unexpected summary %+v (%v)", summary, err)
	}
	if err := h.svc.Unlink(context.Background(), owner.ID); err != nil {
		t.Fatalf("unlink: %v", err)
	}
	if err := h.svc.Unlink(context.Background(), owner.ID); !errors.Is(err, ErrNotLinked) {
		t.Fatalf("expected ErrNotLinked, got %v", err)
	}
}

func TestGoogleOnlyAccountsKeepTheirOnlySignInMethod(t *testing.T) {
	h := newHarness(t)
	outcome := h.run(t, ModeLogin, "", "", idClaimsFor(uniqueSubject(), uniqueEmail("google-only")), "")
	if outcome.User == nil {
		t.Fatalf("expected a new account, got %+v", outcome)
	}
	if err := h.svc.Unlink(context.Background(), outcome.User.ID); !errors.Is(err, ErrPasswordRequired) {
		t.Fatalf("expected ErrPasswordRequired, got %v", err)
	}
	// Setting a password (the reset flow stores it the same way) allows it.
	hash, err := password.HashPassword("A-new-passw0rd-123")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.users.ChangePassword(context.Background(), outcome.User.ID, hash); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.Unlink(context.Background(), outcome.User.ID); err != nil {
		t.Fatalf("unlink after setting a password: %v", err)
	}
}

func TestGoogleRestoresASoftDeletedAccount(t *testing.T) {
	h := newHarness(t)
	address := uniqueEmail("google-restore")
	user := h.seedPasswordUser(t, address)
	if err := h.users.DeleteAccount(context.Background(), user.ID); err != nil {
		t.Fatal(err)
	}
	deleted, err := h.users.FindByIDIncludingDeleted(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}

	outcome := h.run(t, ModeLogin, "", "", idClaimsFor(uniqueSubject(), address), "")
	if outcome.User == nil || outcome.User.ID != user.ID {
		t.Fatalf("expected the original account restored, got %+v", outcome)
	}
	restored, err := h.users.FindByID(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.PasswordSet || restored.SessionVersion != deleted.SessionVersion+1 || outcome.User.SessionVersion != restored.SessionVersion {
		t.Fatalf("unexpected restored state: password_set=%v session %d (deleted %d, issued %d)",
			restored.PasswordSet, restored.SessionVersion, deleted.SessionVersion, outcome.User.SessionVersion)
	}
	// The old password no longer works.
	loginSvc := login.NewLoginService(h.users, password.Verifier{})
	if _, err := loginSvc.Login(context.Background(), login.LoginInput{Email: address, Password: "Original-Passw0rd!"}); !errors.Is(err, login.ErrInvalidCredentials) {
		t.Fatalf("expected the old password to stop working, got %v", err)
	}
}

func TestGoogleFlowRejectsBadCallbacks(t *testing.T) {
	h := newHarness(t)
	start, err := h.svc.Begin(ModeLogin, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	state, code := h.fake.authorize(start.URL, idClaimsFor(uniqueSubject(), uniqueEmail("bad")))

	cases := []struct {
		name     string
		cookie   string
		callback Callback
		want     string
	}{
		{"missing cookie", "", Callback{State: state, Code: code}, "/login?oauth=expired"},
		{"state mismatch", start.StateCookie, Callback{State: "other", Code: code}, "/login?oauth=expired"},
		{"cancelled", start.StateCookie, Callback{State: state, Error: "access_denied"}, "/login?oauth=cancelled"},
		{"provider error", start.StateCookie, Callback{State: state, Error: "server_error"}, "/login?oauth=failed"},
		{"missing code", start.StateCookie, Callback{State: state}, "/login?oauth=failed"},
		{"unknown code", start.StateCookie, Callback{State: state, Code: "forged"}, "/login?oauth=failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outcome := h.svc.Complete(context.Background(), tc.cookie, tc.callback, "")
			if outcome.User != nil || outcome.Path != tc.want {
				t.Fatalf("got %+v, want %s", outcome, tc.want)
			}
		})
	}

	unverified := idClaimsFor(uniqueSubject(), uniqueEmail("unverified"))
	unverified["email_verified"] = false
	if outcome := h.run(t, ModeLogin, "", "", unverified, ""); outcome.User != nil || outcome.Path != "/login?oauth=unverified" {
		t.Fatalf("expected unverified, got %+v", outcome)
	}
	if outcome := h.run(t, ModeLogin, "", "", idClaimsFor(uniqueSubject(), test_mode.PersonaClientEmail), ""); outcome.User != nil || outcome.Path != "/login?oauth=failed" {
		t.Fatalf("expected the Test Mode persona address to be refused, got %+v", outcome)
	}
}

func TestDisabledServiceRefusesToStart(t *testing.T) {
	svc := New(Config{}, nil, nil, nil)
	if svc.Enabled() {
		t.Fatal("service without configuration reports enabled")
	}
	if _, err := svc.Begin(ModeLogin, "", "", ""); !errors.Is(err, ErrDisabled) {
		t.Fatalf("expected ErrDisabled, got %v", err)
	}
}
