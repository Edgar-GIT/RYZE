package auth_test

import (
	"context"
	"net/http"
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
	"ryze/backend/services/program_structure"
	"ryze/backend/services/token"
)

const (
	structureWeeksBase    = "/api/v1/trainer/programs/"
	structureWeeksSuffix  = "/weeks"
	structureWorkoutsBase = "/weeks/"
	structureOrderSuffix  = "/order"
	structureWorkoutsEnd  = "/workouts"
)

// newTrainerProgramStructureTestRouter wires the program structure endpoints
// behind the real Authenticate, TrainerAuthenticate and
// RequireTrainerPermission middleware, backed by a database transaction so
// created records are rolled back. The required permissions can be customized
// to exercise the 403 path.
func newTrainerProgramStructureTestRouter(t *testing.T, permissions ...trainerroles.Permission) (*gin.Engine, repositories.UserRepository, repositories.TrainerRepository, repositories.ProgramRepository, repositories.ProgramWeekRepository, repositories.ProgramWorkoutRepository, token.Service) {
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
	weekRepo := repositories.NewProgramWeekRepository(tx)
	workoutRepo := repositories.NewProgramWorkoutRepository(tx)
	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)

	service := program_structure.NewService(weekRepo, workoutRepo)
	handler := auth.NewTrainerProgramStructureHandler(service)

	router := gin.New()
	trainer := router.Group("/api/v1/trainer")
	trainer.Use(middleware.Authenticate(tokenSvc, userRepo))
	trainer.Use(middleware.TrainerAuthenticate(trainerRepo))
	trainer.POST("/programs/:programID/weeks", middleware.RequireTrainerPermission(permissions...), handler.CreateWeek)
	trainer.GET("/programs/:programID/weeks", middleware.RequireTrainerPermission(permissions...), handler.ListWeeks)
	trainer.PATCH("/programs/:programID/weeks/order", middleware.RequireTrainerPermission(permissions...), handler.ReorderWeeks)
	trainer.GET("/programs/:programID/weeks/:weekID", middleware.RequireTrainerPermission(permissions...), handler.GetWeek)
	trainer.DELETE("/programs/:programID/weeks/:weekID", middleware.RequireTrainerPermission(permissions...), handler.DeleteWeek)
	trainer.POST("/programs/:programID/weeks/:weekID/workouts", middleware.RequireTrainerPermission(permissions...), handler.CreateWorkout)
	trainer.GET("/programs/:programID/weeks/:weekID/workouts", middleware.RequireTrainerPermission(permissions...), handler.ListWorkouts)
	trainer.PATCH("/programs/:programID/weeks/:weekID/workouts/order", middleware.RequireTrainerPermission(permissions...), handler.ReorderWorkouts)
	trainer.GET("/programs/:programID/weeks/:weekID/workouts/:workoutID", middleware.RequireTrainerPermission(permissions...), handler.GetWorkout)
	trainer.DELETE("/programs/:programID/weeks/:weekID/workouts/:workoutID", middleware.RequireTrainerPermission(permissions...), handler.DeleteWorkout)

	return router, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, tokenSvc
}

// newTrainerProgramStructureHandlerRouter mounts only the handler with a pre-set
// trainer context identity, so the handler's own error mapping can be tested
// without the full middleware chain. nil identity simulates a missing context.
func newTrainerProgramStructureHandlerRouter(svc program_structure.Service, identity any) *gin.Engine {
	handler := auth.NewTrainerProgramStructureHandler(svc)
	router := gin.New()
	trainer := router.Group("/api/v1/trainer")
	trainer.Use(func(c *gin.Context) {
		if identity != nil {
			c.Set(trainercontext.TrainerContextKey, identity)
		}
		c.Next()
	})
	trainer.POST("/programs/:programID/weeks", handler.CreateWeek)
	trainer.GET("/programs/:programID/weeks", handler.ListWeeks)
	trainer.PATCH("/programs/:programID/weeks/order", handler.ReorderWeeks)
	trainer.GET("/programs/:programID/weeks/:weekID", handler.GetWeek)
	trainer.DELETE("/programs/:programID/weeks/:weekID", handler.DeleteWeek)
	trainer.POST("/programs/:programID/weeks/:weekID/workouts", handler.CreateWorkout)
	trainer.GET("/programs/:programID/weeks/:weekID/workouts", handler.ListWorkouts)
	trainer.PATCH("/programs/:programID/weeks/:weekID/workouts/order", handler.ReorderWorkouts)
	trainer.GET("/programs/:programID/weeks/:weekID/workouts/:workoutID", handler.GetWorkout)
	trainer.DELETE("/programs/:programID/weeks/:weekID/workouts/:workoutID", handler.DeleteWorkout)
	return router
}

