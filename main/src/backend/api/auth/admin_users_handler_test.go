package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ryze/backend/api/auth"
	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/middleware"
	"ryze/backend/middleware/adminroles"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/admin_users"
	"ryze/backend/services/login"
	"ryze/backend/services/password"
	"ryze/backend/services/registration"
	"ryze/backend/services/token"
)

const (
	adminListUsersRoute = "/api/v1/admin/users"
)

// newAdminUsersTestRouter wires the admin user-management endpoints behind the
// real admin authentication and authorization middleware, plus the login and
// /me endpoints needed to exercise session invalidation. It is backed by a
// database transaction so created users are rolled back. Read endpoints run
// under users.read (both roles); lifecycle endpoints run under users.manage
// (Technical Administrator only).
func newAdminUsersTestRouter(t *testing.T, secure bool) (*gin.Engine, repositories.UserRepository, token.Service) {
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
	meHandler := auth.NewMeHandler(repo)

	registrationSvc := registration.NewRegistrationService(repo, password.Hasher{})
	adminUsersSvc := admin_users.NewAdminUserService(repo, registrationSvc, password.Hasher{})
	adminUsersHandler := auth.NewAdminUserHandler(adminUsersSvc)

	router := gin.New()
	router.POST(loginRoute, loginHandler.Login)
	router.GET(meRoute, middleware.Authenticate(tokenSvc, repo), meHandler.GetMe)

	admin := router.Group("/api/v1/admin")
	admin.Use(middleware.AdminAuthenticate(tokenSvc))

	adminRead := admin.Group("")
	adminRead.Use(middleware.RequireAdminPermission(adminroles.PermissionUsersRead))
	adminRead.GET("/users", adminUsersHandler.ListUsers)
	adminRead.GET("/users/:id", adminUsersHandler.GetUser)

	adminMutate := admin.Group("")
	adminMutate.Use(middleware.RequireAdminPermission(adminroles.PermissionUsersManage))
	adminMutate.GET("/users/deleted", adminUsersHandler.ListDeletedUsers)
	adminMutate.POST("/users", adminUsersHandler.CreateUser)
	adminMutate.PATCH("/users/:id", adminUsersHandler.UpdateUser)
	adminMutate.PATCH("/users/:id/disable", adminUsersHandler.SoftDeleteUser)
	adminMutate.POST("/users/:id/reactivate", adminUsersHandler.ReactivateUser)
	adminMutate.POST("/users/:id/password", adminUsersHandler.ResetUserPassword)

	return router, repo, tokenSvc
}

func adminToken(t *testing.T, tokenSvc token.Service, adminID string) string {
	t.Helper()
	jwtValue, err := tokenSvc.GenerateAdminToken(adminID)
	if err != nil {
		t.Fatalf("GenerateAdminToken: %v", err)
	}
	return jwtValue
}

