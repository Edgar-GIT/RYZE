package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ryze/backend/api/auth"
	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/public_programs"
)

const (
	publicProgramsRoute        = "/api/v1/programs"
	publicProgramsRouteProgram = "/api/v1/programs/"
)

// newPublicProgramsTestRouter wires the public program endpoints backed by a
// database transaction so seeded records are rolled back. The endpoints require
// no authentication and are mounted exactly as routes.go mounts them.
func newPublicProgramsTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
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

	programRepo := repositories.NewProgramRepository(tx)
	service := public_programs.NewService(programRepo)
	handler := auth.NewPublicProgramsHandler(service)

	router := gin.New()
	v1 := router.Group("/api/v1")
	v1.GET("/programs", handler.ListPublishedPrograms)
	v1.GET("/programs/:programID", handler.GetPublishedProgram)

	return router, tx
}

// newPublicProgramsHandlerRouter mounts only the handler with a scripted
// service so the handler's error mapping and parameter forwarding can be tested
// without a database.
func newPublicProgramsHandlerRouter(svc public_programs.Service) *gin.Engine {
	handler := auth.NewPublicProgramsHandler(svc)
	router := gin.New()
	v1 := router.Group("/api/v1")
	v1.GET("/programs", handler.ListPublishedPrograms)
	v1.GET("/programs/:programID", handler.GetPublishedProgram)
	return router
}

// stubPublicProgramsService is a scripted fake used to exercise the handler's
// error mapping and parameter forwarding without touching the database.
type stubPublicProgramsService struct {
	program    *public_programs.Program
	listResult public_programs.ListProgramsResult
	err        error
	gotID      string
	gotPage    int
	gotLimit   int
}

func (s *stubPublicProgramsService) ListPublishedPrograms(_ context.Context, page, limit int) (public_programs.ListProgramsResult, error) {
	s.gotPage = page
	s.gotLimit = limit
	return s.listResult, s.err
}

func (s *stubPublicProgramsService) GetPublishedProgram(_ context.Context, programID string) (*public_programs.Program, error) {
	s.gotID = programID
	return s.program, s.err
}

