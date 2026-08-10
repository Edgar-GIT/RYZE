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

// Claims holds the data embedded in a validated access token.
type Claims struct {
	UserID         string
	SessionVersion int
}

// Service generates and validates access tokens for authenticated users.
type Service interface {
	GenerateAccessToken(userID string, sessionVersion int) (string, error)
	ValidateAccessToken(tokenString string) (*Claims, error)
}

// accessTokenClaims is the JWT payload shape. SessionVersion is carried in the
// "ver" claim so tokens issued for an older session (e.g. before a password
// change) can be rejected.
type accessTokenClaims struct {
	SessionVersion int `json:"ver"`
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
		SessionVersion: sessionVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

// ValidateAccessToken verifies the signature, rejects algorithms other than
// the expected one, enforces an expiration claim, and returns the user UUID
// and session version stored in the token.
func (s *service) ValidateAccessToken(tokenString string) (*Claims, error) {
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

	userID, err := claims.GetSubject()
	if err != nil || userID == "" {
		return nil, ErrInvalidToken
	}
	if _, err := uuid.Parse(userID); err != nil {
		return nil, ErrInvalidToken
	}
	return &Claims{UserID: userID, SessionVersion: claims.SessionVersion}, nil
}
