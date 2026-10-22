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
	"ryze/backend/middleware/trainercontext"
	"ryze/backend/middleware/trainerroles"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/token"
	"ryze/backend/services/workout_exercises"
)

const (
	workoutExercisesBase   = "/api/v1/trainer/programs/"
	workoutExercisesMiddle = "/weeks/"
	workoutExercisesEnd    = "/workouts/"
	workoutExercisesName   = "/exercises"
	workoutExercisesOrder  = "/order"
)

// newTrainerWorkoutExerciseTestRouter wires the workout exercise endpoints
// behind the real Authenticate, TrainerAuthenticate and
// RequireTrainerPermission middleware, backed by a database transaction so
// created records are rolled back. The required permissions can be customized
// to exercise the 403 path. The database handle is returned so tests can seed
// global catalog exercises.
func newTrainerWorkoutExerciseTestRouter(t *testing.T, permissions ...trainerroles.Permission) (*gin.Engine, *gorm.DB, repositories.UserRepository, repositories.TrainerRepository, repositories.ProgramRepository, repositories.ProgramWeekRepository, repositories.ProgramWorkoutRepository, repositories.WorkoutExerciseRepository, token.Service) {
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
	workoutExerciseRepo := repositories.NewWorkoutExerciseRepository(tx)
	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)

	service := workout_exercises.NewService(workoutExerciseRepo)
	handler := auth.NewTrainerWorkoutExerciseHandler(service)

	router := gin.New()
	trainer := router.Group("/api/v1/trainer")
	trainer.Use(middleware.Authenticate(tokenSvc, userRepo))
	trainer.Use(middleware.TrainerAuthenticate(trainerRepo))
	trainer.POST("/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises", middleware.RequireTrainerPermission(permissions...), handler.AddExercise)
	trainer.GET("/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises", middleware.RequireTrainerPermission(permissions...), handler.ListExercises)
	trainer.PATCH("/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises/order", middleware.RequireTrainerPermission(permissions...), handler.ReorderExercises)
	trainer.PATCH("/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises/:workoutExerciseID", middleware.RequireTrainerPermission(permissions...), handler.UpdatePrescription)
	trainer.GET("/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises/:workoutExerciseID", middleware.RequireTrainerPermission(permissions...), handler.GetExercise)
	trainer.DELETE("/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises/:workoutExerciseID", middleware.RequireTrainerPermission(permissions...), handler.DeleteExercise)

	return router, tx, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, workoutExerciseRepo, tokenSvc
}

// newTrainerWorkoutExerciseHandlerRouter mounts only the handler with a pre-set
// trainer context identity, so the handler's own error mapping and identity
// forwarding can be tested without the full middleware chain. nil identity
// simulates a missing context.
func newTrainerWorkoutExerciseHandlerRouter(svc workout_exercises.Service, identity any) *gin.Engine {
	handler := auth.NewTrainerWorkoutExerciseHandler(svc)
	router := gin.New()
	trainer := router.Group("/api/v1/trainer")
	trainer.Use(func(c *gin.Context) {
		if identity != nil {
			c.Set(trainercontext.TrainerContextKey, identity)
		}
		c.Next()
	})
	trainer.POST("/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises", handler.AddExercise)
	trainer.GET("/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises", handler.ListExercises)
	trainer.PATCH("/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises/order", handler.ReorderExercises)
	trainer.PATCH("/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises/:workoutExerciseID", handler.UpdatePrescription)
	trainer.GET("/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises/:workoutExerciseID", handler.GetExercise)
	trainer.DELETE("/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises/:workoutExerciseID", handler.DeleteExercise)
	return router
}

// stubWorkoutExerciseService is a scripted fake used to exercise the handler's
// error mapping and identity forwarding without touching the database.
type stubWorkoutExerciseService struct {
	entry         *workout_exercises.WorkoutExercise
	entries       []workout_exercises.WorkoutExercise
	err           error
	gotTrainerID  string
	gotProgramID  string
	gotWeekID     string
	gotWorkoutID  string
	gotEntryID    string
	gotExerciseID string
	gotOrder      []string
	gotInstr      string
	gotNotes      string
	gotSets       []workout_exercises.SetInput
}

func (s *stubWorkoutExerciseService) AddExercise(_ context.Context, trainerID, programID, weekID, workoutID, exerciseID string) (*workout_exercises.WorkoutExercise, error) {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	s.gotWorkoutID = workoutID
	s.gotExerciseID = exerciseID
	return s.entry, s.err
}

func (s *stubWorkoutExerciseService) ListExercises(_ context.Context, trainerID, programID, weekID, workoutID string) ([]workout_exercises.WorkoutExercise, error) {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	s.gotWorkoutID = workoutID
	return s.entries, s.err
}

