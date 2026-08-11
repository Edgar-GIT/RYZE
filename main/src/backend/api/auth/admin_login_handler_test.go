package auth_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"ryze/backend/api/auth"
	"ryze/backend/services/admin_login"
	"ryze/backend/services/token"
)

const (
	adminLoginRoute     = "/api/v1/admin/auth/login"
	adminVerifyRoute    = "/api/v1/admin/auth/verify"
	testAdminCookieName = "ryze_admin_access_token"
	testStageCookieName = "ryze_admin_stage_token"
)

var testAdminCredentials = []admin_login.AdminCredential{
	{ID: "ADMIN_1", Username: "ryzeADMIN1", Password: "edgar_manager123#", AccessCode: "mz7Qx2!ryze2fa_admin1"},
	{ID: "ADMIN_2", Username: "ryzeADMIN2", Password: "sandro_manager123#", AccessCode: "np9Wv4#ryze2fa_admin2"},
}

// newAdminLoginTestRouter builds a router with the admin login flow wired to
// the configured test administrators.
func newAdminLoginTestRouter(t *testing.T, secure bool) (*gin.Engine, token.Service) {
	t.Helper()

	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)
	loginSvc := admin_login.NewService(testAdminCredentials)
	handler := auth.NewAdminLoginHandler(loginSvc, tokenSvc, testTokenTTL, secure)

	router := gin.New()
	router.POST(adminLoginRoute, handler.Login)
	router.POST(adminVerifyRoute, handler.Verify)

	return router, tokenSvc
}

func adminLoginBody(username, password string) string {
	return fmt.Sprintf(`{"username": %q, "password": %q}`, username, password)
}

func adminVerifyBody(accessCode string) string {
	return fmt.Sprintf(`{"access_code": %q}`, accessCode)
}

// performAdminStage1 submits the username/password stage and returns the stage
// cookie value (if issued) and the raw response body.
func performAdminStage1(router http.Handler, username, password string) (*httptest.ResponseRecorder, string, string) {
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(
		http.MethodPost,
		adminLoginRoute,
		bytes.NewBufferString(adminLoginBody(username, password)),
	))
	return rec, cookieValue(rec, testStageCookieName), rec.Body.String()
}

// attemptAdminVerify submits the access code stage carrying the given stage
// cookie value (empty means no stage state).
func attemptAdminVerify(router http.Handler, stageToken, body string) (*httptest.ResponseRecorder, string) {
	req := httptest.NewRequest(http.MethodPost, adminVerifyRoute, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if stageToken != "" {
		req.AddCookie(&http.Cookie{Name: auth.AdminStageTokenCookieName, Value: stageToken})
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec, rec.Body.String()
}

// cookieValue returns the value of the first Set-Cookie with the given name.
func cookieValue(rec *httptest.ResponseRecorder, name string) string {
	attrs := cookieAttrs(rec, name)
	if attrs == nil {
		return ""
	}
	return attrs["value"]
}

// cookieAttrs parses the Set-Cookie headers for the given cookie name.
func cookieAttrs(rec *httptest.ResponseRecorder, name string) map[string]string {
	for _, header := range rec.Result().Header.Values("Set-Cookie") {
		parts := strings.Split(header, ";")
		first := strings.SplitN(strings.TrimSpace(parts[0]), "=", 2)
		if len(first) != 2 || first[0] != name {
			continue
		}
		attrs := map[string]string{"value": first[1]}
		for _, part := range parts[1:] {
			kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
			value := ""
			if len(kv) == 2 {
				value = kv[1]
			}
			attrs[strings.ToLower(kv[0])] = value
		}
		return attrs
	}
	return nil
}

// expiredStageToken forges a signed stage token whose expiration has already
// passed so the expiry behaviour can be exercised.
func expiredStageToken(t *testing.T) string {
	t.Helper()

	claims := struct {
		Kind string `json:"kind"`
		jwt.RegisteredClaims
	}{
		Kind: "admin-stage",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "ADMIN_1",
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-6 * time.Minute)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
		},
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign expired stage token: %v", err)
	}
	return raw
}

