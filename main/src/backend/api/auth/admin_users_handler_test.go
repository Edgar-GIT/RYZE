package auth_test

import (
	"context"
	"encoding/json"
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
	"ryze/backend/repositories"
	"ryze/backend/services/admin_users"
	"ryze/backend/services/login"
	"ryze/backend/services/password"
	"ryze/backend/services/token"
)

const (
	adminListUsersRoute = "/api/v1/admin/users"
)

// newAdminUsersTestRouter wires the admin user-management endpoints behind the
// real admin authentication and authorization middleware, plus the login and
// /me endpoints needed to exercise session invalidation. It is backed by a
// database transaction so created users are rolled back.
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

	tx := db.Begin()
	t.Cleanup(func() { tx.Rollback() })

	repo := repositories.NewUserRepository(tx)
	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)

	loginSvc := login.NewLoginService(repo, password.Verifier{})
	loginHandler := auth.NewLoginHandler(loginSvc, tokenSvc, testTokenTTL, secure)
	meHandler := auth.NewMeHandler(repo)

	adminUsersSvc := admin_users.NewAdminUserService(repo)
	adminUsersHandler := auth.NewAdminUserHandler(adminUsersSvc)

	router := gin.New()
	router.POST(loginRoute, loginHandler.Login)
	router.GET(meRoute, middleware.Authenticate(tokenSvc, repo), meHandler.GetMe)

	admin := router.Group("/api/v1/admin")
	admin.Use(
		middleware.AdminAuthenticate(tokenSvc),
		middleware.RequireAdminRole(adminroles.RoleTechnicalAdministrator, adminroles.RoleManagementAdministrator),
	)
	admin.GET("/users", adminUsersHandler.ListUsers)
	admin.GET("/users/:id", adminUsersHandler.GetUser)
	admin.PATCH("/users/:id/disable", adminUsersHandler.SoftDeleteUser)

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
	req := httptest.NewRequest(method, path, nil)
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

func TestAdminSoftDeleteUserBothRoles(t *testing.T) {
	router, repo, tokenSvc := newAdminUsersTestRouter(t, true)

	for _, adminID := range []string{config.Admin1ID, config.Admin2ID} {
		t.Run(adminID, func(t *testing.T) {
			user := seedLoginUser(t, repo, uniqueEmail(), "Password123!")
			rec, raw := adminUsersRequest(router, adminToken(t, tokenSvc, adminID), http.MethodPatch, adminListUsersRoute+"/"+user.ID+"/disable")
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"message":"User disabled successfully."`) {
				t.Fatalf("expected disable message, got %s", raw)
			}
		})
	}
}

func TestAdminUsersUnauthenticated(t *testing.T) {
	router, _, _ := newAdminUsersTestRouter(t, true)

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: adminListUsersRoute},
		{method: http.MethodGet, path: adminListUsersRoute + "/" + uuid.NewString()},
		{method: http.MethodPatch, path: adminListUsersRoute + "/" + uuid.NewString() + "/disable"},
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

func TestAdminUsersInternalError(t *testing.T) {
	svc := admin_users.NewAdminUserService(failingLoginRepository{})
	handler := auth.NewAdminUserHandler(svc)

	router := gin.New()
	router.GET("/admin/users", handler.ListUsers)
	router.GET("/admin/users/:id", handler.GetUser)
	router.PATCH("/admin/users/:id/disable", handler.SoftDeleteUser)

	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/admin/users"},
		{method: http.MethodGet, path: "/admin/users/" + uuid.NewString()},
		{method: http.MethodPatch, path: "/admin/users/" + uuid.NewString() + "/disable"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.AddCookie(&http.Cookie{Name: auth.AdminAccessTokenCookieName, Value: cookie})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("%s %s: expected 500, got %d (body: %s)", tc.method, tc.path, rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "repository failure") {
			t.Fatalf("%s %s: internal error details must never be exposed, got %s", tc.method, tc.path, rec.Body.String())
		}
	}
}
