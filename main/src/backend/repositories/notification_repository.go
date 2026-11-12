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

// ErrNotificationNotFound indicates the notification does not exist, belongs
// to another account or has expired. The cases are indistinguishable.
var ErrNotificationNotFound = errors.New("notification not found")

// NotificationRepository stores temporary in-app notifications.
type NotificationRepository interface {
	// Create records a notification once per dedupe key. It reports whether a
	// row was inserted (false for a repeated event).
	Create(ctx context.Context, notification *models.Notification) (bool, error)
	// ListVisible returns the account's unexpired notifications, newest first,
	// and how many of them are unread.
	ListVisible(ctx context.Context, userID string, now time.Time, limit int) ([]models.Notification, int64, error)
	MarkRead(ctx context.Context, userID, id string, now time.Time) error
	MarkAllRead(ctx context.Context, userID string, now time.Time) (int64, error)
	// Purge clears the content of expired notifications and deletes those
	// expired before deleteBefore. It returns the number of rows deleted.
	Purge(ctx context.Context, now, deleteBefore time.Time) (cleared int64, deleted int64, err error)
	// LatestCompletedPurchase returns the most recent completed purchase of a
	// programme by the account (its id and whether it is a Test Mode purchase).
	LatestCompletedPurchase(ctx context.Context, userID, programID string) (*models.Purchase, error)
}

type notificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(ctx context.Context, notification *models.Notification) (bool, error) {
	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "dedupe_key"}}, DoNothing: true}).
		Create(notification)
	if result.Error != nil {
		return false, fmt.Errorf("failed to create notification: %w", result.Error)
	}
	return result.RowsAffected == 1, nil
}

func (r *notificationRepository) ListVisible(ctx context.Context, userID string, now time.Time, limit int) ([]models.Notification, int64, error) {
	var notifications []models.Notification
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND expires_at > ?", userID, now).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&notifications).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list notifications: %w", err)
	}
	var unread int64
	if err := r.db.WithContext(ctx).Model(&models.Notification{}).
		Where("user_id = ? AND expires_at > ? AND read_at IS NULL", userID, now).
		Count(&unread).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count unread notifications: %w", err)
	}
	return notifications, unread, nil
}

func (r *notificationRepository) MarkRead(ctx context.Context, userID, id string, now time.Time) error {
	result := r.db.WithContext(ctx).Model(&models.Notification{}).
		Where("id = ? AND user_id = ? AND expires_at > ?", id, userID, now).
		Update("read_at", gorm.Expr("COALESCE(read_at, ?)", now))
	if result.Error != nil {
		return fmt.Errorf("failed to mark notification read: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

func (r *notificationRepository) MarkAllRead(ctx context.Context, userID string, now time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.Notification{}).
		Where("user_id = ? AND expires_at > ? AND read_at IS NULL", userID, now).
		Update("read_at", now)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to mark notifications read: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *notificationRepository) Purge(ctx context.Context, now, deleteBefore time.Time) (int64, int64, error) {
	cleared := r.db.WithContext(ctx).Model(&models.Notification{}).
		Where("expires_at <= ? AND (link IS NOT NULL OR params <> '{}'::jsonb)", now).
		Updates(map[string]any{"params": gorm.Expr("'{}'::jsonb"), "link": nil})
	if cleared.Error != nil {
		return 0, 0, fmt.Errorf("failed to clear expired notifications: %w", cleared.Error)
	}
	deleted := r.db.WithContext(ctx).Where("expires_at < ?", deleteBefore).Delete(&models.Notification{})
	if deleted.Error != nil {
		return cleared.RowsAffected, 0, fmt.Errorf("failed to delete expired notifications: %w", deleted.Error)
	}
	return cleared.RowsAffected, deleted.RowsAffected, nil
}

func (r *notificationRepository) LatestCompletedPurchase(ctx context.Context, userID, programID string) (*models.Purchase, error) {
	var purchase models.Purchase
	err := r.db.WithContext(ctx).
		Select("id", "user_id", "program_id", "status", "test", "updated_at").
		Where("user_id = ? AND program_id = ? AND status = ?", userID, programID, models.PurchaseStatusCompleted).
		Order("updated_at DESC, id DESC").
		Take(&purchase).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPurchaseNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find completed purchase: %w", err)
	}
	return &purchase, nil
}