func userToken(t *testing.T, tokenSvc token.Service, userID string, sessionVersion int) string {
	t.Helper()
	jwtValue, err := tokenSvc.GenerateAccessToken(userID, sessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	return jwtValue
}

func adminUsersRequest(router http.Handler, cookieValue, method, path string) (*httptest.ResponseRecorder, string) {
	return adminUsersJSONRequest(router, cookieValue, method, path, "")
}

func adminUsersJSONRequest(router http.Handler, cookieValue, method, path, body string) (*httptest.ResponseRecorder, string) {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookieValue != "" {
		req.AddCookie(&http.Cookie{Name: auth.AdminAccessTokenCookieName, Value: cookieValue})
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec, rec.Body.String()
}

func assertNoSensitiveLeak(t *testing.T, raw string) {
	t.Helper()
	for _, needle := range []string{
		"password_hash",
		"password",
		"deleted_at",
		"session_version",
		"created_at\":null",
		"ryze_access_token",
		"JWT",
	} {
		if strings.Contains(raw, needle) {
			t.Fatalf("response must never expose %q, got %s", needle, raw)
		}
	}
}

// assertNoAuthorizationLeak verifies an authorization failure never reveals the
// required permissions or the authenticated identity.
func assertNoAuthorizationLeak(t *testing.T, raw string) {
	t.Helper()
	for _, sensitive := range []string{
		"users.read",
		"users.manage",
		"TECHNICAL_ADMINISTRATOR",
		"MANAGEMENT_ADMINISTRATOR",
		"ADMIN_1",
		"ADMIN_2",
		"permission",
		"role",
	} {
		if strings.Contains(raw, sensitive) {
			t.Fatalf("authorization failure must not reveal %q, got %s", sensitive, raw)
		}
	}
}

func TestAdminListUsersBothRoles(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	seedLoginUser(t, repo, uniqueEmail(), "Password123!")

	for _, adminID := range []string{config.Admin1ID, config.Admin2ID} {
		t.Run(adminID, func(t *testing.T) {
			rec, raw := adminUsersRequest(router, adminToken(t, tokenSvc, adminID), http.MethodGet, adminListUsersRoute)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"success":true`) {
				t.Fatalf("expected success:true, got %s", raw)
			}
		})
	}
}

func TestAdminGetUserBothRoles(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")

	for _, adminID := range []string{config.Admin1ID, config.Admin2ID} {
		t.Run(adminID, func(t *testing.T) {
			rec, raw := adminUsersRequest(router, adminToken(t, tokenSvc, adminID), http.MethodGet, adminListUsersRoute+"/"+user.ID)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, user.ID) {
				t.Fatalf("expected user id in response, got %s", raw)
			}
		})
	}
}

func TestAdminMutateRoutesAdmin1Only(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	admin1 := adminToken(t, tokenSvc, config.Admin1ID)
	admin2 := adminToken(t, tokenSvc, config.Admin2ID)

	active := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	deleted := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	if err := repo.SoftDelete(context.Background(), deleted.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{
			name:   "list deleted users",
			method: http.MethodGet,
			path:   adminListUsersRoute + "/deleted",
		},
		{
			name:   "create user",
			method: http.MethodPost,
			path:   adminListUsersRoute,
			body:   `{"email":"created@ryze.local","password":"Password123!","first_name":"New","last_name":"User"}`,
		},
		{
			name:   "update user",
			method: http.MethodPatch,
			path:   adminListUsersRoute + "/" + active.ID,
			body:   `{"first_name":"Renamed"}`,
		},
		{
			name:   "disable user",
			method: http.MethodPatch,
			path:   adminListUsersRoute + "/" + active.ID + "/disable",
		},
		{
			name:   "reactivate user",
			method: http.MethodPost,
			path:   adminListUsersRoute + "/" + deleted.ID + "/reactivate",
		},
		{
			name:   "reset password",
			method: http.MethodPost,
			path:   adminListUsersRoute + "/" + active.ID + "/password",
			body:   `{"new_password":"NewPassword123!"}`,
		},
	}

	t.Run("admin1 allowed", func(t *testing.T) {
		for _, tc := range cases {
			rec, raw := adminUsersJSONRequest(router, admin1, tc.method, tc.path, tc.body)
			if rec.Code == http.StatusForbidden {
				t.Fatalf("%s: ADMIN_1 must be allowed, got 403 (body: %s)", tc.name, raw)
			}
		}
	})

	t.Run("admin2 forbidden", func(t *testing.T) {
		for _, tc := range cases {
			rec, raw := adminUsersJSONRequest(router, admin2, tc.method, tc.path, tc.body)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("%s: expected 403 for ADMIN_2, got %d (body: %s)", tc.name, rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"FORBIDDEN"`) {
				t.Fatalf("%s: expected FORBIDDEN, got %s", tc.name, raw)
			}
			assertNoAuthorizationLeak(t, raw)
			assertNoSensitiveLeak(t, raw)
		}
	})
}

func TestAdminSoftDeleteUserAdmin1Only(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	admin1 := adminToken(t, tokenSvc, config.Admin1ID)
	admin2 := adminToken(t, tokenSvc, config.Admin2ID)

	t.Run("admin1", func(t *testing.T) {
		user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
		rec, raw := adminUsersRequest(router, admin1, http.MethodPatch, adminListUsersRoute+"/"+user.ID+"/disable")
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
		}
		if !strings.Contains(raw, `"message":"User disabled successfully."`) {
			t.Fatalf("expected disable message, got %s", raw)
		}
	})

	t.Run("admin2 forbidden", func(t *testing.T) {
		user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
		rec, raw := adminUsersRequest(router, admin2, http.MethodPatch, adminListUsersRoute+"/"+user.ID+"/disable")
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d (body: %s)", rec.Code, raw)
		}
		if !strings.Contains(raw, `"code":"FORBIDDEN"`) {
			t.Fatalf("expected FORBIDDEN, got %s", raw)
		}
		assertNoAuthorizationLeak(t, raw)
		assertNoSensitiveLeak(t, raw)
	})
}

