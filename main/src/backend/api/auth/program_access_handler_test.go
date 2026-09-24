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
	"ryze/backend/services/program_access"
	"ryze/backend/services/public_programs"
	"ryze/backend/services/token"
)

const accessRoutePrefix = "/api/v1/me/programs/"

// newProgramAccessTestRouter wires the protected program access endpoint behind
// the real Authenticate middleware, backed by a database transaction so created
// records are rolled back.
func newProgramAccessTestRouter(t *testing.T) (*gin.Engine, repositories.UserRepository, *gorm.DB, token.Service) {
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
	trainerProgramRepo := repositories.NewProgramRepository(tx)
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)

	publicService := public_programs.NewService(trainerProgramRepo)
	accessService := program_access.NewService(entitlementRepo, publicService)
	handler := auth.NewProgramAccessHandler(accessService)

	router := gin.New()
	me := router.Group("/api/v1/me")
	me.Use(middleware.Authenticate(tokenSvc, userRepo))
	me.GET("/programs/:programID", handler.GetProgramAccess)

	return router, userRepo, tx, tokenSvc
}

// newProgramAccessHandlerRouter mounts only the handler with a pre-set
// authentication-context identity, so the handler's own error mapping and
// identity forwarding can be tested without the full middleware chain. nil
// identity simulates a missing context.
func newProgramAccessHandlerRouter(svc program_access.Service, identity any) *gin.Engine {
	handler := auth.NewProgramAccessHandler(svc)
	router := gin.New()
	me := router.Group("/api/v1/me")
	me.Use(func(c *gin.Context) {
		if identity != nil {
			c.Set(authcontext.UserIDContextKey, identity)
		}
		c.Next()
	})
	me.GET("/programs/:programID", handler.GetProgramAccess)
	return router
}

// stubProgramAccessService is a scripted fake used to exercise the handler's
// error mapping and identity forwarding without touching the database.
type stubProgramAccessService struct {
	detail     *public_programs.ProgramDetail
	err        error
	gotUser    string
	gotProgram string
}

func (s *stubProgramAccessService) GetProgramAccess(_ context.Context, userID, programID string) (*public_programs.ProgramDetail, error) {
	s.gotUser = userID
	s.gotProgram = programID
	return s.detail, s.err
}

// seedAccessProgram builds a trainer-owned program (published by default) with
// one week, one workout and one catalog exercise, ready to be entitled. The
// exercise name must be unique per test because the active exercise name is
// unique in the catalog.
func seedAccessProgram(t *testing.T, tx *gorm.DB, programName, exerciseName, status string) (*models.Program, *models.Trainer) {
	t.Helper()
	ctx := context.Background()

	trainerUser := seedLoginUser(t, repositories.NewUserRepository(tx), uniqueEmail(), "Password123!")
	trainer := seedTrainerForUser(t, repositories.NewTrainerRepository(tx), trainerUser)

	weekRepo := repositories.NewProgramWeekRepository(tx)
	workoutRepo := repositories.NewProgramWorkoutRepository(tx)
	workoutExerciseRepo := repositories.NewWorkoutExerciseRepository(tx)
	programRepo := repositories.NewProgramRepository(tx)

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

	program := &models.Program{TrainerID: trainer.ID, Name: programName, Type: models.ProgramTypePremium, Status: status}
	if err := programRepo.Create(ctx, program); err != nil {
		t.Fatalf("seed program: %v", err)
	}

	week := &models.ProgramWeek{}
	if err := weekRepo.Create(ctx, trainer.ID, program.ID, week); err != nil {
		t.Fatalf("seed week: %v", err)
	}

	workout := &models.ProgramWorkout{}
	if err := workoutRepo.Create(ctx, trainer.ID, program.ID, week.ID, workout); err != nil {
		t.Fatalf("seed workout: %v", err)
	}

	if err := workoutExerciseRepo.AddExercise(ctx, trainer.ID, program.ID, week.ID, workout.ID, &models.WorkoutExercise{ExerciseID: exercise.ID}); err != nil {
		t.Fatalf("seed workout exercise: %v", err)
	}

	return program, trainer
}

