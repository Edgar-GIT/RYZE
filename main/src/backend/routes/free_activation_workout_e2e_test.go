package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"ryze/backend/api/auth"
	"ryze/backend/models"
)

type e2eWorkoutSession struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	ProgramID     string `json:"program_id"`
	CompletedSets int    `json:"completed_sets"`
	TotalSets     int    `json:"total_sets"`
	Exercises     []struct {
		Sets []struct {
			ID         string  `json:"id"`
			Status     string  `json:"status"`
			RestEndsAt *string `json:"rest_ends_at"`
		} `json:"sets"`
	} `json:"exercises"`
}

func e2eSessionData(t *testing.T, raw string) e2eWorkoutSession {
	t.Helper()
	var payload struct {
		Data e2eWorkoutSession `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("invalid session JSON: %v (body: %s)", err, raw)
	}
	return payload.Data
}

// TestFreeActivationWorkoutSessionE2E exercises the complete free access and
// interactive workout lifecycle through the real router, authentication and
// database: activation, access, session start, set logging, pause/resume,
// completion, history, statistics and cross-user isolation.
func TestFreeActivationWorkoutSessionE2E(t *testing.T) {
	router, tx, _ := newE2ERouter(t)

	exercise := &models.Exercise{
		Name:          fmt.Sprintf("Free Press %d", time.Now().UnixNano()),
		Description:   "A free plan exercise",
		TargetMuscles: "Chest",
		Equipment:     "Barbell",
		Difficulty:    "Beginner",
	}
	if err := tx.Create(exercise).Error; err != nil {
		t.Fatalf("create exercise: %v", err)
	}

	program := &models.Program{
		Name:        "Free Foundation",
		Description: "A free starter plan",
		Type:        models.ProgramTypeFree,
		ProductType: models.ProgramProductTypeGeneric,
		Status:      models.ProgramStatusPublished,
		Currency:    "EUR",
	}
	if err := tx.Omit("TrainerID").Create(program).Error; err != nil {
		t.Fatalf("create program: %v", err)
	}
	week := &models.ProgramWeek{ProgramID: program.ID, WeekNumber: 1}
	if err := tx.Create(week).Error; err != nil {
		t.Fatalf("create week: %v", err)
	}
	workoutOne := &models.ProgramWorkout{ProgramWeekID: week.ID, Position: 1}
	workoutTwo := &models.ProgramWorkout{ProgramWeekID: week.ID, Position: 2}
	if err := tx.Create(workoutOne).Error; err != nil {
		t.Fatalf("create workout one: %v", err)
	}
	if err := tx.Create(workoutTwo).Error; err != nil {
		t.Fatalf("create workout two: %v", err)
	}
	entry := &models.WorkoutExercise{ProgramWorkoutID: workoutOne.ID, ExerciseID: exercise.ID, Position: 1}
	if err := tx.Create(entry).Error; err != nil {
		t.Fatalf("create workout exercise: %v", err)
	}
	reps := 10
	rest := 60
	set := &models.WorkoutExerciseSet{
		WorkoutExerciseID: entry.ID,
		SetNumber:         1,
		Reps:              &reps,
		RestSeconds:       &rest,
		SetType:           "working",
		Tempo:             "2010",
	}
	if err := tx.Create(set).Error; err != nil {
		t.Fatalf("create prescription set: %v", err)
	}

	clientCookie := e2eRegisterUser(t, router, e2eUniqueEmail())

	// The public catalog is metadata-only.
	rec, raw := e2eRequest(router, "", "", http.MethodGet, "/api/v1/programs/"+program.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("public detail: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, `"weeks"`) || strings.Contains(raw, `"trainer_id"`) || strings.Contains(raw, exercise.Name) {
		t.Fatalf("public detail must not expose structure or trainer identity, got %s", raw)
	}

	// Without activation there is no access.
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodGet, "/api/v1/me/programs/"+program.ID, "")
	if rec.Code != http.StatusNotFound || !strings.Contains(raw, `"code":"PROGRAM_NOT_FOUND"`) {
		t.Fatalf("expected 404 before activation, got %d (body: %s)", rec.Code, raw)
	}

	// Activation is idempotent.
	activationPath := "/api/v1/me/programs/" + program.ID + "/free-activation"
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodPost, activationPath, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("activate: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	var activationPayload struct {
		Data struct {
			ID      string `json:"id"`
			Program struct {
				ID string `json:"id"`
			} `json:"program"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &activationPayload); err != nil {
		t.Fatalf("invalid activation JSON: %v", err)
	}
	firstActivationID := activationPayload.Data.ID
	if firstActivationID == "" || activationPayload.Data.Program.ID != program.ID {
		t.Fatalf("unexpected activation payload: %s", raw)
	}
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodPost, activationPath, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("repeat activation: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, firstActivationID) {
		t.Fatalf("repeat activation must be idempotent, got %s", raw)
	}
	if strings.Contains(raw, `"answers"`) {
		t.Fatalf("activation must never expose questionnaire answers, got %s", raw)
	}

	// Access now exposes the full structure with stable workout and set ids.
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodGet, "/api/v1/me/programs/"+program.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("access: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	for _, want := range []string{workoutOne.ID, exercise.ID, `"product_type":"generic"`, `"rest_seconds":60`} {
		if !strings.Contains(raw, want) {
			t.Fatalf("expected %q in access response, got %s", want, raw)
		}
	}

	// A different Free activation replaces the previous one.
	otherProgram := &models.Program{Name: "Free Other", Type: models.ProgramTypeFree, ProductType: models.ProgramProductTypeGeneric, Status: models.ProgramStatusPublished, Currency: "EUR"}
	if err := tx.Omit("TrainerID").Create(otherProgram).Error; err != nil {
		t.Fatalf("create other free program: %v", err)
	}
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodPost, "/api/v1/me/programs/"+otherProgram.ID+"/free-activation", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("activate replacement: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodGet, "/api/v1/me/free-programs", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list activations: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, program.Name) || !strings.Contains(raw, otherProgram.Name) {
		t.Fatalf("only the replacement activation may remain, got %s", raw)
	}

	// Reactivate the original plan for the session flow.
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodPost, activationPath, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("reactivate original: expected 200, got %d (body: %s)", rec.Code, raw)
	}

	startPath := "/api/v1/me/programs/" + program.ID + "/workouts/" + workoutOne.ID + "/sessions"
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodPost, startPath, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("start session: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	session := e2eSessionData(t, raw)
	if session.ID == "" || session.Status != "in_progress" || session.TotalSets != 1 {
		t.Fatalf("unexpected session payload: %s", raw)
	}
	if len(session.Exercises) != 1 || len(session.Exercises[0].Sets) != 1 {
		t.Fatalf("expected one exercise with one set, got %s", raw)
	}
	setID := session.Exercises[0].Sets[0].ID

	// Starting the same workout is idempotent; a different workout conflicts.
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodPost, startPath, "")
	if rec.Code != http.StatusOK || e2eSessionData(t, raw).ID != session.ID {
		t.Fatalf("repeat start must return the same session, got %d (body: %s)", rec.Code, raw)
	}
	otherStart := "/api/v1/me/programs/" + program.ID + "/workouts/" + workoutTwo.ID + "/sessions"
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodPost, otherStart, "")
	if rec.Code != http.StatusConflict || !strings.Contains(raw, `"code":"WORKOUT_SESSION_ACTIVE"`) {
		t.Fatalf("expected a conflict for another workout, got %d (body: %s)", rec.Code, raw)
	}

	// Log a completed set with actual performance.
	setPath := "/api/v1/me/workout-sessions/" + session.ID + "/sets/" + setID
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodPatch, setPath,
		`{"actual_reps":8,"actual_weight_kg":50,"status":"completed"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("update set: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	updated := e2eSessionData(t, raw)
	if updated.CompletedSets != 1 || updated.Exercises[0].Sets[0].Status != "completed" || updated.Exercises[0].Sets[0].RestEndsAt == nil {
		t.Fatalf("completed set must expose rest_ends_at, got %s", raw)
	}

	// Pause and resume are explicit and own the clock.
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodPost, "/api/v1/me/workout-sessions/"+session.ID+"/pause", "")
	if rec.Code != http.StatusOK || !strings.Contains(raw, `"status":"paused"`) {
		t.Fatalf("pause: expected paused, got %d (body: %s)", rec.Code, raw)
	}
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodPost, "/api/v1/me/workout-sessions/"+session.ID+"/resume", "")
	if rec.Code != http.StatusOK || !strings.Contains(raw, `"status":"in_progress"`) {
		t.Fatalf("resume: expected in_progress, got %d (body: %s)", rec.Code, raw)
	}

	// Another user can never read the session.
	strangerCookie := e2eRegisterUser(t, router, e2eUniqueEmail())
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, strangerCookie, http.MethodGet, "/api/v1/me/workout-sessions/"+session.ID, "")
	if rec.Code != http.StatusNotFound || strings.Contains(raw, program.Name) {
		t.Fatalf("cross-user session read must be 404 without data, got %d (body: %s)", rec.Code, raw)
	}

	// Complete and verify history and derived statistics.
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodPost, "/api/v1/me/workout-sessions/"+session.ID+"/complete", "")
	if rec.Code != http.StatusOK || !strings.Contains(raw, `"status":"completed"`) {
		t.Fatalf("complete: expected completed, got %d (body: %s)", rec.Code, raw)
	}
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodGet, "/api/v1/me/workout-sessions?limit=10", "")
	if rec.Code != http.StatusOK || !strings.Contains(raw, session.ID) {
		t.Fatalf("history: expected the completed session, got %d (body: %s)", rec.Code, raw)
	}
	rec, raw = e2eRequest(router, auth.AccessTokenCookieName, clientCookie, http.MethodGet, "/api/v1/me/statistics", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("statistics: expected 200, got %d (body: %s)", rec.Code, raw)
	}
	for _, want := range []string{
		`"has_active_program":true`,
		`"current_program_name":"Free Foundation"`,
		`"completed_sessions":1`,
		`"completed_sets":1`,
		`"total_workouts_in_program":2`,
		`"unique_workouts_completed":1`,
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("expected %q in statistics, got %s", want, raw)
		}
	}
}
