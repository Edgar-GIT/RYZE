// Package google_auth implements Google Sign-In with OpenID Connect: the
// authorization code flow with PKCE, a signed single-use state cookie, and ID
// token verification against Google's published keys.
//
// Linking rules:
//   - A Google account signs in to the RYZE account it is linked to.
//   - A Google account that is not linked creates a new RYZE account, unless
//     an active account already uses its email address: that account is never
//     linked automatically. The owner signs in with the password and connects
//     Google from the profile, which proves control of both.
//   - A soft-deleted account is restored, as registering again does.
//   - Google can be disconnected only while the account has a password.
package google_auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/test_mode"
)

// Mode is what a sign-in attempt is for.
type Mode string

const (
	// ModeLogin signs in (or creates an account).
	ModeLogin Mode = "login"
	// ModeLink connects Google to the signed-in account.
	ModeLink Mode = "link"
)

// Result codes the web app shows after the callback: on the login page
// (?oauth=) for sign-in, on the profile (?google=) for linking.
const (
	ResultCancelled     = "cancelled"
	ResultFailed        = "failed"
	ResultExpired       = "expired"
	ResultUnverified    = "unverified"
	ResultLinkRequired  = "link_required"
	ResultLinked        = "linked"
	ResultInUse         = "in_use"
	ResultAlreadyLinked = "already_linked"
	// ResultVerifyEmail: the RYZE account must confirm its email before a
	// Google account can be connected to it.
	ResultVerifyEmail = "verify_email"
)

const defaultReturnPath = "/profile"

var (
	// ErrDisabled is returned while Google Sign-In is not configured.
	ErrDisabled = errors.New("google sign-in is not configured")
	// ErrInvalidInput covers a malformed start request.
	ErrInvalidInput = errors.New("invalid input")
	// ErrPasswordRequired is returned when disconnecting Google would leave
	// the account without a way to sign in.
	ErrPasswordRequired = errors.New("set a password before disconnecting google")
	// ErrNotLinked is returned when the account has no Google link.
	ErrNotLinked = errors.New("google is not connected")
	// ErrUserNotFound is returned for an authenticated account that vanished.
	ErrUserNotFound = errors.New("user not found")

	errEmailUnverified = errors.New("google email not verified")
)

// Users is the account lookup the flow needs.
type Users interface {
	FindByID(ctx context.Context, id string) (*models.User, error)
	FindByIDIncludingDeleted(ctx context.Context, id string) (*models.User, error)
	FindByEmailIncludingDeleted(ctx context.Context, email string) (*models.User, error)
}

// PasswordHasher hashes the unusable password of accounts created by Google.
type PasswordHasher interface {
	HashPassword(password string) (string, error)
}

// Config configures the service. A zero ClientID disables it.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	// StateSecret signs the state cookie (a key is derived from it).
	StateSecret []byte
	Endpoints   Endpoints
	HTTPClient  *http.Client
	Now         func() time.Time
}

// Start is the result of beginning a flow: the consent-screen URL and the
// state cookie value to set before redirecting.
type Start struct {
	URL         string
	StateCookie string
}

// Callback carries the query parameters Google returns.
type Callback struct {
	State string
	Code  string
	Error string
}

// Outcome tells the handler where to send the browser and, when User is set,
// to issue a session for that account first.
type Outcome struct {
	User *models.User
	Path string
}

// Summary describes the account's sign-in methods.
type Summary struct {
	PasswordSet bool
	Identities  []models.UserIdentity
}

// Service is the Google Sign-In contract.
type Service interface {
	Enabled() bool
	Begin(mode Mode, userID, returnPath, language string) (Start, error)
	// Complete finishes a flow. sessionUserID is the account signed in on the
	// callback request ("" when none); linking requires it to match the
	// account that started the flow.
	Complete(ctx context.Context, stateCookie string, callback Callback, sessionUserID string) Outcome
	Summary(ctx context.Context, userID string) (Summary, error)
	Unlink(ctx context.Context, userID string) error
}

