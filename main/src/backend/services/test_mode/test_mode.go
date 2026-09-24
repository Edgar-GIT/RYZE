package test_mode

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"ryze/backend/config"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

// Personas are the predefined identities Test Mode may browse as. They are the
// only allowed values; Test Mode intentionally provides no generic
// impersonation so a session can never be aimed at an arbitrary user account.
const (
	PersonaClient  = "client"
	PersonaTrainer = "trainer"

	// PersonaClientEmail and PersonaTrainerEmail are the stable account
	// identifiers of the dedicated test personas. The accounts are created
	// on demand by the service when they do not exist yet, so Test Mode works
	// on any fresh database without requiring manual seeding. The passwords are
	// random and never known to anyone: personas authenticate exclusively
	// through the session token issued at enter.
	PersonaClientEmail  = "test.client.persona@ryze.local"
	PersonaTrainerEmail = "test.trainer.persona@ryze.local"

	maxReturnPathLength = 512
)

var (
	// ErrDisabled is returned when Test Mode is not enabled by configuration.
	ErrDisabled = errors.New("test mode is disabled")
	// ErrInvalidInput indicates malformed or incomplete Test Mode input.
	ErrInvalidInput = errors.New("invalid test mode input")
	// ErrUnknownPersona indicates the requested persona is not predefined.
	ErrUnknownPersona = errors.New("unknown test persona")
	// ErrInvalidReturnPath indicates the supplied return path would let the
	// restored admin leave the platform (external URL, protocol-relative
	// path, control characters...).
	ErrInvalidReturnPath = errors.New("invalid test mode return path")
	// ErrNoActiveSession indicates there is no active Test Mode session for
	// the supplied token.
	ErrNoActiveSession = errors.New("no active test mode session")
)

// PasswordHasher is the hashing surface used to give each persona a random,
// unusable password at creation time. Passwords are never meaningful for the
// personas — they authenticate through the session token — but the schema
// requires a non-empty hash.
type PasswordHasher interface {
	HashPassword(password string) (string, error)
}

// Session is the safe representation of one active Test Mode session.
type Session struct {
	AdminIdentity string
	Persona       string
	PersonaUserID string
	ReturnPath    string
}

// EnterResult carries the session token the handler stores in the HttpOnly
// cookie together with the persona identity needed to mint the persona access
// token. The token is the only secret and is never persisted in plain text.
type EnterResult struct {
	Token         string
	Persona       string
	PersonaUserID string
	ReturnPath    string
}

// ExitResult carries the original admin identity to restore and the return
// path to redirect to after exiting Test Mode.
type ExitResult struct {
	AdminIdentity string
	ReturnPath    string
}

// Service implements the Test Mode lifecycle. The effective identity is always
// established server-side: the handler mints a real persona access token and
// the session cookie, so the client can never self-assign a persona. The user
// identity and original admin identity always come from the caller, never from
// client input.
type Service interface {
	Enter(ctx context.Context, adminIdentity, persona, returnPath string) (EnterResult, error)
	Exit(ctx context.Context, rawToken string) (ExitResult, error)
	ActivePersona(ctx context.Context, rawToken string) (*Session, error)
}

// UserRepository is the user data-access surface Test Mode needs to resolve
// and create the dedicated persona accounts.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
}

// TrainerRepository is the trainer data-access surface Test Mode needs to keep
// the trainer persona's linked profile consistent.
type TrainerRepository interface {
	Create(ctx context.Context, trainer *models.Trainer) error
	FindByUserID(ctx context.Context, userID string) (*models.Trainer, error)
}

type service struct {
	enabled  bool
	sessions repositories.TestSessionRepository
	users    UserRepository
	trainers TrainerRepository
	hasher   PasswordHasher
}

func NewService(
	enabled bool,
	sessions repositories.TestSessionRepository,
	users UserRepository,
	trainers TrainerRepository,
	hasher PasswordHasher,
) Service {
	return &service{
		enabled:  enabled,
		sessions: sessions,
		users:    users,
		trainers: trainers,
		hasher:   hasher,
	}
}

// Enter opens a Test Mode session for one predefined persona. The original
// admin identity is recorded so exiting always restores it. The raw token is
// returned exactly once (to be stored in an HttpOnly cookie); only its SHA-256
// is persisted.
func (s *service) Enter(ctx context.Context, adminIdentity, persona, returnPath string) (EnterResult, error) {
	if !s.enabled {
		return EnterResult{}, ErrDisabled
	}
	if strings.TrimSpace(adminIdentity) == "" {
		return EnterResult{}, ErrInvalidInput
	}
	if !config.IsValidAdminIdentity(adminIdentity) {
		return EnterResult{}, ErrInvalidInput
	}
	if !IsValidPersona(persona) {
		return EnterResult{}, ErrUnknownPersona
	}
	validatedPath, err := sanitizeReturnPath(returnPath)
	if err != nil {
		return EnterResult{}, err
	}

	personaUser, err := s.ensurePersona(ctx, persona)
	if err != nil {
		return EnterResult{}, fmt.Errorf("failed to prepare persona: %w", err)
	}

	rawToken, err := generateToken()
	if err != nil {
		return EnterResult{}, fmt.Errorf("failed to generate test session token: %w", err)
	}
	session := &models.TestSession{
		AdminIdentity: adminIdentity,
		Persona:       persona,
		PersonaUserID: personaUser.ID,
		TokenHash:     hashToken(rawToken),
		ReturnPath:    validatedPath,
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return EnterResult{}, fmt.Errorf("failed to create test session: %w", err)
	}

	return EnterResult{
		Token:         rawToken,
		Persona:       persona,
		PersonaUserID: personaUser.ID,
		ReturnPath:    validatedPath,
	}, nil
}

