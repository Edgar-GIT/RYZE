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
	Complete(ctx context.Context, purchaseID string) error
	CompleteWithEntitlement(ctx context.Context, purchaseID string, entitlement *models.Entitlement) error
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

// Complete atomically transitions a pending purchase to the completed status.
// It only touches rows currently in "pending" state; the WHERE clause ensures
// exactly one row is updated. When the purchase does not exist, is
// soft-deleted or is already in a non-pending state, ErrPurchaseNotFound is
// returned because the caller cannot distinguish between "not found" and
// "already processed" at this layer — the service is responsible for the
// idempotency semantics.
func (r *purchaseRepository) Complete(ctx context.Context, purchaseID string) error {
	result := r.db.WithContext(ctx).
		Model(&models.Purchase{}).
		Where("id = ? AND status = ?", purchaseID, models.PurchaseStatusPending).
		Update("status", models.PurchaseStatusCompleted)
	if result.Error != nil {
		return fmt.Errorf("failed to complete purchase: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrPurchaseNotFound
	}
	return nil
}

// CompleteWithEntitlement atomically transitions a pending purchase to
// completed and creates the entitlement row inside a single database
// transaction. If the purchase is not pending or does not exist,
// ErrPurchaseNotFound is returned and no row is modified. Duplicate entry
// errors on the entitlement are mapped to ErrEntitlementAlreadyExists so the
// service can react with an integrity error rather than exposing raw driver
// details.
func (r *purchaseRepository) CompleteWithEntitlement(ctx context.Context, purchaseID string, entitlement *models.Entitlement) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.Purchase{}).
			Where("id = ? AND status = ?", purchaseID, models.PurchaseStatusPending).
			Update("status", models.PurchaseStatusCompleted)
		if result.Error != nil {
			return fmt.Errorf("failed to complete purchase: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrPurchaseNotFound
		}

		if err := tx.Create(entitlement).Error; err != nil {
			if isDuplicateEntry(err) {
				return ErrEntitlementAlreadyExists
			}
			return fmt.Errorf("failed to create entitlement: %w", err)
		}
		return nil
	})
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
