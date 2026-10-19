package repositories

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"ryze/backend/models"
)

// TrainerApplicationDecisionRepository defines the data-access operations for
// the append-only trainer-application decision history. Rows are created only
// inside the review transactions and are never updated or deleted.
type TrainerApplicationDecisionRepository interface {
	Create(ctx context.Context, decision *models.TrainerApplicationDecision) error
	ListByApplication(ctx context.Context, applicationID string) ([]models.TrainerApplicationDecision, error)
}

type trainerApplicationDecisionRepository struct {
	db *gorm.DB
}

func NewTrainerApplicationDecisionRepository(db *gorm.DB) TrainerApplicationDecisionRepository {
	return &trainerApplicationDecisionRepository{db: db}
}

func (r *trainerApplicationDecisionRepository) Create(ctx context.Context, decision *models.TrainerApplicationDecision) error {
	if err := r.db.WithContext(ctx).Create(decision).Error; err != nil {
		return fmt.Errorf("failed to create trainer application decision: %w", err)
	}
	return nil
}

func (r *trainerApplicationDecisionRepository) ListByApplication(ctx context.Context, applicationID string) ([]models.TrainerApplicationDecision, error) {
	var decisions []models.TrainerApplicationDecision
	if err := r.db.WithContext(ctx).
		Where("application_id = ?", applicationID).
		Order("created_at ASC, id ASC").
		Find(&decisions).Error; err != nil {
		return nil, fmt.Errorf("failed to list trainer application decisions: %w", err)
	}
	return decisions, nil
}
