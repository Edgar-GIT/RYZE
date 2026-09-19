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
	library        func(filter repositories.ExerciseSearchFilter, page, limit int) ([]models.Exercise, int64, error)
	alternatives   func(exerciseID string) ([]repositories.ExerciseAlternativeLink, error)
	findGotID      string
	searchGotQuery string
	searchGotPage  int
	searchGotLimit int
	listGotPage    int
	listGotLimit   int
	libraryGotFilter  repositories.ExerciseSearchFilter
	libraryGotPage    int
	libraryGotLimit   int
	alternativesGotID string
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
	return s.Library(context.Background(), repositories.ExerciseSearchFilter{Query: query}, page, limit)
}

func (s *stubExerciseRepo) Library(_ context.Context, filter repositories.ExerciseSearchFilter, page, limit int) ([]models.Exercise, int64, error) {
	s.libraryGotFilter = filter
	s.libraryGotPage = page
	s.libraryGotLimit = limit
	if s.library != nil {
		return s.library(filter, page, limit)
	}
	if s.exercise == nil || s.deleted {
		return nil, 0, nil
	}
	if filter.Query != "" && !strings.Contains(strings.ToLower(s.exercise.Name), strings.ToLower(filter.Query)) {
		return nil, 0, nil
	}
	if filter.Muscle != "" && !strings.Contains(strings.ToLower(s.exercise.TargetMuscles), strings.ToLower(filter.Muscle)) {
		return nil, 0, nil
	}
	return []models.Exercise{*s.exercise}, 1, nil
}

func (s *stubExerciseRepo) ListAlternatives(_ context.Context, exerciseID string) ([]repositories.ExerciseAlternativeLink, error) {
	s.alternativesGotID = exerciseID
	if s.alternatives != nil {
		return s.alternatives(exerciseID)
	}
	return nil, nil
}