func stubPublicProgramResponse() *public_programs.Program {
	return &public_programs.Program{
		ID:          "22222222-2222-2222-2222-222222222222",
		TrainerID:   "11111111-1111-1111-1111-111111111111",
		Name:        "Strength Builder",
		Description: "A 12 week strength program.",
		Type:        models.ProgramTypePremium,
		Status:      models.ProgramStatusPublished,
		CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
}

func seedPublishedProgramForTest(t *testing.T, db *gorm.DB, trainerID, name string) *models.Program {
	t.Helper()
	program := &models.Program{
		TrainerID:   trainerID,
		Name:        name,
		Description: "A training program.",
		Type:        models.ProgramTypePremium,
		Status:      models.ProgramStatusPublished,
	}
	if err := db.Create(program).Error; err != nil {
		t.Fatalf("seed published program: %v", err)
	}
	return program
}

func seedDraftProgramForTest(t *testing.T, db *gorm.DB, trainerID, name string) *models.Program {
	t.Helper()
	program := &models.Program{
		TrainerID:   trainerID,
		Name:        name,
		Description: "A training program.",
		Type:        models.ProgramTypePremium,
		Status:      models.ProgramStatusDraft,
	}
	if err := db.Create(program).Error; err != nil {
		t.Fatalf("seed draft program: %v", err)
	}
	return program
}

func doPublicProgramsSimpleRequest(router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func doPublicProgramsRequest(router http.Handler, method, path, body string) (*httptest.ResponseRecorder, map[string]any, string) {
	return trainerClientsRequest(router, "", method, path, body)
}

func TestPublicProgramsListSuccess(t *testing.T) {
	router, db := newPublicProgramsTestRouter(t)

	config.LoadEnvFile()
	cfg, _ := config.Load()
	mainDB, _ := database.Connect(cfg)
	tx := db.Begin()
	defer tx.Rollback()
	trainerRepo := repositories.NewTrainerRepository(mainDB)
	userRepo := repositories.NewUserRepository(mainDB)
	trainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	_ = seedPublishedProgramForTest(t, db, trainer.ID, "Published Program")
	_ = seedPublishedProgramForTest(t, db, trainer.ID, "Another Published")

	rec, data, raw := doPublicProgramsRequest(router, http.MethodGet, publicProgramsRoute+"?page=1&limit=10", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	programs, _ := data["programs"].([]any)
	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}
	pagination, _ := data["pagination"].(map[string]any)
	if total, _ := pagination["total"].(float64); total != 2 {
		t.Fatalf("expected total 2, got %v", total)
	}
	_ = tx
}

func TestPublicProgramsListExcludesDrafts(t *testing.T) {
	router, db := newPublicProgramsTestRouter(t)

	config.LoadEnvFile()
	cfg, _ := config.Load()
	mainDB, _ := database.Connect(cfg)
	tx := db.Begin()
	defer tx.Rollback()
	trainerRepo := repositories.NewTrainerRepository(mainDB)
	userRepo := repositories.NewUserRepository(mainDB)
	trainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	_ = seedPublishedProgramForTest(t, db, trainer.ID, "Published")
	_ = seedDraftProgramForTest(t, db, trainer.ID, "Draft Hidden")

	rec, data, raw := doPublicProgramsRequest(router, http.MethodGet, publicProgramsRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	programs, _ := data["programs"].([]any)
	if len(programs) != 1 {
		t.Fatalf("expected 1 published program, got %d", len(programs))
	}
	if name, _ := programs[0].(map[string]any)["name"].(string); name != "Published" {
		t.Fatalf("expected published program, got %q", name)
	}
	_ = tx
}

func TestPublicProgramsListInvalidPagination(t *testing.T) {
	router, _ := newPublicProgramsTestRouter(t)

	for _, query := range []string{"page=abc", "limit=abc", "page=abc&limit=abc"} {
		rec, _, raw := doPublicProgramsRequest(router, http.MethodGet, publicProgramsRoute+"?"+query, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %q, got %d (body: %s)", query, rec.Code, raw)
		}
		if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
			t.Fatalf("expected VALIDATION_ERROR for %q, got %s", query, raw)
		}
	}
}

func TestPublicProgramsGetSuccess(t *testing.T) {
	router, db := newPublicProgramsTestRouter(t)

	config.LoadEnvFile()
	cfg, _ := config.Load()
	mainDB, _ := database.Connect(cfg)
	tx := db.Begin()
	defer tx.Rollback()
	trainerRepo := repositories.NewTrainerRepository(mainDB)
	userRepo := repositories.NewUserRepository(mainDB)
	trainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	program := seedPublishedProgramForTest(t, db, trainer.ID, "Strength Builder")

	rec, data, raw := doPublicProgramsRequest(router, http.MethodGet, publicProgramsRouteProgram+program.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if id, _ := data["id"].(string); id != program.ID {
		t.Fatalf("expected program id %q, got %q", program.ID, id)
	}
	if name, _ := data["name"].(string); name != "Strength Builder" {
		t.Fatalf("expected name, got %q", name)
	}
	_ = tx
}

func TestPublicProgramsGetNotFound(t *testing.T) {
	router, _ := newPublicProgramsTestRouter(t)

	rec, _, raw := doPublicProgramsRequest(router, http.MethodGet, publicProgramsRouteProgram+"00000000-0000-0000-0000-000000000000", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"PROGRAM_NOT_FOUND"`) {
		t.Fatalf("expected PROGRAM_NOT_FOUND, got %s", raw)
	}
}

func TestPublicProgramsGetDraftHidden(t *testing.T) {
	router, db := newPublicProgramsTestRouter(t)

	config.LoadEnvFile()
	cfg, _ := config.Load()
	mainDB, _ := database.Connect(cfg)
	tx := db.Begin()
	defer tx.Rollback()
	trainerRepo := repositories.NewTrainerRepository(mainDB)
	userRepo := repositories.NewUserRepository(mainDB)
	trainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	draft := seedDraftProgramForTest(t, db, trainer.ID, "Draft Program")

	rec, _, raw := doPublicProgramsRequest(router, http.MethodGet, publicProgramsRouteProgram+draft.ID, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for draft program, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "Draft Program") {
		t.Fatalf("response must never contain a draft program's data, got %s", raw)
	}
	_ = tx
}

func TestPublicProgramsGetSoftDeletedHidden(t *testing.T) {
	router, db := newPublicProgramsTestRouter(t)

	config.LoadEnvFile()
	cfg, _ := config.Load()
	mainDB, _ := database.Connect(cfg)
	tx := db.Begin()
	defer tx.Rollback()
	trainerRepo := repositories.NewTrainerRepository(mainDB)
	userRepo := repositories.NewUserRepository(mainDB)
	trainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	program := seedPublishedProgramForTest(t, db, trainer.ID, "To Be Deleted")

	if err := db.Delete(program).Error; err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	rec, _, raw := doPublicProgramsRequest(router, http.MethodGet, publicProgramsRouteProgram+program.ID, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for soft-deleted program, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "To Be Deleted") {
		t.Fatalf("response must never contain a soft-deleted program's data, got %s", raw)
	}
	_ = tx
}

func TestPublicProgramsGetInvalidProgramID(t *testing.T) {
	router, _ := newPublicProgramsTestRouter(t)

	rec, _, raw := doPublicProgramsRequest(router, http.MethodGet, publicProgramsRouteProgram+"not-a-uuid", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestPublicProgramsNeverRequireAuthentication(t *testing.T) {
	router, _ := newPublicProgramsTestRouter(t)

	cases := []struct {
		name   string
		method string
		path   string
	}{
		{name: "list", method: http.MethodGet, path: publicProgramsRoute},
		{name: "get", method: http.MethodGet, path: publicProgramsRouteProgram + "00000000-0000-0000-0000-000000000000"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, _, raw := doPublicProgramsRequest(router, tc.method, tc.path, "")
			if rec.Code == http.StatusUnauthorized {
				t.Fatalf("public endpoint must not require authentication, got 401 (body: %s)", raw)
			}
		})
	}
}

func TestPublicProgramsHandlerErrorMapping(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid input", err: public_programs.ErrInvalidInput, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
		{name: "program not found", err: public_programs.ErrProgramNotFound, status: http.StatusNotFound, code: "PROGRAM_NOT_FOUND"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubPublicProgramsService{err: tc.err}
			router := newPublicProgramsHandlerRouter(svc)

			rec, _, raw := doPublicProgramsRequest(router, http.MethodGet, publicProgramsRouteProgram+"00000000-0000-0000-0000-000000000000", "")
			if rec.Code != tc.status {
				t.Fatalf("expected %d, got %d (body: %s)", tc.status, rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"`+tc.code+`"`) {
				t.Fatalf("expected code %s, got %s", tc.code, raw)
			}
		})
	}
}

