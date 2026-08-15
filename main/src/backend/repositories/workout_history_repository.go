package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"ryze/backend/models"
)

// WorkoutHistoryRepository defines the data-access operations for the client
// workout history. Every operation is scoped to the authenticated user id, which
// always comes from the authentication context and is never accepted from the
// client. Write access is restricted to workouts of the user's currently
// assigned program: the assignment tree is the grant, so no trainer-client
// relationship is checked and no program status filter is applied (publishing
// carries no access semantics).
type WorkoutHistoryRepository interface {
	// Create records that the user completed one workout of their currently
	// assigned program. It verifies the full chain
	// user → active assignment → active program → active week → active workout
	// before inserting; any break in the chain maps to ErrWorkoutNotFound.
	Create(ctx context.Context, userID, programWorkoutID string, entry *models.WorkoutHistory) error
	// ListByUser returns one page of the user's own history entries ordered by
	// completion time (newest first), plus the total number of entries. The
	// caller guarantees page >= 1 and limit >= 1.
	ListByUser(ctx context.Context, userID string, page, limit int) ([]models.WorkoutHistory, int64, error)
}

type workoutHistoryRepository struct {
	db *gorm.DB
}

func NewWorkoutHistoryRepository(db *gorm.DB) WorkoutHistoryRepository {
	return &workoutHistoryRepository{db: db}
}

func (r *workoutHistoryRepository) Create(ctx context.Context, userID, programWorkoutID string, entry *models.WorkoutHistory) error {
	// The user's most recent active assignment is the grant: the workout must
	// belong to the program it points to. Soft-deleted assignments are excluded
	// through GORM's default scope.
	var assignment models.ProgramAssignment
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC, id DESC").
		First(&assignment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrWorkoutNotFound
		}
		return fmt.Errorf("failed to find the active program assignment: %w", err)
	}

	// The assigned program must be active. A soft-deleted program keeps its
	// historical assignment rows, but it is no longer executable.
	var program models.Program
	if err := r.db.WithContext(ctx).
		First(&program, "id = ?", assignment.ProgramID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrWorkoutNotFound
		}
		return fmt.Errorf("failed to find the assigned program: %w", err)
	}

	// The workout must belong to the assigned program and be active. The week is
	// joined explicitly and its own soft-delete scope is applied manually: GORM
	// only applies the soft-delete scope to the primary model of a join query.
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ProgramWorkout{}).
		Joins("JOIN program_weeks ON program_weeks.id = program_workouts.program_week_id").
		Where("program_weeks.program_id = ? AND program_weeks.deleted_at IS NULL AND program_workouts.id = ?", program.ID, programWorkoutID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to verify the workout belongs to the assigned program: %w", err)
	}
	if count == 0 {
		return ErrWorkoutNotFound
	}

	entry.UserID = userID
	entry.ProgramWorkoutID = programWorkoutID
	if err := r.db.WithContext(ctx).Create(entry).Error; err != nil {
		return fmt.Errorf("failed to create workout history entry: %w", err)
	}
	return nil
}

func (r *workoutHistoryRepository) ListByUser(ctx context.Context, userID string, page, limit int) ([]models.WorkoutHistory, int64, error) {
	var entries []models.WorkoutHistory
	var total int64

	if err := r.db.WithContext(ctx).
		Model(&models.WorkoutHistory{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count workout history entries: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("completed_at DESC, id DESC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&entries).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list workout history entries: %w", err)
	}
	return entries, total, nil
}
