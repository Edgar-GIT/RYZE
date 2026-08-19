package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"ryze/backend/api/auth"
	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/middleware"
	"ryze/backend/middleware/authcontext"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/entitlements"
	"ryze/backend/services/token"
)

const entitlementsRoute = "/api/v1/me/entitlements"

func newEntitlementsTestRouter(t *testing.T) (*gin.Engine, repositories.UserRepository, *gorm.DB, token.Service) {
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
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)

	svc := entitlements.NewService(entitlementRepo)
	handler := auth.NewEntitlementsHandler(svc)

	router := gin.New()
	me := router.Group("/api/v1/me")
	me.Use(middleware.Authenticate(tokenSvc, userRepo))
	me.GET("/entitlements", handler.ListEntitlements)

	return router, userRepo, tx, tokenSvc
}

func newEntitlementsHandlerRouter(svc entitlements.Service, identity any) *gin.Engine {
	handler := auth.NewEntitlementsHandler(svc)
	router := gin.New()
	me := router.Group("/api/v1/me")
	me.Use(func(c *gin.Context) {
		if identity != nil {
			c.Set(authcontext.UserIDContextKey, identity)
		}
		c.Next()
	})
	me.GET("/entitlements", handler.ListEntitlements)
	return router
}

type stubEntitlementsService struct {
	entitlements []entitlements.Entitlement
	err          error
	gotUser      string
}

func (s *stubEntitlementsService) ListEntitlements(_ context.Context, userID string) ([]entitlements.Entitlement, error) {
	s.gotUser = userID
	if s.err != nil {
		return nil, s.err
	}
	return s.entitlements, nil
}

func (s *stubEntitlementsService) CreateEntitlement(_ context.Context, userID, programID string) (*entitlements.Entitlement, error) {
	s.gotUser = userID
	if s.err != nil {
		return nil, s.err
	}
	return nil, nil
}

func (s *stubEntitlementsService) RevokeEntitlement(_ context.Context, userID, entitlementID string) error {
	s.gotUser = userID
	return s.err
}

func entitlementsRequest(router http.Handler, cookieValue, method, path, body string) (*httptest.ResponseRecorder, map[string]any, string) {
	var reqBody *bytes.Reader
	if body == "" {
		reqBody = bytes.NewReader(nil)
	} else {
		reqBody = bytes.NewReader([]byte(body))
	}

	req := httptest.NewRequest(method, path, reqBody)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookieValue != "" {
		req.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: cookieValue})
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		return rec, nil, ""
	}
	data, _ := payload["data"].(map[string]any)
	return rec, data, string(rec.Body.Bytes())
}

func seedEntitlementTrainerProgram(t *testing.T, tx *gorm.DB, trainerUser *models.User, programName string) *models.Program {
	t.Helper()
	ctx := context.Background()
	trainerRepo := repositories.NewTrainerRepository(tx)
	programRepo := repositories.NewProgramRepository(tx)

	trainer := seedTrainerForUser(t, trainerRepo, trainerUser)
	program := &models.Program{
		TrainerID: trainer.ID,
		Name:      programName,
		Type:      models.ProgramTypePremium,
		Status:    models.ProgramStatusPublished,
	}
	if err := programRepo.Create(ctx, program); err != nil {
		t.Fatalf("seed program: %v", err)
	}
	return program
}

func TestEntitlementsReadSuccess(t *testing.T) {
	router, userRepo, tx, tokenSvc := newEntitlementsTestRouter(t)
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	ctx := context.Background()

	trainerUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	program := seedEntitlementTrainerProgram(t, tx, trainerUser, "Strength Builder")

	ent := &models.Entitlement{}
	if err := entitlementRepo.Create(ctx, clientUser.ID, program.ID, ent); err != nil {
		t.Fatalf("seed entitlement: %v", err)
	}

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := entitlementsRequest(router, jwtValue, http.MethodGet, entitlementsRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, program.Name) {
		t.Fatalf("expected program name %q, got %s", program.Name, raw)
	}
	if !strings.Contains(raw, ent.ID) {
		t.Fatalf("expected entitlement id %q, got %s", ent.ID, raw)
	}

	// The owning trainer must never be exposed to the client.
	if strings.Contains(raw, trainerUser.ID) {
		t.Fatal("response must never expose the owning trainer user id")
	}
}

