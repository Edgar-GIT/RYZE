package auth_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

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
	"ryze/backend/services/token"
	"ryze/backend/services/workout_history"
)

const (
	completeWorkoutRoute = "/api/v1/me/workouts/%s/complete"
	historyRoute         = "/api/v1/me/workouts/history"
)

// newWorkoutHistoryTestRouter wires the workout completion and history
// endpoints behind the real Authenticate middleware, backed by a database
// transaction so created records are rolled back.
func newWorkoutHistoryTestRouter(t *testing.T) (*gin.Engine, repositories.UserRepository, *gorm.DB, token.Service) {
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
	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)

	historyRepo := repositories.NewWorkoutHistoryRepository(tx)
	service := workout_history.NewService(historyRepo)
	handler := auth.NewWorkoutHistoryHandler(service)

	router := gin.New()
	me := router.Group("/api/v1/me")
	me.Use(middleware.Authenticate(tokenSvc, userRepo))
	me.POST("/workouts/:workoutID/complete", handler.CompleteWorkout)
	me.GET("/workouts/history", handler.ListHistory)

	return router, userRepo, tx, tokenSvc
}

// newWorkoutHistoryHandlerRouter mounts only the handler with a pre-set
// authentication-context identity, so the handler's own error mapping and
// identity forwarding can be tested without the full middleware chain. nil
// identity simulates a missing context.
func newWorkoutHistoryHandlerRouter(svc workout_history.Service, identity any) *gin.Engine {
	handler := auth.NewWorkoutHistoryHandler(svc)
	router := gin.New()
	me := router.Group("/api/v1/me")
	me.Use(func(c *gin.Context) {
		if identity != nil {
			c.Set(authcontext.UserIDContextKey, identity)
		}
		c.Next()
	})
	me.POST("/workouts/:workoutID/complete", handler.CompleteWorkout)
	me.GET("/workouts/history", handler.ListHistory)
	return router
}

// stubWorkoutHistoryService is a scripted fake used to exercise the handler's
// error mapping and identity forwarding without touching the database.
type stubWorkoutHistoryService struct {
	entry      *workout_history.HistoryEntry
	listResult workout_history.ListHistoryResult
	err        error
	gotUser    string
	gotWorkout string
	gotPage    int
	gotLimit   int
}

func (s *stubWorkoutHistoryService) CompleteWorkout(_ context.Context, userID, workoutID string) (*workout_history.HistoryEntry, error) {
	s.gotUser = userID
	s.gotWorkout = workoutID
	return s.entry, s.err
}

func (s *stubWorkoutHistoryService) ListHistory(_ context.Context, userID string, page, limit int) (workout_history.ListHistoryResult, error) {
	s.gotUser = userID
	s.gotPage = page
	s.gotLimit = limit
	return s.listResult, s.err
}

// seedClientAssignedWorkout builds a trainer-owned program with one week and
// one workout, assigns it to the given client user and returns the program and
// the workout.
func seedClientAssignedWorkout(t *testing.T, tx *gorm.DB, trainerID, clientUserID, programName string) (*models.Program, *models.ProgramWorkout) {
	t.Helper()
	ctx := context.Background()

	clientRepo := repositories.NewTrainerClientRepository(tx)
	programRepo := repositories.NewProgramRepository(tx)
	assignmentRepo := repositories.NewProgramAssignmentRepository(tx)
	weekRepo := repositories.NewProgramWeekRepository(tx)
	workoutRepo := repositories.NewProgramWorkoutRepository(tx)

	program := &models.Program{TrainerID: trainerID, Name: programName, Type: models.ProgramTypePremium, Status: models.ProgramStatusDraft}
	if err := programRepo.Create(ctx, program); err != nil {
		t.Fatalf("seed program: %v", err)
	}

	if err := clientRepo.Create(ctx, &models.TrainerClient{TrainerID: trainerID, UserID: clientUserID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}

	week := &models.ProgramWeek{}
	if err := weekRepo.Create(ctx, trainerID, program.ID, week); err != nil {
		t.Fatalf("seed week: %v", err)
	}

	workout := &models.ProgramWorkout{}
	if err := workoutRepo.Create(ctx, trainerID, program.ID, week.ID, workout); err != nil {
		t.Fatalf("seed workout: %v", err)
	}

	if err := assignmentRepo.Create(ctx, trainerID, clientUserID, program.ID, &models.ProgramAssignment{}); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}

	return program, workout
}

