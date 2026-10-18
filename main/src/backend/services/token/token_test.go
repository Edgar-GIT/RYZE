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

// The issuer and audience are part of the token wire format; tests pin the
// expected values so a regression in either is caught.
const (
	testIssuer   = "ryze"
	testAudience = "ryze-api"
)

// standardClaims builds the standard claim set every test-forged token needs
// to pass the shared issuer/audience/expiration validation.
func standardClaims(subject string, issuedAt, expiresAt time.Time) jwt.RegisteredClaims {
	return jwt.RegisteredClaims{
		Issuer:    testIssuer,
		Subject:   subject,
		Audience:  jwt.ClaimStrings{testAudience},
		IssuedAt:  jwt.NewNumericDate(issuedAt),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}
}

// adminTestClaims embeds the kind claim so tests can forge admin tokens.
type adminTestClaims struct {
	Kind string `json:"kind"`
	jwt.RegisteredClaims
}

func tokenAdminClaimsWith(claims jwt.RegisteredClaims) jwt.Claims {
	return adminTestClaims{Kind: "admin", RegisteredClaims: claims}
}

// adminStageTestClaims embeds the kind claim so tests can forge stage tokens.
type adminStageTestClaims struct {
	Kind string `json:"kind"`
	jwt.RegisteredClaims
}

func tokenAdminStageClaimsWith(claims jwt.RegisteredClaims) jwt.Claims {
	return adminStageTestClaims{Kind: "admin-stage", RegisteredClaims: claims}
}

// userTestClaims embeds the kind claim so tests can forge user tokens.
type userTestClaims struct {
	Kind string `json:"kind"`
	jwt.RegisteredClaims
}

func tokenUserClaimsWith(claims jwt.RegisteredClaims) jwt.Claims {
	return userTestClaims{Kind: "user", RegisteredClaims: claims}
}

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

func TestGenerateAndValidateAdminToken(t *testing.T) {
	svc := newService(t, 15*time.Minute)
	svc.RegisterAdminCredentials("ADMIN_1", "admin-password", "admin-access-code")

	raw, err := svc.GenerateAdminToken("ADMIN_1")
	if err != nil {
		t.Fatalf("GenerateAdminToken: %v", err)
	}

	adminID, err := svc.ValidateAdminToken(raw)
	if err != nil {
		t.Fatalf("ValidateAdminToken: %v", err)
	}
	if adminID != "ADMIN_1" {
		t.Fatalf("expected admin identity %q, got %q", "ADMIN_1", adminID)
	}
}

func TestAdminTokenRejectedAfterCredentialChange(t *testing.T) {
	svc := newService(t, 15*time.Minute)
	svc.RegisterAdminCredentials("ADMIN_1", "admin-password", "admin-access-code")

	raw, err := svc.GenerateAdminToken("ADMIN_1")
	if err != nil {
		t.Fatalf("GenerateAdminToken: %v", err)
	}
	if _, err := svc.ValidateAdminToken(raw); err != nil {
		t.Fatalf("token must validate before the credential change: %v", err)
	}

	// Rotating the password (or access code) must revoke the previously issued
	// admin session.
	svc.RegisterAdminCredentials("ADMIN_1", "rotated-password", "admin-access-code")
	if _, err := svc.ValidateAdminToken(raw); !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("token issued before a credential change must be rejected, got %v", err)
	}
}

func TestAdminTokenRejectedWithoutRegisteredCredentials(t *testing.T) {
	svc := newService(t, 15*time.Minute)
	// No credentials registered for ADMIN_1: validation must fail closed.
	raw, err := svc.GenerateAdminToken("ADMIN_1")
	if err != nil {
		t.Fatalf("GenerateAdminToken: %v", err)
	}
	if _, err := svc.ValidateAdminToken(raw); !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("admin token must be rejected when no credentials are registered, got %v", err)
	}
}

func TestAdminTokenRejectedAsUserToken(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	raw, err := svc.GenerateAdminToken("ADMIN_1")
	if err != nil {
		t.Fatalf("GenerateAdminToken: %v", err)
	}

	if _, err := svc.ValidateAccessToken(raw); !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("admin token must never validate as a user token, got %v", err)
	}
}

func TestUserTokenRejectedAsAdminToken(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	raw, err := svc.GenerateAccessToken(uuid.NewString(), 0)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	if _, err := svc.ValidateAdminToken(raw); !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("user token must never validate as an admin token, got %v", err)
	}
}

func TestAdminTokenEmptySubjectRejected(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	if _, err := svc.ValidateAdminToken(""); !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for empty token, got %v", err)
	}

	claims := standardClaims("", time.Now(), time.Now().Add(15*time.Minute))
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenAdminClaimsWith(claims)).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	if _, err := svc.ValidateAdminToken(raw); !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for a token without a subject, got %v", err)
	}
}