func TestPublicProgramsHandlerRepositoryFailureNotExposed(t *testing.T) {
	svc := &stubPublicProgramsService{err: errLoginRepoFailure}
	router := newPublicProgramsHandlerRouter(svc)

	rec, _, raw := doPublicProgramsRequest(router, http.MethodGet, publicProgramsRoute, "")
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

func TestPublicProgramsHandlerGetForwardsPathProgramID(t *testing.T) {
	svc := &stubPublicProgramsService{program: stubPublicProgramResponse()}
	router := newPublicProgramsHandlerRouter(svc)

	rec, _, raw := doPublicProgramsRequest(router, http.MethodGet, publicProgramsRouteProgram+"44444444-4444-4444-4444-444444444444", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotID != "44444444-4444-4444-4444-444444444444" {
		t.Fatalf("expected path program id forwarded, got %q", svc.gotID)
	}
}

func TestPublicProgramsHandlerListForwardsPagination(t *testing.T) {
	svc := &stubPublicProgramsService{
		listResult: public_programs.ListProgramsResult{
			Programs: []public_programs.Program{*stubPublicProgramResponse()},
			Total:    1,
			Page:     3,
			Limit:    5,
		},
	}
	router := newPublicProgramsHandlerRouter(svc)

	rec, data, raw := doPublicProgramsRequest(router, http.MethodGet, publicProgramsRoute+"?page=3&limit=5", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotPage != 3 || svc.gotLimit != 5 {
		t.Fatalf("expected page 3 limit 5, got %d/%d", svc.gotPage, svc.gotLimit)
	}
	programs, _ := data["programs"].([]any)
	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}
	pagination, _ := data["pagination"].(map[string]any)
	if total, _ := pagination["total"].(float64); total != 1 {
		t.Fatalf("expected total 1, got %v", total)
	}
	if page, _ := pagination["page"].(float64); page != 3 {
		t.Fatalf("expected page 3, got %v", page)
	}
}

func TestPublicProgramsHandlerInvalidPagination(t *testing.T) {
	router := newPublicProgramsHandlerRouter(&stubPublicProgramsService{})

	rec, _, raw := doPublicProgramsRequest(router, http.MethodGet, publicProgramsRoute+"?page=abc", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestPublicProgramsHandlerInvalidPathID(t *testing.T) {
	svc := &stubPublicProgramsService{err: public_programs.ErrInvalidInput}
	router := newPublicProgramsHandlerRouter(svc)

	rec, _, raw := doPublicProgramsRequest(router, http.MethodGet, publicProgramsRouteProgram+"not-a-uuid", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestPublicProgramsNeverExposesSecrets(t *testing.T) {
	svc := &stubPublicProgramsService{
		listResult: public_programs.ListProgramsResult{
			Programs: []public_programs.Program{*stubPublicProgramResponse()},
			Total:    1,
			Page:     1,
			Limit:    20,
		},
	}
	router := newPublicProgramsHandlerRouter(svc)

	rec, _, raw := doPublicProgramsRequest(router, http.MethodGet, publicProgramsRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	for _, sensitive := range []string{
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

func TestPublicProgramsFindsCrossTrainerPrograms(t *testing.T) {
	router, db := newPublicProgramsTestRouter(t)

	config.LoadEnvFile()
	cfg, _ := config.Load()
	mainDB, _ := database.Connect(cfg)
	tx := db.Begin()
	defer tx.Rollback()
	trainerRepo := repositories.NewTrainerRepository(mainDB)
	userRepo := repositories.NewUserRepository(mainDB)
	trainerA := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	trainerB := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	programA := seedPublishedProgramForTest(t, db, trainerA.ID, "Trainer A Program")
	programB := seedPublishedProgramForTest(t, db, trainerB.ID, "Trainer B Program")

	// Both programs should be accessible from the public catalog.
	rec, _, raw := doPublicProgramsRequest(router, http.MethodGet, publicProgramsRouteProgram+programA.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for trainer A program, got %d (body: %s)", rec.Code, raw)
	}

	rec, _, raw = doPublicProgramsRequest(router, http.MethodGet, publicProgramsRouteProgram+programB.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for trainer B program, got %d (body: %s)", rec.Code, raw)
	}

	// List should include both.
	rec, data, raw := doPublicProgramsRequest(router, http.MethodGet, publicProgramsRoute+"?page=1&limit=10", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	programs, _ := data["programs"].([]any)
	if len(programs) != 2 {
		t.Fatalf("expected 2 programs from both trainers, got %d", len(programs))
	}
	_ = tx
}