func TestProgramAccessReadSuccess(t *testing.T) {
	router, userRepo, tx, tokenSvc := newProgramAccessTestRouter(t)
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	ctx := context.Background()

	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	program, _ := seedAccessProgram(t, tx, "Strength Builder", "Bench Press", models.ProgramStatusPublished)

	if err := entitlementRepo.Create(ctx, clientUser.ID, program.ID, &models.Entitlement{}); err != nil {
		t.Fatalf("seed entitlement: %v", err)
	}

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, data, raw := entitlementsRequest(router, jwtValue, http.MethodGet, accessRoutePrefix+program.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	if id, _ := data["id"].(string); id != program.ID {
		t.Fatalf("expected program id %q, got %q", program.ID, id)
	}
	if name, _ := data["name"].(string); name != "Strength Builder" {
		t.Fatalf("expected program name, got %q", name)
	}

	weeks, _ := data["weeks"].([]any)
	if len(weeks) != 1 {
		t.Fatalf("expected 1 week, got %d", len(weeks))
	}
	firstWeek, _ := weeks[0].(map[string]any)
	workouts, _ := firstWeek["workouts"].([]any)
	if len(workouts) != 1 {
		t.Fatalf("expected 1 workout, got %d", len(workouts))
	}
	firstWorkout, _ := workouts[0].(map[string]any)
	workoutExercises, _ := firstWorkout["exercises"].([]any)
	if len(workoutExercises) != 1 {
		t.Fatalf("expected 1 workout exercise, got %d", len(workoutExercises))
	}
	firstExercise, _ := workoutExercises[0].(map[string]any)
	if exerciseName, _ := firstExercise["name"].(string); exerciseName != "Bench Press" {
		t.Fatalf("expected exercise name, got %q", exerciseName)
	}
}

func TestProgramAccessReadUnauthenticated(t *testing.T) {
	router, _, _, _ := newProgramAccessTestRouter(t)

	for _, programID := range []string{"not-a-uuid", "44444444-4444-4444-4444-444444444444"} {
		rec, _, raw := entitlementsRequest(router, "", http.MethodGet, accessRoutePrefix+programID, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for %q, got %d (body: %s)", programID, rec.Code, raw)
		}
		if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
			t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
		}
	}
}

