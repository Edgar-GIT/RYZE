package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/login"
	"ryze/backend/services/password"
	"ryze/backend/services/token"
)

const (
	loginRoute     = "/api/v1/auth/login"
	testSecret     = "login-test-secret-that-is-longer-than-32-bytes-77"
	testTokenTTL   = 15 * time.Minute
	testCookieName = "ryze_access_token"
)

// newLoginTestRouter builds a router backed by a real database transaction so
// created users are rolled back and never persisted. The token service uses a
// fixed test secret so returned tokens can be validated in the tests.
func newLoginTestRouter(t *testing.T, secure bool) (*gin.Engine, repositories.UserRepository, token.Service) {
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

	tx := db.Begin()
	t.Cleanup(func() { tx.Rollback() })

	repo := repositories.NewUserRepository(tx)
	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)
	loginSvc := login.NewLoginService(repo, password.Verifier{})
	handler := auth.NewLoginHandler(loginSvc, tokenSvc, testTokenTTL, secure)

	router := gin.New()
	router.POST(loginRoute, handler.Login)

	return router, repo, tokenSvc
}

func seedLoginUser(t *testing.T, repo repositories.UserRepository, email, plaintext string) *models.User {
	t.Helper()

	hash, err := password.HashPassword(plaintext)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	user := &models.User{
		Email:        email,
		PasswordHash: hash,
		FirstName:    "Login",
		LastName:     "Test",
	}
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user
}

