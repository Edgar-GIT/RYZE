package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"ryze/backend/models"
)

var (
	// ErrPurchaseNotFound indicates the purchase does not exist, is
	// soft-deleted or does not belong to the requested user.
	ErrPurchaseNotFound = errors.New("purchase not found")
)

// PurchaseRepository defines the data-access operations for the purchase
// entity. Every operation is scoped to an explicit user id; the repository
// never obtains it from an HTTP context.
type PurchaseRepository interface {
	Create(ctx context.Context, purchase *models.Purchase) error
	FindByID(ctx context.Context, purchaseID string) (*models.Purchase, error)
	FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.Purchase, error)
}

type purchaseRepository struct {
	db *gorm.DB
}

func NewPurchaseRepository(db *gorm.DB) PurchaseRepository {
	return &purchaseRepository{db: db}
}

// Create persists a new purchase record. The caller must verify that the user
// and program exist and that no duplicate active purchase already exists.
func (r *purchaseRepository) Create(ctx context.Context, purchase *models.Purchase) error {
	if err := r.db.WithContext(ctx).Create(purchase).Error; err != nil {
		return fmt.Errorf("failed to create purchase: %w", err)
	}
	return nil
}

// FindByID returns one active (non-deleted) purchase by its id without user
// scoping. An unknown or soft-deleted purchase maps to ErrPurchaseNotFound.
func (r *purchaseRepository) FindByID(ctx context.Context, purchaseID string) (*models.Purchase, error) {
	var purchase models.Purchase
	if err := r.db.WithContext(ctx).
		First(&purchase, "id = ?", purchaseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPurchaseNotFound
		}
		return nil, fmt.Errorf("failed to find purchase: %w", err)
	}
	return &purchase, nil
}

// FindActiveByUserAndProgram returns the most recently created active
// (non-deleted) purchase for the given user and program pair, or
// ErrPurchaseNotFound when none exists. This method is used to detect duplicate
// purchase intent before creating a new purchase.
func (r *purchaseRepository) FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.Purchase, error) {
	var purchase models.Purchase
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND program_id = ?", userID, programID).
		Order("created_at DESC, id DESC").
		First(&purchase).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPurchaseNotFound
		}
		return nil, fmt.Errorf("failed to find active purchase: %w", err)
	}
	return &purchase, nil
}
