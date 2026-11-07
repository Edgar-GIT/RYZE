package repositories

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ryze/backend/models"
)

type TrainerClientSessionSummary struct {
	ID                 string
	ProgramID          string
	ProgramName        string
	WeekNumber         int
	WorkoutPosition    int
	Status             string
	StartedAt          time.Time
	LastResumedAt      *time.Time
	PausedAt           *time.Time
	CompletedAt        *time.Time
	AbandonedAt        *time.Time
	AccumulatedSeconds int
	TotalSets          int64
	CompletedSets      int64
	SkippedSets        int64
}

type TrainerClientLegacySummary struct {
	Count           int64
	LastCompletedAt *time.Time
}

type TrainerClientExerciseSummary struct {
	ExerciseID      string
	Name            string
	SessionCount    int64
	LastPerformedAt time.Time
}

type TrainerClientInsightRepository interface {
	ListActiveAssignmentsByUser(ctx context.Context, userID string) ([]models.ProgramAssignment, error)
	ListActiveEntitlementsByUser(ctx context.Context, userID string) ([]models.Entitlement, error)
	ListActiveFreeActivationsByUser(ctx context.Context, userID string) ([]models.FreeProgramActivation, error)
	ListSessions(ctx context.Context, userID string, limit, offset int) ([]TrainerClientSessionSummary, error)
	CountSessions(ctx context.Context, userID string) (int64, error)
	FindActiveSession(ctx context.Context, userID string) (*TrainerClientSessionSummary, error)
	LegacySummary(ctx context.Context, userID string) (TrainerClientLegacySummary, error)
	ListExerciseSummaries(ctx context.Context, userID string) ([]TrainerClientExerciseSummary, error)
	ListExerciseHistory(ctx context.Context, userID, exerciseID string, limit int) ([]models.WorkoutSession, error)
}

type trainerClientInsightRepository struct {
	db *gorm.DB
}

func NewTrainerClientInsightRepository(db *gorm.DB) TrainerClientInsightRepository {
	return &trainerClientInsightRepository{db: db}
}