func TestEntitlementsReadEmptyList(t *testing.T) {
	router, userRepo, _, tokenSvc := newEntitlementsTestRouter(t)

	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := entitlementsRequest(router, jwtValue, http.MethodGet, entitlementsRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"data":[]`) {
		t.Fatalf("expected empty data array, got %s", raw)
	}
}

func TestEntitlementsReadUnauthenticated(t *testing.T) {
	router, _, _, _ := newEntitlementsTestRouter(t)

	rec, _, raw := entitlementsRequest(router, "", http.MethodGet, entitlementsRoute, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestEntitlementsReadIDOR(t *testing.T) {
	router, userRepo, tx, tokenSvc := newEntitlementsTestRouter(t)
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	ctx := context.Background()

	trainerUserA := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	trainerUserB := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	userA := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	userB := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	programA := seedEntitlementTrainerProgram(t, tx, trainerUserA, "Program A")
	programB := seedEntitlementTrainerProgram(t, tx, trainerUserB, "Program B")

	if err := entitlementRepo.Create(ctx, userA.ID, programA.ID, &models.Entitlement{}); err != nil {
		t.Fatalf("seed entitlement A: %v", err)
	}
	if err := entitlementRepo.Create(ctx, userB.ID, programB.ID, &models.Entitlement{}); err != nil {
		t.Fatalf("seed entitlement B: %v", err)
	}

	jwtA, err := tokenSvc.GenerateAccessToken(userA.ID, userA.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken A: %v", err)
	}
	jwtB, err := tokenSvc.GenerateAccessToken(userB.ID, userB.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken B: %v", err)
	}

	recA, _, rawA := entitlementsRequest(router, jwtA, http.MethodGet, entitlementsRoute, "")
	if recA.Code != http.StatusOK {
		t.Fatalf("expected 200 for A, got %d (body: %s)", recA.Code, rawA)
	}
	if !strings.Contains(rawA, "Program A") {
		t.Fatalf("expected Program A, got %s", rawA)
	}
	if strings.Contains(rawA, "Program B") {
		t.Fatalf("user A must never see Program B, got %s", rawA)
	}

	recB, _, rawB := entitlementsRequest(router, jwtB, http.MethodGet, entitlementsRoute, "")
	if recB.Code != http.StatusOK {
		t.Fatalf("expected 200 for B, got %d (body: %s)", recB.Code, rawB)
	}
	if !strings.Contains(rawB, "Program B") {
		t.Fatalf("expected Program B, got %s", rawB)
	}
	if strings.Contains(rawB, "Program A") {
		t.Fatalf("user B must never see Program A, got %s", rawB)
	}
}

func TestEntitlementsHandlerForwardsContextIdentity(t *testing.T) {
	identity := uuid.NewString()
	svc := &stubEntitlementsService{
		entitlements: []entitlements.Entitlement{
			{
				ID:        uuid.NewString(),
				ProgramID: uuid.NewString(),
				Program: entitlements.Program{
					ID:   uuid.NewString(),
					Name: "Strength Builder",
					Type: models.ProgramTypePremium,
				},
			},
		},
	}
	router := newEntitlementsHandlerRouter(svc, identity)

	rec, _, raw := entitlementsRequest(router, "", http.MethodGet, entitlementsRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotUser != identity {
		t.Fatalf("expected context user %q, got %q", identity, svc.gotUser)
	}
}

func TestEntitlementsHandlerMissingContext(t *testing.T) {
	router := newEntitlementsHandlerRouter(&stubEntitlementsService{}, nil)

	rec, _, raw := entitlementsRequest(router, "", http.MethodGet, entitlementsRoute, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestEntitlementsHandlerErrorMapping(t *testing.T) {
	identity := uuid.NewString()

	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid input", err: entitlements.ErrInvalidInput, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubEntitlementsService{err: tc.err}
			router := newEntitlementsHandlerRouter(svc, identity)

			rec, _, raw := entitlementsRequest(router, "", http.MethodGet, entitlementsRoute, "")
			if rec.Code != tc.status {
				t.Fatalf("expected %d, got %d (body: %s)", tc.status, rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"`+tc.code+`"`) {
				t.Fatalf("expected code %s, got %s", tc.code, raw)
			}
		})
	}
}

func TestEntitlementsHandlerRepositoryFailureNotExposed(t *testing.T) {
	svc := &stubEntitlementsService{err: errLoginRepoFailure}
	router := newEntitlementsHandlerRouter(svc, uuid.NewString())

	rec, _, raw := entitlementsRequest(router, "", http.MethodGet, entitlementsRoute, "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "repository failure") {
		t.Fatalf("internal error details must never be exposed, got %s", raw)
	}
	if !strings.Contains(raw, `"code":"INTERNAL_ERROR"`) {
		t.Fatalf("expected INTERNAL_ERROR, got %s", raw)
	}
}

func TestEntitlementsReadNeverExposesSecrets(t *testing.T) {
	router, userRepo, tx, tokenSvc := newEntitlementsTestRouter(t)
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	ctx := context.Background()

	trainerUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	program := seedEntitlementTrainerProgram(t, tx, trainerUser, "Strength Builder")

	if err := entitlementRepo.Create(ctx, clientUser.ID, program.ID, &models.Entitlement{}); err != nil {
		t.Fatalf("seed entitlement: %v", err)
	}

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := entitlementsRequest(router, jwtValue, http.MethodGet, entitlementsRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	for _, sensitive := range []string{
		jwtValue,
		"access_token",
		testSecret,
		"password_hash",
		"session_version",
		"deleted_at",
		"trainer_id",
		clientUser.Email,
	} {
		if strings.Contains(raw, sensitive) {
			t.Fatalf("response must never contain %q", sensitive)
		}
	}
}
