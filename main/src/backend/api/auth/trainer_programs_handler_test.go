package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ryze/backend/api/auth"
	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/middleware"
	"ryze/backend/middleware/trainercontext"
	"ryze/backend/middleware/trainerroles"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/programs"
	"ryze/backend/services/token"
)

const (
	programsRoute        = "/api/v1/trainer/programs"
	programsRouteProgram = "/api/v1/trainer/programs/"
)

// newTrainerProgramsTestRouter wires the trainer program endpoints behind the
// real Authenticate, TrainerAuthenticate and RequireTrainerPermission
// middleware, backed by a database transaction so created records are rolled
// back. The required permissions can be customized to exercise the 403 path.
func newTrainerProgramsTestRouter(t *testing.T, permissions ...trainerroles.Permission) (*gin.Engine, repositories.UserRepository, repositories.TrainerRepository, repositories.ProgramRepository, token.Service) {
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
	programRepo := repositories.NewProgramRepository(tx)
	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)

	service := programs.NewService(programRepo, config.PricingConfig{MinProgramPriceMinorUnits: 100})
	handler := auth.NewTrainerProgramsHandler(service)

	router := gin.New()
	trainer := router.Group("/api/v1/trainer")
	trainer.Use(middleware.Authenticate(tokenSvc, userRepo))
	trainer.Use(middleware.TrainerAuthenticate(trainerRepo))
	trainer.GET("/programs", middleware.RequireTrainerPermission(permissions...), handler.ListPrograms)
	trainer.POST("/programs", middleware.RequireTrainerPermission(permissions...), handler.CreateProgram)
	trainer.GET("/programs/:programID", middleware.RequireTrainerPermission(permissions...), handler.GetProgram)
	trainer.PATCH("/programs/:programID", middleware.RequireTrainerPermission(permissions...), handler.UpdateProgram)
	trainer.POST("/programs/:programID/publish", middleware.RequireTrainerPermission(permissions...), handler.PublishProgram)
	trainer.DELETE("/programs/:programID", middleware.RequireTrainerPermission(permissions...), handler.DeleteProgram)

	return router, userRepo, trainerRepo, programRepo, tokenSvc
}

// newTrainerProgramsHandlerRouter mounts only the handler with a pre-set trainer
// context identity, so the handler's own error mapping can be tested without
// the full middleware chain. nil identity simulates a missing context.
func newTrainerProgramsHandlerRouter(svc programs.Service, identity any) *gin.Engine {
	handler := auth.NewTrainerProgramsHandler(svc)
	router := gin.New()
	trainer := router.Group("/api/v1/trainer")
	trainer.Use(func(c *gin.Context) {
		if identity != nil {
			c.Set(trainercontext.TrainerContextKey, identity)
		}
		c.Next()
	})
	trainer.GET("/programs", handler.ListPrograms)
	trainer.POST("/programs", handler.CreateProgram)
	trainer.GET("/programs/:programID", handler.GetProgram)
	trainer.PATCH("/programs/:programID", handler.UpdateProgram)
	trainer.POST("/programs/:programID/publish", handler.PublishProgram)
	trainer.DELETE("/programs/:programID", handler.DeleteProgram)
	return router
}

// stubTrainerProgramsService is a scripted fake used to exercise the handler's
// error mapping and identity forwarding without touching the database.
type stubTrainerProgramsService struct {
	program        *programs.Program
	listResult     programs.ListProgramsResult
	err            error
	gotTrainer     string
	gotProgramID   string
	gotPage        int
	gotLimit       int
	gotCreateInput programs.CreateProgramInput
	gotUpdateInput programs.UpdateProgramInput
}

func (s *stubTrainerProgramsService) CreateProgram(_ context.Context, trainerID string, input programs.CreateProgramInput) (*programs.Program, error) {
	s.gotTrainer = trainerID
	s.gotCreateInput = input
	return s.program, s.err
}

func (s *stubTrainerProgramsService) ListPrograms(_ context.Context, trainerID string, page, limit int) (programs.ListProgramsResult, error) {
	s.gotTrainer = trainerID
	s.gotPage = page
	s.gotLimit = limit
	return s.listResult, s.err
}

func (s *stubTrainerProgramsService) GetProgram(_ context.Context, trainerID, programID string) (*programs.Program, error) {
	s.gotTrainer = trainerID
	s.gotProgramID = programID
	return s.program, s.err
}

