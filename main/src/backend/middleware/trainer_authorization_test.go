package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ryze/backend/middleware"
	"ryze/backend/middleware/trainercontext"
	"ryze/backend/middleware/trainerroles"
)

// newTrainerPermissionRouter mounts RequireTrainerPermission(permissions...) on
// /trainer/protected and records whether the handler ran. The identity value is
// stored in trainercontext before the middleware runs, simulating
// TrainerAuthenticate. Pass nil to simulate an absent trainer identity and any
// other value to simulate a malformed identity.
func newTrainerPermissionRouter(identity any, permissions ...trainerroles.Permission) (*gin.Engine, *bool) {
	reached := false
	router := gin.New()

	router.GET("/trainer/protected", func(c *gin.Context) {
		if identity != nil {
			c.Set(trainercontext.TrainerContextKey, identity)
		}

		middleware.RequireTrainerPermission(permissions...)(c)

		if !c.IsAborted() {
			reached = true
			c.Status(http.StatusOK)
		}
	})

	return router, &reached
}

func validTrainerIdentity() trainercontext.Identity {
	return trainercontext.Identity{
		UserID:    uuid.NewString(),
		TrainerID: uuid.NewString(),
	}
}

func trainerProtectedRequest(router http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/trainer/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func trainerProtectedRequestWithClaims(
	router http.Handler,
	query map[string]string,
	headers map[string]string,
) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/trainer/protected", nil)

	values := req.URL.Query()
	for key, value := range query {
		values.Set(key, value)
	}
	req.URL.RawQuery = values.Encode()

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// assertTrainerForbiddenNoLeak verifies the FORBIDDEN envelope never reveals
// the permission model or the required permissions.
func assertTrainerForbiddenNoLeak(t *testing.T, body string) {
	t.Helper()

	for _, sensitive := range []string{
		"trainer.profile",
		"trainer.clients",
		"trainer.programs",
		"trainer.statistics",
		"permission",
		"role",
	} {
		if strings.Contains(body, sensitive) {
			t.Fatalf("forbidden error must not reveal %q, got %s", sensitive, body)
		}
	}
}

func TestRequireTrainerPermissionAllowsEveryOfficialPermission(t *testing.T) {
	for _, permission := range trainerroles.AllPermissions() {
		t.Run(string(permission), func(t *testing.T) {
			router, reached := newTrainerPermissionRouter(validTrainerIdentity(), permission)
			rec := trainerProtectedRequest(router)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}
			if !*reached {
				t.Fatal("handler must be reached when the trainer holds the permission")
			}
		})
	}
}

func TestRequireTrainerPermissionAllowsAnyOfMultiplePermissions(t *testing.T) {
	router, reached := newTrainerPermissionRouter(
		validTrainerIdentity(),
		trainerroles.PermissionPrograms,
		trainerroles.PermissionStatistics,
	)
	rec := trainerProtectedRequest(router)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !*reached {
		t.Fatal("handler must be reached when at least one permission is granted")
	}
}

func TestRequireTrainerPermissionAllowsGrantedPermissionAmongUnknownOnes(t *testing.T) {
	router, reached := newTrainerPermissionRouter(
		validTrainerIdentity(),
		trainerroles.Permission("trainer.schedule"),
		trainerroles.PermissionProfile,
	)
	rec := trainerProtectedRequest(router)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !*reached {
		t.Fatal("a granted permission among unknown ones must be enough")
	}
}

func TestRequireTrainerPermissionIgnoresRequestPermissionInput(t *testing.T) {
	router, reached := newTrainerPermissionRouter(validTrainerIdentity(), trainerroles.PermissionProfile)

	rec := trainerProtectedRequestWithClaims(
		router,
		map[string]string{"permission": "trainer.schedule"},
		map[string]string{"X-Permission": "trainer.schedule"},
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !*reached {
		t.Fatal("request-supplied permissions must never be honored")
	}
}

func TestRequireTrainerPermissionRejectsMissingIdentity(t *testing.T) {
	router, reached := newTrainerPermissionRouter(nil, trainerroles.PermissionProfile)

	rec := trainerProtectedRequest(router)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("handler must not be reached without a trainer identity")
	}
	assertTrainerAuthenticationError(t, rec.Body.String())
}

