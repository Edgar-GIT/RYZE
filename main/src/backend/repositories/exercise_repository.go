package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"ryze/backend/models"
)

// ErrExerciseNotFound indicates the exercise does not exist or is soft-deleted.
// The catalog is global: there is no ownership scoping on this entity.
var ErrExerciseNotFound = errors.New("exercise not found")

// ExerciseSearchFilter groups the optional criteria the exercise library can be
// browsed by. Empty fields are ignored. The vocabulary of the fields is
// validated by the service layer, never here.
type ExerciseSearchFilter struct {
	Query      string
	Muscle     string
	Equipment  string
	Difficulty string
	Category   string
}

// ExerciseAlternativeLink is one active alternative link with the linked
// exercise name already resolved from the catalog, so callers never need a
// second lookup.
type ExerciseAlternativeLink struct {
	ID                    string
	ExerciseID            string
	AlternativeExerciseID string
	AlternativeName       string
	CreatedAt             time.Time
}

// ExerciseRepository defines the read-only data-access operations for the
// global exercise catalog. The catalog is platform-owned and is only ever read
// through this surface in the current foundation; writing exercises is
// intentionally not part of the repository so that no write path can be
// invented. Soft-deleted exercises are excluded through GORM's default scope.
type ExerciseRepository interface {
	FindByID(ctx context.Context, exerciseID string) (*models.Exercise, error)
	List(ctx context.Context, page, limit int) ([]models.Exercise, int64, error)
	Search(ctx context.Context, query string, page, limit int) ([]models.Exercise, int64, error)
	Library(ctx context.Context, filter ExerciseSearchFilter, page, limit int) ([]models.Exercise, int64, error)
	ListAlternatives(ctx context.Context, exerciseID string) ([]ExerciseAlternativeLink, error)
}

type exerciseRepository struct {
	db *gorm.DB
}

func NewExerciseRepository(db *gorm.DB) ExerciseRepository {
	return &exerciseRepository{db: db}
}

// Library returns one page of active exercises ordered alphabetically by name,
// restricted by the optional filter criteria, plus the total number of matches.
// The Muscle criterion matches either the primary muscle group or any of the
// comma-separated secondary groups; Equipment and Query are case-insensitive
// substring matches over free-text columns. LIKE wildcards twice escaped so a
// term can never widen into a full scan. The caller guarantees page >= 1,
// limit >= 1 and that the filter uses validated vocabulary.
func (r *exerciseRepository) Library(ctx context.Context, filter ExerciseSearchFilter, page, limit int) ([]models.Exercise, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Exercise{})
	query = applyExerciseFilters(query, filter)

	var exercises []models.Exercise
	var total int64

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count exercises: %w", err)
	}

	if err := query.
		Order("name ASC, id ASC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&exercises).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list exercises: %w", err)
	}
	return exercises, total, nil
}

// ListAlternatives returns the active directed alternative links leaving the
// given exercise, ordered by the linked exercise name. Alternative links that
// point to a soft-deleted exercise are never returned.
func (r *exerciseRepository) ListAlternatives(ctx context.Context, exerciseID string) ([]ExerciseAlternativeLink, error) {
	var links []ExerciseAlternativeLink
	if err := r.db.WithContext(ctx).
		Model(&models.ExerciseAlternative{}).
		Select("exercise_alternatives.id, exercise_alternatives.exercise_id, exercise_alternatives.alternative_exercise_id, exercises.name AS alternative_name, exercise_alternatives.created_at").
		Joins("JOIN exercises ON exercises.id = exercise_alternatives.alternative_exercise_id").
		Where("exercise_alternatives.exercise_id = ? AND exercises.deleted_at IS NULL", exerciseID).
		Order("exercises.name ASC, exercise_alternatives.id ASC").
		Scan(&links).Error; err != nil {
		return nil, fmt.Errorf("failed to list exercise alternatives: %w", err)
	}
	return links, nil
}

// FindByID returns one active exercise. Soft-deleted exercises are never
// returned and an unknown id is indistinguishable from a missing one.
func (r *exerciseRepository) FindByID(ctx context.Context, exerciseID string) (*models.Exercise, error) {
	var exercise models.Exercise
	if err := r.db.WithContext(ctx).
		First(&exercise, "id = ?", exerciseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExerciseNotFound
		}
		return nil, fmt.Errorf("failed to find exercise: %w", err)
	}
	return &exercise, nil
}

// List returns one page of active exercises ordered alphabetically by name,
// plus the total number of active exercises. The caller guarantees
// page >= 1 and limit >= 1.
func (r *exerciseRepository) List(ctx context.Context, page, limit int) ([]models.Exercise, int64, error) {
	return r.Library(ctx, ExerciseSearchFilter{}, page, limit)
}

// Search returns one page of active exercises whose name contains the query
// (case-insensitive), plus the total number of matches. The caller guarantees
// page >= 1, limit >= 1 and a non-empty trimmed query.
func (r *exerciseRepository) Search(ctx context.Context, query string, page, limit int) ([]models.Exercise, int64, error) {
	return r.Library(ctx, ExerciseSearchFilter{Query: query}, page, limit)
}

// applyExerciseFilters narrows a base exercises query with every non-empty
// filter field. LIKE wildcards in every user-supplied term are escaped.
func applyExerciseFilters(query *gorm.DB, filter ExerciseSearchFilter) *gorm.DB {
	if filter.Query != "" {
		pattern := "%" + escapeLikePattern(filter.Query) + "%"
		query = query.Where("name LIKE ?", pattern)
	}
	if filter.Muscle != "" {
		musclePattern := "%" + escapeLikePattern(filter.Muscle) + "%"
		query = query.Where("(primary_muscle_group = ? OR secondary_muscle_groups LIKE ?)", filter.Muscle, musclePattern)
	}
	if filter.Equipment != "" {
		pattern := "%" + escapeLikePattern(filter.Equipment) + "%"
		query = query.Where("equipment LIKE ?", pattern)
	}
	if filter.Difficulty != "" {
		query = query.Where("difficulty = ?", filter.Difficulty)
	}
	if filter.Category != "" {
		query = query.Where("movement_category = ?", filter.Category)
	}
	return query
}

// escapeLikePattern neutralizes LIKE wildcards so that user input is matched
// literally. GORM already binds the value as a parameter, so this only guards
// against the wildcards themselves, not SQL injection.
func escapeLikePattern(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}