func (s *stubTrainerProgramsService) UpdateProgram(_ context.Context, trainerID, programID string, input programs.UpdateProgramInput) (*programs.Program, error) {
	s.gotTrainer = trainerID
	s.gotProgramID = programID
	s.gotUpdateInput = input
	return s.program, s.err
}

func (s *stubTrainerProgramsService) PublishProgram(_ context.Context, trainerID, programID string) (*programs.Program, error) {
	s.gotTrainer = trainerID
	s.gotProgramID = programID
	return s.program, s.err
}

func (s *stubTrainerProgramsService) DeleteProgram(_ context.Context, trainerID, programID string) error {
	s.gotTrainer = trainerID
	s.gotProgramID = programID
	return s.err
}

func (s *stubTrainerProgramsService) UpdateProgramPricing(_ context.Context, programID string, input programs.UpdatePricingInput) (*programs.Program, error) {
	s.gotProgramID = programID
	return s.program, s.err
}

func (s *stubTrainerProgramsService) GetProgramByID(_ context.Context, programID string) (*programs.Program, error) {
	s.gotProgramID = programID
	return s.program, s.err
}

func trainerProgramsRequest(router http.Handler, cookieValue, method, path, body string) (*httptest.ResponseRecorder, map[string]any, string) {
	return trainerClientsRequest(router, cookieValue, method, path, body)
}

func stubProgramResponse(trainerID string) *programs.Program {
	return &programs.Program{
		ID:          uuid.NewString(),
		TrainerID:   trainerID,
		Name:        "Strength Builder",
		Description: "A 12 week strength program.",
		Type:        models.ProgramTypePremium,
		Status:      models.ProgramStatusDraft,
		CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
}

// seedTestProgram creates a program owned by the given trainer through the
// repository, so the full middleware chain and the real service are exercised.
func seedTestProgram(t *testing.T, programRepo repositories.ProgramRepository, trainerID, name string) *models.Program {
	t.Helper()
	program := &models.Program{
		TrainerID:   trainerID,
		Name:        name,
		Description: "A training program.",
		Type:        models.ProgramTypePremium,
		Status:      models.ProgramStatusPublished,
	}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed program: %v", err)
	}
	return program
}

// seedTestDraftProgram creates a draft program owned by the given trainer.
func seedTestDraftProgram(t *testing.T, programRepo repositories.ProgramRepository, trainerID, name string) *models.Program {
	t.Helper()
	program := &models.Program{
		TrainerID:   trainerID,
		Name:        name,
		Description: "A training program.",
		Type:        models.ProgramTypePremium,
		Status:      models.ProgramStatusDraft,
	}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed draft program: %v", err)
	}
	return program
}

func TestTrainerProgramsCreateSuccess(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)

	body := `{"name":"Strength Builder","description":"A 12 week strength program.","type":"premium","status":"published"}`
	rec, data, raw := trainerProgramsRequest(router, jwtValue, http.MethodPost, programsRoute, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}

	if id, _ := data["id"].(string); id == "" {
		t.Fatal("expected program id")
	}
	if trainerID, _ := data["trainer_id"].(string); trainerID != trainer.ID {
		t.Fatalf("expected trainer_id %q, got %q", trainer.ID, trainerID)
	}
	if name, _ := data["name"].(string); name != "Strength Builder" {
		t.Fatalf("expected name, got %q", name)
	}
	if programType, _ := data["type"].(string); programType != "premium" {
		t.Fatalf("expected premium type, got %q", programType)
	}
	if status, _ := data["status"].(string); status != "published" {
		t.Fatalf("expected published status, got %q", status)
	}

	persisted, err := programRepo.FindByIDAndTrainer(context.Background(), trainer.ID, data["id"].(string))
	if err != nil {
		t.Fatalf("expected persisted program: %v", err)
	}
	if persisted.Name != "Strength Builder" {
		t.Fatalf("unexpected persisted program %+v", persisted)
	}
}

func TestTrainerProgramsCreateIgnoresClientSuppliedTrainer(t *testing.T) {
	router, userRepo, trainerRepo, _, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))

	// A client-supplied trainer_id in the body must be ignored: the program can
	// only ever be created for the authenticated trainer.
	body := `{"name":"Strength Builder","type":"free","trainer_id":"` + otherTrainer.ID + `"}`
	rec, data, raw := trainerProgramsRequest(router, jwtValue, http.MethodPost, programsRoute, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if trainerID, _ := data["trainer_id"].(string); trainerID != trainer.ID {
		t.Fatalf("authenticated trainer must own the program, got %q", trainerID)
	}
}

