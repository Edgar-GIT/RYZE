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
	// ErrWorkoutNotFound indicates the program workout does not exist, is
	// soft-deleted, does not belong to the requested week or the owning week or
	// program is not owned by the trainer performing the operation.
	ErrWorkoutNotFound = errors.New("program workout not found")
	// ErrWorkoutReorderConflict indicates the reorder list does not match the
	// set of active workouts of the week.
	ErrWorkoutReorderConflict = errors.New("workout reorder list mismatch")
)

// ProgramWorkoutRepository defines the data-access operations for the program
// workouts entity. Every operation receives the trainer id explicitly and
// scopes the query through the owning week and program; the repository never
// obtains the trainer id from an HTTP context, so a client-supplied trainer id
// can never influence a query. Soft-deleted workouts are excluded from regular
// queries through GORM's default scope.
type ProgramWorkoutRepository interface {
	Create(ctx context.Context, trainerID, programID, weekID string, workout *models.ProgramWorkout) error
	ListByWeek(ctx context.Context, trainerID, programID, weekID string) ([]models.ProgramWorkout, error)
	FindByIDAndWeek(ctx context.Context, trainerID, programID, weekID, workoutID string) (*models.ProgramWorkout, error)
	Reorder(ctx context.Context, trainerID, programID, weekID string, orderedIDs []string) error
	SoftDelete(ctx context.Context, trainerID, programID, weekID, workoutID string) error
}

type programWorkoutRepository struct {
	db *gorm.DB
}

func NewProgramWorkoutRepository(db *gorm.DB) ProgramWorkoutRepository {
	return &programWorkoutRepository{db: db}
}

// Create appends a new workout to the end of a week of the trainer's program.
// The week row is locked inside the transaction so concurrent workout creation
// is serialized and the next position is always computed against a stable
// state; the unique active_workout index is the final backstop.
func (r *programWorkoutRepository) Create(ctx context.Context, trainerID, programID, weekID string, workout *models.ProgramWorkout) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var week models.ProgramWeek
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND program_id = ? AND program_id IN (?)", weekID, programID,
				trainerScopedProgramID(tx, trainerID, programID)).
			First(&week).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrWeekNotFound
			}
			return fmt.Errorf("failed to verify week ownership: %w", err)
		}

		var maxPosition int
		if err := tx.Model(&models.ProgramWorkout{}).
			Where("program_week_id = ?", weekID).
			Select("COALESCE(MAX(position), 0)").
			Scan(&maxPosition).Error; err != nil {
			return fmt.Errorf("failed to resolve next workout position: %w", err)
		}

		workout.ProgramWeekID = weekID
		workout.Position = maxPosition + 1
		if err := tx.Create(workout).Error; err != nil {
			return fmt.Errorf("failed to create program workout: %w", err)
		}
		return nil
	})
}

// ListByWeek returns every active workout of the week, ordered by position. The
// week must belong to the trainer's program; a missing, soft-deleted or foreign
// week simply yields an empty list.
func (r *programWorkoutRepository) ListByWeek(ctx context.Context, trainerID, programID, weekID string) ([]models.ProgramWorkout, error) {
	var workouts []models.ProgramWorkout
	if err := r.db.WithContext(ctx).
		Where("program_week_id = ? AND program_week_id IN (?)", weekID,
			trainerScopedWeekID(r.db, trainerID, programID, weekID)).
		Order("position ASC, id ASC").
		Find(&workouts).Error; err != nil {
		return nil, fmt.Errorf("failed to list program workouts: %w", err)
	}
	return workouts, nil
}

// FindByIDAndWeek returns one active workout of the week scoped by the trainer,
// the program, the week and the workout id. A workout that is missing,
// soft-deleted, under another week or under a foreign program is
// indistinguishable and never revealed.
func (r *programWorkoutRepository) FindByIDAndWeek(ctx context.Context, trainerID, programID, weekID, workoutID string) (*models.ProgramWorkout, error) {
	var workout models.ProgramWorkout
	if err := r.db.WithContext(ctx).
		Where("id = ? AND program_week_id = ? AND program_week_id IN (?)", workoutID, weekID,
			trainerScopedWeekID(r.db, trainerID, programID, weekID)).
		First(&workout).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkoutNotFound
		}
		return nil, fmt.Errorf("failed to find program workout: %w", err)
	}
	return &workout, nil
}

