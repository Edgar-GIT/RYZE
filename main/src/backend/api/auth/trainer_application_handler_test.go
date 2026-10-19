package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ryze/backend/api/auth"
	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/middleware"
	"ryze/backend/middleware/adminauthcontext"
	"ryze/backend/middleware/adminroles"
	"ryze/backend/middleware/authcontext"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/login"
	"ryze/backend/services/password"
	"ryze/backend/services/token"
	"ryze/backend/services/trainer_applications"
	"ryze/backend/services/trainer_evidence"
)

const (
	applyRoute             = "/api/v1/trainer/apply"
	accessRoute            = "/api/v1/trainer/access"
	documentsRoute         = "/api/v1/trainer/application/documents"
	adminApplicationsRoute = "/api/v1/admin/trainer-applications"
	evidenceTestMaxBytes   = 1 << 20
)

// Minimal magic-byte-valid payloads for each allow-listed document type.
var (
	pdfPayload  = []byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\ntrailer\n%%EOF\n")
	jpegPayload = append([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'}, bytes.Repeat([]byte{0x00}, 32)...)
)

// trainerApplicationEnv is the full trainer-application test environment: the
// router plus every repository and configuration the evidence tests need.
type trainerApplicationEnv struct {
	router          *gin.Engine
	userRepo        repositories.UserRepository
	trainerRepo     repositories.TrainerRepository
	applicationRepo repositories.TrainerApplicationRepository
	documentRepo    repositories.TrainerApplicationDocumentRepository
	decisionRepo    repositories.TrainerApplicationDecisionRepository
	evidenceDir     string
	tokenSvc        token.Service
}

// newTrainerApplicationTestRouter wires the trainer-application endpoints
// behind the real authentication and authorization middleware, plus the login
// and /me endpoints needed to exercise the user lifecycle. It is backed by a
// database transaction so created records are rolled back. The user apply
// endpoint runs under Authenticate; the admin endpoints run under
// trainer_applications.read and trainer_applications.manage, both shared by
// every admin role.
func newTrainerApplicationTestRouter(t *testing.T, secure bool) (*gin.Engine, repositories.UserRepository, repositories.TrainerRepository, repositories.TrainerApplicationRepository, token.Service) {
	t.Helper()
	env := newTrainerApplicationEnv(t, secure)
	return env.router, env.userRepo, env.trainerRepo, env.applicationRepo, env.tokenSvc
}

func newTrainerApplicationEnv(t *testing.T, secure bool) trainerApplicationEnv {
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

	userRepo := repositories.NewUserRepository(tx)
	trainerRepo := repositories.NewTrainerRepository(tx)
	applicationRepo := repositories.NewTrainerApplicationRepository(tx)
	documentRepo := repositories.NewTrainerApplicationDocumentRepository(tx)
	decisionRepo := repositories.NewTrainerApplicationDecisionRepository(tx)
	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)

	// The evidence store is a real filesystem store rooted in the test's own
	// temporary directory, so uploads exercise the same path as production
	// (magic bytes, atomic rename, private permissions) without touching the
	// development storage.
	evidenceDir := t.TempDir()
	evidenceStore := trainer_evidence.NewFileStore(evidenceDir, evidenceTestMaxBytes)

	loginSvc := login.NewLoginService(userRepo, password.Verifier{})
	loginHandler := auth.NewLoginHandler(loginSvc, tokenSvc, testTokenTTL, secure)
	meHandler := auth.NewMeHandler(userRepo)

	service := trainer_applications.NewService(applicationRepo, userRepo, trainerRepo, documentRepo, decisionRepo, evidenceStore)
	adminHandler := auth.NewAdminTrainerApplicationHandler(service)
	userHandler := auth.NewTrainerApplicationHandler(service)

	router := gin.New()
	router.POST(loginRoute, loginHandler.Login)
	router.GET(meRoute, middleware.Authenticate(tokenSvc, userRepo), meHandler.GetMe)
	router.POST(applyRoute, middleware.Authenticate(tokenSvc, userRepo), userHandler.Apply)
	router.GET(accessRoute, middleware.Authenticate(tokenSvc, userRepo), userHandler.GetAccessStatus)
	router.POST(documentsRoute, middleware.Authenticate(tokenSvc, userRepo), userHandler.UploadDocument)
	router.GET(documentsRoute, middleware.Authenticate(tokenSvc, userRepo), userHandler.ListOwnDocuments)
	router.GET(documentsRoute+"/:documentID/download", middleware.Authenticate(tokenSvc, userRepo), userHandler.DownloadOwnDocument)
	router.DELETE(documentsRoute+"/:documentID", middleware.Authenticate(tokenSvc, userRepo), userHandler.DeleteOwnDocument)

	admin := router.Group("/api/v1/admin")
	admin.Use(middleware.AdminAuthenticate(tokenSvc))

	adminRead := admin.Group("")
	adminRead.Use(middleware.RequireAdminPermission(adminroles.PermissionTrainerApplicationsRead))
	adminRead.GET("/trainer-applications", adminHandler.ListApplications)
	adminRead.GET("/trainer-applications/:id", adminHandler.GetApplication)
	adminRead.GET("/trainer-applications/:id/documents", adminHandler.ListDocuments)
	adminRead.GET("/trainer-applications/:id/documents/:documentID/download", adminHandler.DownloadDocument)
	adminRead.GET("/trainer-applications/:id/decisions", adminHandler.ListDecisions)

	adminMutate := admin.Group("")
	adminMutate.Use(middleware.RequireAdminPermission(adminroles.PermissionTrainerApplicationsManage))
	adminMutate.POST("/trainer-applications/:id/approve", adminHandler.ApproveApplication)
	adminMutate.POST("/trainer-applications/:id/reject", adminHandler.RejectApplication)

	return trainerApplicationEnv{
		router:          router,
		userRepo:        userRepo,
		trainerRepo:     trainerRepo,
		applicationRepo: applicationRepo,
		documentRepo:    documentRepo,
		decisionRepo:    decisionRepo,
		evidenceDir:     evidenceDir,
		tokenSvc:        tokenSvc,
	}
}

// seedApplication creates an active user with a PENDING trainer application.
func seedApplication(t *testing.T, userRepo repositories.UserRepository, applicationRepo repositories.TrainerApplicationRepository, email string) (*models.User, *models.TrainerApplication) {
	t.Helper()

	user := seedLoginUser(t, userRepo, email, "Password123!")
	application := &models.TrainerApplication{
		UserID: user.ID,
		Status: models.ApplicationStatusPending,
	}
	if err := applicationRepo.Create(context.Background(), application); err != nil {
		t.Fatalf("seed application: %v", err)
	}
	return user, application
}

// applyRequest performs a trainer-application HTTP request authenticated with
// a user cookie.
func applyRequest(router http.Handler, cookieValue, method, path, body string) (*httptest.ResponseRecorder, string) {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookieValue != "" {
		req.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: cookieValue})
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec, rec.Body.String()
}

func adminApplicationRequest(router http.Handler, cookieValue, method, path string) (*httptest.ResponseRecorder, string) {
	return adminApplicationJSONRequest(router, cookieValue, method, path, "")
}

