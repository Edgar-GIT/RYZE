package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ryze/backend/api/auth"
	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/exercises"
)

// newExercisesTestRouter wires the public exercise catalog endpoints backed by
// a database transaction so seeded records are rolled back. The endpoints
// require no authentication and are mounted exactly as routes.go mounts them.
func newExercisesTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
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

	exerciseRepo := repositories.NewExerciseRepository(tx)
	service := exercises.NewService(exerciseRepo)
	handler := auth.NewExercisesHandler(service)

	router := gin.New()
	v1 := router.Group("/api/v1")
	v1.GET("/exercises", handler.ListExercises)
	v1.GET("/exercises/search", handler.SearchExercises)
	v1.GET("/exercises/:exerciseID", handler.GetExercise)

	return router, tx
}

// newExercisesHandlerRouter mounts only the handler with a scripted service, so
// the handler's error mapping and parameter forwarding can be tested without a
// database.
func newExercisesHandlerRouter(svc exercises.Service) *gin.Engine {
	handler := auth.NewExercisesHandler(svc)
	router := gin.New()
	v1 := router.Group("/api/v1")
	v1.GET("/exercises", handler.ListExercises)
	v1.GET("/exercises/search", handler.SearchExercises)
	v1.GET("/exercises/:exerciseID", handler.GetExercise)
	return router
}

// stubExercisesService is a scripted fake used to exercise the handler's error
// mapping and parameter forwarding without touching the database.
type stubExercisesService struct {
	detail     *exercises.ExerciseDetail
	listResult exercises.ListExercisesResult
	err        error
	gotID      string
	gotFilter  repositories.ExerciseSearchFilter
	gotPage    int
	gotLimit   int
}

func (s *stubExercisesService) ListExercises(_ context.Context, page, limit int) (exercises.ListExercisesResult, error) {
	return s.listResult, s.err
}

func (s *stubExercisesService) GetExercise(_ context.Context, exerciseID string) (*exercises.ExerciseDetail, error) {
	s.gotID = exerciseID
	return s.detail, s.err
}

func (s *stubExercisesService) SearchExercises(_ context.Context, query string, page, limit int) (exercises.ListExercisesResult, error) {
	s.gotFilter.Query = query
	s.gotPage = page
	s.gotLimit = limit
	return s.listResult, s.err
}

func (s *stubExercisesService) BrowseExercises(_ context.Context, filter repositories.ExerciseSearchFilter, page, limit int) (exercises.ListExercisesResult, error) {
	s.gotFilter = filter
	s.gotPage = page
	s.gotLimit = limit
	return s.listResult, s.err
}

func seedExercise(t *testing.T, db *gorm.DB, name string) *models.Exercise {
	t.Helper()
	exercise := &models.Exercise{Name: name}
	if err := db.Create(exercise).Error; err != nil {
		t.Fatalf("seed exercise: %v", err)
	}
	return exercise
}

func seedFullExercise(t *testing.T, db *gorm.DB, name, primary, category, difficulty string) *models.Exercise {
	t.Helper()
	exercise := &models.Exercise{
		Name:                  name,
		Description:           "Handler test exercise.",
		Instructions:          "Perform the movement under control.",
		TargetMuscles:         primary,
		PrimaryMuscleGroup:    primary,
		SecondaryMuscleGroups: "Core",
		Equipment:             "Barbell, Bench",
		Difficulty:            difficulty,
		MovementCategory:      category,
	}
	if err := db.Create(exercise).Error; err != nil {
		t.Fatalf("seed full exercise: %v", err)
	}
	return exercise
}

