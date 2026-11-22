package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

var ErrSessionPhotoSessionNotFound = errors.New("workout session not found")

// SessionPhotoRepository reads and writes the photo key of a member's own
// workout session (owner request, 2026-11-22).
type SessionPhotoRepository interface {
	// Find returns the session status and photo key, scoped to its owner.
	Find(ctx context.Context, userID, sessionID string) (status string, key *string, err error)
	// Set stores the key (nil removes it) and returns the previous key.
	Set(ctx context.Context, userID, sessionID string, key *string) (*string, error)
}

type sessionPhotoRepository struct {
	db *gorm.DB
}

func NewSessionPhotoRepository(db *gorm.DB) SessionPhotoRepository {
	return &sessionPhotoRepository{db: db}
}

type sessionPhotoRow struct {
	Status   string
	PhotoKey *string
}

func (r *sessionPhotoRepository) Find(ctx context.Context, userID, sessionID string) (string, *string, error) {
	var rows []sessionPhotoRow
	if err := r.db.WithContext(ctx).Raw(`SELECT status, photo_key FROM workout_sessions WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		sessionID, userID).Scan(&rows).Error; err != nil {
		return "", nil, fmt.Errorf("failed to load workout photo: %w", err)
	}
	if len(rows) == 0 {
		return "", nil, ErrSessionPhotoSessionNotFound
	}
	return rows[0].Status, rows[0].PhotoKey, nil
}

func (r *sessionPhotoRepository) Set(ctx context.Context, userID, sessionID string, key *string) (*string, error) {
	var previous *string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []sessionPhotoRow
		if err := tx.Raw(`SELECT status, photo_key FROM workout_sessions WHERE id = ? AND user_id = ? AND deleted_at IS NULL FOR UPDATE`,
			sessionID, userID).Scan(&rows).Error; err != nil {
			return fmt.Errorf("failed to lock workout session: %w", err)
		}
		if len(rows) == 0 {
			return ErrSessionPhotoSessionNotFound
		}
		previous = rows[0].PhotoKey
		if err := tx.Exec(`UPDATE workout_sessions SET photo_key = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, key, sessionID).Error; err != nil {
			return fmt.Errorf("failed to save workout photo: %w", err)
		}
		return nil
	})
	return previous, err
}
