package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"ryze/backend/models"
)

var (
	// ErrCommissionRuleNotFound indicates the commission rule does not exist,
	// is soft-deleted or is not owned by the trainer performing the operation.
	ErrCommissionRuleNotFound = errors.New("commission rule not found")
)

// CommissionRuleRepository defines the data-access operations for the
// commission-rule entity. Every operation receives the trainer id explicitly;
// the repository never obtains it from an HTTP context.
type CommissionRuleRepository interface {
	// FindActiveOverride returns the active (non-deleted) commission rule for
	// the given trainer. When multiple rules exist the one with the most
	// recent valid_from is returned. A missing, soft-deleted or expired rule
	// maps to ErrCommissionRuleNotFound.
	FindActiveOverride(ctx context.Context, trainerID string) (*models.CommissionRule, error)
	// UpsertOverride creates a new commission rule or replaces the existing
	// active one for the given trainer. The previous active rule is soft-deleted
	// before the new one is inserted, so exactly one active override exists per
	// trainer at any time.
	UpsertOverride(ctx context.Context, rule *models.CommissionRule) error
	// SoftDelete removes the active commission rule for the given trainer.
	// A missing rule maps to ErrCommissionRuleNotFound.
	SoftDelete(ctx context.Context, trainerID string) error
}

type commissionRuleRepository struct {
	db *gorm.DB
}

func NewCommissionRuleRepository(db *gorm.DB) CommissionRuleRepository {
	return &commissionRuleRepository{db: db}
}

// FindActiveOverride returns the active (non-deleted) commission rule for the
// given trainer. When multiple non-deleted rules exist the one with the most
// recent valid_from is returned. A missing or soft-deleted rule maps to
// ErrCommissionRuleNotFound.
func (r *commissionRuleRepository) FindActiveOverride(ctx context.Context, trainerID string) (*models.CommissionRule, error) {
	var rule models.CommissionRule
	if err := r.db.WithContext(ctx).
		Where("trainer_id = ?", trainerID).
		Order("valid_from DESC, id ASC").
		First(&rule).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCommissionRuleNotFound
		}
		return nil, fmt.Errorf("failed to find active commission rule: %w", err)
	}
	return &rule, nil
}

// UpsertOverride soft-deletes any existing active commission rule for the given
// trainer and inserts the new rule. The operation is performed inside a
// transaction so either both steps succeed or neither is persisted.
func (r *commissionRuleRepository) UpsertOverride(ctx context.Context, rule *models.CommissionRule) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Model(&models.CommissionRule{}).
			Where("trainer_id = ? AND deleted_at IS NULL", rule.TrainerID).
			Update("deleted_at", gorm.Expr("NOW(6)")).Error; err != nil {
			return fmt.Errorf("failed to soft-delete previous commission rule: %w", err)
		}
		if err := tx.Create(rule).Error; err != nil {
			return fmt.Errorf("failed to create commission rule: %w", err)
		}
		return nil
	})
}

// SoftDelete soft-deletes the active commission rule for the given trainer. A
// missing rule maps to ErrCommissionRuleNotFound.
func (r *commissionRuleRepository) SoftDelete(ctx context.Context, trainerID string) error {
	result := r.db.WithContext(ctx).
		Delete(&models.CommissionRule{}, "trainer_id = ? AND deleted_at IS NULL", trainerID)
	if result.Error != nil {
		return fmt.Errorf("failed to soft delete commission rule: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrCommissionRuleNotFound
	}
	return nil
}
