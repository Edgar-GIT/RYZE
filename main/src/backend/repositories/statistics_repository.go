package repositories

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"

	"ryze/backend/models"
)

// ClientStatistics holds the raw aggregated statistics for a client. All fields
// are derived from existing tables through SQL aggregation.
type ClientStatistics struct {
	// HasActiveAssignment is true when the client has at least one active
	// trainer assignment.
	HasActiveAssignment bool
	// HasActiveProgram is true when the client currently has any access grant
	// (assignment, purchased entitlement or Free activation) for a readable
	// program.
	HasActiveProgram bool
	// CurrentProgramName is the name of the most recent accessible program.
	CurrentProgramName string
	// TotalExecutions counts every completed execution: legacy workout history
	// entries plus completed interactive sessions.
	TotalExecutions int64
	// UniqueWorkoutsCompleted counts distinct exercises' workouts of the
	// current program completed at least once. It is scoped to the current
	// program so the completion percentage can never exceed 100%.
	UniqueWorkoutsCompleted int64
	// TotalWorkoutsInProgram is the total number of active workouts in the
	// current program.
	TotalWorkoutsInProgram int64
	// LastWorkoutDate is the most recent completion timestamp.
	LastWorkoutDate *string
	// CompletedSessions and AbandonedSessions are all-time session counters.
	CompletedSessions int64
	AbandonedSessions int64
	// CompletedSets and SkippedSets are all-time counters over completed
	// sessions only. Abandoned sessions are not counted as performed work.
	CompletedSets int64
	SkippedSets   int64
	// TotalActiveSeconds is the accumulated active (non-paused) time across
	// finished sessions.
	TotalActiveSeconds int64
}

// StatisticsRepository defines the read-only data-access operations for
// computing client workout statistics. Every operation is scoped to the
// authenticated user id.
type StatisticsRepository interface {
	GetClientStats(ctx context.Context, userID string) (ClientStatistics, error)
}

type statisticsRepository struct {
	db *gorm.DB
}

func NewStatisticsRepository(db *gorm.DB) StatisticsRepository {
	return &statisticsRepository{db: db}
}

type programCandidate struct {
	programID string
	at        time.Time
}

func (r *statisticsRepository) GetClientStats(ctx context.Context, userID string) (ClientStatistics, error) {
	stats := ClientStatistics{}

	candidates, hasAssignment, err := r.currentProgramCandidates(ctx, userID)
	if err != nil {
		return ClientStatistics{}, err
	}
	stats.HasActiveAssignment = hasAssignment

	program, ok, err := r.pickCurrentProgram(ctx, candidates)
	if err != nil {
		return ClientStatistics{}, err
	}

	if err := r.countExecutions(ctx, userID, &stats); err != nil {
		return ClientStatistics{}, err
	}
	if err := r.countSessionAggregates(ctx, userID, &stats); err != nil {
		return ClientStatistics{}, err
	}

	if ok {
		stats.HasActiveProgram = true
		stats.CurrentProgramName = program.Name
		if err := r.countCurrentProgramWorkouts(ctx, userID, program.ID, &stats); err != nil {
			return ClientStatistics{}, err
		}
	}
	if err := r.lastWorkoutDate(ctx, userID, &stats); err != nil {
		return ClientStatistics{}, err
	}

	return stats, nil
}

// currentProgramCandidates collects the timestamp of every live grant the user
// holds. The newest readable program wins, matching the client-facing "current
// program" concept without merging assignments, purchases and Free activations
// into a single table.
func (r *statisticsRepository) currentProgramCandidates(ctx context.Context, userID string) ([]programCandidate, bool, error) {
	candidates := make([]programCandidate, 0, 3)

	var assignment models.ProgramAssignment
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC, id DESC").
		First(&assignment).Error
	switch {
	case err == nil:
		candidates = append(candidates, programCandidate{programID: assignment.ProgramID, at: assignment.CreatedAt})
	case errors.Is(err, gorm.ErrRecordNotFound):
	default:
		return nil, false, fmt.Errorf("failed to find active program assignment: %w", err)
	}
	hasAssignment := err == nil

	var entitlement models.Entitlement
	err = r.db.WithContext(ctx).
		Where("user_id = ? AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)", userID).
		Order("created_at DESC, id DESC").
		First(&entitlement).Error
	switch {
	case err == nil:
		candidates = append(candidates, programCandidate{programID: entitlement.ProgramID, at: entitlement.CreatedAt})
	case errors.Is(err, gorm.ErrRecordNotFound):
	default:
		return nil, false, fmt.Errorf("failed to find active entitlement: %w", err)
	}

	var activation models.FreeProgramActivation
	err = r.db.WithContext(ctx).
		Where("user_id = ? AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)", userID).
		Order("activated_at DESC, id DESC").
		First(&activation).Error
	switch {
	case err == nil:
		candidates = append(candidates, programCandidate{programID: activation.ProgramID, at: activation.ActivatedAt})
	case errors.Is(err, gorm.ErrRecordNotFound):
	default:
		return nil, false, fmt.Errorf("failed to find active free activation: %w", err)
	}

	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].at.After(candidates[j].at) })
	return candidates, hasAssignment, nil
}

func (r *statisticsRepository) pickCurrentProgram(ctx context.Context, candidates []programCandidate) (*models.Program, bool, error) {
	for _, candidate := range candidates {
		var program models.Program
		if err := r.db.WithContext(ctx).First(&program, "id = ?", candidate.programID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, false, fmt.Errorf("failed to find current program: %w", err)
		}
		return &program, true, nil
	}
	return nil, false, nil
}