func TestTrainerProgramsCreateInvalidBody(t *testing.T) {
	router, userRepo, trainerRepo, _, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	_, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)

	cases := []struct {
		name string
		body string
	}{
		{name: "not json", body: `not json`},
		{name: "empty name", body: `{"name":"","type":"free"}`},
		{name: "blank name", body: `{"name":"   ","type":"free"}`},
		{name: "invalid type", body: `{"name":"P","type":"random"}`},
		{name: "invalid status", body: `{"name":"P","type":"free","status":"archived"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, _, raw := trainerProgramsRequest(router, jwtValue, http.MethodPost, programsRoute, tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
				t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
			}
		})
	}
}

func TestTrainerProgramsListSuccess(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	first := seedTestProgram(t, programRepo, trainer.ID, "Strength Builder")
	second := seedTestProgram(t, programRepo, trainer.ID, "Conditioning")

	rec, data, raw := trainerProgramsRequest(router, jwtValue, http.MethodGet, programsRoute+"?page=1&limit=10", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	programs, _ := data["programs"].([]any)
	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}
	firstResponse, _ := programs[0].(map[string]any)
	if id, _ := firstResponse["id"].(string); id != first.ID && id != second.ID {
		t.Fatalf("unexpected program id %q", id)
	}
	pagination, _ := data["pagination"].(map[string]any)
	if total, _ := pagination["total"].(float64); total != 2 {
		t.Fatalf("expected total 2, got %v", total)
	}
	if pages, _ := pagination["total_pages"].(float64); pages != 1 {
		t.Fatalf("expected total_pages 1, got %v", pages)
	}
}

func TestTrainerProgramsListIgnoresQueryIdentity(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	seedTestProgram(t, programRepo, trainer.ID, "My Program")
	seedTestProgram(t, programRepo, otherTrainer.ID, "Foreign Program")

	// A client-supplied trainer_id in the query must be ignored: only the
	// authenticated trainer's programs are ever listed.
	path := programsRoute + "?trainer_id=" + otherTrainer.ID
	rec, data, raw := trainerProgramsRequest(router, jwtValue, http.MethodGet, path, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	programs, _ := data["programs"].([]any)
	if len(programs) != 1 {
		t.Fatalf("expected only the authenticated trainer's programs, got %d", len(programs))
	}
	first, _ := programs[0].(map[string]any)
	if name, _ := first["name"].(string); name != "My Program" {
		t.Fatalf("expected own program, got %q", name)
	}
}

func TestTrainerProgramsListInvalidPagination(t *testing.T) {
	router, userRepo, trainerRepo, _, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	_, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)

	for _, query := range []string{"page=abc", "limit=abc", "page=abc&limit=abc"} {
		rec, _, raw := trainerProgramsRequest(router, jwtValue, http.MethodGet, programsRoute+"?"+query, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %q, got %d (body: %s)", query, rec.Code, raw)
		}
		if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
			t.Fatalf("expected VALIDATION_ERROR for %q, got %s", query, raw)
		}
	}
}

func TestTrainerProgramsGetSuccess(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedTestProgram(t, programRepo, trainer.ID, "Strength Builder")

	rec, data, raw := trainerProgramsRequest(router, jwtValue, http.MethodGet, programsRouteProgram+program.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if id, _ := data["id"].(string); id != program.ID {
		t.Fatalf("expected program id %q, got %q", program.ID, id)
	}
	if trainerID, _ := data["trainer_id"].(string); trainerID != trainer.ID {
		t.Fatalf("expected trainer_id %q, got %q", trainer.ID, trainerID)
	}
	if name, _ := data["name"].(string); name != "Strength Builder" {
		t.Fatalf("expected name, got %q", name)
	}
}

func TestTrainerProgramsGetIDOR(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)

	trainerA, _, jwtA := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	trainerB, _, jwtB := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	programA := seedTestProgram(t, programRepo, trainerA.ID, "Program A")
	programB := seedTestProgram(t, programRepo, trainerB.ID, "Program B")

	// Trainer A must never read trainer B's program.
	rec, _, raw := trainerProgramsRequest(router, jwtA, http.MethodGet, programsRouteProgram+programB.ID, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-trainer read, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "Program B") {
		t.Fatalf("response must never contain the foreign program's data, got %s", raw)
	}

	// Trainer B must never read trainer A's program.
	rec, _, raw = trainerProgramsRequest(router, jwtB, http.MethodGet, programsRouteProgram+programA.ID, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-trainer read, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "Program A") {
		t.Fatalf("response must never contain the foreign program's data, got %s", raw)
	}
}

func TestTrainerProgramsGetIgnoresQueryIdentity(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	program := seedTestProgram(t, programRepo, trainer.ID, "My Program")

	// A client-supplied trainer_id query value must be ignored: the program is
	// always resolved for the authenticated trainer only.
	path := programsRouteProgram + program.ID + "?trainer_id=" + otherTrainer.ID
	rec, data, raw := trainerProgramsRequest(router, jwtValue, http.MethodGet, path, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if trainerID, _ := data["trainer_id"].(string); trainerID != trainer.ID {
		t.Fatalf("authenticated trainer must own the program, got %q", trainerID)
	}
}

func TestTrainerProgramsGetIgnoresHeaderIdentity(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	program := seedTestProgram(t, programRepo, trainer.ID, "My Program")

	req := httptest.NewRequest(http.MethodGet, programsRouteProgram+program.ID, nil)
	req.Header.Set("X-Trainer-Id", otherTrainer.ID)
	req.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: jwtValue})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	raw, _ := json.Marshal(payload)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	data, _ := payload["data"].(map[string]any)
	if trainerID, _ := data["trainer_id"].(string); trainerID != trainer.ID {
		t.Fatalf("header trainer id must never change ownership, got %q", trainerID)
	}
}

func TestTrainerProgramsGetIgnoresBodyIdentity(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	program := seedTestProgram(t, programRepo, trainer.ID, "My Program")

	// A trainer_id in the body must be ignored: the program is always resolved
	// for the authenticated trainer only.
	body := `{"trainer_id":"` + otherTrainer.ID + `"}`
	rec, data, raw := trainerProgramsRequest(router, jwtValue, http.MethodGet, programsRouteProgram+program.ID, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if trainerID, _ := data["trainer_id"].(string); trainerID != trainer.ID {
		t.Fatalf("body trainer id must never change ownership, got %q", trainerID)
	}
}

func TestTrainerProgramsGetMissingProgram(t *testing.T) {
	router, userRepo, trainerRepo, _, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	_, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)

	rec, _, raw := trainerProgramsRequest(router, jwtValue, http.MethodGet, programsRouteProgram+uuid.NewString(), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown program, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"PROGRAM_NOT_FOUND"`) {
		t.Fatalf("expected PROGRAM_NOT_FOUND, got %s", raw)
	}
}

