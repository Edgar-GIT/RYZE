package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ryze/backend/models"
)

// ErrAchievementSessionNotFound indicates the workout session does not exist
// or belongs to someone else (deliberately indistinguishable).
var ErrAchievementSessionNotFound = errors.New("workout session not found")

// CompletedSetRow is one completed set of one of the member's completed
// workout sessions, as the achievements need it.
type CompletedSetRow struct {
	SessionID    string
	CompletedAt  time.Time
	ExerciseID   string
	ExerciseName string
	SetType      string
	WeightKg     *float64
	Reps         *int
}

// AchievementsRepository reads the training history the experience points and
// medals are derived from. Every query is scoped to the authenticated user.
type AchievementsRepository interface {
	// ListCompletedSets returns every completed set of the user's completed
	// sessions, oldest session first.
	ListCompletedSets(ctx context.Context, userID string) ([]CompletedSetRow, error)
	// SessionStatus returns the status of one of the user's sessions.
	SessionStatus(ctx context.Context, userID, sessionID string) (string, error)
}

type achievementsRepository struct {
	db *gorm.DB
}

func NewAchievementsRepository(db *gorm.DB) AchievementsRepository {
	return &achievementsRepository{db: db}
}

func (r *achievementsRepository) ListCompletedSets(ctx context.Context, userID string) ([]CompletedSetRow, error) {
	var rows []CompletedSetRow
	err := r.db.WithContext(ctx).Raw(`
SELECT s.id AS session_id,
       COALESCE(s.completed_at, s.updated_at) AS completed_at,
       e.exercise_id AS exercise_id,
       e.name AS exercise_name,
       ws.set_type AS set_type,
       ws.actual_weight_kg AS weight_kg,
       ws.actual_reps AS reps
FROM workout_sessions s
JOIN workout_session_exercises e ON e.workout_session_id = s.id AND e.deleted_at IS NULL
JOIN workout_session_sets ws ON ws.workout_session_exercise_id = e.id AND ws.deleted_at IS NULL
WHERE s.user_id = ? AND s.status = ? AND s.deleted_at IS NULL AND ws.status = ?
ORDER BY COALESCE(s.completed_at, s.updated_at) ASC, s.id ASC, e.position ASC, ws.set_number ASC`,
		userID, models.WorkoutSessionStatusCompleted, models.WorkoutSetStatusCompleted).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list completed sets: %w", err)
	}
	return rows, nil
}

func (r *achievementsRepository) SessionStatus(ctx context.Context, userID, sessionID string) (string, error) {
	var session models.WorkoutSession
	err := r.db.WithContext(ctx).Select("id", "status").
		Where("id = ? AND user_id = ?", sessionID, userID).
		First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrAchievementSessionNotFound
		}
		return "", fmt.Errorf("failed to load workout session: %w", err)
	}
	return session.Status, nil
}

// XPAdjustment returns the sum of the user's EXP adjustments (0 when none).
func (r *achievementsRepository) XPAdjustment(ctx context.Context, userID string) (int, error) {
	var total int
	if err := r.db.WithContext(ctx).Raw(`SELECT COALESCE(SUM(xp), 0) FROM xp_adjustments WHERE user_id = ?`, userID).Scan(&total).Error; err != nil {
		return 0, fmt.Errorf("failed to load EXP adjustments: %w", err)
	}
	return total, nil
}

// AddXPAdjustment records an EXP adjustment with its reason.
func (r *achievementsRepository) AddXPAdjustment(ctx context.Context, userID string, xp int, reason string) error {
	if err := r.db.WithContext(ctx).Exec(`INSERT INTO xp_adjustments (user_id, xp, reason) VALUES (?, ?, ?)`, userID, xp, reason).Error; err != nil {
		return fmt.Errorf("failed to add EXP adjustment: %w", err)
	}
	return nil
}