type service struct {
	enabled    bool
	oidc       *oidcClient
	stateKey   []byte
	users      Users
	identities repositories.UserIdentityRepository
	hasher     PasswordHasher
	now        func() time.Time
}

// New builds the service. It is disabled when the configuration is incomplete.
func New(cfg Config, users Users, identities repositories.UserIdentityRepository, hasher PasswordHasher) Service {
	now := cfg.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	endpoints := cfg.Endpoints
	if endpoints == (Endpoints{}) {
		endpoints = GoogleEndpoints()
	}
	return &service{
		enabled:    cfg.ClientID != "" && cfg.ClientSecret != "" && cfg.RedirectURL != "" && len(cfg.StateSecret) > 0,
		oidc:       newOIDCClient(endpoints, cfg.ClientID, cfg.ClientSecret, cfg.RedirectURL, cfg.HTTPClient, now),
		stateKey:   stateKey(cfg.StateSecret),
		users:      users,
		identities: identities,
		hasher:     hasher,
		now:        now,
	}
}

func (s *service) Enabled() bool { return s.enabled }

func (s *service) Begin(mode Mode, userID, returnPath, language string) (Start, error) {
	if !s.enabled {
		return Start{}, ErrDisabled
	}
	if mode != ModeLogin && mode != ModeLink {
		return Start{}, ErrInvalidInput
	}
	if mode == ModeLink && userID == "" {
		return Start{}, ErrInvalidInput
	}
	if mode == ModeLink {
		returnPath = ""
	}
	if language != "en" && language != "pt" {
		language = ""
	}
	stateValue, err := randomToken(32)
	if err != nil {
		return Start{}, err
	}
	nonce, err := randomToken(32)
	if err != nil {
		return Start{}, err
	}
	verifier, err := randomToken(32)
	if err != nil {
		return Start{}, err
	}
	cookie, err := signState(s.stateKey, flowState{
		State:    stateValue,
		Nonce:    nonce,
		Verifier: verifier,
		Mode:     mode,
		UserID:   userID,
		Return:   safeReturnPath(returnPath),
		Language: language,
	}, s.now())
	if err != nil {
		return Start{}, err
	}
	return Start{URL: s.oidc.authorizationURL(stateValue, nonce, pkceChallenge(verifier)), StateCookie: cookie}, nil
}

func loginPath(code string) string   { return "/login?oauth=" + code }
func profilePath(code string) string { return "/profile?google=" + code }

func (s *service) Complete(ctx context.Context, stateCookie string, callback Callback, sessionUserID string) Outcome {
	if !s.enabled {
		return Outcome{Path: loginPath(ResultFailed)}
	}
	state, err := parseState(s.stateKey, stateCookie, s.now())
	if err != nil || !constantTimeEqual(state.State, callback.State) {
		return Outcome{Path: loginPath(ResultExpired)}
	}
	fail := func(code string) Outcome {
		if state.Mode == ModeLink {
			return Outcome{Path: profilePath(code)}
		}
		return Outcome{Path: loginPath(code)}
	}
	if callback.Error != "" {
		if callback.Error == "access_denied" {
			return fail(ResultCancelled)
		}
		return fail(ResultFailed)
	}
	if callback.Code == "" || len(callback.Code) > 2048 {
		return fail(ResultFailed)
	}
	if state.Mode == ModeLink && (sessionUserID == "" || sessionUserID != state.UserID) {
		// The session changed (or ended) while the consent screen was open.
		return Outcome{Path: loginPath(ResultExpired)}
	}

	idToken, err := s.oidc.exchange(ctx, callback.Code, state.Verifier)
	if err != nil {
		logFailure("exchange", err)
		return fail(ResultFailed)
	}
	profile, err := s.oidc.verify(ctx, idToken, state.Nonce)
	if errors.Is(err, errEmailUnverified) {
		return fail(ResultUnverified)
	}
	if err != nil {
		logFailure("verify", err)
		return fail(ResultFailed)
	}

	if state.Mode == ModeLink {
		return s.link(ctx, state.UserID, profile)
	}
	return s.signIn(ctx, state, profile)
}

