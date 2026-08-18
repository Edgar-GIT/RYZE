package auth_test

import (
	"context"
	"net/http"
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
	"ryze/backend/services/statistics"
	"ryze/backend/services/token"
	"ryze/backend/services/workout_history"
)

const statisticsRoute = "/api/v1/me/statistics"

// newStatisticsTestRouter wires the statistics endpoint behind the real
// Authenticate middleware, backed by a database transaction so created records
// are rolled back.
func newStatisticsTestRouter(t *testing.T) (*gin.Engine, repositories.UserRepository, *gorm.DB, token.Service) {
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

	statsRepo := repositories.NewStatisticsRepository(tx)
	statsService := statistics.NewService(statsRepo)
	statsHandler := auth.NewStatisticsHandler(statsService)

	historyRepo := repositories.NewWorkoutHistoryRepository(tx)
	historyService := workout_history.NewService(historyRepo)
	historyHandler := auth.NewWorkoutHistoryHandler(historyService)

	router := gin.New()
	me := router.Group("/api/v1/me")
	me.Use(middleware.Authenticate(tokenSvc, userRepo))
	me.GET("/statistics", statsHandler.GetStatistics)
	me.POST("/workouts/:workoutID/complete", historyHandler.CompleteWorkout)

	return router, userRepo, tx, tokenSvc
}

// newStatisticsHandlerRouter mounts only the handler with a pre-set
// authentication-context identity, so the handler's own error mapping and
// identity forwarding can be tested without the full middleware chain. nil
// identity simulates a missing context.
func newStatisticsHandlerRouter(svc statistics.Service, identity any) *gin.Engine {
	handler := auth.NewStatisticsHandler(svc)
	router := gin.New()
	me := router.Group("/api/v1/me")
	me.Use(func(c *gin.Context) {
		if identity != nil {
			c.Set(authcontext.UserIDContextKey, identity)
		}
		c.Next()
	})
	me.GET("/statistics", handler.GetStatistics)
	return router
}

// stubStatisticsService is a scripted fake used to exercise the handler's error
// mapping and identity forwarding without touching the database.
type stubStatisticsService struct {
	resp    *statistics.ClientStatisticsResponse
	err     error
	gotUser string
}

func (s *stubStatisticsService) GetClientStatistics(_ context.Context, userID string) (*statistics.ClientStatisticsResponse, error) {
	s.gotUser = userID
	return s.resp, s.err
}

