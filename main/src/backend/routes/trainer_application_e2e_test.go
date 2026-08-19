package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ryze/backend/api/auth"
	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/repositories"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// e2eRequest performs an HTTP request against the router, optionally carrying a
// cookie.
func e2eRequest(router http.Handler, cookieName, cookieValue, method, path, body string) (*httptest.ResponseRecorder, string) {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookieValue != "" {
		req.AddCookie(&http.Cookie{Name: cookieName, Value: cookieValue})
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec, rec.Body.String()
}

// e2eCookieValue extracts the value of the first Set-Cookie with the given
// name.
func e2eCookieValue(rec *httptest.ResponseRecorder, name string) string {
	for _, header := range rec.Result().Header.Values("Set-Cookie") {
		parts := strings.SplitN(header, ";", 2)
		kv := strings.SplitN(strings.TrimSpace(parts[0]), "=", 2)
		if len(kv) == 2 && kv[0] == name {
			return kv[1]
		}
	}
	return ""
}

func e2eUniqueEmail() string {
	return fmt.Sprintf("e2e-%d@ryze.local", time.Now().UnixNano())
}

// e2eRegisterUser registers and logs in a fresh user, returning the login
// cookie.
func e2eRegisterUser(t *testing.T, router http.Handler, email string) string {
	t.Helper()

	rec, raw := e2eRequest(router, "", "", http.MethodPost, "/api/v1/auth/register", fmt.Sprintf(
		`{"email": %q, "password": "Password123!", "first_name": "E2E", "last_name": "Trainer"}`, email))
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d (body: %s)", rec.Code, raw)
	}

	rec, raw = e2eRequest(router, "", "", http.MethodPost, "/api/v1/auth/login", fmt.Sprintf(
		`{"email": %q, "password": "Password123!"}`, email))
	if rec.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	cookie := e2eCookieValue(rec, auth.AccessTokenCookieName)
	if cookie == "" {
		t.Fatalf("login must issue a user cookie (body: %s)", raw)
	}
	return cookie
}

