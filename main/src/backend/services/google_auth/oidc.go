package google_auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Google's OpenID Connect endpoints. Tests point a Provider at local servers.
const (
	defaultAuthURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	defaultTokenURL = "https://oauth2.googleapis.com/token"
	defaultJWKSURL  = "https://www.googleapis.com/oauth2/v3/certs"

	maxResponseBytes = 1 << 20
	httpTimeout      = 10 * time.Second
	// The key set is refetched for an unknown key id at most this often, so a
	// forged token cannot make the server hammer Google.
	minKeyRefresh   = 30 * time.Second
	defaultKeysTTL  = time.Hour
	idTokenLeeway   = time.Minute
	maxIDTokenBytes = 16 << 10
)

// validIssuers are the two issuer spellings Google documents for ID tokens.
var validIssuers = map[string]bool{"https://accounts.google.com": true, "accounts.google.com": true}

// Endpoints are the provider URLs used by the flow.
type Endpoints struct {
	AuthURL  string
	TokenURL string
	JWKSURL  string
}

// GoogleEndpoints returns Google's production endpoints.
func GoogleEndpoints() Endpoints {
	return Endpoints{AuthURL: defaultAuthURL, TokenURL: defaultTokenURL, JWKSURL: defaultJWKSURL}
}

// idClaims are the ID token claims RYZE reads.
type idClaims struct {
	Email         string `json:"email"`
	EmailVerified any    `json:"email_verified"`
	Nonce         string `json:"nonce"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	jwt.RegisteredClaims
}

// emailVerified accepts the boolean Google sends (and the string form some
// libraries produce).
func (c idClaims) emailVerified() bool {
	switch value := c.EmailVerified.(type) {
	case bool:
		return value
	case string:
		return value == "true"
	default:
		return false
	}
}

// Profile is the verified identity taken from an ID token.
type Profile struct {
	Subject    string
	Email      string
	GivenName  string
	FamilyName string
	Name       string
}

var (
	errExchange   = errors.New("authorization code exchange failed")
	errInvalidIDT = errors.New("invalid id token")
)

// oidcClient performs the code exchange and ID token verification.
type oidcClient struct {
	endpoints    Endpoints
	clientID     string
	clientSecret string
	redirectURL  string
	http         *http.Client
	now          func() time.Time

	mu          sync.Mutex
	keys        map[string]*rsa.PublicKey
	keysExpiry  time.Time
	lastRefresh time.Time
}

func newOIDCClient(endpoints Endpoints, clientID, clientSecret, redirectURL string, client *http.Client, now func() time.Time) *oidcClient {
	if client == nil {
		client = &http.Client{Timeout: httpTimeout}
	}
	return &oidcClient{
		endpoints:    endpoints,
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		http:         client,
		now:          now,
	}
}

// authorizationURL builds the consent-screen URL for one sign-in attempt.
func (c *oidcClient) authorizationURL(state, nonce, challenge string) string {
	query := url.Values{
		"client_id":             {c.clientID},
		"redirect_uri":          {c.redirectURL},
		"response_type":         {"code"},
		"scope":                 {"openid email profile"},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"prompt":                {"select_account"},
	}
	return c.endpoints.AuthURL + "?" + query.Encode()
}

// exchange trades the authorization code for an ID token. The response body
// is never logged: it carries tokens.
func (c *oidcClient) exchange(ctx context.Context, code, verifier string) (string, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"redirect_uri":  {c.redirectURL},
		"code_verifier": {verifier},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoints.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("%w: %v", errExchange, err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return "", fmt.Errorf("%w: token endpoint unreachable", errExchange)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("%w: reading token response", errExchange)
	}
	var payload struct {
		IDToken string `json:"id_token"`
		Error   string `json:"error"`
	}
	_ = json.Unmarshal(body, &payload)
	if response.StatusCode != http.StatusOK {
		// Only Google's short error code is kept (e.g. "invalid_grant").
		return "", fmt.Errorf("%w: status %d %s", errExchange, response.StatusCode, safeCode(payload.Error))
	}
	if payload.IDToken == "" || len(payload.IDToken) > maxIDTokenBytes {
		return "", fmt.Errorf("%w: no id token", errExchange)
	}
	return payload.IDToken, nil
}

// safeCode keeps an OAuth error code printable and short.
func safeCode(code string) string {
	if len(code) > 40 {
		code = code[:40]
	}
	return strings.Map(func(r rune) rune {
		if r == '_' || r == '-' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, code)
}

// verify checks the ID token signature against Google's published keys and
// validates issuer, audience, expiry, issue time, nonce and email
// verification.
func (c *oidcClient) verify(ctx context.Context, rawToken, expectedNonce string) (Profile, error) {
	claims := &idClaims{}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithAudience(c.clientID),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(idTokenLeeway),
		jwt.WithTimeFunc(c.now),
	)
	_, err := parser.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("missing key id")
		}
		return c.key(ctx, kid)
	})
	if err != nil {
		return Profile{}, fmt.Errorf("%w: %v", errInvalidIDT, err)
	}
	if !validIssuers[claims.Issuer] {
		return Profile{}, fmt.Errorf("%w: unexpected issuer", errInvalidIDT)
	}
	// Google issues single-audience tokens; reject any extra audience so a
	// token minted for another client is never accepted.
	if len(claims.Audience) != 1 {
		return Profile{}, fmt.Errorf("%w: unexpected audience", errInvalidIDT)
	}
	if expectedNonce == "" || !constantTimeEqual(claims.Nonce, expectedNonce) {
		return Profile{}, fmt.Errorf("%w: nonce mismatch", errInvalidIDT)
	}
	if claims.Subject == "" || len(claims.Subject) > 255 {
		return Profile{}, fmt.Errorf("%w: missing subject", errInvalidIDT)
	}
	email := strings.TrimSpace(claims.Email)
	if email == "" || len(email) > 255 || !strings.Contains(email, "@") {
		return Profile{}, fmt.Errorf("%w: missing email", errInvalidIDT)
	}
	if !claims.emailVerified() {
		return Profile{}, errEmailUnverified
	}
	return Profile{
		Subject:    claims.Subject,
		Email:      email,
		GivenName:  strings.TrimSpace(claims.GivenName),
		FamilyName: strings.TrimSpace(claims.FamilyName),
		Name:       strings.TrimSpace(claims.Name),
	}, nil
}

// key returns the public key for kid, refreshing the cached key set when it
// expired or does not know the key yet.
func (c *oidcClient) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if key, ok := c.keys[kid]; ok && now.Before(c.keysExpiry) {
		return key, nil
	}
	if !c.lastRefresh.IsZero() && now.Sub(c.lastRefresh) < minKeyRefresh {
		if key, ok := c.keys[kid]; ok {
			return key, nil
		}
		return nil, errors.New("unknown signing key")
	}
	c.lastRefresh = now
	keys, ttl, err := c.fetchKeys(ctx)
	if err != nil {
		if key, ok := c.keys[kid]; ok {
			// Keep using a known key while Google is unreachable.
			return key, nil
		}
		return nil, err
	}
	c.keys = keys
	c.keysExpiry = now.Add(ttl)
	if key, ok := keys[kid]; ok {
		return key, nil
	}
	return nil, errors.New("unknown signing key")
}

func (c *oidcClient) fetchKeys(ctx context.Context) (map[string]*rsa.PublicKey, time.Duration, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoints.JWKSURL, nil)
	if err != nil {
		return nil, 0, err
	}
	response, err := c.http.Do(request)
	if err != nil {
		return nil, 0, errors.New("signing keys unreachable")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("signing keys status %d", response.StatusCode)
	}
	var set struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			Use string `json:"use"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes)).Decode(&set); err != nil {
		return nil, 0, errors.New("signing keys unreadable")
	}
	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, jwk := range set.Keys {
		if jwk.Kty != "RSA" || jwk.Kid == "" || (jwk.Use != "" && jwk.Use != "sig") {
			continue
		}
		key, err := rsaKey(jwk.N, jwk.E)
		if err != nil {
			continue
		}
		keys[jwk.Kid] = key
	}
	if len(keys) == 0 {
		return nil, 0, errors.New("no usable signing keys")
	}
	return keys, cacheTTL(response.Header.Get("Cache-Control")), nil
}

// rsaKey decodes a JWK modulus and exponent.
func rsaKey(n, e string) (*rsa.PublicKey, error) {
	modulus, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil || len(modulus) < 256 {
		return nil, errors.New("invalid modulus")
	}
	exponent, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil || len(exponent) == 0 || len(exponent) > 4 {
		return nil, errors.New("invalid exponent")
	}
	value := 0
	for _, b := range exponent {
		value = value<<8 | int(b)
	}
	if value < 3 {
		return nil, errors.New("invalid exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(modulus), E: value}, nil
}

// cacheTTL reads max-age from a Cache-Control header, bounded to a sane range.
func cacheTTL(header string) time.Duration {
	for _, directive := range strings.Split(header, ",") {
		directive = strings.TrimSpace(directive)
		if value, ok := strings.CutPrefix(directive, "max-age="); ok {
			seconds, err := strconv.Atoi(value)
			if err != nil {
				break
			}
			ttl := time.Duration(seconds) * time.Second
			if ttl < 5*time.Minute {
				return 5 * time.Minute
			}
			if ttl > 24*time.Hour {
				return 24 * time.Hour
			}
			return ttl
		}
	}
	return defaultKeysTTL
}
