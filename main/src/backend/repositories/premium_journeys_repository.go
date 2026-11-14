package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ryze/backend/models"
)

// ErrPremiumLevel2DraftNotFound indicates the client has no pre-purchase
// questionnaire for the offer.
var ErrPremiumLevel2DraftNotFound = errors.New("premium level 2 intake draft not found")

// PremiumJourneyRepository stores the Premium Level 1 picks and the Premium
// Level 2 pre-purchase questionnaires. Every query is scoped by the user.
type PremiumJourneyRepository interface {
	CountPublishedPackages(ctx context.Context, objective, sex string) (int64, error)
	ListMatchCandidates(ctx context.Context, userID, objective, sex string) ([]models.Program, error)
	CreateMatch(ctx context.Context, match *models.PremiumLevel1Match) error
	FindDraft(ctx context.Context, userID, programID string) (*models.PremiumLevel2IntakeDraft, error)
	SaveDraft(ctx context.Context, draft *models.PremiumLevel2IntakeDraft) error
	DeleteDraft(ctx context.Context, userID, programID string) error
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

func (r *premiumJourneyRepository) FindDraft(ctx context.Context, userID, programID string) (*models.PremiumLevel2IntakeDraft, error) {
	var draft models.PremiumLevel2IntakeDraft
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("program_id = ?", programID).
		First(&draft).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPremiumLevel2DraftNotFound
		}
		return nil, fmt.Errorf("failed to load premium level 2 intake draft: %w", err)
	}
	return &draft, nil
}

// SaveDraft inserts or replaces the user's draft for the offer.
func (r *premiumJourneyRepository) SaveDraft(ctx context.Context, draft *models.PremiumLevel2IntakeDraft) error {
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "program_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"answers", "complete"}),
	}).Create(draft).Error
	if err != nil {
		return fmt.Errorf("failed to save premium level 2 intake draft: %w", err)
	}
	return nil
}

func (r *premiumJourneyRepository) DeleteDraft(ctx context.Context, userID, programID string) error {
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("program_id = ?", programID).
		Delete(&models.PremiumLevel2IntakeDraft{}).Error; err != nil {
		return fmt.Errorf("failed to delete premium level 2 intake draft: %w", err)
	}
	return nil
}
