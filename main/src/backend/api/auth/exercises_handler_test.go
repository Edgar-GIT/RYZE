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
	exercise   *exercises.Exercise
	listResult exercises.ListExercisesResult
	err        error
	gotID      string
	gotQuery   string
	gotPage    int
	gotLimit   int
}

func (s *stubExercisesService) ListExercises(_ context.Context, page, limit int) (exercises.ListExercisesResult, error) {
	s.gotPage = page
	s.gotLimit = limit
	return s.listResult, s.err
}

func (s *stubExercisesService) GetExercise(_ context.Context, exerciseID string) (*exercises.Exercise, error) {
	s.gotID = exerciseID
	return s.exercise, s.err
}

func (s *stubExercisesService) SearchExercises(_ context.Context, query string, page, limit int) (exercises.ListExercisesResult, error) {
	s.gotQuery = query
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

func doExercisesRequest(router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestExercisesHandlerRealRouter(t *testing.T) {
	router, tx := newExercisesTestRouter(t)

	squat := seedExercise(t, tx, "Barbell Squat")
	deadlift := seedExercise(t, tx, "Deadlift")
	seedExercise(t, tx, "Push-Up")

	// 1. List returns the catalog alphabetically with pagination metadata.
	w := doExercisesRequest(router, http.MethodGet, "/api/v1/exercises")
	if w.Code != http.StatusOK {
		t.Fatalf("list exercises: expected 200, got %d", w.Code)
	}
	var listBody struct {
		Success bool `json:"success"`
		Data    struct {
			Exercises []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
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
	if listBody.Data.Pagination.Total != 3 || listBody.Data.Pagination.TotalPages != 1 {
		t.Fatalf("list exercises: unexpected pagination %+v", listBody.Data.Pagination)
	}
	if len(listBody.Data.Exercises) != 3 {
		t.Fatalf("list exercises: expected 3, got %d", len(listBody.Data.Exercises))
	}
	names := []string{listBody.Data.Exercises[0].Name, listBody.Data.Exercises[1].Name, listBody.Data.Exercises[2].Name}
	if strings.Join(names, ",") != "Barbell Squat,Deadlift,Push-Up" {
		t.Fatalf("list exercises: expected alphabetical order, got %v", names)
	}

	// 2. Get returns one exercise.
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises/"+squat.ID)
	if w.Code != http.StatusOK {
		t.Fatalf("get exercise: expected 200, got %d", w.Code)
	}
	var getBody struct {
		Data struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &getBody); err != nil {
		t.Fatalf("get exercise: unmarshal: %v", err)
	}
	if getBody.Data.ID != squat.ID || getBody.Data.Name != "Barbell Squat" {
		t.Fatalf("get exercise: unexpected body %+v", getBody.Data)
	}

	// 3. Search matches a name substring, case-insensitively.
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises/search?q=SQUAT")
	if w.Code != http.StatusOK {
		t.Fatalf("search exercises: expected 200, got %d", w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("search exercises: unmarshal: %v", err)
	}
	if listBody.Data.Pagination.Total != 1 || len(listBody.Data.Exercises) != 1 || listBody.Data.Exercises[0].ID != squat.ID {
		t.Fatalf("search exercises: expected exactly the squat, got %+v", listBody.Data)
	}

	// 4. An unknown exercise maps to 404.
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises/00000000-0000-0000-0000-000000000000")
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown exercise: expected 404, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "EXERCISE_NOT_FOUND") {
		t.Fatalf("unknown exercise: expected EXERCISE_NOT_FOUND, got %s", w.Body.String())
	}

	// 5. A malformed exercise id maps to 400.
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises/not-a-uuid")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("malformed exercise id: expected 400, got %d", w.Code)
	}

	// 6. Invalid pagination maps to 400.
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

	// 7. Soft-deleted exercises disappear from every public read.
	if err := tx.Delete(&models.Exercise{}, "id = ?", deadlift.ID).Error; err != nil {
		t.Fatalf("soft delete exercise: %v", err)
	}
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises/"+deadlift.ID)
	if w.Code != http.StatusNotFound {
		t.Fatalf("soft-deleted exercise: expected 404, got %d", w.Code)
	}
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises")
	if err := json.Unmarshal(w.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("list after delete: unmarshal: %v", err)
	}
	if listBody.Data.Pagination.Total != 2 {
		t.Fatalf("list after delete: expected 2, got %d", listBody.Data.Pagination.Total)
	}
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
		exercise: &exercises.Exercise{
			ID:        "44444444-4444-4444-4444-444444444444",
			Name:      "Barbell Squat",
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
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

	svc.err = nil
	w = doExercisesRequest(router, http.MethodGet, "/api/v1/exercises/search?q=press&page=3&limit=5")
	if w.Code != http.StatusOK {
		t.Fatalf("search exercise: expected 200, got %d", w.Code)
	}
	if svc.gotQuery != "press" || svc.gotPage != 3 || svc.gotLimit != 5 {
		t.Fatalf("search exercise: expected query/page/limit forwarded, got %q/%d/%d", svc.gotQuery, svc.gotPage, svc.gotLimit)
	}
}