// stubProgramStructureService is a scripted fake used to exercise the handler's
// error mapping and identity forwarding without touching the database.
type stubProgramStructureService struct {
	week         *program_structure.Week
	weeks        []program_structure.Week
	workout      *program_structure.Workout
	workouts     []program_structure.Workout
	err          error
	gotTrainerID string
	gotProgramID string
	gotWeekID    string
	gotWorkoutID string
	gotOrder     []string
}

func (s *stubProgramStructureService) CreateWeek(_ context.Context, trainerID, programID string) (*program_structure.Week, error) {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	return s.week, s.err
}

func (s *stubProgramStructureService) ListWeeks(_ context.Context, trainerID, programID string) ([]program_structure.Week, error) {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	return s.weeks, s.err
}

func (s *stubProgramStructureService) GetWeek(_ context.Context, trainerID, programID, weekID string) (*program_structure.Week, error) {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	return s.week, s.err
}

func (s *stubProgramStructureService) ReorderWeeks(_ context.Context, trainerID, programID string, orderedIDs []string) error {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotOrder = orderedIDs
	return s.err
}

func (s *stubProgramStructureService) DeleteWeek(_ context.Context, trainerID, programID, weekID string) error {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	return s.err
}

func (s *stubProgramStructureService) CreateWorkout(_ context.Context, trainerID, programID, weekID string) (*program_structure.Workout, error) {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	return s.workout, s.err
}

func (s *stubProgramStructureService) ListWorkouts(_ context.Context, trainerID, programID, weekID string) ([]program_structure.Workout, error) {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	return s.workouts, s.err
}

func (s *stubProgramStructureService) GetWorkout(_ context.Context, trainerID, programID, weekID, workoutID string) (*program_structure.Workout, error) {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	s.gotWorkoutID = workoutID
	return s.workout, s.err
}

func (s *stubProgramStructureService) ReorderWorkouts(_ context.Context, trainerID, programID, weekID string, orderedIDs []string) error {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	s.gotOrder = orderedIDs
	return s.err
}

func (s *stubProgramStructureService) DeleteWorkout(_ context.Context, trainerID, programID, weekID, workoutID string) error {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	s.gotWorkoutID = workoutID
	return s.err
}

func structureWeeksPath(programID string) string {
	return structureWeeksBase + programID + structureWeeksSuffix
}

func structureWeeksOrderPath(programID string) string {
	return structureWeeksBase + programID + structureWeeksSuffix + structureOrderSuffix
}

func structureWeekPath(programID, weekID string) string {
	return structureWeeksBase + programID + structureWeeksSuffix + "/" + weekID
}

func structureWorkoutsPath(programID, weekID string) string {
	return structureWeekPath(programID, weekID) + structureWorkoutsEnd
}

func structureWorkoutsOrderPath(programID, weekID string) string {
	return structureWorkoutsPath(programID, weekID) + structureOrderSuffix
}

func structureWorkoutPath(programID, weekID, workoutID string) string {
	return structureWorkoutsPath(programID, weekID) + "/" + workoutID
}

func seedStructureProgram(t *testing.T, programRepo repositories.ProgramRepository, trainerID string) *models.Program {
	t.Helper()
	program := &models.Program{
		TrainerID: trainerID,
		Name:      "Strength Builder",
		Type:      models.ProgramTypePremium,
		Status:    models.ProgramStatusDraft,
	}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed program: %v", err)
	}
	return program
}

func TestTrainerProgramStructureCreateWeekSuccess(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, weekRepo, _, tokenSvc := newTrainerProgramStructureTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedStructureProgram(t, programRepo, trainer.ID)

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, structureWeeksPath(program.ID), "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if id, _ := data["id"].(string); id == "" {
		t.Fatal("expected week id")
	}
	if number, _ := data["week_number"].(float64); number != 1 {
		t.Fatalf("expected week_number 1, got %v", number)
	}
	if workouts, _ := data["workouts"].([]any); len(workouts) != 0 {
		t.Fatalf("expected empty workouts, got %v", workouts)
	}

	weeks, err := weekRepo.ListByProgram(context.Background(), trainer.ID, program.ID)
	if err != nil {
		t.Fatalf("list weeks: %v", err)
	}
	if len(weeks) != 1 {
		t.Fatalf("expected one persisted week, got %d", len(weeks))
	}
}