func TestAdminUsersUnauthenticated(t *testing.T) {
	router, _, _ := newAdminUsersTestRouter(t, true)
	id := uuid.NewString()

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: adminListUsersRoute},
		{method: http.MethodGet, path: adminListUsersRoute + "/" + id},
		{method: http.MethodGet, path: adminListUsersRoute + "/deleted"},
		{method: http.MethodPost, path: adminListUsersRoute},
		{method: http.MethodPatch, path: adminListUsersRoute + "/" + id},
		{method: http.MethodPatch, path: adminListUsersRoute + "/" + id + "/disable"},
		{method: http.MethodPost, path: adminListUsersRoute + "/" + id + "/reactivate"},
		{method: http.MethodPost, path: adminListUsersRoute + "/" + id + "/password"},
	} {
		rec, raw := adminUsersRequest(router, "", tc.method, tc.path)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: expected 401, got %d (body: %s)", tc.method, tc.path, rec.Code, raw)
		}
		if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
			t.Fatalf("%s %s: expected AUTHENTICATION_REQUIRED, got %s", tc.method, tc.path, raw)
		}
	}
}

func TestAdminUsersRegularUserRejected(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")

	rec, raw := adminUsersRequest(router, userToken(t, tokenSvc, user.ID, user.SessionVersion), http.MethodGet, adminListUsersRoute)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	assertNoSensitiveLeak(t, raw)
}

func TestAdminUsersStageTokenRejected(t *testing.T) {
	router, _, tokenSvc := newAdminUsersTestRouter(t, true)

	stageToken, err := tokenSvc.GenerateAdminStageToken(config.Admin1ID)
	if err != nil {
		t.Fatalf("GenerateAdminStageToken: %v", err)
	}

	rec, raw := adminUsersRequest(router, stageToken, http.MethodGet, adminListUsersRoute)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
}

func TestAdminUsersInvalidTokenRejected(t *testing.T) {
	router, _, _ := newAdminUsersTestRouter(t, true)

	for _, value := range []string{"garbage", "not.a.jwt", ""} {
		rec, raw := adminUsersRequest(router, value, http.MethodGet, adminListUsersRoute)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("value %q: expected 401, got %d (body: %s)", value, rec.Code, raw)
		}
	}
}

func TestAdminListUsersReturnsActiveOnly(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	active1 := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	active2 := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	deleted := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	if err := repo.SoftDelete(context.Background(), deleted.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	rec, raw := adminUsersRequest(router, adminToken(t, tokenSvc, config.Admin1ID), http.MethodGet, adminListUsersRoute)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, active1.ID) || !strings.Contains(raw, active2.ID) {
		t.Fatalf("expected active users in response, got %s", raw)
	}
	if strings.Contains(raw, deleted.ID) {
		t.Fatalf("soft-deleted user must not be listed, got %s", raw)
	}
}