func TestTrainerProgramsGetInvalidProgramID(t *testing.T) {
	router, userRepo, trainerRepo, _, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	_, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)

	rec, _, raw := trainerProgramsRequest(router, jwtValue, http.MethodGet, programsRouteProgram+"not-a-uuid", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed program id, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestTrainerProgramsGetSoftDeletedProgram(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedTestProgram(t, programRepo, trainer.ID, "My Program")

	if err := programRepo.SoftDelete(context.Background(), trainer.ID, program.ID); err != nil {
		t.Fatalf("seed soft-deleted program: %v", err)
	}

	rec, _, raw := trainerProgramsRequest(router, jwtValue, http.MethodGet, programsRouteProgram+program.ID, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for soft-deleted program, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "My Program") {
		t.Fatalf("response must never contain a soft-deleted program's data, got %s", raw)
	}
}

func TestTrainerProgramsUpdateSuccess(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedTestProgram(t, programRepo, trainer.ID, "Strength Builder")

	body := `{"name":"Hypertrophy 101","status":"published"}`
	rec, data, raw := trainerProgramsRequest(router, jwtValue, http.MethodPatch, programsRouteProgram+program.ID, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if name, _ := data["name"].(string); name != "Hypertrophy 101" {
		t.Fatalf("expected updated name, got %q", name)
	}
	if trainerID, _ := data["trainer_id"].(string); trainerID != trainer.ID {
		t.Fatalf("expected trainer_id %q, got %q", trainer.ID, trainerID)
	}

	persisted, err := programRepo.FindByIDAndTrainer(context.Background(), trainer.ID, program.ID)
	if err != nil {
		t.Fatalf("expected persisted program: %v", err)
	}
	if persisted.Name != "Hypertrophy 101" || persisted.Status != models.ProgramStatusPublished {
		t.Fatalf("unexpected persisted program %+v", persisted)
	}
}

func TestTrainerProgramsUpdateIgnoresClientSuppliedTrainer(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	program := seedTestProgram(t, programRepo, trainer.ID, "Strength Builder")

	// trainer_id is immutable: a client-supplied value in the body is ignored.
	body := `{"name":"Hypertrophy 101","trainer_id":"` + otherTrainer.ID + `"}`
	rec, data, raw := trainerProgramsRequest(router, jwtValue, http.MethodPatch, programsRouteProgram+program.ID, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if trainerID, _ := data["trainer_id"].(string); trainerID != trainer.ID {
		t.Fatalf("authenticated trainer must keep ownership, got %q", trainerID)
	}
}

func TestTrainerProgramsUpdateIDOR(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)

	_, _, jwtA := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	trainerB, _, _ := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	programB := seedTestProgram(t, programRepo, trainerB.ID, "Program B")

	// Trainer A must never update trainer B's program.
	body := `{"name":"Hijacked"}`
	rec, _, raw := trainerProgramsRequest(router, jwtA, http.MethodPatch, programsRouteProgram+programB.ID, body)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-trainer update, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "Hijacked") {
		t.Fatalf("response must never contain the attempted update, got %s", raw)
	}
	persisted, err := programRepo.FindByIDAndTrainer(context.Background(), trainerB.ID, programB.ID)
	if err != nil {
		t.Fatalf("program must survive the rejected update: %v", err)
	}
	if persisted.Name == "Hijacked" {
		t.Fatal("cross-trainer update must never persist")
	}
}

func TestTrainerProgramsUpdateInvalidInput(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedTestProgram(t, programRepo, trainer.ID, "Strength Builder")

	cases := []struct {
		name string
		body string
	}{
		{name: "empty name", body: `{"name":""}`},
		{name: "blank name", body: `{"name":"   "}`},
		{name: "invalid type", body: `{"type":"random"}`},
		{name: "invalid status", body: `{"status":"archived"}`},
		{name: "empty update", body: `{}`},
		{name: "not json", body: `not json`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, _, raw := trainerProgramsRequest(router, jwtValue, http.MethodPatch, programsRouteProgram+program.ID, tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for %q, got %d (body: %s)", tc.name, rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
				t.Fatalf("expected VALIDATION_ERROR for %q, got %s", tc.name, raw)
			}
		})
	}
}