// Exit ends the active session for the given raw token and returns the
// original admin identity and the recorded return path. The token must still
// be active; an unknown or already-exited session reports ErrNoActiveSession.
func (s *service) Exit(ctx context.Context, rawToken string) (ExitResult, error) {
	if strings.TrimSpace(rawToken) == "" {
		return ExitResult{}, ErrNoActiveSession
	}
	session, err := s.sessions.FindActiveByTokenHash(ctx, hashToken(rawToken))
	if err != nil {
		if errors.Is(err, repositories.ErrTestSessionNotFound) {
			return ExitResult{}, ErrNoActiveSession
		}
		return ExitResult{}, fmt.Errorf("failed to load test session: %w", err)
	}

	if err := s.sessions.SoftDelete(ctx, session.ID); err != nil {
		if errors.Is(err, repositories.ErrTestSessionNotFound) {
			return ExitResult{}, ErrNoActiveSession
		}
		return ExitResult{}, fmt.Errorf("failed to end test session: %w", err)
	}

	return ExitResult{
		AdminIdentity: session.AdminIdentity,
		ReturnPath:    session.ReturnPath,
	}, nil
}

// ActivePersona resolves the active session bound to a raw token. The returned
// persona user id lets the authenticated endpoints verify that the caller is
// precisely the persona the session was opened for.
func (s *service) ActivePersona(ctx context.Context, rawToken string) (*Session, error) {
	if strings.TrimSpace(rawToken) == "" {
		return nil, ErrNoActiveSession
	}
	session, err := s.sessions.FindActiveByTokenHash(ctx, hashToken(rawToken))
	if err != nil {
		if errors.Is(err, repositories.ErrTestSessionNotFound) {
			return nil, ErrNoActiveSession
		}
		return nil, fmt.Errorf("failed to load test session: %w", err)
	}
	return &Session{
		AdminIdentity: session.AdminIdentity,
		Persona:       session.Persona,
		PersonaUserID: session.PersonaUserID,
		ReturnPath:    session.ReturnPath,
	}, nil
}

// IsValidPersona reports whether persona is one of the predefined identities.
func IsValidPersona(persona string) bool {
	return persona == PersonaClient || persona == PersonaTrainer
}

// personaEmail maps a persona to its dedicated account identifier.
func personaEmail(persona string) string {
	if persona == PersonaTrainer {
		return PersonaTrainerEmail
	}
	return PersonaClientEmail
}

// personaName maps a persona to its human-readable profile name.
func personaName(persona string) string {
	if persona == PersonaTrainer {
		return "Trainer"
	}
	return "Client"
}

// ensurePersona returns the dedicated account for the persona, creating it
// (with a random unusable password) when it does not exist yet, and making the
// trainer profile row exist for the trainer persona. The accounts are stable:
// registrations, purchases and entitlements made inside Test Mode therefore
// persist across sessions instead of being discarded each time.
func (s *service) ensurePersona(ctx context.Context, persona string) (*models.User, error) {
	email := personaEmail(persona)

	user, err := s.users.FindByEmail(ctx, email)
	if errors.Is(err, repositories.ErrUserNotFound) {
		user, err = s.createPersonaUser(ctx, persona, email)
	}
	if err != nil {
		return nil, err
	}

	if persona == PersonaTrainer {
		if _, err := s.trainers.FindByUserID(ctx, user.ID); errors.Is(err, repositories.ErrTrainerNotFound) {
			if err := s.trainers.Create(ctx, &models.Trainer{UserID: user.ID}); err != nil && !errors.Is(err, repositories.ErrTrainerAlreadyLinked) {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (s *service) createPersonaUser(ctx context.Context, persona, email string) (*models.User, error) {
	randomSecret, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate persona password: %w", err)
	}
	hash, err := s.hasher.HashPassword(randomSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to hash persona password: %w", err)
	}

	user := &models.User{
		Email:        email,
		PasswordHash: hash,
		FirstName:    "Test",
		LastName:     personaName(persona),
	}
	if err := s.users.Create(ctx, user); err != nil {
		// The user account already exists (concurrent ensure): treat it as a
		// successful resolution instead of failing the enter flow.
		if errors.Is(err, repositories.ErrDuplicateEmail) {
			existing, findErr := s.users.FindByEmail(ctx, email)
			if findErr != nil {
				return nil, findErr
			}
			return existing, nil
		}
		return nil, err
	}
	return user, nil
}

// generateToken returns a full-entropy hex token. It is used both for the
// session bearer token (64 characters) and the random persona passwords.
func generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// hashToken returns the SHA-256 digest of a raw token in hex form. The raw
// token is never stored anywhere.
func hashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

// sanitizeReturnPath validates and normalizes the path an admin is redirected
// to after exiting Test Mode. Only same-origin absolute paths are accepted:
// external URLs (scheme://...), protocol-relative paths (//...), backslashes
// and control characters are rejected so exit can never be used as an open
// redirect. An empty path is allowed and means "stay at the current admin
// area" (the frontend falls back to /admin).
func sanitizeReturnPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	if len(path) > maxReturnPathLength {
		return "", ErrInvalidReturnPath
	}
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return "", ErrInvalidReturnPath
	}
	if strings.Contains(path, "://") || strings.Contains(path, "\\") {
		return "", ErrInvalidReturnPath
	}
	for _, r := range path {
		if unicode.IsControl(r) {
			return "", ErrInvalidReturnPath
		}
	}
	return path, nil
}