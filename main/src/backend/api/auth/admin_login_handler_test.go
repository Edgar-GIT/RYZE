package auth_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/services/admin_login"
	"ryze/backend/services/token"
)

const (
	adminLoginRoute     = "/api/v1/admin/auth/login"
	testAdminCookieName = "ryze_admin_access_token"
)

var testAdminCredentials = []admin_login.AdminCredential{
	{ID: "ADMIN_1", Username: "ryzeADMIN1", Password: "edgar_manager123#"},
	{ID: "ADMIN_2", Username: "ryzeADMIN2", Password: "sandro_manager123#"},
}

// newAdminLoginTestRouter builds a router with the admin login handler wired
// to the configured test administrators.
func newAdminLoginTestRouter(t *testing.T, secure bool) (*gin.Engine, token.Service) {
	t.Helper()

	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)
	loginSvc := admin_login.NewService(testAdminCredentials)
	handler := auth.NewAdminLoginHandler(loginSvc, tokenSvc, testTokenTTL, secure)

	router := gin.New()
	router.POST(adminLoginRoute, handler.Login)

	return router, tokenSvc
}

func attemptAdminLogin(router http.Handler, body string) (*httptest.ResponseRecorder, string) {
	req := httptest.NewRequest(http.MethodPost, adminLoginRoute, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec, rec.Body.String()
}

func adminLoginBody(username, password string) string {
	return fmt.Sprintf(`{"username": %q, "password": %q}`, username, password)
}

func TestAdminLoginSuccess(t *testing.T) {
	for _, tc := range []struct {
		username string
		password string
	}{
		{username: "ryzeADMIN1", password: "edgar_manager123#"},
		{username: "ryzeADMIN2", password: "sandro_manager123#"},
	} {
		t.Run(tc.username, func(t *testing.T) {
			router, _ := newAdminLoginTestRouter(t, true)

			rec, raw := attemptAdminLogin(router, adminLoginBody(tc.username, tc.password))
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
			}

			var payload map[string]any
			if err := json.Unmarshal([]byte(raw), &payload); err != nil {
				t.Fatalf("invalid JSON response: %v", err)
			}
			if strings.Contains(raw, "token") {
				t.Fatal("JWT must never appear in the JSON response")
			}
		})
	}
}

func TestAdminLoginSetsHttpOnlyCookie(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	rec, raw := attemptAdminLogin(router, adminLoginBody("ryzeADMIN1", "edgar_manager123#"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	cookie := parseSetCookie(t, rec)

	tokenValue, ok := cookie[testAdminCookieName]
	if !ok {
		t.Fatal("cookie ryze_admin_access_token must be set")
	}
	if tokenValue == "" {
		t.Fatal("cookie must contain the generated JWT")
	}
	if _, present := cookie["httponly"]; !present {
		t.Fatal("cookie must be HttpOnly")
	}
	if _, present := cookie["secure"]; !present {
		t.Fatal("cookie must be Secure by default")
	}
	if sameSite, _ := cookie["samesite"]; sameSite != "Lax" {
		t.Fatalf("expected SameSite=Lax, got %q", sameSite)
	}
	if path, _ := cookie["path"]; path != "/" {
		t.Fatalf("expected Path=/, got %q", path)
	}
	if maxAge, _ := cookie["max-age"]; maxAge != "900" {
		t.Fatalf("expected Max-Age=900 matching the TTL, got %q", maxAge)
	}
}

func TestAdminLoginUsesDedicatedCookieName(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	rec, _ := attemptAdminLogin(router, adminLoginBody("ryzeADMIN1", "edgar_manager123#"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	cookie := parseSetCookie(t, rec)
	if _, ok := cookie[testCookieName]; ok {
		t.Fatal("admin login must never write the client access-token cookie")
	}
}

func TestAdminLoginSecureCookieDisabledForLocalHTTP(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, false)

	rec, raw := attemptAdminLogin(router, adminLoginBody("ryzeADMIN1", "edgar_manager123#"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	cookie := parseSetCookie(t, rec)
	if _, present := cookie["secure"]; present {
		t.Fatal("Secure must be disabled for local HTTP development")
	}
}

func TestAdminLoginTokenValidatesWithIdentity(t *testing.T) {
	router, tokenSvc := newAdminLoginTestRouter(t, true)

	rec, raw := attemptAdminLogin(router, adminLoginBody("ryzeADMIN2", "sandro_manager123#"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	cookie := parseSetCookie(t, rec)
	jwtValue, ok := cookie[testAdminCookieName]
	if !ok {
		t.Fatal("cookie ryze_admin_access_token must be set")
	}

	adminID, err := tokenSvc.ValidateAdminToken(jwtValue)
	if err != nil {
		t.Fatalf("ValidateAdminToken: %v", err)
	}
	if adminID != "ADMIN_2" {
		t.Fatalf("expected token identity ADMIN_2, got %q", adminID)
	}
}

func TestAdminLoginInvalidCredentials(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	for name, body := range map[string]string{
		"wrong password": adminLoginBody("ryzeADMIN1", "WrongPassword123!"),
		"unknown user":   adminLoginBody("nobody", "edgar_manager123#"),
	} {
		t.Run(name, func(t *testing.T) {
			rec, raw := attemptAdminLogin(router, body)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"INVALID_CREDENTIALS"`) {
				t.Fatalf("expected INVALID_CREDENTIALS, got %s", raw)
			}
		})
	}
}

func TestAdminLoginInvalidJSON(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	rec, raw := attemptAdminLogin(router, `{"username": "broken"`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
}

func TestAdminLoginMissingFields(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	for name, body := range map[string]string{
		"empty object":     `{}`,
		"missing username": `{"password": "edgar_manager123#"}`,
		"missing password": `{"username": "ryzeADMIN1"}`,
		"empty username":   `{"username": "", "password": "edgar_manager123#"}`,
		"empty password":   `{"username": "ryzeADMIN1", "password": ""}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec, raw := attemptAdminLogin(router, body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
			}
			if strings.Contains(raw, `"code":"INVALID_CREDENTIALS"`) {
				t.Fatalf("validation failures must not be reported as invalid credentials: %s", raw)
			}
		})
	}
}

func TestAdminLoginResponseNeverExposesSecrets(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	rec, raw := attemptAdminLogin(router, adminLoginBody("ryzeADMIN1", "edgar_manager123#"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	if strings.Contains(raw, "edgar_manager123#") || strings.Contains(raw, "sandro_manager123#") {
		t.Fatal("response must never contain configured admin passwords")
	}
	if strings.Contains(raw, "ryzeADMIN") {
		t.Fatal("response must never contain configured admin usernames")
	}
	if strings.Contains(raw, testSecret) {
		t.Fatal("response must never expose JWT_SECRET")
	}

	cookie := parseSetCookie(t, rec)
	jwtValue := cookie[testAdminCookieName]
	if jwtValue != "" && strings.Contains(raw, jwtValue) {
		t.Fatal("JWT must exist only in the Set-Cookie header, never in the JSON response")
	}
}
