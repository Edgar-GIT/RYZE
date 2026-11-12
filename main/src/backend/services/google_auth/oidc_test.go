package google_auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func newTestClient(fake *fakeGoogle) *oidcClient {
	return newOIDCClient(fake.endpoints(), testClientID, testSecret, testRedirectURL, fake.server.Client(), func() time.Time { return time.Now().UTC() })
}

func TestVerifyAcceptsAValidIDToken(t *testing.T) {
	fake := newFakeGoogle(t)
	client := newTestClient(fake)
	claims := idClaimsFor("1234567890", "rita@example.com")
	claims["nonce"] = "nonce-1"

	profile, err := client.verify(context.Background(), fake.sign(claims), "nonce-1")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if profile.Subject != "1234567890" || profile.Email != "rita@example.com" || profile.GivenName != "Rita" {
		t.Fatalf("unexpected profile %+v", profile)
	}
	// The key set is cached: a second token does not refetch it.
	if _, err := client.verify(context.Background(), fake.sign(claims), "nonce-1"); err != nil {
		t.Fatalf("second verify: %v", err)
	}
	if fake.keyFetch != 1 {
		t.Fatalf("expected one key fetch, got %d", fake.keyFetch)
	}
}

func TestVerifyRejectsInvalidTokens(t *testing.T) {
	fake := newFakeGoogle(t)
	cases := []struct {
		name   string
		mutate func(jwt.MapClaims)
		nonce  string
	}{
		{name: "wrong audience", mutate: func(c jwt.MapClaims) { c["aud"] = "another-client" }},
		{name: "extra audience", mutate: func(c jwt.MapClaims) { c["aud"] = []string{testClientID, "another-client"} }},
		{name: "wrong issuer", mutate: func(c jwt.MapClaims) { c["iss"] = "https://evil.example" }},
		{name: "expired", mutate: func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-10 * time.Minute).Unix() }},
		{name: "no expiry", mutate: func(c jwt.MapClaims) { delete(c, "exp") }},
		{name: "issued in the future", mutate: func(c jwt.MapClaims) { c["iat"] = time.Now().Add(30 * time.Minute).Unix() }},
		{name: "nonce mismatch", mutate: func(c jwt.MapClaims) { c["nonce"] = "other" }},
		{name: "missing nonce", mutate: func(c jwt.MapClaims) { delete(c, "nonce") }},
		{name: "missing subject", mutate: func(c jwt.MapClaims) { delete(c, "sub") }},
		{name: "missing email", mutate: func(c jwt.MapClaims) { delete(c, "email") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient(fake)
			claims := idClaimsFor("1234567890", "rita@example.com")
			claims["nonce"] = "nonce-1"
			tc.mutate(claims)
			if _, err := client.verify(context.Background(), fake.sign(claims), "nonce-1"); err == nil {
				t.Fatal("expected the token to be rejected")
			}
		})
	}
}

func TestVerifyRejectsUnverifiedEmail(t *testing.T) {
	fake := newFakeGoogle(t)
	client := newTestClient(fake)
	claims := idClaimsFor("1234567890", "rita@example.com")
	claims["nonce"] = "n"
	claims["email_verified"] = false
	if _, err := client.verify(context.Background(), fake.sign(claims), "n"); !errors.Is(err, errEmailUnverified) {
		t.Fatalf("expected errEmailUnverified, got %v", err)
	}
}