func attemptLogin(router http.Handler, body string) (*httptest.ResponseRecorder, map[string]any, string) {
	req := httptest.NewRequest(http.MethodPost, loginRoute, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
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

func loginBody(email, password string) string {
	return fmt.Sprintf(`{"email": %q, "password": %q}`, email, password)
}

// parseSetCookie extracts the attributes of the first Set-Cookie header.
func parseSetCookie(t *testing.T, rec *httptest.ResponseRecorder) map[string]string {
	t.Helper()

	header := rec.Header().Get("Set-Cookie")
	if header == "" {
		t.Fatal("expected a Set-Cookie header")
	}

	attrs := map[string]string{}
	for _, part := range strings.Split(header, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		value := ""
		if len(kv) == 2 {
			value = kv[1]
		}
		attrs[strings.ToLower(kv[0])] = value
	}
	return attrs
}

func TestLoginSuccess(t *testing.T) {
	router, repo, _ := newLoginTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "CorrectPassword1!")

	rec, data, raw := attemptLogin(router, loginBody(user.Email, "CorrectPassword1!"))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if data == nil {
		t.Fatal("expected data object in response")
	}
	if _, ok := data["access_token"]; ok {
		t.Fatal("JSON response must not contain access_token")
	}
	if _, ok := data["token"]; ok {
		t.Fatal("JSON response must not contain a token")
	}
}

func TestLoginSetsHttpOnlyCookie(t *testing.T) {
	router, repo, _ := newLoginTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "CorrectPassword1!")

	rec, _, raw := attemptLogin(router, loginBody(user.Email, "CorrectPassword1!"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	cookie := parseSetCookie(t, rec)

	token, ok := cookie[testCookieName]
	if !ok {
		t.Fatal("cookie ryze_access_token must be set")
	}
	if token == "" {
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

func TestLoginSecureCookieDisabledForLocalHTTP(t *testing.T) {
	router, repo, _ := newLoginTestRouter(t, false)
	user := seedLoginUser(t, repo, uniqueEmail(), "CorrectPassword1!")

	rec, _, raw := attemptLogin(router, loginBody(user.Email, "CorrectPassword1!"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	cookie := parseSetCookie(t, rec)
	if _, present := cookie["secure"]; present {
		t.Fatal("Secure must be disabled for local HTTP development")
	}
}

func TestLoginTokenValidatesAndMatchesSubject(t *testing.T) {
	router, repo, tokenSvc := newLoginTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "CorrectPassword1!")

	rec, _, raw := attemptLogin(router, loginBody(user.Email, "CorrectPassword1!"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	cookie := parseSetCookie(t, rec)
	jwtValue, ok := cookie[testCookieName]
	if !ok {
		t.Fatal("cookie ryze_access_token must be set")
	}

	subject, err := tokenSvc.ValidateAccessToken(jwtValue)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}
	if subject != user.ID {
		t.Fatalf("expected token subject %q, got %q", user.ID, subject)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	router, repo, _ := newLoginTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "CorrectPassword1!")

	rec, _, _ := attemptLogin(router, loginBody(user.Email, "WrongPassword1!"))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password: expected 401, got %d", rec.Code)
	}
}

func TestLoginNonExistentEmail(t *testing.T) {
	router, _, _ := newLoginTestRouter(t, true)

	rec, _, _ := attemptLogin(router, loginBody(uniqueEmail(), "Whatever123!"))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("non-existent email: expected 401, got %d", rec.Code)
	}
}

func TestLoginSoftDeletedUser(t *testing.T) {
	router, repo, _ := newLoginTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "CorrectPassword1!")

	if err := repo.SoftDelete(context.Background(), user.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	rec, _, _ := attemptLogin(router, loginBody(user.Email, "CorrectPassword1!"))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("soft-deleted user: expected 401, got %d", rec.Code)
	}
}

func TestLoginInvalidJSON(t *testing.T) {
	router, _, _ := newLoginTestRouter(t, true)

	rec, _, _ := attemptLogin(router, `{"email": "broken"`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON: expected 400, got %d", rec.Code)
	}
}

func TestLoginMissingFields(t *testing.T) {
	router, _, _ := newLoginTestRouter(t, true)

	for name, body := range map[string]string{
		"empty object":     `{}`,
		"missing email":    `{"password": "Whatever123!"}`,
		"missing password": `{"email": "someone@ryze.local"}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec, _, _ := attemptLogin(router, body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d (body: %s)", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestLoginInternalError(t *testing.T) {
	loginSvc := login.NewLoginService(failingLoginRepository{}, password.Verifier{})
	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)
	handler := auth.NewLoginHandler(loginSvc, tokenSvc, testTokenTTL, true)

	router := gin.New()
	router.POST(loginRoute, handler.Login)

	rec, _, raw := attemptLogin(router, loginBody("someone@ryze.local", "Whatever123!"))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "repository failure") {
		t.Fatal("internal error details must never be exposed")
	}
}

func TestLoginResponseNeverExposesSecrets(t *testing.T) {
	router, repo, _ := newLoginTestRouter(t, true)
	plaintext := "SuperSecret1!"
	user := seedLoginUser(t, repo, uniqueEmail(), plaintext)

	rec, data, raw := attemptLogin(router, loginBody(user.Email, plaintext))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if _, ok := data["password_hash"]; ok {
		t.Fatal("response must not expose password_hash")
	}
	if strings.Contains(raw, plaintext) {
		t.Fatal("response must never contain the plaintext password")
	}
	if strings.Contains(raw, testSecret) {
		t.Fatal("response must never expose JWT_SECRET")
	}

	cookie := parseSetCookie(t, rec)
	jwtValue := cookie[testCookieName]
	if jwtValue != "" && strings.Contains(raw, jwtValue) {
		t.Fatal("JWT must exist only in the Set-Cookie header, never in the JSON response")
	}
}

// failingLoginRepository simulates an unexpected repository failure during
// credential lookup.
type failingLoginRepository struct{}

var errLoginRepoFailure = errors.New("repository failure")

func (failingLoginRepository) Create(_ context.Context, _ *models.User) error {
	return errLoginRepoFailure
}
func (failingLoginRepository) FindByID(_ context.Context, _ string) (*models.User, error) {
	return nil, errLoginRepoFailure
}
func (failingLoginRepository) FindByEmail(_ context.Context, _ string) (*models.User, error) {
	return nil, errLoginRepoFailure
}
func (failingLoginRepository) FindByEmailIncludingDeleted(_ context.Context, _ string) (*models.User, error) {
	return nil, errLoginRepoFailure
}
func (failingLoginRepository) Update(_ context.Context, _ *models.User) error {
	return errLoginRepoFailure
}
func (failingLoginRepository) Reactivate(_ context.Context, _ *models.User) error {
	return errLoginRepoFailure
}
func (failingLoginRepository) SoftDelete(_ context.Context, _ string) error {
	return errLoginRepoFailure
}
