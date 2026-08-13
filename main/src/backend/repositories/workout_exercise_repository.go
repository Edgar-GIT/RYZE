package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ryze/backend/models"
)

var (
	// ErrWorkoutExerciseNotFound indicates the workout exercise does not exist,
	// is soft-deleted, does not belong to the requested workout or the owning
	// chain (workout → week → program) is not owned by the trainer performing
	// the operation.
	ErrWorkoutExerciseNotFound = errors.New("workout exercise not found")
	// ErrWorkoutExerciseReorderConflict indicates the reorder list does not
	// match the set of active workout exercises of the workout.
	ErrWorkoutExerciseReorderConflict = errors.New("workout exercise reorder list mismatch")
)

// WorkoutExerciseRepository defines the data-access operations for the workout
// exercises entity. Every operation receives the trainer id explicitly and
// scopes the query through the owning chain (workout → week → program); the
// repository never obtains the trainer id from an HTTP context, so a
// client-supplied trainer id can never influence a query. Soft-deleted workout
// exercises are excluded from regular queries through GORM's default scope.
type WorkoutExerciseRepository interface {
	AddExercise(ctx context.Context, trainerID, programID, weekID, workoutID string, workoutExercise *models.WorkoutExercise) error
	ListByWorkout(ctx context.Context, trainerID, programID, weekID, workoutID string) ([]models.WorkoutExercise, error)
	FindByIDAndWorkout(ctx context.Context, trainerID, programID, weekID, workoutID, workoutExerciseID string) (*models.WorkoutExercise, error)
	Reorder(ctx context.Context, trainerID, programID, weekID, workoutID string, orderedIDs []string) error
	SoftDelete(ctx context.Context, trainerID, programID, weekID, workoutID, workoutExerciseID string) error
}

type workoutExerciseRepository struct {
	db *gorm.DB
}

func NewWorkoutExerciseRepository(db *gorm.DB) WorkoutExerciseRepository {
	return &workoutExerciseRepository{db: db}
}

// AddExercise appends an exercise to the end of a workout of the trainer's
// program. The workout row is locked inside the transaction so concurrent
// creation is serialized and the next position is always computed against a
// stable state; the unique active_workout_exercise index is the final
// backstop. The referenced exercise must exist and be active: a missing or
// soft-deleted exercise can never be added to a workout.
func (r *workoutExerciseRepository) AddExercise(ctx context.Context, trainerID, programID, weekID, workoutID string, workoutExercise *models.WorkoutExercise) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var workout models.ProgramWorkout
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND id IN (?)", workoutID,
				trainerScopedWorkoutID(tx, trainerID, programID, weekID, workoutID)).
			First(&workout).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrWorkoutNotFound
			}
			return fmt.Errorf("failed to verify workout ownership: %w", err)
		}

		var exercise models.Exercise
		if err := tx.First(&exercise, "id = ?", workoutExercise.ExerciseID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrExerciseNotFound
			}
			return fmt.Errorf("failed to verify exercise: %w", err)
		}

		var maxPosition int
		if err := tx.Model(&models.WorkoutExercise{}).
			Where("program_workout_id = ?", workoutID).
			Select("COALESCE(MAX(position), 0)").
			Scan(&maxPosition).Error; err != nil {
			return fmt.Errorf("failed to resolve next workout exercise position: %w", err)
		}

		workoutExercise.ProgramWorkoutID = workoutID
		workoutExercise.Position = maxPosition + 1
		if err := tx.Create(workoutExercise).Error; err != nil {
			return fmt.Errorf("failed to create workout exercise: %w", err)
		}
		if err := tx.Model(workoutExercise).Association("Exercise").Find(&workoutExercise.Exercise); err != nil {
			return fmt.Errorf("failed to load the assigned exercise: %w", err)
		}
		return nil
	})
}

// ListByWorkout returns every active workout exercise of the workout, ordered by
// position, with its safe exercise data preloaded. The workout must belong to
// the trainer's program; a missing, soft-deleted or foreign workout simply
// yields an empty list.
func (r *workoutExerciseRepository) ListByWorkout(ctx context.Context, trainerID, programID, weekID, workoutID string) ([]models.WorkoutExercise, error) {
	var exercises []models.WorkoutExercise
	if err := r.db.WithContext(ctx).
		Where("program_workout_id = ? AND program_workout_id IN (?)", workoutID,
			trainerScopedWorkoutID(r.db, trainerID, programID, weekID, workoutID)).
		Preload("Exercise").
		Order("position ASC, id ASC").
		Find(&exercises).Error; err != nil {
		return nil, fmt.Errorf("failed to list workout exercises: %w", err)
	}
	return exercises, nil
}

