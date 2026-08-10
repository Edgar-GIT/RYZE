package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ryze/backend/api/auth"
	"ryze/backend/middleware"
	"ryze/backend/services/token"
)

const testSecret = "middleware-test-secret-that-is-longer-than-32-bytes-7"

func init() {
	gin.SetMode(gin.TestMode)
}

// failingTokenService simulates an unexpected token validation failure.
type failingTokenService struct{}

func (failingTokenService) GenerateAccessToken(string) (string, error) {
	return "", errors.New("generation failure")
}
func (failingTokenService) ValidateAccessToken(string) (string, error) {
	return "", errors.New("database exploded")
}

// newProtectedRouter mounts a guarded test route with the middleware and
// records whether the handler ran and the context user ID.
func newProtectedRouter(t *testing.T, svc token.Service) (*gin.Engine, *bool, *string) {
	t.Helper()

	reached := false
	userID := ""

	router := gin.New()
	router.GET("/protected", middleware.Authenticate(svc), func(c *gin.Context) {
		reached = true
		id, err := middleware.UserIDFromContext(c)
		if err == nil {
			userID = id
		}
		c.Status(http.StatusOK)
	})

	return router, &reached, &userID
}

func requestWithCookie(router http.Handler, cookieValue string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if cookieValue != "" {
		req.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: cookieValue})
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestValidCookieContinues(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached, _ := newProtectedRouter(t, svc)

	jwtValue, err := svc.GenerateAccessToken(uuid.NewString())
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec := requestWithCookie(router, jwtValue)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !*reached {
		t.Fatal("protected handler must be reached with a valid cookie")
	}
}

func TestValidCookieStoresUserUUID(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, _, userID := newProtectedRouter(t, svc)

	expected := uuid.NewString()
	jwtValue, err := svc.GenerateAccessToken(expected)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec := requestWithCookie(router, jwtValue)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if *userID != expected {
		t.Fatalf("expected context user id %q, got %q", expected, *userID)
	}
}

func TestMissingCookieRejected(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached, _ := newProtectedRouter(t, svc)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("protected handler must not be reached without a cookie")
	}
	assertAuthenticationError(t, rec.Body.String())
}

func TestEmptyCookieRejected(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached, _ := newProtectedRouter(t, svc)

	rec := requestWithCookie(router, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("protected handler must not be reached with an empty cookie")
	}
	assertAuthenticationError(t, rec.Body.String())
}

func TestMalformedJWTRejected(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, _, _ := newProtectedRouter(t, svc)

	for _, value := range []string{"not.a.jwt", "garbage", "a.b.c.d.e"} {
		rec := requestWithCookie(router, value)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("value %q: expected 401, got %d", value, rec.Code)
		}
		assertAuthenticationError(t, rec.Body.String())
	}
}

func TestExpiredJWTRejected(t *testing.T) {
	svc := token.NewService([]byte(testSecret), -1*time.Minute)
	router, _, _ := newProtectedRouter(t, svc)

	jwtValue, err := svc.GenerateAccessToken(uuid.NewString())
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec := requestWithCookie(router, jwtValue)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	assertAuthenticationError(t, rec.Body.String())
}

func TestWrongSecretJWTRejected(t *testing.T) {
	validSvc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, _, _ := newProtectedRouter(t, validSvc)

	other := token.NewService([]byte("other-secret-that-is-longer-than-32-bytes-99"), 15*time.Minute)
	jwtValue, err := other.GenerateAccessToken(uuid.NewString())
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec := requestWithCookie(router, jwtValue)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	assertAuthenticationError(t, rec.Body.String())
}

func TestInvalidUUIDSubjectRejected(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, _, _ := newProtectedRouter(t, svc)

	jwtValue, err := svc.GenerateAccessToken("not-a-uuid")
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec := requestWithCookie(router, jwtValue)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	assertAuthenticationError(t, rec.Body.String())
}

func TestMissingSubjectRejected(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, _, _ := newProtectedRouter(t, svc)

	jwtValue, err := svc.GenerateAccessToken("")
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec := requestWithCookie(router, jwtValue)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	assertAuthenticationError(t, rec.Body.String())
}

func TestInternalValidationFailureDoesNotExposeDetails(t *testing.T) {
	router, reached, _ := newProtectedRouter(t, failingTokenService{})

	jwtValue, err := token.NewService([]byte(testSecret), 15*time.Minute).GenerateAccessToken(uuid.NewString())
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec := requestWithCookie(router, jwtValue)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("protected handler must not be reached")
	}
	body := rec.Body.String()
	if strings.Contains(body, "database exploded") {
		t.Fatal("internal validation details must never be exposed")
	}
	assertAuthenticationError(t, body)
}

func TestUserIDFromContextValid(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	expected := uuid.NewString()
	c.Set(middleware.UserIDContextKey, expected)

	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		t.Fatalf("UserIDFromContext: %v", err)
	}
	if userID != expected {
		t.Fatalf("expected %q, got %q", expected, userID)
	}
}

func TestUserIDFromContextInvalidValues(t *testing.T) {
	cases := map[string]any{
		"missing":    nil,
		"wrong type": 12345,
		"empty":      "",
		"invalid":    "not-a-uuid",
	}

	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			if value != nil {
				c.Set(middleware.UserIDContextKey, value)
			}

			userID, err := middleware.UserIDFromContext(c)
			if !errors.Is(err, middleware.ErrNoAuthenticatedUser) {
				t.Fatalf("expected ErrNoAuthenticatedUser, got %v", err)
			}
			if userID != "" {
				t.Fatalf("expected empty user id, got %q", userID)
			}
		})
	}
}

func assertAuthenticationError(t *testing.T, body string) {
	t.Helper()

	if !strings.Contains(body, `"success":false`) {
		t.Fatalf("expected success:false, got %s", body)
	}
	if !strings.Contains(body, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", body)
	}
	if strings.Contains(body, "expired") || strings.Contains(body, "signature") || strings.Contains(body, "malformed") {
		t.Fatalf("authentication failures must remain indistinguishable, got %s", body)
	}
}
