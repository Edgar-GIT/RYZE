package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/models"
	"ryze/backend/services/google_auth"
	"ryze/backend/services/token"
)

const (
	googleUserID   = "5f0c2f6e-1d7a-4c0e-9a59-0c1b8f7f3a11"
	googleAppURL   = "https://app.ryze.example"
	googleCookieAt = "/api/v1/auth/google"
)

// stubGoogle records what the handler passes to the service.
type stubGoogle struct {
	enabled bool
	outcome google_auth.Outcome

	beganMode   google_auth.Mode
	beganUser   string
	beganReturn string
	completed   bool
	gotCookie   string
	gotCallback google_auth.Callback
	gotSession  string
}

func (s *stubGoogle) Enabled() bool { return s.enabled }

func (s *stubGoogle) Begin(mode google_auth.Mode, userID, returnPath, _ string) (google_auth.Start, error) {
	s.beganMode, s.beganUser, s.beganReturn = mode, userID, returnPath
	return google_auth.Start{URL: "https://accounts.google.test/auth?state=abc", StateCookie: "signed-state"}, nil
}

func (s *stubGoogle) Complete(_ context.Context, cookie string, callback google_auth.Callback, session string) google_auth.Outcome {
	s.completed, s.gotCookie, s.gotCallback, s.gotSession = true, cookie, callback, session
	return s.outcome
}

func (s *stubGoogle) Summary(context.Context, string) (google_auth.Summary, error) {
	return google_auth.Summary{}, nil
}

func (s *stubGoogle) Unlink(context.Context, string) error { return nil }

type fixedSessions struct{ version int }

func (f fixedSessions) GetSessionVersion(context.Context, string) (int, error) { return f.version, nil }

func newGoogleRouter(stub *stubGoogle, tokens token.Service, appURL string) *gin.Engine {
	handler := auth.NewGoogleHandler(stub, tokens, fixedSessions{version: 3}, time.Hour, true, appURL, googleCookieAt)
	router := gin.New()
	router.GET("/api/v1/auth/providers", handler.Providers)
	router.GET("/api/v1/auth/google/start", handler.Start)
	router.GET("/api/v1/auth/google/callback", handler.Callback)
	return router
}

func cookieNamed(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func TestGoogleProvidersAndUnavailableStart(t *testing.T) {
	stub := &stubGoogle{}
	router := newGoogleRouter(stub, token.NewService([]byte("secret"), time.Hour), googleAppURL)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/auth/providers", nil))
	var payload struct {
		Data map[string]bool `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil || payload.Data["google"] {
		t.Fatalf("expected google=false, got %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/start", nil))
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != googleAppURL+"/login?oauth=unavailable" {
		t.Fatalf("expected a redirect to the unavailable notice, got %d %s", rec.Code, rec.Header().Get("Location"))
	}
}

func TestGoogleStartSetsTheStateCookie(t *testing.T) {
	stub := &stubGoogle{enabled: true}
	router := newGoogleRouter(stub, token.NewService([]byte("secret"), time.Hour), googleAppURL)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/start?mode=login&return=/programs", nil))
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "https://accounts.google.test/auth?state=abc" {
		t.Fatalf("expected a redirect to Google, got %d %s", rec.Code, rec.Header().Get("Location"))
	}
	state := cookieNamed(rec, auth.GoogleStateCookieName)
	if state == nil || state.Value != "signed-state" || !state.HttpOnly || !state.Secure || state.Path != googleCookieAt || state.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected state cookie %+v", state)
	}
	if stub.beganMode != google_auth.ModeLogin || stub.beganReturn != "/programs" {
		t.Fatalf("unexpected begin call %+v", stub)
	}
}

func TestGoogleLinkStartNeedsACurrentSession(t *testing.T) {
	stub := &stubGoogle{enabled: true}
	tokens := token.NewService([]byte("secret"), time.Hour)
	router := newGoogleRouter(stub, tokens, googleAppURL)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/start?mode=link", nil))
	if rec.Header().Get("Location") != googleAppURL+"/login?oauth=expired" || stub.beganMode != "" {
		t.Fatalf("expected a redirect to sign in, got %s", rec.Header().Get("Location"))
	}

	// A token from an older session (version 2, current is 3) is not enough.
	stale, _ := tokens.GenerateAccessToken(googleUserID, 2)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/start?mode=link", nil)
	req.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: stale})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Header().Get("Location") != googleAppURL+"/login?oauth=expired" {
		t.Fatalf("expected a stale session to be refused, got %s", rec.Header().Get("Location"))
	}

	current, _ := tokens.GenerateAccessToken(googleUserID, 3)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/start?mode=link", nil)
	req.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: current})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if stub.beganMode != google_auth.ModeLink || stub.beganUser != googleUserID {
		t.Fatalf("expected a link flow for the session account, got %+v", stub)
	}
}

func TestGoogleCallbackIssuesTheSessionAndClearsTheState(t *testing.T) {
	stub := &stubGoogle{enabled: true, outcome: google_auth.Outcome{
		User: &models.User{ID: googleUserID, SessionVersion: 3},
		Path: "/programs",
	}}
	tokens := token.NewService([]byte("secret"), time.Hour)
	router := newGoogleRouter(stub, tokens, googleAppURL)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?state=abc&code=xyz", nil)
	req.AddCookie(&http.Cookie{Name: auth.GoogleStateCookieName, Value: "signed-state"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound || rec.Header().Get("Location") != googleAppURL+"/programs" {
		t.Fatalf("expected a redirect to the app, got %d %s", rec.Code, rec.Header().Get("Location"))
	}
	if stub.gotCookie != "signed-state" || stub.gotCallback.State != "abc" || stub.gotCallback.Code != "xyz" {
		t.Fatalf("unexpected complete call %+v", stub)
	}
	state := cookieNamed(rec, auth.GoogleStateCookieName)
	if state == nil || state.MaxAge >= 0 || state.Value != "" || state.Path != googleCookieAt {
		t.Fatalf("expected the state cookie to be cleared, got %+v", state)
	}
	session := cookieNamed(rec, auth.AccessTokenCookieName)
	if session == nil || !session.HttpOnly || !session.Secure {
		t.Fatalf("expected a session cookie, got %+v", session)
	}
	claims, err := tokens.ValidateAccessToken(session.Value)
	if err != nil || claims.UserID != googleUserID || claims.SessionVersion != 3 {
		t.Fatalf("unexpected session token claims %+v (%v)", claims, err)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected Cache-Control: no-store")
	}
}

func TestGoogleCallbackWithoutSessionDoesNotSignIn(t *testing.T) {
	stub := &stubGoogle{enabled: true, outcome: google_auth.Outcome{Path: "/login?oauth=link_required"}}
	router := newGoogleRouter(stub, token.NewService([]byte("secret"), time.Hour), googleAppURL)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?state=abc&code=xyz", nil))
	if rec.Header().Get("Location") != googleAppURL+"/login?oauth=link_required" {
		t.Fatalf("unexpected redirect %s", rec.Header().Get("Location"))
	}
	if cookieNamed(rec, auth.AccessTokenCookieName) != nil {
		t.Fatal("a session cookie was issued without a signed-in outcome")
	}
}
