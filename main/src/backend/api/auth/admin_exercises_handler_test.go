package auth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/repositories"
	"ryze/backend/services/exercises"
)

// stubAdminExercisesService is a scripted fake used to exercise the admin
// handler's forwarding and error mapping without a database.
type stubAdminExercisesService struct {
	listResult exercises.AdminExerciseListResult
	exercise   *exercises.AdminExercise
	err        error
	gotFilter  repositories.AdminExerciseFilter
	gotPage    int
	gotLimit   int
	gotID      string
	gotCreate  exercises.AdminCreateExerciseInput
	gotUpdate  exercises.AdminUpdateExerciseInput
}

func (s *stubAdminExercisesService) AdminListExercises(_ context.Context, filter repositories.AdminExerciseFilter, page, limit int) (exercises.AdminExerciseListResult, error) {
	s.gotFilter = filter
	s.gotPage = page
	s.gotLimit = limit
	return s.listResult, s.err
}

func (s *stubAdminExercisesService) AdminGetExercise(_ context.Context, exerciseID string) (*exercises.AdminExercise, error) {
	s.gotID = exerciseID
	return s.exercise, s.err
}

func (s *stubAdminExercisesService) AdminCreateExercise(_ context.Context, input exercises.AdminCreateExerciseInput) (*exercises.AdminExercise, error) {
	s.gotCreate = input
	return s.exercise, s.err
}

func (s *stubAdminExercisesService) AdminUpdateExercise(_ context.Context, exerciseID string, input exercises.AdminUpdateExerciseInput) (*exercises.AdminExercise, error) {
	s.gotID = exerciseID
	s.gotUpdate = input
	return s.exercise, s.err
}

func newAdminExercisesHandlerRouter(svc exercises.AdminService) *gin.Engine {
	handler := auth.NewAdminExercisesHandler(svc)
	router := gin.New()
	v1 := router.Group("/api/v1")
	v1.GET("/admin/exercises", handler.ListExercises)
	v1.GET("/admin/exercises/:exerciseID", handler.GetExercise)
	v1.POST("/admin/exercises", handler.CreateExercise)
	v1.PATCH("/admin/exercises/:exerciseID", handler.UpdateExercise)
	return router
}

func doAdminExercisesRequest(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestAdminExercisesHandlerForwarding(t *testing.T) {
	svc := &stubAdminExercisesService{
		listResult: exercises.AdminExerciseListResult{
			Exercises: []exercises.AdminExercise{{
				ID:       "44444444-4444-4444-4444-444444444444",
				Name:     "Barbell Squat",
				Source:   "gymvisual",
				SourceID: "0043",
				Status:   "published",
			}},
			Total: 1,
			Page:  2,
			Limit: 5,
		},
		exercise: &exercises.AdminExercise{
			ID:     "44444444-4444-4444-4444-444444444444",
			Name:   "Barbell Squat",
			Status: "draft",
		},
	}
	router := newAdminExercisesHandlerRouter(svc)

	// List forwards the administrative filters and pagination, and the response
	// carries provenance and publication status.
	w := doAdminExercisesRequest(router, http.MethodGet, "/api/v1/admin/exercises?q=squat&status=draft&body_part=upper+legs&source=gymvisual&page=2&limit=5", "")
	if w.Code != http.StatusOK {
		t.Fatalf("admin list: expected 200, got %d %s", w.Code, w.Body.String())
	}
	if svc.gotFilter.Query != "squat" || svc.gotFilter.Status != "draft" || svc.gotFilter.BodyPart != "upper legs" || svc.gotFilter.Source != "gymvisual" {
		t.Fatalf("admin list: expected filters forwarded, got %+v", svc.gotFilter)
	}
	if svc.gotPage != 2 || svc.gotLimit != 5 {
		t.Fatalf("admin list: expected page/limit forwarded, got %d/%d", svc.gotPage, svc.gotLimit)
	}
	for _, expected := range []string{`"source":"gymvisual"`, `"source_id":"0043"`, `"status":"published"`} {
		if !strings.Contains(w.Body.String(), expected) {
			t.Fatalf("admin list: expected %s in %s", expected, w.Body.String())
		}
	}

	// Get forwards the path id.
	w = doAdminExercisesRequest(router, http.MethodGet, "/api/v1/admin/exercises/44444444-4444-4444-4444-444444444444", "")
	if w.Code != http.StatusOK {
		t.Fatalf("admin get: expected 200, got %d", w.Code)
	}
	if svc.gotID != "44444444-4444-4444-4444-444444444444" {
		t.Fatalf("admin get: expected id forwarded, got %q", svc.gotID)
	}

	// Create forwards every field and responds 201.
	createBody := `{
		"name": "New Movement",
		"description": "Desc",
		"instruction_steps": ["Step one", "Step two"],
		"body_part": "chest",
		"difficulty": "Beginner"
	}`
	w = doAdminExercisesRequest(router, http.MethodPost, "/api/v1/admin/exercises", createBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("admin create: expected 201, got %d %s", w.Code, w.Body.String())
	}
	if svc.gotCreate.Name != "New Movement" || len(svc.gotCreate.InstructionSteps) != 2 || svc.gotCreate.BodyPart != "chest" {
		t.Fatalf("admin create: expected input forwarded, got %+v", svc.gotCreate)
	}

	// Update forwards present pointer fields and the status transition.
	updateBody := `{"status": "archived", "media_attribution": "© Gym visual"}`
	w = doAdminExercisesRequest(router, http.MethodPatch, "/api/v1/admin/exercises/44444444-4444-4444-4444-444444444444", updateBody)
	if w.Code != http.StatusOK {
		t.Fatalf("admin update: expected 200, got %d %s", w.Code, w.Body.String())
	}
	if svc.gotUpdate.Status == nil || *svc.gotUpdate.Status != "archived" {
		t.Fatalf("admin update: expected status forwarded, got %+v", svc.gotUpdate)
	}
	if svc.gotUpdate.Description != nil {
		t.Fatal("admin update: absent fields must stay nil so they are not written")
	}

	// Malformed bodies are rejected before the service is reached.
	beforeCalls := svc.gotCreate.Name
	w = doAdminExercisesRequest(router, http.MethodPost, "/api/v1/admin/exercises", `{"name":`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("admin create malformed: expected 400, got %d", w.Code)
	}
	if svc.gotCreate.Name != beforeCalls {
		t.Fatal("admin create malformed: the service must not be reached")
	}
}

func TestAdminExercisesHandlerErrorMapping(t *testing.T) {
	svc := &stubAdminExercisesService{}
	router := newAdminExercisesHandlerRouter(svc)

	cases := []struct {
		name string
		err  error
		code int
		body string
	}{
		{"validation", exercises.ErrInvalidInput, http.StatusBadRequest, "VALIDATION_ERROR"},
		{"not found", exercises.ErrExerciseNotFound, http.StatusNotFound, "EXERCISE_NOT_FOUND"},
		{"conflict", exercises.ErrExerciseConflict, http.StatusConflict, "EXERCISE_NAME_TAKEN"},
		{"internal", errors.New("root cause must never leak"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, testCase := range cases {
		svc.err = testCase.err
		w := doAdminExercisesRequest(router, http.MethodGet, "/api/v1/admin/exercises/44444444-4444-4444-4444-444444444444", "")
		if w.Code != testCase.code || !strings.Contains(w.Body.String(), testCase.body) {
			t.Fatalf("%s: expected %d %s, got %d %s", testCase.name, testCase.code, testCase.body, w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "root cause must never leak") {
			t.Fatalf("%s: internal details must never be exposed", testCase.name)
		}
	}
}
