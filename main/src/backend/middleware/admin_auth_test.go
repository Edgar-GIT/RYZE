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
	"ryze/backend/config"
	"ryze/backend/middleware"
	"ryze/backend/middleware/adminauthcontext"
	"ryze/backend/services/token"
)

// newAdminProtectedRouter mounts a guarded test route with the admin
// middleware and records whether the handler ran and the context admin
// identity. The middleware only depends on the Token Service, so this router
// never touches GORM or MariaDB.
func newAdminProtectedRouter(t *testing.T, svc token.Service) (*gin.Engine, *bool, *string) {
	t.Helper()

	reached := false
	adminID := ""

	router := gin.New()
	router.GET("/admin-protected", middleware.AdminAuthenticate(svc), func(c *gin.Context) {
		reached = true
		id, err := adminauthcontext.AdminIdentityFromContext(c)
		if err == nil {
			adminID = id
		}
		c.Status(http.StatusOK)
	})

	return router, &reached, &adminID
}

func adminRequestWithCookie(router http.Handler, cookieValue string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/admin-protected", nil)
	req.AddCookie(&http.Cookie{Name: auth.AdminAccessTokenCookieName, Value: cookieValue})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func adminRequest(router http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/admin-protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func validAdminToken(t *testing.T, svc token.Service, adminID string) string {
	t.Helper()
	jwtValue, err := svc.GenerateAdminToken(adminID)
	if err != nil {
		t.Fatalf("GenerateAdminToken: %v", err)
	}
	return jwtValue
}

func TestAdminValidTokenContinues(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)

	for _, adminID := range []string{config.Admin1ID, config.Admin2ID} {
		t.Run(adminID, func(t *testing.T) {
			router, reached, _ := newAdminProtectedRouter(t, svc)
			rec := adminRequestWithCookie(router, validAdminToken(t, svc, adminID))

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rec.Code)
			}
			if !*reached {
				t.Fatal("protected handler must be reached with a valid admin token")
			}
		})
	}
}

func TestAdminIdentityStoredInContext(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)

	for _, adminID := range []string{config.Admin1ID, config.Admin2ID} {
		t.Run(adminID, func(t *testing.T) {
			router, _, stored := newAdminProtectedRouter(t, svc)
			rec := adminRequestWithCookie(router, validAdminToken(t, svc, adminID))

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rec.Code)
			}
			if *stored != adminID {
				t.Fatalf("expected context admin identity %q, got %q", adminID, *stored)
			}
		})
	}
}

func TestAdminMissingCookieRejected(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached, _ := newAdminProtectedRouter(t, svc)

	rec := adminRequest(router)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("protected handler must not be reached without an admin cookie")
	}
	assertAuthenticationError(t, rec.Body.String())
}

func TestAdminEmptyCookieRejected(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached, _ := newAdminProtectedRouter(t, svc)

	rec := adminRequestWithCookie(router, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("protected handler must not be reached with an empty admin cookie")
	}
	assertAuthenticationError(t, rec.Body.String())
}

func TestAdminMalformedJWTRejected(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, _, _ := newAdminProtectedRouter(t, svc)

	for _, value := range []string{"not.a.jwt", "garbage", "a.b.c.d.e"} {
		rec := adminRequestWithCookie(router, value)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("value %q: expected 401, got %d", value, rec.Code)
		}
		assertAuthenticationError(t, rec.Body.String())
	}
}

func TestAdminExpiredJWTRejected(t *testing.T) {
	svc := token.NewService([]byte(testSecret), -1*time.Minute)
	router, reached, _ := newAdminProtectedRouter(t, svc)

	rec := adminRequestWithCookie(router, validAdminToken(t, svc, config.Admin1ID))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("protected handler must not be reached with an expired admin token")
	}
	assertAuthenticationError(t, rec.Body.String())
}

