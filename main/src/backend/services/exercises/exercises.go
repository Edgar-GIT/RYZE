package exercises

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/admin_users"
)

var (
	// ErrInvalidInput indicates the exercise query was malformed.
	ErrInvalidInput = errors.New("invalid exercise input")
	// ErrExerciseNotFound indicates the exercise does not exist or is
	// soft-deleted.
	ErrExerciseNotFound = errors.New("exercise not found")
)

const (
	// MaxPageSize caps the number of exercises returned in a single page.
	MaxPageSize = admin_users.MaxPageSize
	// MaxSearchLength caps the length of the exercise search query.
	MaxSearchLength = 100
)

// ExerciseRepository is the read-only data-access surface required by the
// exercises service. Writing the catalog is intentionally not part of it: the
// catalog is platform-owned and is populated outside this service.
type ExerciseRepository interface {
	FindByID(ctx context.Context, exerciseID string) (*models.Exercise, error)
	List(ctx context.Context, page, limit int) ([]models.Exercise, int64, error)
	Search(ctx context.Context, query string, page, limit int) ([]models.Exercise, int64, error)
}

// Exercise is the safe representation of one exercise catalog entry. It
// carries only the public descriptive metadata and never exposes deletion
// markers or any internal data.
type Exercise struct {
	ID            string
	Name          string
	Description   string
	TargetMuscles string
	Equipment     string
	Difficulty    string
	VideoURL      string
	ImageURL      string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ListExercisesResult carries one page of exercises plus the pagination
// metadata needed to render the list.
type ListExercisesResult struct {
	Exercises []Exercise
	Total     int64
	Page      int
	Limit     int
}

// Service exposes the public, read-only exercise catalog. The catalog is
// platform-owned: there is no trainer or admin ownership on this entity, the
// same data is served to every caller, and no write operation is exposed.
type Service interface {
	ListExercises(ctx context.Context, page, limit int) (ListExercisesResult, error)
	GetExercise(ctx context.Context, exerciseID string) (*Exercise, error)
	SearchExercises(ctx context.Context, query string, page, limit int) (ListExercisesResult, error)
}

type service struct {
	exercises ExerciseRepository
}

func NewService(exercises ExerciseRepository) Service {
	return &service{exercises: exercises}
}

// ListExercises returns one page of the catalog ordered alphabetically by name.
func (s *service) ListExercises(ctx context.Context, page, limit int) (ListExercisesResult, error) {
	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return ListExercisesResult{}, err
	}

	exerciseModels, total, err := s.exercises.List(ctx, page, limit)
	if err != nil {
		return ListExercisesResult{}, fmt.Errorf("failed to list exercises: %w", err)
	}

	return ListExercisesResult{
		Exercises: toSafeList(exerciseModels),
		Total:     total,
		Page:      page,
		Limit:     limit,
	}, nil
}

// GetExercise returns one active catalog entry. Soft-deleted exercises are
// indistinguishable from missing ones.
func (s *service) GetExercise(ctx context.Context, exerciseID string) (*Exercise, error) {
	if err := validateExerciseID(exerciseID); err != nil {
		return nil, err
	}

	exerciseModel, err := s.exercises.FindByID(ctx, exerciseID)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrExerciseNotFound):
			return nil, ErrExerciseNotFound
		default:
			return nil, fmt.Errorf("failed to find exercise: %w", err)
		}
	}

	return toSafe(exerciseModel), nil
}

// SearchExercises returns one page of catalog entries whose name contains the
// query, case-insensitively.
func (s *service) SearchExercises(ctx context.Context, query string, page, limit int) (ListExercisesResult, error) {
	if err := validateSearchQuery(query); err != nil {
		return ListExercisesResult{}, err
	}
	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return ListExercisesResult{}, err
	}

	exerciseModels, total, err := s.exercises.Search(ctx, strings.TrimSpace(query), page, limit)
	if err != nil {
		return ListExercisesResult{}, fmt.Errorf("failed to search exercises: %w", err)
	}

	return ListExercisesResult{
		Exercises: toSafeList(exerciseModels),
		Total:     total,
		Page:      page,
		Limit:     limit,
	}, nil
}

func toSafe(model *models.Exercise) *Exercise {
	return &Exercise{
		ID:            model.ID,
		Name:          model.Name,
		Description:   model.Description,
		TargetMuscles: model.TargetMuscles,
		Equipment:     model.Equipment,
		Difficulty:    model.Difficulty,
		VideoURL:      model.VideoURL,
		ImageURL:      model.ImageURL,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}
}

func toSafeList(models []models.Exercise) []Exercise {
	list := make([]Exercise, 0, len(models))
	for i := range models {
		list = append(list, *toSafe(&models[i]))
	}
	return list
}

// normalizePagination validates the pagination parameters and clamps oversized
// limits to MaxPageSize.
func normalizePagination(page, limit int) (int, int, error) {
	if page < 1 {
		return 0, 0, fmt.Errorf("%w: page must be at least 1", ErrInvalidInput)
	}
	if limit < 1 {
		return 0, 0, fmt.Errorf("%w: limit must be at least 1", ErrInvalidInput)
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	return page, limit, nil
}

// validateExerciseID rejects empty and malformed identifiers before any
// database access.
func validateExerciseID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: exercise id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid exercise id", ErrInvalidInput)
	}
	return nil
}

// validateSearchQuery rejects empty, blank and oversized search queries.
func validateSearchQuery(query string) error {
	if strings.TrimSpace(query) == "" {
		return fmt.Errorf("%w: search query is required", ErrInvalidInput)
	}
	if len([]rune(strings.TrimSpace(query))) > MaxSearchLength {
		return fmt.Errorf("%w: search query exceeds the maximum length", ErrInvalidInput)
	}
	return nil
}