func TestTrainerProgramStructureCreateWeekCrossTrainer(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, _, _, tokenSvc := newTrainerProgramStructureTestRouter(t, trainerroles.PermissionPrograms)
	_, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	foreignProgram := seedStructureProgram(t, programRepo, otherTrainer.ID)

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, structureWeeksPath(foreignProgram.ID), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for foreign program, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"PROGRAM_NOT_FOUND"`) {
		t.Fatalf("expected PROGRAM_NOT_FOUND, got %s", raw)
	}
}

func TestTrainerProgramStructureListWeeksSuccess(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, tokenSvc := newTrainerProgramStructureTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedStructureProgram(t, programRepo, trainer.ID)

	week := &models.ProgramWeek{}
	if err := weekRepo.Create(context.Background(), trainer.ID, program.ID, week); err != nil {
		t.Fatalf("seed week: %v", err)
	}
	workout := &models.ProgramWorkout{}
	if err := workoutRepo.Create(context.Background(), trainer.ID, program.ID, week.ID, workout); err != nil {
		t.Fatalf("seed workout: %v", err)
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, structureWeeksPath(program.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	weeks, _ := data["weeks"].([]any)
	if len(weeks) != 1 {
		t.Fatalf("expected one week, got %d", len(weeks))
	}
	first, _ := weeks[0].(map[string]any)
	if number, _ := first["week_number"].(float64); number != 1 {
		t.Fatalf("expected week_number 1, got %v", number)
	}
	nested, _ := first["workouts"].([]any)
	if len(nested) != 1 {
		t.Fatalf("expected one nested workout, got %d", len(nested))
	}
	nestedWorkout, _ := nested[0].(map[string]any)
	if position, _ := nestedWorkout["position"].(float64); position != 1 {
		t.Fatalf("expected nested position 1, got %v", position)
	}
	if id, _ := nestedWorkout["id"].(string); id != workout.ID {
		t.Fatalf("expected nested workout id %q, got %q", workout.ID, id)
	}
}

func TestTrainerProgramStructureListWeeksIDOR(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, weekRepo, _, tokenSvc := newTrainerProgramStructureTestRouter(t, trainerroles.PermissionPrograms)
	_, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	foreignProgram := seedStructureProgram(t, programRepo, otherTrainer.ID)
	if err := weekRepo.Create(context.Background(), otherTrainer.ID, foreignProgram.ID, &models.ProgramWeek{}); err != nil {
		t.Fatalf("seed foreign week: %v", err)
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, structureWeeksPath(foreignProgram.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with empty list, got %d (body: %s)", rec.Code, raw)
	}
	weeks, _ := data["weeks"].([]any)
	if len(weeks) != 0 {
		t.Fatalf("foreign program weeks must never be listed, got %d", len(weeks))
	}
}

func TestTrainerProgramStructureGetWeekSuccess(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, weekRepo, _, tokenSvc := newTrainerProgramStructureTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedStructureProgram(t, programRepo, trainer.ID)
	week := &models.ProgramWeek{}
	if err := weekRepo.Create(context.Background(), trainer.ID, program.ID, week); err != nil {
		t.Fatalf("seed week: %v", err)
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, structureWeekPath(program.ID, week.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if id, _ := data["id"].(string); id != week.ID {
		t.Fatalf("expected week id %q, got %q", week.ID, id)
	}
}

func TestTrainerProgramStructureGetWeekIDOR(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, weekRepo, _, tokenSvc := newTrainerProgramStructureTestRouter(t, trainerroles.PermissionPrograms)
	_, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	foreignProgram := seedStructureProgram(t, programRepo, otherTrainer.ID)
	foreignWeek := &models.ProgramWeek{}
	if err := weekRepo.Create(context.Background(), otherTrainer.ID, foreignProgram.ID, foreignWeek); err != nil {
		t.Fatalf("seed foreign week: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, structureWeekPath(foreignProgram.ID, foreignWeek.ID), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for foreign week, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"WEEK_NOT_FOUND"`) {
		t.Fatalf("expected WEEK_NOT_FOUND, got %s", raw)
	}
}