// Reorder replaces the position of every active workout of the week with the
// index of its id inside orderedIDs. The list must contain every active workout
// exactly once; anything else is rejected before any write. The reassignment
// runs in two phases inside one transaction: every active slot is first released
// to a distinct value far outside the final range, then each workout is assigned
// its final position, so the unique active_workout index can never reject an
// intermediate state and the position > 0 check is never violated.
func (r *programWorkoutRepository) Reorder(ctx context.Context, trainerID, programID, weekID string, orderedIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var week models.ProgramWeek
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND program_id = ? AND program_id IN (?)", weekID, programID,
				trainerScopedProgramID(tx, trainerID, programID)).
			First(&week).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrWeekNotFound
			}
			return fmt.Errorf("failed to verify week ownership: %w", err)
		}

		var workouts []models.ProgramWorkout
		if err := tx.Where("program_week_id = ?", weekID).
			Order("position ASC, id ASC").
			Find(&workouts).Error; err != nil {
			return fmt.Errorf("failed to load program workouts for reorder: %w", err)
		}
		if err := validateWorkoutOrder(orderedIDs, workouts); err != nil {
			return err
		}

		if err := tx.Model(&models.ProgramWorkout{}).
			Where("program_week_id = ?", weekID).
			Update("position", gorm.Expr("position + ?", releaseSlotOffset)).Error; err != nil {
			return fmt.Errorf("failed to release program workout slots: %w", err)
		}
		for i, id := range orderedIDs {
			if err := tx.Model(&models.ProgramWorkout{}).
				Where("id = ? AND program_week_id = ?", id, weekID).
				Update("position", i+1).Error; err != nil {
				return fmt.Errorf("failed to assign program workout slot: %w", err)
			}
		}
		return nil
	})
}

// SoftDelete soft-deletes one of the trainer's own program workouts. Only the
// workout row is touched; the week and the program are never touched.
func (r *programWorkoutRepository) SoftDelete(ctx context.Context, trainerID, programID, weekID, workoutID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND program_week_id = ? AND program_week_id IN (?)", workoutID, weekID,
			trainerScopedWeekID(r.db, trainerID, programID, weekID)).
		Delete(&models.ProgramWorkout{})
	if result.Error != nil {
		return fmt.Errorf("failed to soft delete program workout: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrWorkoutNotFound
	}
	return nil
}

// validateWorkoutOrder rejects reorder lists that are not an exact permutation
// of the active workouts.
func validateWorkoutOrder(orderedIDs []string, workouts []models.ProgramWorkout) error {
	if len(orderedIDs) != len(workouts) {
		return fmt.Errorf("%w: the workout list must contain every active workout exactly once", ErrWorkoutReorderConflict)
	}
	seen := make(map[string]struct{}, len(orderedIDs))
	for _, id := range orderedIDs {
		if _, exists := seen[id]; exists {
			return fmt.Errorf("%w: duplicate workout id %q", ErrWorkoutReorderConflict, id)
		}
		seen[id] = struct{}{}
	}
	for _, workout := range workouts {
		if _, exists := seen[workout.ID]; !exists {
			return fmt.Errorf("%w: active workout %q is missing from the order list", ErrWorkoutReorderConflict, workout.ID)
		}
	}
	return nil
}

// releaseSlotOffset (declared in program_week_repository.go) is added to every
// position during the first reorder phase. It moves every slot far outside the
// final 1..n range so the unique active_workout index can never observe a
// collision, while keeping every value positive for the position > 0 constraint.

// trainerScopedWeekID builds a subquery selecting the id of the active week of
// the trainer's program. It keeps every workout query scoped by the trainer
// without trusting any client-supplied identity.
func trainerScopedWeekID(db *gorm.DB, trainerID, programID, weekID string) *gorm.DB {
	return db.Model(&models.ProgramWeek{}).Select("id").
		Where("id = ? AND program_id IN (?)", weekID, trainerScopedProgramID(db, trainerID, programID))
}