func (s *stubWorkoutExerciseService) GetExercise(_ context.Context, trainerID, programID, weekID, workoutID, workoutExerciseID string) (*workout_exercises.WorkoutExercise, error) {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	s.gotWorkoutID = workoutID
	s.gotEntryID = workoutExerciseID
	return s.entry, s.err
}

func (s *stubWorkoutExerciseService) UpdatePrescription(_ context.Context, trainerID, programID, weekID, workoutID, workoutExerciseID, instructions, notes string, sets []workout_exercises.SetInput) (*workout_exercises.WorkoutExercise, error) {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	s.gotWorkoutID = workoutID
	s.gotEntryID = workoutExerciseID
	s.gotInstr = instructions
	s.gotNotes = notes
	s.gotSets = sets
	return s.entry, s.err
}

func (s *stubWorkoutExerciseService) ReorderExercises(_ context.Context, trainerID, programID, weekID, workoutID string, orderedIDs []string) error {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	s.gotWorkoutID = workoutID
	s.gotOrder = orderedIDs
	return s.err
}

func (s *stubWorkoutExerciseService) RemoveExercise(_ context.Context, trainerID, programID, weekID, workoutID, workoutExerciseID string) error {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	s.gotWorkoutID = workoutID
	s.gotEntryID = workoutExerciseID
	return s.err
}

func workoutExercisesPath(programID, weekID, workoutID string) string {
	return workoutExercisesBase + programID + workoutExercisesMiddle + weekID + workoutExercisesEnd + workoutID + workoutExercisesName
}

func workoutExercisesOrderPath(programID, weekID, workoutID string) string {
	return workoutExercisesPath(programID, weekID, workoutID) + workoutExercisesOrder
}

func workoutExercisePath(programID, weekID, workoutID, entryID string) string {
	return workoutExercisesPath(programID, weekID, workoutID) + "/" + entryID
}

func seedWorkoutExerciseProgram(t *testing.T, programRepo repositories.ProgramRepository, weekRepo repositories.ProgramWeekRepository, workoutRepo repositories.ProgramWorkoutRepository, trainerID string) (*models.Program, *models.ProgramWeek, *models.ProgramWorkout) {
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
	week := &models.ProgramWeek{}
	if err := weekRepo.Create(context.Background(), trainerID, program.ID, week); err != nil {
		t.Fatalf("seed week: %v", err)
	}
	workout := &models.ProgramWorkout{}
	if err := workoutRepo.Create(context.Background(), trainerID, program.ID, week.ID, workout); err != nil {
		t.Fatalf("seed workout: %v", err)
	}
	return program, week, workout
}

func seedWorkoutExerciseCatalog(t *testing.T, db *gorm.DB, name string) *models.Exercise {
	t.Helper()
	exercise := &models.Exercise{Name: name}
	if err := db.Create(exercise).Error; err != nil {
		t.Fatalf("seed exercise: %v", err)
	}
	return exercise
}

func TestTrainerWorkoutExerciseAddSuccess(t *testing.T) {
	router, db, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, _, tokenSvc := newTrainerWorkoutExerciseTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program, week, workout := seedWorkoutExerciseProgram(t, programRepo, weekRepo, workoutRepo, trainer.ID)
	squat := seedWorkoutExerciseCatalog(t, db, "Barbell Squat")

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, workoutExercisesPath(program.ID, week.ID, workout.ID), `{"exercise_id":"`+squat.ID+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if id, _ := data["id"].(string); id == "" {
		t.Fatal("expected workout exercise id")
	}
	if position, _ := data["position"].(float64); position != 1 {
		t.Fatalf("expected position 1, got %v", position)
	}
	if workoutID, _ := data["program_workout_id"].(string); workoutID != workout.ID {
		t.Fatalf("expected program_workout_id %q, got %q", workout.ID, workoutID)
	}
	embedded, _ := data["exercise"].(map[string]any)
	if embeddedID, _ := embedded["id"].(string); embeddedID != squat.ID {
		t.Fatalf("expected embedded exercise %q, got %q", squat.ID, embeddedID)
	}
	if name, _ := embedded["name"].(string); name != "Barbell Squat" {
		t.Fatalf("expected embedded name Barbell Squat, got %q", name)
	}
}

func TestTrainerWorkoutExerciseAddForeignWorkout(t *testing.T) {
	router, db, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, _, tokenSvc := newTrainerWorkoutExerciseTestRouter(t, trainerroles.PermissionPrograms)
	_, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	foreignProgram, foreignWeek, foreignWorkout := seedWorkoutExerciseProgram(t, programRepo, weekRepo, workoutRepo, otherTrainer.ID)
	squat := seedWorkoutExerciseCatalog(t, db, "Barbell Squat")

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, workoutExercisesPath(foreignProgram.ID, foreignWeek.ID, foreignWorkout.ID), `{"exercise_id":"`+squat.ID+`"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for foreign workout, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"WORKOUT_NOT_FOUND"`) {
		t.Fatalf("expected WORKOUT_NOT_FOUND, got %s", raw)
	}
}

