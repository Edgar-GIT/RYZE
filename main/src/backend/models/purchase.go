package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PurchaseStatus values describe the lifecycle of a purchase record. This is
// a schema-only preparation: no purchase creation path exists yet.
// PurchaseStatus values describe the lifecycle of a purchase record. RYZE
// purchases are final: there is no refund workflow, so "refunded" is not a
// valid status. This is a schema-only preparation: no purchase creation path
// exists yet.
const (
	PurchaseStatusPending   = "pending"
	PurchaseStatusCompleted = "completed"
	PurchaseStatusFailed    = "failed"
	// PurchaseStatusRefunded: ADMIN_2 refunded the purchase (v2.131.0); its
	// access was removed and it is no longer a sale.
	PurchaseStatusRefunded = "refunded"
)

// Purchase corresponds to the purchases table. It records a user's commercial
// intent to acquire a published program. The purchase carries a full commercial
// snapshot (price, currency, commission split) captured at creation time so the
// transaction remains historically accurate even if program pricing or
// commission rules change later. Soft-deleted purchases are excluded from
// regular queries through GORM's DeletedAt handling.
type Purchase struct {
	ID              string `gorm:"type:char(36);primaryKey" json:"id"`
	UserID          string `gorm:"column:user_id;type:varchar(36);not null" json:"user_id"`
	ProgramID       string `gorm:"column:program_id;type:varchar(36);not null" json:"program_id"`
	PriceMinorUnits int64  `gorm:"column:price_minor_units;type:bigint;not null;default:0" json:"price_minor_units"`
	Currency        string `gorm:"column:currency;type:varchar(3);not null;default:EUR" json:"currency"`
	CommissionBPS   uint32 `gorm:"column:commission_bps;type:int unsigned;not null;default:0" json:"commission_bps"`
	PlatformAmount  int64  `gorm:"column:platform_amount;type:bigint;not null;default:0" json:"platform_amount"`
	TrainerAmount   int64  `gorm:"column:trainer_amount;type:bigint;not null;default:0" json:"trainer_amount"`
	Status          string `gorm:"column:status;type:varchar(20);not null;default:pending" json:"status"`
	// PaymentMethod records the payment method selected at initiation. It is
	// immutable once set and lets the capture flow resolve the correct provider
	// server-side. NULL means the purchase never reached a payment provider
	// (Test Mode purchases, legacy pending purchases); the pointer keeps the
	// empty string out of the database so the CHECK constraint only ever sees
	// NULL or one of the allowlisted values.
	PaymentMethod *string `gorm:"column:payment_method;type:varchar(20)" json:"payment_method"`
	Test          bool    `gorm:"column:test;not null;default:false" json:"-"`
	// BuyerTaxID is the optional Portuguese taxpayer number (NIF) the buyer
	// gave at checkout, validated by the server; printed on the purchase
	// document. NULL means "final consumer". Never logged.
	BuyerTaxID *string `gorm:"column:buyer_tax_id;type:varchar(9)" json:"-"`
	// TermsAcceptedAt and ImmediateAccessRequestedAt record the buyer's
	// checkout consents (v2.130.12, EU consumer rules). Never exposed.
	TermsAcceptedAt            *time.Time `gorm:"column:terms_accepted_at" json:"-"`
	ImmediateAccessRequestedAt *time.Time `gorm:"column:immediate_access_requested_at" json:"-"`
	// The discount code applied at checkout (migration 000069): the list
	// price before it and the amount off; PriceMinorUnits is what was paid.
	DiscountCodeID          *string `gorm:"column:discount_code_id;type:varchar(36)" json:"-"`
	OriginalPriceMinorUnits *int64  `gorm:"column:original_price_minor_units" json:"original_price_minor_units,omitempty"`
	DiscountMinorUnits      int64   `gorm:"column:discount_minor_units;not null;default:0" json:"discount_minor_units"`
	// The refund (migration 000068).
	RefundedAt        *time.Time     `gorm:"column:refunded_at" json:"refunded_at"`
	RefundReason      *string        `gorm:"column:refund_reason;type:varchar(32)" json:"refund_reason"`
	RefundNote        *string        `gorm:"column:refund_note;type:varchar(500)" json:"-"`
	RefundedBy        *string        `gorm:"column:refunded_by;type:varchar(36)" json:"-"`
	RefundProviderRef *string        `gorm:"column:refund_provider_ref;type:varchar(255)" json:"-"`
	User              User           `gorm:"foreignKey:UserID;references:ID" json:"-"`
	Program           Program        `gorm:"foreignKey:ProgramID;references:ID" json:"-"`
	CreatedAt         time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"-"`
}

func (p *Purchase) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}
