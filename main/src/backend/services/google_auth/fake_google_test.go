package google_auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testClientID    = "client-123.apps.googleusercontent.com"
	testSecret      = "client-secret"
	testRedirectURL = "http://localhost:8080/api/v1/auth/google/callback"
)

// fakeGoogle serves the token and key endpoints of an OpenID provider and
// enforces PKCE like Google does.
type fakeGoogle struct {
	t      *testing.T
	key    *rsa.PrivateKey
	kid    string
	server *httptest.Server

	mu       sync.Mutex
	grants   map[string]grant
	keyFetch int
}

type grant struct {
	challenge string
	claims    jwt.MapClaims
}

func newFakeGoogle(t *testing.T) *fakeGoogle {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeGoogle{t: t, key: key, kid: "test-key-1", grants: map[string]grant{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/token", fake.token)
	mux.HandleFunc("/certs", fake.certs)
	fake.server = httptest.NewServer(mux)
	t.Cleanup(fake.server.Close)
	return fake
}

func (f *fakeGoogle) endpoints() Endpoints {
	return Endpoints{AuthURL: f.server.URL + "/auth", TokenURL: f.server.URL + "/token", JWKSURL: f.server.URL + "/certs"}
}

func (f *fakeGoogle) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || r.Method != http.MethodPost {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if r.Form.Get("client_id") != testClientID || r.Form.Get("client_secret") != testSecret ||
		r.Form.Get("redirect_uri") != testRedirectURL || r.Form.Get("grant_type") != "authorization_code" {
		http.Error(w, `{"error":"invalid_client"}`, http.StatusUnauthorized)
		return
	}
	f.mu.Lock()
	issued, ok := f.grants[r.Form.Get("code")]
	delete(f.grants, r.Form.Get("code")) // codes are single use
	f.mu.Unlock()
	if !ok || pkceChallenge(r.Form.Get("code_verifier")) != issued.challenge {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"id_token": f.sign(issued.claims), "token_type": "Bearer"})
}

func (f *fakeGoogle) certs(w http.ResponseWriter, _ *http.Request) {
	f.mu.Lock()
	f.keyFetch++
	f.mu.Unlock()
	public := f.key.PublicKey
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
		"kty": "RSA",
		"alg": "RS256",
		"use": "sig",
		"kid": f.kid,
		"n":   base64.RawURLEncoding.EncodeToString(public.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(public.E)).Bytes()),
	}}})
}

func (f *fakeGoogle) sign(claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = f.kid
	signed, err := token.SignedString(f.key)
	if err != nil {
		f.t.Fatal(err)
	}
	return signed
}

// idClaimsFor returns valid ID token claims for a Google account.
func idClaimsFor(subject, address string) jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss":            "https://accounts.google.com",
		"aud":            testClientID,
		"sub":            subject,
		"email":          address,
		"email_verified": true,
		"given_name":     "Rita",
		"family_name":    "Costa",
		"iat":            now.Unix(),
		"exp":            now.Add(time.Hour).Unix(),
	}
}

// authorize plays the consent screen: it reads the state, nonce and PKCE
// challenge from the authorization URL and issues a code whose ID token
// carries the claims (the nonce is filled in unless the claims set one).
func (f *fakeGoogle) authorize(authURL string, claims jwt.MapClaims) (state, code string) {
	f.t.Helper()
	parsed, err := url.Parse(authURL)
	if err != nil {
		f.t.Fatal(err)
	}
	query := parsed.Query()
	if query.Get("code_challenge_method") != "S256" || query.Get("client_id") != testClientID || query.Get("redirect_uri") != testRedirectURL {
		f.t.Fatalf("unexpected authorization request: %s", authURL)
	}
	if _, set := claims["nonce"]; !set {
		claims["nonce"] = query.Get("nonce")
	}
	code = "code-" + randomString(f.t)
	f.mu.Lock()
	f.grants[code] = grant{challenge: query.Get("code_challenge"), claims: claims}
	f.mu.Unlock()
	return query.Get("state"), code
}

func randomString(t *testing.T) string {
	t.Helper()
	value, err := randomToken(12)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