func TestStatisticsSuccess(t *testing.T) {
	router, userRepo, tx, tokenSvc := newStatisticsTestRouter(t)
	trainer := seedTrainerForUser(t, repositories.NewTrainerRepository(tx), seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	_, workout := seedClientAssignedWorkout(t, tx, trainer.ID, clientUser.ID, "Strength Builder")

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	// Complete the workout first so there is history to report.
	if rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, sprintfPath(completeWorkoutRoute, workout.ID), ""); rec.Code != http.StatusCreated {
		t.Fatalf("complete workout: expected 201, got %d (body: %s)", rec.Code, raw)
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, statisticsRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if hasActive, _ := data["has_active_assignment"].(bool); !hasActive {
		t.Fatal("expected has_active_assignment=true")
	}
	if programName, _ := data["current_program_name"].(string); programName != "Strength Builder" {
		t.Fatalf("expected program name 'Strength Builder', got %q", programName)
	}
	if totalExecutions, _ := data["total_executions"].(float64); totalExecutions != 1 {
		t.Fatalf("expected total_executions=1, got %v", data["total_executions"])
	}
	if uniqueCompleted, _ := data["unique_workouts_completed"].(float64); uniqueCompleted != 1 {
		t.Fatalf("expected unique_workouts_completed=1, got %v", data["unique_workouts_completed"])
	}
	if totalInProgram, _ := data["total_workouts_in_program"].(float64); totalInProgram != 1 {
		t.Fatalf("expected total_workouts_in_program=1, got %v", data["total_workouts_in_program"])
	}
	if pct, _ := data["completion_percentage"].(float64); pct != 100.0 {
		t.Fatalf("expected completion_percentage=100.0, got %v", data["completion_percentage"])
	}
	if lastDate, _ := data["last_workout_date"].(string); lastDate == "" {
		t.Fatal("expected a non-empty last_workout_date")
	}
}

func TestStatisticsNoAssignment(t *testing.T) {
	router, userRepo, _, tokenSvc := newStatisticsTestRouter(t)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, statisticsRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if hasActive, _ := data["has_active_assignment"].(bool); hasActive {
		t.Fatal("expected has_active_assignment=false for user with no assignment")
	}
	if totalExecutions, _ := data["total_executions"].(float64); totalExecutions != 0 {
		t.Fatalf("expected total_executions=0, got %v", data["total_executions"])
	}
}

func TestStatisticsUnauthenticated(t *testing.T) {
	router, _, _, _ := newStatisticsTestRouter(t)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, statisticsRoute, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestStatisticsIsolation(t *testing.T) {
	router, userRepo, tx, tokenSvc := newStatisticsTestRouter(t)
	trainerRepo := repositories.NewTrainerRepository(tx)

	trainerA := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	trainerB := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientA := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	clientB := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	_, workoutA := seedClientAssignedWorkout(t, tx, trainerA.ID, clientA.ID, "Program A")
	_, workoutB := seedClientAssignedWorkout(t, tx, trainerB.ID, clientB.ID, "Program B")

	jwtA, err := tokenSvc.GenerateAccessToken(clientA.ID, clientA.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken A: %v", err)
	}
	jwtB, err := tokenSvc.GenerateAccessToken(clientB.ID, clientB.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken B: %v", err)
	}

	// Client A completes their workout.
	if rec, _, raw := trainerClientsRequest(router, jwtA, http.MethodPost, sprintfPath(completeWorkoutRoute, workoutA.ID), ""); rec.Code != http.StatusCreated {
		t.Fatalf("client A complete: expected 201, got %d (body: %s)", rec.Code, raw)
	}
	// Client B completes their workout.
	if rec, _, raw := trainerClientsRequest(router, jwtB, http.MethodPost, sprintfPath(completeWorkoutRoute, workoutB.ID), ""); rec.Code != http.StatusCreated {
		t.Fatalf("client B complete: expected 201, got %d (body: %s)", rec.Code, raw)
	}

	// Client A sees only their own statistics.
	rec, data, raw := trainerClientsRequest(router, jwtA, http.MethodGet, statisticsRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("client A stats: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if totalExecutions, _ := data["total_executions"].(float64); totalExecutions != 1 {
		t.Fatalf("client A: expected total_executions=1, got %v", data["total_executions"])
	}
	if programName, _ := data["current_program_name"].(string); programName != "Program A" {
		t.Fatalf("client A: expected program name 'Program A', got %q", programName)
	}

	// Client B sees only their own statistics.
	rec, data, raw = trainerClientsRequest(router, jwtB, http.MethodGet, statisticsRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("client B stats: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if totalExecutions, _ := data["total_executions"].(float64); totalExecutions != 1 {
		t.Fatalf("client B: expected total_executions=1, got %v", data["total_executions"])
	}
	if programName, _ := data["current_program_name"].(string); programName != "Program B" {
		t.Fatalf("client B: expected program name 'Program B', got %q", programName)
	}
}

func TestStatisticsHandlerForwardsContextIdentity(t *testing.T) {
	identity := uuid.NewString()
	svc := &stubStatisticsService{
		resp: &statistics.ClientStatisticsResponse{
			HasActiveAssignment: false,
		},
	}
	router := newStatisticsHandlerRouter(svc, identity)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, statisticsRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotUser != identity {
		t.Fatalf("expected context user %q, got %q", identity, svc.gotUser)
	}
}

func TestStatisticsHandlerMissingContext(t *testing.T) {
	router := newStatisticsHandlerRouter(&stubStatisticsService{}, nil)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, statisticsRoute, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestStatisticsHandlerErrorMapping(t *testing.T) {
	identity := uuid.NewString()

	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid input", err: statistics.ErrInvalidInput, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubStatisticsService{err: tc.err}
			router := newStatisticsHandlerRouter(svc, identity)

			rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, statisticsRoute, "")
			if rec.Code != tc.status {
				t.Fatalf("expected %d, got %d (body: %s)", tc.status, rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"`+tc.code+`"`) {
				t.Fatalf("expected code %s, got %s", tc.code, raw)
			}
		})
	}
}

func TestStatisticsHandlerRepositoryFailureNotExposed(t *testing.T) {
	svc := &stubStatisticsService{err: errLoginRepoFailure}
	router := newStatisticsHandlerRouter(svc, uuid.NewString())

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, statisticsRoute, "")
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

func TestStatisticsNeverExposesSecrets(t *testing.T) {
	router, userRepo, tx, tokenSvc := newStatisticsTestRouter(t)
	trainer := seedTrainerForUser(t, repositories.NewTrainerRepository(tx), seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	_, workout := seedClientAssignedWorkout(t, tx, trainer.ID, clientUser.ID, "Strength Builder")

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	// Complete a workout so there is history.
	if rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, sprintfPath(completeWorkoutRoute, workout.ID), ""); rec.Code != http.StatusCreated {
		t.Fatalf("complete workout: expected 201, got %d (body: %s)", rec.Code, raw)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, statisticsRoute, "")
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
		"user_id",
		clientUser.Email,
	} {
		if strings.Contains(raw, sensitive) {
			t.Fatalf("response must never contain %q", sensitive)
		}
	}
}

func TestStatisticsCompletionPercentageMultipleWorkouts(t *testing.T) {
	router, userRepo, tx, tokenSvc := newStatisticsTestRouter(t)
	trainer := seedTrainerForUser(t, repositories.NewTrainerRepository(tx), seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	// Create a program with 2 workouts (2 weeks, 1 workout each).
	programRepo := repositories.NewProgramRepository(tx)
	weekRepo := repositories.NewProgramWeekRepository(tx)
	workoutRepo := repositories.NewProgramWorkoutRepository(tx)
	clientRepo := repositories.NewTrainerClientRepository(tx)
	assignmentRepo := repositories.NewProgramAssignmentRepository(tx)
	ctx := context.Background()

	program := &models.Program{TrainerID: trainer.ID, Name: "Two-Week Program", Type: models.ProgramTypePremium, Status: models.ProgramStatusDraft}
	if err := programRepo.Create(ctx, program); err != nil {
		t.Fatalf("create program: %v", err)
	}
	if err := clientRepo.Create(ctx, &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("create relationship: %v", err)
	}
	week1 := &models.ProgramWeek{}
	if err := weekRepo.Create(ctx, trainer.ID, program.ID, week1); err != nil {
		t.Fatalf("create week 1: %v", err)
	}
	workout1 := &models.ProgramWorkout{}
	if err := workoutRepo.Create(ctx, trainer.ID, program.ID, week1.ID, workout1); err != nil {
		t.Fatalf("create workout 1: %v", err)
	}
	week2 := &models.ProgramWeek{}
	if err := weekRepo.Create(ctx, trainer.ID, program.ID, week2); err != nil {
		t.Fatalf("create week 2: %v", err)
	}
	workout2 := &models.ProgramWorkout{}
	if err := workoutRepo.Create(ctx, trainer.ID, program.ID, week2.ID, workout2); err != nil {
		t.Fatalf("create workout 2: %v", err)
	}
	if err := assignmentRepo.Create(ctx, trainer.ID, clientUser.ID, program.ID, &models.ProgramAssignment{}); err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	// Complete only workout1 — 50% completion.
	if rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, sprintfPath(completeWorkoutRoute, workout1.ID), ""); rec.Code != http.StatusCreated {
		t.Fatalf("complete workout 1: expected 201, got %d (body: %s)", rec.Code, raw)
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, statisticsRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if totalInProgram, _ := data["total_workouts_in_program"].(float64); totalInProgram != 2 {
		t.Fatalf("expected total_workouts_in_program=2, got %v", data["total_workouts_in_program"])
	}
	if uniqueCompleted, _ := data["unique_workouts_completed"].(float64); uniqueCompleted != 1 {
		t.Fatalf("expected unique_workouts_completed=1, got %v", data["unique_workouts_completed"])
	}
	if pct, _ := data["completion_percentage"].(float64); pct != 50.0 {
		t.Fatalf("expected completion_percentage=50.0, got %v", data["completion_percentage"])
	}
}