// FindByIDAndWorkout returns one active workout exercise of the workout scoped
// by the trainer, the program, the week, the workout and the workout exercise
// id, with its safe exercise data preloaded. A workout exercise that is
// missing, soft-deleted, under another workout or under a foreign program is
// indistinguishable and never revealed.
func (r *workoutExerciseRepository) FindByIDAndWorkout(ctx context.Context, trainerID, programID, weekID, workoutID, workoutExerciseID string) (*models.WorkoutExercise, error) {
	var workoutExercise models.WorkoutExercise
	if err := r.db.WithContext(ctx).
		Where("id = ? AND program_workout_id = ? AND program_workout_id IN (?)", workoutExerciseID, workoutID,
			trainerScopedWorkoutID(r.db, trainerID, programID, weekID, workoutID)).
		Preload("Exercise").
		First(&workoutExercise).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkoutExerciseNotFound
		}
		return nil, fmt.Errorf("failed to find workout exercise: %w", err)
	}
	return &workoutExercise, nil
}

// Reorder replaces the position of every active workout exercise of the workout
// with the index of its id inside orderedIDs. The list must contain every
// active workout exercise exactly once; anything else is rejected before any
// write. The reassignment runs in two phases inside one transaction: every
// active slot is first released to a distinct value far outside the final
// range, then each workout exercise is assigned its final position, so the
// unique active_workout_exercise index can never reject an intermediate state
// and the position > 0 check is never violated.
func (r *workoutExerciseRepository) Reorder(ctx context.Context, trainerID, programID, weekID, workoutID string, orderedIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var workout models.ProgramWorkout
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND id IN (?)", workoutID,
				trainerScopedWorkoutID(tx, trainerID, programID, weekID, workoutID)).
			First(&workout).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrWorkoutNotFound
			}
			return fmt.Errorf("failed to verify workout ownership: %w", err)
		}

		var exercises []models.WorkoutExercise
		if err := tx.Where("program_workout_id = ?", workoutID).
			Order("position ASC, id ASC").
			Find(&exercises).Error; err != nil {
			return fmt.Errorf("failed to load workout exercises for reorder: %w", err)
		}
		if err := validateWorkoutExerciseOrder(orderedIDs, exercises); err != nil {
			return err
		}

		if err := tx.Model(&models.WorkoutExercise{}).
			Where("program_workout_id = ?", workoutID).
			Update("position", gorm.Expr("position + ?", releaseSlotOffset)).Error; err != nil {
			return fmt.Errorf("failed to release workout exercise slots: %w", err)
		}
		for i, id := range orderedIDs {
			if err := tx.Model(&models.WorkoutExercise{}).
				Where("id = ? AND program_workout_id = ?", id, workoutID).
				Update("position", i+1).Error; err != nil {
				return fmt.Errorf("failed to assign workout exercise slot: %w", err)
			}
		}
		return nil
	})
}

// SoftDelete soft-deletes one of the trainer's own workout exercises. Only the
// workout exercise row is touched; the workout, the week and the program are
// never touched.
func (r *workoutExerciseRepository) SoftDelete(ctx context.Context, trainerID, programID, weekID, workoutID, workoutExerciseID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND program_workout_id = ? AND program_workout_id IN (?)", workoutExerciseID, workoutID,
			trainerScopedWorkoutID(r.db, trainerID, programID, weekID, workoutID)).
		Delete(&models.WorkoutExercise{})
	if result.Error != nil {
		return fmt.Errorf("failed to soft delete workout exercise: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrWorkoutExerciseNotFound
	}
	return nil
}

// validateWorkoutExerciseOrder rejects reorder lists that are not an exact
// permutation of the active workout exercises.
func validateWorkoutExerciseOrder(orderedIDs []string, exercises []models.WorkoutExercise) error {
	if len(orderedIDs) != len(exercises) {
		return fmt.Errorf("%w: the workout exercise list must contain every active workout exercise exactly once", ErrWorkoutExerciseReorderConflict)
	}
	seen := make(map[string]struct{}, len(orderedIDs))
	for _, id := range orderedIDs {
		if _, exists := seen[id]; exists {
			return fmt.Errorf("%w: duplicate workout exercise id %q", ErrWorkoutExerciseReorderConflict, id)
		}
		seen[id] = struct{}{}
	}
	for _, exercise := range exercises {
		if _, exists := seen[exercise.ID]; !exists {
			return fmt.Errorf("%w: active workout exercise %q is missing from the order list", ErrWorkoutExerciseReorderConflict, exercise.ID)
		}
	}
	return nil
}

// trainerScopedWorkoutID builds a subquery selecting the id of the active
// workout of the trainer's week and program. It keeps every workout exercise
// query scoped by the trainer without trusting any client-supplied identity.
func trainerScopedWorkoutID(db *gorm.DB, trainerID, programID, weekID, workoutID string) *gorm.DB {
	return db.Model(&models.ProgramWorkout{}).Select("id").
		Where("id = ? AND program_week_id IN (?)", workoutID,
			trainerScopedWeekID(db, trainerID, programID, weekID))
}
