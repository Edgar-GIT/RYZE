package repositories

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"ryze/backend/models"
)

// PremiumJourneyRepository stores the Premium Level 1 picks. Every query is
// scoped by the user.
type PremiumJourneyRepository interface {
	CountPublishedPackages(ctx context.Context, objective, sex string) (int64, error)
	ListMatchCandidates(ctx context.Context, userID, objective, sex string) ([]models.Program, error)
	CreateMatch(ctx context.Context, match *models.PremiumLevel1Match) error
}

type premiumJourneyRepository struct {
	db *gorm.DB
}

func NewPremiumJourneyRepository(db *gorm.DB) PremiumJourneyRepository {
	return &premiumJourneyRepository{db: db}
}

// curatedPackages scopes to the published, platform-owned, curated Premium
// Level 1 packages of one objective.
// curatedPackages scopes the published curated packages of an objective that
// suit the client's sex: packages for everyone plus those designed for it.
func curatedPackages(db *gorm.DB, objective, sex string) *gorm.DB {
	return db.Model(&models.Program{}).
		Where("trainer_id IS NULL").
		Where("product_type = ?", models.ProgramProductTypePremiumLevel1).
		Where("status = ?", models.ProgramStatusPublished).
		Where("nutrition_mode = ?", models.ProgramNutritionModeCurated).
		Where("objective = ?", objective).
		Where("(audience IS NULL OR audience = ?)", sex)
}

func (r *premiumJourneyRepository) CountPublishedPackages(ctx context.Context, objective, sex string) (int64, error) {
	var count int64
	if err := curatedPackages(r.db.WithContext(ctx), objective, sex).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count premium level 1 packages: %w", err)
	}
	return count, nil
}

// ListMatchCandidates returns the packages of the objective the user does not
// own yet: no active entitlement and no completed purchase.
func (r *premiumJourneyRepository) ListMatchCandidates(ctx context.Context, userID, objective, sex string) ([]models.Program, error) {
	var programs []models.Program
	err := curatedPackages(r.db.WithContext(ctx), objective, sex).
		Where("id NOT IN (SELECT program_id FROM entitlements WHERE user_id = ? AND deleted_at IS NULL)", userID).
		Where("id NOT IN (SELECT program_id FROM purchases WHERE user_id = ? AND status = ? AND deleted_at IS NULL)", userID, models.PurchaseStatusCompleted).
		Order("created_at DESC").
		Find(&programs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list premium level 1 candidates: %w", err)
	}
	return programs, nil
}

func (r *premiumJourneyRepository) CreateMatch(ctx context.Context, match *models.PremiumLevel1Match) error {
	if err := r.db.WithContext(ctx).Create(match).Error; err != nil {
		return fmt.Errorf("failed to record premium level 1 match: %w", err)
	}
	return nil
}
