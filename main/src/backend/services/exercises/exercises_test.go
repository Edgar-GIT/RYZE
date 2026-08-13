package exercises_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/exercises"
)

const (
	exerciseID = "44444444-4444-4444-4444-444444444444"
)

var errRepoFailure = errors.New("repository failure")

// stubExerciseRepo is an in-memory fake of the read-only exercise catalog
// surface. It behaves like the real repository (find/list/search respect
// soft-deletes) and records the arguments passed to every operation so tests
// can prove the service forwards them untouched.
type stubExerciseRepo struct {
	exercise       *models.Exercise
	deleted        bool
	list           func(page, limit int) ([]models.Exercise, int64, error)
	find           func(exerciseID string) (*models.Exercise, error)
	search         func(query string, page, limit int) ([]models.Exercise, int64, error)
	findGotID      string
	searchGotQuery string
	searchGotPage  int
	searchGotLimit int
	listGotPage    int
	listGotLimit   int
}

func (s *stubExerciseRepo) List(_ context.Context, page, limit int) ([]models.Exercise, int64, error) {
	s.listGotPage = page
	s.listGotLimit = limit
	if s.list != nil {
		return s.list(page, limit)
	}
	if s.exercise == nil || s.deleted {
		return nil, 0, nil
	}
	return []models.Exercise{*s.exercise}, 1, nil
}

func (s *stubExerciseRepo) FindByID(_ context.Context, exerciseID string) (*models.Exercise, error) {
	s.findGotID = exerciseID
	if s.find != nil {
		return s.find(exerciseID)
	}
	if s.exercise == nil || s.deleted {
		return nil, repositories.ErrExerciseNotFound
	}
	if s.exercise.ID != exerciseID {
		return nil, repositories.ErrExerciseNotFound
	}
	return s.exercise, nil
}

func (s *stubExerciseRepo) Search(_ context.Context, query string, page, limit int) ([]models.Exercise, int64, error) {
	s.searchGotQuery = query
	s.searchGotPage = page
	s.searchGotLimit = limit
	if s.search != nil {
		return s.search(query, page, limit)
	}
	if s.exercise == nil || s.deleted || !strings.Contains(strings.ToLower(s.exercise.Name), strings.ToLower(query)) {
		return nil, 0, nil
	}
	return []models.Exercise{*s.exercise}, 1, nil
}