func adminApplicationJSONRequest(router http.Handler, cookieValue, method, path, body string) (*httptest.ResponseRecorder, string) {
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

func TestApplyUnauthenticated(t *testing.T) {
	router, _, _, _, _ := newTrainerApplicationTestRouter(t, true)

	rec, raw := applyRequest(router, "", http.MethodPost, applyRoute, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestAccessStatusUnauthenticated(t *testing.T) {
	router, _, _, _, _ := newTrainerApplicationTestRouter(t, true)

	rec, raw := applyRequest(router, "", http.MethodGet, accessRoute, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestAccessStatusWithoutApplication(t *testing.T) {
	router, userRepo, _, _, tokenSvc := newTrainerApplicationTestRouter(t, true)
	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	cookie := userToken(t, tokenSvc, user.ID, user.SessionVersion)

	rec, raw := applyRequest(router, cookie, http.MethodGet, accessRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"is_trainer":false`) {
		t.Fatalf("expected is_trainer false, got %s", raw)
	}
	if !strings.Contains(raw, `"application":null`) {
		t.Fatalf("expected a null application, got %s", raw)
	}
	if strings.Contains(raw, user.Email) {
		t.Fatalf("access status must not echo the caller's own email, got %s", raw)
	}
}

func TestAccessStatusPendingApplication(t *testing.T) {
	router, userRepo, _, applicationRepo, tokenSvc := newTrainerApplicationTestRouter(t, true)
	user, _ := seedApplication(t, userRepo, applicationRepo, uniqueEmail())
	cookie := userToken(t, tokenSvc, user.ID, user.SessionVersion)

	rec, raw := applyRequest(router, cookie, http.MethodGet, accessRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"is_trainer":false`) {
		t.Fatalf("expected is_trainer false while pending, got %s", raw)
	}
	if !strings.Contains(raw, `"status":"PENDING"`) {
		t.Fatalf("expected PENDING application, got %s", raw)
	}
}

func TestAccessStatusApprovedTrainer(t *testing.T) {
	router, userRepo, _, applicationRepo, tokenSvc := newTrainerApplicationTestRouter(t, true)
	user, application := seedApplication(t, userRepo, applicationRepo, uniqueEmail())
	if _, err := applicationRepo.Approve(context.Background(), application.ID, nil); err != nil {
		t.Fatalf("approve application: %v", err)
	}
	cookie := userToken(t, tokenSvc, user.ID, user.SessionVersion)

	rec, raw := applyRequest(router, cookie, http.MethodGet, accessRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"is_trainer":true`) {
		t.Fatalf("expected is_trainer true for an approved trainer, got %s", raw)
	}
	if !strings.Contains(raw, `"status":"APPROVED"`) {
		t.Fatalf("expected APPROVED application, got %s", raw)
	}
}

func TestAccessStatusRejectedApplication(t *testing.T) {
	router, userRepo, _, applicationRepo, tokenSvc := newTrainerApplicationTestRouter(t, true)
	user, application := seedApplication(t, userRepo, applicationRepo, uniqueEmail())
	if err := applicationRepo.Reject(context.Background(), application.ID, nil); err != nil {
		t.Fatalf("reject application: %v", err)
	}
	cookie := userToken(t, tokenSvc, user.ID, user.SessionVersion)

	rec, raw := applyRequest(router, cookie, http.MethodGet, accessRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"is_trainer":false`) {
		t.Fatalf("expected is_trainer false after rejection, got %s", raw)
	}
	if !strings.Contains(raw, `"status":"REJECTED"`) {
		t.Fatalf("expected REJECTED application so the outcome is visible, got %s", raw)
	}
}

func TestApplySuccess(t *testing.T) {
	router, userRepo, _, _, tokenSvc := newTrainerApplicationTestRouter(t, true)
	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	cookie := userToken(t, tokenSvc, user.ID, user.SessionVersion)

	rec, raw := applyRequest(router, cookie, http.MethodPost, applyRoute, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"success":true`) {
		t.Fatalf("expected success:true, got %s", raw)
	}
	if !strings.Contains(raw, `"message":"Trainer application submitted successfully."`) {
		t.Fatalf("expected submit message, got %s", raw)
	}
	if !strings.Contains(raw, `"status":"PENDING"`) {
		t.Fatalf("expected PENDING status, got %s", raw)
	}
	if !strings.Contains(raw, `"user_id":"`+user.ID+`"`) {
		t.Fatalf("expected the authenticated user id in response, got %s", raw)
	}
	if !strings.Contains(raw, user.Email) {
		t.Fatalf("expected the applicant email in response, got %s", raw)
	}
	assertNoSensitiveLeak(t, raw)
}

// TestApplyIgnoresClientUserID verifies the apply endpoint never trusts a
// client-provided identity: even when the request body carries a foreign
// user_id, the application is always created for the authenticated user.
func TestApplyIgnoresClientUserID(t *testing.T) {
	router, userRepo, _, _, tokenSvc := newTrainerApplicationTestRouter(t, true)
	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	cookie := userToken(t, tokenSvc, user.ID, user.SessionVersion)

	foreignID := uuid.NewString()
	rec, raw := applyRequest(router, cookie, http.MethodPost, applyRoute,
		`{"user_id":"`+foreignID+`","status":"APPROVED"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, foreignID) {
		t.Fatalf("client-provided user_id must be ignored, got %s", raw)
	}
	if !strings.Contains(raw, `"user_id":"`+user.ID+`"`) {
		t.Fatalf("expected the authenticated user id in response, got %s", raw)
	}
	if !strings.Contains(raw, `"status":"PENDING"`) {
		t.Fatalf("client-provided status must be ignored, expected PENDING, got %s", raw)
	}
}

func TestApplyDuplicatePendingApplication(t *testing.T) {
	router, userRepo, _, applicationRepo, tokenSvc := newTrainerApplicationTestRouter(t, true)
	user, _ := seedApplication(t, userRepo, applicationRepo, uniqueEmail())
	cookie := userToken(t, tokenSvc, user.ID, user.SessionVersion)

	rec, raw := applyRequest(router, cookie, http.MethodPost, applyRoute, "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"APPLICATION_ALREADY_EXISTS"`) {
		t.Fatalf("expected APPLICATION_ALREADY_EXISTS, got %s", raw)
	}
}

func TestApplyAlreadyTrainer(t *testing.T) {
	router, userRepo, trainerRepo, _, tokenSvc := newTrainerApplicationTestRouter(t, true)
	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	if err := trainerRepo.Create(context.Background(), &models.Trainer{UserID: user.ID}); err != nil {
		t.Fatalf("seed trainer: %v", err)
	}
	cookie := userToken(t, tokenSvc, user.ID, user.SessionVersion)

	rec, raw := applyRequest(router, cookie, http.MethodPost, applyRoute, "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"ALREADY_TRAINER"`) {
		t.Fatalf("expected ALREADY_TRAINER, got %s", raw)
	}
}

func TestApplySoftDeletedUser(t *testing.T) {
	router, userRepo, _, _, tokenSvc := newTrainerApplicationTestRouter(t, true)
	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	cookie := userToken(t, tokenSvc, user.ID, user.SessionVersion)
	if err := userRepo.SoftDelete(context.Background(), user.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	rec, raw := applyRequest(router, cookie, http.MethodPost, applyRoute, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestAdminListApplicationsBothRoles(t *testing.T) {
	router, userRepo, _, applicationRepo, tokenSvc := newTrainerApplicationTestRouter(t, true)
	_, application := seedApplication(t, userRepo, applicationRepo, uniqueEmail())

	for _, adminID := range []string{config.Admin1ID, config.Admin2ID} {
		t.Run(adminID, func(t *testing.T) {
			rec, raw := adminApplicationRequest(router, adminToken(t, tokenSvc, adminID), http.MethodGet, adminApplicationsRoute)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"success":true`) {
				t.Fatalf("expected success:true, got %s", raw)
			}
			if !strings.Contains(raw, application.ID) {
				t.Fatalf("expected seeded application in response, got %s", raw)
			}
			assertNoSensitiveLeak(t, raw)
		})
	}
}

func TestAdminListApplicationsFilterByStatus(t *testing.T) {
	router, userRepo, _, applicationRepo, tokenSvc := newTrainerApplicationTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	_, pending := seedApplication(t, userRepo, applicationRepo, uniqueEmail())
	_, approved := seedApplication(t, userRepo, applicationRepo, uniqueEmail())
	if err := applicationRepo.Reject(context.Background(), approved.ID, nil); err != nil {
		t.Fatalf("Reject: %v", err)
	}

	rec, raw := adminApplicationRequest(router, cookie, http.MethodGet, adminApplicationsRoute+"?status=PENDING")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, pending.ID) {
		t.Fatalf("expected pending application in response, got %s", raw)
	}
	if strings.Contains(raw, approved.ID) {
		t.Fatalf("rejected application must not appear in PENDING filter, got %s", raw)
	}

	rec, raw = adminApplicationRequest(router, cookie, http.MethodGet, adminApplicationsRoute+"?status=REJECTED")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, approved.ID) {
		t.Fatalf("expected rejected application in response, got %s", raw)
	}
}

func TestAdminListApplicationsInvalidStatus(t *testing.T) {
	router, _, _, _, tokenSvc := newTrainerApplicationTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	rec, raw := adminApplicationRequest(router, cookie, http.MethodGet, adminApplicationsRoute+"?status=DELETED")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestAdminListApplicationsInvalidPagination(t *testing.T) {
	router, _, _, _, tokenSvc := newTrainerApplicationTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	for _, query := range []string{
		"?page=0&limit=20",
		"?page=-1&limit=20",
		"?page=1&limit=0",
		"?page=1&limit=-3",
		"?page=abc&limit=20",
		"?page=1&limit=abc",
	} {
		rec, raw := adminApplicationRequest(router, cookie, http.MethodGet, adminApplicationsRoute+query)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d (body: %s)", query, rec.Code, raw)
		}
		if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
			t.Fatalf("%s: expected VALIDATION_ERROR, got %s", query, raw)
		}
	}
}

func TestAdminListApplicationsPagination(t *testing.T) {
	router, userRepo, _, applicationRepo, tokenSvc := newTrainerApplicationTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	baseline := applicationListTotal(t, router, cookie)
	created := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		_, application := seedApplication(t, userRepo, applicationRepo, uniqueEmail())
		created = append(created, application.ID)
	}
	expectedTotal := baseline + 3

	pageIDs := make([]string, 0, 3)
	for page := 1; page <= 3; page++ {
		rec, raw := adminApplicationRequest(router, cookie, http.MethodGet, adminApplicationsRoute+"?page="+strconv.Itoa(page)+"&limit=1")
		if rec.Code != http.StatusOK {
			t.Fatalf("page %d: expected 200, got %d (body: %s)", page, rec.Code, raw)
		}
		var payload struct {
			Data struct {
				Applications []struct {
					ID string `json:"id"`
				} `json:"applications"`
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
		if len(payload.Data.Applications) != 1 {
			t.Fatalf("page %d: expected 1 application, got %d", page, len(payload.Data.Applications))
		}
		pageIDs = append(pageIDs, payload.Data.Applications[0].ID)
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
			t.Fatalf("created application %q was not returned across pages %v", id, pageIDs)
		}
	}
}

func TestAdminListApplicationsClampsOversizedLimit(t *testing.T) {
	router, _, _, _, tokenSvc := newTrainerApplicationTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	rec, raw := adminApplicationRequest(router, cookie, http.MethodGet, adminApplicationsRoute+"?page=1&limit=9999")
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
	if payload.Data.Pagination.Limit != trainer_applications.MaxPageSize {
		t.Fatalf("expected limit %d, got %d", trainer_applications.MaxPageSize, payload.Data.Pagination.Limit)
	}
}

// applicationListTotal fetches the total application count reported by the list
// endpoint.
func applicationListTotal(t *testing.T, router http.Handler, cookie string) int {
	t.Helper()
	rec, raw := adminApplicationRequest(router, cookie, http.MethodGet, adminApplicationsRoute+"?page=1&limit=1")
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

func TestAdminGetApplicationBothRoles(t *testing.T) {
	router, userRepo, _, applicationRepo, tokenSvc := newTrainerApplicationTestRouter(t, true)
	_, application := seedApplication(t, userRepo, applicationRepo, uniqueEmail())

	for _, adminID := range []string{config.Admin1ID, config.Admin2ID} {
		t.Run(adminID, func(t *testing.T) {
			rec, raw := adminApplicationRequest(router, adminToken(t, tokenSvc, adminID), http.MethodGet, adminApplicationsRoute+"/"+application.ID)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, application.ID) {
				t.Fatalf("expected application id in response, got %s", raw)
			}
			if !strings.Contains(raw, `"status":"PENDING"`) {
				t.Fatalf("expected PENDING status, got %s", raw)
			}
			assertNoSensitiveLeak(t, raw)
		})
	}
}

func TestAdminGetApplicationNotFound(t *testing.T) {
	router, _, _, _, tokenSvc := newTrainerApplicationTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	rec, raw := adminApplicationRequest(router, cookie, http.MethodGet, adminApplicationsRoute+"/"+uuid.NewString())
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"APPLICATION_NOT_FOUND"`) {
		t.Fatalf("expected APPLICATION_NOT_FOUND, got %s", raw)
	}
}

func TestAdminGetApplicationMalformedID(t *testing.T) {
	router, _, _, _, tokenSvc := newTrainerApplicationTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	rec, raw := adminApplicationRequest(router, cookie, http.MethodGet, adminApplicationsRoute+"/not-a-uuid")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
}

// TestAdminApproveApplicationBothRoles approves the same PENDING application
// once per role and verifies the full expected outcome: the application becomes
// APPROVED, the trainer profile is created for the same user, the user identity
// is unchanged and the original login still works.
func TestAdminApproveApplicationBothRoles(t *testing.T) {
	for _, adminID := range []string{config.Admin1ID, config.Admin2ID} {
		t.Run(adminID, func(t *testing.T) {
			router, userRepo, trainerRepo, applicationRepo, tokenSvc := newTrainerApplicationTestRouter(t, true)
			user, application := seedApplication(t, userRepo, applicationRepo, uniqueEmail())
			cookie := loginForCookie(t, router, user.Email, "Password123!")

			rec, raw := adminApplicationRequest(router, adminToken(t, tokenSvc, adminID), http.MethodPost, adminApplicationsRoute+"/"+application.ID+"/approve")
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"message":"Trainer application approved successfully."`) {
				t.Fatalf("expected approve message, got %s", raw)
			}
			if !strings.Contains(raw, `"status":"APPROVED"`) {
				t.Fatalf("expected APPROVED status, got %s", raw)
			}
			if !strings.Contains(raw, user.ID) {
				t.Fatalf("expected the same user id in response, got %s", raw)
			}
			assertNoSensitiveLeak(t, raw)

			trainer, err := trainerRepo.FindByUserID(context.Background(), user.ID)
			if err != nil {
				t.Fatalf("expected a trainer profile to exist after approval: %v", err)
			}
			if trainer.UserID != user.ID {
				t.Fatalf("expected trainer linked to the same user id, got %q", trainer.UserID)
			}

			unchanged, err := userRepo.FindByID(context.Background(), user.ID)
			if err != nil {
				t.Fatalf("expected the user to remain active: %v", err)
			}
			if unchanged.ID != user.ID || unchanged.Email != user.Email {
				t.Fatalf("the user identity must never change after approval, got %+v", unchanged)
			}

			if rec, _, _ := requestMe(router, cookie); rec.Code != http.StatusOK {
				t.Fatalf("the original user login must still work after approval, got %d", rec.Code)
			}
			if rec, _, _ := attemptLogin(router, loginBody(user.Email, "Password123!")); rec.Code != http.StatusOK {
				t.Fatalf("the user must still log in after approval, got %d", rec.Code)
			}
		})
	}
}

func TestAdminApproveApplicationAlreadyApproved(t *testing.T) {
	router, userRepo, trainerRepo, applicationRepo, tokenSvc := newTrainerApplicationTestRouter(t, true)
	_, application := seedApplication(t, userRepo, applicationRepo, uniqueEmail())
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	if _, err := applicationRepo.Approve(context.Background(), application.ID, nil); err != nil {
		t.Fatalf("Approve: %v", err)
	}

	beforeTrainers, beforeTotal, err := trainerRepo.ListActive(context.Background(), 1, trainer_applications.MaxPageSize)
	if err != nil {
		t.Fatalf("ListActive before: %v", err)
	}

	rec, raw := adminApplicationRequest(router, cookie, http.MethodPost, adminApplicationsRoute+"/"+application.ID+"/approve")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"APPLICATION_STATE_CONFLICT"`) {
		t.Fatalf("expected APPLICATION_STATE_CONFLICT, got %s", raw)
	}

	trainers, total, err := trainerRepo.ListActive(context.Background(), 1, trainer_applications.MaxPageSize)
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if total != beforeTotal || len(trainers) != len(beforeTrainers) {
		t.Fatalf("approval must never create a duplicate trainer profile, got total=%d before=%d", total, beforeTotal)
	}
}

func TestAdminApproveApplicationRejectedApplication(t *testing.T) {
	router, userRepo, _, applicationRepo, tokenSvc := newTrainerApplicationTestRouter(t, true)
	_, application := seedApplication(t, userRepo, applicationRepo, uniqueEmail())
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	if err := applicationRepo.Reject(context.Background(), application.ID, nil); err != nil {
		t.Fatalf("Reject: %v", err)
	}

	rec, raw := adminApplicationRequest(router, cookie, http.MethodPost, adminApplicationsRoute+"/"+application.ID+"/approve")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"APPLICATION_STATE_CONFLICT"`) {
		t.Fatalf("expected APPLICATION_STATE_CONFLICT, got %s", raw)
	}
}

func TestAdminApproveApplicationNotFound(t *testing.T) {
	router, _, _, _, tokenSvc := newTrainerApplicationTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	rec, raw := adminApplicationRequest(router, cookie, http.MethodPost, adminApplicationsRoute+"/"+uuid.NewString()+"/approve")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"APPLICATION_NOT_FOUND"`) {
		t.Fatalf("expected APPLICATION_NOT_FOUND, got %s", raw)
	}
}

func TestAdminApproveApplicationMalformedID(t *testing.T) {
	router, _, _, _, tokenSvc := newTrainerApplicationTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	rec, raw := adminApplicationRequest(router, cookie, http.MethodPost, adminApplicationsRoute+"/not-a-uuid/approve")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
}

// TestAdminApproveApplicationAlreadyTrainerAtomicity forces the approval of a
// user that already owns a trainer profile (an impossible state through the
// normal flow, produced here by direct repository access) and verifies the
// approval fails atomically: the application stays PENDING and no additional
// trainer profile is created.
func TestAdminApproveApplicationAlreadyTrainerAtomicity(t *testing.T) {
	router, userRepo, trainerRepo, applicationRepo, tokenSvc := newTrainerApplicationTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	if err := trainerRepo.Create(context.Background(), &models.Trainer{UserID: user.ID}); err != nil {
		t.Fatalf("seed trainer: %v", err)
	}
	_, beforeTotal, err := trainerRepo.ListActive(context.Background(), 1, trainer_applications.MaxPageSize)
	if err != nil {
		t.Fatalf("ListActive before: %v", err)
	}
	application := &models.TrainerApplication{
		UserID: user.ID,
		Status: models.ApplicationStatusPending,
	}
	if err := applicationRepo.Create(context.Background(), application); err != nil {
		t.Fatalf("seed application: %v", err)
	}

	rec, raw := adminApplicationRequest(router, cookie, http.MethodPost, adminApplicationsRoute+"/"+application.ID+"/approve")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"USER_ALREADY_TRAINER"`) {
		t.Fatalf("expected USER_ALREADY_TRAINER, got %s", raw)
	}

	persisted, err := applicationRepo.FindByID(context.Background(), application.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if persisted.Status != models.ApplicationStatusPending {
		t.Fatalf("failed approval must leave the application PENDING, got %q", persisted.Status)
	}

	_, total, err := trainerRepo.ListActive(context.Background(), 1, trainer_applications.MaxPageSize)
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if total != beforeTotal {
		t.Fatalf("failed approval must never create a trainer profile, got total=%d before=%d", total, beforeTotal)
	}
}

func TestAdminRejectApplicationBothRoles(t *testing.T) {
	for _, adminID := range []string{config.Admin1ID, config.Admin2ID} {
		t.Run(adminID, func(t *testing.T) {
			router, userRepo, trainerRepo, applicationRepo, tokenSvc := newTrainerApplicationTestRouter(t, true)
			user, application := seedApplication(t, userRepo, applicationRepo, uniqueEmail())

			rec, raw := adminApplicationRequest(router, adminToken(t, tokenSvc, adminID), http.MethodPost, adminApplicationsRoute+"/"+application.ID+"/reject")
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"message":"Trainer application rejected successfully."`) {
				t.Fatalf("expected reject message, got %s", raw)
			}

			persisted, err := applicationRepo.FindByID(context.Background(), application.ID)
			if err != nil {
				t.Fatalf("FindByID: %v", err)
			}
			if persisted.Status != models.ApplicationStatusRejected {
				t.Fatalf("expected REJECTED status, got %q", persisted.Status)
			}

			if _, err := trainerRepo.FindByUserID(context.Background(), user.ID); err == nil {
				t.Fatal("a rejection must never create a trainer profile")
			}
		})
	}
}

func TestAdminRejectApplicationAlreadyReviewed(t *testing.T) {
	router, userRepo, _, applicationRepo, tokenSvc := newTrainerApplicationTestRouter(t, true)
	_, application := seedApplication(t, userRepo, applicationRepo, uniqueEmail())
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	if err := applicationRepo.Reject(context.Background(), application.ID, nil); err != nil {
		t.Fatalf("Reject: %v", err)
	}

	rec, raw := adminApplicationRequest(router, cookie, http.MethodPost, adminApplicationsRoute+"/"+application.ID+"/reject")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"APPLICATION_STATE_CONFLICT"`) {
		t.Fatalf("expected APPLICATION_STATE_CONFLICT, got %s", raw)
	}
}

func TestAdminRejectApplicationNotFound(t *testing.T) {
	router, _, _, _, tokenSvc := newTrainerApplicationTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	rec, raw := adminApplicationRequest(router, cookie, http.MethodPost, adminApplicationsRoute+"/"+uuid.NewString()+"/reject")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"APPLICATION_NOT_FOUND"`) {
		t.Fatalf("expected APPLICATION_NOT_FOUND, got %s", raw)
	}
}

func TestAdminRejectApplicationMalformedID(t *testing.T) {
	router, _, _, _, tokenSvc := newTrainerApplicationTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	rec, raw := adminApplicationRequest(router, cookie, http.MethodPost, adminApplicationsRoute+"/not-a-uuid/reject")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
}

// TestAdminRejectThenReapplyApproved exercises the full review lifecycle: a
// rejected application stays in history, the user can apply again, and the new
// application can be approved, creating exactly one trainer profile.
func TestAdminRejectThenReapplyApproved(t *testing.T) {
	router, userRepo, trainerRepo, applicationRepo, tokenSvc := newTrainerApplicationTestRouter(t, true)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	userCookie := userToken(t, tokenSvc, user.ID, user.SessionVersion)

	_, beforeTotal, err := trainerRepo.ListActive(context.Background(), 1, trainer_applications.MaxPageSize)
	if err != nil {
		t.Fatalf("ListActive before: %v", err)
	}

	rec, raw := applyRequest(router, userCookie, http.MethodPost, applyRoute, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("apply: expected 201, got %d (body: %s)", rec.Code, raw)
	}
	var firstPayload struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &firstPayload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	firstID := firstPayload.Data.ID

	rec, raw = adminApplicationRequest(router, cookie, http.MethodPost, adminApplicationsRoute+"/"+firstID+"/reject")
	if rec.Code != http.StatusOK {
		t.Fatalf("reject: expected 200, got %d (body: %s)", rec.Code, raw)
	}

	first, err := applicationRepo.FindByID(context.Background(), firstID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if first.Status != models.ApplicationStatusRejected {
		t.Fatalf("expected the first application to stay REJECTED, got %q", first.Status)
	}

	rec, raw = applyRequest(router, userCookie, http.MethodPost, applyRoute, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("re-apply: expected 201, got %d (body: %s)", rec.Code, raw)
	}
	var secondPayload struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &secondPayload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	secondID := secondPayload.Data.ID
	if secondID == firstID {
		t.Fatal("re-apply must create a new application, not reuse the rejected one")
	}

	rec, raw = adminApplicationRequest(router, cookie, http.MethodPost, adminApplicationsRoute+"/"+secondID+"/approve")
	if rec.Code != http.StatusOK {
		t.Fatalf("approve: expected 200, got %d (body: %s)", rec.Code, raw)
	}

	trainer, err := trainerRepo.FindByUserID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("expected a trainer profile after approving the re-application: %v", err)
	}
	if trainer.UserID != user.ID {
		t.Fatalf("expected trainer linked to the same user id, got %q", trainer.UserID)
	}

	_, total, err := trainerRepo.ListActive(context.Background(), 1, trainer_applications.MaxPageSize)
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if total != beforeTotal+1 {
		t.Fatalf("the whole lifecycle must produce exactly one new trainer profile, got total=%d before=%d", total, beforeTotal)
	}

	if rec, _, _ := attemptLogin(router, loginBody(user.Email, "Password123!")); rec.Code != http.StatusOK {
		t.Fatalf("the user must still log in after the full lifecycle, got %d", rec.Code)
	}
}

func TestAdminApplicationsUnauthenticated(t *testing.T) {
	router, _, _, _, _ := newTrainerApplicationTestRouter(t, true)
	id := uuid.NewString()

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: adminApplicationsRoute},
		{method: http.MethodGet, path: adminApplicationsRoute + "/" + id},
		{method: http.MethodPost, path: adminApplicationsRoute + "/" + id + "/approve"},
		{method: http.MethodPost, path: adminApplicationsRoute + "/" + id + "/reject"},
	} {
		rec, raw := adminApplicationRequest(router, "", tc.method, tc.path)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: expected 401, got %d (body: %s)", tc.method, tc.path, rec.Code, raw)
		}
		if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
			t.Fatalf("%s %s: expected AUTHENTICATION_REQUIRED, got %s", tc.method, tc.path, raw)
		}
	}
}

func TestAdminApplicationsRegularUserRejected(t *testing.T) {
	router, userRepo, _, _, tokenSvc := newTrainerApplicationTestRouter(t, true)
	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	cookie := userToken(t, tokenSvc, user.ID, user.SessionVersion)

	rec, raw := adminApplicationRequest(router, cookie, http.MethodGet, adminApplicationsRoute)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	assertNoSensitiveLeak(t, raw)
}

func TestAdminApplicationsStageTokenRejected(t *testing.T) {
	router, _, _, _, tokenSvc := newTrainerApplicationTestRouter(t, true)

	stageToken, err := tokenSvc.GenerateAdminStageToken(config.Admin1ID)
	if err != nil {
		t.Fatalf("GenerateAdminStageToken: %v", err)
	}

	rec, raw := adminApplicationRequest(router, stageToken, http.MethodGet, adminApplicationsRoute)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
}

func TestAdminApplicationsInvalidTokenRejected(t *testing.T) {
	router, _, _, _, _ := newTrainerApplicationTestRouter(t, true)

	for _, value := range []string{"garbage", "not.a.jwt", ""} {
		rec, raw := adminApplicationRequest(router, value, http.MethodGet, adminApplicationsRoute)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("value %q: expected 401, got %d (body: %s)", value, rec.Code, raw)
		}
	}
}

// failingApplicationRepo and friends simulate unexpected dependency failures so
// the internal-error test can verify the handlers never leak error details.
type failingApplicationRepo struct{}

var errApplicationRepoFailure = errors.New("application repository failure")

func (failingApplicationRepo) Create(_ context.Context, _ *models.TrainerApplication) error {
	return errApplicationRepoFailure
}
func (failingApplicationRepo) FindActiveByUserID(_ context.Context, _ string) (*models.TrainerApplication, error) {
	return nil, errApplicationRepoFailure
}
func (failingApplicationRepo) FindLatestByUserID(_ context.Context, _ string) (*models.TrainerApplication, error) {
	return nil, errApplicationRepoFailure
}
func (failingApplicationRepo) FindByID(_ context.Context, _ string) (*models.TrainerApplication, error) {
	return nil, errApplicationRepoFailure
}
func (failingApplicationRepo) List(_ context.Context, _ int, _ int, _ string) ([]models.TrainerApplication, int64, error) {
	return nil, 0, errApplicationRepoFailure
}
func (failingApplicationRepo) Approve(_ context.Context, _ string, _ *models.TrainerApplicationDecision) (*models.TrainerApplication, error) {
	return nil, errApplicationRepoFailure
}
func (failingApplicationRepo) Reject(_ context.Context, _ string, _ *models.TrainerApplicationDecision) error {
	return errApplicationRepoFailure
}

type failingApplicationDocumentRepo struct{}

func (failingApplicationDocumentRepo) Create(_ context.Context, _ *models.TrainerApplicationDocument) error {
	return errApplicationRepoFailure
}
func (failingApplicationDocumentRepo) ListByApplication(_ context.Context, _ string) ([]models.TrainerApplicationDocument, error) {
	return nil, errApplicationRepoFailure
}
func (failingApplicationDocumentRepo) FindByID(_ context.Context, _ string) (*models.TrainerApplicationDocument, error) {
	return nil, errApplicationRepoFailure
}
func (failingApplicationDocumentRepo) SoftDelete(_ context.Context, _ string) error {
	return errApplicationRepoFailure
}

type failingApplicationDecisionRepo struct{}

func (failingApplicationDecisionRepo) ListByApplication(_ context.Context, _ string) ([]models.TrainerApplicationDecision, error) {
	return nil, errApplicationRepoFailure
}

type failingApplicationUserRepo struct{}

func (failingApplicationUserRepo) FindByID(_ context.Context, _ string) (*models.User, error) {
	return nil, errApplicationRepoFailure
}

type failingApplicationTrainerRepo struct{}

func (failingApplicationTrainerRepo) FindByUserID(_ context.Context, _ string) (*models.Trainer, error) {
	return nil, errApplicationRepoFailure
}

func TestTrainerApplicationsInternalError(t *testing.T) {
	svc := trainer_applications.NewService(
		failingApplicationRepo{},
		failingApplicationUserRepo{},
		failingApplicationTrainerRepo{},
		failingApplicationDocumentRepo{},
		failingApplicationDecisionRepo{},
		trainer_evidence.NewFileStore(t.TempDir(), evidenceTestMaxBytes),
	)
	adminHandler := auth.NewAdminTrainerApplicationHandler(svc)
	userHandler := auth.NewTrainerApplicationHandler(svc)

	// The review handlers resolve the reviewer identity from the admin
	// context, so this router injects a valid identity directly instead of
	// running the admin middleware.
	withAdminIdentity := func(handler gin.HandlerFunc) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Set(adminauthcontext.AdminIdentityContextKey, config.Admin1ID)
			handler(c)
		}
	}

	router := gin.New()
	router.GET("/admin/apps", adminHandler.ListApplications)
	router.GET("/admin/apps/:id", adminHandler.GetApplication)
	router.POST("/admin/apps/:id/approve", withAdminIdentity(adminHandler.ApproveApplication))
	router.POST("/admin/apps/:id/reject", withAdminIdentity(adminHandler.RejectApplication))
	router.GET("/admin/apps/:id/documents", adminHandler.ListDocuments)
	router.GET("/admin/apps/:id/decisions", adminHandler.ListDecisions)
	router.POST("/apply", func(c *gin.Context) {
		c.Set(authcontext.UserIDContextKey, uuid.NewString())
		userHandler.Apply(c)
	})

	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	id := uuid.NewString()
	for _, tc := range []struct {
		method string
		path   string
		cookie string
		admin  bool
	}{
		{method: http.MethodGet, path: "/admin/apps", cookie: cookie, admin: true},
		{method: http.MethodGet, path: "/admin/apps/" + id, cookie: cookie, admin: true},
		{method: http.MethodPost, path: "/admin/apps/" + id + "/approve", cookie: cookie, admin: true},
		{method: http.MethodPost, path: "/admin/apps/" + id + "/reject", cookie: cookie, admin: true},
		{method: http.MethodGet, path: "/admin/apps/" + id + "/documents", cookie: cookie, admin: true},
		{method: http.MethodGet, path: "/admin/apps/" + id + "/decisions", cookie: cookie, admin: true},
		{method: http.MethodPost, path: "/apply", admin: false},
	} {
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(""))
		if tc.cookie != "" {
			name := auth.AdminAccessTokenCookieName
			if !tc.admin {
				name = auth.AccessTokenCookieName
			}
			req.AddCookie(&http.Cookie{Name: name, Value: tc.cookie})
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("%s %s: expected 500, got %d (body: %s)", tc.method, tc.path, rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "application repository failure") {
			t.Fatalf("%s %s: internal error details must never be exposed, got %s", tc.method, tc.path, rec.Body.String())
		}
	}
}