func TestTrainerWorkoutExerciseAddSoftDeletedExercise(t *testing.T) {
	router, db, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, _, tokenSvc := newTrainerWorkoutExerciseTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program, week, workout := seedWorkoutExerciseProgram(t, programRepo, weekRepo, workoutRepo, trainer.ID)
	deletedExercise := seedWorkoutExerciseCatalog(t, db, "Soon Deleted")
	if err := db.Delete(&models.Exercise{}, "id = ?", deletedExercise.ID).Error; err != nil {
		t.Fatalf("soft delete exercise: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, workoutExercisesPath(program.ID, week.ID, workout.ID), `{"exercise_id":"`+deletedExercise.ID+`"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for soft-deleted exercise, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"EXERCISE_NOT_FOUND"`) {
		t.Fatalf("expected EXERCISE_NOT_FOUND, got %s", raw)
	}
}

func TestTrainerWorkoutExerciseAddInvalidBody(t *testing.T) {
	router, _, userRepo, trainerRepo, _, _, _, _, tokenSvc := newTrainerWorkoutExerciseTestRouter(t, trainerroles.PermissionPrograms)
	_, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)

	cases := []struct {
		name string
		body string
	}{
		{name: "not json", body: `not json`},
		{name: "missing exercise_id", body: `{}`},
		{name: "invalid exercise_id", body: `{"exercise_id":"not-a-uuid"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, workoutExercisesPath(uuid.NewString(), uuid.NewString(), uuid.NewString()), tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
				t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
			}
		})
	}
}

func TestTrainerWorkoutExerciseListSuccess(t *testing.T) {
	router, db, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, workoutExerciseRepo, tokenSvc := newTrainerWorkoutExerciseTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program, week, workout := seedWorkoutExerciseProgram(t, programRepo, weekRepo, workoutRepo, trainer.ID)
	squat := seedWorkoutExerciseCatalog(t, db, "Barbell Squat")
	deadlift := seedWorkoutExerciseCatalog(t, db, "Deadlift Variant")
	for _, exercise := range []*models.Exercise{squat, deadlift} {
		entry := &models.WorkoutExercise{ExerciseID: exercise.ID}
		if err := workoutExerciseRepo.AddExercise(context.Background(), trainer.ID, program.ID, week.ID, workout.ID, entry); err != nil {
			t.Fatalf("seed workout exercise: %v", err)
		}
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, workoutExercisesPath(program.ID, week.ID, workout.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	entries, _ := data["exercises"].([]any)
	if len(entries) != 2 {
		t.Fatalf("expected two workout exercises, got %d", len(entries))
	}
	first, _ := entries[0].(map[string]any)
	if position, _ := first["position"].(float64); position != 1 {
		t.Fatalf("expected position 1, got %v", position)
	}
	firstExercise, _ := first["exercise"].(map[string]any)
	if name, _ := firstExercise["name"].(string); name != "Barbell Squat" {
		t.Fatalf("expected embedded name Barbell Squat, got %q", name)
	}
}

func TestTrainerWorkoutExerciseListIDOR(t *testing.T) {
	router, db, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, workoutExerciseRepo, tokenSvc := newTrainerWorkoutExerciseTestRouter(t, trainerroles.PermissionPrograms)
	_, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	foreignProgram, foreignWeek, foreignWorkout := seedWorkoutExerciseProgram(t, programRepo, weekRepo, workoutRepo, otherTrainer.ID)
	squat := seedWorkoutExerciseCatalog(t, db, "Barbell Squat")
	entry := &models.WorkoutExercise{ExerciseID: squat.ID}
	if err := workoutExerciseRepo.AddExercise(context.Background(), otherTrainer.ID, foreignProgram.ID, foreignWeek.ID, foreignWorkout.ID, entry); err != nil {
		t.Fatalf("seed foreign workout exercise: %v", err)
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, workoutExercisesPath(foreignProgram.ID, foreignWeek.ID, foreignWorkout.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with empty list, got %d (body: %s)", rec.Code, raw)
	}
	entries, _ := data["exercises"].([]any)
	if len(entries) != 0 {
		t.Fatalf("foreign workout exercises must never be listed, got %d", len(entries))
	}
}

func TestTrainerWorkoutExerciseGetSuccess(t *testing.T) {
	router, db, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, workoutExerciseRepo, tokenSvc := newTrainerWorkoutExerciseTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program, week, workout := seedWorkoutExerciseProgram(t, programRepo, weekRepo, workoutRepo, trainer.ID)
	squat := seedWorkoutExerciseCatalog(t, db, "Barbell Squat")
	entry := &models.WorkoutExercise{ExerciseID: squat.ID}
	if err := workoutExerciseRepo.AddExercise(context.Background(), trainer.ID, program.ID, week.ID, workout.ID, entry); err != nil {
		t.Fatalf("seed workout exercise: %v", err)
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, workoutExercisePath(program.ID, week.ID, workout.ID, entry.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if id, _ := data["id"].(string); id != entry.ID {
		t.Fatalf("expected entry id %q, got %q", entry.ID, id)
	}
	if position, _ := data["position"].(float64); position != 1 {
		t.Fatalf("expected position 1, got %v", position)
	}
}

func TestTrainerWorkoutExerciseGetIDOR(t *testing.T) {
	router, db, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, workoutExerciseRepo, tokenSvc := newTrainerWorkoutExerciseTestRouter(t, trainerroles.PermissionPrograms)
	_, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	foreignProgram, foreignWeek, foreignWorkout := seedWorkoutExerciseProgram(t, programRepo, weekRepo, workoutRepo, otherTrainer.ID)
	squat := seedWorkoutExerciseCatalog(t, db, "Barbell Squat")
	foreignEntry := &models.WorkoutExercise{ExerciseID: squat.ID}
	if err := workoutExerciseRepo.AddExercise(context.Background(), otherTrainer.ID, foreignProgram.ID, foreignWeek.ID, foreignWorkout.ID, foreignEntry); err != nil {
		t.Fatalf("seed foreign workout exercise: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, workoutExercisePath(foreignProgram.ID, foreignWeek.ID, foreignWorkout.ID, foreignEntry.ID), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for foreign workout exercise, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"WORKOUT_EXERCISE_NOT_FOUND"`) {
		t.Fatalf("expected WORKOUT_EXERCISE_NOT_FOUND, got %s", raw)
	}
}

func TestTrainerWorkoutExerciseGetDeleted(t *testing.T) {
	router, db, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, workoutExerciseRepo, tokenSvc := newTrainerWorkoutExerciseTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program, week, workout := seedWorkoutExerciseProgram(t, programRepo, weekRepo, workoutRepo, trainer.ID)
	squat := seedWorkoutExerciseCatalog(t, db, "Barbell Squat")
	entry := &models.WorkoutExercise{ExerciseID: squat.ID}
	if err := workoutExerciseRepo.AddExercise(context.Background(), trainer.ID, program.ID, week.ID, workout.ID, entry); err != nil {
		t.Fatalf("seed workout exercise: %v", err)
	}
	if err := workoutExerciseRepo.SoftDelete(context.Background(), trainer.ID, program.ID, week.ID, workout.ID, entry.ID); err != nil {
		t.Fatalf("soft delete workout exercise: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, workoutExercisePath(program.ID, week.ID, workout.ID, entry.ID), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for soft-deleted entry, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"WORKOUT_EXERCISE_NOT_FOUND"`) {
		t.Fatalf("expected WORKOUT_EXERCISE_NOT_FOUND, got %s", raw)
	}
}

func TestTrainerWorkoutExerciseUpdatePrescriptionSuccess(t *testing.T) {
	router, db, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, workoutExerciseRepo, tokenSvc := newTrainerWorkoutExerciseTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program, week, workout := seedWorkoutExerciseProgram(t, programRepo, weekRepo, workoutRepo, trainer.ID)
	squat := seedWorkoutExerciseCatalog(t, db, "Barbell Squat")
	entry := &models.WorkoutExercise{ExerciseID: squat.ID}
	if err := workoutExerciseRepo.AddExercise(context.Background(), trainer.ID, program.ID, week.ID, workout.ID, entry); err != nil {
		t.Fatalf("seed workout exercise: %v", err)
	}

	body := `{"instructions":"Brace before every rep.","notes":"Keep the bar over mid-foot.","sets":[{"set_number":99,"set_type":"warmup","reps":12},{"set_number":98,"set_type":"working","reps":8,"weight_kg":80,"rir":1,"rpe":8,"rest_seconds":90,"tempo":"3110"}]}`
	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodPatch, workoutExercisePath(program.ID, week.ID, workout.ID, entry.ID), body)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if instructions, _ := data["instructions"].(string); instructions != "Brace before every rep." {
		t.Fatalf("expected instructions, got %q", instructions)
	}
	if notes, _ := data["notes"].(string); notes != "Keep the bar over mid-foot." {
		t.Fatalf("expected notes, got %q", notes)
	}
	sets, _ := data["sets"].([]any)
	if len(sets) != 2 {
		t.Fatalf("expected 2 sets, got %d", len(sets))
	}
	firstSet, _ := sets[0].(map[string]any)
	secondSet, _ := sets[1].(map[string]any)
	if setNumber, _ := firstSet["set_number"].(float64); setNumber != 1 {
		t.Fatalf("client-provided set_number must be ignored, got %v", firstSet["set_number"])
	}
	if setType, _ := firstSet["set_type"].(string); setType != "warmup" {
		t.Fatalf("expected warmup, got %q", setType)
	}
	if setNumber, _ := secondSet["set_number"].(float64); setNumber != 2 {
		t.Fatalf("expected second set_number 2, got %v", secondSet["set_number"])
	}
	if tempo, _ := secondSet["tempo"].(string); tempo != "3110" {
		t.Fatalf("expected tempo 3110, got %q", tempo)
	}

	rec, data, raw = trainerClientsRequest(router, jwtValue, http.MethodGet, workoutExercisePath(program.ID, week.ID, workout.ID, entry.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on read, got %d (body: %s)", rec.Code, raw)
	}
	if instructions, _ := data["instructions"].(string); instructions != "Brace before every rep." {
		t.Fatalf("prescription was not persisted, got instructions %q", instructions)
	}
	if sets, _ := data["sets"].([]any); len(sets) != 2 {
		t.Fatalf("prescription sets were not persisted, got %d", len(sets))
	}

	var active []models.WorkoutExerciseSet
	if err := db.Where("workout_exercise_id = ?", entry.ID).Order("set_number ASC").Find(&active).Error; err != nil {
		t.Fatalf("load active sets: %v", err)
	}
	if len(active) != 2 || active[0].SetNumber != 1 || active[1].SetNumber != 2 {
		t.Fatalf("unexpected active sets %+v", active)
	}

	rec, _, raw = trainerClientsRequest(router, jwtValue, http.MethodPatch, workoutExercisePath(program.ID, week.ID, workout.ID, entry.ID),
		`{"instructions":"Brace before every rep.","notes":"Keep the bar over mid-foot.","sets":[{"set_type":"failure","reps":6}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on replacement, got %d (body: %s)", rec.Code, raw)
	}

	active = nil
	if err := db.Where("workout_exercise_id = ?", entry.ID).Order("set_number ASC").Find(&active).Error; err != nil {
		t.Fatalf("reload active sets: %v", err)
	}
	if len(active) != 1 || active[0].SetType != "failure" || active[0].SetNumber != 1 {
		t.Fatalf("expected one replacement set, got %+v", active)
	}
	var total int64
	if err := db.Unscoped().Model(&models.WorkoutExerciseSet{}).Where("workout_exercise_id = ?", entry.ID).Count(&total).Error; err != nil {
		t.Fatalf("count historical sets: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected the removed tail set to be preserved, got %d total rows", total)
	}
	var removed int64
	if err := db.Unscoped().Model(&models.WorkoutExerciseSet{}).
		Where("workout_exercise_id = ? AND deleted_at IS NOT NULL", entry.ID).
		Count(&removed).Error; err != nil {
		t.Fatalf("count removed sets: %v", err)
	}
	if removed != 1 {
		t.Fatalf("expected the removed tail to be soft-deleted, got %d", removed)
	}
}

func TestTrainerWorkoutExerciseUpdatePrescriptionIDOR(t *testing.T) {
	router, db, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, workoutExerciseRepo, tokenSvc := newTrainerWorkoutExerciseTestRouter(t, trainerroles.PermissionPrograms)
	_, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	foreignProgram, foreignWeek, foreignWorkout := seedWorkoutExerciseProgram(t, programRepo, weekRepo, workoutRepo, otherTrainer.ID)
	squat := seedWorkoutExerciseCatalog(t, db, "Barbell Squat")
	foreignEntry := &models.WorkoutExercise{ExerciseID: squat.ID}
	if err := workoutExerciseRepo.AddExercise(context.Background(), otherTrainer.ID, foreignProgram.ID, foreignWeek.ID, foreignWorkout.ID, foreignEntry); err != nil {
		t.Fatalf("seed foreign workout exercise: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPatch, workoutExercisePath(foreignProgram.ID, foreignWeek.ID, foreignWorkout.ID, foreignEntry.ID),
		`{"instructions":"Hijacked","notes":"Hijacked","sets":[{"set_type":"working","reps":10}]}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for foreign workout exercise, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"WORKOUT_EXERCISE_NOT_FOUND"`) {
		t.Fatalf("expected WORKOUT_EXERCISE_NOT_FOUND, got %s", raw)
	}

	var stored models.WorkoutExercise
	if err := db.Unscoped().Where("id = ?", foreignEntry.ID).First(&stored).Error; err != nil {
		t.Fatalf("load foreign entry: %v", err)
	}
	if stored.Instructions != "" || stored.Notes != "" {
		t.Fatalf("foreign prescription must remain untouched, got %+v", stored)
	}
}

func TestTrainerWorkoutExerciseUpdatePrescriptionInvalidSet(t *testing.T) {
	router, db, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, workoutExerciseRepo, tokenSvc := newTrainerWorkoutExerciseTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program, week, workout := seedWorkoutExerciseProgram(t, programRepo, weekRepo, workoutRepo, trainer.ID)
	squat := seedWorkoutExerciseCatalog(t, db, "Barbell Squat")
	entry := &models.WorkoutExercise{ExerciseID: squat.ID}
	if err := workoutExerciseRepo.AddExercise(context.Background(), trainer.ID, program.ID, week.ID, workout.ID, entry); err != nil {
		t.Fatalf("seed workout exercise: %v", err)
	}

	for name, body := range map[string]string{
		"missing set type": `{"instructions":"","notes":"","sets":[{"reps":10}]}`,
		"unknown set type": `{"instructions":"","notes":"","sets":[{"set_type":"cardio","reps":10}]}`,
		"zero reps":        `{"instructions":"","notes":"","sets":[{"set_type":"working","reps":0}]}`,
		"invalid rpe":      `{"instructions":"","notes":"","sets":[{"set_type":"working","rpe":11}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPatch, workoutExercisePath(program.ID, week.ID, workout.ID, entry.ID), body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
				t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
			}
		})
	}
}

func TestTrainerWorkoutExerciseReorderSuccess(t *testing.T) {
	router, db, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, workoutExerciseRepo, tokenSvc := newTrainerWorkoutExerciseTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program, week, workout := seedWorkoutExerciseProgram(t, programRepo, weekRepo, workoutRepo, trainer.ID)
	exercises := []*models.Exercise{
		seedWorkoutExerciseCatalog(t, db, "Barbell Squat"),
		seedWorkoutExerciseCatalog(t, db, "Deadlift Variant"),
		seedWorkoutExerciseCatalog(t, db, "Push-Up Variant"),
	}
	var entryIDs []string
	for _, exercise := range exercises {
		entry := &models.WorkoutExercise{ExerciseID: exercise.ID}
		if err := workoutExerciseRepo.AddExercise(context.Background(), trainer.ID, program.ID, week.ID, workout.ID, entry); err != nil {
			t.Fatalf("seed workout exercise: %v", err)
		}
		entryIDs = append(entryIDs, entry.ID)
	}

	order := []string{entryIDs[2], entryIDs[0], entryIDs[1]}
	body := `{"ids":["` + strings.Join(order, `","`) + `"]}`
	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPatch, workoutExercisesOrderPath(program.ID, week.ID, workout.ID), body)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	entries, err := workoutExerciseRepo.ListByWorkout(context.Background(), trainer.ID, program.ID, week.ID, workout.ID)
	if err != nil {
		t.Fatalf("list workout exercises: %v", err)
	}
	if len(entries) != 3 || entries[0].ID != entryIDs[2] || entries[1].ID != entryIDs[0] || entries[2].ID != entryIDs[1] {
		t.Fatalf("expected reordered workout exercises, got %+v", entries)
	}
}