func logFailure(stage string, err error) {
	slog.Warn("google sign-in failed", slog.String("stage", stage), slog.String("error", err.Error()))
}

func (s *service) signIn(ctx context.Context, state *flowState, profile Profile) Outcome {
	now := s.now()
	success := state.Return
	if success == "" {
		success = defaultReturnPath
	}
	failed := Outcome{Path: loginPath(ResultFailed)}

	identity, err := s.identities.FindByProviderSubject(ctx, models.IdentityProviderGoogle, profile.Subject)
	switch {
	case err == nil:
		user, err := s.users.FindByIDIncludingDeleted(ctx, identity.UserID)
		if err != nil {
			logFailure("load_user", err)
			return failed
		}
		if user.DeletedAt.Valid {
			return s.restore(ctx, user, profile, success)
		}
		if user.TestPersona != nil {
			return failed
		}
		if err := s.identities.Touch(ctx, identity.ID, profile.Email, now); err != nil {
			logFailure("touch", err)
		}
		return Outcome{User: user, Path: success}
	case errors.Is(err, repositories.ErrUserIdentityNotFound):
	default:
		logFailure("find_identity", err)
		return failed
	}

	// A Test Mode persona address is a platform identity.
	if test_mode.IsReservedEmail(profile.Email) {
		return failed
	}
	existing, err := s.users.FindByEmailIncludingDeleted(ctx, profile.Email)
	switch {
	case err == nil && existing.DeletedAt.Valid:
		return s.restore(ctx, existing, profile, success)
	case err == nil:
		// Never link an existing account without its password.
		return Outcome{Path: loginPath(ResultLinkRequired)}
	case errors.Is(err, repositories.ErrUserNotFound):
	default:
		logFailure("find_user", err)
		return failed
	}

	hash, err := s.unusablePasswordHash()
	if err != nil {
		logFailure("hash", err)
		return failed
	}
	firstName, lastName := names(profile)
	user := &models.User{
		Email:        profile.Email,
		PasswordHash: hash,
		FirstName:    firstName,
		LastName:     lastName,
	}
	if state.Language != "" {
		language := state.Language
		user.PreferredLanguage = &language
	}
	newIdentity := &models.UserIdentity{Provider: models.IdentityProviderGoogle, Subject: profile.Subject, Email: profile.Email}
	if err := s.identities.CreateUserWithIdentity(ctx, user, newIdentity, now); err != nil {
		if errors.Is(err, repositories.ErrDuplicateEmail) {
			return Outcome{Path: loginPath(ResultLinkRequired)}
		}
		logFailure("create_user", err)
		return failed
	}
	user.PasswordHash = ""
	return Outcome{User: user, Path: success}
}

// restore reactivates a soft-deleted account for the Google account.
func (s *service) restore(ctx context.Context, user *models.User, profile Profile, success string) Outcome {
	failed := Outcome{Path: loginPath(ResultFailed)}
	if user.TestPersona != nil {
		return failed
	}
	hash, err := s.unusablePasswordHash()
	if err != nil {
		logFailure("hash", err)
		return failed
	}
	firstName, lastName := names(profile)
	restored := *user
	restored.FirstName = firstName
	restored.LastName = lastName
	restored.PasswordHash = hash
	identity := &models.UserIdentity{Provider: models.IdentityProviderGoogle, Subject: profile.Subject, Email: profile.Email}
	if err := s.identities.ReactivateWithIdentity(ctx, &restored, identity, s.now()); err != nil {
		if errors.Is(err, repositories.ErrDuplicateEmail) || errors.Is(err, repositories.ErrIdentityInUse) {
			return Outcome{Path: loginPath(ResultLinkRequired)}
		}
		logFailure("restore_user", err)
		return failed
	}
	fresh, err := s.users.FindByID(ctx, user.ID)
	if err != nil {
		logFailure("load_user", err)
		return failed
	}
	return Outcome{User: fresh, Path: success}
}