func TestRequireTrainerPermissionRejectsWrongTypeIdentity(t *testing.T) {
	for name, value := range map[string]any{
		"string":  "some-user-id",
		"integer": 42,
		"slice":   []string{"some-user-id"},
	} {
		t.Run(name, func(t *testing.T) {
			router, reached := newTrainerPermissionRouter(value, trainerroles.PermissionProfile)
			rec := trainerProtectedRequest(router)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", rec.Code)
			}
			if *reached {
				t.Fatal("handler must not be reached with a malformed identity")
			}
			assertTrainerAuthenticationError(t, rec.Body.String())
		})
	}
}

func TestRequireTrainerPermissionRejectsInvalidIdentity(t *testing.T) {
	valid := validTrainerIdentity()

	cases := map[string]trainercontext.Identity{
		"empty both":      {UserID: "", TrainerID: ""},
		"empty user":      {UserID: "", TrainerID: valid.TrainerID},
		"empty trainer":   {UserID: valid.UserID, TrainerID: ""},
		"invalid user":    {UserID: "not-a-uuid", TrainerID: valid.TrainerID},
		"invalid trainer": {UserID: valid.UserID, TrainerID: "not-a-uuid"},
	}

	for name, identity := range cases {
		t.Run(name, func(t *testing.T) {
			router, reached := newTrainerPermissionRouter(identity, trainerroles.PermissionProfile)
			rec := trainerProtectedRequest(router)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", rec.Code)
			}
			if *reached {
				t.Fatal("handler must not be reached with an invalid identity")
			}
			assertTrainerAuthenticationError(t, rec.Body.String())
		})
	}
}

func TestRequireTrainerPermissionRejectsUnknownPermission(t *testing.T) {
	router, reached := newTrainerPermissionRouter(
		validTrainerIdentity(),
		trainerroles.Permission("trainer.schedule"),
	)

	rec := trainerProtectedRequest(router)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("handler must not be reached with an unknown permission")
	}
	assertTrainerForbiddenError(t, rec.Body.String())
	assertTrainerForbiddenNoLeak(t, rec.Body.String())
}

func TestRequireTrainerPermissionRejectsNoRequiredPermissions(t *testing.T) {
	router, reached := newTrainerPermissionRouter(validTrainerIdentity())

	rec := trainerProtectedRequest(router)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("handler must not be reached without required permissions")
	}
	assertTrainerForbiddenError(t, rec.Body.String())
	assertTrainerForbiddenNoLeak(t, rec.Body.String())
}

func TestRequireTrainerPermissionRejectsRequestIdentityClaims(t *testing.T) {
	router, reached := newTrainerPermissionRouter(nil, trainerroles.PermissionProfile)

	rec := trainerProtectedRequestWithClaims(
		router,
		map[string]string{
			"user_id":    uuid.NewString(),
			"trainer_id": uuid.NewString(),
		},
		map[string]string{
			"X-User-ID":    uuid.NewString(),
			"X-Trainer-ID": uuid.NewString(),
		},
	)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("request-supplied identity must never authenticate a request")
	}
	assertTrainerAuthenticationError(t, rec.Body.String())
}

func TestRequireTrainerPermissionFailsClosedOnMisconfiguredMount(t *testing.T) {
	router, reached := newTrainerPermissionRouter(
		nil,
		trainerroles.Permission("trainer.schedule"),
	)

	rec := trainerProtectedRequest(router)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("handler must not be reached on a misconfigured mount")
	}
	assertTrainerAuthenticationError(t, rec.Body.String())
}

func TestRequireTrainerPermissionForbiddenDoesNotRevealAuthorizationDetails(t *testing.T) {
	identity := validTrainerIdentity()
	router, reached := newTrainerPermissionRouter(
		identity,
		trainerroles.Permission("trainer.schedule"),
	)

	rec := trainerProtectedRequest(router)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("handler must not be reached")
	}

	body := rec.Body.String()
	for _, sensitive := range []string{
		"trainer.schedule",
		identity.UserID,
		identity.TrainerID,
	} {
		if strings.Contains(body, sensitive) {
			t.Fatalf("forbidden error must not reveal %q, got %s", sensitive, body)
		}
	}
	assertTrainerForbiddenNoLeak(t, body)
}