func (r *trainerClientInsightRepository) ListActiveAssignmentsByUser(ctx context.Context, userID string) ([]models.ProgramAssignment, error) {
	var assignments []models.ProgramAssignment
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Preload("Program", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Order("created_at DESC, id DESC").
		Find(&assignments).Error; err != nil {
		return nil, fmt.Errorf("failed to list client assignments: %w", err)
	}
	return assignments, nil
}

func (r *trainerClientInsightRepository) ListActiveEntitlementsByUser(ctx context.Context, userID string) ([]models.Entitlement, error) {
	var entitlements []models.Entitlement
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND (expires_at IS NULL OR expires_at > UTC_TIMESTAMP(6))", userID).
		Preload("Program", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Order("created_at ASC, id ASC").
		Find(&entitlements).Error; err != nil {
		return nil, fmt.Errorf("failed to list client entitlements: %w", err)
	}
	return entitlements, nil
}

func (r *trainerClientInsightRepository) ListActiveFreeActivationsByUser(ctx context.Context, userID string) ([]models.FreeProgramActivation, error) {
	var activations []models.FreeProgramActivation
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND (expires_at IS NULL OR expires_at > UTC_TIMESTAMP(6))", userID).
		Preload("Program", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Order("activated_at DESC, id DESC").
		Find(&activations).Error; err != nil {
		return nil, fmt.Errorf("failed to list client free activations: %w", err)
	}
	return activations, nil
}

func (r *trainerClientInsightRepository) ListSessions(ctx context.Context, userID string, limit, offset int) ([]TrainerClientSessionSummary, error) {
	var summaries []TrainerClientSessionSummary
	if err := r.sessionSummaryQuery(ctx, userID).
		Where("ws.status IN ?", []string{
			models.WorkoutSessionStatusCompleted,
			models.WorkoutSessionStatusAbandoned,
		}).
		Order("ws.started_at DESC, ws.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&summaries).Error; err != nil {
		return nil, fmt.Errorf("failed to list client workout sessions: %w", err)
	}
	return summaries, nil
}

func (r *trainerClientInsightRepository) CountSessions(ctx context.Context, userID string) (int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&models.WorkoutSession{}).
		Where("user_id = ? AND status IN ?", userID, []string{
			models.WorkoutSessionStatusCompleted,
			models.WorkoutSessionStatusAbandoned,
		}).
		Count(&total).Error; err != nil {
		return 0, fmt.Errorf("failed to count client workout sessions: %w", err)
	}
	return total, nil
}

func (r *trainerClientInsightRepository) FindActiveSession(ctx context.Context, userID string) (*TrainerClientSessionSummary, error) {
	var summary TrainerClientSessionSummary
	if err := r.sessionSummaryQuery(ctx, userID).
		Where("ws.status IN ?", []string{
			models.WorkoutSessionStatusInProgress,
			models.WorkoutSessionStatusPaused,
		}).
		Order("ws.started_at DESC, ws.id DESC").
		Limit(1).
		Scan(&summary).Error; err != nil {
		return nil, fmt.Errorf("failed to find client active workout session: %w", err)
	}
	if summary.ID == "" {
		return nil, nil
	}
	return &summary, nil
}

func (r *trainerClientInsightRepository) LegacySummary(ctx context.Context, userID string) (TrainerClientLegacySummary, error) {
	var summary TrainerClientLegacySummary
	if err := r.db.WithContext(ctx).
		Model(&models.WorkoutHistory{}).
		Select("COUNT(*) AS count, MAX(completed_at) AS last_completed_at").
		Where("user_id = ?", userID).
		Scan(&summary).Error; err != nil {
		return TrainerClientLegacySummary{}, fmt.Errorf("failed to summarise client legacy history: %w", err)
	}
	return summary, nil
}

func (r *trainerClientInsightRepository) ListExerciseSummaries(ctx context.Context, userID string) ([]TrainerClientExerciseSummary, error) {
	var summaries []TrainerClientExerciseSummary
	if err := r.db.WithContext(ctx).
		Table("workout_session_exercises AS wse").
		Select(`
			wse.exercise_id AS exercise_id,
			MAX(wse.name) AS name,
			COUNT(DISTINCT ws.id) AS session_count,
			MAX(COALESCE(ws.completed_at, ws.started_at)) AS last_performed_at
		`).
		Joins("JOIN workout_sessions AS ws ON ws.id = wse.workout_session_id").
		Where("ws.user_id = ? AND ws.status = ? AND ws.deleted_at IS NULL AND wse.deleted_at IS NULL",
			userID, models.WorkoutSessionStatusCompleted).
		Group("wse.exercise_id").
		Order("last_performed_at DESC, name ASC").
		Scan(&summaries).Error; err != nil {
		return nil, fmt.Errorf("failed to list client exercise summaries: %w", err)
	}
	return summaries, nil
}

func (r *trainerClientInsightRepository) ListExerciseHistory(ctx context.Context, userID, exerciseID string, limit int) ([]models.WorkoutSession, error) {
	exerciseSessions := r.db.Model(&models.WorkoutSessionExercise{}).
		Select("workout_session_id").
		Where("exercise_id = ?", exerciseID)

	var sessions []models.WorkoutSession
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, models.WorkoutSessionStatusCompleted).
		Where("id IN (?)", exerciseSessions).
		Preload("Exercises", "exercise_id = ?", exerciseID).
		Preload("Exercises.Sets", func(db *gorm.DB) *gorm.DB {
			return db.Order("set_number ASC, id ASC")
		}).
		Order("started_at DESC, id DESC").
		Limit(limit).
		Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("failed to list client exercise history: %w", err)
	}
	return sessions, nil
}

func (r *trainerClientInsightRepository) sessionSummaryQuery(ctx context.Context, userID string) *gorm.DB {
	aggregates := r.db.Table("workout_session_sets AS wss").
		Select(`
			wse.workout_session_id AS workout_session_id,
			COUNT(wss.id) AS total_sets,
			COALESCE(SUM(CASE WHEN wss.status = ? THEN 1 ELSE 0 END), 0) AS completed_sets,
			COALESCE(SUM(CASE WHEN wss.status = ? THEN 1 ELSE 0 END), 0) AS skipped_sets
		`, models.WorkoutSetStatusCompleted, models.WorkoutSetStatusSkipped).
		Joins("JOIN workout_session_exercises AS wse ON wse.id = wss.workout_session_exercise_id").
		Where("wse.deleted_at IS NULL AND wss.deleted_at IS NULL").
		Group("wse.workout_session_id")

	return r.db.WithContext(ctx).
		Table("workout_sessions AS ws").
		Select(`
			ws.id,
			ws.program_id,
			ws.program_name,
			ws.week_number,
			ws.workout_position,
			ws.status,
			ws.started_at,
			ws.last_resumed_at,
			ws.paused_at,
			ws.completed_at,
			ws.abandoned_at,
			ws.accumulated_seconds,
			COALESCE(agg.total_sets, 0) AS total_sets,
			COALESCE(agg.completed_sets, 0) AS completed_sets,
			COALESCE(agg.skipped_sets, 0) AS skipped_sets
		`).
		Joins("LEFT JOIN (?) AS agg ON agg.workout_session_id = ws.id", aggregates).
		Where("ws.user_id = ? AND ws.deleted_at IS NULL", userID)
}

var _ TrainerClientInsightRepository = (*trainerClientInsightRepository)(nil)
