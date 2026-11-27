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

// ErrRefundNotAllowed: only a completed purchase can be refunded.
var ErrRefundNotAllowed = errors.New("purchase cannot be refunded")

// RefundRecord is what the refund writes on the purchase.
type RefundRecord struct {
	Reason      string
	Note        string
	AdminID     string
	ProviderRef string
	At          time.Time
}

// RefundRepository marks a purchase refunded and removes the access it gave
// in one transaction (v2.131.0).
type RefundRepository interface {
	// FindRefundable returns the purchase when it can be refunded.
	FindRefundable(ctx context.Context, purchaseID string) (*models.Purchase, error)
	Refund(ctx context.Context, purchaseID string, record RefundRecord) (*models.Purchase, error)
}

type refundRepository struct {
	db *gorm.DB
}

func NewRefundRepository(db *gorm.DB) RefundRepository {
	return &refundRepository{db: db}
}

func (r *refundRepository) FindRefundable(ctx context.Context, purchaseID string) (*models.Purchase, error) {
	var purchase models.Purchase
	if err := r.db.WithContext(ctx).First(&purchase, "id = ?", purchaseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPurchaseNotFound
		}
		return nil, fmt.Errorf("failed to load purchase: %w", err)
	}
	if purchase.Status != models.PurchaseStatusCompleted {
		return nil, ErrRefundNotAllowed
	}
	return &purchase, nil
}

func (r *refundRepository) Refund(ctx context.Context, purchaseID string, record RefundRecord) (*models.Purchase, error) {
	var purchase models.Purchase
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&purchase, "id = ?", purchaseID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPurchaseNotFound
			}
			return fmt.Errorf("failed to lock purchase: %w", err)
		}
		if purchase.Status != models.PurchaseStatusCompleted {
			return ErrRefundNotAllowed
		}
		updates := map[string]any{
			"status":        models.PurchaseStatusRefunded,
			"refunded_at":   record.At,
			"refund_reason": record.Reason,
			"refunded_by":   record.AdminID,
		}
		if record.Note != "" {
			updates["refund_note"] = record.Note
		}
		if record.ProviderRef != "" {
			updates["refund_provider_ref"] = record.ProviderRef
		}
		if err := tx.Model(&models.Purchase{}).Where("id = ?", purchaseID).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to mark the purchase refunded: %w", err)
		}
		// The access the purchase gave ends now.
		if err := tx.Exec(`UPDATE entitlements SET deleted_at = ? WHERE user_id = ? AND program_id = ? AND deleted_at IS NULL`,
			record.At, purchase.UserID, purchase.ProgramID).Error; err != nil {
			return fmt.Errorf("failed to revoke the access: %w", err)
		}
		// Premium Level 2: the request ends and a delivered personal plan is
		// removed too; open offers are withdrawn.
		var requests []models.PremiumLevel2Request
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND program_id = ? AND status <> ?", purchase.UserID, purchase.ProgramID, models.PremiumLevel2StatusCancelled).
			Find(&requests).Error; err != nil {
			return fmt.Errorf("failed to load the service request: %w", err)
		}
		for _, request := range requests {
			if request.TrainingProgramID != nil && request.Status == models.PremiumLevel2StatusDelivered {
				if err := tx.Exec(`UPDATE entitlements SET deleted_at = ? WHERE user_id = ? AND program_id = ? AND deleted_at IS NULL`,
					record.At, purchase.UserID, *request.TrainingProgramID).Error; err != nil {
					return fmt.Errorf("failed to revoke the personal plan: %w", err)
				}
			}
			if err := tx.Model(&models.PremiumLevel2Request{}).Where("id = ?", request.ID).
				Update("status", models.PremiumLevel2StatusCancelled).Error; err != nil {
				return fmt.Errorf("failed to cancel the service request: %w", err)
			}
			if err := withdrawPremiumLevel2Offers(tx, request.ID); err != nil {
				return err
			}
			if err := createPremiumLevel2Event(tx, request.ID, "admin", record.AdminID, "refunded", request.Status, models.PremiumLevel2StatusCancelled); err != nil {
				return err
			}
		}
		purchase.Status = models.PurchaseStatusRefunded
		purchase.RefundedAt = &record.At
		purchase.RefundReason = &record.Reason
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &purchase, nil
}
