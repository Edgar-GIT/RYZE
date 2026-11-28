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
	CountPublishedPackages(ctx context.Context, profile MatchProfile) (int64, error)
	ListMatchCandidates(ctx context.Context, userID string, profile MatchProfile) ([]models.Program, error)
	CreateMatch(ctx context.Context, match *models.PremiumLevel1Match) error
}

// MatchProfile is the client's Level 1 questionnaire answer: the objective, the
// sex, and the body data each package's ranges are checked against (v2.133.0).
type MatchProfile struct {
	Objective string
	Sex       string
	HeightCm  float64
	WeightKg  float64
	Age       int
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
// Since v2.132.0 (owner decision, 2026-11-27) trainers' Level 1 packages are
// matched too; archived packages are not.
//
// Since v2.133.0 the package's optional height, weight and age ranges must
// contain the client's answer (a NULL bound is "any").
func curatedPackages(db *gorm.DB, profile MatchProfile) *gorm.DB {
	return db.Model(&models.Program{}).
		Where("archived_at IS NULL").
		Where("product_type = ?", models.ProgramProductTypePremiumLevel1).
		Where("status = ?", models.ProgramStatusPublished).
		Where("nutrition_mode = ?", models.ProgramNutritionModeCurated).
		Where("objective = ?", profile.Objective).
		Where("(audience IS NULL OR audience = ?)", profile.Sex).
		Where("(min_height_cm IS NULL OR min_height_cm <= ?) AND (max_height_cm IS NULL OR max_height_cm >= ?)", profile.HeightCm, profile.HeightCm).
		Where("(min_weight_kg IS NULL OR min_weight_kg <= ?) AND (max_weight_kg IS NULL OR max_weight_kg >= ?)", profile.WeightKg, profile.WeightKg).
		Where("(min_age IS NULL OR min_age <= ?) AND (max_age IS NULL OR max_age >= ?)", profile.Age, profile.Age)
}

func (r *premiumJourneyRepository) CountPublishedPackages(ctx context.Context, profile MatchProfile) (int64, error) {
	var count int64
	if err := curatedPackages(r.db.WithContext(ctx), profile).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count premium level 1 packages: %w", err)
	}
	return count, nil
}

// ListMatchCandidates returns the packages of the objective the user does not
// own yet: no active entitlement and no completed purchase.
func (r *premiumJourneyRepository) ListMatchCandidates(ctx context.Context, userID string, profile MatchProfile) ([]models.Program, error) {
	var programs []models.Program
	// A package is "owned" through any of its versions (v2.133.0): the lineage
	// root identifies the package across versions.
	err := curatedPackages(r.db.WithContext(ctx), profile).
		Where(`COALESCE(version_root_id, id) NOT IN (
SELECT COALESCE(owned.version_root_id, owned.id) FROM programs owned
WHERE owned.id IN (SELECT program_id FROM entitlements WHERE user_id = ? AND deleted_at IS NULL)
   OR owned.id IN (SELECT program_id FROM purchases WHERE user_id = ? AND status = ? AND deleted_at IS NULL))`,
			userID, userID, models.PurchaseStatusCompleted).
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