func doExercisesRequest(router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestExercisesHandlerRealRouter(t *testing.T) {
	router, tx := newExercisesTestRouter(t)
	ctx := context.Background()

	// The catalog ships with seed data, so totals are measured relative to the
	// transaction baseline. The test's own rows use a prefix that the catalog
	// never contains.
	exerciseRepo := repositories.NewExerciseRepository(tx)
	_, baselineTotal, err := exerciseRepo.List(ctx, 1, 1)
	if err != nil {
		t.Fatalf("baseline list: %v", err)
	}

	alpha := seedFullExercise(t, tx, "RT-Alpha", "Chest", "Compound", "Beginner")
	beta := seedFullExercise(t, tx, "RT-Beta", "Quads", "Compound", "Intermediate")
	gamma := seedFullExercise(t, tx, "RT-Gamma", "Quads", "Isolation", "Beginner")

	// 1. List returns the catalog with pagination metadata relative to the
	// seeded baseline and no more than the requested page size.
	w := doExercisesRequest(router, http.MethodGet, "/api/v1/exercises")
	if w.Code != http.StatusOK {
		t.Fatalf("list exercises: expected 200, got %d", w.Code)
	}
	var listBody struct {
		Success bool `json:"success"`
		Data    struct {
			Exercises []struct {
				ID              string `json:"id"`
				Name            string `json:"name"`
				Difficulty      string `json:"difficulty"`
				MovementCategory string `json:"movement_category"`
			} `json:"exercises"`
			Pagination struct {
				Page       int   `json:"page"`
				Limit      int   `json:"limit"`
				Total      int64 `json:"total"`
				TotalPages int   `json:"total_pages"`
			} `json:"pagination"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("list exercises: unmarshal: %v", err)
	}
	if !listBody.Success {
		t.Fatal("list exercises: expected success true")
	}
	if listBody.Data.Pagination.Total != baselineTotal+3 || listBody.Data.Pagination.TotalPages != totalPagesOf(baselineTotal+3, 20) {
		t.Fatalf("list exercises: unexpected pagination %+v", listBody.Data.Pagination)
	}
	if len(listBody.Data.Exercises) != 20 {
		t.Fatalf("list exercises: expected 20 per page, got %d", len(listBody.Data.Exercises))
	}

	// 2. Search matches a name substring, case-insensitively, in alphabetical
	// order within the isolated test prefix, and entries carry the structured
	// catalog fields.
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises/search?q=RT-")
	if w.Code != http.StatusOK {
		t.Fatalf("search exercises: expected 200, got %d", w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("search exercises: unmarshal: %v", err)
	}
	if listBody.Data.Pagination.Total != 3 || len(listBody.Data.Exercises) != 3 {
		t.Fatalf("search exercises: expected the three test rows, got %+v", listBody.Data.Pagination)
	}
	names := []string{listBody.Data.Exercises[0].Name, listBody.Data.Exercises[1].Name, listBody.Data.Exercises[2].Name}
	if strings.Join(names, ",") != "RT-Alpha,RT-Beta,RT-Gamma" {
		t.Fatalf("search exercises: expected alphabetical order, got %v", names)
	}
	for _, exercise := range listBody.Data.Exercises {
		if exercise.Difficulty == "" || exercise.MovementCategory == "" {
			t.Fatalf("search exercises: expected structured fields on %+v", exercise)
		}
	}

	// 3. Search narrows by difficulty and movement-category filters.
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises/search?q=RT-&difficulty=Intermediate")
	if err := json.Unmarshal(w.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("search exercises filters: unmarshal: %v", err)
	}
	if listBody.Data.Pagination.Total != 1 || listBody.Data.Exercises[0].ID != beta.ID {
		t.Fatalf("search exercises filters: expected only RT-Beta, got %+v", listBody.Data.Pagination)
	}
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises/search?q=RT-&category=Isolation")
	if err := json.Unmarshal(w.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("search exercises category: unmarshal: %v", err)
	}
	if listBody.Data.Pagination.Total != 1 || listBody.Data.Exercises[0].ID != gamma.ID {
		t.Fatalf("search exercises category: expected only RT-Gamma, got %+v", listBody.Data.Pagination)
	}

	// 4. Get returns one exercise together with its curated alternatives.
	altLink := &models.ExerciseAlternative{ExerciseID: alpha.ID, AlternativeExerciseID: gamma.ID}
	if err := tx.Create(altLink).Error; err != nil {
		t.Fatalf("seed alternative link: %v", err)
	}
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises/"+alpha.ID)
	if w.Code != http.StatusOK {
		t.Fatalf("get exercise: expected 200, got %d", w.Code)
	}
	var getBody struct {
		Data struct {
			Exercise struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Instructions string `json:"instructions"`
			} `json:"exercise"`
			Alternatives []struct {
				AlternativeName string `json:"alternative_name"`
			} `json:"alternatives"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &getBody); err != nil {
		t.Fatalf("get exercise: unmarshal: %v", err)
	}
	if getBody.Data.Exercise.ID != alpha.ID || getBody.Data.Exercise.Name != "RT-Alpha" || getBody.Data.Exercise.Instructions == "" {
		t.Fatalf("get exercise: unexpected body %+v", getBody.Data.Exercise)
	}
	if len(getBody.Data.Alternatives) != 1 || getBody.Data.Alternatives[0].AlternativeName != "RT-Gamma" {
		t.Fatalf("get exercise: expected the seeded alternative, got %+v", getBody.Data.Alternatives)
	}

	// 6. An unknown exercise maps to 404.
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises/00000000-0000-0000-0000-000000000000")
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown exercise: expected 404, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "EXERCISE_NOT_FOUND") {
		t.Fatalf("unknown exercise: expected EXERCISE_NOT_FOUND, got %s", w.Body.String())
	}

	// 7. A malformed exercise id maps to 400.
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises/not-a-uuid")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("malformed exercise id: expected 400, got %d", w.Code)
	}

	// 8. Invalid pagination and search inputs map to 400.
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises?page=0")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("list page 0: expected 400, got %d", w.Code)
	}
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises?limit=abc")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("list limit abc: expected 400, got %d", w.Code)
	}
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises/search?q=")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("search empty query: expected 400, got %d", w.Code)
	}
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises/search?q=RT-&difficulty=Pro")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("search invalid difficulty: expected 400, got %d", w.Code)
	}

	// 9. Soft-deleted exercises disappear from every public read.
	if err := tx.Delete(&models.Exercise{}, "id = ?", beta.ID).Error; err != nil {
		t.Fatalf("soft delete exercise: %v", err)
	}
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises/"+beta.ID)
	if w.Code != http.StatusNotFound {
		t.Fatalf("soft-deleted exercise: expected 404, got %d", w.Code)
	}
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises/search?q=RT-")
	if err := json.Unmarshal(w.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("search after delete: unmarshal: %v", err)
	}
	if listBody.Data.Pagination.Total != 2 {
		t.Fatalf("search after delete: expected 2, got %d", listBody.Data.Pagination.Total)
	}
}