func TestVerifyRejectsForgedSignatures(t *testing.T) {
	fake := newFakeGoogle(t)
	client := newTestClient(fake)
	claims := idClaimsFor("1234567890", "rita@example.com")
	claims["nonce"] = "n"

	// Signed by a different key under the same key id.
	other := newFakeGoogle(t)
	if _, err := client.verify(context.Background(), other.sign(claims), "n"); err == nil {
		t.Fatal("token signed by another key was accepted")
	}

	// HMAC with the public modulus as the secret (algorithm confusion).
	hmacToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	hmacToken.Header["kid"] = fake.kid
	signed, err := hmacToken.SignedString(fake.key.PublicKey.N.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.verify(context.Background(), signed, "n"); err == nil {
		t.Fatal("HS256 token was accepted")
	}

	// Unsigned token.
	none := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	unsigned, err := none.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.verify(context.Background(), unsigned, "n"); err == nil {
		t.Fatal("unsigned token was accepted")
	}
}

func TestUnknownKeyRefetchIsThrottled(t *testing.T) {
	fake := newFakeGoogle(t)
	client := newTestClient(fake)
	claims := idClaimsFor("1", "rita@example.com")
	claims["nonce"] = "n"
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "unknown-kid"
	signed, err := token.SignedString(fake.key)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := client.verify(context.Background(), signed, "n"); err == nil {
			t.Fatal("token with an unknown key id was accepted")
		}
	}
	if fake.keyFetch != 1 {
		t.Fatalf("expected a single key fetch for repeated unknown key ids, got %d", fake.keyFetch)
	}
}

func TestExchangeEnforcesPKCEAndSingleUse(t *testing.T) {
	fake := newFakeGoogle(t)
	client := newTestClient(fake)
	verifier := "verifier-value-with-enough-entropy-000000000"
	authURL := client.authorizationURL("state", "nonce", pkceChallenge(verifier))
	_, code := fake.authorize(authURL, idClaimsFor("1", "rita@example.com"))

	if _, err := client.exchange(context.Background(), code, "wrong-verifier"); !errors.Is(err, errExchange) {
		t.Fatalf("expected an exchange error for a wrong verifier, got %v", err)
	}
	// The failed attempt consumed the code, as Google does.
	if _, err := client.exchange(context.Background(), code, verifier); err == nil {
		t.Fatal("a used code was exchanged again")
	}
	_, code = fake.authorize(authURL, idClaimsFor("1", "rita@example.com"))
	if _, err := client.exchange(context.Background(), code, verifier); err != nil {
		t.Fatalf("exchange: %v", err)
	}
}

func TestStateCookieIsSignedAndBounded(t *testing.T) {
	key := stateKey([]byte("secret"))
	now := time.Now()
	raw, err := signState(key, flowState{State: "s", Nonce: "n", Verifier: "v", Mode: ModeLogin}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseState(key, raw, now); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, err := parseState(stateKey([]byte("other")), raw, now); err == nil {
		t.Fatal("state signed with another key was accepted")
	}
	if _, err := parseState(key, raw, now.Add(stateTTL+2*time.Minute)); err == nil {
		t.Fatal("expired state was accepted")
	}
	parts := strings.Split(raw, ".")
	if _, err := parseState(key, parts[0]+"."+parts[1]+"x."+parts[2], now); err == nil {
		t.Fatal("tampered state was accepted")
	}
	link, err := signState(key, flowState{State: "s", Nonce: "n", Verifier: "v", Mode: ModeLink}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseState(key, link, now); err == nil {
		t.Fatal("link state without an account was accepted")
	}
}

func TestSafeReturnPath(t *testing.T) {
	cases := map[string]string{
		"/programs":             "/programs",
		"/programs?tab=mine":    "/programs?tab=mine",
		"":                      "",
		"https://evil.example":  "",
		"//evil.example":        "",
		"/\\evil.example":       "",
		"programs":              "",
		"/x\r\nSet-Cookie: a=b": "",
	}
	for input, want := range cases {
		if got := safeReturnPath(input); got != want {
			t.Errorf("safeReturnPath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNamesFallBackSensibly(t *testing.T) {
	cases := []struct {
		profile     Profile
		first, last string
	}{
		{Profile{GivenName: "Rita", FamilyName: "Costa"}, "Rita", "Costa"},
		{Profile{Name: "Rita Maria Costa"}, "Rita", "Maria Costa"},
		{Profile{Email: "rita.c@example.com"}, "rita.c", ""},
	}
	for _, tc := range cases {
		first, last := names(tc.profile)
		if first != tc.first || last != tc.last {
			t.Errorf("names(%+v) = %q %q, want %q %q", tc.profile, first, last, tc.first, tc.last)
		}
	}
}
