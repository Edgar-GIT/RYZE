package middleware_test

import (
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
	"ryze/backend/middleware/adminroles"
	"ryze/backend/services/token"
)

// newRoleProtectedRouter mounts AdminAuthenticate followed by
// RequireAdminRole(roles...) on /role-protected and records whether the
// handler ran. The middleware only depends on the Token Service, so this
// router never touches GORM or MariaDB.
func newRoleProtectedRouter(t *testing.T, svc token.Service, roles ...adminroles.Role) (*gin.Engine, *bool) {
	t.Helper()

	reached := false
	router := gin.New()
	router.GET("/role-protected", middleware.AdminAuthenticate(svc), middleware.RequireAdminRole(roles...), func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})

	return router, &reached
}

// newRoleOnlyRouter mounts RequireAdminRole without AdminAuthenticate to prove
// the authorization middleware fails closed on misconfiguration.
func newRoleOnlyRouter(roles ...adminroles.Role) (*gin.Engine, *bool) {
	reached := false
	router := gin.New()
	router.GET("/role-only", middleware.RequireAdminRole(roles...), func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})
	return router, &reached
}

// newPermissionProtectedRouter mounts AdminAuthenticate followed by
// RequireAdminPermission(permissions...) on /permission-protected and records
// whether the handler ran.
func newPermissionProtectedRouter(t *testing.T, svc token.Service, permissions ...adminroles.Permission) (*gin.Engine, *bool) {
	t.Helper()

	reached := false
	router := gin.New()
	router.GET("/permission-protected", middleware.AdminAuthenticate(svc), middleware.RequireAdminPermission(permissions...), func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})

	return router, &reached
}

// newPermissionOnlyRouter mounts RequireAdminPermission without AdminAuthenticate
// to prove the authorization middleware fails closed on misconfiguration.
func newPermissionOnlyRouter(permissions ...adminroles.Permission) (*gin.Engine, *bool) {
	reached := false
	router := gin.New()
	router.GET("/permission-only", middleware.RequireAdminPermission(permissions...), func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})
	return router, &reached
}

func roleRequestWithCookie(router http.Handler, cookieValue string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/role-protected", nil)
	req.AddCookie(&http.Cookie{Name: auth.AdminAccessTokenCookieName, Value: cookieValue})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func roleRequest(router http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/role-protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func roleOnlyRequest(router http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/role-only", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertForbiddenError(t *testing.T, body string) {
	t.Helper()
	if !strings.Contains(body, `"success":false`) {
		t.Fatalf("expected success:false, got %s", body)
	}
	if !strings.Contains(body, `"code":"FORBIDDEN"`) {
		t.Fatalf("expected FORBIDDEN, got %s", body)
	}
}

func assertNoAuthorizationLeak(t *testing.T, body string) {
	t.Helper()
	for _, sensitive := range []string{
		"TECHNICAL_ADMINISTRATOR",
		"MANAGEMENT_ADMINISTRATOR",
		"ADMIN_1",
		"ADMIN_2",
		"role",
		"permission",
	} {
		if strings.Contains(body, sensitive) {
			t.Fatalf("authorization failure must not reveal %q, got %s", sensitive, body)
		}
	}
}

func TestTechnicalAdminAllowedOnTechnicalResource(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached := newRoleProtectedRouter(t, svc, adminroles.RoleTechnicalAdministrator)

	rec := roleRequestWithCookie(router, validAdminToken(t, svc, config.Admin1ID))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !*reached {
		t.Fatal("technical admin must reach the technical resource")
	}
}

func TestManagementAdminRejectedOnTechnicalResource(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached := newRoleProtectedRouter(t, svc, adminroles.RoleTechnicalAdministrator)

	rec := roleRequestWithCookie(router, validAdminToken(t, svc, config.Admin2ID))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("management admin must not reach the technical resource")
	}
	assertForbiddenError(t, rec.Body.String())
	assertNoAuthorizationLeak(t, rec.Body.String())
}

func TestManagementAdminAllowedOnManagementResource(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached := newRoleProtectedRouter(t, svc, adminroles.RoleManagementAdministrator)

	rec := roleRequestWithCookie(router, validAdminToken(t, svc, config.Admin2ID))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !*reached {
		t.Fatal("management admin must reach the management resource")
	}
}

func TestTechnicalAdminRejectedOnManagementResource(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached := newRoleProtectedRouter(t, svc, adminroles.RoleManagementAdministrator)

	rec := roleRequestWithCookie(router, validAdminToken(t, svc, config.Admin1ID))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("technical admin must not reach the management resource")
	}
	assertForbiddenError(t, rec.Body.String())
	assertNoAuthorizationLeak(t, rec.Body.String())
}