// uploadEvidenceRequest performs a multipart evidence upload. partType lets a
// test lie about the content type to prove the server validates magic bytes
// instead of trusting the client.
func uploadEvidenceRequest(t *testing.T, router http.Handler, cookie, kind, filename, partType string, content []byte) (*httptest.ResponseRecorder, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if kind != "" {
		if err := writer.WriteField("kind", kind); err != nil {
			t.Fatalf("write kind field: %v", err)
		}
	}
	if partType == "" {
		partType = "application/octet-stream"
	}
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	header.Set("Content-Type", partType)
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("create file part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write file part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, documentsRoute, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: cookie})
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec, rec.Body.String()
}

type documentPayload struct {
	ID               string `json:"id"`
	Kind             string `json:"kind"`
	OriginalFilename string `json:"original_filename"`
	ContentType      string `json:"content_type"`
	ByteSize         int64  `json:"byte_size"`
}

func parseDocumentPayload(t *testing.T, raw string) documentPayload {
	t.Helper()
	var payload struct {
		Data documentPayload `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("invalid JSON: %v (body: %s)", err, raw)
	}
	return payload.Data
}

// TestTrainerEvidenceLifecycle exercises the full applicant evidence flow:
// upload (with a hostile filename and a lying part content type), list,
// download and withdrawal while the application is still PENDING.
func TestTrainerEvidenceLifecycle(t *testing.T) {
	env := newTrainerApplicationEnv(t, true)
	user, _ := seedApplication(t, env.userRepo, env.applicationRepo, uniqueEmail())
	cookie := userToken(t, env.tokenSvc, user.ID, user.SessionVersion)

	rec, raw := uploadEvidenceRequest(t, env.router, cookie, "certification", "../../etc/passwd.pdf", "application/octet-stream", pdfPayload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	document := parseDocumentPayload(t, raw)
	if document.ID == "" {
		t.Fatalf("expected a document id, got %s", raw)
	}
	if document.Kind != "certification" {
		t.Fatalf("expected kind certification, got %q", document.Kind)
	}
	if document.OriginalFilename != "passwd.pdf" {
		t.Fatalf("expected a sanitized filename, got %q", document.OriginalFilename)
	}
	if document.ContentType != "application/pdf" {
		t.Fatalf("expected the sniffed content type, got %q", document.ContentType)
	}
	if document.ByteSize != int64(len(pdfPayload)) {
		t.Fatalf("expected byte_size %d, got %d", len(pdfPayload), document.ByteSize)
	}
	if strings.Contains(raw, "storage_key") || strings.Contains(raw, "sha256") {
		t.Fatalf("internal storage metadata must never be exposed, got %s", raw)
	}

	rec, raw = applyRequest(env.router, cookie, http.MethodGet, documentsRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, document.ID) {
		t.Fatalf("expected the uploaded document in the list, got %s", raw)
	}

	rec, raw = applyRequest(env.router, cookie, http.MethodGet, documentsRoute+"/"+document.ID+"/download", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("download: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !bytes.Equal(rec.Body.Bytes(), pdfPayload) {
		t.Fatalf("download: expected the stored bytes, got %d bytes", rec.Body.Len())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("download: expected validated content type, got %q", got)
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "attachment") {
		t.Fatalf("download: expected an attachment disposition, got %q", got)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("download: expected nosniff, got %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "no-store") {
		t.Fatalf("download: expected no-store, got %q", got)
	}
	body, err := io.ReadAll(rec.Body)
	if err != nil || !bytes.Equal(body, pdfPayload) {
		t.Fatalf("download: unexpected streamed body (err=%v, len=%d)", err, len(body))
	}

	rec, raw = applyRequest(env.router, cookie, http.MethodDelete, documentsRoute+"/"+document.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	rec, raw = applyRequest(env.router, cookie, http.MethodGet, documentsRoute, "")
	if rec.Code != http.StatusOK || strings.Contains(raw, document.ID) {
		t.Fatalf("after withdrawal the document must be gone from the list, got %d (body: %s)", rec.Code, raw)
	}
	rec, raw = applyRequest(env.router, cookie, http.MethodGet, documentsRoute+"/"+document.ID+"/download", "")
	if rec.Code != http.StatusNotFound || !strings.Contains(raw, `"code":"DOCUMENT_NOT_FOUND"`) {
		t.Fatalf("after withdrawal the download must be 404 DOCUMENT_NOT_FOUND, got %d (body: %s)", rec.Code, raw)
	}
}

func TestTrainerEvidenceUploadValidation(t *testing.T) {
	env := newTrainerApplicationEnv(t, true)
	user, _ := seedApplication(t, env.userRepo, env.applicationRepo, uniqueEmail())
	cookie := userToken(t, env.tokenSvc, user.ID, user.SessionVersion)

	rec, raw := uploadEvidenceRequest(t, env.router, cookie, "certification", "notes.pdf", "application/pdf", []byte("plain text pretending to be a PDF"))
	if rec.Code != http.StatusBadRequest || !strings.Contains(raw, `"code":"DOCUMENT_TYPE_NOT_ALLOWED"`) {
		t.Fatalf("disallowed content: expected 400 DOCUMENT_TYPE_NOT_ALLOWED, got %d (body: %s)", rec.Code, raw)
	}

	oversized := append(append([]byte{}, pdfPayload...), bytes.Repeat([]byte{'x'}, evidenceTestMaxBytes)...)
	rec, raw = uploadEvidenceRequest(t, env.router, cookie, "other", "huge.pdf", "application/pdf", oversized)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized: expected 413, got %d (body: %s)", rec.Code, raw)
	}

	rec, raw = uploadEvidenceRequest(t, env.router, cookie, "passport", "scan.pdf", "application/pdf", pdfPayload)
	if rec.Code != http.StatusBadRequest || !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("invalid kind: expected 400 VALIDATION_ERROR, got %d (body: %s)", rec.Code, raw)
	}

	rec, raw = uploadEvidenceRequest(t, env.router, "", "certification", "scan.pdf", "application/pdf", pdfPayload)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated upload: expected 401, got %d (body: %s)", rec.Code, raw)
	}

	// A user without any application has nothing to attach evidence to.
	other := seedLoginUser(t, env.userRepo, uniqueEmail(), "Password123!")
	rec, raw = uploadEvidenceRequest(t, env.router, userToken(t, env.tokenSvc, other.ID, other.SessionVersion), "certification", "scan.pdf", "application/pdf", pdfPayload)
	if rec.Code != http.StatusNotFound || !strings.Contains(raw, `"code":"APPLICATION_NOT_FOUND"`) {
		t.Fatalf("no application: expected 404 APPLICATION_NOT_FOUND, got %d (body: %s)", rec.Code, raw)
	}
}

// TestTrainerEvidenceLockedAfterApproval verifies the submission lock: once the
// application is reviewed, evidence can no longer be added or withdrawn.
func TestTrainerEvidenceLockedAfterApproval(t *testing.T) {
	env := newTrainerApplicationEnv(t, true)
	user, application := seedApplication(t, env.userRepo, env.applicationRepo, uniqueEmail())
	cookie := userToken(t, env.tokenSvc, user.ID, user.SessionVersion)
	adminCookie := adminToken(t, env.tokenSvc, config.Admin1ID)

	rec, raw := uploadEvidenceRequest(t, env.router, cookie, "identification", "id.pdf", "application/pdf", pdfPayload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload: expected 201, got %d (body: %s)", rec.Code, raw)
	}
	document := parseDocumentPayload(t, raw)

	rec, raw = adminApplicationRequest(env.router, adminCookie, http.MethodPost, adminApplicationsRoute+"/"+application.ID+"/approve")
	if rec.Code != http.StatusOK {
		t.Fatalf("approve: expected 200, got %d (body: %s)", rec.Code, raw)
	}

	rec, raw = uploadEvidenceRequest(t, env.router, cookie, "other", "extra.pdf", "application/pdf", pdfPayload)
	if rec.Code != http.StatusConflict || !strings.Contains(raw, `"code":"APPLICATION_STATE_CONFLICT"`) {
		t.Fatalf("upload after approval: expected 409 APPLICATION_STATE_CONFLICT, got %d (body: %s)", rec.Code, raw)
	}
	rec, raw = applyRequest(env.router, cookie, http.MethodDelete, documentsRoute+"/"+document.ID, "")
	if rec.Code != http.StatusConflict || !strings.Contains(raw, `"code":"APPLICATION_STATE_CONFLICT"`) {
		t.Fatalf("delete after approval: expected 409 APPLICATION_STATE_CONFLICT, got %d (body: %s)", rec.Code, raw)
	}
}

// TestTrainerEvidenceIsolation verifies one applicant can never touch another
// applicant's evidence.
func TestTrainerEvidenceIsolation(t *testing.T) {
	env := newTrainerApplicationEnv(t, true)
	first, _ := seedApplication(t, env.userRepo, env.applicationRepo, uniqueEmail())
	second, _ := seedApplication(t, env.userRepo, env.applicationRepo, uniqueEmail())
	firstCookie := userToken(t, env.tokenSvc, first.ID, first.SessionVersion)
	secondCookie := userToken(t, env.tokenSvc, second.ID, second.SessionVersion)

	rec, raw := uploadEvidenceRequest(t, env.router, firstCookie, "insurance", "policy.pdf", "application/pdf", pdfPayload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload: expected 201, got %d (body: %s)", rec.Code, raw)
	}
	document := parseDocumentPayload(t, raw)

	rec, raw = applyRequest(env.router, secondCookie, http.MethodGet, documentsRoute+"/"+document.ID+"/download", "")
	if rec.Code != http.StatusNotFound || !strings.Contains(raw, `"code":"DOCUMENT_NOT_FOUND"`) {
		t.Fatalf("foreign download: expected 404 DOCUMENT_NOT_FOUND, got %d (body: %s)", rec.Code, raw)
	}
	rec, raw = applyRequest(env.router, secondCookie, http.MethodDelete, documentsRoute+"/"+document.ID, "")
	if rec.Code != http.StatusNotFound || !strings.Contains(raw, `"code":"DOCUMENT_NOT_FOUND"`) {
		t.Fatalf("foreign delete: expected 404 DOCUMENT_NOT_FOUND, got %d (body: %s)", rec.Code, raw)
	}
	rec, _ = applyRequest(env.router, firstCookie, http.MethodGet, documentsRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatal("the owner's document must survive a foreign delete attempt")
	}
}

// TestAdminEvidenceReviewSurfaces covers the reviewer side: document listing
// and download scoped to the application, the recorded reject reason, and
// cross-application isolation.
func TestAdminEvidenceReviewSurfaces(t *testing.T) {
	env := newTrainerApplicationEnv(t, true)
	user, application := seedApplication(t, env.userRepo, env.applicationRepo, uniqueEmail())
	other, otherApplication := seedApplication(t, env.userRepo, env.applicationRepo, uniqueEmail())
	userCookie := userToken(t, env.tokenSvc, user.ID, user.SessionVersion)
	otherCookie := userToken(t, env.tokenSvc, other.ID, other.SessionVersion)
	adminCookie := adminToken(t, env.tokenSvc, config.Admin2ID)

	rec, raw := uploadEvidenceRequest(t, env.router, userCookie, "certification", "cert.pdf", "application/pdf", pdfPayload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload pdf: expected 201, got %d (body: %s)", rec.Code, raw)
	}
	pdfDocument := parseDocumentPayload(t, raw)

	rec, raw = uploadEvidenceRequest(t, env.router, userCookie, "identification", "photo.jpg", "text/plain", jpegPayload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload jpeg: expected 201, got %d (body: %s)", rec.Code, raw)
	}
	jpegDocument := parseDocumentPayload(t, raw)
	if jpegDocument.ContentType != "image/jpeg" {
		t.Fatalf("expected the sniffed image/jpeg type, got %q", jpegDocument.ContentType)
	}

	rec, raw = uploadEvidenceRequest(t, env.router, otherCookie, "other", "foreign.pdf", "application/pdf", pdfPayload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload foreign: expected 201, got %d (body: %s)", rec.Code, raw)
	}
	foreignDocument := parseDocumentPayload(t, raw)

	rec, raw = adminApplicationRequest(env.router, adminCookie, http.MethodGet, adminApplicationsRoute+"/"+application.ID+"/documents")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin list: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, pdfDocument.ID) || !strings.Contains(raw, jpegDocument.ID) {
		t.Fatalf("expected both documents in the admin list, got %s", raw)
	}
	if strings.Contains(raw, foreignDocument.ID) {
		t.Fatalf("documents of another application must never be listed, got %s", raw)
	}
	if strings.Contains(raw, "storage_key") {
		t.Fatalf("admin list must not expose storage keys, got %s", raw)
	}

	rec, raw = adminApplicationRequest(env.router, adminCookie, http.MethodGet, adminApplicationsRoute+"/"+application.ID+"/documents/"+pdfDocument.ID+"/download")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin download: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !bytes.Equal(rec.Body.Bytes(), pdfPayload) {
		t.Fatal("admin download must stream the stored bytes")
	}
	rec, raw = adminApplicationRequest(env.router, adminCookie, http.MethodGet, adminApplicationsRoute+"/"+application.ID+"/documents/"+foreignDocument.ID+"/download")
	if rec.Code != http.StatusNotFound || !strings.Contains(raw, `"code":"DOCUMENT_NOT_FOUND"`) {
		t.Fatalf("cross-application download: expected 404 DOCUMENT_NOT_FOUND, got %d (body: %s)", rec.Code, raw)
	}

	rec, raw = adminApplicationRequest(env.router, adminCookie, http.MethodPost, adminApplicationsRoute+"/"+application.ID+"/reject")
	if rec.Code != http.StatusOK {
		t.Fatalf("reject: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	_ = otherApplication

	rec, raw = adminApplicationRequest(env.router, adminCookie, http.MethodGet, adminApplicationsRoute+"/"+application.ID+"/decisions")
	if rec.Code != http.StatusOK {
		t.Fatalf("decisions: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, config.Admin2ID) || !strings.Contains(raw, `"to_status":"REJECTED"`) || !strings.Contains(raw, `"from_status":"PENDING"`) {
		t.Fatalf("expected the recorded rejection decision, got %s", raw)
	}
}

// TestAdminRejectReasonRecorded verifies the optional reject reason is stored
// and echoed in the decision history, and that an oversized reason is rejected
// without changing the application.
func TestAdminRejectReasonRecorded(t *testing.T) {
	env := newTrainerApplicationEnv(t, true)
	_, application := seedApplication(t, env.userRepo, env.applicationRepo, uniqueEmail())
	adminCookie := adminToken(t, env.tokenSvc, config.Admin1ID)

	rec, raw := adminApplicationJSONRequest(env.router, adminCookie, http.MethodPost, adminApplicationsRoute+"/"+application.ID+"/reject", `{"reason":"Blurred identification document."}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("reject: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	rec, raw = adminApplicationRequest(env.router, adminCookie, http.MethodGet, adminApplicationsRoute+"/"+application.ID+"/decisions")
	if rec.Code != http.StatusOK {
		t.Fatalf("decisions: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"reason":"Blurred identification document."`) {
		t.Fatalf("expected the recorded reason, got %s", raw)
	}

	_, second := seedApplication(t, env.userRepo, env.applicationRepo, uniqueEmail())
	oversized := `{"reason":"` + strings.Repeat("x", 501) + `"}`
	rec, raw = adminApplicationJSONRequest(env.router, adminCookie, http.MethodPost, adminApplicationsRoute+"/"+second.ID+"/reject", oversized)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("oversized reason: expected 400, got %d (body: %s)", rec.Code, raw)
	}
	persisted, err := env.applicationRepo.FindByID(context.Background(), second.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if persisted.Status != models.ApplicationStatusPending {
		t.Fatalf("a rejected review must leave the application PENDING, got %q", persisted.Status)
	}
}
