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

var (
	ErrDiscountCodeNotFound = errors.New("discount code not found")
	ErrDiscountCodeTaken    = errors.New("discount code already exists")
	// ErrDiscountLocked: the purchase's payment already opened; its price no
	// longer changes.
	ErrDiscountLocked = errors.New("purchase price is locked")
)

// DiscountCodeView is a code with how many completed purchases used it.
type DiscountCodeView struct {
	models.DiscountCode
	Uses int64 `gorm:"column:uses"`
}

// DiscountTarget is what a code is checked against: the purchase's program.
type DiscountTarget struct {
	ProductType string
	TrainerID   *string
}

type DiscountCodeRepository interface {
	Create(ctx context.Context, code *models.DiscountCode) error
	List(ctx context.Context) ([]DiscountCodeView, error)
	Disable(ctx context.Context, id string, at time.Time) error
	FindByCode(ctx context.Context, code string) (*models.DiscountCode, error)
	// CountUses counts the completed purchases that used the code.
	CountUses(ctx context.Context, codeID string) (int64, error)
	// UsedBy reports whether the person completed a purchase with the code.
	UsedBy(ctx context.Context, codeID, userID string) (bool, error)
	// Target returns the program a purchase is for (the buyer's own pending
	// purchase only).
	Target(ctx context.Context, purchaseID, userID string) (*DiscountTarget, error)
	// Apply sets the purchase's discount (nil removes it) while its payment
	// has not opened, recomputing the price and the commission split.
	Apply(ctx context.Context, purchaseID, userID string, code *models.DiscountCode) (*models.Purchase, error)
}

type discountCodeRepository struct {
	db *gorm.DB
}

func NewDiscountCodeRepository(db *gorm.DB) DiscountCodeRepository {
	return &discountCodeRepository{db: db}
}

func (r *discountCodeRepository) Create(ctx context.Context, code *models.DiscountCode) error {
	if err := inSavepoint(r.db.WithContext(ctx), func(tx *gorm.DB) error { return tx.Create(code).Error }); err != nil {
		if isDuplicateEntry(err) {
			return ErrDiscountCodeTaken
		}
		return fmt.Errorf("failed to create discount code: %w", err)
	}
	return nil
}

func (r *discountCodeRepository) List(ctx context.Context) ([]DiscountCodeView, error) {
	var rows []DiscountCodeView
	if err := r.db.WithContext(ctx).Raw(`
		SELECT d.*, (SELECT COUNT(*) FROM purchases p WHERE p.discount_code_id = d.id AND p.status = ? AND p.deleted_at IS NULL) AS uses
		FROM discount_codes d
		ORDER BY d.created_at DESC
		LIMIT 200`, models.PurchaseStatusCompleted).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to list discount codes: %w", err)
	}
	return rows, nil
}

func (r *discountCodeRepository) Disable(ctx context.Context, id string, at time.Time) error {
	result := r.db.WithContext(ctx).Exec(`UPDATE discount_codes SET disabled_at = COALESCE(disabled_at, ?) WHERE id = ?`, at, id)
	if result.Error != nil {
		return fmt.Errorf("failed to disable discount code: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrDiscountCodeNotFound
	}
	return nil
}

func (r *discountCodeRepository) FindByCode(ctx context.Context, code string) (*models.DiscountCode, error) {
	var found models.DiscountCode
	if err := r.db.WithContext(ctx).Where("code = ?", code).Take(&found).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDiscountCodeNotFound
		}
		return nil, fmt.Errorf("failed to find discount code: %w", err)
	}
	return &found, nil
}

func (r *discountCodeRepository) CountUses(ctx context.Context, codeID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Purchase{}).
		Where("discount_code_id = ? AND status = ?", codeID, models.PurchaseStatusCompleted).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count discount uses: %w", err)
	}
	return count, nil
}

func (r *discountCodeRepository) UsedBy(ctx context.Context, codeID, userID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Purchase{}).
		Where("discount_code_id = ? AND user_id = ? AND status = ?", codeID, userID, models.PurchaseStatusCompleted).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check discount use: %w", err)
	}
	return count > 0, nil
}

func (r *discountCodeRepository) Target(ctx context.Context, purchaseID, userID string) (*DiscountTarget, error) {
	var rows []struct {
		ProductType string
		TrainerID   *string
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT p.product_type, p.trainer_id
		FROM purchases pu JOIN programs p ON p.id = pu.program_id
		WHERE pu.id = ? AND pu.user_id = ? AND pu.deleted_at IS NULL`, purchaseID, userID).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load the purchase's program: %w", err)
	}
	if len(rows) == 0 {
		return nil, ErrPurchaseNotFound
	}
	return &DiscountTarget{ProductType: rows[0].ProductType, TrainerID: rows[0].TrainerID}, nil
}

func (r *discountCodeRepository) Apply(ctx context.Context, purchaseID, userID string, code *models.DiscountCode) (*models.Purchase, error) {
	var purchase models.Purchase
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&purchase, "id = ? AND user_id = ?", purchaseID, userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPurchaseNotFound
			}
			return fmt.Errorf("failed to lock purchase: %w", err)
		}
		if purchase.Status != models.PurchaseStatusPending {
			return ErrDiscountLocked
		}
		currentCode := ""
		if purchase.DiscountCodeID != nil {
			currentCode = *purchase.DiscountCodeID
		}
		wantedCode := ""
		if code != nil {
			wantedCode = code.ID
		}
		if currentCode == wantedCode {
			return nil
		}
		// The provider payment was opened for the current price.
		if purchase.PaymentMethod != nil {
			return ErrDiscountLocked
		}
		listPrice := purchase.PriceMinorUnits
		if purchase.OriginalPriceMinorUnits != nil {
			listPrice = *purchase.OriginalPriceMinorUnits
		}
		price, discount := listPrice, int64(0)
		var codeID, original any
		if code != nil {
			discount = listPrice * int64(code.PercentOff) / 100
			price = listPrice - discount
			codeID, original = code.ID, listPrice
		}
		if discount == 0 {
			codeID, original = nil, nil
		}
		// The split follows the price: a trainer's plan keeps its commission
		// rate; a platform plan stays wholly the platform's.
		platform, trainer := price, int64(0)
		if purchase.TrainerAmount > 0 {
			platform = price * int64(purchase.CommissionBPS) / 10000
			trainer = price - platform
		}
		updates := map[string]any{
			"price_minor_units":          price,
			"original_price_minor_units": original,
			"discount_minor_units":       discount,
			"discount_code_id":           codeID,
			"platform_amount":            platform,
			"trainer_amount":             trainer,
		}
		if err := tx.Model(&models.Purchase{}).Where("id = ?", purchaseID).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to apply the discount: %w", err)
		}
		return tx.First(&purchase, "id = ?", purchaseID).Error
	})
	if err != nil {
		return nil, err
	}
	return &purchase, nil
}