func TestSharedResourceAllowedForBothRoles(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)

	for _, adminID := range []string{config.Admin1ID, config.Admin2ID} {
		t.Run(adminID, func(t *testing.T) {
			router, reached := newRoleProtectedRouter(t, svc, adminroles.RoleTechnicalAdministrator, adminroles.RoleManagementAdministrator)
			rec := roleRequestWithCookie(router, validAdminToken(t, svc, adminID))

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rec.Code)
			}
			if !*reached {
				t.Fatalf("%s must reach the shared resource", adminID)
			}
		})
	}
}

func TestMissingAuthenticationRejectedOnRoleRoute(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached := newRoleProtectedRouter(t, svc, adminroles.RoleTechnicalAdministrator)

	rec := roleRequest(router)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("protected handler must not be reached without an admin cookie")
	}
	assertAuthenticationError(t, rec.Body.String())
}

func TestInvalidAuthenticationRejectedOnRoleRoute(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached := newRoleProtectedRouter(t, svc, adminroles.RoleTechnicalAdministrator)

	for _, value := range []string{"not.a.jwt", "garbage"} {
		rec := roleRequestWithCookie(router, value)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("value %q: expected 401, got %d", value, rec.Code)
		}
		if *reached {
			t.Fatalf("value %q: protected handler must not be reached", value)
		}
		assertAuthenticationError(t, rec.Body.String())
	}
}

func TestExpiredAuthenticationRejectedOnRoleRoute(t *testing.T) {
	svc := token.NewService([]byte(testSecret), -1*time.Minute)
	router, reached := newRoleProtectedRouter(t, svc, adminroles.RoleTechnicalAdministrator)

	rec := roleRequestWithCookie(router, validAdminToken(t, svc, config.Admin1ID))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("protected handler must not be reached with an expired token")
	}
	assertAuthenticationError(t, rec.Body.String())
}

func TestUserTokenRejectedOnRoleRoute(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached := newRoleProtectedRouter(t, svc, adminroles.RoleTechnicalAdministrator)

	jwtValue, err := svc.GenerateAccessToken(uuid.NewString(), 0)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec := roleRequestWithCookie(router, jwtValue)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("protected handler must not be reached with a user token")
	}
	assertAuthenticationError(t, rec.Body.String())
}

func TestStageTokenRejectedOnRoleRoute(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached := newRoleProtectedRouter(t, svc, adminroles.RoleTechnicalAdministrator)

	jwtValue, err := svc.GenerateAdminStageToken(config.Admin1ID)
	if err != nil {
		t.Fatalf("GenerateAdminStageToken: %v", err)
	}

	rec := roleRequestWithCookie(router, jwtValue)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("protected handler must not be reached with a stage token")
	}
	assertAuthenticationError(t, rec.Body.String())
}

func TestUnknownIdentityRejectedOnRoleRoute(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached := newRoleProtectedRouter(t, svc, adminroles.RoleTechnicalAdministrator)

	for _, identity := range []string{"ADMIN_3", "admin", "user-42"} {
		rec := roleRequestWithCookie(router, validAdminToken(t, svc, identity))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("identity %q: expected 401, got %d", identity, rec.Code)
		}
		if *reached {
			t.Fatalf("identity %q: protected handler must not be reached", identity)
		}
		assertAuthenticationError(t, rec.Body.String())
	}
}

func TestAuthorizationMiddlewareFailsClosedWithoutAuthentication(t *testing.T) {
	router, reached := newRoleOnlyRouter(adminroles.RoleTechnicalAdministrator, adminroles.RoleManagementAdministrator)

	rec := roleOnlyRequest(router)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("handler must not be reached when RequireAdminRole is mounted without AdminAuthenticate")
	}
	assertForbiddenError(t, rec.Body.String())
	assertNoAuthorizationLeak(t, rec.Body.String())
}