// e2eAdminLogin completes the two-stage admin authentication and returns the
// final admin cookie.
func e2eAdminLogin(t *testing.T, router http.Handler, admin config.Admin) string {
	t.Helper()

	rec, raw := e2eRequest(router, "", "", http.MethodPost, "/api/v1/admin/auth/login", fmt.Sprintf(
		`{"username": %q, "password": %q}`, admin.Username, admin.Password))
	if rec.Code != http.StatusOK {
		t.Fatalf("admin stage 1: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	stageCookie := e2eCookieValue(rec, auth.AdminStageTokenCookieName)
	if stageCookie == "" {
		t.Fatalf("admin stage 1 must issue a stage cookie (body: %s)", raw)
	}

	rec, raw = e2eRequest(router, auth.AdminStageTokenCookieName, stageCookie, http.MethodPost, "/api/v1/admin/auth/verify", fmt.Sprintf(
		`{"access_code": %q}`, admin.AccessCode))
	if rec.Code != http.StatusOK {
		t.Fatalf("admin verify: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	cookie := e2eCookieValue(rec, auth.AdminAccessTokenCookieName)
	if cookie == "" {
		t.Fatalf("admin verify must issue the admin cookie (body: %s)", raw)
	}
	return cookie
}

func adminByID(t *testing.T, cfg config.AdminConfig, id string) config.Admin {
	t.Helper()
	for _, admin := range cfg.Admins {
		if admin.ID == id {
			return admin
		}
	}
	t.Fatalf("admin %s is not configured", id)
	return config.Admin{}
}

type e2eApplication struct {
	ID     string
	Status string
	UserID string
}

func e2eDecodeApplication(t *testing.T, raw string) e2eApplication {
	t.Helper()
	var payload struct {
		Data struct {
			ID     string `json:"id"`
			Status string `json:"status"`
			UserID string `json:"user_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("invalid JSON: %v (body: %s)", err, raw)
	}
	if payload.Data.ID == "" || payload.Data.Status == "" || payload.Data.UserID == "" {
		t.Fatalf("unexpected application payload (body: %s)", raw)
	}
	return e2eApplication{ID: payload.Data.ID, Status: payload.Data.Status, UserID: payload.Data.UserID}
}

// newE2ERouter builds the complete application router through routes.Setup,
// backed by a real database transaction so every record created by the test is
// rolled back. It returns the router, the transaction and the configured
// administrators.
func newE2ERouter(t *testing.T) (*gin.Engine, *gorm.DB, config.AdminConfig) {
	t.Helper()

	config.LoadEnvFile()
	dbCfg, err := config.Load()
	if err != nil {
		t.Fatalf("load database config: %v", err)
	}
	jwtCfg, err := config.LoadJWT()
	if err != nil {
		t.Fatalf("load jwt config: %v", err)
	}
	corsCfg, err := config.LoadCORS()
	if err != nil {
		t.Fatalf("load cors config: %v", err)
	}
	adminCfg, err := config.LoadAdmin()
	if err != nil {
		t.Fatalf("load admin config: %v", err)
	}

	db, err := database.Connect(dbCfg)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	tx := db.Begin()
	t.Cleanup(func() { tx.Rollback() })

	return Setup(tx, jwtCfg, corsCfg, adminCfg, config.PricingConfig{MinProgramPriceMinorUnits: 100}, config.CommissionConfig{DefaultPlatformCommissionBPS: 2000}), tx, adminCfg
}

// TestTrainerApplicationLifecycleE2E exercises the full trainer-application
// journey through the complete router: register, login, apply, admin review by
// both administrators, approval (which creates the trainer profile for the same
// user identity), rejection with re-application, and a second approval by the
// other administrator. It also verifies the user identity never changes and the
// original login keeps working after approval.
func TestTrainerApplicationLifecycleE2E(t *testing.T) {
	router, tx, adminCfg := newE2ERouter(t)
	trainerRepo := repositories.NewTrainerRepository(tx)

	admin1 := adminByID(t, adminCfg, config.Admin1ID)
	admin2 := adminByID(t, adminCfg, config.Admin2ID)

	// --- Scenario 1: ADMIN_1 approves a fresh application ---
	userCookie := e2eRegisterUser(t, router, e2eUniqueEmail())
	rec, raw := e2eRequest(router, auth.AccessTokenCookieName, userCookie, http.MethodPost, "/api/v1/trainer/apply", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("apply: expected 201, got %d (body: %s)", rec.Code, raw)
	}
	firstApp := e2eDecodeApplication(t, raw)
	if firstApp.Status != "PENDING" {
		t.Fatalf("expected PENDING after apply, got %q", firstApp.Status)
	}

	admin1Cookie := e2eAdminLogin(t, router, admin1)

	rec, raw = e2eRequest(router, auth.AdminAccessTokenCookieName, admin1Cookie, http.MethodGet, "/api/v1/admin/trainer-applications?page=1&limit=100", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin list: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, firstApp.ID) {
		t.Fatalf("expected the fresh application in the admin list, got %s", raw)
	}

	rec, raw = e2eRequest(router, auth.AdminAccessTokenCookieName, admin1Cookie, http.MethodGet, "/api/v1/admin/trainer-applications/"+firstApp.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin get: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"status":"PENDING"`) {
		t.Fatalf("expected PENDING application, got %s", raw)
	}

	rec, raw = e2eRequest(router, auth.AdminAccessTokenCookieName, admin1Cookie, http.MethodPost, "/api/v1/admin/trainer-applications/"+firstApp.ID+"/approve", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin approve: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"status":"APPROVED"`) {
		t.Fatalf("expected APPROVED after approval, got %s", raw)
	}
	if !strings.Contains(raw, firstApp.UserID) {
		t.Fatalf("expected the same user id after approval, got %s", raw)
	}

	trainer, err := trainerRepo.FindByUserID(context.Background(), firstApp.UserID)
	if err != nil {
		t.Fatalf("expected a trainer profile to exist after approval: %v", err)
	}
	if trainer.UserID != firstApp.UserID {
		t.Fatalf("expected the trainer linked to the same user id, got %q", trainer.UserID)
	}

	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, userCookie, http.MethodGet, "/api/v1/me", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("the original login must keep working after approval, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, firstApp.UserID) {
		t.Fatalf("expected the same user identity in /me, got %s", raw)
	}

	// --- Scenario 2: ADMIN_2 rejects, the user reapplies, ADMIN_2 approves ---
	secondCookie := e2eRegisterUser(t, router, e2eUniqueEmail())
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, secondCookie, http.MethodPost, "/api/v1/trainer/apply", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("apply (second user): expected 201, got %d (body: %s)", rec.Code, raw)
	}
	secondApp := e2eDecodeApplication(t, raw)

	admin2Cookie := e2eAdminLogin(t, router, admin2)

	rec, raw = e2eRequest(router, auth.AdminAccessTokenCookieName, admin2Cookie, http.MethodPost, "/api/v1/admin/trainer-applications/"+secondApp.ID+"/reject", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin reject: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if _, err := trainerRepo.FindByUserID(context.Background(), secondApp.UserID); err == nil {
		t.Fatal("a rejection must never create a trainer profile")
	}

	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, secondCookie, http.MethodPost, "/api/v1/trainer/apply", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("re-apply: expected 201, got %d (body: %s)", rec.Code, raw)
	}
	reapplied := e2eDecodeApplication(t, raw)
	if reapplied.Status != "PENDING" {
		t.Fatalf("expected PENDING after re-apply, got %q", reapplied.Status)
	}

	rec, raw = e2eRequest(router, auth.AdminAccessTokenCookieName, admin2Cookie, http.MethodPost, "/api/v1/admin/trainer-applications/"+reapplied.ID+"/approve", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin approve after re-apply: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"status":"APPROVED"`) {
		t.Fatalf("expected APPROVED after re-apply approval, got %s", raw)
	}

	trainer, err = trainerRepo.FindByUserID(context.Background(), reapplied.UserID)
	if err != nil {
		t.Fatalf("expected exactly one trainer profile after the re-apply approval: %v", err)
	}
	if trainer.UserID != reapplied.UserID {
		t.Fatalf("expected the trainer linked to the reapplying user, got %q", trainer.UserID)
	}

	// --- Scenario 3: both administrators can list and review applications ---
	for _, adminCookie := range []string{admin1Cookie, admin2Cookie} {
		rec, raw = e2eRequest(router, auth.AdminAccessTokenCookieName, adminCookie, http.MethodGet, "/api/v1/admin/trainer-applications?page=1&limit=100", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("admin list must work for every role, got %d (body: %s)", rec.Code, raw)
		}
	}
}