func totalPagesOf(total int64, limit int) int {
	pages := int((total + int64(limit) - 1) / int64(limit))
	if pages < 1 {
		return 1
	}
	return pages
}

func TestExercisesHandlerErrorMapping(t *testing.T) {
	svc := &stubExercisesService{}
	router := newExercisesHandlerRouter(svc)

	// ErrInvalidInput maps to 400 VALIDATION_ERROR.
	svc.err = exercises.ErrInvalidInput
	w := doExercisesRequest(router, http.MethodGet, "/api/v1/exercises?page=0")
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "VALIDATION_ERROR") {
		t.Fatalf("ErrInvalidInput: expected 400 VALIDATION_ERROR, got %d %s", w.Code, w.Body.String())
	}

	// ErrExerciseNotFound maps to 404 EXERCISE_NOT_FOUND.
	svc.err = exercises.ErrExerciseNotFound
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises/00000000-0000-0000-0000-000000000000")
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "EXERCISE_NOT_FOUND") {
		t.Fatalf("ErrExerciseNotFound: expected 404 EXERCISE_NOT_FOUND, got %d %s", w.Code, w.Body.String())
	}

	// Unexpected errors map to 500 and never leak internal details.
	svc.err = errors.New("root cause must never leak")
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("internal error: expected 500, got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "root cause must never leak") {
		t.Fatalf("internal error details must never be exposed: %s", w.Body.String())
	}
}

func TestExercisesHandlerForwarding(t *testing.T) {
	svc := &stubExercisesService{
		detail: &exercises.ExerciseDetail{
			Exercise: exercises.Exercise{
				ID:        "44444444-4444-4444-4444-444444444444",
				Name:      "Barbell Squat",
				CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	router := newExercisesHandlerRouter(svc)

	w := doExercisesRequest(router, http.MethodGet, "/api/v1/exercises/44444444-4444-4444-4444-444444444444")
	if w.Code != http.StatusOK {
		t.Fatalf("get exercise: expected 200, got %d", w.Code)
	}
	if svc.gotID != "44444444-4444-4444-4444-444444444444" {
		t.Fatalf("get exercise: expected id forwarded, got %q", svc.gotID)
	}
	if !strings.Contains(w.Body.String(), `"alternatives"`) {
		t.Fatalf("get exercise: expected an alternatives field in the response")
	}

	svc.err = nil
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises/search?q=press&page=3&limit=5")
	if w.Code != http.StatusOK {
		t.Fatalf("search exercise: expected 200, got %d", w.Code)
	}
	if svc.gotFilter.Query != "press" || svc.gotPage != 3 || svc.gotLimit != 5 {
		t.Fatalf("search exercise: expected query/page/limit forwarded, got %q/%d/%d", svc.gotFilter.Query, svc.gotPage, svc.gotLimit)
	}

	// List forwards the browse filters exactly as received.
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises?muscle=Quads&equipment=Barbell&difficulty=Beginner&category=Compound")
	if w.Code != http.StatusOK {
		t.Fatalf("list exercise filters: expected 200, got %d", w.Code)
	}
	if svc.gotFilter.Muscle != "Quads" || svc.gotFilter.Equipment != "Barbell" ||
		svc.gotFilter.Difficulty != "Beginner" || svc.gotFilter.Category != "Compound" {
		t.Fatalf("list exercise filters: expected filters forwarded, got %+v", svc.gotFilter)
	}
}