func TestAdminTamperedJWTRejected(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached, _ := newAdminProtectedRouter(t, svc)

	valid := validAdminToken(t, svc, config.Admin1ID)
	tampered := valid[:len(valid)-1]
	if tampered == valid {
		tampered += "x"
	}

	rec := adminRequestWithCookie(router, tampered)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("protected handler must not be reached with a tampered admin token")
	}
	assertAuthenticationError(t, rec.Body.String())
}

func TestAdminUserTokenRejected(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached, _ := newAdminProtectedRouter(t, svc)

	jwtValue, err := svc.GenerateAccessToken(uuid.NewString(), 0)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec := adminRequestWithCookie(router, jwtValue)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("protected handler must not be reached with a user token")
	}
	assertAuthenticationError(t, rec.Body.String())
}

func TestAdminStageTokenRejected(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached, _ := newAdminProtectedRouter(t, svc)

	jwtValue, err := svc.GenerateAdminStageToken(config.Admin1ID)
	if err != nil {
		t.Fatalf("GenerateAdminStageToken: %v", err)
	}

	rec := adminRequestWithCookie(router, jwtValue)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("protected handler must not be reached with a stage token")
	}
	assertAuthenticationError(t, rec.Body.String())
}

func TestAdminMissingIdentityRejected(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached, _ := newAdminProtectedRouter(t, svc)

	rec := adminRequestWithCookie(router, validAdminToken(t, svc, ""))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("protected handler must not be reached with a missing admin identity")
	}
	assertAuthenticationError(t, rec.Body.String())
}

func TestAdminInvalidIdentityRejected(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached, _ := newAdminProtectedRouter(t, svc)

	for _, identity := range []string{"ADMIN_3", "admin", "user-42"} {
		rec := adminRequestWithCookie(router, validAdminToken(t, svc, identity))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("identity %q: expected 401, got %d", identity, rec.Code)
		}
		if *reached {
			t.Fatalf("identity %q: protected handler must not be reached", identity)
		}
		assertAuthenticationError(t, rec.Body.String())
	}
}

func TestAdminWrongSecretRejected(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached, _ := newAdminProtectedRouter(t, svc)

	other := token.NewService([]byte("other-secret-that-is-longer-than-32-bytes-99"), 15*time.Minute)
	rec := adminRequestWithCookie(router, validAdminToken(t, other, config.Admin1ID))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("protected handler must not be reached with a wrong-secret token")
	}
	assertAuthenticationError(t, rec.Body.String())
}

func TestAdminInternalValidationFailureDoesNotExposeDetails(t *testing.T) {
	router, reached, _ := newAdminProtectedRouter(t, failingTokenService{})

	rec := adminRequestWithCookie(router, validAdminToken(t, token.NewService([]byte(testSecret), 15*time.Minute), config.Admin1ID))
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

func TestAdminIdentityFromContextValid(t *testing.T) {
	for _, adminID := range []string{config.Admin1ID, config.Admin2ID} {
		t.Run(adminID, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Set(adminauthcontext.AdminIdentityContextKey, adminID)

			got, err := adminauthcontext.AdminIdentityFromContext(c)
			if err != nil {
				t.Fatalf("AdminIdentityFromContext: %v", err)
			}
			if got != adminID {
				t.Fatalf("expected %q, got %q", adminID, got)
			}
		})
	}
}

func TestAdminIdentityFromContextInvalidValues(t *testing.T) {
	cases := map[string]any{
		"missing":           nil,
		"wrong type":        12345,
		"empty":             "",
		"not an admin":      "ADMIN_3",
		"arbitrary string":  "some-arbitrary-identity",
		"user-looking name": "user-42",
	}

	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			if value != nil {
				c.Set(adminauthcontext.AdminIdentityContextKey, value)
			}

			adminID, err := adminauthcontext.AdminIdentityFromContext(c)
			if !errors.Is(err, adminauthcontext.ErrNoAuthenticatedAdmin) {
				t.Fatalf("expected ErrNoAuthenticatedAdmin, got %v", err)
			}
			if adminID != "" {
				t.Fatalf("expected empty admin identity, got %q", adminID)
			}
		})
	}
}