func TestTrainerProgramStructureReorderWeeksSuccess(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, weekRepo, _, tokenSvc := newTrainerProgramStructureTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedStructureProgram(t, programRepo, trainer.ID)

	var weekIDs []string
	for i := 0; i < 3; i++ {
		week := &models.ProgramWeek{}
		if err := weekRepo.Create(context.Background(), trainer.ID, program.ID, week); err != nil {
			t.Fatalf("seed week: %v", err)
		}
		weekIDs = append(weekIDs, week.ID)
	}

	order := []string{weekIDs[2], weekIDs[0], weekIDs[1]}
	body := `{"ids":["` + strings.Join(order, `","`) + `"]}`
	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPatch, structureWeeksOrderPath(program.ID), body)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	weeks, err := weekRepo.ListByProgram(context.Background(), trainer.ID, program.ID)
	if err != nil {
		t.Fatalf("list weeks: %v", err)
	}
	if len(weeks) != 3 || weeks[0].ID != weekIDs[2] || weeks[1].ID != weekIDs[0] || weeks[2].ID != weekIDs[1] {
		t.Fatalf("expected reordered weeks, got %+v", weeks)
	}
}

func TestTrainerProgramStructureReorderWeeksConflict(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, weekRepo, _, tokenSvc := newTrainerProgramStructureTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedStructureProgram(t, programRepo, trainer.ID)
	week := &models.ProgramWeek{}
	if err := weekRepo.Create(context.Background(), trainer.ID, program.ID, week); err != nil {
		t.Fatalf("seed week: %v", err)
	}

	body := `{"ids":["` + uuid.NewString() + `"]}`
	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPatch, structureWeeksOrderPath(program.ID), body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"REORDER_CONFLICT"`) {
		t.Fatalf("expected REORDER_CONFLICT, got %s", raw)
	}
}

func TestTrainerProgramStructureDeleteWeekSuccess(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, tokenSvc := newTrainerProgramStructureTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedStructureProgram(t, programRepo, trainer.ID)
	week := &models.ProgramWeek{}
	if err := weekRepo.Create(context.Background(), trainer.ID, program.ID, week); err != nil {
		t.Fatalf("seed week: %v", err)
	}
	workout := &models.ProgramWorkout{}
	if err := workoutRepo.Create(context.Background(), trainer.ID, program.ID, week.ID, workout); err != nil {
		t.Fatalf("seed workout: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodDelete, structureWeekPath(program.ID, week.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	if _, err := weekRepo.FindByIDAndProgram(context.Background(), trainer.ID, program.ID, week.ID); err == nil {
		t.Fatal("expected the week to be soft-deleted")
	}
	workouts, err := workoutRepo.ListByWeek(context.Background(), trainer.ID, program.ID, week.ID)
	if err != nil {
		t.Fatalf("list workouts: %v", err)
	}
	if len(workouts) != 0 {
		t.Fatalf("workouts of a deleted week must be unreachable, got %d", len(workouts))
	}
}

func TestTrainerProgramStructureCreateWorkoutSuccess(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, tokenSvc := newTrainerProgramStructureTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedStructureProgram(t, programRepo, trainer.ID)
	week := &models.ProgramWeek{}
	if err := weekRepo.Create(context.Background(), trainer.ID, program.ID, week); err != nil {
		t.Fatalf("seed week: %v", err)
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, structureWorkoutsPath(program.ID, week.ID), "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if position, _ := data["position"].(float64); position != 1 {
		t.Fatalf("expected position 1, got %v", position)
	}
	if id, _ := data["program_week_id"].(string); id != week.ID {
		t.Fatalf("expected program_week_id %q, got %q", week.ID, id)
	}

	workouts, err := workoutRepo.ListByWeek(context.Background(), trainer.ID, program.ID, week.ID)
	if err != nil {
		t.Fatalf("list workouts: %v", err)
	}
	if len(workouts) != 1 {
		t.Fatalf("expected one persisted workout, got %d", len(workouts))
	}
}

func TestTrainerProgramStructureCreateWorkoutForeignWeek(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, weekRepo, _, tokenSvc := newTrainerProgramStructureTestRouter(t, trainerroles.PermissionPrograms)
	_, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	foreignProgram := seedStructureProgram(t, programRepo, otherTrainer.ID)
	foreignWeek := &models.ProgramWeek{}
	if err := weekRepo.Create(context.Background(), otherTrainer.ID, foreignProgram.ID, foreignWeek); err != nil {
		t.Fatalf("seed foreign week: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, structureWorkoutsPath(foreignProgram.ID, foreignWeek.ID), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for foreign week, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"WEEK_NOT_FOUND"`) {
		t.Fatalf("expected WEEK_NOT_FOUND, got %s", raw)
	}
}

