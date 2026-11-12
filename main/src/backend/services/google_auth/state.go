package google_auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// stateTTL bounds how long the consent screen may stay open.
	stateTTL      = 10 * time.Minute
	stateAudience = "ryze-google-state"
	stateIssuer   = "ryze"
)

var errInvalidState = errors.New("invalid sign-in state")

// flowState is what the browser carries between the start and the callback,
// in a signed HttpOnly cookie. It binds the callback to the browser that
// started the flow (state), the ID token to this attempt (nonce) and the code
// exchange to this browser (PKCE verifier).
type flowState struct {
	State    string `json:"st"`
	Nonce    string `json:"nn"`
	Verifier string `json:"cv"`
	Mode     Mode   `json:"md"`
	UserID   string `json:"uid,omitempty"`
	Return   string `json:"rt,omitempty"`
	Language string `json:"lg,omitempty"`
	jwt.RegisteredClaims
}

// stateKey derives the state-cookie signing key from the platform secret, so
// a state cookie can never be confused with an access token.
func stateKey(secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte("ryze/google-sign-in/state/v1"))
	return mac.Sum(nil)
}

func signState(key []byte, state flowState, now time.Time) (string, error) {
	state.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    stateIssuer,
		Audience:  jwt.ClaimStrings{stateAudience},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(stateTTL)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, state).SignedString(key)
}

func parseState(key []byte, raw string, now time.Time) (*flowState, error) {
	if raw == "" || len(raw) > 4096 {
		return nil, errInvalidState
	}
	state := &flowState{}
	_, err := jwt.ParseWithClaims(raw, state, func(*jwt.Token) (any, error) { return key, nil },
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(stateIssuer),
		jwt.WithAudience(stateAudience),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	if err != nil || state.State == "" || state.Nonce == "" || state.Verifier == "" {
		return nil, errInvalidState
	}
	if state.Mode != ModeLogin && state.Mode != ModeLink {
		return nil, errInvalidState
	}
	if state.Mode == ModeLink && state.UserID == "" {
		return nil, errInvalidState
	}
	return state, nil
}

// randomToken returns n random bytes, base64url encoded without padding.
func randomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// pkceChallenge is the S256 code challenge for a verifier.
func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// safeReturnPath accepts only a same-app absolute path, so the callback can
// never redirect the browser to another site.
func safeReturnPath(value string) string {
	if value == "" || len(value) > 200 || !strings.HasPrefix(value, "/") ||
		strings.HasPrefix(value, "//") || strings.HasPrefix(value, "/\\") {
		return ""
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f || r == '\\' {
			return ""
		}
	}
	return value
}
