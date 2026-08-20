package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PurchaseStatus values describe the lifecycle of a purchase record. This is
// a schema-only preparation: no purchase creation path exists yet.
const (
	PurchaseStatusPending   = "pending"
	PurchaseStatusCompleted = "completed"
	PurchaseStatusFailed    = "failed"
	PurchaseStatusRefunded  = "refunded"
)

// Purchase corresponds to the purchases table. It records a user's commercial
// intent to acquire a published program. The purchase carries a full commercial
// snapshot (price, currency, commission split) captured at creation time so the
// transaction remains historically accurate even if program pricing or
// commission rules change later. Soft-deleted purchases are excluded from
// regular queries through GORM's DeletedAt handling.
type Purchase struct {
	ID              string         `gorm:"type:char(36);primaryKey" json:"id"`
	UserID          string         `gorm:"column:user_id;type:varchar(36);not null" json:"user_id"`
	ProgramID       string         `gorm:"column:program_id;type:varchar(36);not null" json:"program_id"`
	PriceMinorUnits int64          `gorm:"column:price_minor_units;type:bigint;not null;default:0" json:"price_minor_units"`
	Currency        string         `gorm:"column:currency;type:varchar(3);not null;default:EUR" json:"currency"`
	CommissionBPS   uint32         `gorm:"column:commission_bps;type:int unsigned;not null;default:0" json:"commission_bps"`
	PlatformAmount  int64          `gorm:"column:platform_amount;type:bigint;not null;default:0" json:"platform_amount"`
	TrainerAmount   int64          `gorm:"column:trainer_amount;type:bigint;not null;default:0" json:"trainer_amount"`
	Status          string         `gorm:"column:status;type:varchar(20);not null;default:pending" json:"status"`
	User            User           `gorm:"foreignKey:UserID;references:ID" json:"-"`
	Program         Program        `gorm:"foreignKey:ProgramID;references:ID" json:"-"`
	CreatedAt       time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"-"`
}

func (p *Purchase) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}
