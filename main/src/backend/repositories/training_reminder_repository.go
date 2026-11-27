package repositories

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ReminderCandidate is a member to remind and how long since they trained.
type ReminderCandidate struct {
	UserID       string
	LastTraining *time.Time
}

// TrainingReminderRepository finds the members who stopped training
// (v2.131.0) and keeps their reminder preference.
type TrainingReminderRepository interface {
	// Due lists members with a plan, reminders on and a verified e-mail who
	// have not trained since idleSince and were not reminded since
	// remindedSince.
	Due(ctx context.Context, idleSince, remindedSince time.Time, limit int) ([]ReminderCandidate, error)
	MarkReminded(ctx context.Context, userID string, at time.Time) error
	Enabled(ctx context.Context, userID string) (bool, error)
	SetEnabled(ctx context.Context, userID string, enabled bool) error
}

type trainingReminderRepository struct {
	db *gorm.DB
}

func NewTrainingReminderRepository(db *gorm.DB) TrainingReminderRepository {
	return &trainingReminderRepository{db: db}
}

func (r *trainingReminderRepository) Due(ctx context.Context, idleSince, remindedSince time.Time, limit int) ([]ReminderCandidate, error) {
	var rows []ReminderCandidate
	err := r.db.WithContext(ctx).Raw(`
		SELECT u.id AS user_id, last.at AS last_training
		FROM users u
		LEFT JOIN LATERAL (
			SELECT MAX(ws.completed_at) AS at FROM workout_sessions ws WHERE ws.user_id = u.id AND ws.completed_at IS NOT NULL
		) last ON TRUE
		WHERE u.deleted_at IS NULL AND u.test_persona IS NULL AND u.email_verified_at IS NOT NULL
		  AND u.training_reminders
		  AND (u.last_training_reminder_at IS NULL OR u.last_training_reminder_at < ?)
		  AND COALESCE(last.at, u.created_at) < ?
		  AND (
		      EXISTS (SELECT 1 FROM free_program_activations f WHERE f.user_id = u.id AND f.deleted_at IS NULL)
		      OR EXISTS (SELECT 1 FROM entitlements e WHERE e.user_id = u.id AND e.deleted_at IS NULL
		                 AND (e.expires_at IS NULL OR e.expires_at > NOW()))
		  )
		ORDER BY COALESCE(last.at, u.created_at) ASC
		LIMIT ?`, remindedSince, idleSince, limit).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find members to remind: %w", err)
	}
	return rows, nil
}

func (r *trainingReminderRepository) MarkReminded(ctx context.Context, userID string, at time.Time) error {
	if err := r.db.WithContext(ctx).Exec(`UPDATE users SET last_training_reminder_at = ? WHERE id = ?`, at, userID).Error; err != nil {
		return fmt.Errorf("failed to record the reminder: %w", err)
	}
	return nil
}

func (r *trainingReminderRepository) Enabled(ctx context.Context, userID string) (bool, error) {
	var rows []struct{ TrainingReminders bool }
	if err := r.db.WithContext(ctx).Raw(`SELECT training_reminders FROM users WHERE id = ? AND deleted_at IS NULL`, userID).
		Scan(&rows).Error; err != nil {
		return false, fmt.Errorf("failed to read the reminder preference: %w", err)
	}
	if len(rows) == 0 {
		return false, ErrUserNotFound
	}
	return rows[0].TrainingReminders, nil
}

func (r *trainingReminderRepository) SetEnabled(ctx context.Context, userID string, enabled bool) error {
	if err := r.db.WithContext(ctx).Exec(`UPDATE users SET training_reminders = ? WHERE id = ? AND deleted_at IS NULL`, enabled, userID).Error; err != nil {
		return fmt.Errorf("failed to save the reminder preference: %w", err)
	}
	return nil
}
