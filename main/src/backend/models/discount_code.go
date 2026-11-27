package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Discount code scopes (migration 000069).
const (
	DiscountAppliesToAll           = "all"
	DiscountAppliesToPremiumLevel1 = "premium_level_1"
	DiscountAppliesToPremiumLevel2 = "premium_level_2"
	DiscountAppliesToTrainerPlans  = "trainer_plans"
)

// DiscountCode is a percentage off a paid plan, usable for a limited time
// (at most 30 days) and, optionally, a limited number of purchases.
type DiscountCode struct {
	ID         string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	Code       string     `gorm:"column:code;type:varchar(32);not null" json:"code"`
	PercentOff int        `gorm:"column:percent_off;not null" json:"percent_off"`
	AppliesTo  string     `gorm:"column:applies_to;type:varchar(20);not null" json:"applies_to"`
	StartsAt   time.Time  `gorm:"column:starts_at;not null" json:"starts_at"`
	ExpiresAt  time.Time  `gorm:"column:expires_at;not null" json:"expires_at"`
	MaxUses    *int       `gorm:"column:max_uses" json:"max_uses"`
	DisabledAt *time.Time `gorm:"column:disabled_at" json:"disabled_at"`
	CreatedBy  string     `gorm:"column:created_by;type:varchar(36);not null" json:"-"`
	CreatedAt  time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (DiscountCode) TableName() string { return "discount_codes" }

func (d *DiscountCode) BeforeCreate(_ *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	return nil
}
