package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"ryze/backend/models"
)

// ErrExerciseNotFound indicates the exercise does not exist or is soft-deleted.
// The catalog is global: there is no ownership scoping on this entity.
var ErrExerciseNotFound = errors.New("exercise not found")

// ExerciseRepository defines the read-only data-access operations for the
// global exercise catalog. The catalog is platform-owned and is only ever read
// through this surface in the current foundation; writing exercises is
// intentionally not part of the repository so that no write path can be
// invented. Soft-deleted exercises are excluded through GORM's default scope.
type ExerciseRepository interface {
	FindByID(ctx context.Context, exerciseID string) (*models.Exercise, error)
	List(ctx context.Context, page, limit int) ([]models.Exercise, int64, error)
	Search(ctx context.Context, query string, page, limit int) ([]models.Exercise, int64, error)
}

type exerciseRepository struct {
	db *gorm.DB
}

func NewExerciseRepository(db *gorm.DB) ExerciseRepository {
	return &exerciseRepository{db: db}
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
	var exercises []models.Exercise
	var total int64

	if err := r.db.WithContext(ctx).
		Model(&models.Exercise{}).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count exercises: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Order("name ASC, id ASC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&exercises).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list exercises: %w", err)
	}
	return exercises, total, nil
}

// Search returns one page of active exercises whose name contains the query
// (case-insensitive), plus the total number of matches. LIKE wildcards supplied
// by the caller are escaped so a search term can never widen into a full scan.
// The caller guarantees page >= 1, limit >= 1 and a non-empty trimmed query.
func (r *exerciseRepository) Search(ctx context.Context, query string, page, limit int) ([]models.Exercise, int64, error) {
	pattern := "%" + escapeLikePattern(query) + "%"

	var exercises []models.Exercise
	var total int64

	if err := r.db.WithContext(ctx).
		Model(&models.Exercise{}).
		Where("name LIKE ?", pattern).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count exercises: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Where("name LIKE ?", pattern).
		Order("name ASC, id ASC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&exercises).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to search exercises: %w", err)
	}
	return exercises, total, nil
}

// escapeLikePattern neutralizes LIKE wildcards so that user input is matched
// literally. GORM already binds the value as a parameter, so this only guards
// against the wildcards themselves, not SQL injection.
func escapeLikePattern(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}