func TestTrainerProgramsDeleteSuccess(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedTestProgram(t, programRepo, trainer.ID, "Strength Builder")

	rec, _, raw := trainerProgramsRequest(router, jwtValue, http.MethodDelete, programsRouteProgram+program.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	// Only the program is soft-deleted; the trainer profile stays active.
	if _, err := programRepo.FindByIDAndTrainer(context.Background(), trainer.ID, program.ID); err == nil {
		t.Fatal("expected the program to be soft-deleted")
	}
	if _, err := trainerRepo.FindByID(context.Background(), trainer.ID); err != nil {
		t.Fatalf("trainer profile must survive: %v", err)
	}
}

func TestTrainerProgramsDeleteIDOR(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)

	_, _, jwtA := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	trainerB, _, _ := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	programB := seedTestProgram(t, programRepo, trainerB.ID, "Program B")

	// Trainer A must never delete trainer B's program.
	rec, _, raw := trainerProgramsRequest(router, jwtA, http.MethodDelete, programsRouteProgram+programB.ID, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-trainer delete, got %d (body: %s)", rec.Code, raw)
	}
	if _, err := programRepo.FindByIDAndTrainer(context.Background(), trainerB.ID, programB.ID); err != nil {
		t.Fatalf("program must survive the rejected delete: %v", err)
	}
}

func TestTrainerProgramsNotAuthenticated(t *testing.T) {
	router, _, _, _, _ := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "list", method: http.MethodGet, path: programsRoute},
		{name: "create", method: http.MethodPost, path: programsRoute, body: `{"name":"P","type":"free"}`},
		{name: "get", method: http.MethodGet, path: programsRouteProgram + uuid.NewString()},
		{name: "update", method: http.MethodPatch, path: programsRouteProgram + uuid.NewString(), body: `{"name":"P"}`},
		{name: "publish", method: http.MethodPost, path: programsRouteProgram + uuid.NewString() + "/publish"},
		{name: "delete", method: http.MethodDelete, path: programsRouteProgram + uuid.NewString()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, _, raw := trainerProgramsRequest(router, "", tc.method, tc.path, tc.body)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
				t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
			}
		})
	}
}

