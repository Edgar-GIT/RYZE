package repositories

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ryze/backend/models"
)

// EmailOutboxRepository persists transactional emails and hands due ones to the
// delivery worker.
type EmailOutboxRepository interface {
	// Enqueue records an email. When the dedupe key is already recorded the
	// call is a no-op and reports false, so a repeated event never produces a
	// second email.
	Enqueue(ctx context.Context, email *models.EmailOutbox) (bool, error)
	// ClaimDue moves up to limit due pending emails to "sending" and returns
	// them. Rows are locked with SKIP LOCKED, so concurrent workers never claim
	// the same email.
	ClaimDue(ctx context.Context, now time.Time, limit int) ([]models.EmailOutbox, error)
	// MarkSent records a delivered email and clears its payload.
	MarkSent(ctx context.Context, id string, at time.Time) error
	// MarkRetry returns an email to pending for a later attempt.
	MarkRetry(ctx context.Context, id string, next time.Time, reason string) error
	// MarkFailed records a permanent failure and clears the payload.
	MarkFailed(ctx context.Context, id string, reason string) error
	// ExpireOverdue closes pending emails whose expiry has passed (for example a
	// password-reset link that could no longer be used) and clears them.
	ExpireOverdue(ctx context.Context, now time.Time) (int64, error)
	// RecoverStale returns emails left in "sending" by a crashed worker.
	RecoverStale(ctx context.Context, olderThan time.Time) (int64, error)
}

type emailOutboxRepository struct {
	db *gorm.DB
}

func NewEmailOutboxRepository(db *gorm.DB) EmailOutboxRepository {
	return &emailOutboxRepository{db: db}
}

func (r *emailOutboxRepository) Enqueue(ctx context.Context, email *models.EmailOutbox) (bool, error) {
	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "dedupe_key"}}, DoNothing: true}).
		Create(email)
	if result.Error != nil {
		return false, fmt.Errorf("failed to enqueue email: %w", result.Error)
	}
	return result.RowsAffected == 1, nil
}

func (r *emailOutboxRepository) ClaimDue(ctx context.Context, now time.Time, limit int) ([]models.EmailOutbox, error) {
	var claimed []models.EmailOutbox
	err := r.db.WithContext(ctx).Raw(`
		UPDATE email_outbox SET status = ?, attempts = attempts + 1, updated_at = ?
		WHERE id IN (
			SELECT id FROM email_outbox
			WHERE status = ? AND next_attempt_at <= ? AND (expires_at IS NULL OR expires_at > ?)
			ORDER BY next_attempt_at
			LIMIT ?
			FOR UPDATE SKIP LOCKED
		)
		RETURNING *`,
		models.EmailStatusSending, now, models.EmailStatusPending, now, now, limit,
	).Scan(&claimed).Error
	if err != nil {
		return nil, fmt.Errorf("failed to claim due emails: %w", err)
	}
	return claimed, nil
}

func (r *emailOutboxRepository) MarkSent(ctx context.Context, id string, at time.Time) error {
	return r.finish(ctx, id, map[string]any{
		"status":     models.EmailStatusSent,
		"sent_at":    at,
		"last_error": nil,
		"payload":    gorm.Expr("'{}'::jsonb"),
	})
}

func (r *emailOutboxRepository) MarkRetry(ctx context.Context, id string, next time.Time, reason string) error {
	return r.finish(ctx, id, map[string]any{
		"status":          models.EmailStatusPending,
		"next_attempt_at": next,
		"last_error":      truncateReason(reason),
	})
}

func (r *emailOutboxRepository) MarkFailed(ctx context.Context, id string, reason string) error {
	return r.finish(ctx, id, map[string]any{
		"status":     models.EmailStatusFailed,
		"last_error": truncateReason(reason),
		"payload":    gorm.Expr("'{}'::jsonb"),
	})
}

func (r *emailOutboxRepository) ExpireOverdue(ctx context.Context, now time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.EmailOutbox{}).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at <= ?", models.EmailStatusPending, now).
		Updates(map[string]any{"status": models.EmailStatusExpired, "payload": gorm.Expr("'{}'::jsonb")})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to expire emails: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *emailOutboxRepository) RecoverStale(ctx context.Context, olderThan time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.EmailOutbox{}).
		Where("status = ? AND updated_at < ?", models.EmailStatusSending, olderThan).
		Update("status", models.EmailStatusPending)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to recover stale emails: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// finish records the outcome of a delivery attempt. It applies only while the
// row is still in the "sending" state this worker claimed: if the row was
// recovered as stale and claimed again elsewhere, a late outcome from the
// first attempt can no longer overwrite the newer state.
func (r *emailOutboxRepository) finish(ctx context.Context, id string, updates map[string]any) error {
	if err := r.db.WithContext(ctx).Model(&models.EmailOutbox{}).
		Where("id = ? AND status = ?", id, models.EmailStatusSending).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update email %s: %w", id, err)
	}
	return nil
}

// truncateReason keeps the stored diagnostic within the column. Callers pass
// only generic delivery errors, never message content or credentials.
func truncateReason(reason string) string {
	if len(reason) > 250 {
		return reason[:250]
	}
	return reason
}
