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

// Purchase corresponds to the purchases table. It records a user's purchase
// of a published program. Currently no creation path exists; this model is
// prepared for future commerce integration. Soft-deleted purchases are excluded
// from regular queries through GORM's DeletedAt handling.
type Purchase struct {
	ID        string         `gorm:"type:char(36);primaryKey" json:"id"`
	UserID    string         `gorm:"column:user_id;type:varchar(36);not null" json:"user_id"`
	ProgramID string         `gorm:"column:program_id;type:varchar(36);not null" json:"program_id"`
	Status    string         `gorm:"column:status;type:varchar(20);not null;default:completed" json:"status"`
	User      User           `gorm:"foreignKey:UserID;references:ID" json:"-"`
	Program   Program        `gorm:"foreignKey:ProgramID;references:ID" json:"-"`
	CreatedAt time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"-"`
}

func (p *Purchase) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}