func TestAdminListUsersPagination(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	baselineTotal := listTotal(t, router, cookie)
	created := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		created = append(created, seedLoginUser(t, repo, uniqueEmail(), "Password123!").ID)
	}
	expectedTotal := baselineTotal + 3

	pageIDs := make([]string, 0, 3)
	for page := 1; page <= 3; page++ {
		rec, raw := adminUsersRequest(router, cookie, http.MethodGet, adminListUsersRoute+"?page="+strconv.Itoa(page)+"&limit=1")
		if rec.Code != http.StatusOK {
			t.Fatalf("page %d: expected 200, got %d (body: %s)", page, rec.Code, raw)
		}
		var payload struct {
			Data struct {
				Users []struct {
					ID string `json:"id"`
				} `json:"users"`
				Pagination struct {
					Total      int `json:"total"`
					TotalPages int `json:"total_pages"`
				} `json:"pagination"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			t.Fatalf("page %d: invalid JSON: %v (body: %s)", page, err, raw)
		}
		if payload.Data.Pagination.Total != expectedTotal {
			t.Fatalf("page %d: expected total %d, got %d", page, expectedTotal, payload.Data.Pagination.Total)
		}
		if payload.Data.Pagination.TotalPages != expectedTotal {
			t.Fatalf("page %d: expected total_pages %d, got %d", page, expectedTotal, payload.Data.Pagination.TotalPages)
		}
		if len(payload.Data.Users) != 1 {
			t.Fatalf("page %d: expected 1 user, got %d", page, len(payload.Data.Users))
		}
		pageIDs = append(pageIDs, payload.Data.Users[0].ID)
	}

	if len(pageIDs) != 3 {
		t.Fatalf("expected 3 page ids, got %d", len(pageIDs))
	}
	for _, id := range created {
		found := false
		for _, pid := range pageIDs {
			if pid == id {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("created user %q was not returned across pages %v", id, pageIDs)
		}
	}
}

// listTotal fetches the total user count reported by the list endpoint.
func listTotal(t *testing.T, router http.Handler, cookie string) int {
	t.Helper()
	rec, raw := adminUsersRequest(router, cookie, http.MethodGet, adminListUsersRoute+"?page=1&limit=1")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	var payload struct {
		Data struct {
			Pagination struct {
				Total int `json:"total"`
			} `json:"pagination"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("invalid JSON: %v (body: %s)", err, raw)
	}
	return payload.Data.Pagination.Total
}

func TestAdminListUsersInvalidPagination(t *testing.T) {
	router, _, tokenSvc := newAdminUsersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	for _, query := range []string{
		"?page=0&limit=20",
		"?page=-1&limit=20",
		"?page=1&limit=0",
		"?page=1&limit=-3",
		"?page=abc&limit=20",
		"?page=1&limit=abc",
	} {
		rec, raw := adminUsersRequest(router, cookie, http.MethodGet, adminListUsersRoute+query)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d (body: %s)", query, rec.Code, raw)
		}
		if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
			t.Fatalf("%s: expected VALIDATION_ERROR, got %s", query, raw)
		}
	}
}