func TestTrainerWorkoutExerciseReorderConflict(t *testing.T) {
	router, db, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, workoutExerciseRepo, tokenSvc := newTrainerWorkoutExerciseTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program, week, workout := seedWorkoutExerciseProgram(t, programRepo, weekRepo, workoutRepo, trainer.ID)
	squat := seedWorkoutExerciseCatalog(t, db, "Barbell Squat")
	entry := &models.WorkoutExercise{ExerciseID: squat.ID}
	if err := workoutExerciseRepo.AddExercise(context.Background(), trainer.ID, program.ID, week.ID, workout.ID, entry); err != nil {
		t.Fatalf("seed workout exercise: %v", err)
	}

	body := `{"ids":["` + uuid.NewString() + `"]}`
	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPatch, workoutExercisesOrderPath(program.ID, week.ID, workout.ID), body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"REORDER_CONFLICT"`) {
		t.Fatalf("expected REORDER_CONFLICT, got %s", raw)
	}
}

func TestTrainerWorkoutExerciseDeleteSuccess(t *testing.T) {
	router, db, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, workoutExerciseRepo, tokenSvc := newTrainerWorkoutExerciseTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program, week, workout := seedWorkoutExerciseProgram(t, programRepo, weekRepo, workoutRepo, trainer.ID)
	squat := seedWorkoutExerciseCatalog(t, db, "Barbell Squat")
	entry := &models.WorkoutExercise{ExerciseID: squat.ID}
	if err := workoutExerciseRepo.AddExercise(context.Background(), trainer.ID, program.ID, week.ID, workout.ID, entry); err != nil {
		t.Fatalf("seed workout exercise: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodDelete, workoutExercisePath(program.ID, week.ID, workout.ID, entry.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	if _, err := workoutExerciseRepo.FindByIDAndWorkout(context.Background(), trainer.ID, program.ID, week.ID, workout.ID, entry.ID); err == nil {
		t.Fatal("expected the workout exercise to be soft-deleted")
	}
}

func TestTrainerWorkoutExerciseNotAuthenticated(t *testing.T) {
	router, _, _, _, _, _, _, _, _ := newTrainerWorkoutExerciseTestRouter(t, trainerroles.PermissionPrograms)

	programID := uuid.NewString()
	weekID := uuid.NewString()
	workoutID := uuid.NewString()
	entryID := uuid.NewString()
	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "add exercise", method: http.MethodPost, path: workoutExercisesPath(programID, weekID, workoutID), body: `{"exercise_id":"` + uuid.NewString() + `"}`},
		{name: "list exercises", method: http.MethodGet, path: workoutExercisesPath(programID, weekID, workoutID)},
		{name: "reorder exercises", method: http.MethodPatch, path: workoutExercisesOrderPath(programID, weekID, workoutID), body: `{"ids":["` + uuid.NewString() + `"]}`},
		{name: "get exercise", method: http.MethodGet, path: workoutExercisePath(programID, weekID, workoutID, entryID)},
		{name: "delete exercise", method: http.MethodDelete, path: workoutExercisePath(programID, weekID, workoutID, entryID)},
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

func TestTrainerWorkoutExerciseAuthenticatedNonTrainer(t *testing.T) {
	router, _, userRepo, _, _, _, _, _, tokenSvc := newTrainerWorkoutExerciseTestRouter(t, trainerroles.PermissionPrograms)
	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	jwtValue, err := tokenSvc.GenerateAccessToken(user.ID, user.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	path := workoutExercisesPath(uuid.NewString(), uuid.NewString(), uuid.NewString())
	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, path, "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"FORBIDDEN"`) {
		t.Fatalf("expected FORBIDDEN, got %s", raw)
	}
}

