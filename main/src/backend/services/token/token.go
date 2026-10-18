package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"sync"
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

	// tokenIssuer and tokenAudience bind every token to this platform. They are
	// verified on parse, so a token signed with the same secret by another
	// service (or replayed against a different audience) is never accepted.
	tokenIssuer   = "ryze"
	tokenAudience = "ryze-api"

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
	// RegisterAdminCredentials records an administrator's secret credentials.
	// Only a non-reversible keyed version of the credentials is derived and
	// stored, never the credentials themselves. Admin tokens embed the version
	// current when they are issued; validation rejects a token whose version no
	// longer matches the registered one, so changing an administrator's password
	// or access code revokes every previously issued admin session.
	RegisterAdminCredentials(adminID, password, accessCode string)
}

// accessTokenClaims is the JWT payload shape. SessionVersion is carried in the
// "ver" claim so tokens issued for an older session (e.g. before a password
// change) can be rejected. Kind separates user tokens from admin tokens.
// AdminCredentialVersion is carried in the "av" claim and is only meaningful on
// admin tokens.
type accessTokenClaims struct {
	Kind                   string `json:"kind"`
	SessionVersion         int    `json:"ver"`
	AdminCredentialVersion string `json:"av,omitempty"`
	jwt.RegisteredClaims
}

type service struct {
	secret []byte
	ttl    time.Duration

	mu                      sync.RWMutex
	adminCredentialVersions map[string]string
}

func NewService(secret []byte, ttl time.Duration) Service {
	return &service{
		secret:                  secret,
		ttl:                     ttl,
		adminCredentialVersions: make(map[string]string),
	}
}

// RegisterAdminCredentials derives and stores the credential version for an
// administrator. The stored value is a keyed MAC of the admin identity and its
// credentials; it reveals nothing about the credentials without the signing
// secret and is never embedded verbatim as a credential.
func (s *service) RegisterAdminCredentials(adminID, password, accessCode string) {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte("ryze-admin-credential-version\x00"))
	mac.Write([]byte(adminID))
	mac.Write([]byte{0})
	mac.Write([]byte(password))
	mac.Write([]byte{0})
	mac.Write([]byte(accessCode))
	version := hex.EncodeToString(mac.Sum(nil))

	s.mu.Lock()
	s.adminCredentialVersions[adminID] = version
	s.mu.Unlock()
}

// adminCredentialVersion returns the registered version for an admin identity.
// The second result is false when no version was registered, which validation
// treats as a failure so an unregistered administrator can never authenticate.
func (s *service) adminCredentialVersion(adminID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	version, ok := s.adminCredentialVersions[adminID]
	return version, ok && version != ""
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
// identity (ADMIN_1/ADMIN_2) as subject. The token also carries the credential
// version registered for that identity, so a later credential change revokes
// it. Admin tokens never carry user session state and are only accepted by
// ValidateAdminToken.
func (s *service) GenerateAdminToken(adminID string) (string, error) {
	now := time.Now()
	version, _ := s.adminCredentialVersion(adminID)
	claims := accessTokenClaims{
		Kind:                   kindAdmin,
		AdminCredentialVersion: version,
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
// the expected one, enforces an expiration claim, requires the user token kind
// (a token without the expected kind is never interpreted as a user token and
// therefore never accepted) and returns the user UUID and session version
// stored in the token.
func (s *service) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims, err := s.parse(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.Kind != kindUser {
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
// expected one, enforces an expiration claim, requires the admin token kind,
// rejects user and stage tokens, and requires the credential version carried by
// the token to match the version currently registered for the admin identity.
// An identity with no registered credentials fails closed, so an admin token is
// never accepted without a known credential version. It returns the admin
// identity stored as the token subject.
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

	expected, ok := s.adminCredentialVersion(adminID)
	if !ok || subtle.ConstantTimeCompare([]byte(expected), []byte(claims.AdminCredentialVersion)) != 1 {
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
