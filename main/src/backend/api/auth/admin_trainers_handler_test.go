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
	"ryze/backend/services/admin_trainers"
	"ryze/backend/services/admin_users"
	"ryze/backend/services/login"
	"ryze/backend/services/password"
	"ryze/backend/services/registration"
	"ryze/backend/services/token"
)

const adminTrainersRoute = "/api/v1/admin/trainers"

// newAdminTrainersTestRouter wires the admin trainer-management endpoints behind
// the real admin authentication and authorization middleware, plus the login and
// /me endpoints needed to exercise the user lifecycle around a trainer. It is
// backed by a database transaction so created records are rolled back. Read
// endpoints run under trainers.read (both roles); lifecycle endpoints run under
// trainers.manage (Technical Administrator only).
func newAdminTrainersTestRouter(t *testing.T, secure bool) (*gin.Engine, repositories.UserRepository, repositories.TrainerRepository, token.Service) {
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

	userRepo := repositories.NewUserRepository(tx)
	trainerRepo := repositories.NewTrainerRepository(tx)
	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)

	loginSvc := login.NewLoginService(userRepo, password.Verifier{})
	loginHandler := auth.NewLoginHandler(loginSvc, tokenSvc, testTokenTTL, secure)
	meHandler := auth.NewMeHandler(userRepo)

	registrationSvc := registration.NewRegistrationService(userRepo, password.Hasher{})
	adminUsersSvc := admin_users.NewAdminUserService(userRepo, registrationSvc, password.Hasher{})
	adminTrainersSvc := admin_trainers.NewAdminTrainerService(trainerRepo, userRepo, registrationSvc, adminUsersSvc)
	adminTrainersHandler := auth.NewAdminTrainerHandler(adminTrainersSvc)

	router := gin.New()
	router.POST(loginRoute, loginHandler.Login)
	router.GET(meRoute, middleware.Authenticate(tokenSvc, userRepo), meHandler.GetMe)

	admin := router.Group("/api/v1/admin")
	admin.Use(middleware.AdminAuthenticate(tokenSvc))

	adminRead := admin.Group("")
	adminRead.Use(middleware.RequireAdminPermission(adminroles.PermissionTrainersRead))
	adminRead.GET("/trainers", adminTrainersHandler.ListTrainers)
	adminRead.GET("/trainers/:id", adminTrainersHandler.GetTrainer)

	adminMutate := admin.Group("")
	adminMutate.Use(middleware.RequireAdminPermission(adminroles.PermissionTrainersManage))
	adminMutate.GET("/trainers/deleted", adminTrainersHandler.ListDeletedTrainers)
	adminMutate.POST("/trainers", adminTrainersHandler.CreateTrainer)
	adminMutate.PATCH("/trainers/:id", adminTrainersHandler.UpdateTrainer)
	adminMutate.PATCH("/trainers/:id/disable", adminTrainersHandler.SoftDeleteTrainer)
	adminMutate.POST("/trainers/:id/reactivate", adminTrainersHandler.ReactivateTrainer)

	return router, userRepo, trainerRepo, tokenSvc
}

// seedTrainer creates an active user and links an active trainer profile to it.
func seedTrainer(t *testing.T, userRepo repositories.UserRepository, trainerRepo repositories.TrainerRepository, email string) *models.Trainer {
	t.Helper()

	user := seedLoginUser(t, userRepo, email, "Password123!")
	trainer := &models.Trainer{UserID: user.ID}
	if err := trainerRepo.Create(context.Background(), trainer); err != nil {
		t.Fatalf("seed trainer: %v", err)
	}
	return trainer
}

func adminTrainerRequest(router http.Handler, cookieValue, method, path string) (*httptest.ResponseRecorder, string) {
	return adminTrainerJSONRequest(router, cookieValue, method, path, "")
}