func TestAdminStage1Success(t *testing.T) {
	for _, tc := range []struct {
		username string
		password string
	}{
		{username: "ryzeADMIN1", password: "edgar_manager123#"},
		{username: "ryzeADMIN2", password: "sandro_manager123#"},
	} {
		t.Run(tc.username, func(t *testing.T) {
			router, _ := newAdminLoginTestRouter(t, true)

			rec, stageToken, raw := performAdminStage1(router, tc.username, tc.password)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
			}
			if stageToken == "" {
				t.Fatal("stage 1 must issue the temporary stage cookie")
			}
			if strings.Contains(raw, "token") {
				t.Fatal("no token may ever appear in the JSON response")
			}
		})
	}
}

func TestAdminStage1DoesNotIssueFinalCookie(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	rec, _, raw := performAdminStage1(router, "ryzeADMIN1", "edgar_manager123#")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	if cookieAttrs(rec, testAdminCookieName) != nil {
		t.Fatal("stage 1 must NOT issue the final admin access-token cookie")
	}
}

func TestAdminStage1WrongPassword(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	rec, stageToken, raw := performAdminStage1(router, "ryzeADMIN1", "WrongPassword123!")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if stageToken != "" {
		t.Fatal("no stage cookie may be issued for invalid credentials")
	}
	if !strings.Contains(raw, `"code":"INVALID_CREDENTIALS"`) {
		t.Fatalf("expected INVALID_CREDENTIALS, got %s", raw)
	}
}

func TestAdminStage1UnknownUsername(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	rec, stageToken, raw := performAdminStage1(router, "nobody", "edgar_manager123#")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if stageToken != "" {
		t.Fatal("no stage cookie may be issued for an unknown username")
	}
}

func TestAdminStage1InvalidJSON(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(
		http.MethodPost,
		adminLoginRoute,
		bytes.NewBufferString(`{"username": "broken"`),
	))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestAdminStage1MissingFields(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	for name, body := range map[string]string{
		"empty object":     `{}`,
		"missing username": `{"password": "edgar_manager123#"}`,
		"missing password": `{"username": "ryzeADMIN1"}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(
				http.MethodPost,
				adminLoginRoute,
				bytes.NewBufferString(body),
			))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d (body: %s)", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAdminVerifyWithoutStageStateRejected(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	for _, stageToken := range []string{"", "not.a.jwt", "garbage"} {
		rec, raw := attemptAdminVerify(router, stageToken, adminVerifyBody("mz7Qx2!ryze2fa_admin1"))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("stage token %q: expected 401, got %d (body: %s)", stageToken, rec.Code, raw)
		}
		if cookieAttrs(rec, testAdminCookieName) != nil {
			t.Fatal("verify without valid stage state must never issue the final admin cookie")
		}
	}
}

func TestAdminVerifySuccess(t *testing.T) {
	for _, tc := range []struct {
		username   string
		password   string
		accessCode string
		identity   string
	}{
		{username: "ryzeADMIN1", password: "edgar_manager123#", accessCode: "mz7Qx2!ryze2fa_admin1", identity: "ADMIN_1"},
		{username: "ryzeADMIN2", password: "sandro_manager123#", accessCode: "np9Wv4#ryze2fa_admin2", identity: "ADMIN_2"},
	} {
		t.Run(tc.identity, func(t *testing.T) {
			router, tokenSvc := newAdminLoginTestRouter(t, true)

			_, stageToken, raw := performAdminStage1(router, tc.username, tc.password)
			if stageToken == "" {
				t.Fatalf("expected a stage cookie (body: %s)", raw)
			}

			rec, raw := attemptAdminVerify(router, stageToken, adminVerifyBody(tc.accessCode))
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
			}

			final := cookieAttrs(rec, testAdminCookieName)
			if final == nil {
				t.Fatal("verify must issue the final admin access-token cookie")
			}

			adminID, err := tokenSvc.ValidateAdminToken(final["value"])
			if err != nil {
				t.Fatalf("ValidateAdminToken: %v", err)
			}
			if adminID != tc.identity {
				t.Fatalf("expected token identity %q, got %q", tc.identity, adminID)
			}
		})
	}
}

func TestAdminVerifyClearsStageCookie(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	_, stageToken, _ := performAdminStage1(router, "ryzeADMIN1", "edgar_manager123#")
	rec, raw := attemptAdminVerify(router, stageToken, adminVerifyBody("mz7Qx2!ryze2fa_admin1"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	stage := cookieAttrs(rec, testStageCookieName)
	if stage == nil {
		t.Fatal("verify must clear the stage cookie")
	}
	if stage["max-age"] != "0" && stage["value"] != "" {
		t.Fatalf("stage cookie must be cleared, got attrs %v", stage)
	}
}

func TestAdminVerifyWrongAccessCode(t *testing.T) {
	for _, tc := range []struct {
		username string
		password string
		wrong    string
	}{
		{username: "ryzeADMIN1", password: "edgar_manager123#", wrong: "wrong_code_admin_1!"},
		{username: "ryzeADMIN2", password: "sandro_manager123#", wrong: "wrong_code_admin_2!"},
	} {
		t.Run(tc.username, func(t *testing.T) {
			router, _ := newAdminLoginTestRouter(t, true)

			_, stageToken, raw := performAdminStage1(router, tc.username, tc.password)
			if stageToken == "" {
				t.Fatalf("expected a stage cookie (body: %s)", raw)
			}

			rec, raw := attemptAdminVerify(router, stageToken, adminVerifyBody(tc.wrong))
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"INVALID_CREDENTIALS"`) {
				t.Fatalf("expected INVALID_CREDENTIALS, got %s", raw)
			}
			if cookieAttrs(rec, testAdminCookieName) != nil {
				t.Fatal("wrong access code must never issue the final admin cookie")
			}
		})
	}
}

