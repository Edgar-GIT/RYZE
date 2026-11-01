package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ryze/backend/models"
)

var (
	// ErrWorkoutSessionNotFound indicates the session does not exist, is
	// soft-deleted or belongs to another user.
	ErrWorkoutSessionNotFound = errors.New("workout session not found")
	// ErrWorkoutSessionAlreadyActive indicates another session is already in
	// progress or paused for the user.
	ErrWorkoutSessionAlreadyActive = errors.New("workout session already active")
	// ErrWorkoutSessionNotActive indicates a transition was attempted on a
	// completed or abandoned session.
	ErrWorkoutSessionNotActive = errors.New("workout session not active")
	// ErrWorkoutSetNotFound indicates the set does not belong to the session.
	ErrWorkoutSetNotFound = errors.New("workout session set not found")
)

// WorkoutSetUpdate carries the final persisted values of one set update.
type WorkoutSetUpdate struct {
	ActualReps     *int
	ActualWeightKg *float64
	ActualRIR      *int
	ActualRPE      *float64
	Status         string
	CompletedAt    *time.Time
}

// WorkoutSessionRepository defines the data-access operations for interactive
// workout sessions. Every operation receives the user id from the caller and
// never resolves it from an HTTP context.
type WorkoutSessionRepository interface {
	Create(ctx context.Context, session *models.WorkoutSession) error
	FindActiveByUser(ctx context.Context, userID string) (*models.WorkoutSession, error)
	FindByIDAndUser(ctx context.Context, userID, sessionID string) (*models.WorkoutSession, error)
	ListByUser(ctx context.Context, userID string, limit int) ([]models.WorkoutSession, error)
	UpdateSet(ctx context.Context, userID, sessionID, setID string, update WorkoutSetUpdate) (*models.WorkoutSession, error)
	Pause(ctx context.Context, userID, sessionID string, now time.Time) (*models.WorkoutSession, error)
	Resume(ctx context.Context, userID, sessionID string, now time.Time) (*models.WorkoutSession, error)
	Complete(ctx context.Context, userID, sessionID string, now time.Time) (*models.WorkoutSession, error)
	Abandon(ctx context.Context, userID, sessionID string, now time.Time) (*models.WorkoutSession, error)
}

type workoutSessionRepository struct {
	db *gorm.DB
}

func NewWorkoutSessionRepository(db *gorm.DB) WorkoutSessionRepository {
	return &workoutSessionRepository{db: db}
}

func sessionPreloads(db *gorm.DB) *gorm.DB {
	return db.
		Preload("Exercises", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC, id ASC") }).
		Preload("Exercises.Sets", func(db *gorm.DB) *gorm.DB { return db.Order("set_number ASC, id ASC") })
}

// Create persists a session with its prescription snapshot. The active-session
// unique index is the final backstop against concurrent starts.
func (r *workoutSessionRepository) Create(ctx context.Context, session *models.WorkoutSession) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.WorkoutSession{}).
			Where("user_id = ? AND status IN ?", session.UserID, []string{
				models.WorkoutSessionStatusInProgress,
				models.WorkoutSessionStatusPaused,
			}).
			Count(&count).Error; err != nil {
			return fmt.Errorf("failed to check active workout session: %w", err)
		}
		if count > 0 {
			return ErrWorkoutSessionAlreadyActive
		}
		if err := tx.Create(session).Error; err != nil {
			if isDuplicateEntry(err) {
				return ErrWorkoutSessionAlreadyActive
			}
			return fmt.Errorf("failed to create workout session: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	loaded, err := r.FindByIDAndUser(ctx, session.UserID, session.ID)
	if err != nil {
		return err
	}
	*session = *loaded
	return nil
}

// FindActiveByUser returns the user's in-progress or paused session, or
// ErrWorkoutSessionNotFound when none exists.
func (r *workoutSessionRepository) FindActiveByUser(ctx context.Context, userID string) (*models.WorkoutSession, error) {
	var session models.WorkoutSession
	if err := sessionPreloads(r.db.WithContext(ctx)).
		Where("user_id = ? AND status IN ?", userID, []string{
			models.WorkoutSessionStatusInProgress,
			models.WorkoutSessionStatusPaused,
		}).
		First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkoutSessionNotFound
		}
		return nil, fmt.Errorf("failed to find active workout session: %w", err)
	}
	return &session, nil
}