func TestAdminTokenWrongSecretRejected(t *testing.T) {
	svc := newService(t, 15*time.Minute)
	other := token.NewService([]byte("another-secret-that-is-longer-than-32-bytes-99"), 15*time.Minute)

	raw, err := other.GenerateAdminToken("ADMIN_2")
	if err != nil {
		t.Fatalf("GenerateAdminToken: %v", err)
	}

	if _, err := svc.ValidateAdminToken(raw); !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestExpiredAdminTokenIsRejected(t *testing.T) {
	svc := newService(t, -1*time.Minute)

	raw, err := svc.GenerateAdminToken("ADMIN_1")
	if err != nil {
		t.Fatalf("GenerateAdminToken: %v", err)
	}

	if _, err := svc.ValidateAdminToken(raw); !errors.Is(err, token.ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}

func TestGenerateAndValidateAdminStageToken(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	raw, err := svc.GenerateAdminStageToken("ADMIN_2")
	if err != nil {
		t.Fatalf("GenerateAdminStageToken: %v", err)
	}

	adminID, err := svc.ValidateAdminStageToken(raw)
	if err != nil {
		t.Fatalf("ValidateAdminStageToken: %v", err)
	}
	if adminID != "ADMIN_2" {
		t.Fatalf("expected admin identity %q, got %q", "ADMIN_2", adminID)
	}
}

func TestStageTokenRejectedAsAdminToken(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	raw, err := svc.GenerateAdminStageToken("ADMIN_1")
	if err != nil {
		t.Fatalf("GenerateAdminStageToken: %v", err)
	}

	if _, err := svc.ValidateAdminToken(raw); !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("stage token must never validate as a final admin token, got %v", err)
	}
}

func TestStageTokenRejectedAsUserToken(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	raw, err := svc.GenerateAdminStageToken("ADMIN_1")
	if err != nil {
		t.Fatalf("GenerateAdminStageToken: %v", err)
	}

	if _, err := svc.ValidateAccessToken(raw); !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("stage token must never validate as a user token, got %v", err)
	}
}

func TestAdminTokenRejectedAsStageToken(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	raw, err := svc.GenerateAdminToken("ADMIN_1")
	if err != nil {
		t.Fatalf("GenerateAdminToken: %v", err)
	}

	if _, err := svc.ValidateAdminStageToken(raw); !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("admin token must never validate as a stage token, got %v", err)
	}
}

func TestUserTokenRejectedAsStageToken(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	raw, err := svc.GenerateAccessToken(uuid.NewString(), 0)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	if _, err := svc.ValidateAdminStageToken(raw); !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("user token must never validate as a stage token, got %v", err)
	}
}

func TestExpiredAdminStageTokenIsRejected(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	claims := tokenAdminStageClaimsWith(standardClaims("ADMIN_1", time.Now().Add(-6*time.Minute), time.Now().Add(-1*time.Minute)))
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := svc.ValidateAdminStageToken(raw); !errors.Is(err, token.ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}

func TestStageTokenEmptySubjectRejected(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	claims := tokenAdminStageClaimsWith(standardClaims("", time.Now(), time.Now().Add(15*time.Minute)))
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := svc.ValidateAdminStageToken(raw); !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for a stage token without a subject, got %v", err)
	}
}

func TestMissingSessionVersionDefaultsToZero(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	claims := tokenUserClaimsWith(standardClaims(uuid.NewString(), time.Now(), time.Now().Add(15*time.Minute)))
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

func TestTokenWithoutKindRejected(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	// A well-formed, correctly signed token that does not carry the expected
	// kind claim must never be interpreted as a valid user token.
	claims := standardClaims(uuid.NewString(), time.Now(), time.Now().Add(15*time.Minute))
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := svc.ValidateAccessToken(raw); !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("token without the user kind must be rejected, got %v", err)
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

func TestTokenWithoutIssuerIsRejected(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	// Same secret, same audience, but no issuer claim: only tokens minted by
	// this platform may authenticate.
	claims := tokenUserClaimsWith(jwt.RegisteredClaims{
		Subject:   uuid.NewString(),
		Audience:  jwt.ClaimStrings{testAudience},
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
	})
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := svc.ValidateAccessToken(raw); !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for a token without an issuer, got %v", err)
	}
}

func TestTokenWithWrongAudienceIsRejected(t *testing.T) {
	svc := newService(t, 15*time.Minute)

	claims := tokenUserClaimsWith(jwt.RegisteredClaims{
		Issuer:    testIssuer,
		Subject:   uuid.NewString(),
		Audience:  jwt.ClaimStrings{"other-service"},
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
	})
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := svc.ValidateAccessToken(raw); !errors.Is(err, token.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for a token minted for another audience, got %v", err)
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

	claims := standardClaims("", time.Now(), time.Now().Add(15*time.Minute))
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

	claims := standardClaims("not-a-uuid", time.Now(), time.Now().Add(15*time.Minute))
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
		Issuer:   testIssuer,
		Audience: jwt.ClaimStrings{testAudience},
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

	t.Run("ttl above ceiling", func(t *testing.T) {
		t.Setenv("JWT_SECRET", testSecret)
		t.Setenv("JWT_ACCESS_TOKEN_TTL", "25h")
		if _, err := config.LoadJWT(); err == nil {
			t.Fatal("expected error when JWT_ACCESS_TOKEN_TTL exceeds the ceiling")
		}
	})

	t.Run("ttl at ceiling", func(t *testing.T) {
		t.Setenv("JWT_SECRET", testSecret)
		t.Setenv("JWT_ACCESS_TOKEN_TTL", "24h")
		cfg, err := config.LoadJWT()
		if err != nil {
			t.Fatalf("LoadJWT: %v", err)
		}
		if cfg.AccessTokenTTL != 24*time.Hour {
			t.Fatalf("expected ttl 24h, got %s", cfg.AccessTokenTTL)
		}
	})
}
