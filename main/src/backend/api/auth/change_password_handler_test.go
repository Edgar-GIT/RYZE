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
	"ryze/backend/services/change_password"
	"ryze/backend/services/login"
	"ryze/backend/services/password"
	"ryze/backend/services/token"
)

const changePasswordRoute = "/api/v1/auth/change-password"

// newChangePasswordTestRouter wires the real login, change-password and /me
// endpoints behind the real authentication middleware, backed by a database
// transaction so created users are rolled back.
func newChangePasswordTestRouter(t *testing.T, secure bool) (*gin.Engine, repositories.UserRepository, token.Service) {
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
	loginHandler := auth.NewLoginHandler(loginSvc, tokenSvc, testTokenTTL, secure)

	changeSvc := change_password.NewChangePasswordService(repo, password.Verifier{}, password.Hasher{})
	changeHandler := auth.NewChangePasswordHandler(changeSvc, secure)

	meHandler := auth.NewMeHandler(repo)

	router := gin.New()
	router.POST(loginRoute, loginHandler.Login)
	router.POST(changePasswordRoute, middleware.Authenticate(tokenSvc, repo), changeHandler.ChangePassword)
	router.GET(meRoute, middleware.Authenticate(tokenSvc, repo), meHandler.GetMe)

	return router, repo, tokenSvc
}

func changePasswordBody(current, newPassword string) string {
	return fmt.Sprintf(`{"current_password": %q, "new_password": %q}`, current, newPassword)
}

