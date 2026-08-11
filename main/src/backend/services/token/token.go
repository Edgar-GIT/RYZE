package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	// ErrInvalidToken covers malformed tokens, tokens signed with another
	// secret, unexpected signing algorithms and missing/invalid claims.
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken is returned when the token's expiration time has passed.
	ErrExpiredToken = errors.New("token has expired")
)

var signingAlgorithm = jwt.SigningMethodHS256.Alg()

const (
	kindUser       = "user"
	kindAdmin      = "admin"
	kindAdminStage = "admin-stage"

	// AdminStageTokenTTL is the lifetime of the temporary authentication state
	// issued after the username/password stage of the admin login flow. The
	// state expires quickly and must be completed with the access code before
	// the final admin session is created.
	AdminStageTokenTTL = 5 * time.Minute
)

// Claims holds the data embedded in a validated access token.
type Claims struct {
	UserID         string
	SessionVersion int
}

// Service generates and validates access tokens for authenticated users and
// administrators. Both token kinds share the same signing secret so a single
// secret configures the whole platform; they are kept distinct through the
// "kind" claim so a user token can never be accepted as an admin token and
// vice versa.
type Service interface {
	GenerateAccessToken(userID string, sessionVersion int) (string, error)
	ValidateAccessToken(tokenString string) (*Claims, error)
	GenerateAdminToken(adminID string) (string, error)
	ValidateAdminToken(tokenString string) (string, error)
	GenerateAdminStageToken(adminID string) (string, error)
	ValidateAdminStageToken(tokenString string) (string, error)
}

// accessTokenClaims is the JWT payload shape. SessionVersion is carried in the
// "ver" claim so tokens issued for an older session (e.g. before a password
// change) can be rejected. Kind separates user tokens from admin tokens.
type accessTokenClaims struct {
	Kind           string `json:"kind"`
	SessionVersion int    `json:"ver"`
	jwt.RegisteredClaims
}

type service struct {
	secret []byte
	ttl    time.Duration
}

func NewService(secret []byte, ttl time.Duration) Service {
	return &service{secret: secret, ttl: ttl}
}

// GenerateAccessToken returns a signed HMAC-SHA256 JWT carrying only the
// user UUID as subject, the session version and issued-at/expiration claims.
// No personal information is embedded in the token.
func (s *service) GenerateAccessToken(userID string, sessionVersion int) (string, error) {
	now := time.Now()
	claims := accessTokenClaims{
		Kind:           kindUser,
		SessionVersion: sessionVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

// GenerateAdminToken returns a signed HMAC-SHA256 JWT carrying the admin
// identity (ADMIN_1/ADMIN_2) as subject. Admin tokens never carry user session
// state and are only accepted by ValidateAdminToken.
func (s *service) GenerateAdminToken(adminID string) (string, error) {
	now := time.Now()
	claims := accessTokenClaims{
		Kind: kindAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   adminID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

// GenerateAdminStageToken returns the short-lived temporary authentication
// state issued after the username/password stage of the admin login flow. It
// identifies the intended admin identity and only proves that stage one
// succeeded; it is never accepted as a final admin session.
func (s *service) GenerateAdminStageToken(adminID string) (string, error) {
	now := time.Now()
	claims := accessTokenClaims{
		Kind: kindAdminStage,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   adminID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AdminStageTokenTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

// ValidateAccessToken verifies the signature, rejects algorithms other than
// the expected one, enforces an expiration claim, rejects admin tokens and
// returns the user UUID and session version stored in the token.
func (s *service) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims, err := s.parse(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.Kind != "" && claims.Kind != kindUser {
		return nil, ErrInvalidToken
	}

	userID, err := claims.GetSubject()
	if err != nil || userID == "" {
		return nil, ErrInvalidToken
	}
	if _, err := uuid.Parse(userID); err != nil {
		return nil, ErrInvalidToken
	}
	return &Claims{UserID: userID, SessionVersion: claims.SessionVersion}, nil
}

// ValidateAdminToken verifies the signature, rejects algorithms other than the
// expected one, enforces an expiration claim, rejects user and stage tokens
// and returns the admin identity stored as the token subject.
func (s *service) ValidateAdminToken(tokenString string) (string, error) {
	claims, err := s.parse(tokenString)
	if err != nil {
		return "", err
	}
	if claims.Kind != kindAdmin {
		return "", ErrInvalidToken
	}

	adminID, err := claims.GetSubject()
	if err != nil || adminID == "" {
		return "", ErrInvalidToken
	}
	return adminID, nil
}

// ValidateAdminStageToken verifies the signature, rejects algorithms other
// than the expected one, enforces an expiration claim, rejects every other
// token kind and returns the admin identity stored as the token subject. Only
// tokens issued by GenerateAdminStageToken are accepted, so a stage state can
// never be used as the final admin session.
func (s *service) ValidateAdminStageToken(tokenString string) (string, error) {
	claims, err := s.parse(tokenString)
	if err != nil {
		return "", err
	}
	if claims.Kind != kindAdminStage {
		return "", ErrInvalidToken
	}

	adminID, err := claims.GetSubject()
	if err != nil || adminID == "" {
		return "", ErrInvalidToken
	}
	return adminID, nil
}

// parse performs the signature, algorithm and expiration validation shared by
// both token kinds.
func (s *service) parse(tokenString string) (*accessTokenClaims, error) {
	claims := &accessTokenClaims{}
	parsed, err := jwt.ParseWithClaims(tokenString, claims,
		func(_ *jwt.Token) (any, error) { return s.secret, nil },
		jwt.WithValidMethods([]string{signingAlgorithm}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	if !parsed.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