func TestProgramAccessReadNoEntitlement(t *testing.T) {
	router, userRepo, tx, tokenSvc := newProgramAccessTestRouter(t)

	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	program, _ := seedAccessProgram(t, tx, "Strength Builder", "Bench Press", models.ProgramStatusPublished)

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := entitlementsRequest(router, jwtValue, http.MethodGet, accessRoutePrefix+program.ID, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 without an entitlement, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"PROGRAM_NOT_FOUND"`) {
		t.Fatalf("expected PROGRAM_NOT_FOUND, got %s", raw)
	}
	// The program content must never leak without access.
	if strings.Contains(raw, "Strength Builder") || strings.Contains(raw, "Bench Press") {
		t.Fatalf("program content must never leak without access, got %s", raw)
	}
}

func TestProgramAccessReadCrossUserIsolation(t *testing.T) {
	router, userRepo, tx, tokenSvc := newProgramAccessTestRouter(t)
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	ctx := context.Background()

	ownerUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	otherUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	program, _ := seedAccessProgram(t, tx, "Strength Builder", "Bench Press", models.ProgramStatusPublished)

	if err := entitlementRepo.Create(ctx, ownerUser.ID, program.ID, &models.Entitlement{}); err != nil {
		t.Fatalf("seed entitlement: %v", err)
	}

	jwtOwner, err := tokenSvc.GenerateAccessToken(ownerUser.ID, ownerUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken owner: %v", err)
	}
	jwtOther, err := tokenSvc.GenerateAccessToken(otherUser.ID, otherUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken other: %v", err)
	}

	recOwner, _, rawOwner := entitlementsRequest(router, jwtOwner, http.MethodGet, accessRoutePrefix+program.ID, "")
	if recOwner.Code != http.StatusOK {
		t.Fatalf("owner expected 200, got %d (body: %s)", recOwner.Code, rawOwner)
	}

	recOther, _, rawOther := entitlementsRequest(router, jwtOther, http.MethodGet, accessRoutePrefix+program.ID, "")
	if recOther.Code != http.StatusNotFound {
		t.Fatalf("non-owner expected 404, got %d (body: %s)", recOther.Code, rawOther)
	}
	if strings.Contains(rawOther, "Strength Builder") {
		t.Fatalf("another user must never read the entitled program, got %s", rawOther)
	}
}

func TestProgramAccessReadUnpublishedProgram(t *testing.T) {
	router, userRepo, tx, tokenSvc := newProgramAccessTestRouter(t)
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	ctx := context.Background()

	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	program, _ := seedAccessProgram(t, tx, "Draft Plan", "Bench Press", models.ProgramStatusDraft)

	if err := entitlementRepo.Create(ctx, clientUser.ID, program.ID, &models.Entitlement{}); err != nil {
		t.Fatalf("seed entitlement: %v", err)
	}

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := entitlementsRequest(router, jwtValue, http.MethodGet, accessRoutePrefix+program.ID, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for an unpublished program, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "Draft Plan") {
		t.Fatalf("unpublished program content must never leak, got %s", raw)
	}
}

func TestProgramAccessReadSoftDeletedProgram(t *testing.T) {
	router, userRepo, tx, tokenSvc := newProgramAccessTestRouter(t)
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	programRepo := repositories.NewProgramRepository(tx)
	ctx := context.Background()

	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	program, trainer := seedAccessProgram(t, tx, "Removed Plan", "Bench Press", models.ProgramStatusPublished)

	if err := entitlementRepo.Create(ctx, clientUser.ID, program.ID, &models.Entitlement{}); err != nil {
		t.Fatalf("seed entitlement: %v", err)
	}

	if err := programRepo.SoftDelete(ctx, trainer.ID, program.ID); err != nil {
		t.Fatalf("soft delete program: %v", err)
	}

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := entitlementsRequest(router, jwtValue, http.MethodGet, accessRoutePrefix+program.ID, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after soft delete, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "Removed Plan") {
		t.Fatalf("soft-deleted program content must never leak, got %s", raw)
	}
}

func TestProgramAccessHandlerForwardsContextIdentity(t *testing.T) {
	identity := uuid.NewString()
	svc := &stubProgramAccessService{
		detail: &public_programs.ProgramDetail{
			Program: public_programs.Program{ID: uuid.NewString(), Name: "Strength Builder", Status: models.ProgramStatusPublished},
		},
	}
	router := newProgramAccessHandlerRouter(svc, identity)

	path := accessRoutePrefix + uuid.NewString()
	rec, _, raw := entitlementsRequest(router, "", http.MethodGet, path, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotUser != identity {
		t.Fatalf("expected context user %q, got %q", identity, svc.gotUser)
	}
}

func TestProgramAccessHandlerMissingContext(t *testing.T) {
	router := newProgramAccessHandlerRouter(&stubProgramAccessService{}, nil)

	rec, _, raw := entitlementsRequest(router, "", http.MethodGet, accessRoutePrefix+uuid.NewString(), "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestProgramAccessHandlerErrorMapping(t *testing.T) {
	identity := uuid.NewString()

	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid input", err: program_access.ErrInvalidInput, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
		{name: "not accessible", err: program_access.ErrProgramNotAccessible, status: http.StatusNotFound, code: "PROGRAM_NOT_FOUND"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubProgramAccessService{err: tc.err}
			router := newProgramAccessHandlerRouter(svc, identity)

			rec, _, raw := entitlementsRequest(router, "", http.MethodGet, accessRoutePrefix+uuid.NewString(), "")
			if rec.Code != tc.status {
				t.Fatalf("expected %d, got %d (body: %s)", tc.status, rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"`+tc.code+`"`) {
				t.Fatalf("expected code %s, got %s", tc.code, raw)
			}
		})
	}
}

func TestProgramAccessHandlerRepositoryFailureNotExposed(t *testing.T) {
	svc := &stubProgramAccessService{err: errLoginRepoFailure}
	router := newProgramAccessHandlerRouter(svc, uuid.NewString())

	rec, _, raw := entitlementsRequest(router, "", http.MethodGet, accessRoutePrefix+uuid.NewString(), "")
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

func TestProgramAccessReadNeverExposesSecrets(t *testing.T) {
	router, userRepo, tx, tokenSvc := newProgramAccessTestRouter(t)
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	ctx := context.Background()

	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	program, _ := seedAccessProgram(t, tx, "Strength Builder", "Bench Press", models.ProgramStatusPublished)

	ent := &models.Entitlement{}
	if err := entitlementRepo.Create(ctx, clientUser.ID, program.ID, ent); err != nil {
		t.Fatalf("seed entitlement: %v", err)
	}

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := entitlementsRequest(router, jwtValue, http.MethodGet, accessRoutePrefix+program.ID, "")
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
		clientUser.ID,
		clientUser.Email,
		ent.ID,
		"platform_amount",
		"trainer_amount",
		"commission",
		"refund",
	} {
		if strings.Contains(raw, sensitive) {
			t.Fatalf("response must never contain %q", sensitive)
		}
	}
}