func validExercise() *models.Exercise {
	return &models.Exercise{
		ID:            exerciseID,
		Name:          "Barbell Squat",
		Description:   "A lower body compound lift.",
		TargetMuscles: "Quads, Glutes",
		Equipment:     "Barbell",
		Difficulty:    "Intermediate",
		CreatedAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func newService(repo exercises.ExerciseRepository) exercises.Service {
	return exercises.NewService(repo)
}

func TestExerciseServiceList(t *testing.T) {
	repo := &stubExerciseRepo{exercise: validExercise()}
	svc := newService(repo)

	result, err := svc.ListExercises(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("ListExercises: %v", err)
	}
	if result.Total != 1 || len(result.Exercises) != 1 {
		t.Fatalf("ListExercises: expected 1 exercise, got %d/%d", result.Total, len(result.Exercises))
	}
	if repo.listGotPage != 1 || repo.listGotLimit != 20 {
		t.Fatalf("ListExercises: expected page 1 limit 20 forwarded, got %d/%d", repo.listGotPage, repo.listGotLimit)
	}

	// Pagination defaults are validated and the limit is clamped.
	if _, err := svc.ListExercises(context.Background(), 0, 20); !errors.Is(err, exercises.ErrInvalidInput) {
		t.Fatalf("ListExercises page 0: expected ErrInvalidInput, got %v", err)
	}
	if _, err := svc.ListExercises(context.Background(), 1, 0); !errors.Is(err, exercises.ErrInvalidInput) {
		t.Fatalf("ListExercises limit 0: expected ErrInvalidInput, got %v", err)
	}
	if _, err := svc.ListExercises(context.Background(), 1, 99999); err != nil {
		t.Fatalf("ListExercises oversized limit: %v", err)
	}
	if repo.listGotLimit != exercises.MaxPageSize {
		t.Fatalf("ListExercises oversized limit: expected clamp to %d, got %d", exercises.MaxPageSize, repo.listGotLimit)
	}
}

func TestExerciseServiceGet(t *testing.T) {
	repo := &stubExerciseRepo{exercise: validExercise()}
	svc := newService(repo)

	exercise, err := svc.GetExercise(context.Background(), exerciseID)
	if err != nil {
		t.Fatalf("GetExercise: %v", err)
	}
	if exercise.Name != "Barbell Squat" || exercise.ID != exerciseID {
		t.Fatalf("GetExercise: unexpected exercise %+v", exercise)
	}
	if repo.findGotID != exerciseID {
		t.Fatalf("GetExercise: expected id %q forwarded, got %q", exerciseID, repo.findGotID)
	}

	// Missing and soft-deleted exercises map to the domain error.
	repo.deleted = true
	if _, err := svc.GetExercise(context.Background(), exerciseID); !errors.Is(err, exercises.ErrExerciseNotFound) {
		t.Fatalf("GetExercise deleted: expected ErrExerciseNotFound, got %v", err)
	}

	// Malformed identifiers are rejected before any repository call.
	repo.deleted = false
	for _, bad := range []string{"", "not-a-uuid"} {
		if _, err := svc.GetExercise(context.Background(), bad); !errors.Is(err, exercises.ErrInvalidInput) {
			t.Fatalf("GetExercise %q: expected ErrInvalidInput, got %v", bad, err)
		}
	}
}

func TestExerciseServiceSearch(t *testing.T) {
	repo := &stubExerciseRepo{exercise: validExercise()}
	svc := newService(repo)

	result, err := svc.SearchExercises(context.Background(), "squat", 1, 20)
	if err != nil {
		t.Fatalf("SearchExercises: %v", err)
	}
	if result.Total != 1 || len(result.Exercises) != 1 {
		t.Fatalf("SearchExercises: expected 1 match, got %d/%d", result.Total, len(result.Exercises))
	}
	if repo.searchGotQuery != "squat" || repo.searchGotPage != 1 || repo.searchGotLimit != 20 {
		t.Fatalf("SearchExercises: unexpected forwarding %q/%d/%d", repo.searchGotQuery, repo.searchGotPage, repo.searchGotLimit)
	}

	// The query is trimmed before being forwarded.
	if _, err := svc.SearchExercises(context.Background(), "  squat  ", 1, 20); err != nil {
		t.Fatalf("SearchExercises trimmed: %v", err)
	}
	if repo.searchGotQuery != "squat" {
		t.Fatalf("SearchExercises trimmed: expected %q forwarded, got %q", "squat", repo.searchGotQuery)
	}

	// Empty, blank and oversized queries are rejected.
	for _, bad := range []string{"", "   ", strings.Repeat("a", exercises.MaxSearchLength+1)} {
		if _, err := svc.SearchExercises(context.Background(), bad, 1, 20); !errors.Is(err, exercises.ErrInvalidInput) {
			t.Fatalf("SearchExercises %q: expected ErrInvalidInput, got %v", bad, err)
		}
	}

	// A repository failure must never be mapped to a domain error.
	repo.search = func(string, int, int) ([]models.Exercise, int64, error) {
		return nil, 0, errRepoFailure
	}
	_, err = svc.SearchExercises(context.Background(), "squat", 1, 20)
	if errors.Is(err, exercises.ErrInvalidInput) || errors.Is(err, exercises.ErrExerciseNotFound) {
		t.Fatalf("SearchExercises repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("SearchExercises expected an error")
	}
}

func TestExerciseServiceRepositoryFailureIsNotNotfound(t *testing.T) {
	repo := &stubExerciseRepo{find: func(string) (*models.Exercise, error) { return nil, errRepoFailure }}
	svc := newService(repo)

	if _, err := svc.GetExercise(context.Background(), exerciseID); errors.Is(err, exercises.ErrExerciseNotFound) || errors.Is(err, exercises.ErrInvalidInput) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if _, err := svc.GetExercise(context.Background(), exerciseID); err == nil {
		t.Fatal("expected an error")
	}
}

func TestExerciseNeverExposesSecrets(t *testing.T) {
	repo := &stubExerciseRepo{exercise: validExercise()}
	svc := newService(repo)

	exercise, err := svc.GetExercise(context.Background(), exerciseID)
	if err != nil {
		t.Fatalf("GetExercise: %v", err)
	}
	if exercise.Name == "" || exercise.Description == "" {
		t.Fatal("safe exercise fields must be present")
	}

	// The Exercise struct is the only shape this service ever returns. Structural
	// guarantee: it carries metadata only and no sensitive or internal field can
	// reach the caller.
	typ := reflect.TypeOf(*exercise)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i).Name
		for _, sensitive := range []string{"password", "token", "secret", "session", "deleted"} {
			if strings.Contains(strings.ToLower(field), sensitive) {
				t.Fatalf("Exercise must not expose %q", field)
			}
		}
	}
}