func (s *service) link(ctx context.Context, userID string, profile Profile) Outcome {
	now := s.now()
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return Outcome{Path: loginPath(ResultExpired)}
		}
		logFailure("load_user", err)
		return Outcome{Path: profilePath(ResultFailed)}
	}
	if user.TestPersona != nil {
		return Outcome{Path: profilePath(ResultFailed)}
	}
	// An unconfirmed address may belong to someone else: whoever registered
	// it must not be able to attach their own Google account and keep a way
	// in after the real owner recovers the account.
	if user.EmailVerifiedAt == nil {
		return Outcome{Path: profilePath(ResultVerifyEmail)}
	}
	existing, err := s.identities.FindByProviderSubject(ctx, models.IdentityProviderGoogle, profile.Subject)
	switch {
	case err == nil && existing.UserID == userID:
		if err := s.identities.Touch(ctx, existing.ID, profile.Email, now); err != nil {
			logFailure("touch", err)
		}
		return Outcome{Path: profilePath(ResultLinked)}
	case err == nil:
		return Outcome{Path: profilePath(ResultInUse)}
	case errors.Is(err, repositories.ErrUserIdentityNotFound):
	default:
		logFailure("find_identity", err)
		return Outcome{Path: profilePath(ResultFailed)}
	}
	identity := &models.UserIdentity{
		UserID:     userID,
		Provider:   models.IdentityProviderGoogle,
		Subject:    profile.Subject,
		Email:      profile.Email,
		LastUsedAt: &now,
	}
	switch err := s.identities.Link(ctx, identity); {
	case err == nil:
		return Outcome{Path: profilePath(ResultLinked)}
	case errors.Is(err, repositories.ErrProviderAlreadyLinked):
		return Outcome{Path: profilePath(ResultAlreadyLinked)}
	case errors.Is(err, repositories.ErrIdentityInUse):
		return Outcome{Path: profilePath(ResultInUse)}
	case errors.Is(err, repositories.ErrUserNotFound):
		return Outcome{Path: loginPath(ResultExpired)}
	default:
		logFailure("link", err)
		return Outcome{Path: profilePath(ResultFailed)}
	}
}

func (s *service) Summary(ctx context.Context, userID string) (Summary, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return Summary{}, ErrUserNotFound
		}
		return Summary{}, err
	}
	identities, err := s.identities.ListByUser(ctx, userID)
	if err != nil {
		return Summary{}, err
	}
	return Summary{PasswordSet: user.PasswordSet, Identities: identities}, nil
}

func (s *service) Unlink(ctx context.Context, userID string) error {
	switch err := s.identities.Unlink(ctx, userID, models.IdentityProviderGoogle); {
	case err == nil:
		return nil
	case errors.Is(err, repositories.ErrPasswordRequired):
		return ErrPasswordRequired
	case errors.Is(err, repositories.ErrUserIdentityNotFound):
		return ErrNotLinked
	case errors.Is(err, repositories.ErrUserNotFound):
		return ErrUserNotFound
	default:
		return err
	}
}

// unusablePasswordHash hashes a random secret nobody knows, so password
// sign-in fails for the account (with the same timing as a wrong password)
// until the owner sets a password through the reset link.
func (s *service) unusablePasswordHash() (string, error) {
	secret, err := randomToken(32)
	if err != nil {
		return "", err
	}
	return s.hasher.HashPassword(secret)
}

// names picks the account names from the Google profile.
func names(profile Profile) (string, string) {
	first, last := profile.GivenName, profile.FamilyName
	if first == "" && profile.Name != "" {
		parts := strings.Fields(profile.Name)
		first = parts[0]
		if last == "" && len(parts) > 1 {
			last = strings.Join(parts[1:], " ")
		}
	}
	if first == "" {
		first, _, _ = strings.Cut(profile.Email, "@")
	}
	return truncate(first, 100), truncate(last, 100)
}

func truncate(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	return string([]rune(value)[:limit])
}
