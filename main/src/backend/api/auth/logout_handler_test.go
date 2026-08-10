package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/services/token"
)

const logoutRoute = "/api/v1/auth/logout"

func newLogoutRouter(t *testing.T, secure bool) *gin.Engine {
	t.Helper()

	handler := auth.NewLogoutHandler(secure)
	router := gin.New()
	router.POST(logoutRoute, handler.Logout)
	return router
}

func performLogout(router http.Handler, cookieValue string) (*httptest.ResponseRecorder, map[string]any, string) {
	req := httptest.NewRequest(http.MethodPost, logoutRoute, nil)
	if cookieValue != "" {
		req.AddCookie(&http.Cookie{Name: "ryze_access_token", Value: cookieValue})
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		return rec, nil, ""
	}
	data, _ := payload["data"].(map[string]any)
	return rec, data, string(rec.Body.Bytes())
}

func TestLogoutSuccess(t *testing.T) {
	router := newLogoutRouter(t, true)

	rec, data, raw := performLogout(router, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if data == nil {
		t.Fatal("expected data object in response")
	}
}

func TestLogoutClearsCookie(t *testing.T) {
	router := newLogoutRouter(t, true)

	rec, _, _ := performLogout(router, "some.invalid.token")
	cookie := parseSetCookie(t, rec)

	tokenValue, ok := cookie[testCookieName]
	if !ok {
		t.Fatal("cookie ryze_access_token must be present in the clearing Set-Cookie")
	}
	if tokenValue != "" {
		t.Fatalf("cookie value must be empty, got %q", tokenValue)
	}
	if _, present := cookie["httponly"]; !present {
		t.Fatal("cookie must be HttpOnly")
	}
	if path, _ := cookie["path"]; path != "/" {
		t.Fatalf("expected Path=/, got %q", path)
	}
	if sameSite, _ := cookie["samesite"]; sameSite != "Lax" {
		t.Fatalf("expected SameSite=Lax, got %q", sameSite)
	}
	if maxAge, _ := cookie["max-age"]; maxAge != "0" {
		t.Fatalf("expected Max-Age=0, got %q", maxAge)
	}
}

func TestLogoutSecureCookieDefault(t *testing.T) {
	router := newLogoutRouter(t, true)

	rec, _, _ := performLogout(router, "")
	if _, present := parseSetCookie(t, rec)["secure"]; !present {
		t.Fatal("cookie must be Secure by default")
	}
}

func TestLogoutSecureCookieDisabledForLocalHTTP(t *testing.T) {
	router := newLogoutRouter(t, false)

	rec, _, _ := performLogout(router, "")
	if _, present := parseSetCookie(t, rec)["secure"]; present {
		t.Fatal("Secure must be disabled for local HTTP development")
	}
}

func TestLogoutWithoutCookie(t *testing.T) {
	router := newLogoutRouter(t, true)

	rec, _, raw := performLogout(router, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("logout must succeed without an existing cookie, got %d (body: %s)", rec.Code, raw)
	}
}

func TestLogoutWithInvalidCookie(t *testing.T) {
	router := newLogoutRouter(t, true)

	for _, value := range []string{"garbage", "not.a.jwt", "eyJhbGciOiJIUzI1NiJ9.expired.invalid"} {
		rec, _, raw := performLogout(router, value)
		if rec.Code != http.StatusOK {
			t.Fatalf("logout must succeed with an invalid cookie %q, got %d (body: %s)", value, rec.Code, raw)
		}
	}
}

func TestLogoutResponseNeverExposesSecrets(t *testing.T) {
	router := newLogoutRouter(t, true)

	svc := token.NewService([]byte(testSecret), testTokenTTL)
	jwtValue, err := svc.GenerateAccessToken("00000000-0000-0000-0000-000000000000", 0)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, data, raw := performLogout(router, jwtValue)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if strings.Contains(raw, jwtValue) {
		t.Fatal("response must never contain the JWT")
	}
	if strings.Contains(raw, testSecret) {
		t.Fatal("response must never expose JWT_SECRET")
	}
	if _, ok := data["password_hash"]; ok {
		t.Fatal("response must not expose password_hash")
	}
}