func TestWorkoutCompleteSuccess(t *testing.T) {
	router, userRepo, tx, tokenSvc := newWorkoutHistoryTestRouter(t)
	trainer := seedTrainerForUser(t, repositories.NewTrainerRepository(tx), seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	_, workout := seedClientAssignedWorkout(t, tx, trainer.ID, clientUser.ID, "Strength Builder")

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, sprintfPath(completeWorkoutRoute, workout.ID), "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if id, _ := data["id"].(string); id == "" {
		t.Fatal("expected a generated history entry id")
	}
	if programWorkoutID, _ := data["program_workout_id"].(string); programWorkoutID != workout.ID {
		t.Fatalf("expected program_workout_id %q, got %q", workout.ID, programWorkoutID)
	}
	if completedAt, _ := data["completed_at"].(string); completedAt == "" {
		t.Fatal("expected a completion timestamp")
	}
	if _, exists := data["user_id"]; exists {
		t.Fatal("a history entry must never expose the owning user id")
	}
	if _, exists := data["deleted_at"]; exists {
		t.Fatal("a history entry must never expose deletion markers")
	}
}

func TestWorkoutHistoryReadAfterCompletion(t *testing.T) {
	router, userRepo, tx, tokenSvc := newWorkoutHistoryTestRouter(t)
	trainer := seedTrainerForUser(t, repositories.NewTrainerRepository(tx), seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	_, workout := seedClientAssignedWorkout(t, tx, trainer.ID, clientUser.ID, "Strength Builder")

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	for i := 0; i < 2; i++ {
		rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, sprintfPath(completeWorkoutRoute, workout.ID), "")
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 on completion %d, got %d (body: %s)", i+1, rec.Code, raw)
		}
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, historyRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	entries, _ := data["entries"].([]any)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	pagination, _ := data["pagination"].(map[string]any)
	if total, _ := pagination["total"].(float64); total != 2 {
		t.Fatalf("expected pagination total 2, got %v", pagination["total"])
	}
	first, _ := entries[0].(map[string]any)
	if programWorkoutID, _ := first["program_workout_id"].(string); programWorkoutID != workout.ID {
		t.Fatalf("expected program_workout_id %q, got %q", workout.ID, programWorkoutID)
	}
	if _, exists := first["user_id"]; exists {
		t.Fatal("a history entry must never expose the owning user id")
	}
}

func TestWorkoutCompleteUnauthenticated(t *testing.T) {
	router, _, _, _ := newWorkoutHistoryTestRouter(t)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, sprintfPath(completeWorkoutRoute, uuid.NewString()), "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}

	rec, _, raw = trainerClientsRequest(router, "", http.MethodGet, historyRoute, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestWorkoutCompleteNoAssignment(t *testing.T) {
	router, userRepo, _, tokenSvc := newWorkoutHistoryTestRouter(t)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, sprintfPath(completeWorkoutRoute, uuid.NewString()), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"WORKOUT_NOT_FOUND"`) {
		t.Fatalf("expected WORKOUT_NOT_FOUND, got %s", raw)
	}
}

func TestWorkoutCompleteForeignWorkout(t *testing.T) {
	router, userRepo, tx, tokenSvc := newWorkoutHistoryTestRouter(t)
	trainerRepo := repositories.NewTrainerRepository(tx)

	trainerA := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	trainerB := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientA := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	clientB := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	_, workoutA := seedClientAssignedWorkout(t, tx, trainerA.ID, clientA.ID, "Program A")
	_, _ = seedClientAssignedWorkout(t, tx, trainerB.ID, clientB.ID, "Program B")

	jwtA, err := tokenSvc.GenerateAccessToken(clientA.ID, clientA.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken A: %v", err)
	}
	jwtB, err := tokenSvc.GenerateAccessToken(clientB.ID, clientB.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken B: %v", err)
	}

	// Client B can never complete client A's workout: a foreign workout is
	// indistinguishable from an unknown one.
	rec, _, raw := trainerClientsRequest(router, jwtB, http.MethodPost, sprintfPath(completeWorkoutRoute, workoutA.ID), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"WORKOUT_NOT_FOUND"`) {
		t.Fatalf("expected WORKOUT_NOT_FOUND, got %s", raw)
	}

	// Client A completes their own workout and the history never leaks it to B.
	if rec, _, raw := trainerClientsRequest(router, jwtA, http.MethodPost, sprintfPath(completeWorkoutRoute, workoutA.ID), ""); rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for A, got %d (body: %s)", rec.Code, raw)
	}

	rec, data, raw := trainerClientsRequest(router, jwtB, http.MethodGet, historyRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for B history, got %d (body: %s)", rec.Code, raw)
	}
	entries, _ := data["entries"].([]any)
	if len(entries) != 0 {
		t.Fatalf("client B must never see client A's history, got %d entries", len(entries))
	}
}

