package token_test

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"ryze/backend/config"
	"ryze/backend/services/token"
)

const testSecret = "test-secret-that-is-longer-than-32-bytes-1234"

func newService(t *testing.T, ttl time.Duration) token.Service {
	t.Helper()
	return token.NewService([]byte(testSecret), ttl)
}

func TestGenerateAndValidateAccessToken(t *testing.T) {
	svc := newService(t, 15*time.Minute)
	userID := uuid.NewString()

	raw, err := svc.GenerateAccessToken(userID, 0)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	claims, err := svc.ValidateAccessToken(raw)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}
	if claims.UserID != userID {
		t.Fatalf("expected subject %q, got %q", userID, claims.UserID)
	}
}

func TestValidateReturnsCorrectUUID(t *testing.T) {
	svc := newService(t, 15*time.Minute)
	userID := uuid.NewString()

	raw, err := svc.GenerateAccessToken(userID, 0)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	claims, err := svc.ValidateAccessToken(raw)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}
	if _, err := uuid.Parse(claims.UserID); err != nil {
		t.Fatalf("subject must be a valid UUID, got %q: %v", claims.UserID, err)
	}
	if claims.UserID != userID {
		t.Fatalf("expected subject %q, got %q", userID, claims.UserID)
	}
}

func TestSessionVersionRoundTrip(t *testing.T) {
	svc := newService(t, 15*time.Minute)
	userID := uuid.NewString()

	for _, version := range []int{0, 1, 42} {
		raw, err := svc.GenerateAccessToken(userID, version)
		if err != nil {
			t.Fatalf("GenerateAccessToken(%d): %v", version, err)
		}

		claims, err := svc.ValidateAccessToken(raw)
		if err != nil {
			t.Fatalf("ValidateAccessToken(%d): %v", version, err)
		}
		if claims.SessionVersion != version {
			t.Fatalf("expected session version %d, got %d", version, claims.SessionVersion)
		}
	}
}

func TestMissingSessionVersionDefaultsToZero(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	claims := jwt.RegisteredClaims{
		Subject:   uuid.NewString(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	parsed, err := svc.ValidateAccessToken(raw)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}
	if parsed.SessionVersion != 0 {
		t.Fatalf("expected session version 0 for a token without a ver claim, got %d", parsed.SessionVersion)
	}
}

func TestExpiredTokenIsRejected(t *testing.T) {
	svc := newService(t, -1*time.Minute)
	userID := uuid.NewString()

	raw, err := svc.GenerateAccessToken(userID, 0)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	_, err = svc.ValidateAccessToken(raw)
	if !errors.Is(err, token.ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}

func TestMalformedTokenIsRejected(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	_, err := svc.ValidateAccessToken("not.a.jwt")
	if !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}

	_, err = svc.ValidateAccessToken("")
	if !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for empty token, got %v", err)
	}
}

func TestTokenSignedWithWrongSecretIsRejected(t *testing.T) {
	svc := newService(t, 15*time.Minute)
	other := token.NewService([]byte("another-secret-that-is-longer-than-32-bytes-99"), 15*time.Minute)

	raw, err := other.GenerateAccessToken(uuid.NewString(), 0)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	_, err = svc.ValidateAccessToken(raw)
	if !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestUnexpectedSigningAlgorithmIsRejected(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	claims := jwt.RegisteredClaims{
		Subject:   uuid.NewString(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS384, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = svc.ValidateAccessToken(raw)
	if !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestMissingSubjectIsRejected(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = svc.ValidateAccessToken(raw)
	if !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestInvalidUUIDSubjectIsRejected(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	claims := jwt.RegisteredClaims{
		Subject:   "not-a-uuid",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = svc.ValidateAccessToken(raw)
	if !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestMissingExpirationIsRejected(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	claims := jwt.RegisteredClaims{
		Subject:  uuid.NewString(),
		IssuedAt: jwt.NewNumericDate(time.Now()),
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = svc.ValidateAccessToken(raw)
	if !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestLoadJWTConfig(t *testing.T) {
	t.Run("missing secret", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "")
		if _, err := config.LoadJWT(); err == nil {
			t.Fatal("expected error when JWT_SECRET is missing")
		}
	})

	t.Run("short secret", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "too-short")
		if _, err := config.LoadJWT(); err == nil {
			t.Fatal("expected error when JWT_SECRET is too short")
		}
	})

	t.Run("invalid ttl", func(t *testing.T) {
		t.Setenv("JWT_SECRET", testSecret)
		t.Setenv("JWT_ACCESS_TOKEN_TTL", "not-a-duration")
		if _, err := config.LoadJWT(); err == nil {
			t.Fatal("expected error when JWT_ACCESS_TOKEN_TTL is invalid")
		}
	})

	t.Run("default ttl", func(t *testing.T) {
		t.Setenv("JWT_SECRET", testSecret)
		t.Setenv("JWT_ACCESS_TOKEN_TTL", "")
		cfg, err := config.LoadJWT()
		if err != nil {
			t.Fatalf("LoadJWT: %v", err)
		}
		if cfg.AccessTokenTTL != 15*time.Minute {
			t.Fatalf("expected default ttl 15m, got %s", cfg.AccessTokenTTL)
		}
	})

	t.Run("custom ttl", func(t *testing.T) {
		t.Setenv("JWT_SECRET", testSecret)
		t.Setenv("JWT_ACCESS_TOKEN_TTL", "1h")
		cfg, err := config.LoadJWT()
		if err != nil {
			t.Fatalf("LoadJWT: %v", err)
		}
		if cfg.AccessTokenTTL != time.Hour {
			t.Fatalf("expected ttl 1h, got %s", cfg.AccessTokenTTL)
		}
	})
}