func TestAdminVerifyCrossAdminAccessCodes(t *testing.T) {
	for _, tc := range []struct {
		username string
		password string
		foreign  string
	}{
		{username: "ryzeADMIN1", password: "edgar_manager123#", foreign: "np9Wv4#ryze2fa_admin2"},
		{username: "ryzeADMIN2", password: "sandro_manager123#", foreign: "mz7Qx2!ryze2fa_admin1"},
	} {
		t.Run(tc.username, func(t *testing.T) {
			router, _ := newAdminLoginTestRouter(t, true)

			_, stageToken, raw := performAdminStage1(router, tc.username, tc.password)
			if stageToken == "" {
				t.Fatalf("expected a stage cookie (body: %s)", raw)
			}

			rec, raw := attemptAdminVerify(router, stageToken, adminVerifyBody(tc.foreign))
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
			}
			if cookieAttrs(rec, testAdminCookieName) != nil {
				t.Fatal("a foreign admin access code must never issue the final admin cookie")
			}
		})
	}
}

func TestAdminVerifyExpiredStageStateRejected(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	rec, raw := attemptAdminVerify(router, expiredStageToken(t), adminVerifyBody("mz7Qx2!ryze2fa_admin1"))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for an expired stage state, got %d (body: %s)", rec.Code, raw)
	}
	if cookieAttrs(rec, testAdminCookieName) != nil {
		t.Fatal("expired stage state must never issue the final admin cookie")
	}
}

