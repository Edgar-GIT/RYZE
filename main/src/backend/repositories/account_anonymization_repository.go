package repositories

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// AnonymizedFiles are the stored pictures of an anonymised account, deleted
// from the uploads after the database transaction.
type AnonymizedFiles struct {
	Avatar         *string
	SessionPhotos  []string
	FeedbackImages []string
}

// AccountAnonymizationRepository erases the personal data of accounts
// deleted more than the grace period ago (GDPR art. 17, migration 000067).
type AccountAnonymizationRepository interface {
	// Due lists accounts deleted before the cutoff and not anonymised yet.
	Due(ctx context.Context, cutoff time.Time, limit int) ([]string, error)
	// Anonymize erases the account's personal data in one transaction and
	// returns the pictures to delete.
	Anonymize(ctx context.Context, userID string, now time.Time) (*AnonymizedFiles, error)
}

type accountAnonymizationRepository struct {
	db *gorm.DB
}

func NewAccountAnonymizationRepository(db *gorm.DB) AccountAnonymizationRepository {
	return &accountAnonymizationRepository{db: db}
}

func (r *accountAnonymizationRepository) Due(ctx context.Context, cutoff time.Time, limit int) ([]string, error) {
	var ids []string
	if err := r.db.WithContext(ctx).Raw(`
		SELECT id FROM users
		WHERE deleted_at IS NOT NULL AND deleted_at < ? AND anonymized_at IS NULL AND test_persona IS NULL
		ORDER BY deleted_at ASC
		LIMIT ?`, cutoff, limit).Scan(&ids).Error; err != nil {
		return nil, fmt.Errorf("failed to list accounts to anonymise: %w", err)
	}
	return ids, nil
}

func (r *accountAnonymizationRepository) Anonymize(ctx context.Context, userID string, now time.Time) (*AnonymizedFiles, error) {
	files := &AnonymizedFiles{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var account []struct {
			AvatarKey *string
		}
		if err := tx.Raw(`SELECT avatar_key FROM users WHERE id = ? AND deleted_at IS NOT NULL AND anonymized_at IS NULL FOR UPDATE`, userID).
			Scan(&account).Error; err != nil {
			return fmt.Errorf("failed to lock the account: %w", err)
		}
		if len(account) == 0 {
			return nil // restored or already anonymised meanwhile
		}
		files.Avatar = account[0].AvatarKey
		if err := tx.Raw(`SELECT photo_key FROM workout_sessions WHERE user_id = ? AND photo_key IS NOT NULL`, userID).
			Scan(&files.SessionPhotos).Error; err != nil {
			return fmt.Errorf("failed to list session photos: %w", err)
		}
		if err := tx.Raw(`SELECT image_key FROM member_feedback WHERE user_id = ? AND image_key IS NOT NULL`, userID).
			Scan(&files.FeedbackImages).Error; err != nil {
			return fmt.Errorf("failed to list feedback pictures: %w", err)
		}
		statements := []struct {
			sql  string
			args []any
		}{
			// Identity: nothing left points to the person. The username keeps
			// the account's id so it stays unique.
			{`UPDATE users SET
				email = 'deleted-' || id || '@anonymized.invalid',
				password_hash = '',
				password_set = FALSE,
				first_name = '',
				last_name = '',
				username = 'deleted_' || substr(replace(id, '-', ''), 1, 20),
				username_chosen = FALSE,
				display_name = NULL,
				avatar_key = NULL,
				preferred_language = NULL,
				sex = NULL,
				plan_audience_filter = NULL,
				health_consent_at = NULL,
				anonymized_at = ?
			  WHERE id = ?`, []any{now, userID}},
			{`DELETE FROM user_profile_details WHERE user_id = ?`, []any{userID}},
			{`DELETE FROM user_identities WHERE user_id = ?`, []any{userID}},
			{`DELETE FROM user_tokens WHERE user_id = ?`, []any{userID}},
			{`DELETE FROM user_follows WHERE follower_id = ? OR followee_id = ?`, []any{userID, userID}},
			{`DELETE FROM user_plan_focus WHERE user_id = ?`, []any{userID}},
			{`DELETE FROM notifications WHERE user_id = ?`, []any{userID}},
			{`DELETE FROM email_outbox WHERE user_id = ?`, []any{userID}},
			// Health data.
			{`DELETE FROM body_measurements WHERE user_id = ?`, []any{userID}},
			{`UPDATE premium_level_2_intakes SET answers = '{}', health_consent_at = NULL
			  WHERE request_id IN (SELECT id FROM premium_level_2_requests WHERE user_id = ?)`, []any{userID}},
			{`UPDATE nutrition_questionnaires SET answers = '{}' WHERE user_id = ?`, []any{userID}},
			// Content they published or kept privately.
			{`DELETE FROM member_feedback WHERE user_id = ?`, []any{userID}},
			{`UPDATE workout_sessions SET photo_key = NULL WHERE user_id = ?`, []any{userID}},
		}
		for _, statement := range statements {
			if err := tx.Exec(statement.sql, statement.args...).Error; err != nil {
				return fmt.Errorf("failed to anonymise the account: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
