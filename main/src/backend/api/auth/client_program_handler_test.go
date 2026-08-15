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
	"ryze/backend/services/client_programs"
	"ryze/backend/services/token"
)

const clientProgramRoute = "/api/v1/me/program"

// newClientProgramTestRouter wires the client assigned-program endpoint behind
// the real Authenticate middleware, backed by a database transaction so created
// records are rolled back.
func newClientProgramTestRouter(t *testing.T) (*gin.Engine, repositories.UserRepository, *gorm.DB, token.Service) {
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
	programAssignmentRepo := repositories.NewProgramAssignmentRepository(tx)
	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)

	service := client_programs.NewService(programAssignmentRepo)
	handler := auth.NewClientProgramHandler(service)

	router := gin.New()
	me := router.Group("/api/v1/me")
	me.Use(middleware.Authenticate(tokenSvc, userRepo))
	me.GET("/program", handler.GetProgram)

	return router, userRepo, tx, tokenSvc
}

// newClientProgramHandlerRouter mounts only the handler with a pre-set
// authentication-context identity, so the handler's own error mapping and
// identity forwarding can be tested without the full middleware chain. nil
// identity simulates a missing context.
func newClientProgramHandlerRouter(svc client_programs.Service, identity any) *gin.Engine {
	handler := auth.NewClientProgramHandler(svc)
	router := gin.New()
	me := router.Group("/api/v1/me")
	me.Use(func(c *gin.Context) {
		if identity != nil {
			c.Set(authcontext.UserIDContextKey, identity)
		}
		c.Next()
	})
	me.GET("/program", handler.GetProgram)
	return router
}

// stubClientProgramService is a scripted fake used to exercise the handler's
// error mapping and identity forwarding without touching the database.
type stubClientProgramService struct {
	program *client_programs.Program
	err     error
	gotUser string
}

func (s *stubClientProgramService) GetProgram(_ context.Context, userID string) (*client_programs.Program, error) {
	s.gotUser = userID
	return s.program, s.err
}

// seedClientAssignedProgram builds a trainer-owned program with one week, one
// workout, one workout exercise and one catalog exercise, and assigns it to the
// given client user. The exercise name must be unique per test because the
// active exercise name is unique in the catalog.
func seedClientAssignedProgram(t *testing.T, tx *gorm.DB, trainerID, clientUserID, programName, exerciseName string) (*models.Program, *models.Exercise) {
	t.Helper()
	ctx := context.Background()

	clientRepo := repositories.NewTrainerClientRepository(tx)
	programRepo := repositories.NewProgramRepository(tx)
	assignmentRepo := repositories.NewProgramAssignmentRepository(tx)
	weekRepo := repositories.NewProgramWeekRepository(tx)
	workoutRepo := repositories.NewProgramWorkoutRepository(tx)
	workoutExerciseRepo := repositories.NewWorkoutExerciseRepository(tx)

	exercise := &models.Exercise{
		Name:          exerciseName,
		Description:   "A chest press",
		TargetMuscles: "Chest",
		Equipment:     "Barbell",
		Difficulty:    "Intermediate",
		VideoURL:      "https://example.com/video",
		ImageURL:      "https://example.com/image",
	}
	if err := tx.Create(exercise).Error; err != nil {
		t.Fatalf("seed exercise: %v", err)
	}

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

	if err := workoutExerciseRepo.AddExercise(ctx, trainerID, program.ID, week.ID, workout.ID, &models.WorkoutExercise{ExerciseID: exercise.ID}); err != nil {
		t.Fatalf("seed workout exercise: %v", err)
	}

	if err := assignmentRepo.Create(ctx, trainerID, clientUserID, program.ID, &models.ProgramAssignment{}); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}

	return program, exercise
}