func (r *statisticsRepository) countExecutions(ctx context.Context, userID string, stats *ClientStatistics) error {
	var legacy int64
	if err := r.db.WithContext(ctx).
		Model(&models.WorkoutHistory{}).
		Where("user_id = ?", userID).
		Count(&legacy).Error; err != nil {
		return fmt.Errorf("failed to count workout history: %w", err)
	}

	var completed int64
	if err := r.db.WithContext(ctx).
		Model(&models.WorkoutSession{}).
		Where("user_id = ? AND status = ?", userID, models.WorkoutSessionStatusCompleted).
		Count(&completed).Error; err != nil {
		return fmt.Errorf("failed to count completed sessions: %w", err)
	}

	stats.CompletedSessions = completed
	stats.TotalExecutions = legacy + completed
	return nil
}

func (r *statisticsRepository) countSessionAggregates(ctx context.Context, userID string, stats *ClientStatistics) error {
	var abandoned int64
	if err := r.db.WithContext(ctx).
		Model(&models.WorkoutSession{}).
		Where("user_id = ? AND status = ?", userID, models.WorkoutSessionStatusAbandoned).
		Count(&abandoned).Error; err != nil {
		return fmt.Errorf("failed to count abandoned sessions: %w", err)
	}
	stats.AbandonedSessions = abandoned

	var activeSeconds int64
	if err := r.db.WithContext(ctx).
		Model(&models.WorkoutSession{}).
		Select("COALESCE(SUM(accumulated_seconds), 0)").
		Where("user_id = ? AND status IN ?", userID, []string{
			models.WorkoutSessionStatusCompleted,
			models.WorkoutSessionStatusAbandoned,
		}).
		Scan(&activeSeconds).Error; err != nil {
		return fmt.Errorf("failed to sum active session time: %w", err)
	}
	stats.TotalActiveSeconds = activeSeconds

	var setCounts struct {
		Completed int64
		Skipped   int64
	}
	if err := r.db.WithContext(ctx).
		Model(&models.WorkoutSessionSet{}).
		Select(`
			COALESCE(SUM(CASE WHEN workout_session_sets.status = 'completed' THEN 1 ELSE 0 END), 0) AS completed,
			COALESCE(SUM(CASE WHEN workout_session_sets.status = 'skipped' THEN 1 ELSE 0 END), 0) AS skipped
		`).
		Joins("JOIN workout_session_exercises ON workout_session_exercises.id = workout_session_sets.workout_session_exercise_id").
		Joins("JOIN workout_sessions ON workout_sessions.id = workout_session_exercises.workout_session_id").
		Where("workout_sessions.user_id = ? AND workout_sessions.status = ?", userID, models.WorkoutSessionStatusCompleted).
		Where("workout_sessions.deleted_at IS NULL AND workout_session_exercises.deleted_at IS NULL AND workout_session_sets.deleted_at IS NULL").
		Scan(&setCounts).Error; err != nil {
		return fmt.Errorf("failed to count session sets: %w", err)
	}
	stats.CompletedSets = setCounts.Completed
	stats.SkippedSets = setCounts.Skipped
	return nil
}

func (r *statisticsRepository) countCurrentProgramWorkouts(ctx context.Context, userID, programID string, stats *ClientStatistics) error {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&models.ProgramWorkout{}).
		Joins("JOIN program_weeks ON program_weeks.id = program_workouts.program_week_id").
		Where("program_weeks.program_id = ? AND program_weeks.deleted_at IS NULL AND program_workouts.deleted_at IS NULL", programID).
		Count(&total).Error; err != nil {
		return fmt.Errorf("failed to count program workouts: %w", err)
	}
	stats.TotalWorkoutsInProgram = total

	var unique int64
	if err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(DISTINCT workout_id) FROM (
			SELECT history.program_workout_id AS workout_id
			FROM workout_history history
			JOIN program_workouts workout ON workout.id = history.program_workout_id AND workout.deleted_at IS NULL
			JOIN program_weeks week ON week.id = workout.program_week_id AND week.deleted_at IS NULL
			WHERE history.user_id = ? AND history.deleted_at IS NULL AND week.program_id = ?
			UNION
			SELECT session.program_workout_id AS workout_id
			FROM workout_sessions session
			JOIN program_workouts workout ON workout.id = session.program_workout_id AND workout.deleted_at IS NULL
			JOIN program_weeks week ON week.id = workout.program_week_id AND week.deleted_at IS NULL
			WHERE session.user_id = ? AND session.program_id = ? AND session.status = 'completed' AND session.deleted_at IS NULL
		) completed_workouts
	`, userID, programID, userID, programID).Scan(&unique).Error; err != nil {
		return fmt.Errorf("failed to count unique completed program workouts: %w", err)
	}
	stats.UniqueWorkoutsCompleted = unique
	return nil
}

func (r *statisticsRepository) lastWorkoutDate(ctx context.Context, userID string, stats *ClientStatistics) error {
	var lastDate *string
	if err := r.db.WithContext(ctx).Raw(`
		SELECT MAX(completed_at) FROM (
			SELECT completed_at FROM workout_history WHERE user_id = ? AND deleted_at IS NULL
			UNION ALL
			SELECT completed_at FROM workout_sessions WHERE user_id = ? AND status = 'completed' AND deleted_at IS NULL
		) completions
	`, userID, userID).Scan(&lastDate).Error; err != nil {
		return fmt.Errorf("failed to find last workout date: %w", err)
	}
	stats.LastWorkoutDate = lastDate
	return nil
}