func TestTrainerWorkoutExercisePermissionNotGranted(t *testing.T) {
	router, _, userRepo, trainerRepo, _, _, _, _, tokenSvc := newTrainerWorkoutExerciseTestRouter(t, trainerroles.Permission("trainer.schedule"))
	_, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, workoutExercisesPath(uuid.NewString(), uuid.NewString(), uuid.NewString()), "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "trainer.schedule") {
		t.Fatalf("forbidden error must not reveal the permission, got %s", raw)
	}
}

func TestTrainerWorkoutExerciseHandlerForwardsContextIdentity(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	programID := uuid.NewString()
	weekID := uuid.NewString()
	workoutID := uuid.NewString()
	svc := &stubWorkoutExerciseService{
		entry: &workout_exercises.WorkoutExercise{
			ID:               uuid.NewString(),
			ProgramWorkoutID: workoutID,
			Position:         1,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}
	router := newTrainerWorkoutExerciseHandlerRouter(svc, identity)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, workoutExercisesPath(programID, weekID, workoutID), `{"exercise_id":"`+uuid.NewString()+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotTrainerID != identity.TrainerID {
		t.Fatalf("expected context trainer %q, got %q", identity.TrainerID, svc.gotTrainerID)
	}
	if svc.gotProgramID != programID || svc.gotWeekID != weekID || svc.gotWorkoutID != workoutID {
		t.Fatalf("expected path identities %q/%q/%q, got %q/%q/%q", programID, weekID, workoutID, svc.gotProgramID, svc.gotWeekID, svc.gotWorkoutID)
	}
}

func TestTrainerWorkoutExerciseHandlerRejectsBodySpoofing(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	programID := uuid.NewString()
	weekID := uuid.NewString()
	workoutID := uuid.NewString()
	svc := &stubWorkoutExerciseService{
		entry: &workout_exercises.WorkoutExercise{
			ID:               uuid.NewString(),
			ProgramWorkoutID: workoutID,
			Position:         1,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}
	router := newTrainerWorkoutExerciseHandlerRouter(svc, identity)

	// A client-supplied trainer_id or workout_id in the body must never be
	// honored: the identity comes from the context and the target workout from
	// the path only.
	body := `{"exercise_id":"` + uuid.NewString() + `","trainer_id":"` + uuid.NewString() + `","workout_id":"` + uuid.NewString() + `","program_id":"` + uuid.NewString() + `","week_id":"` + uuid.NewString() + `"}`
	rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, workoutExercisesPath(programID, weekID, workoutID), body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotTrainerID != identity.TrainerID {
		t.Fatalf("spoofed trainer id must be ignored, expected %q, got %q", identity.TrainerID, svc.gotTrainerID)
	}
	if svc.gotWorkoutID != workoutID {
		t.Fatalf("spoofed workout id must be ignored, expected %q, got %q", workoutID, svc.gotWorkoutID)
	}
	if svc.gotProgramID != programID || svc.gotWeekID != weekID {
		t.Fatalf("spoofed program/week ids must be ignored, expected %q/%q, got %q/%q", programID, weekID, svc.gotProgramID, svc.gotWeekID)
	}
}

func TestTrainerWorkoutExerciseHandlerForwardsPathIdentities(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	programID := uuid.NewString()
	weekID := uuid.NewString()
	workoutID := uuid.NewString()
	entryID := uuid.NewString()
	svc := &stubWorkoutExerciseService{
		entry: &workout_exercises.WorkoutExercise{
			ID:               entryID,
			ProgramWorkoutID: workoutID,
			Position:         1,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}
	router := newTrainerWorkoutExerciseHandlerRouter(svc, identity)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, workoutExercisePath(programID, weekID, workoutID, entryID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotTrainerID != identity.TrainerID {
		t.Fatalf("expected context trainer %q, got %q", identity.TrainerID, svc.gotTrainerID)
	}
	if svc.gotProgramID != programID || svc.gotWeekID != weekID || svc.gotWorkoutID != workoutID || svc.gotEntryID != entryID {
		t.Fatalf("expected path identities %q/%q/%q/%q, got %q/%q/%q/%q", programID, weekID, workoutID, entryID, svc.gotProgramID, svc.gotWeekID, svc.gotWorkoutID, svc.gotEntryID)
	}
}

func TestTrainerWorkoutExerciseHandlerForwardsOrder(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	programID := uuid.NewString()
	weekID := uuid.NewString()
	workoutID := uuid.NewString()
	order := []string{uuid.NewString(), uuid.NewString()}
	svc := &stubWorkoutExerciseService{}
	router := newTrainerWorkoutExerciseHandlerRouter(svc, identity)

	body := `{"ids":["` + strings.Join(order, `","`) + `"]}`
	rec, _, raw := trainerClientsRequest(router, "", http.MethodPatch, workoutExercisesOrderPath(programID, weekID, workoutID), body)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotProgramID != programID || svc.gotWeekID != weekID || svc.gotWorkoutID != workoutID {
		t.Fatalf("expected path identities %q/%q/%q, got %q/%q/%q", programID, weekID, workoutID, svc.gotProgramID, svc.gotWeekID, svc.gotWorkoutID)
	}
	if len(svc.gotOrder) != 2 || svc.gotOrder[0] != order[0] || svc.gotOrder[1] != order[1] {
		t.Fatalf("expected order %v, got %v", order, svc.gotOrder)
	}
}

func TestTrainerWorkoutExerciseHandlerMissingContext(t *testing.T) {
	router := newTrainerWorkoutExerciseHandlerRouter(&stubWorkoutExerciseService{}, nil)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, workoutExercisesPath(uuid.NewString(), uuid.NewString(), uuid.NewString()), "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestTrainerWorkoutExerciseHandlerErrorMapping(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}

	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid input", err: workout_exercises.ErrInvalidInput, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
		{name: "workout not found", err: workout_exercises.ErrWorkoutNotFound, status: http.StatusNotFound, code: "WORKOUT_NOT_FOUND"},
		{name: "exercise not found", err: workout_exercises.ErrExerciseNotFound, status: http.StatusNotFound, code: "EXERCISE_NOT_FOUND"},
		{name: "entry not found", err: workout_exercises.ErrWorkoutExerciseNotFound, status: http.StatusNotFound, code: "WORKOUT_EXERCISE_NOT_FOUND"},
		{name: "reorder conflict", err: workout_exercises.ErrReorderConflict, status: http.StatusConflict, code: "REORDER_CONFLICT"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubWorkoutExerciseService{err: tc.err}
			router := newTrainerWorkoutExerciseHandlerRouter(svc, identity)

			rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, workoutExercisesPath(uuid.NewString(), uuid.NewString(), uuid.NewString()), `{"exercise_id":"`+uuid.NewString()+`"}`)
			if rec.Code != tc.status {
				t.Fatalf("expected %d, got %d (body: %s)", tc.status, rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"`+tc.code+`"`) {
				t.Fatalf("expected code %s, got %s", tc.code, raw)
			}
		})
	}
}

func TestTrainerWorkoutExerciseHandlerRepositoryFailureNotExposed(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	svc := &stubWorkoutExerciseService{err: errLoginRepoFailure}
	router := newTrainerWorkoutExerciseHandlerRouter(svc, identity)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, workoutExercisesPath(uuid.NewString(), uuid.NewString(), uuid.NewString()), "")
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

func TestTrainerWorkoutExerciseNeverExposesSensitiveData(t *testing.T) {
	router, db, userRepo, trainerRepo, programRepo, weekRepo, workoutRepo, workoutExerciseRepo, tokenSvc := newTrainerWorkoutExerciseTestRouter(t, trainerroles.PermissionPrograms)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	program, week, workout := seedWorkoutExerciseProgram(t, programRepo, weekRepo, workoutRepo, trainer.ID)
	squat := seedWorkoutExerciseCatalog(t, db, "Barbell Squat")
	entry := &models.WorkoutExercise{ExerciseID: squat.ID}
	if err := workoutExerciseRepo.AddExercise(context.Background(), trainer.ID, program.ID, week.ID, workout.ID, entry); err != nil {
		t.Fatalf("seed workout exercise: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, workoutExercisesPath(program.ID, week.ID, workout.ID), "")
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
