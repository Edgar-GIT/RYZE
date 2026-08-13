package auth_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/middleware"
	"ryze/backend/repositories"
	"ryze/backend/services/delete_account"
	"ryze/backend/services/login"
	"ryze/backend/services/password"
	"ryze/backend/services/token"
)

const deleteAccountRoute = "/api/v1/auth/delete-account"

// newDeleteAccountTestRouter wires the real login, delete-account and /me
// endpoints behind the real authentication middleware, backed by a database
// transaction so created users are rolled back.
func newDeleteAccountTestRouter(t *testing.T, secure bool) (*gin.Engine, repositories.UserRepository, token.Service) {
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

	loginSvc := login.NewLoginService(repo, password.Verifier{})
	loginHandler := auth.NewLoginHandler(loginSvc, tokenSvc, testTokenTTL, secure)

	deleteSvc := delete_account.NewDeleteAccountService(repo, password.Verifier{})
	deleteHandler := auth.NewDeleteAccountHandler(deleteSvc, secure)

	meHandler := auth.NewMeHandler(repo)

	router := gin.New()
	router.POST(loginRoute, loginHandler.Login)
	router.POST(deleteAccountRoute, middleware.Authenticate(tokenSvc, repo), deleteHandler.DeleteAccount)
	router.GET(meRoute, middleware.Authenticate(tokenSvc, repo), meHandler.GetMe)

	return router, repo, tokenSvc
}

func deleteAccountBody(password string) string {
	return fmt.Sprintf(`{"password": %q}`, password)
}