func attemptChangePassword(router http.Handler, cookieValue, body string) (*httptest.ResponseRecorder, string) {
	req := httptest.NewRequest(http.MethodPost, changePasswordRoute, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if cookieValue != "" {
		req.AddCookie(&http.Cookie{Name: "ryze_access_token", Value: cookieValue})
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec, rec.Body.String()
}

func loginForCookie(t *testing.T, router http.Handler, email, password string) string {
	t.Helper()

	rec, _, raw := attemptLogin(router, loginBody(email, password))
	if rec.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	return parseSetCookie(t, rec)[testCookieName]
}

func TestChangePasswordSuccess(t *testing.T) {
	router, repo, _ := newChangePasswordTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "OldPassword123!")
	cookie := loginForCookie(t, router, user.Email, "OldPassword123!")

	rec, raw := attemptChangePassword(router, cookie, changePasswordBody("OldPassword123!", "NewPassword456!"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"success":true`) {
		t.Fatalf("expected success:true, got %s", raw)
	}
	if !strings.Contains(raw, `"message":"Password changed successfully."`) {
		t.Fatalf("expected success message, got %s", raw)
	}
	if !strings.Contains(raw, `"data":{}`) {
		t.Fatalf("expected empty data object, got %s", raw)
	}
}

func TestChangePasswordClearsCookie(t *testing.T) {
	router, repo, _ := newChangePasswordTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "OldPassword123!")
	cookie := loginForCookie(t, router, user.Email, "OldPassword123!")

	rec, _ := attemptChangePassword(router, cookie, changePasswordBody("OldPassword123!", "NewPassword456!"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	cleared := parseSetCookie(t, rec)
	if value, ok := cleared[testCookieName]; !ok {
		t.Fatal("cookie ryze_access_token must be present in the clearing Set-Cookie")
	} else if value != "" {
		t.Fatalf("cookie value must be empty after the change, got %q", value)
	}
	if maxAge, _ := cleared["max-age"]; maxAge != "0" {
		t.Fatalf("expected Max-Age=0, got %q", maxAge)
	}
	if _, present := cleared["httponly"]; !present {
		t.Fatal("cookie must be HttpOnly")
	}
}

func TestChangePasswordRevokesOldToken(t *testing.T) {
	router, repo, _ := newChangePasswordTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "OldPassword123!")
	cookie := loginForCookie(t, router, user.Email, "OldPassword123!")

	rec, _, _ := requestMe(router, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected /me 200 before the change, got %d", rec.Code)
	}

	rec, raw := attemptChangePassword(router, cookie, changePasswordBody("OldPassword123!", "NewPassword456!"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	// The same token must no longer reach any protected endpoint: the session
	// is revoked server-side, not only cleared from the cookie.
	rec, _, raw = requestMe(router, cookie)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected /me 401 after the change, got %d (body: %s)", rec.Code, raw)
	}
}

func TestChangePasswordOldLoginFailsNewLoginWorks(t *testing.T) {
	router, repo, _ := newChangePasswordTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "OldPassword123!")
	cookie := loginForCookie(t, router, user.Email, "OldPassword123!")

	rec, raw := attemptChangePassword(router, cookie, changePasswordBody("OldPassword123!", "NewPassword456!"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	rec, _, _ = attemptLogin(router, loginBody(user.Email, "OldPassword123!"))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("login with old password: expected 401, got %d", rec.Code)
	}

	// A fresh login with the new password issues a valid session for the same
	// identity and /me succeeds again.
	newCookie := loginForCookie(t, router, user.Email, "NewPassword456!")
	rec, data, raw := requestMe(router, newCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected /me 200 after fresh login, got %d (body: %s)", rec.Code, raw)
	}
	if id, _ := data["id"].(string); id != user.ID {
		t.Fatalf("identity must be unchanged after the change: expected id %q, got %q", user.ID, id)
	}
	if email, _ := data["email"].(string); email != user.Email {
		t.Fatalf("expected email %q, got %q", user.Email, email)
	}
}

func TestChangePasswordWrongCurrentPassword(t *testing.T) {
	router, repo, _ := newChangePasswordTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "OldPassword123!")
	cookie := loginForCookie(t, router, user.Email, "OldPassword123!")

	rec, raw := attemptChangePassword(router, cookie, changePasswordBody("WrongPassword123!", "NewPassword456!"))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"INVALID_CREDENTIALS"`) {
		t.Fatalf("expected INVALID_CREDENTIALS, got %s", raw)
	}

	// The password must be unchanged after a rejected attempt.
	if rec, _, _ := attemptLogin(router, loginBody(user.Email, "OldPassword123!")); rec.Code != http.StatusOK {
		t.Fatal("password must be unchanged after a wrong-current-password attempt")
	}
}

func TestChangePasswordSoftDeletedUser(t *testing.T) {
	router, repo, tokenSvc := newChangePasswordTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "OldPassword123!")

	jwtValue, err := tokenSvc.GenerateAccessToken(user.ID, user.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	if err := repo.SoftDelete(t.Context(), user.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	rec, raw := attemptChangePassword(router, jwtValue, changePasswordBody("OldPassword123!", "NewPassword456!"))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
}

func TestChangePasswordEmptyCurrentPassword(t *testing.T) {
	router, repo, _ := newChangePasswordTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "OldPassword123!")
	cookie := loginForCookie(t, router, user.Email, "OldPassword123!")

	rec, raw := attemptChangePassword(router, cookie, changePasswordBody("", "NewPassword456!"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestChangePasswordEmptyNewPassword(t *testing.T) {
	router, repo, _ := newChangePasswordTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "OldPassword123!")
	cookie := loginForCookie(t, router, user.Email, "OldPassword123!")

	rec, raw := attemptChangePassword(router, cookie, changePasswordBody("OldPassword123!", ""))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestChangePasswordInvalidJSON(t *testing.T) {
	router, repo, _ := newChangePasswordTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "OldPassword123!")
	cookie := loginForCookie(t, router, user.Email, "OldPassword123!")

	rec, _ := attemptChangePassword(router, cookie, `{"current_password": "broken"`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON: expected 400, got %d", rec.Code)
	}
}

func TestChangePasswordNotAuthenticated(t *testing.T) {
	router, _, _ := newChangePasswordTestRouter(t, true)

	rec, raw := attemptChangePassword(router, "", changePasswordBody("OldPassword123!", "NewPassword456!"))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestChangePasswordResponseNeverExposesSecrets(t *testing.T) {
	router, repo, _ := newChangePasswordTestRouter(t, true)
	oldPlaintext := "OldPassword123!"
	newPlaintext := "NewPassword456!"
	user := seedLoginUser(t, repo, uniqueEmail(), oldPlaintext)
	cookie := loginForCookie(t, router, user.Email, oldPlaintext)

	rec, raw := attemptChangePassword(router, cookie, changePasswordBody(oldPlaintext, newPlaintext))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "password_hash") {
		t.Fatal("response must never contain password_hash")
	}
	if strings.Contains(raw, oldPlaintext) || strings.Contains(raw, newPlaintext) {
		t.Fatal("response must never contain plaintext passwords")
	}
	if strings.Contains(raw, cookie) {
		t.Fatal("response must never contain the JWT")
	}
	if strings.Contains(raw, testSecret) {
		t.Fatal("response must never expose JWT_SECRET")
	}
}

func TestChangePasswordInternalError(t *testing.T) {
	changeSvc := change_password.NewChangePasswordService(failingLoginRepository{}, password.Verifier{}, password.Hasher{})
	handler := auth.NewChangePasswordHandler(changeSvc, true)

	router := gin.New()
	router.POST(changePasswordRoute, middleware.Authenticate(token.NewService([]byte(testSecret), testTokenTTL), stubSessionProvider{version: 0}), handler.ChangePassword)

	jwtValue, err := token.NewService([]byte(testSecret), testTokenTTL).GenerateAccessToken("00000000-0000-0000-0000-000000000000", 0)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, raw := attemptChangePassword(router, jwtValue, changePasswordBody("OldPassword123!", "NewPassword456!"))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "repository failure") {
		t.Fatal("internal error details must never be exposed")
	}
}