func permissionRequestWithCookie(router http.Handler, cookieValue string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/permission-protected", nil)
	req.AddCookie(&http.Cookie{Name: auth.AdminAccessTokenCookieName, Value: cookieValue})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func permissionRequest(router http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/permission-protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func permissionOnlyRequest(router http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/permission-only", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestReadPermissionAllowedForBothRoles(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)

	for _, adminID := range []string{config.Admin1ID, config.Admin2ID} {
		t.Run(adminID, func(t *testing.T) {
			router, reached := newPermissionProtectedRouter(t, svc, adminroles.PermissionUsersRead)
			rec := permissionRequestWithCookie(router, validAdminToken(t, svc, adminID))

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rec.Code)
			}
			if !*reached {
				t.Fatalf("%s must reach the read resource", adminID)
			}
		})
	}
}

func TestManagePermissionTechnicalAdminAllowed(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached := newPermissionProtectedRouter(t, svc, adminroles.PermissionUsersManage)

	rec := permissionRequestWithCookie(router, validAdminToken(t, svc, config.Admin1ID))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !*reached {
		t.Fatal("technical admin must reach the manage resource")
	}
}

func TestManagePermissionManagementAdminForbidden(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached := newPermissionProtectedRouter(t, svc, adminroles.PermissionUsersManage)

	rec := permissionRequestWithCookie(router, validAdminToken(t, svc, config.Admin2ID))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("management admin must not reach the manage resource")
	}
	assertForbiddenError(t, rec.Body.String())
	assertNoAuthorizationLeak(t, rec.Body.String())
}

func TestTrainerReadPermissionAllowedForBothRoles(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)

	for _, adminID := range []string{config.Admin1ID, config.Admin2ID} {
		t.Run(adminID, func(t *testing.T) {
			router, reached := newPermissionProtectedRouter(t, svc, adminroles.PermissionTrainersRead)
			rec := permissionRequestWithCookie(router, validAdminToken(t, svc, adminID))

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rec.Code)
			}
			if !*reached {
				t.Fatalf("%s must reach the trainer read resource", adminID)
			}
		})
	}
}

func TestTrainerManagePermissionTechnicalAdminAllowed(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached := newPermissionProtectedRouter(t, svc, adminroles.PermissionTrainersManage)

	rec := permissionRequestWithCookie(router, validAdminToken(t, svc, config.Admin1ID))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !*reached {
		t.Fatal("technical admin must reach the trainer manage resource")
	}
}

func TestTrainerManagePermissionManagementAdminForbidden(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached := newPermissionProtectedRouter(t, svc, adminroles.PermissionTrainersManage)

	rec := permissionRequestWithCookie(router, validAdminToken(t, svc, config.Admin2ID))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("management admin must not reach the trainer manage resource")
	}
	assertForbiddenError(t, rec.Body.String())
	assertNoAuthorizationLeak(t, rec.Body.String())
}

func TestPermissionMiddlewareFailsClosedWithoutAuthentication(t *testing.T) {
	router, reached := newPermissionOnlyRouter(adminroles.PermissionUsersRead, adminroles.PermissionUsersManage)

	rec := permissionOnlyRequest(router)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if *reached {
		t.Fatal("handler must not be reached when RequireAdminPermission is mounted without AdminAuthenticate")
	}
	assertForbiddenError(t, rec.Body.String())
	assertNoAuthorizationLeak(t, rec.Body.String())
}

func TestPermissionMiddlewareRejectsUnknownIdentity(t *testing.T) {
	svc := token.NewService([]byte(testSecret), 15*time.Minute)
	router, reached := newPermissionProtectedRouter(t, svc, adminroles.PermissionUsersRead)

	for _, identity := range []string{"ADMIN_3", "admin", "user-42"} {
		rec := permissionRequestWithCookie(router, validAdminToken(t, svc, identity))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("identity %q: expected 401, got %d", identity, rec.Code)
		}
		if *reached {
			t.Fatalf("identity %q: protected handler must not be reached", identity)
		}
		assertAuthenticationError(t, rec.Body.String())
	}
}