func TestAdminListUsersClampsOversizedLimit(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	seedLoginUser(t, repo, uniqueEmail(), "Password123!")

	rec, raw := adminUsersRequest(router, adminToken(t, tokenSvc, config.Admin1ID), http.MethodGet, adminListUsersRoute+"?page=1&limit=9999")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	var payload struct {
		Data struct {
			Pagination struct {
				Limit int `json:"limit"`
			} `json:"pagination"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if payload.Data.Pagination.Limit != admin_users.MaxPageSize {
		t.Fatalf("expected limit %d, got %d", admin_users.MaxPageSize, payload.Data.Pagination.Limit)
	}
}

func TestAdminListUsersNoSensitiveFields(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	seedLoginUser(t, repo, uniqueEmail(), "Password123!")

	rec, raw := adminUsersRequest(router, adminToken(t, tokenSvc, config.Admin1ID), http.MethodGet, adminListUsersRoute)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	assertNoSensitiveLeak(t, raw)
}

func TestAdminGetUserSafeRepresentation(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")

	rec, raw := adminUsersRequest(router, adminToken(t, tokenSvc, config.Admin1ID), http.MethodGet, adminListUsersRoute+"/"+user.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	for _, field := range []string{`"id"`, `"email"`, `"first_name"`, `"last_name"`, `"created_at"`, `"updated_at"`} {
		if !strings.Contains(raw, field) {
			t.Fatalf("expected field %s in response, got %s", field, raw)
		}
	}
	assertNoSensitiveLeak(t, raw)
}

func TestAdminGetUserUnknownUUID(t *testing.T) {
	router, _, tokenSvc := newAdminUsersTestRouter(t, true)

	rec, raw := adminUsersRequest(router, adminToken(t, tokenSvc, config.Admin1ID), http.MethodGet, adminListUsersRoute+"/"+uuid.NewString())
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"USER_NOT_FOUND"`) {
		t.Fatalf("expected USER_NOT_FOUND, got %s", raw)
	}
}

func TestAdminGetUserMalformedID(t *testing.T) {
	router, _, tokenSvc := newAdminUsersTestRouter(t, true)

	rec, raw := adminUsersRequest(router, adminToken(t, tokenSvc, config.Admin1ID), http.MethodGet, adminListUsersRoute+"/not-a-uuid")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
}

func TestAdminGetUserSoftDeleted(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	if err := repo.SoftDelete(context.Background(), user.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	rec, raw := adminUsersRequest(router, adminToken(t, tokenSvc, config.Admin1ID), http.MethodGet, adminListUsersRoute+"/"+user.ID)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
}

func TestAdminGetUserNoSensitiveLeak(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")

	rec, raw := adminUsersRequest(router, adminToken(t, tokenSvc, config.Admin1ID), http.MethodGet, adminListUsersRoute+"/"+user.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	assertNoSensitiveLeak(t, raw)
}

func TestAdminSoftDeleteRevokesSessionAndPreservesRow(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	cookie := loginForCookie(t, router, user.Email, "Password123!")

	if rec, _, _ := requestMe(router, cookie); rec.Code != http.StatusOK {
		t.Fatal("expected /me 200 before the soft delete")
	}

	rec, raw := adminUsersRequest(router, adminToken(t, tokenSvc, config.Admin1ID), http.MethodPatch, adminListUsersRoute+"/"+user.ID+"/disable")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	if rec, _, _ := requestMe(router, cookie); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected /me 401 after the soft delete, got %d", rec.Code)
	}

	if rec, _, _ := attemptLogin(router, loginBody(user.Email, "Password123!")); rec.Code != http.StatusUnauthorized {
		t.Fatalf("login for a disabled account: expected 401, got %d", rec.Code)
	}

	stored, err := repo.FindByEmailIncludingDeleted(context.Background(), user.Email)
	if err != nil {
		t.Fatalf("FindByEmailIncludingDeleted: %v", err)
	}
	if !stored.DeletedAt.Valid {
		t.Fatal("deleted_at must be populated after the soft delete")
	}
	if stored.SessionVersion != user.SessionVersion+1 {
		t.Fatalf("expected session version %d, got %d", user.SessionVersion+1, stored.SessionVersion)
	}
	if stored.ID != user.ID {
		t.Fatalf("id must be preserved: expected %q, got %q", user.ID, stored.ID)
	}
	if stored.Email != user.Email {
		t.Fatalf("email must be preserved: expected %q, got %q", user.Email, stored.Email)
	}
	if !stored.CreatedAt.Equal(user.CreatedAt) {
		t.Fatal("created_at must be preserved after the soft delete")
	}
}

func TestAdminSoftDeleteMissingOrAlreadyDeleted(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	rec, raw := adminUsersRequest(router, cookie, http.MethodPatch, adminListUsersRoute+"/"+uuid.NewString()+"/disable")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing user: expected 404, got %d (body: %s)", rec.Code, raw)
	}

	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	if err := repo.SoftDelete(context.Background(), user.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	rec, raw = adminUsersRequest(router, cookie, http.MethodPatch, adminListUsersRoute+"/"+user.ID+"/disable")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("already-deleted user: expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"USER_NOT_FOUND"`) {
		t.Fatalf("expected USER_NOT_FOUND, got %s", raw)
	}
}

func TestAdminSoftDeleteNoSensitiveLeak(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")

	rec, raw := adminUsersRequest(router, adminToken(t, tokenSvc, config.Admin1ID), http.MethodPatch, adminListUsersRoute+"/"+user.ID+"/disable")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	assertNoSensitiveLeak(t, raw)
}

func TestAdminListDeletedUsersReturnsDeletedOnly(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	active := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	deleted := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	if err := repo.SoftDelete(context.Background(), deleted.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	rec, raw := adminUsersRequest(router, cookie, http.MethodGet, adminListUsersRoute+"/deleted")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"message":"Deleted users retrieved successfully."`) {
		t.Fatalf("expected deleted-list message, got %s", raw)
	}
	if !strings.Contains(raw, deleted.ID) {
		t.Fatalf("expected soft-deleted user in deleted list, got %s", raw)
	}
	if strings.Contains(raw, active.ID) {
		t.Fatalf("active users must not appear in the deleted list, got %s", raw)
	}
	assertNoSensitiveLeak(t, raw)
}

func TestAdminCreateUserSuccessAndCanLogin(t *testing.T) {
	router, _, tokenSvc := newAdminUsersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	email := uniqueEmail()

	rec, raw := adminUsersJSONRequest(router, cookie, http.MethodPost, adminListUsersRoute,
		`{"email":"`+email+`","password":"Password123!","first_name":"New","last_name":"User"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"message":"User created successfully."`) {
		t.Fatalf("expected create message, got %s", raw)
	}
	if !strings.Contains(raw, email) {
		t.Fatalf("expected created email in response, got %s", raw)
	}
	assertNoSensitiveLeak(t, raw)

	rec, _, _ = attemptLogin(router, loginBody(email, "Password123!"))
	if rec.Code != http.StatusOK {
		t.Fatalf("created user must be able to log in, got %d", rec.Code)
	}
}

func TestAdminCreateUserDuplicateEmail(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")

	rec, raw := adminUsersJSONRequest(router, cookie, http.MethodPost, adminListUsersRoute,
		`{"email":"`+user.Email+`","password":"Password123!","first_name":"Dup","last_name":"User"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"EMAIL_ALREADY_REGISTERED"`) {
		t.Fatalf("expected EMAIL_ALREADY_REGISTERED, got %s", raw)
	}
}

func TestAdminCreateUserInvalidInput(t *testing.T) {
	router, _, tokenSvc := newAdminUsersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	for _, body := range []string{
		`{"email":"missing-pass@ryze.local","password":"","first_name":"A","last_name":"B"}`,
		`{"email":"not-an-email","password":"Password123!","first_name":"A","last_name":"B"}`,
		`{"email":"missing-fields@ryze.local","password":"Password123!"}`,
		`{"email":"x","password":"Password123!","first_name":"","last_name":""}`,
	} {
		rec, raw := adminUsersJSONRequest(router, cookie, http.MethodPost, adminListUsersRoute, body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("body %s: expected 400, got %d (body: %s)", body, rec.Code, raw)
		}
		if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
			t.Fatalf("body %s: expected VALIDATION_ERROR, got %s", body, raw)
		}
	}
}

func TestAdminCreateUserReactivatesSoftDeletedEmail(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	email := uniqueEmail()

	user := seedLoginUser(t, repo, email, "Password123!")
	if err := repo.SoftDelete(context.Background(), user.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	rec, raw := adminUsersJSONRequest(router, cookie, http.MethodPost, adminListUsersRoute,
		`{"email":"`+email+`","password":"AnotherPass456!","first_name":"Reborn","last_name":"User"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}

	stored, err := repo.FindByEmailIncludingDeleted(context.Background(), email)
	if err != nil {
		t.Fatalf("FindByEmailIncludingDeleted: %v", err)
	}
	if stored.ID != user.ID {
		t.Fatalf("reactivated user must keep the original id: expected %q, got %q", user.ID, stored.ID)
	}
	if !stored.CreatedAt.Equal(user.CreatedAt) {
		t.Fatal("created_at must be preserved across reactivation")
	}
	if stored.DeletedAt.Valid {
		t.Fatal("deleted_at must be cleared after reactivation")
	}

	if rec, _, _ := attemptLogin(router, loginBody(email, "AnotherPass456!")); rec.Code != http.StatusOK {
		t.Fatalf("reactivated user must log in with the new password, got %d", rec.Code)
	}
	if rec, _, _ := attemptLogin(router, loginBody(email, "Password123!")); rec.Code != http.StatusUnauthorized {
		t.Fatalf("the old password must fail after reactivation, got %d", rec.Code)
	}
}

func TestAdminUpdateUserSuccess(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")

	rec, raw := adminUsersJSONRequest(router, cookie, http.MethodPatch, adminListUsersRoute+"/"+user.ID,
		`{"first_name":"Renamed","last_name":"Surname"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"first_name":"Renamed"`) {
		t.Fatalf("expected updated first name in response, got %s", raw)
	}
	assertNoSensitiveLeak(t, raw)
}

func TestAdminUpdateUserDuplicateEmail(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	target := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	owner := seedLoginUser(t, repo, uniqueEmail(), "Password123!")

	rec, raw := adminUsersJSONRequest(router, cookie, http.MethodPatch, adminListUsersRoute+"/"+target.ID,
		`{"email":"`+owner.Email+`"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"EMAIL_ALREADY_REGISTERED"`) {
		t.Fatalf("expected EMAIL_ALREADY_REGISTERED, got %s", raw)
	}
}

func TestAdminUpdateUserNotFound(t *testing.T) {
	router, _, tokenSvc := newAdminUsersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	rec, raw := adminUsersJSONRequest(router, cookie, http.MethodPatch, adminListUsersRoute+"/"+uuid.NewString(),
		`{"first_name":"Nobody"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"USER_NOT_FOUND"`) {
		t.Fatalf("expected USER_NOT_FOUND, got %s", raw)
	}
}

func TestAdminUpdateUserInvalidInput(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")

	for _, body := range []string{
		`{}`,
		`{"email":"not-an-email"}`,
		`{"first_name":""}`,
	} {
		rec, raw := adminUsersJSONRequest(router, cookie, http.MethodPatch, adminListUsersRoute+"/"+user.ID, body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("body %s: expected 400, got %d (body: %s)", body, rec.Code, raw)
		}
		if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
			t.Fatalf("body %s: expected VALIDATION_ERROR, got %s", body, raw)
		}
	}
}

func TestAdminReactivateUserRestoresAccountAndKeepsSessionsInvalid(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	oldCookie := loginForCookie(t, router, user.Email, "Password123!")

	rec, raw := adminUsersRequest(router, cookie, http.MethodPatch, adminListUsersRoute+"/"+user.ID+"/disable")
	if rec.Code != http.StatusOK {
		t.Fatalf("disable: expected 200, got %d (body: %s)", rec.Code, raw)
	}

	rec, raw = adminUsersJSONRequest(router, cookie, http.MethodPost, adminListUsersRoute+"/"+user.ID+"/reactivate", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"message":"User reactivated successfully."`) {
		t.Fatalf("expected reactivate message, got %s", raw)
	}

	stored, err := repo.FindByEmailIncludingDeleted(context.Background(), user.Email)
	if err != nil {
		t.Fatalf("FindByEmailIncludingDeleted: %v", err)
	}
	if stored.ID != user.ID || stored.DeletedAt.Valid {
		t.Fatalf("expected the same user active again, got %+v", stored)
	}

	if rec, _, _ := requestMe(router, oldCookie); rec.Code != http.StatusUnauthorized {
		t.Fatalf("pre-deletion session must stay invalid after reactivation, got %d", rec.Code)
	}
	if rec, _, _ := attemptLogin(router, loginBody(user.Email, "Password123!")); rec.Code != http.StatusOK {
		t.Fatalf("reactivated user must log in again, got %d", rec.Code)
	}
}

func TestAdminReactivateUserAlreadyActive(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")

	rec, raw := adminUsersJSONRequest(router, cookie, http.MethodPost, adminListUsersRoute+"/"+user.ID+"/reactivate", "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"USER_ALREADY_ACTIVE"`) {
		t.Fatalf("expected USER_ALREADY_ACTIVE, got %s", raw)
	}
}

func TestAdminReactivateUserNotFound(t *testing.T) {
	router, _, tokenSvc := newAdminUsersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	rec, raw := adminUsersJSONRequest(router, cookie, http.MethodPost, adminListUsersRoute+"/"+uuid.NewString()+"/reactivate", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"USER_NOT_FOUND"`) {
		t.Fatalf("expected USER_NOT_FOUND, got %s", raw)
	}
}

func TestAdminResetUserPasswordInvalidatesSessions(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
	oldCookie := loginForCookie(t, router, user.Email, "Password123!")

	rec, raw := adminUsersJSONRequest(router, cookie, http.MethodPost, adminListUsersRoute+"/"+user.ID+"/password",
		`{"new_password":"NewPassword456!"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"message":"Password reset successfully."`) {
		t.Fatalf("expected reset message, got %s", raw)
	}
	assertNoSensitiveLeak(t, raw)

	if rec, _, _ := requestMe(router, oldCookie); rec.Code != http.StatusUnauthorized {
		t.Fatalf("old session must be invalid after the reset, got %d", rec.Code)
	}
	if rec, _, _ := attemptLogin(router, loginBody(user.Email, "Password123!")); rec.Code != http.StatusUnauthorized {
		t.Fatalf("old password must fail after the reset, got %d", rec.Code)
	}
	newCookie := loginForCookie(t, router, user.Email, "NewPassword456!")
	if rec, _, _ := requestMe(router, newCookie); rec.Code != http.StatusOK {
		t.Fatal("expected /me 200 with the new password")
	}
}

func TestAdminResetUserPasswordInvalidInput(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")

	rec, raw := adminUsersJSONRequest(router, cookie, http.MethodPost, adminListUsersRoute+"/"+user.ID+"/password",
		`{"new_password":""}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestAdminResetUserPasswordNotFound(t *testing.T) {
	router, _, tokenSvc := newAdminUsersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	rec, raw := adminUsersJSONRequest(router, cookie, http.MethodPost, adminListUsersRoute+"/"+uuid.NewString()+"/password",
		`{"new_password":"NewPassword456!"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"USER_NOT_FOUND"`) {
		t.Fatalf("expected USER_NOT_FOUND, got %s", raw)
	}
}

// failingRegistrar and failingHasher simulate unexpected dependency failures for
// the internal-error test.
type failingRegistrar struct{}

func (failingRegistrar) Register(_ context.Context, _ registration.RegisterInput) (*models.User, error) {
	return nil, errors.New("registrar failure")
}

type failingHasher struct{}

func (failingHasher) HashPassword(_ string) (string, error) {
	return "", errors.New("hasher failure")
}

func TestAdminUsersInternalError(t *testing.T) {
	svc := admin_users.NewAdminUserService(failingLoginRepository{}, failingRegistrar{}, failingHasher{})
	handler := auth.NewAdminUserHandler(svc)

	router := gin.New()
	router.GET("/admin/users", handler.ListUsers)
	router.GET("/admin/users/deleted", handler.ListDeletedUsers)
	router.GET("/admin/users/:id", handler.GetUser)
	router.POST("/admin/users", handler.CreateUser)
	router.PATCH("/admin/users/:id", handler.UpdateUser)
	router.PATCH("/admin/users/:id/disable", handler.SoftDeleteUser)
	router.POST("/admin/users/:id/reactivate", handler.ReactivateUser)
	router.POST("/admin/users/:id/password", handler.ResetUserPassword)

	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	id := uuid.NewString()
	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/admin/users"},
		{method: http.MethodGet, path: "/admin/users/deleted"},
		{method: http.MethodGet, path: "/admin/users/" + id},
		{
			method: http.MethodPost,
			path:   "/admin/users",
			body:   `{"email":"x@ryze.local","password":"Password123!","first_name":"A","last_name":"B"}`,
		},
		{method: http.MethodPatch, path: "/admin/users/" + id, body: `{"first_name":"Renamed"}`},
		{method: http.MethodPatch, path: "/admin/users/" + id + "/disable"},
		{method: http.MethodPost, path: "/admin/users/" + id + "/reactivate"},
		{method: http.MethodPost, path: "/admin/users/" + id + "/password", body: `{"new_password":"NewPassword456!"}`},
	} {
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
		if tc.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		req.AddCookie(&http.Cookie{Name: auth.AdminAccessTokenCookieName, Value: cookie})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("%s %s: expected 500, got %d (body: %s)", tc.method, tc.path, rec.Code, rec.Body.String())
		}
		for _, leaked := range []string{"repository failure", "registrar failure", "hasher failure"} {
			if strings.Contains(rec.Body.String(), leaked) {
				t.Fatalf("%s %s: internal error details must never be exposed, got %s", tc.method, tc.path, rec.Body.String())
			}
		}
	}
}
