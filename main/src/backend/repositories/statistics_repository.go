package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"ryze/backend/models"
)

// ClientStatistics holds the raw aggregated statistics for a client. All fields
// are derived from existing tables through SQL aggregation; no new tables are
// introduced. The zero value represents a client with no assignment or history.
type ClientStatistics struct {
	// HasActiveAssignment is true when the client has at least one active
	// program assignment (not soft-deleted).
	HasActiveAssignment bool
	// CurrentProgramName is the name of the currently assigned program. Empty
	// when no active assignment exists.
	CurrentProgramName string
	// TotalExecutions is the total number of workout completions recorded for
	// the client, including repeated executions of the same workout.
	TotalExecutions int64
	// UniqueWorkoutsCompleted is the number of distinct program workouts the
	// client has completed at least once.
	UniqueWorkoutsCompleted int64
	// TotalWorkoutsInProgram is the total number of active program workouts
	// across all active weeks of the currently assigned program.
	TotalWorkoutsInProgram int64
	// LastWorkoutDate is the most recent completion timestamp, nil when no
	// completions exist.
	LastWorkoutDate *string
}

// StatisticsRepository defines the read-only data-access operations for
// computing client workout statistics. Every operation is scoped to the
// authenticated user id, which always comes from the authentication context and
// is never accepted from the client.
type StatisticsRepository interface {
	// GetClientStats computes the client statistics by aggregating data from
	// the assignment chain and workout history. It returns zero-value stats
	// when the client has no active assignment.
	GetClientStats(ctx context.Context, userID string) (ClientStatistics, error)
}

type statisticsRepository struct {
	db *gorm.DB
}

func NewStatisticsRepository(db *gorm.DB) StatisticsRepository {
	return &statisticsRepository{db: db}
}

func (r *statisticsRepository) GetClientStats(ctx context.Context, userID string) (ClientStatistics, error) {
	stats := ClientStatistics{}

	// Step 1: Find the most recent active assignment.
	var assignment models.ProgramAssignment
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC, id DESC").
		First(&assignment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No active assignment — return empty stats (not an error).
			return stats, nil
		}
		return ClientStatistics{}, fmt.Errorf("failed to find the active program assignment: %w", err)
	}
	stats.HasActiveAssignment = true

	// Step 2: Find the assigned program.
	var program models.Program
	if err := r.db.WithContext(ctx).
		First(&program, "id = ?", assignment.ProgramID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Assignment points to a missing/soft-deleted program — return
			// assignment present but no program details.
			return stats, nil
		}
		return ClientStatistics{}, fmt.Errorf("failed to find the assigned program: %w", err)
	}
	stats.CurrentProgramName = program.Name

	// Step 3: Count total active workouts in the assigned program.
	// The chain is: programs → program_weeks → program_workouts. Only active
	// (non-soft-deleted) weeks and workouts are counted.
	var totalWorkouts int64
	if err := r.db.WithContext(ctx).
		Model(&models.ProgramWorkout{}).
		Joins("JOIN program_weeks ON program_weeks.id = program_workouts.program_week_id").
		Where("program_weeks.program_id = ? AND program_weeks.deleted_at IS NULL AND program_workouts.deleted_at IS NULL", program.ID).
		Count(&totalWorkouts).Error; err != nil {
		return ClientStatistics{}, fmt.Errorf("failed to count program workouts: %w", err)
	}
	stats.TotalWorkoutsInProgram = totalWorkouts

	// Step 4: Count total executions and unique workouts completed for this
	// client. Both queries are scoped to the user — no program filter is
	// applied because history rows reference workouts from whatever program
	// was active at the time of completion.
	var totalExecutions int64
	if err := r.db.WithContext(ctx).
		Model(&models.WorkoutHistory{}).
		Where("user_id = ?", userID).
		Count(&totalExecutions).Error; err != nil {
		return ClientStatistics{}, fmt.Errorf("failed to count total executions: %w", err)
	}
	stats.TotalExecutions = totalExecutions

	var uniqueCompleted int64
	if err := r.db.WithContext(ctx).
		Model(&models.WorkoutHistory{}).
		Select("COUNT(DISTINCT program_workout_id)").
		Where("user_id = ?", userID).
		Scan(&uniqueCompleted).Error; err != nil {
		return ClientStatistics{}, fmt.Errorf("failed to count unique workouts completed: %w", err)
	}
	stats.UniqueWorkoutsCompleted = uniqueCompleted

	// Step 5: Find the most recent completion date.
	var lastDate *string
	if err := r.db.WithContext(ctx).
		Model(&models.WorkoutHistory{}).
		Select("MAX(completed_at)").
		Where("user_id = ?", userID).
		Scan(&lastDate).Error; err != nil {
		return ClientStatistics{}, fmt.Errorf("failed to find last workout date: %w", err)
	}
	stats.LastWorkoutDate = lastDate

	return stats, nil
}