func attemptDeleteAccount(router http.Handler, cookieValue, body string) (*httptest.ResponseRecorder, string) {
	req := httptest.NewRequest(http.MethodPost, deleteAccountRoute, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if cookieValue != "" {
		req.AddCookie(&http.Cookie{Name: "ryze_access_token", Value: cookieValue})
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec, rec.Body.String()
}

func TestDeleteAccountSuccess(t *testing.T) {
	router, repo, _ := newDeleteAccountTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	cookie := loginForCookie(t, router, user.Email, "Password123!")

	rec, raw := attemptDeleteAccount(router, cookie, deleteAccountBody("Password123!"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"success":true`) {
		t.Fatalf("expected success:true, got %s", raw)
	}
	if !strings.Contains(raw, `"message":"Account deleted successfully."`) {
		t.Fatalf("expected success message, got %s", raw)
	}
	if !strings.Contains(raw, `"data":{}`) {
		t.Fatalf("expected empty data object, got %s", raw)
	}
}

func TestDeleteAccountClearsCookie(t *testing.T) {
	router, repo, _ := newDeleteAccountTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	cookie := loginForCookie(t, router, user.Email, "Password123!")

	rec, _ := attemptDeleteAccount(router, cookie, deleteAccountBody("Password123!"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	cleared := parseSetCookie(t, rec)
	if value, ok := cleared[testCookieName]; !ok {
		t.Fatal("cookie ryze_access_token must be present in the clearing Set-Cookie")
	} else if value != "" {
		t.Fatalf("cookie value must be empty after deletion, got %q", value)
	}
	if maxAge, _ := cleared["max-age"]; maxAge != "0" {
		t.Fatalf("expected Max-Age=0, got %q", maxAge)
	}
	if _, present := cleared["httponly"]; !present {
		t.Fatal("cookie must be HttpOnly")
	}
}

func TestDeleteAccountRevokesOldToken(t *testing.T) {
	router, repo, _ := newDeleteAccountTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	cookie := loginForCookie(t, router, user.Email, "Password123!")

	rec, _, _ := requestMe(router, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected /me 200 before the deletion, got %d", rec.Code)
	}

	rec, raw := attemptDeleteAccount(router, cookie, deleteAccountBody("Password123!"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	// The same token must no longer reach any protected endpoint: the session
	// is revoked server-side, not only cleared from the cookie.
	rec, _, raw = requestMe(router, cookie)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected /me 401 after the deletion, got %d (body: %s)", rec.Code, raw)
	}
}

func TestDeleteAccountOldLoginFailsAndRowPreserved(t *testing.T) {
	router, repo, _ := newDeleteAccountTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	cookie := loginForCookie(t, router, user.Email, "Password123!")

	rec, raw := attemptDeleteAccount(router, cookie, deleteAccountBody("Password123!"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	rec, _, _ = attemptLogin(router, loginBody(user.Email, "Password123!"))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("login for a deleted account: expected 401, got %d", rec.Code)
	}

	// The row is soft-deleted, not removed: id, email and created_at survive.
	stored, err := repo.FindByEmailIncludingDeleted(t.Context(), user.Email)
	if err != nil {
		t.Fatalf("FindByEmailIncludingDeleted: %v", err)
	}
	if !stored.DeletedAt.Valid {
		t.Fatal("deleted_at must be set after account deletion")
	}
	if stored.ID != user.ID {
		t.Fatalf("id must be preserved: expected %q, got %q", user.ID, stored.ID)
	}
	if stored.Email != user.Email {
		t.Fatalf("email must be preserved: expected %q, got %q", user.Email, stored.Email)
	}
	if !stored.CreatedAt.Equal(user.CreatedAt) {
		t.Fatal("created_at must be preserved after account deletion")
	}
	if stored.SessionVersion != 1 {
		t.Fatalf("expected session version 1 after deletion, got %d", stored.SessionVersion)
	}
}

func TestDeleteAccountWrongPassword(t *testing.T) {
	router, repo, _ := newDeleteAccountTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	cookie := loginForCookie(t, router, user.Email, "Password123!")

	rec, raw := attemptDeleteAccount(router, cookie, deleteAccountBody("WrongPassword123!"))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"INVALID_CREDENTIALS"`) {
		t.Fatalf("expected INVALID_CREDENTIALS, got %s", raw)
	}

	// The account must remain active after a rejected attempt.
	if rec, _, _ := requestMe(router, cookie); rec.Code != http.StatusOK {
		t.Fatal("account must remain active after a wrong-password attempt")
	}
}

func TestDeleteAccountEmptyPassword(t *testing.T) {
	router, repo, _ := newDeleteAccountTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	cookie := loginForCookie(t, router, user.Email, "Password123!")

	rec, raw := attemptDeleteAccount(router, cookie, deleteAccountBody(""))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestDeleteAccountInvalidJSON(t *testing.T) {
	router, repo, _ := newDeleteAccountTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	cookie := loginForCookie(t, router, user.Email, "Password123!")

	rec, _ := attemptDeleteAccount(router, cookie, `{"password": "broken"`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON: expected 400, got %d", rec.Code)
	}
}

func TestDeleteAccountNotAuthenticated(t *testing.T) {
	router, _, _ := newDeleteAccountTestRouter(t, true)

	rec, raw := attemptDeleteAccount(router, "", deleteAccountBody("Password123!"))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestDeleteAccountSoftDeletedUser(t *testing.T) {
	router, repo, tokenSvc := newDeleteAccountTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")

	jwtValue, err := tokenSvc.GenerateAccessToken(user.ID, user.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	if err := repo.SoftDelete(t.Context(), user.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	rec, raw := attemptDeleteAccount(router, jwtValue, deleteAccountBody("Password123!"))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestDeleteAccountResponseNeverExposesSecrets(t *testing.T) {
	router, repo, _ := newDeleteAccountTestRouter(t, true)
	plaintext := "Password123!"
	user := seedLoginUser(t, repo, uniqueEmail(), plaintext)
	cookie := loginForCookie(t, router, user.Email, plaintext)

	rec, raw := attemptDeleteAccount(router, cookie, deleteAccountBody(plaintext))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "password") {
		t.Fatal("response must never mention the password")
	}
	if strings.Contains(raw, plaintext) {
		t.Fatal("response must never contain plaintext passwords")
	}
	if strings.Contains(raw, cookie) {
		t.Fatal("response must never contain the JWT")
	}
	if strings.Contains(raw, testSecret) {
		t.Fatal("response must never expose JWT_SECRET")
	}
	if strings.Contains(raw, "deleted_at") {
		t.Fatal("response must never expose internal deleted_at details")
	}
}

func TestDeleteAccountInternalError(t *testing.T) {
	deleteSvc := delete_account.NewDeleteAccountService(failingLoginRepository{}, password.Verifier{})
	handler := auth.NewDeleteAccountHandler(deleteSvc, true)

	router := gin.New()
	router.POST(deleteAccountRoute, middleware.Authenticate(token.NewService([]byte(testSecret), testTokenTTL), stubSessionProvider{version: 0}), handler.DeleteAccount)

	jwtValue, err := token.NewService([]byte(testSecret), testTokenTTL).GenerateAccessToken("00000000-0000-0000-0000-000000000000", 0)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, raw := attemptDeleteAccount(router, jwtValue, deleteAccountBody("Password123!"))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "repository failure") {
		t.Fatal("internal error details must never be exposed")
	}
}