func TestTrainerProgramStructureListWorkoutsSuccess(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, tokenSvc := newTrainerProgramStructureTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedStructureProgram(t, programRepo, trainer.ID)
	week := &models.ProgramWeek{}
	if err := weekRepo.Create(context.Background(), trainer.ID, program.ID, week); err != nil {
		t.Fatalf("seed week: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := workoutRepo.Create(context.Background(), trainer.ID, program.ID, week.ID, &models.ProgramWorkout{}); err != nil {
			t.Fatalf("seed workout: %v", err)
		}
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, structureWorkoutsPath(program.ID, week.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	workouts, _ := data["workouts"].([]any)
	if len(workouts) != 2 {
		t.Fatalf("expected two workouts, got %d", len(workouts))
	}
	first, _ := workouts[0].(map[string]any)
	if position, _ := first["position"].(float64); position != 1 {
		t.Fatalf("expected position 1, got %v", position)
	}
}

func TestTrainerProgramStructureGetWorkoutSuccess(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, tokenSvc := newTrainerProgramStructureTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedStructureProgram(t, programRepo, trainer.ID)
	week := &models.ProgramWeek{}
	if err := weekRepo.Create(context.Background(), trainer.ID, program.ID, week); err != nil {
		t.Fatalf("seed week: %v", err)
	}
	workout := &models.ProgramWorkout{}
	if err := workoutRepo.Create(context.Background(), trainer.ID, program.ID, week.ID, workout); err != nil {
		t.Fatalf("seed workout: %v", err)
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, structureWorkoutPath(program.ID, week.ID, workout.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if id, _ := data["id"].(string); id != workout.ID {
		t.Fatalf("expected workout id %q, got %q", workout.ID, id)
	}
}

func TestTrainerProgramStructureGetWorkoutIDOR(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, tokenSvc := newTrainerProgramStructureTestRouter(t, trainerroles.PermissionPrograms)
	_, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	foreignProgram := seedStructureProgram(t, programRepo, otherTrainer.ID)
	foreignWeek := &models.ProgramWeek{}
	if err := weekRepo.Create(context.Background(), otherTrainer.ID, foreignProgram.ID, foreignWeek); err != nil {
		t.Fatalf("seed foreign week: %v", err)
	}
	foreignWorkout := &models.ProgramWorkout{}
	if err := workoutRepo.Create(context.Background(), otherTrainer.ID, foreignProgram.ID, foreignWeek.ID, foreignWorkout); err != nil {
		t.Fatalf("seed foreign workout: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, structureWorkoutPath(foreignProgram.ID, foreignWeek.ID, foreignWorkout.ID), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for foreign workout, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"WORKOUT_NOT_FOUND"`) {
		t.Fatalf("expected WORKOUT_NOT_FOUND, got %s", raw)
	}
}

func TestTrainerProgramStructureReorderWorkoutsSuccess(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, tokenSvc := newTrainerProgramStructureTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedStructureProgram(t, programRepo, trainer.ID)
	week := &models.ProgramWeek{}
	if err := weekRepo.Create(context.Background(), trainer.ID, program.ID, week); err != nil {
		t.Fatalf("seed week: %v", err)
	}
	var workoutIDs []string
	for i := 0; i < 3; i++ {
		workout := &models.ProgramWorkout{}
		if err := workoutRepo.Create(context.Background(), trainer.ID, program.ID, week.ID, workout); err != nil {
			t.Fatalf("seed workout: %v", err)
		}
		workoutIDs = append(workoutIDs, workout.ID)
	}

	order := []string{workoutIDs[2], workoutIDs[0], workoutIDs[1]}
	body := `{"ids":["` + strings.Join(order, `","`) + `"]}`
	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPatch, structureWorkoutsOrderPath(program.ID, week.ID), body)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	workouts, err := workoutRepo.ListByWeek(context.Background(), trainer.ID, program.ID, week.ID)
	if err != nil {
		t.Fatalf("list workouts: %v", err)
	}
	if len(workouts) != 3 || workouts[0].ID != workoutIDs[2] || workouts[1].ID != workoutIDs[0] || workouts[2].ID != workoutIDs[1] {
		t.Fatalf("expected reordered workouts, got %+v", workouts)
	}
}

func TestTrainerProgramStructureDeleteWorkoutSuccess(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, tokenSvc := newTrainerProgramStructureTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedStructureProgram(t, programRepo, trainer.ID)
	week := &models.ProgramWeek{}
	if err := weekRepo.Create(context.Background(), trainer.ID, program.ID, week); err != nil {
		t.Fatalf("seed week: %v", err)
	}
	workout := &models.ProgramWorkout{}
	if err := workoutRepo.Create(context.Background(), trainer.ID, program.ID, week.ID, workout); err != nil {
		t.Fatalf("seed workout: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodDelete, structureWorkoutPath(program.ID, week.ID, workout.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	if _, err := workoutRepo.FindByIDAndWeek(context.Background(), trainer.ID, program.ID, week.ID, workout.ID); err == nil {
		t.Fatal("expected the workout to be soft-deleted")
	}
}

func TestTrainerProgramStructureNotAuthenticated(t *testing.T) {
	router, _, _, _, _, _, _ := newTrainerProgramStructureTestRouter(t, trainerroles.PermissionPrograms)

	programID := uuid.NewString()
	weekID := uuid.NewString()
	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "create week", method: http.MethodPost, path: structureWeeksPath(programID)},
		{name: "list weeks", method: http.MethodGet, path: structureWeeksPath(programID)},
		{name: "reorder weeks", method: http.MethodPatch, path: structureWeeksOrderPath(programID), body: `{"ids":["` + uuid.NewString() + `"]}`},
		{name: "get week", method: http.MethodGet, path: structureWeekPath(programID, weekID)},
		{name: "delete week", method: http.MethodDelete, path: structureWeekPath(programID, weekID)},
		{name: "create workout", method: http.MethodPost, path: structureWorkoutsPath(programID, weekID)},
		{name: "list workouts", method: http.MethodGet, path: structureWorkoutsPath(programID, weekID)},
		{name: "reorder workouts", method: http.MethodPatch, path: structureWorkoutsOrderPath(programID, weekID), body: `{"ids":["` + uuid.NewString() + `"]}`},
		{name: "get workout", method: http.MethodGet, path: structureWorkoutPath(programID, weekID, uuid.NewString())},
		{name: "delete workout", method: http.MethodDelete, path: structureWorkoutPath(programID, weekID, uuid.NewString())},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, _, raw := trainerClientsRequest(router, "", tc.method, tc.path, tc.body)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
				t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
			}
		})
	}
}