// FindByIDAndUser returns one session scoped by the authenticated user.
func (r *workoutSessionRepository) FindByIDAndUser(ctx context.Context, userID, sessionID string) (*models.WorkoutSession, error) {
	var session models.WorkoutSession
	if err := sessionPreloads(r.db.WithContext(ctx)).
		Where("id = ? AND user_id = ?", sessionID, userID).
		First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkoutSessionNotFound
		}
		return nil, fmt.Errorf("failed to find workout session: %w", err)
	}
	return &session, nil
}

// ListByUser returns the user's finished sessions, newest first.
func (r *workoutSessionRepository) ListByUser(ctx context.Context, userID string, limit int) ([]models.WorkoutSession, error) {
	var sessions []models.WorkoutSession
	if err := sessionPreloads(r.db.WithContext(ctx)).
		Where("user_id = ? AND status IN ?", userID, []string{
			models.WorkoutSessionStatusCompleted,
			models.WorkoutSessionStatusAbandoned,
		}).
		Order("started_at DESC, id DESC").
		Limit(limit).
		Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("failed to list workout sessions: %w", err)
	}
	return sessions, nil
}

// UpdateSet updates actual performance on one set and returns the refreshed
// session. The session row is locked so a concurrent completion cannot race
// the update.
func (r *workoutSessionRepository) UpdateSet(ctx context.Context, userID, sessionID, setID string, update WorkoutSetUpdate) (*models.WorkoutSession, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var session models.WorkoutSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", sessionID, userID).
			First(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrWorkoutSessionNotFound
			}
			return fmt.Errorf("failed to lock workout session: %w", err)
		}
		if session.Status != models.WorkoutSessionStatusInProgress &&
			session.Status != models.WorkoutSessionStatusPaused {
			return ErrWorkoutSessionNotActive
		}

		var set models.WorkoutSessionSet
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Joins("JOIN workout_session_exercises ON workout_session_exercises.id = workout_session_sets.workout_session_exercise_id").
			Where("workout_session_sets.id = ? AND workout_session_exercises.workout_session_id = ?", setID, sessionID).
			First(&set).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrWorkoutSetNotFound
			}
			return fmt.Errorf("failed to lock workout session set: %w", err)
		}

		values := map[string]any{
			"actual_reps":      update.ActualReps,
			"actual_weight_kg": update.ActualWeightKg,
			"actual_rir":       update.ActualRIR,
			"actual_rpe":       update.ActualRPE,
			"status":           update.Status,
			"completed_at":     update.CompletedAt,
		}
		if err := tx.Model(&set).Updates(values).Error; err != nil {
			return fmt.Errorf("failed to update workout session set: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.FindByIDAndUser(ctx, userID, sessionID)
}

// Pause freezes the active clock. Pausing an already paused session is
// idempotent.
func (r *workoutSessionRepository) Pause(ctx context.Context, userID, sessionID string, now time.Time) (*models.WorkoutSession, error) {
	err := r.transition(ctx, userID, sessionID, func(tx *gorm.DB, session *models.WorkoutSession) error {
		if session.Status == models.WorkoutSessionStatusPaused {
			return nil
		}
		if session.Status != models.WorkoutSessionStatusInProgress {
			return ErrWorkoutSessionNotActive
		}
		seconds := elapsedSince(session.LastResumedAt, now)
		return tx.Model(session).Updates(map[string]any{
			"accumulated_seconds": gorm.Expr("accumulated_seconds + ?", seconds),
			"status":              models.WorkoutSessionStatusPaused,
			"last_resumed_at":     nil,
			"paused_at":           now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return r.FindByIDAndUser(ctx, userID, sessionID)
}

// Resume restarts the active clock. Resuming an in-progress session is
// idempotent.
func (r *workoutSessionRepository) Resume(ctx context.Context, userID, sessionID string, now time.Time) (*models.WorkoutSession, error) {
	err := r.transition(ctx, userID, sessionID, func(tx *gorm.DB, session *models.WorkoutSession) error {
		if session.Status == models.WorkoutSessionStatusInProgress {
			return nil
		}
		if session.Status != models.WorkoutSessionStatusPaused {
			return ErrWorkoutSessionNotActive
		}
		return tx.Model(session).Updates(map[string]any{
			"status":          models.WorkoutSessionStatusInProgress,
			"last_resumed_at": now,
			"paused_at":       nil,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return r.FindByIDAndUser(ctx, userID, sessionID)
}

// Complete freezes duration and marks the session completed. Completing an
// already completed session is idempotent.
func (r *workoutSessionRepository) Complete(ctx context.Context, userID, sessionID string, now time.Time) (*models.WorkoutSession, error) {
	err := r.transition(ctx, userID, sessionID, func(tx *gorm.DB, session *models.WorkoutSession) error {
		if session.Status == models.WorkoutSessionStatusCompleted {
			return nil
		}
		if session.Status != models.WorkoutSessionStatusInProgress && session.Status != models.WorkoutSessionStatusPaused {
			return ErrWorkoutSessionNotActive
		}
		updates := map[string]any{
			"status":          models.WorkoutSessionStatusCompleted,
			"completed_at":    now,
			"last_resumed_at": nil,
			"paused_at":       nil,
		}
		if session.Status == models.WorkoutSessionStatusInProgress {
			updates["accumulated_seconds"] = gorm.Expr("accumulated_seconds + ?", elapsedSince(session.LastResumedAt, now))
		}
		return tx.Model(session).Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	return r.FindByIDAndUser(ctx, userID, sessionID)
}

// Abandon freezes duration and marks the session abandoned. Abandoning an
// already abandoned session is idempotent.
func (r *workoutSessionRepository) Abandon(ctx context.Context, userID, sessionID string, now time.Time) (*models.WorkoutSession, error) {
	err := r.transition(ctx, userID, sessionID, func(tx *gorm.DB, session *models.WorkoutSession) error {
		if session.Status == models.WorkoutSessionStatusAbandoned {
			return nil
		}
		if session.Status != models.WorkoutSessionStatusInProgress && session.Status != models.WorkoutSessionStatusPaused {
			return ErrWorkoutSessionNotActive
		}
		updates := map[string]any{
			"status":          models.WorkoutSessionStatusAbandoned,
			"abandoned_at":    now,
			"last_resumed_at": nil,
			"paused_at":       nil,
		}
		if session.Status == models.WorkoutSessionStatusInProgress {
			updates["accumulated_seconds"] = gorm.Expr("accumulated_seconds + ?", elapsedSince(session.LastResumedAt, now))
		}
		return tx.Model(session).Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	return r.FindByIDAndUser(ctx, userID, sessionID)
}

func (r *workoutSessionRepository) transition(
	ctx context.Context,
	userID, sessionID string,
	update func(tx *gorm.DB, session *models.WorkoutSession) error,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var session models.WorkoutSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", sessionID, userID).
			First(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrWorkoutSessionNotFound
			}
			return fmt.Errorf("failed to lock workout session: %w", err)
		}
		if err := update(tx, &session); err != nil {
			return err
		}
		return nil
	})
}

func elapsedSince(lastResumedAt *time.Time, now time.Time) int {
	if lastResumedAt == nil {
		return 0
	}
	seconds := int(now.Sub(*lastResumedAt).Seconds())
	if seconds < 0 {
		return 0
	}
	return seconds
}