func TestWorkoutCompleteIgnoresClientSuppliedIdentity(t *testing.T) {
	router, userRepo, tx, tokenSvc := newWorkoutHistoryTestRouter(t)
	trainer := seedTrainerForUser(t, repositories.NewTrainerRepository(tx), seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	otherUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	_, workout := seedClientAssignedWorkout(t, tx, trainer.ID, clientUser.ID, "Strength Builder")

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	// A client-supplied user_id, trainer_id or client_id in the query must be
	// ignored: completion is always attributed to the authentication-context
	// identity and verified against that user's assigned program.
	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost,
		sprintfPath(completeWorkoutRoute, workout.ID)+"?user_id="+otherUser.ID+"&trainer_id="+uuid.NewString()+"&client_id="+otherUser.ID, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}

	// The entry is attributed to the authenticated user, not the spoofed one.
	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, historyRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	entries, _ := data["entries"].([]any)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for the authenticated user, got %d", len(entries))
	}
}

func TestWorkoutHistoryHandlerForwardsContextIdentity(t *testing.T) {
	identity := uuid.NewString()
	svc := &stubWorkoutHistoryService{
		entry: &workout_history.HistoryEntry{
			ID:               uuid.NewString(),
			ProgramWorkoutID: uuid.NewString(),
			CompletedAt:      time.Now().UTC(),
		},
	}
	router := newWorkoutHistoryHandlerRouter(svc, identity)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, sprintfPath(completeWorkoutRoute, uuid.NewString()), "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotUser != identity {
		t.Fatalf("expected context user %q, got %q", identity, svc.gotUser)
	}

	router2 := newWorkoutHistoryHandlerRouter(svc, identity)
	rec2, _, _ := trainerClientsRequest(router2, "", http.MethodGet, historyRoute, "")
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec2.Code)
	}
	if svc.gotUser != identity {
		t.Fatalf("expected context user %q, got %q", identity, svc.gotUser)
	}
}

func TestWorkoutHistoryHandlerMissingContext(t *testing.T) {
	router := newWorkoutHistoryHandlerRouter(&stubWorkoutHistoryService{}, nil)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, sprintfPath(completeWorkoutRoute, uuid.NewString()), "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}

	rec, _, raw = trainerClientsRequest(router, "", http.MethodGet, historyRoute, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestWorkoutHistoryHandlerErrorMapping(t *testing.T) {
	identity := uuid.NewString()

	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid input", err: workout_history.ErrInvalidInput, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
		{name: "workout not found", err: workout_history.ErrWorkoutNotFound, status: http.StatusNotFound, code: "WORKOUT_NOT_FOUND"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubWorkoutHistoryService{err: tc.err}
			router := newWorkoutHistoryHandlerRouter(svc, identity)

			rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, sprintfPath(completeWorkoutRoute, uuid.NewString()), "")
			if rec.Code != tc.status {
				t.Fatalf("expected %d, got %d (body: %s)", tc.status, rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"`+tc.code+`"`) {
				t.Fatalf("expected code %s, got %s", tc.code, raw)
			}
		})
	}
}

func TestWorkoutHistoryHandlerRepositoryFailureNotExposed(t *testing.T) {
	svc := &stubWorkoutHistoryService{err: errLoginRepoFailure}
	router := newWorkoutHistoryHandlerRouter(svc, uuid.NewString())

	rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, sprintfPath(completeWorkoutRoute, uuid.NewString()), "")
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

func TestWorkoutHistoryHandlerInvalidPagination(t *testing.T) {
	router := newWorkoutHistoryHandlerRouter(&stubWorkoutHistoryService{}, uuid.NewString())

	// Non-integer page/limit values are rejected by the handler; out-of-range
	// values are validated by the service (covered by service tests).
	for _, query := range []string{"?page=abc", "?limit=abc"} {
		rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, historyRoute+query, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("query %q: expected 400, got %d (body: %s)", query, rec.Code, raw)
		}
		if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
			t.Fatalf("query %q: expected VALIDATION_ERROR, got %s", query, raw)
		}
	}
}

func TestWorkoutCompleteNeverExposesSecrets(t *testing.T) {
	router, userRepo, tx, tokenSvc := newWorkoutHistoryTestRouter(t)
	trainer := seedTrainerForUser(t, repositories.NewTrainerRepository(tx), seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	_, workout := seedClientAssignedWorkout(t, tx, trainer.ID, clientUser.ID, "Strength Builder")

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, sprintfPath(completeWorkoutRoute, workout.ID), "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}

	for _, sensitive := range []string{
		jwtValue,
		"access_token",
		testSecret,
		"password_hash",
		"session_version",
		"deleted_at",
		"user_id",
		clientUser.Email,
	} {
		if strings.Contains(raw, sensitive) {
			t.Fatalf("response must never contain %q", sensitive)
		}
	}
}

func sprintfPath(format, arg string) string {
	return strings.Replace(format, "%s", arg, 1)
}
