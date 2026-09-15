package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/middleware"
	"ryze/backend/services/token"
)

const adminMeRoute = "/api/v1/admin/auth/me"

// newAdminMeTestRouter wires the AdminAuthenticate middleware in front of the
// AdminMeHandler so the endpoint is exercised exactly like production.
func newAdminMeTestRouter(t *testing.T) (*gin.Engine, token.Service) {
	t.Helper()

	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)
	handler := auth.NewAdminMeHandler()

	router := gin.New()
	router.GET(adminMeRoute, middleware.AdminAuthenticate(tokenSvc), handler.GetMe)

	return router, tokenSvc
}

// performAdminMe calls the admin identity endpoint carrying the given admin
// access token (empty means no cookie).
func performAdminMe(router http.Handler, adminToken string) (*httptest.ResponseRecorder, string) {
	req := httptest.NewRequest(http.MethodGet, adminMeRoute, nil)
	if adminToken != "" {
		req.AddCookie(&http.Cookie{Name: auth.AdminAccessTokenCookieName, Value: adminToken})
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec, rec.Body.String()
}

func TestAdminMeWithoutSessionRejected(t *testing.T) {
	router, _ := newAdminMeTestRouter(t)

	for _, adminToken := range []string{"", "not.a.jwt", "garbage"} {
		rec, raw := performAdminMe(router, adminToken)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("token %q: expected 401, got %d (body: %s)", adminToken, rec.Code, raw)
		}
		if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
			t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
		}
	}
}

func TestAdminMeIdentityAndRole(t *testing.T) {
	for _, tc := range []struct {
		adminID string
		role    string
	}{
		{adminID: "ADMIN_1", role: "TECHNICAL_ADMINISTRATOR"},
		{adminID: "ADMIN_2", role: "MANAGEMENT_ADMINISTRATOR"},
	} {
		t.Run(tc.adminID, func(t *testing.T) {
			router, tokenSvc := newAdminMeTestRouter(t)

			adminToken, err := tokenSvc.GenerateAdminToken(tc.adminID)
			if err != nil {
				t.Fatalf("GenerateAdminToken: %v", err)
			}

			rec, raw := performAdminMe(router, adminToken)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
			}

			var payload struct {
				Success bool `json:"success"`
				Data    struct {
					ID   string `json:"id"`
					Role string `json:"role"`
				} `json:"data"`
			}
			if err := json.Unmarshal([]byte(raw), &payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if !payload.Success {
				t.Fatalf("expected success response, got %s", raw)
			}
			if payload.Data.ID != tc.adminID {
				t.Fatalf("expected id %q, got %q", tc.adminID, payload.Data.ID)
			}
			if payload.Data.Role != tc.role {
				t.Fatalf("expected role %q, got %q", tc.role, payload.Data.Role)
			}
		})
	}
}

func TestAdminMeRejectsUserToken(t *testing.T) {
	router, tokenSvc := newAdminMeTestRouter(t)

	userToken, err := tokenSvc.GenerateAccessToken("11111111-1111-4111-8111-111111111111", 0)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, raw := performAdminMe(router, userToken)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for a user token, got %d (body: %s)", rec.Code, raw)
	}
}

func TestAdminLogoutClearsSessionCookies(t *testing.T) {
	handler := auth.NewAdminLogoutHandler(true)

	router := gin.New()
	router.POST("/api/v1/admin/auth/logout", handler.Logout)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/logout", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	for _, name := range []string{auth.AdminAccessTokenCookieName, auth.AdminStageTokenCookieName} {
		attrs := cookieAttrs(rec, name)
		if attrs == nil {
			t.Fatalf("logout must clear cookie %s", name)
		}
		if attrs["max-age"] != "0" && attrs["value"] != "" {
			t.Fatalf("cookie %s must be cleared, got attrs %v", name, attrs)
		}
	}

	if cookieAttrs(rec, auth.AccessTokenCookieName) != nil {
		t.Fatal("admin logout must never touch the client access-token cookie")
	}

	if !strings.Contains(rec.Body.String(), `"message":"Logout successful."`) {
		t.Fatalf("expected logout confirmation, got %s", rec.Body.String())
	}
}

func TestAdminLogoutClearsSessionCookiesInsecureLocalHTTP(t *testing.T) {
	handler := auth.NewAdminLogoutHandler(false)

	router := gin.New()
	router.POST("/api/v1/admin/auth/logout", handler.Logout)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/logout", nil))

	attrs := cookieAttrs(rec, auth.AdminAccessTokenCookieName)
	if attrs == nil {
		t.Fatal("logout must clear the admin access-token cookie")
	}
	if _, present := attrs["secure"]; present {
		t.Fatal("Secure must be disabled for local HTTP development")
	}
}

func TestAdminLogoutResponseNeverExposesSecrets(t *testing.T) {
	handler := auth.NewAdminLogoutHandler(true)

	router := gin.New()
	router.POST("/api/v1/admin/auth/logout", handler.Logout)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/logout", nil))

	if strings.Contains(rec.Body.String(), testSecret) {
		t.Fatal("logout response must never expose the JWT secret")
	}
	if strings.Contains(rec.Body.String(), "ryzeADMIN1") || strings.Contains(rec.Body.String(), "edgar_manager123#") {
		t.Fatal("logout response must never expose admin credentials")
	}
}