func TestRoleForAdminID(t *testing.T) {
	technical, err := adminroles.RoleForAdminID(config.Admin1ID)
	if err != nil {
		t.Fatalf("ADMIN_1: %v", err)
	}
	if technical != adminroles.RoleTechnicalAdministrator {
		t.Fatalf("ADMIN_1 must be %s, got %s", adminroles.RoleTechnicalAdministrator, technical)
	}

	management, err := adminroles.RoleForAdminID(config.Admin2ID)
	if err != nil {
		t.Fatalf("ADMIN_2: %v", err)
	}
	if management != adminroles.RoleManagementAdministrator {
		t.Fatalf("ADMIN_2 must be %s, got %s", adminroles.RoleManagementAdministrator, management)
	}

	for _, identity := range []string{"ADMIN_3", "admin", "", "user-42"} {
		if _, err := adminroles.RoleForAdminID(identity); err == nil {
			t.Fatalf("identity %q must not resolve to a role", identity)
		}
	}
}

func TestPermissionsTechnicalAdministrator(t *testing.T) {
	for _, granted := range []adminroles.Permission{
		adminroles.PermissionUsers,
		adminroles.PermissionUsersRead,
		adminroles.PermissionUsersManage,
		adminroles.PermissionTrainers,
		adminroles.PermissionTrainersRead,
		adminroles.PermissionTrainersManage,
		adminroles.PermissionStatistics,
		adminroles.PermissionSystem,
		adminroles.PermissionInfrastructure,
		adminroles.PermissionTechnicalConfiguration,
		adminroles.PermissionSecurity,
		adminroles.PermissionDevelopment,
	} {
		if !adminroles.HasPermission(config.Admin1ID, granted) {
			t.Fatalf("technical admin must hold %s", granted)
		}
	}

	for _, denied := range []adminroles.Permission{
		adminroles.PermissionPlans,
		adminroles.PermissionFinance,
		adminroles.PermissionMarketing,
	} {
		if adminroles.HasPermission(config.Admin1ID, denied) {
			t.Fatalf("technical admin must not hold %s", denied)
		}
	}
}

func TestPermissionsManagementAdministrator(t *testing.T) {
	for _, granted := range []adminroles.Permission{
		adminroles.PermissionUsers,
		adminroles.PermissionUsersRead,
		adminroles.PermissionTrainers,
		adminroles.PermissionTrainersRead,
		adminroles.PermissionStatistics,
		adminroles.PermissionPlans,
		adminroles.PermissionFinance,
		adminroles.PermissionMarketing,
	} {
		if !adminroles.HasPermission(config.Admin2ID, granted) {
			t.Fatalf("management admin must hold %s", granted)
		}
	}

	for _, denied := range []adminroles.Permission{
		adminroles.PermissionUsersManage,
		adminroles.PermissionTrainersManage,
		adminroles.PermissionSystem,
		adminroles.PermissionInfrastructure,
		adminroles.PermissionTechnicalConfiguration,
		adminroles.PermissionSecurity,
		adminroles.PermissionDevelopment,
	} {
		if adminroles.HasPermission(config.Admin2ID, denied) {
			t.Fatalf("management admin must not hold %s", denied)
		}
	}
}

func TestHasRole(t *testing.T) {
	if !adminroles.HasRole(config.Admin1ID, adminroles.RoleTechnicalAdministrator) {
		t.Fatal("ADMIN_1 must hold the technical role")
	}
	if !adminroles.HasRole(config.Admin2ID, adminroles.RoleManagementAdministrator) {
		t.Fatal("ADMIN_2 must hold the management role")
	}
	if adminroles.HasRole(config.Admin1ID, adminroles.RoleManagementAdministrator) {
		t.Fatal("ADMIN_1 must not hold the management role")
	}
	if adminroles.HasRole(config.Admin2ID, adminroles.RoleTechnicalAdministrator) {
		t.Fatal("ADMIN_2 must not hold the technical role")
	}
	if adminroles.HasRole("ADMIN_3", adminroles.RoleTechnicalAdministrator) {
		t.Fatal("unknown identity must not hold any role")
	}
}

func TestHasPermissionUnknownIdentity(t *testing.T) {
	for _, permission := range []adminroles.Permission{
		adminroles.PermissionUsers,
		adminroles.PermissionSecurity,
		adminroles.PermissionPlans,
	} {
		if adminroles.HasPermission("ADMIN_3", permission) {
			t.Fatalf("unknown identity must not hold %s", permission)
		}
	}
}