func TestAdminVerifyMissingFields(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	_, stageToken, _ := performAdminStage1(router, "ryzeADMIN1", "edgar_manager123#")

	for name, body := range map[string]string{
		"empty object": `{}`,
		"missing code": `{"access_code": ""}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec, raw := attemptAdminVerify(router, stageToken, body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
			}
			if cookieAttrs(rec, testAdminCookieName) != nil {
				t.Fatal("invalid verify input must never issue the final admin cookie")
			}
		})
	}
}

func TestAdminVerifyInvalidJSON(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	_, stageToken, _ := performAdminStage1(router, "ryzeADMIN1", "edgar_manager123#")
	rec, raw := attemptAdminVerify(router, stageToken, `{"access_code": "broken"`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
}

func TestAdminCookieAttributes(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	_, stageToken, _ := performAdminStage1(router, "ryzeADMIN1", "edgar_manager123#")
	rec, raw := attemptAdminVerify(router, stageToken, adminVerifyBody("mz7Qx2!ryze2fa_admin1"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	final := cookieAttrs(rec, testAdminCookieName)
	if final == nil {
		t.Fatal("final admin cookie must be set")
	}
	if _, present := final["httponly"]; !present {
		t.Fatal("final admin cookie must be HttpOnly")
	}
	if _, present := final["secure"]; !present {
		t.Fatal("final admin cookie must be Secure by default")
	}
	if sameSite, _ := final["samesite"]; sameSite != "Lax" {
		t.Fatalf("expected SameSite=Lax, got %q", sameSite)
	}
	if path, _ := final["path"]; path != "/" {
		t.Fatalf("expected Path=/, got %q", path)
	}
	if maxAge, _ := final["max-age"]; maxAge != "900" {
		t.Fatalf("expected Max-Age=900 matching the TTL, got %q", maxAge)
	}
}

func TestAdminSecureCookieDisabledForLocalHTTP(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, false)

	_, stageToken, _ := performAdminStage1(router, "ryzeADMIN1", "edgar_manager123#")
	rec, raw := attemptAdminVerify(router, stageToken, adminVerifyBody("mz7Qx2!ryze2fa_admin1"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	final := cookieAttrs(rec, testAdminCookieName)
	if _, present := final["secure"]; present {
		t.Fatal("Secure must be disabled for local HTTP development")
	}
}

func TestAdminStageCookieAttributes(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	rec, _, _ := performAdminStage1(router, "ryzeADMIN1", "edgar_manager123#")

	stage := cookieAttrs(rec, testStageCookieName)
	if stage == nil {
		t.Fatal("stage cookie must be set")
	}
	if _, present := stage["httponly"]; !present {
		t.Fatal("stage cookie must be HttpOnly")
	}
	if sameSite, _ := stage["samesite"]; sameSite != "Lax" {
		t.Fatalf("expected SameSite=Lax, got %q", sameSite)
	}
	if maxAge, _ := stage["max-age"]; maxAge != "300" {
		t.Fatalf("expected Max-Age=300 matching the short stage lifetime, got %q", maxAge)
	}
}

func TestAdminLoginUsesDedicatedCookieName(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	_, stageToken, _ := performAdminStage1(router, "ryzeADMIN1", "edgar_manager123#")
	rec, raw := attemptAdminVerify(router, stageToken, adminVerifyBody("mz7Qx2!ryze2fa_admin1"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	if cookieAttrs(rec, testCookieName) != nil {
		t.Fatal("admin login must never write the client access-token cookie")
	}
}

func TestFinalAdminTokenRejectedAsUserToken(t *testing.T) {
	router, tokenSvc := newAdminLoginTestRouter(t, true)

	_, stageToken, _ := performAdminStage1(router, "ryzeADMIN1", "edgar_manager123#")
	rec, raw := attemptAdminVerify(router, stageToken, adminVerifyBody("mz7Qx2!ryze2fa_admin1"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	final := cookieAttrs(rec, testAdminCookieName)
	if final == nil {
		t.Fatal("final admin cookie must be set")
	}
	if _, err := tokenSvc.ValidateAccessToken(final["value"]); err == nil {
		t.Fatal("the final admin token must never validate as a user access token")
	}
}

func TestUserTokenRejectedAsAdminToken(t *testing.T) {
	router, tokenSvc := newAdminLoginTestRouter(t, true)

	userToken, err := tokenSvc.GenerateAccessToken("11111111-1111-4111-8111-111111111111", 0)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, adminVerifyRoute, bytes.NewBufferString(adminVerifyBody("mz7Qx2!ryze2fa_admin1")))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: auth.AdminStageTokenCookieName, Value: userToken})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when a user token is used as stage state, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestAdminLoginResponseNeverExposesSecrets(t *testing.T) {
	router, _ := newAdminLoginTestRouter(t, true)

	for _, secrets := range [][]string{
		{"edgar_manager123#", "sandro_manager123#"},
		{"mz7Qx2!ryze2fa_admin1", "np9Wv4#ryze2fa_admin2"},
		{"ryzeADMIN1", "ryzeADMIN2"},
		{testSecret},
	} {
		_, stageToken, raw := performAdminStage1(router, "ryzeADMIN1", "edgar_manager123#")
		if stageToken == "" {
			t.Fatal("expected a stage cookie")
		}
		for _, secret := range secrets {
			if strings.Contains(raw, secret) {
				t.Fatalf("stage 1 response must never expose %q", secret)
			}
		}

		rec, raw := attemptAdminVerify(router, stageToken, adminVerifyBody("mz7Qx2!ryze2fa_admin1"))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
		}
		for _, secret := range secrets {
			if strings.Contains(raw, secret) {
				t.Fatalf("verify response must never expose %q", secret)
			}
		}

		final := cookieAttrs(rec, testAdminCookieName)
		if final != nil && strings.Contains(raw, final["value"]) {
			t.Fatal("JWT must exist only in the Set-Cookie header, never in the JSON response")
		}
	}
}
