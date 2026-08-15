package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"ryze/backend/api/auth"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

// e2eUserID resolves the authenticated user id through the real /me endpoint.
func e2eUserID(t *testing.T, router http.Handler, cookie string) string {
	t.Helper()

	rec, raw := e2eRequest(router, auth.AccessTokenCookieName, cookie, http.MethodGet, "/api/v1/me", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("me: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	var payload struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("invalid JSON: %v (body: %s)", err, raw)
	}
	if payload.Data.ID == "" {
		t.Fatalf("unexpected me payload (body: %s)", raw)
	}
	return payload.Data.ID
}

// TestClientProgramReadE2E exercises the client assigned-program read through
// the complete router: a real registration, login and authentication flow, a
// trainer-owned program with one full structural level, and the
// /api/v1/me/program response. It also verifies that a user without an active
// assignment gets the single indistinguishable 404 and learns nothing.
func TestClientProgramReadE2E(t *testing.T) {
	router, tx, _ := newE2ERouter(t)
	ctx := context.Background()

	trainerRepo := repositories.NewTrainerRepository(tx)
	clientRepo := repositories.NewTrainerClientRepository(tx)
	programRepo := repositories.NewProgramRepository(tx)
	assignmentRepo := repositories.NewProgramAssignmentRepository(tx)
	weekRepo := repositories.NewProgramWeekRepository(tx)
	workoutRepo := repositories.NewProgramWorkoutRepository(tx)
	workoutExerciseRepo := repositories.NewWorkoutExerciseRepository(tx)

	trainerCookie := e2eRegisterUser(t, router, e2eUniqueEmail())
	trainer := &models.Trainer{UserID: e2eUserID(t, router, trainerCookie)}
	if err := trainerRepo.Create(ctx, trainer); err != nil {
		t.Fatalf("create trainer: %v", err)
	}

	clientCookie := e2eRegisterUser(t, router, e2eUniqueEmail())
	clientUserID := e2eUserID(t, router, clientCookie)

	if err := clientRepo.Create(ctx, &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUserID}); err != nil {
		t.Fatalf("create trainer-client relationship: %v", err)
	}

	exercise := &models.Exercise{
		Name:          "Bench Press",
		Description:   "A chest press",
		TargetMuscles: "Chest",
		Equipment:     "Barbell",
		Difficulty:    "Intermediate",
	}
	if err := tx.Create(exercise).Error; err != nil {
		t.Fatalf("create exercise: %v", err)
	}

	program := &models.Program{TrainerID: trainer.ID, Name: "Strength Builder", Type: models.ProgramTypePremium, Status: models.ProgramStatusDraft}
	if err := programRepo.Create(ctx, program); err != nil {
		t.Fatalf("create program: %v", err)
	}

	week := &models.ProgramWeek{}
	if err := weekRepo.Create(ctx, trainer.ID, program.ID, week); err != nil {
		t.Fatalf("create program week: %v", err)
	}
	workout := &models.ProgramWorkout{}
	if err := workoutRepo.Create(ctx, trainer.ID, program.ID, week.ID, workout); err != nil {
		t.Fatalf("create program workout: %v", err)
	}
	if err := workoutExerciseRepo.AddExercise(ctx, trainer.ID, program.ID, week.ID, workout.ID, &models.WorkoutExercise{ExerciseID: exercise.ID}); err != nil {
		t.Fatalf("add workout exercise: %v", err)
	}

	if err := assignmentRepo.Create(ctx, trainer.ID, clientUserID, program.ID, &models.ProgramAssignment{}); err != nil {
		t.Fatalf("assign program: %v", err)
	}

	// The client reads the assigned program through the complete router.
	rec, raw := e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodGet, "/api/v1/me/program", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	for _, want := range []string{
		program.ID,
		"Strength Builder",
		"Bench Press",
		`"weeks":[`,
		`"workouts":[`,
		`"exercises":[`,
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("expected %q in the response, got %s", want, raw)
		}
	}
	if strings.Contains(raw, trainer.ID) {
		t.Fatalf("the response must never expose the trainer id, got %s", raw)
	}

	// An unrelated authenticated user without an assignment is not found and
	// learns nothing.
	strangerCookie := e2eRegisterUser(t, router, e2eUniqueEmail())
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, strangerCookie, http.MethodGet, "/api/v1/me/program", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for a user without an assignment, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"PROGRAM_NOT_FOUND"`) {
		t.Fatalf("expected PROGRAM_NOT_FOUND, got %s", raw)
	}
	if strings.Contains(raw, "Strength Builder") {
		t.Fatalf("a user without an assignment must never see program data, got %s", raw)
	}
}
