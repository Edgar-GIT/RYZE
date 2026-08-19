package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CommissionRule corresponds to the commission_rules table. It stores
// trainer-specific commission overrides expressed as basis points (1 bps =
// 0.01%). The platform always retains the commission; the trainer receives the
// remainder. Soft-deleted rules are excluded from regular queries through
// GORM's DeletedAt handling. The active_commission_rule generated column
// enforces at most one active override per trainer.
type CommissionRule struct {
	ID                   string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	TrainerID            string         `gorm:"column:trainer_id;type:varchar(36);not null" json:"trainer_id"`
	CommissionBPS        uint32         `gorm:"column:commission_bps;type:int unsigned;not null" json:"commission_bps"`
	ValidFrom            time.Time      `gorm:"column:valid_from;type:datetime(6);not null;default:CURRENT_TIMESTAMP(6)" json:"valid_from"`
	ValidUntil           *time.Time     `gorm:"column:valid_until;type:datetime(6);default:null" json:"valid_until"`
	CreatedAt            time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"-"`
	ActiveCommissionRule *string        `gorm:"column:active_commission_rule;type:varchar(39);not null;uniqueIndex:uk_active_commission_rule" json:"-"`
}

func (cr *CommissionRule) BeforeCreate(_ *gorm.DB) error {
	if cr.ID == "" {
		cr.ID = uuid.NewString()
	}
	return nil
}