func TestTrainerProgramsAuthenticatedNonTrainer(t *testing.T) {
	router, userRepo, _, _, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	jwtValue, err := tokenSvc.GenerateAccessToken(user.ID, user.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	cases := []struct {
		name string
		path string
	}{
		{name: "list", path: programsRoute},
		{name: "get", path: programsRouteProgram + uuid.NewString()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, _, raw := trainerProgramsRequest(router, jwtValue, http.MethodGet, tc.path, "")
			if rec.Code != http.StatusForbidden {
				t.Fatalf("expected 403, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"FORBIDDEN"`) {
				t.Fatalf("expected FORBIDDEN, got %s", raw)
			}
		})
	}
}

func TestTrainerProgramsPermissionNotGranted(t *testing.T) {
	router, userRepo, trainerRepo, _, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.Permission("trainer.schedule"))
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	_ = trainer

	cases := []struct {
		name string
		path string
	}{
		{name: "list", path: programsRoute},
		{name: "get", path: programsRouteProgram + uuid.NewString()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, _, raw := trainerProgramsRequest(router, jwtValue, http.MethodGet, tc.path, "")
			if rec.Code != http.StatusForbidden {
				t.Fatalf("expected 403, got %d (body: %s)", rec.Code, raw)
			}
			if strings.Contains(raw, "trainer.schedule") {
				t.Fatalf("forbidden error must not reveal the permission, got %s", raw)
			}
		})
	}
}

func TestTrainerProgramsNeverExposesSecrets(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	seedTestProgram(t, programRepo, trainer.ID, "My Program")

	rec, _, raw := trainerProgramsRequest(router, jwtValue, http.MethodGet, programsRoute, "")
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
	} {
		if strings.Contains(raw, sensitive) {
			t.Fatalf("response must never contain %q", sensitive)
		}
	}
}

func TestTrainerProgramsHandlerForwardsContextIdentity(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	svc := &stubTrainerProgramsService{program: stubProgramResponse(identity.TrainerID)}
	router := newTrainerProgramsHandlerRouter(svc, identity)

	body := `{"name":"P","type":"free"}`
	rec, _, raw := trainerProgramsRequest(router, "", http.MethodPost, programsRoute, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotTrainer != identity.TrainerID {
		t.Fatalf("expected context trainer %q, got %q", identity.TrainerID, svc.gotTrainer)
	}
	if svc.gotCreateInput.Name != "P" || svc.gotCreateInput.Type != "free" {
		t.Fatalf("unexpected create input %+v", svc.gotCreateInput)
	}
}

func TestTrainerProgramsHandlerListForwardsContextIdentityAndPagination(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	svc := &stubTrainerProgramsService{
		listResult: programs.ListProgramsResult{
			Programs: []programs.Program{*stubProgramResponse(identity.TrainerID)},
			Total:    1,
			Page:     2,
			Limit:    5,
		},
	}
	router := newTrainerProgramsHandlerRouter(svc, identity)

	rec, data, raw := trainerProgramsRequest(router, "", http.MethodGet, programsRoute+"?page=2&limit=5", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotTrainer != identity.TrainerID {
		t.Fatalf("expected context trainer %q, got %q", identity.TrainerID, svc.gotTrainer)
	}
	if svc.gotPage != 2 || svc.gotLimit != 5 {
		t.Fatalf("expected page 2 limit 5, got %d/%d", svc.gotPage, svc.gotLimit)
	}
	programs, _ := data["programs"].([]any)
	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}
	pagination, _ := data["pagination"].(map[string]any)
	if total, _ := pagination["total"].(float64); total != 1 {
		t.Fatalf("expected total 1, got %v", total)
	}
	if page, _ := pagination["page"].(float64); page != 2 {
		t.Fatalf("expected page 2, got %v", page)
	}
}

func TestTrainerProgramsHandlerGetForwardsPathProgramID(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	svc := &stubTrainerProgramsService{program: stubProgramResponse(identity.TrainerID)}
	router := newTrainerProgramsHandlerRouter(svc, identity)

	pathProgramID := uuid.NewString()
	rec, _, raw := trainerProgramsRequest(router, "", http.MethodGet, programsRouteProgram+pathProgramID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotTrainer != identity.TrainerID {
		t.Fatalf("expected context trainer %q, got %q", identity.TrainerID, svc.gotTrainer)
	}
	if svc.gotProgramID != pathProgramID {
		t.Fatalf("expected path program %q, got %q", pathProgramID, svc.gotProgramID)
	}
}

func TestTrainerProgramsHandlerUpdateForwardsPathProgramID(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	svc := &stubTrainerProgramsService{program: stubProgramResponse(identity.TrainerID)}
	router := newTrainerProgramsHandlerRouter(svc, identity)

	pathProgramID := uuid.NewString()
	body := `{"name":"New Name"}`
	rec, _, raw := trainerProgramsRequest(router, "", http.MethodPatch, programsRouteProgram+pathProgramID, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotTrainer != identity.TrainerID {
		t.Fatalf("expected context trainer %q, got %q", identity.TrainerID, svc.gotTrainer)
	}
	if svc.gotProgramID != pathProgramID {
		t.Fatalf("expected path program %q, got %q", pathProgramID, svc.gotProgramID)
	}
	if svc.gotUpdateInput.Name == nil || *svc.gotUpdateInput.Name != "New Name" {
		t.Fatalf("unexpected update input %+v", svc.gotUpdateInput)
	}
}

func TestTrainerProgramsHandlerDeleteForwardsPathProgramID(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	svc := &stubTrainerProgramsService{}
	router := newTrainerProgramsHandlerRouter(svc, identity)

	pathProgramID := uuid.NewString()
	rec, _, raw := trainerProgramsRequest(router, "", http.MethodDelete, programsRouteProgram+pathProgramID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotTrainer != identity.TrainerID {
		t.Fatalf("expected context trainer %q, got %q", identity.TrainerID, svc.gotTrainer)
	}
	if svc.gotProgramID != pathProgramID {
		t.Fatalf("expected path program %q, got %q", pathProgramID, svc.gotProgramID)
	}
}

func TestTrainerProgramsHandlerMissingContext(t *testing.T) {
	router := newTrainerProgramsHandlerRouter(&stubTrainerProgramsService{}, nil)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "list", method: http.MethodGet, path: programsRoute},
		{name: "create", method: http.MethodPost, path: programsRoute, body: `{"name":"P","type":"free"}`},
		{name: "get", method: http.MethodGet, path: programsRouteProgram + uuid.NewString()},
		{name: "update", method: http.MethodPatch, path: programsRouteProgram + uuid.NewString(), body: `{"name":"P"}`},
		{name: "publish", method: http.MethodPost, path: programsRouteProgram + uuid.NewString() + "/publish"},
		{name: "delete", method: http.MethodDelete, path: programsRouteProgram + uuid.NewString()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, _, raw := trainerProgramsRequest(router, "", tc.method, tc.path, tc.body)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
				t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
			}
		})
	}
}