func TestTrainerProgramStructureAuthenticatedNonTrainer(t *testing.T) {
	router, userRepo, _, _, _, _, tokenSvc := newTrainerProgramStructureTestRouter(t, trainerroles.PermissionPrograms)
	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	jwtValue, err := tokenSvc.GenerateAccessToken(user.ID, user.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	programID := uuid.NewString()
	cases := []struct {
		name   string
		method string
		path   string
	}{
		{name: "list weeks", method: http.MethodGet, path: structureWeeksPath(programID)},
		{name: "create week", method: http.MethodPost, path: structureWeeksPath(programID)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, _, raw := trainerClientsRequest(router, jwtValue, tc.method, tc.path, "")
			if rec.Code != http.StatusForbidden {
				t.Fatalf("expected 403, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"FORBIDDEN"`) {
				t.Fatalf("expected FORBIDDEN, got %s", raw)
			}
		})
	}
}

func TestTrainerProgramStructurePermissionNotGranted(t *testing.T) {
	router, userRepo, trainerRepo, _, _, _, tokenSvc := newTrainerProgramStructureTestRouter(t, trainerroles.Permission("trainer.schedule"))
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	_ = trainer

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, structureWeeksPath(uuid.NewString()), "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "trainer.schedule") {
		t.Fatalf("forbidden error must not reveal the permission, got %s", raw)
	}
}

func TestTrainerProgramStructureHandlerForwardsContextIdentity(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	programID := uuid.NewString()
	svc := &stubProgramStructureService{
		week: &program_structure.Week{
			ID:         uuid.NewString(),
			ProgramID:  programID,
			WeekNumber: 1,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}
	router := newTrainerProgramStructureHandlerRouter(svc, identity)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, structureWeeksPath(programID), "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotTrainerID != identity.TrainerID {
		t.Fatalf("expected context trainer %q, got %q", identity.TrainerID, svc.gotTrainerID)
	}
	if svc.gotProgramID != programID {
		t.Fatalf("expected path program %q, got %q", programID, svc.gotProgramID)
	}
}

