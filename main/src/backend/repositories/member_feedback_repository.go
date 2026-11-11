package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ryze/backend/models"
)

var (
	// ErrMemberFeedbackNotFound indicates the requested feedback does not exist
	// (soft-deleted feedback is never visible to regular operations).
	ErrMemberFeedbackNotFound = errors.New("member feedback not found")
	// ErrMemberFeedbackStateConflict indicates the feedback is no longer pending,
	// so it cannot be reviewed again.
	ErrMemberFeedbackStateConflict = errors.New("member feedback is not pending")
	// ErrMemberFeedbackAlreadyPending indicates the member already has a pending
	// submission. The rule is enforced by the unique index on pending_user_id.
	ErrMemberFeedbackAlreadyPending = errors.New("member already has pending feedback")
)

// MemberFeedbackRepository defines the data-access operations for member
// feedback and the plan-ownership check that gates submissions.
type MemberFeedbackRepository interface {
	Create(ctx context.Context, feedback *models.MemberFeedback) error
	FindPendingByUserID(ctx context.Context, userID string) (*models.MemberFeedback, error)
	// HasHeldPlan reports whether the user holds or has held a plan: a
	// purchase-backed entitlement (every paid product, including Premium Level
	// 2) or a Free activation. Expired access still counts; revoked
	// (soft-deleted) access does not.
	HasHeldPlan(ctx context.Context, userID string) (bool, error)
	// ListPublished returns approved feedback from active accounts, newest first.
	ListPublished(ctx context.Context, page, limit int) ([]models.MemberFeedback, int64, error)
	// ListByStatus returns feedback for review, optionally filtered by status,
	// newest first.
	ListByStatus(ctx context.Context, status string, page, limit int) ([]models.MemberFeedback, int64, error)
	// Review moves a pending submission to approved or rejected and records the
	// reviewer. A submission that is no longer pending is never changed.
	Review(ctx context.Context, id, status, reviewer string, at time.Time) (*models.MemberFeedback, error)
}

type memberFeedbackRepository struct {
	db *gorm.DB
}

func NewMemberFeedbackRepository(db *gorm.DB) MemberFeedbackRepository {
	return &memberFeedbackRepository{db: db}
}

func (r *memberFeedbackRepository) Create(ctx context.Context, feedback *models.MemberFeedback) error {
	if err := r.db.WithContext(ctx).Create(feedback).Error; err != nil {
		if isDuplicateEntry(err) {
			return ErrMemberFeedbackAlreadyPending
		}
		return fmt.Errorf("failed to create member feedback: %w", err)
	}
	return nil
}

func (r *memberFeedbackRepository) FindPendingByUserID(ctx context.Context, userID string) (*models.MemberFeedback, error) {
	var feedback models.MemberFeedback
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, models.MemberFeedbackStatusPending).
		First(&feedback).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMemberFeedbackNotFound
		}
		return nil, fmt.Errorf("failed to find pending member feedback: %w", err)
	}
	return &feedback, nil
}

func (r *memberFeedbackRepository) HasHeldPlan(ctx context.Context, userID string) (bool, error) {
	var entitlements int64
	if err := r.db.WithContext(ctx).Model(&models.Entitlement{}).
		Where("user_id = ?", userID).
		Count(&entitlements).Error; err != nil {
		return false, fmt.Errorf("failed to check entitlements: %w", err)
	}
	if entitlements > 0 {
		return true, nil
	}
	var activations int64
	if err := r.db.WithContext(ctx).Model(&models.FreeProgramActivation{}).
		Where("user_id = ?", userID).
		Count(&activations).Error; err != nil {
		return false, fmt.Errorf("failed to check free activations: %w", err)
	}
	return activations > 0, nil
}

func (r *memberFeedbackRepository) ListPublished(ctx context.Context, page, limit int) ([]models.MemberFeedback, int64, error) {
	// Feedback from deleted accounts is never shown.
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Model(&models.MemberFeedback{}).
			Joins("JOIN users ON users.id = member_feedback.user_id AND users.deleted_at IS NULL").
			Where("member_feedback.status = ?", models.MemberFeedbackStatusApproved)
	}
	var total int64
	if err := scope(r.db.WithContext(ctx)).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count published member feedback: %w", err)
	}
	var feedback []models.MemberFeedback
	// Select only feedback columns: the join must never let users.id or
	// users.created_at overwrite the feedback's own fields.
	if err := scope(r.db.WithContext(ctx)).
		Select("member_feedback.*").
		Order("member_feedback.reviewed_at DESC, member_feedback.id ASC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&feedback).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list published member feedback: %w", err)
	}
	return feedback, total, nil
}

func (r *memberFeedbackRepository) ListByStatus(ctx context.Context, status string, page, limit int) ([]models.MemberFeedback, int64, error) {
	scope := func(db *gorm.DB) *gorm.DB {
		query := db.Model(&models.MemberFeedback{})
		if status != "" {
			query = query.Where("status = ?", status)
		}
		return query
	}
	var total int64
	if err := scope(r.db.WithContext(ctx)).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count member feedback: %w", err)
	}
	var feedback []models.MemberFeedback
	if err := scope(r.db.WithContext(ctx)).
		Order("created_at DESC, id ASC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&feedback).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list member feedback: %w", err)
	}
	return feedback, total, nil
}

func (r *memberFeedbackRepository) Review(ctx context.Context, id, status, reviewer string, at time.Time) (*models.MemberFeedback, error) {
	var reviewed models.MemberFeedback
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current models.MemberFeedback
		if err := tx.First(&current, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrMemberFeedbackNotFound
			}
			return fmt.Errorf("failed to find member feedback: %w", err)
		}
		// The status condition makes the transition atomic: a concurrent review
		// that already moved the row leaves nothing to update.
		result := tx.Model(&models.MemberFeedback{}).
			Where("id = ? AND status = ?", id, models.MemberFeedbackStatusPending).
			Updates(map[string]any{"status": status, "reviewed_by": reviewer, "reviewed_at": at})
		if result.Error != nil {
			return fmt.Errorf("failed to review member feedback: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrMemberFeedbackStateConflict
		}
		return tx.First(&reviewed, "id = ?", id).Error
	})
	if err != nil {
		return nil, err
	}
	return &reviewed, nil
}