func TestTrainerProgramsHandlerErrorMapping(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}

	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid input", err: programs.ErrInvalidInput, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
		{name: "program not found", err: programs.ErrProgramNotFound, status: http.StatusNotFound, code: "PROGRAM_NOT_FOUND"},
		{name: "program already published", err: programs.ErrProgramAlreadyPublished, status: http.StatusConflict, code: "PROGRAM_ALREADY_PUBLISHED"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubTrainerProgramsService{err: tc.err}
			router := newTrainerProgramsHandlerRouter(svc, identity)

			rec, _, raw := trainerProgramsRequest(router, "", http.MethodPost, programsRoute, `{"name":"P","type":"free"}`)
			if rec.Code != tc.status {
				t.Fatalf("expected %d, got %d (body: %s)", tc.status, rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"`+tc.code+`"`) {
				t.Fatalf("expected code %s, got %s", tc.code, raw)
			}
		})
	}
}

func TestTrainerProgramsHandlerRepositoryFailureNotExposed(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	svc := &stubTrainerProgramsService{err: errLoginRepoFailure}
	router := newTrainerProgramsHandlerRouter(svc, identity)

	rec, _, raw := trainerProgramsRequest(router, "", http.MethodPost, programsRoute, `{"name":"P","type":"free"}`)
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

func TestTrainerProgramsHandlerCreateInvalidJSON(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	router := newTrainerProgramsHandlerRouter(&stubTrainerProgramsService{}, identity)

	rec, _, raw := trainerProgramsRequest(router, "", http.MethodPost, programsRoute, `not json`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestTrainerProgramsHandlerUpdateInvalidJSON(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	router := newTrainerProgramsHandlerRouter(&stubTrainerProgramsService{}, identity)

	rec, _, raw := trainerProgramsRequest(router, "", http.MethodPatch, programsRouteProgram+uuid.NewString(), `not json`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestTrainerProgramsHandlerInvalidPagination(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	router := newTrainerProgramsHandlerRouter(&stubTrainerProgramsService{}, identity)

	rec, _, raw := trainerProgramsRequest(router, "", http.MethodGet, programsRoute+"?page=abc", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestTrainerProgramsPublishSuccess(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedTestDraftProgram(t, programRepo, trainer.ID, "Draft Program")

	rec, data, raw := trainerProgramsRequest(router, jwtValue, http.MethodPost, programsRouteProgram+program.ID+"/publish", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if status, _ := data["status"].(string); status != "published" {
		t.Fatalf("expected published status, got %q", status)
	}
	if trainerID, _ := data["trainer_id"].(string); trainerID != trainer.ID {
		t.Fatalf("expected trainer_id %q, got %q", trainer.ID, trainerID)
	}

	persisted, err := programRepo.FindByIDAndTrainer(context.Background(), trainer.ID, program.ID)
	if err != nil {
		t.Fatalf("expected persisted program: %v", err)
	}
	if persisted.Status != models.ProgramStatusPublished {
		t.Fatalf("expected persisted published status, got %q", persisted.Status)
	}
}

func TestTrainerProgramsPublishIDOR(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)

	trainerA, _, jwtA := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	trainerB, _, _ := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	draftB := seedTestDraftProgram(t, programRepo, trainerB.ID, "Trainer B Draft")

	// Trainer A must never publish trainer B's program.
	rec, _, raw := trainerProgramsRequest(router, jwtA, http.MethodPost, programsRouteProgram+draftB.ID+"/publish", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-trainer publish, got %d (body: %s)", rec.Code, raw)
	}

	// The program must remain draft.
	persisted, err := programRepo.FindByIDAndTrainer(context.Background(), trainerB.ID, draftB.ID)
	if err != nil {
		t.Fatalf("find persisted program: %v", err)
	}
	if persisted.Status != models.ProgramStatusDraft {
		t.Fatalf("cross-trainer publish must not change status, got %q", persisted.Status)
	}

	_ = trainerA
}

func TestTrainerProgramsPublishAlreadyPublished(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedTestProgram(t, programRepo, trainer.ID, "Already Published")

	rec, _, raw := trainerProgramsRequest(router, jwtValue, http.MethodPost, programsRouteProgram+program.ID+"/publish", "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"PROGRAM_ALREADY_PUBLISHED"`) {
		t.Fatalf("expected PROGRAM_ALREADY_PUBLISHED, got %s", raw)
	}
}

func TestTrainerProgramsPublishMissing(t *testing.T) {
	router, userRepo, trainerRepo, _, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	_, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)

	rec, _, raw := trainerProgramsRequest(router, jwtValue, http.MethodPost, programsRouteProgram+uuid.NewString()+"/publish", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"PROGRAM_NOT_FOUND"`) {
		t.Fatalf("expected PROGRAM_NOT_FOUND, got %s", raw)
	}
}

func TestTrainerProgramsPublishInvalidProgramID(t *testing.T) {
	router, userRepo, trainerRepo, _, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	_, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)

	rec, _, raw := trainerProgramsRequest(router, jwtValue, http.MethodPost, programsRouteProgram+"not-a-uuid/publish", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestTrainerProgramsPublishSoftDeleted(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, tokenSvc := newTrainerProgramsTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedTestDraftProgram(t, programRepo, trainer.ID, "To Be Deleted")

	if err := programRepo.SoftDelete(context.Background(), trainer.ID, program.ID); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	rec, _, raw := trainerProgramsRequest(router, jwtValue, http.MethodPost, programsRouteProgram+program.ID+"/publish", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for soft-deleted program, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"PROGRAM_NOT_FOUND"`) {
		t.Fatalf("expected PROGRAM_NOT_FOUND, got %s", raw)
	}
}

func TestTrainerProgramsHandlerPublishForwardsPathProgramID(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	publishedProgram := stubProgramResponse(identity.TrainerID)
	publishedProgram.Status = models.ProgramStatusPublished
	svc := &stubTrainerProgramsService{program: publishedProgram}
	router := newTrainerProgramsHandlerRouter(svc, identity)

	pathProgramID := uuid.NewString()
	rec, _, raw := trainerProgramsRequest(router, "", http.MethodPost, programsRouteProgram+pathProgramID+"/publish", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotTrainer != identity.TrainerID {
		t.Fatalf("expected context trainer %q, got %q", identity.TrainerID, svc.gotTrainer)
	}
	if svc.gotProgramID != pathProgramID {
		t.Fatalf("expected path program %q, got %q", pathProgramID, svc.gotProgramID)
	}
}