func TestTrainerProgramStructureHandlerForwardsPathIdentities(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	programID := uuid.NewString()
	weekID := uuid.NewString()
	workoutID := uuid.NewString()
	svc := &stubProgramStructureService{
		workout: &program_structure.Workout{
			ID:            workoutID,
			ProgramWeekID: weekID,
			Position:      1,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
	}
	router := newTrainerProgramStructureHandlerRouter(svc, identity)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, structureWorkoutPath(programID, weekID, workoutID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotTrainerID != identity.TrainerID {
		t.Fatalf("expected context trainer %q, got %q", identity.TrainerID, svc.gotTrainerID)
	}
	if svc.gotProgramID != programID || svc.gotWeekID != weekID || svc.gotWorkoutID != workoutID {
		t.Fatalf("expected path identities %q/%q/%q, got %q/%q/%q", programID, weekID, workoutID, svc.gotProgramID, svc.gotWeekID, svc.gotWorkoutID)
	}
}

func TestTrainerProgramStructureHandlerForwardsOrder(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	programID := uuid.NewString()
	order := []string{uuid.NewString(), uuid.NewString()}
	svc := &stubProgramStructureService{}
	router := newTrainerProgramStructureHandlerRouter(svc, identity)

	body := `{"ids":["` + strings.Join(order, `","`) + `"]}`
	rec, _, raw := trainerClientsRequest(router, "", http.MethodPatch, structureWeeksOrderPath(programID), body)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if len(svc.gotOrder) != 2 || svc.gotOrder[0] != order[0] || svc.gotOrder[1] != order[1] {
		t.Fatalf("expected order %v, got %v", order, svc.gotOrder)
	}
}

func TestTrainerProgramStructureHandlerMissingContext(t *testing.T) {
	router := newTrainerProgramStructureHandlerRouter(&stubProgramStructureService{}, nil)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, structureWeeksPath(uuid.NewString()), "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestTrainerProgramStructureHandlerErrorMapping(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}

	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid input", err: program_structure.ErrInvalidInput, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
		{name: "program not found", err: program_structure.ErrProgramNotFound, status: http.StatusNotFound, code: "PROGRAM_NOT_FOUND"},
		{name: "week not found", err: program_structure.ErrWeekNotFound, status: http.StatusNotFound, code: "WEEK_NOT_FOUND"},
		{name: "workout not found", err: program_structure.ErrWorkoutNotFound, status: http.StatusNotFound, code: "WORKOUT_NOT_FOUND"},
		{name: "reorder conflict", err: program_structure.ErrReorderConflict, status: http.StatusConflict, code: "REORDER_CONFLICT"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubProgramStructureService{err: tc.err}
			router := newTrainerProgramStructureHandlerRouter(svc, identity)

			rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, structureWeeksPath(uuid.NewString()), "")
			if rec.Code != tc.status {
				t.Fatalf("expected %d, got %d (body: %s)", tc.status, rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"`+tc.code+`"`) {
				t.Fatalf("expected code %s, got %s", tc.code, raw)
			}
		})
	}
}

func TestTrainerProgramStructureHandlerRepositoryFailureNotExposed(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	svc := &stubProgramStructureService{err: errLoginRepoFailure}
	router := newTrainerProgramStructureHandlerRouter(svc, identity)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, structureWeeksPath(uuid.NewString()), "")
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

func TestTrainerProgramStructureInvalidJSONBody(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	router := newTrainerProgramStructureHandlerRouter(&stubProgramStructureService{}, identity)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodPatch, structureWeeksOrderPath(uuid.NewString()), `not json`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestTrainerProgramStructureNeverExposesSecrets(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, tokenSvc := newTrainerProgramStructureTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program := seedStructureProgram(t, programRepo, trainer.ID)
	week := &models.ProgramWeek{}
	if err := weekRepo.Create(context.Background(), trainer.ID, program.ID, week); err != nil {
		t.Fatalf("seed week: %v", err)
	}
	workout := &models.ProgramWorkout{}
	if err := workoutRepo.Create(context.Background(), trainer.ID, program.ID, week.ID, workout); err != nil {
		t.Fatalf("seed workout: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, structureWeeksPath(program.ID), "")
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