func TestClientProgramReadSuccess(t *testing.T) {
	router, userRepo, tx, tokenSvc := newClientProgramTestRouter(t)
	trainer := seedTrainerForUser(t, repositories.NewTrainerRepository(tx), seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	program, exercise := seedClientAssignedProgram(t, tx, trainer.ID, clientUser.ID, "Strength Builder", "Bench Press")

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, clientProgramRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	if id, _ := data["id"].(string); id != program.ID {
		t.Fatalf("expected program id %q, got %q", program.ID, id)
	}
	if name, _ := data["name"].(string); name != "Strength Builder" {
		t.Fatalf("expected program name, got %q", name)
	}
	if status, _ := data["status"].(string); status != models.ProgramStatusDraft {
		t.Fatalf("a draft program must be readable, got status %q", status)
	}

	weeks, _ := data["weeks"].([]any)
	if len(weeks) != 1 {
		t.Fatalf("expected 1 week, got %d", len(weeks))
	}
	firstWeek, _ := weeks[0].(map[string]any)
	if weekNumber, _ := firstWeek["week_number"].(float64); weekNumber != 1 {
		t.Fatalf("expected week_number 1, got %v", firstWeek["week_number"])
	}

	workouts, _ := firstWeek["workouts"].([]any)
	if len(workouts) != 1 {
		t.Fatalf("expected 1 workout, got %d", len(workouts))
	}
	firstWorkout, _ := workouts[0].(map[string]any)
	if position, _ := firstWorkout["position"].(float64); position != 1 {
		t.Fatalf("expected position 1, got %v", firstWorkout["position"])
	}

	workoutExercises, _ := firstWorkout["exercises"].([]any)
	if len(workoutExercises) != 1 {
		t.Fatalf("expected 1 workout exercise, got %d", len(workoutExercises))
	}
	firstWorkoutExercise, _ := workoutExercises[0].(map[string]any)
	nested, _ := firstWorkoutExercise["exercise"].(map[string]any)
	if exerciseName, _ := nested["name"].(string); exerciseName != exercise.Name {
		t.Fatalf("expected exercise name %q, got %q", exercise.Name, exerciseName)
	}
	if exerciseID, _ := nested["id"].(string); exerciseID != exercise.ID {
		t.Fatalf("expected exercise id %q, got %q", exercise.ID, exerciseID)
	}

	// The owning trainer and every parent identifier are never exposed to the
	// client.
	if _, exists := data["trainer_id"]; exists {
		t.Fatal("response must never expose the owning trainer id")
	}
	if _, exists := firstWeek["program_id"]; exists {
		t.Fatal("a week must never expose its program id")
	}
	if _, exists := firstWorkout["program_week_id"]; exists {
		t.Fatal("a workout must never expose its week id")
	}
	if _, exists := firstWorkoutExercise["program_workout_id"]; exists {
		t.Fatal("a workout exercise must never expose its workout id")
	}
}

func TestClientProgramReadEmptyStructure(t *testing.T) {
	router, userRepo, tx, tokenSvc := newClientProgramTestRouter(t)
	trainer := seedTrainerForUser(t, repositories.NewTrainerRepository(tx), seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	ctx := context.Background()

	clientRepo := repositories.NewTrainerClientRepository(tx)
	programRepo := repositories.NewProgramRepository(tx)
	assignmentRepo := repositories.NewProgramAssignmentRepository(tx)

	program := &models.Program{TrainerID: trainer.ID, Name: "Empty Program", Type: models.ProgramTypeFree, Status: models.ProgramStatusPublished}
	if err := programRepo.Create(ctx, program); err != nil {
		t.Fatalf("seed program: %v", err)
	}
	if err := clientRepo.Create(ctx, &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}
	if err := assignmentRepo.Create(ctx, trainer.ID, clientUser.ID, program.ID, &models.ProgramAssignment{}); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, clientProgramRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"weeks":[]`) {
		t.Fatalf("expected an empty weeks array, got %s", raw)
	}
}

func TestClientProgramReadUnauthenticated(t *testing.T) {
	router, _, _, _ := newClientProgramTestRouter(t)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, clientProgramRoute, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestClientProgramReadNoAssignment(t *testing.T) {
	router, userRepo, _, tokenSvc := newClientProgramTestRouter(t)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, clientProgramRoute, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"PROGRAM_NOT_FOUND"`) {
		t.Fatalf("expected PROGRAM_NOT_FOUND, got %s", raw)
	}
}

func TestClientProgramReadSoftDeletedAssignment(t *testing.T) {
	router, userRepo, tx, tokenSvc := newClientProgramTestRouter(t)
	trainer := seedTrainerForUser(t, repositories.NewTrainerRepository(tx), seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	program, _ := seedClientAssignedProgram(t, tx, trainer.ID, clientUser.ID, "Strength Builder", "Bench Press")

	ctx := context.Background()
	assignmentRepo := repositories.NewProgramAssignmentRepository(tx)
	assignments, err := assignmentRepo.ListByClient(ctx, trainer.ID, clientUser.ID)
	if err != nil || len(assignments) != 1 {
		t.Fatalf("expected one assignment, got %d (err: %v)", len(assignments), err)
	}
	if err := assignmentRepo.SoftDelete(ctx, trainer.ID, clientUser.ID, assignments[0].ID); err != nil {
		t.Fatalf("soft delete assignment: %v", err)
	}

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, clientProgramRoute, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after soft delete, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"PROGRAM_NOT_FOUND"`) {
		t.Fatalf("expected PROGRAM_NOT_FOUND, got %s", raw)
	}
	if strings.Contains(raw, program.Name) {
		t.Fatalf("a soft-deleted assignment must not leak program data, got %s", raw)
	}
}

func TestClientProgramReadIDOR(t *testing.T) {
	router, userRepo, tx, tokenSvc := newClientProgramTestRouter(t)
	trainerRepo := repositories.NewTrainerRepository(tx)

	trainerA := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	trainerB := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientA := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	clientB := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	seedClientAssignedProgram(t, tx, trainerA.ID, clientA.ID, "Program A", "Press A")
	seedClientAssignedProgram(t, tx, trainerB.ID, clientB.ID, "Program B", "Press B")

	jwtA, err := tokenSvc.GenerateAccessToken(clientA.ID, clientA.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken A: %v", err)
	}
	jwtB, err := tokenSvc.GenerateAccessToken(clientB.ID, clientB.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken B: %v", err)
	}

	recA, _, rawA := trainerClientsRequest(router, jwtA, http.MethodGet, clientProgramRoute, "")
	if recA.Code != http.StatusOK {
		t.Fatalf("expected 200 for A, got %d (body: %s)", recA.Code, rawA)
	}
	if !strings.Contains(rawA, "Program A") {
		t.Fatalf("expected Program A, got %s", rawA)
	}
	if strings.Contains(rawA, "Program B") {
		t.Fatalf("client A must never see Program B, got %s", rawA)
	}

	recB, _, rawB := trainerClientsRequest(router, jwtB, http.MethodGet, clientProgramRoute, "")
	if recB.Code != http.StatusOK {
		t.Fatalf("expected 200 for B, got %d (body: %s)", recB.Code, rawB)
	}
	if !strings.Contains(rawB, "Program B") {
		t.Fatalf("expected Program B, got %s", rawB)
	}
	if strings.Contains(rawB, "Program A") {
		t.Fatalf("client B must never see Program A, got %s", rawB)
	}
}

func TestClientProgramReadIgnoresClientSuppliedIdentity(t *testing.T) {
	router, userRepo, tx, tokenSvc := newClientProgramTestRouter(t)
	trainer := seedTrainerForUser(t, repositories.NewTrainerRepository(tx), seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	otherUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	seedClientAssignedProgram(t, tx, trainer.ID, clientUser.ID, "Strength Builder", "Bench Press")

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	// A client-supplied user_id, trainer_id or client_id in the query must be
	// ignored: the assigned program is always resolved from the
	// authentication context.
	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet,
		clientProgramRoute+"?user_id="+otherUser.ID+"&trainer_id="+uuid.NewString()+"&client_id="+otherUser.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if name, _ := data["name"].(string); name != "Strength Builder" {
		t.Fatalf("expected the authenticated user's assigned program, got %q", name)
	}
}

func TestClientProgramHandlerForwardsContextIdentity(t *testing.T) {
	identity := uuid.NewString()
	svc := &stubClientProgramService{
		program: &client_programs.Program{
			ID:     uuid.NewString(),
			Name:   "Strength Builder",
			Type:   models.ProgramTypePremium,
			Status: models.ProgramStatusDraft,
			Weeks:  []client_programs.Week{},
		},
	}
	router := newClientProgramHandlerRouter(svc, identity)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, clientProgramRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotUser != identity {
		t.Fatalf("expected context user %q, got %q", identity, svc.gotUser)
	}
}

func TestClientProgramHandlerMissingContext(t *testing.T) {
	router := newClientProgramHandlerRouter(&stubClientProgramService{}, nil)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, clientProgramRoute, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestClientProgramHandlerErrorMapping(t *testing.T) {
	identity := uuid.NewString()

	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid input", err: client_programs.ErrInvalidInput, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
		{name: "program not found", err: client_programs.ErrProgramNotFound, status: http.StatusNotFound, code: "PROGRAM_NOT_FOUND"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubClientProgramService{err: tc.err}
			router := newClientProgramHandlerRouter(svc, identity)

			rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, clientProgramRoute, "")
			if rec.Code != tc.status {
				t.Fatalf("expected %d, got %d (body: %s)", tc.status, rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"`+tc.code+`"`) {
				t.Fatalf("expected code %s, got %s", tc.code, raw)
			}
		})
	}
}

func TestClientProgramHandlerRepositoryFailureNotExposed(t *testing.T) {
	svc := &stubClientProgramService{err: errLoginRepoFailure}
	router := newClientProgramHandlerRouter(svc, uuid.NewString())

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, clientProgramRoute, "")
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

func TestClientProgramReadNeverExposesSecrets(t *testing.T) {
	router, userRepo, tx, tokenSvc := newClientProgramTestRouter(t)
	trainer := seedTrainerForUser(t, repositories.NewTrainerRepository(tx), seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	seedClientAssignedProgram(t, tx, trainer.ID, clientUser.ID, "Strength Builder", "Bench Press")

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, clientProgramRoute, "")
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
