package repositories

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// HealthConsentRepository keeps the person's explicit consent for health
// data (GDPR art. 9(2)(a), migration 000066).
type HealthConsentRepository interface {
	// ConsentedAt returns when the person consented (nil: not now).
	ConsentedAt(ctx context.Context, userID string) (*time.Time, error)
	// Give records the consent (the first time stays).
	Give(ctx context.Context, userID string, at time.Time) error
	// StampIntake records that a Premium Level 2 questionnaire was sent with
	// the consent.
	StampIntake(ctx context.Context, intakeID string, at time.Time) error
	// Withdraw removes the consent and erases the health data: the body
	// measurements and the answers of the Premium Level 2 questionnaires.
	Withdraw(ctx context.Context, userID string) error
}

type healthConsentRepository struct {
	db *gorm.DB
}

func NewHealthConsentRepository(db *gorm.DB) HealthConsentRepository {
	return &healthConsentRepository{db: db}
}

func (r *healthConsentRepository) ConsentedAt(ctx context.Context, userID string) (*time.Time, error) {
	var rows []struct{ HealthConsentAt *time.Time }
	if err := r.db.WithContext(ctx).Raw(`SELECT health_consent_at FROM users WHERE id = ? AND deleted_at IS NULL`, userID).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to read health consent: %w", err)
	}
	if len(rows) == 0 {
		return nil, ErrUserNotFound
	}
	return rows[0].HealthConsentAt, nil
}

func (r *healthConsentRepository) Give(ctx context.Context, userID string, at time.Time) error {
	if err := r.db.WithContext(ctx).Exec(`UPDATE users SET health_consent_at = COALESCE(health_consent_at, ?) WHERE id = ? AND deleted_at IS NULL`,
		at, userID).Error; err != nil {
		return fmt.Errorf("failed to record health consent: %w", err)
	}
	return nil
}

func (r *healthConsentRepository) StampIntake(ctx context.Context, intakeID string, at time.Time) error {
	if err := r.db.WithContext(ctx).Exec(`UPDATE premium_level_2_intakes SET health_consent_at = ? WHERE id = ?`, at, intakeID).Error; err != nil {
		return fmt.Errorf("failed to record intake consent: %w", err)
	}
	return nil
}

func (r *healthConsentRepository) Withdraw(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`UPDATE users SET health_consent_at = NULL WHERE id = ?`, userID).Error; err != nil {
			return fmt.Errorf("failed to withdraw health consent: %w", err)
		}
		if err := tx.Exec(`DELETE FROM body_measurements WHERE user_id = ?`, userID).Error; err != nil {
			return fmt.Errorf("failed to erase body measurements: %w", err)
		}
		if err := tx.Exec(`UPDATE premium_level_2_intakes SET answers = '{}', health_consent_at = NULL
			WHERE request_id IN (SELECT id FROM premium_level_2_requests WHERE user_id = ?)`, userID).Error; err != nil {
			return fmt.Errorf("failed to erase questionnaire answers: %w", err)
		}
		return nil
	})
}
