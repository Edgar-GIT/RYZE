package routes

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"ryze/backend/api/auth"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

// TestWorkoutHistoryE2E exercises the client workout completion and history
// flow through the complete router: a real registration, login and
// authentication flow, a trainer-owned program with one workout assigned to the
// client, the completion of that workout and the history read. It also verifies
// that a foreign or unknown workout is rejected with the single
// indistinguishable 404 and that a user without an assignment learns nothing.
func TestWorkoutHistoryE2E(t *testing.T) {
	router, tx, _ := newE2ERouter(t)
	ctx := context.Background()

	trainerRepo := repositories.NewTrainerRepository(tx)
	clientRepo := repositories.NewTrainerClientRepository(tx)
	programRepo := repositories.NewProgramRepository(tx)
	assignmentRepo := repositories.NewProgramAssignmentRepository(tx)
	weekRepo := repositories.NewProgramWeekRepository(tx)
	workoutRepo := repositories.NewProgramWorkoutRepository(tx)

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
	if err := assignmentRepo.Create(ctx, trainer.ID, clientUserID, program.ID, &models.ProgramAssignment{}); err != nil {
		t.Fatalf("assign program: %v", err)
	}

	// The client completes the assigned workout through the complete router.
	rec, raw := e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodPost,
		"/api/v1/me/workouts/"+workout.ID+"/complete", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("complete: expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, workout.ID) {
		t.Fatalf("expected the completed workout in the response, got %s", raw)
	}
	if strings.Contains(raw, `"user_id"`) {
		t.Fatalf("the response must never expose the owning user id, got %s", raw)
	}

	// The client reads the history through the complete router.
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodGet,
		"/api/v1/me/workouts/history", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("history: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	for _, want := range []string{
		workout.ID,
		`"entries":[`,
		`"pagination":{`,
		`"total":1`,
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("expected %q in the response, got %s", want, raw)
		}
	}
	if strings.Contains(raw, trainer.ID) {
		t.Fatalf("the response must never expose the trainer id, got %s", raw)
	}

	// A foreign workout is indistinguishable from an unknown one: the client
	// completing a random workout id gets the same 404 and learns nothing.
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodPost,
		"/api/v1/me/workouts/00000000-0000-0000-0000-000000000000/complete", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("foreign complete: expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"WORKOUT_NOT_FOUND"`) {
		t.Fatalf("expected WORKOUT_NOT_FOUND, got %s", raw)
	}

	// An unrelated authenticated user without an assignment is rejected and
	// learns nothing.
	strangerCookie := e2eRegisterUser(t, router, e2eUniqueEmail())
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, strangerCookie, http.MethodPost,
		"/api/v1/me/workouts/"+workout.ID+"/complete", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("stranger complete: expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "Strength Builder") {
		t.Fatalf("a user without an assignment must never see program data, got %s", raw)
	}

	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, strangerCookie, http.MethodGet,
		"/api/v1/me/workouts/history", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("stranger history: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"entries":[]`) {
		t.Fatalf("a user without history must see an empty list, got %s", raw)
	}
}