func adminTrainerJSONRequest(router http.Handler, cookieValue, method, path, body string) (*httptest.ResponseRecorder, string) {
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

func TestAdminListTrainersBothRoles(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	seedTrainer(t, userRepo, trainerRepo, uniqueEmail())

	for _, adminID := range []string{config.Admin1ID, config.Admin2ID} {
		t.Run(adminID, func(t *testing.T) {
			rec, raw := adminTrainerRequest(router, adminToken(t, tokenSvc, adminID), http.MethodGet, adminTrainersRoute)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"success":true`) {
				t.Fatalf("expected success:true, got %s", raw)
			}
		})
	}
}

func TestAdminGetTrainerBothRoles(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	trainer := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())

	for _, adminID := range []string{config.Admin1ID, config.Admin2ID} {
		t.Run(adminID, func(t *testing.T) {
			rec, raw := adminTrainerRequest(router, adminToken(t, tokenSvc, adminID), http.MethodGet, adminTrainersRoute+"/"+trainer.ID)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, trainer.ID) {
				t.Fatalf("expected trainer id in response, got %s", raw)
			}
		})
	}
}

func TestAdminTrainerMutateRoutesAdmin1Only(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	admin1 := adminToken(t, tokenSvc, config.Admin1ID)
	admin2 := adminToken(t, tokenSvc, config.Admin2ID)

	active := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())
	deleted := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())
	if err := trainerRepo.SoftDelete(context.Background(), deleted.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{
			name:   "list deleted trainers",
			method: http.MethodGet,
			path:   adminTrainersRoute + "/deleted",
		},
		{
			name:   "create trainer",
			method: http.MethodPost,
			path:   adminTrainersRoute,
			body:   `{"email":"trainer-created@ryze.local","password":"Password123!","first_name":"New","last_name":"Trainer"}`,
		},
		{
			name:   "update trainer",
			method: http.MethodPatch,
			path:   adminTrainersRoute + "/" + active.ID,
			body:   `{"first_name":"Renamed"}`,
		},
		{
			name:   "disable trainer",
			method: http.MethodPatch,
			path:   adminTrainersRoute + "/" + active.ID + "/disable",
		},
		{
			name:   "reactivate trainer",
			method: http.MethodPost,
			path:   adminTrainersRoute + "/" + deleted.ID + "/reactivate",
		},
	}

	t.Run("admin1 allowed", func(t *testing.T) {
		for _, tc := range cases {
			rec, raw := adminTrainerJSONRequest(router, admin1, tc.method, tc.path, tc.body)
			if rec.Code == http.StatusForbidden {
				t.Fatalf("%s: ADMIN_1 must be allowed, got 403 (body: %s)", tc.name, raw)
			}
		}
	})

	t.Run("admin2 forbidden", func(t *testing.T) {
		for _, tc := range cases {
			rec, raw := adminTrainerJSONRequest(router, admin2, tc.method, tc.path, tc.body)
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

func TestAdminTrainersUnauthenticated(t *testing.T) {
	router, _, _, _ := newAdminTrainersTestRouter(t, true)
	id := uuid.NewString()

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: adminTrainersRoute},
		{method: http.MethodGet, path: adminTrainersRoute + "/" + id},
		{method: http.MethodGet, path: adminTrainersRoute + "/deleted"},
		{method: http.MethodPost, path: adminTrainersRoute},
		{method: http.MethodPatch, path: adminTrainersRoute + "/" + id},
		{method: http.MethodPatch, path: adminTrainersRoute + "/" + id + "/disable"},
		{method: http.MethodPost, path: adminTrainersRoute + "/" + id + "/reactivate"},
	} {
		rec, raw := adminTrainerRequest(router, "", tc.method, tc.path)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: expected 401, got %d (body: %s)", tc.method, tc.path, rec.Code, raw)
		}
		if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
			t.Fatalf("%s %s: expected AUTHENTICATION_REQUIRED, got %s", tc.method, tc.path, raw)
		}
	}
}

func TestAdminTrainersRegularUserRejected(t *testing.T) {
	router, userRepo, _, tokenSvc := newAdminTrainersTestRouter(t, true)
	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	rec, raw := adminTrainerRequest(router, userToken(t, tokenSvc, user.ID, user.SessionVersion), http.MethodGet, adminTrainersRoute)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	assertNoSensitiveLeak(t, raw)
}

func TestAdminTrainersStageTokenRejected(t *testing.T) {
	router, _, _, tokenSvc := newAdminTrainersTestRouter(t, true)

	stageToken, err := tokenSvc.GenerateAdminStageToken(config.Admin1ID)
	if err != nil {
		t.Fatalf("GenerateAdminStageToken: %v", err)
	}

	rec, raw := adminTrainerRequest(router, stageToken, http.MethodGet, adminTrainersRoute)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
}

func TestAdminTrainersInvalidTokenRejected(t *testing.T) {
	router, _, _, _ := newAdminTrainersTestRouter(t, true)

	for _, value := range []string{"garbage", "not.a.jwt", ""} {
		rec, raw := adminTrainerRequest(router, value, http.MethodGet, adminTrainersRoute)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("value %q: expected 401, got %d (body: %s)", value, rec.Code, raw)
		}
	}
}

func TestAdminListTrainersReturnsActiveOnly(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	active1 := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())
	active2 := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())
	deleted := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())
	if err := trainerRepo.SoftDelete(context.Background(), deleted.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	rec, raw := adminTrainerRequest(router, adminToken(t, tokenSvc, config.Admin1ID), http.MethodGet, adminTrainersRoute)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, active1.ID) || !strings.Contains(raw, active2.ID) {
		t.Fatalf("expected active trainers in response, got %s", raw)
	}
	if strings.Contains(raw, deleted.ID) {
		t.Fatalf("soft-deleted trainer must not be listed, got %s", raw)
	}
}

func TestAdminListTrainersPagination(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	baselineTotal := trainerListTotal(t, router, cookie)
	created := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		created = append(created, seedTrainer(t, userRepo, trainerRepo, uniqueEmail()).ID)
	}
	expectedTotal := baselineTotal + 3

	pageIDs := make([]string, 0, 3)
	for page := 1; page <= 3; page++ {
		rec, raw := adminTrainerRequest(router, cookie, http.MethodGet, adminTrainersRoute+"?page="+strconv.Itoa(page)+"&limit=1")
		if rec.Code != http.StatusOK {
			t.Fatalf("page %d: expected 200, got %d (body: %s)", page, rec.Code, raw)
		}
		var payload struct {
			Data struct {
				Trainers []struct {
					ID string `json:"id"`
				} `json:"trainers"`
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
		if len(payload.Data.Trainers) != 1 {
			t.Fatalf("page %d: expected 1 trainer, got %d", page, len(payload.Data.Trainers))
		}
		pageIDs = append(pageIDs, payload.Data.Trainers[0].ID)
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
			t.Fatalf("created trainer %q was not returned across pages %v", id, pageIDs)
		}
	}
}

// trainerListTotal fetches the total trainer count reported by the list endpoint.
func trainerListTotal(t *testing.T, router http.Handler, cookie string) int {
	t.Helper()
	rec, raw := adminTrainerRequest(router, cookie, http.MethodGet, adminTrainersRoute+"?page=1&limit=1")
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

func TestAdminListTrainersInvalidPagination(t *testing.T) {
	router, _, _, tokenSvc := newAdminTrainersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	for _, query := range []string{
		"?page=0&limit=20",
		"?page=-1&limit=20",
		"?page=1&limit=0",
		"?page=1&limit=-3",
		"?page=abc&limit=20",
		"?page=1&limit=abc",
	} {
		rec, raw := adminTrainerRequest(router, cookie, http.MethodGet, adminTrainersRoute+query)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d (body: %s)", query, rec.Code, raw)
		}
		if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
			t.Fatalf("%s: expected VALIDATION_ERROR, got %s", query, raw)
		}
	}
}

func TestAdminListTrainersClampsOversizedLimit(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	seedTrainer(t, userRepo, trainerRepo, uniqueEmail())

	rec, raw := adminTrainerRequest(router, adminToken(t, tokenSvc, config.Admin1ID), http.MethodGet, adminTrainersRoute+"?page=1&limit=9999")
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
	if payload.Data.Pagination.Limit != admin_trainers.MaxPageSize {
		t.Fatalf("expected limit %d, got %d", admin_trainers.MaxPageSize, payload.Data.Pagination.Limit)
	}
}

func TestAdminListTrainersNoSensitiveFields(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	seedTrainer(t, userRepo, trainerRepo, uniqueEmail())

	rec, raw := adminTrainerRequest(router, adminToken(t, tokenSvc, config.Admin1ID), http.MethodGet, adminTrainersRoute)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	assertNoSensitiveLeak(t, raw)
}

func TestAdminListDeletedTrainersReturnsDeletedOnly(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	active := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())
	deleted := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())
	if err := trainerRepo.SoftDelete(context.Background(), deleted.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	rec, raw := adminTrainerRequest(router, cookie, http.MethodGet, adminTrainersRoute+"/deleted")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"message":"Deleted trainers retrieved successfully."`) {
		t.Fatalf("expected deleted-list message, got %s", raw)
	}
	if !strings.Contains(raw, deleted.ID) {
		t.Fatalf("expected soft-deleted trainer in deleted list, got %s", raw)
	}
	if strings.Contains(raw, active.ID) {
		t.Fatalf("active trainers must not appear in the deleted list, got %s", raw)
	}
	assertNoSensitiveLeak(t, raw)
}

func TestAdminGetTrainerSafeRepresentation(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	trainer := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())

	rec, raw := adminTrainerRequest(router, adminToken(t, tokenSvc, config.Admin1ID), http.MethodGet, adminTrainersRoute+"/"+trainer.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	for _, field := range []string{`"id"`, `"user_id"`, `"email"`, `"first_name"`, `"last_name"`, `"status"`, `"created_at"`, `"updated_at"`} {
		if !strings.Contains(raw, field) {
			t.Fatalf("expected field %s in response, got %s", field, raw)
		}
	}
	if !strings.Contains(raw, `"status":"active"`) {
		t.Fatalf("expected active status in response, got %s", raw)
	}
	assertNoSensitiveLeak(t, raw)
}

func TestAdminGetTrainerUnknownUUID(t *testing.T) {
	router, _, _, tokenSvc := newAdminTrainersTestRouter(t, true)

	rec, raw := adminTrainerRequest(router, adminToken(t, tokenSvc, config.Admin1ID), http.MethodGet, adminTrainersRoute+"/"+uuid.NewString())
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"TRAINER_NOT_FOUND"`) {
		t.Fatalf("expected TRAINER_NOT_FOUND, got %s", raw)
	}
}

func TestAdminGetTrainerMalformedID(t *testing.T) {
	router, _, _, tokenSvc := newAdminTrainersTestRouter(t, true)

	rec, raw := adminTrainerRequest(router, adminToken(t, tokenSvc, config.Admin1ID), http.MethodGet, adminTrainersRoute+"/not-a-uuid")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
}

func TestAdminGetTrainerSoftDeleted(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	trainer := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())
	if err := trainerRepo.SoftDelete(context.Background(), trainer.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	rec, raw := adminTrainerRequest(router, adminToken(t, tokenSvc, config.Admin1ID), http.MethodGet, adminTrainersRoute+"/"+trainer.ID)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
}

func TestAdminCreateTrainerSuccessAndCanLogin(t *testing.T) {
	router, _, _, tokenSvc := newAdminTrainersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	email := uniqueEmail()

	rec, raw := adminTrainerJSONRequest(router, cookie, http.MethodPost, adminTrainersRoute,
		`{"email":"`+email+`","password":"Password123!","first_name":"New","last_name":"Trainer"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"message":"Trainer created successfully."`) {
		t.Fatalf("expected create message, got %s", raw)
	}
	if !strings.Contains(raw, email) {
		t.Fatalf("expected created email in response, got %s", raw)
	}
	if !strings.Contains(raw, `"status":"active"`) {
		t.Fatalf("expected active status in response, got %s", raw)
	}
	assertNoSensitiveLeak(t, raw)

	rec, _, _ = attemptLogin(router, loginBody(email, "Password123!"))
	if rec.Code != http.StatusOK {
		t.Fatalf("created trainer user must be able to log in, got %d", rec.Code)
	}
}

func TestAdminCreateTrainerDuplicateEmail(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	trainer := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())

	linked, err := userRepo.FindByID(context.Background(), trainer.UserID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}

	rec, raw := adminTrainerJSONRequest(router, cookie, http.MethodPost, adminTrainersRoute,
		`{"email":"`+linked.Email+`","password":"Password123!","first_name":"Dup","last_name":"Trainer"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"EMAIL_ALREADY_REGISTERED"`) {
		t.Fatalf("expected EMAIL_ALREADY_REGISTERED, got %s", raw)
	}
}

func TestAdminCreateTrainerInvalidInput(t *testing.T) {
	router, _, _, tokenSvc := newAdminTrainersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	for _, body := range []string{
		`{"email":"missing-pass@ryze.local","password":"","first_name":"A","last_name":"B"}`,
		`{"email":"not-an-email","password":"Password123!","first_name":"A","last_name":"B"}`,
		`{"email":"missing-fields@ryze.local","password":"Password123!"}`,
		`{"email":"x","password":"Password123!","first_name":"","last_name":""}`,
	} {
		rec, raw := adminTrainerJSONRequest(router, cookie, http.MethodPost, adminTrainersRoute, body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("body %s: expected 400, got %d (body: %s)", body, rec.Code, raw)
		}
		if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
			t.Fatalf("body %s: expected VALIDATION_ERROR, got %s", body, raw)
		}
	}
}

func TestAdminCreateTrainerLinkedUserConflict(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	trainer := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())
	linked, err := userRepo.FindByID(context.Background(), trainer.UserID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if err := userRepo.SoftDelete(context.Background(), linked.ID); err != nil {
		t.Fatalf("SoftDelete user: %v", err)
	}

	rec, raw := adminTrainerJSONRequest(router, cookie, http.MethodPost, adminTrainersRoute,
		`{"email":"`+linked.Email+`","password":"Password123!","first_name":"Again","last_name":"Trainer"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"TRAINER_ALREADY_LINKED"`) {
		t.Fatalf("expected TRAINER_ALREADY_LINKED, got %s", raw)
	}

	if rec, _, _ := attemptLogin(router, loginBody(linked.Email, "Password123!")); rec.Code != http.StatusUnauthorized {
		t.Fatalf("the reactivated-then-compensated user must stay disabled, got %d", rec.Code)
	}
}

func TestAdminUpdateTrainerSuccess(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	trainer := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())

	rec, raw := adminTrainerJSONRequest(router, cookie, http.MethodPatch, adminTrainersRoute+"/"+trainer.ID,
		`{"first_name":"Renamed","last_name":"Surname"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"first_name":"Renamed"`) {
		t.Fatalf("expected updated first name in response, got %s", raw)
	}
	assertNoSensitiveLeak(t, raw)
}

func TestAdminUpdateTrainerDuplicateEmail(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	target := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())
	owner := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())

	ownerUser, err := userRepo.FindByID(context.Background(), owner.UserID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}

	rec, raw := adminTrainerJSONRequest(router, cookie, http.MethodPatch, adminTrainersRoute+"/"+target.ID,
		`{"email":"`+ownerUser.Email+`"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"EMAIL_ALREADY_REGISTERED"`) {
		t.Fatalf("expected EMAIL_ALREADY_REGISTERED, got %s", raw)
	}
}

func TestAdminUpdateTrainerNotFound(t *testing.T) {
	router, _, _, tokenSvc := newAdminTrainersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	rec, raw := adminTrainerJSONRequest(router, cookie, http.MethodPatch, adminTrainersRoute+"/"+uuid.NewString(),
		`{"first_name":"Nobody"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"TRAINER_NOT_FOUND"`) {
		t.Fatalf("expected TRAINER_NOT_FOUND, got %s", raw)
	}
}

func TestAdminUpdateTrainerInvalidInput(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	trainer := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())

	for _, body := range []string{
		`{}`,
		`{"email":"not-an-email"}`,
		`{"first_name":""}`,
	} {
		rec, raw := adminTrainerJSONRequest(router, cookie, http.MethodPatch, adminTrainersRoute+"/"+trainer.ID, body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("body %s: expected 400, got %d (body: %s)", body, rec.Code, raw)
		}
		if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
			t.Fatalf("body %s: expected VALIDATION_ERROR, got %s", body, raw)
		}
	}
}

func TestAdminUpdateTrainerUserDisabled(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	trainer := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())
	if err := userRepo.SoftDelete(context.Background(), trainer.UserID); err != nil {
		t.Fatalf("SoftDelete user: %v", err)
	}

	rec, raw := adminTrainerJSONRequest(router, cookie, http.MethodPatch, adminTrainersRoute+"/"+trainer.ID,
		`{"first_name":"Renamed"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"USER_DISABLED"`) {
		t.Fatalf("expected USER_DISABLED, got %s", raw)
	}
}

func TestAdminSoftDeleteTrainerPreservesUser(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	trainer := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())
	user, err := userRepo.FindByID(context.Background(), trainer.UserID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	userCookie := loginForCookie(t, router, user.Email, "Password123!")

	if rec, _, _ := requestMe(router, userCookie); rec.Code != http.StatusOK {
		t.Fatal("expected /me 200 before the trainer soft delete")
	}

	rec, raw := adminTrainerRequest(router, cookie, http.MethodPatch, adminTrainersRoute+"/"+trainer.ID+"/disable")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"message":"Trainer disabled successfully."`) {
		t.Fatalf("expected disable message, got %s", raw)
	}
	assertNoSensitiveLeak(t, raw)

	if rec, _, _ := requestMe(router, userCookie); rec.Code != http.StatusOK {
		t.Fatalf("the user account must remain active after the trainer soft delete, got %d", rec.Code)
	}
	if rec, _, _ := attemptLogin(router, loginBody(user.Email, "Password123!")); rec.Code != http.StatusOK {
		t.Fatalf("the user account must still log in after the trainer soft delete, got %d", rec.Code)
	}
}

func TestAdminSoftDeleteTrainerMissing(t *testing.T) {
	router, _, _, tokenSvc := newAdminTrainersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	rec, raw := adminTrainerRequest(router, cookie, http.MethodPatch, adminTrainersRoute+"/"+uuid.NewString()+"/disable")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"TRAINER_NOT_FOUND"`) {
		t.Fatalf("expected TRAINER_NOT_FOUND, got %s", raw)
	}
}

func TestAdminReactivateTrainerRestores(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	trainer := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())

	rec, raw := adminTrainerRequest(router, cookie, http.MethodPatch, adminTrainersRoute+"/"+trainer.ID+"/disable")
	if rec.Code != http.StatusOK {
		t.Fatalf("disable: expected 200, got %d (body: %s)", rec.Code, raw)
	}

	rec, raw = adminTrainerJSONRequest(router, cookie, http.MethodPost, adminTrainersRoute+"/"+trainer.ID+"/reactivate", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"message":"Trainer reactivated successfully."`) {
		t.Fatalf("expected reactivate message, got %s", raw)
	}
	if !strings.Contains(raw, trainer.ID) {
		t.Fatalf("expected the same trainer id in response, got %s", raw)
	}
	if !strings.Contains(raw, `"status":"active"`) {
		t.Fatalf("expected active status after reactivation, got %s", raw)
	}
	assertNoSensitiveLeak(t, raw)
}

func TestAdminReactivateTrainerAlreadyActive(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	trainer := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())

	rec, raw := adminTrainerJSONRequest(router, cookie, http.MethodPost, adminTrainersRoute+"/"+trainer.ID+"/reactivate", "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"TRAINER_ALREADY_ACTIVE"`) {
		t.Fatalf("expected TRAINER_ALREADY_ACTIVE, got %s", raw)
	}
}

func TestAdminReactivateTrainerNotFound(t *testing.T) {
	router, _, _, tokenSvc := newAdminTrainersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	rec, raw := adminTrainerJSONRequest(router, cookie, http.MethodPost, adminTrainersRoute+"/"+uuid.NewString()+"/reactivate", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"TRAINER_NOT_FOUND"`) {
		t.Fatalf("expected TRAINER_NOT_FOUND, got %s", raw)
	}
}

func TestAdminReactivateTrainerUserDisabled(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newAdminTrainersTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)
	trainer := seedTrainer(t, userRepo, trainerRepo, uniqueEmail())

	if err := trainerRepo.SoftDelete(context.Background(), trainer.ID); err != nil {
		t.Fatalf("SoftDelete trainer: %v", err)
	}
	if err := userRepo.SoftDelete(context.Background(), trainer.UserID); err != nil {
		t.Fatalf("SoftDelete user: %v", err)
	}

	rec, raw := adminTrainerJSONRequest(router, cookie, http.MethodPost, adminTrainersRoute+"/"+trainer.ID+"/reactivate", "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"USER_DISABLED"`) {
		t.Fatalf("expected USER_DISABLED, got %s", raw)
	}
}

// failingTrainerRepo and friends simulate unexpected dependency failures for the
// internal-error test. The handler must never leak the underlying error message.
type failingTrainerRepo struct{}

var errTrainerRepoFailure = errors.New("trainer repository failure")

func (failingTrainerRepo) Create(_ context.Context, _ *models.Trainer) error {
	return errTrainerRepoFailure
}
func (failingTrainerRepo) FindByID(_ context.Context, _ string) (*models.Trainer, error) {
	return nil, errTrainerRepoFailure
}
func (failingTrainerRepo) FindByIDIncludingDeleted(_ context.Context, _ string) (*models.Trainer, error) {
	return nil, errTrainerRepoFailure
}
func (failingTrainerRepo) FindByUserID(_ context.Context, _ string) (*models.Trainer, error) {
	return nil, errTrainerRepoFailure
}
func (failingTrainerRepo) ListActive(_ context.Context, _ int, _ int) ([]models.Trainer, int64, error) {
	return nil, 0, errTrainerRepoFailure
}
func (failingTrainerRepo) ListDeleted(_ context.Context, _ int, _ int) ([]models.Trainer, int64, error) {
	return nil, 0, errTrainerRepoFailure
}
func (failingTrainerRepo) SoftDelete(_ context.Context, _ string) error {
	return errTrainerRepoFailure
}
func (failingTrainerRepo) Reactivate(_ context.Context, _ string) error {
	return errTrainerRepoFailure
}

type failingTrainerUserRepo struct{}

func (failingTrainerUserRepo) FindByID(_ context.Context, _ string) (*models.User, error) {
	return nil, errTrainerRepoFailure
}
func (failingTrainerUserRepo) SoftDelete(_ context.Context, _ string) error {
	return errTrainerRepoFailure
}

type failingTrainerRegistrar struct{}

func (failingTrainerRegistrar) Register(_ context.Context, _ registration.RegisterInput) (*models.User, error) {
	return nil, errors.New("trainer registrar failure")
}

type failingTrainerUpdater struct{}

func (failingTrainerUpdater) UpdateUser(_ context.Context, _ string, _ admin_users.UpdateUserInput) (*models.User, error) {
	return nil, errors.New("trainer updater failure")
}

func TestAdminTrainersInternalError(t *testing.T) {
	svc := admin_trainers.NewAdminTrainerService(failingTrainerRepo{}, failingTrainerUserRepo{}, failingTrainerRegistrar{}, failingTrainerUpdater{})
	handler := auth.NewAdminTrainerHandler(svc)

	router := gin.New()
	router.GET("/admin/trainers", handler.ListTrainers)
	router.GET("/admin/trainers/deleted", handler.ListDeletedTrainers)
	router.GET("/admin/trainers/:id", handler.GetTrainer)
	router.POST("/admin/trainers", handler.CreateTrainer)
	router.PATCH("/admin/trainers/:id", handler.UpdateTrainer)
	router.PATCH("/admin/trainers/:id/disable", handler.SoftDeleteTrainer)
	router.POST("/admin/trainers/:id/reactivate", handler.ReactivateTrainer)

	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	id := uuid.NewString()
	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/admin/trainers"},
		{method: http.MethodGet, path: "/admin/trainers/deleted"},
		{method: http.MethodGet, path: "/admin/trainers/" + id},
		{
			method: http.MethodPost,
			path:   "/admin/trainers",
			body:   `{"email":"x@ryze.local","password":"Password123!","first_name":"A","last_name":"B"}`,
		},
		{method: http.MethodPatch, path: "/admin/trainers/" + id, body: `{"first_name":"Renamed"}`},
		{method: http.MethodPatch, path: "/admin/trainers/" + id + "/disable"},
		{method: http.MethodPost, path: "/admin/trainers/" + id + "/reactivate"},
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
		for _, leaked := range []string{"trainer repository failure", "trainer registrar failure", "trainer updater failure"} {
			if strings.Contains(rec.Body.String(), leaked) {
				t.Fatalf("%s %s: internal error details must never be exposed, got %s", tc.method, tc.path, rec.Body.String())
			}
		}
	}
}