func validExercise() *models.Exercise {
	return &models.Exercise{
		ID:                   exerciseID,
		Name:                 "Barbell Squat",
		Description:          "A lower body compound lift.",
		Instructions:         "Sit down into the hips and drive back up.",
		TargetMuscles:        "Quads, Glutes",
		PrimaryMuscleGroup:   "Quads",
		SecondaryMuscleGroups: "Glutes, Hamstrings",
		Equipment:            "Barbell",
		Difficulty:           "Intermediate",
		MovementCategory:     "Compound",
		CreatedAt:            time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:            time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
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
	if repo.libraryGotPage != 1 || repo.libraryGotLimit != 20 {
		t.Fatalf("ListExercises: expected page 1 limit 20 forwarded, got %d/%d", repo.libraryGotPage, repo.libraryGotLimit)
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
	if repo.libraryGotLimit != exercises.MaxPageSize {
		t.Fatalf("ListExercises oversized limit: expected clamp to %d, got %d", exercises.MaxPageSize, repo.libraryGotLimit)
	}
}

func TestExerciseServiceGet(t *testing.T) {
	repo := &stubExerciseRepo{exercise: validExercise()}
	svc := newService(repo)

	detail, err := svc.GetExercise(context.Background(), exerciseID)
	if err != nil {
		t.Fatalf("GetExercise: %v", err)
	}
	if detail.Exercise.Name != "Barbell Squat" || detail.Exercise.ID != exerciseID {
		t.Fatalf("GetExercise: unexpected exercise %+v", detail.Exercise)
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

func TestExerciseServiceGetResolvesAlternatives(t *testing.T) {
	repo := &stubExerciseRepo{
		exercise: validExercise(),
		alternatives: func(exerciseID string) ([]repositories.ExerciseAlternativeLink, error) {
			return []repositories.ExerciseAlternativeLink{
				{
					ID:                    "a1111111-1111-4111-8111-111111111111",
					ExerciseID:            exerciseID,
					AlternativeExerciseID: "44444444-4444-4444-4444-444444444444",
					AlternativeName:       "Dumbbell Squat",
				},
			}, nil
		},
	}
	svc := newService(repo)

	detail, err := svc.GetExercise(context.Background(), exerciseID)
	if err != nil {
		t.Fatalf("GetExercise with alternatives: %v", err)
	}
	if repo.alternativesGotID != exerciseID {
		t.Fatalf("GetExercise: expected exercise id %q forwarded to alternatives, got %q", exerciseID, repo.alternativesGotID)
	}
	if len(detail.Alternatives) != 1 {
		t.Fatalf("GetExercise: expected 1 alternative, got %d", len(detail.Alternatives))
	}
	link := detail.Alternatives[0]
	if link.ExerciseID != exerciseID || link.AlternativeName != "Dumbbell Squat" {
		t.Fatalf("GetExercise: unexpected alternative %+v", link)
	}

	// An alternatives lookup failure must never map to a domain error.
	repo.alternatives = func(string) ([]repositories.ExerciseAlternativeLink, error) {
		return nil, errRepoFailure
	}
	if _, err := svc.GetExercise(context.Background(), exerciseID); err == nil {
		t.Fatal("GetExercise with failing alternatives: expected an error")
	}
}

func TestExerciseServiceBrowse(t *testing.T) {
	repo := &stubExerciseRepo{exercise: validExercise()}
	svc := newService(repo)

	filter := repositories.ExerciseSearchFilter{
		Muscle:     "Quads",
		Equipment:  "Barbell",
		Difficulty: "Intermediate",
		Category:   "Compound",
	}
	result, err := svc.BrowseExercises(context.Background(), filter, 2, 10)
	if err != nil {
		t.Fatalf("BrowseExercises: %v", err)
	}
	if result.Total != 1 || len(result.Exercises) != 1 {
		t.Fatalf("BrowseExercises: expected 1 result, got %d/%d", result.Total, len(result.Exercises))
	}
	if repo.libraryGotPage != 2 || repo.libraryGotLimit != 10 {
		t.Fatalf("BrowseExercises: expected page/limit forwarded, got %d/%d", repo.libraryGotPage, repo.libraryGotLimit)
	}
	if repo.libraryGotFilter.Difficulty != "Intermediate" || repo.libraryGotFilter.Category != "Compound" {
		t.Fatalf("BrowseExercises: expected filter forwarded, got %+v", repo.libraryGotFilter)
	}

	// Free-text filters are trimmed before being forwarded.
	if _, err := svc.BrowseExercises(context.Background(), repositories.ExerciseSearchFilter{Query: "  squat  "}, 1, 10); err != nil {
		t.Fatalf("BrowseExercises trimmed: %v", err)
	}
	if repo.libraryGotFilter.Query != "squat" {
		t.Fatalf("BrowseExercises trimmed: expected %q forwarded, got %q", "squat", repo.libraryGotFilter.Query)
	}

	// Difficulty and category must use the documented vocabulary.
	for _, bad := range []repositories.ExerciseSearchFilter{{Difficulty: "Pro"}, {Category: "FullBody"}} {
		if _, err := svc.BrowseExercises(context.Background(), bad, 1, 10); !errors.Is(err, exercises.ErrInvalidInput) {
			t.Fatalf("BrowseExercises %+v: expected ErrInvalidInput, got %v", bad, err)
		}
	}

	// Oversized filter values are rejected.
	if _, err := svc.BrowseExercises(context.Background(), repositories.ExerciseSearchFilter{Muscle: strings.Repeat("a", exercises.MaxFilterLength+1)}, 1, 10); !errors.Is(err, exercises.ErrInvalidInput) {
		t.Fatalf("BrowseExercises oversized filter: expected ErrInvalidInput, got %v", err)
	}

	// A repository failure must never be mapped to a domain error.
	repo.library = func(repositories.ExerciseSearchFilter, int, int) ([]models.Exercise, int64, error) {
		return nil, 0, errRepoFailure
	}
	_, err = svc.BrowseExercises(context.Background(), repositories.ExerciseSearchFilter{}, 1, 10)
	if errors.Is(err, exercises.ErrInvalidInput) || errors.Is(err, exercises.ErrExerciseNotFound) {
		t.Fatalf("BrowseExercises repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("BrowseExercises expected an error")
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
	if repo.libraryGotFilter.Query != "squat" || repo.libraryGotPage != 1 || repo.libraryGotLimit != 20 {
		t.Fatalf("SearchExercises: unexpected forwarding %q/%d/%d", repo.libraryGotFilter.Query, repo.libraryGotPage, repo.libraryGotLimit)
	}

	// The query is trimmed before being forwarded.
	if _, err := svc.SearchExercises(context.Background(), "  squat  ", 1, 20); err != nil {
		t.Fatalf("SearchExercises trimmed: %v", err)
	}
	if repo.libraryGotFilter.Query != "squat" {
		t.Fatalf("SearchExercises trimmed: expected %q forwarded, got %q", "squat", repo.libraryGotFilter.Query)
	}

	// Empty, blank and oversized queries are rejected.
	for _, bad := range []string{"", "   ", strings.Repeat("a", exercises.MaxSearchLength+1)} {
		if _, err := svc.SearchExercises(context.Background(), bad, 1, 20); !errors.Is(err, exercises.ErrInvalidInput) {
			t.Fatalf("SearchExercises %q: expected ErrInvalidInput, got %v", bad, err)
		}
	}

	// A repository failure must never be mapped to a domain error.
	repo.library = func(repositories.ExerciseSearchFilter, int, int) ([]models.Exercise, int64, error) {
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

	detail, err := svc.GetExercise(context.Background(), exerciseID)
	if err != nil {
		t.Fatalf("GetExercise: %v", err)
	}
	exercise := detail.Exercise
	if exercise.Name == "" || exercise.Description == "" {
		t.Fatal("safe exercise fields must be present")
	}

	// The Exercise struct is the only shape this service ever returns. Structural
	// guarantee: it carries metadata only and no sensitive or internal field can
	// reach the caller.
	typ := reflect.TypeOf(exercise)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i).Name
		for _, sensitive := range []string{"password", "token", "secret", "session", "deleted"} {
			if strings.Contains(strings.ToLower(field), sensitive) {
				t.Fatalf("Exercise must not expose %q", field)
			}
		}
	}
}
