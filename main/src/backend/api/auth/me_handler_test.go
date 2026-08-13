package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ryze/backend/api/auth"
	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/middleware"
	"ryze/backend/repositories"
	"ryze/backend/services/token"
)

const meRoute = "/api/v1/me"

// newMeTestRouter builds a /me route protected by the real authentication
// middleware, backed by a real database transaction so users are rolled back.
func newMeTestRouter(t *testing.T) (*gin.Engine, repositories.UserRepository, token.Service) {
	t.Helper()

	config.LoadEnvFile()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("retrieve database handle: %v", err)
	}
	tx := db.Begin()
	t.Cleanup(func() {
		_ = tx.Rollback()
		_ = sqlDB.Close()
	})

	repo := repositories.NewUserRepository(tx)
	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)
	handler := auth.NewMeHandler(repo)

	router := gin.New()
	router.GET(meRoute, middleware.Authenticate(tokenSvc, repo), handler.GetMe)

	return router, repo, tokenSvc
}

// stubSessionProvider returns a fixed session version so handler tests can
// satisfy the middleware session check without touching the database.
type stubSessionProvider struct {
	version int
}

func (s stubSessionProvider) GetSessionVersion(_ context.Context, _ string) (int, error) {
	return s.version, nil
}

func requestMe(router http.Handler, cookieValue string) (*httptest.ResponseRecorder, map[string]any, string) {
	req := httptest.NewRequest(http.MethodGet, meRoute, bytes.NewBuffer(nil))
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
	raw, _ := json.Marshal(payload)
	return rec, data, string(raw)
}

func TestMeSuccess(t *testing.T) {
	router, repo, tokenSvc := newMeTestRouter(t)
	user := seedLoginUser(t, repo, uniqueEmail(), "Whatever123!")

	jwtValue, err := tokenSvc.GenerateAccessToken(user.ID, user.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, data, raw := requestMe(router, jwtValue)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if data == nil {
		t.Fatal("expected data object in response")
	}
	if id, _ := data["id"].(string); id != user.ID {
		t.Fatalf("expected id %q, got %q", user.ID, id)
	}
	if email, _ := data["email"].(string); email != user.Email {
		t.Fatalf("expected email %q, got %q", user.Email, email)
	}
	if firstName, _ := data["first_name"].(string); firstName != user.FirstName {
		t.Fatalf("expected first_name %q, got %q", user.FirstName, firstName)
	}
	if lastName, _ := data["last_name"].(string); lastName != user.LastName {
		t.Fatalf("expected last_name %q, got %q", user.LastName, lastName)
	}
	for _, field := range []string{"created_at", "updated_at"} {
		if value, _ := data[field].(string); value == "" {
			t.Fatalf("expected non-empty %s, got %q", field, value)
		}
	}
}

func TestMeNeverExposesSecrets(t *testing.T) {
	router, repo, tokenSvc := newMeTestRouter(t)
	user := seedLoginUser(t, repo, uniqueEmail(), "Whatever123!")

	jwtValue, err := tokenSvc.GenerateAccessToken(user.ID, user.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, data, raw := requestMe(router, jwtValue)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if _, ok := data["password_hash"]; ok {
		t.Fatal("response must not expose password_hash")
	}
	if _, ok := data["deleted_at"]; ok {
		t.Fatal("response must not expose deleted_at")
	}
	if strings.Contains(raw, jwtValue) {
		t.Fatal("response must never contain the JWT")
	}
	if strings.Contains(raw, "access_token") {
		t.Fatal("response must not contain an access_token field")
	}
	if strings.Contains(raw, testSecret) {
		t.Fatal("response must never expose JWT_SECRET")
	}
}

func TestMeMissingCookie(t *testing.T) {
	router, _, _ := newMeTestRouter(t)

	rec, _, raw := requestMe(router, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestMeInvalidJWT(t *testing.T) {
	router, _, _ := newMeTestRouter(t)

	rec, _, _ := requestMe(router, "not.a.jwt")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestMeExpiredJWT(t *testing.T) {
	router, _, _ := newMeTestRouter(t)
	expired := token.NewService([]byte(testSecret), -1*time.Minute)

	jwtValue, err := expired.GenerateAccessToken(uuid.NewString(), 0)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, _ := requestMe(router, jwtValue)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestMeUnknownUserID(t *testing.T) {
	router, _, tokenSvc := newMeTestRouter(t)

	jwtValue, err := tokenSvc.GenerateAccessToken(uuid.NewString(), 0)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := requestMe(router, jwtValue)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
}

func TestMeSoftDeletedUser(t *testing.T) {
	router, repo, tokenSvc := newMeTestRouter(t)
	user := seedLoginUser(t, repo, uniqueEmail(), "Whatever123!")

	if err := repo.SoftDelete(context.Background(), user.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	jwtValue, err := tokenSvc.GenerateAccessToken(user.ID, user.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, _ := requestMe(router, jwtValue)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestMeRepositoryFailure(t *testing.T) {
	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)
	handler := auth.NewMeHandler(failingLoginRepository{})

	router := gin.New()
	router.GET(meRoute, middleware.Authenticate(tokenSvc, stubSessionProvider{version: 0}), handler.GetMe)

	jwtValue, err := tokenSvc.GenerateAccessToken(uuid.NewString(), 0)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := requestMe(router, jwtValue)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "repository failure") {
		t.Fatal("internal error details must never be exposed")
	}
}
