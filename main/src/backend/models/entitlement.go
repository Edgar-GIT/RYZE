package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entitlement corresponds to the entitlements table. It records that a user
// has a purchase-backed right to access a published program. Currently no
// creation path exists; this model is prepared for future commerce integration.
// The active_entitlement generated column enforces at most one active
// entitlement per (user, program) pair. Soft-deleted entitlements are excluded
// from regular queries through GORM's DeletedAt handling.
type Entitlement struct {
	ID        string         `gorm:"type:char(36);primaryKey" json:"id"`
	UserID    string         `gorm:"column:user_id;type:varchar(36);not null" json:"user_id"`
	ProgramID string         `gorm:"column:program_id;type:varchar(36);not null" json:"program_id"`
	User      User           `gorm:"foreignKey:UserID;references:ID" json:"-"`
	Program   Program        `gorm:"foreignKey:ProgramID;references:ID" json:"-"`
	CreatedAt time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"-"`
}

func (e *Entitlement) BeforeCreate(_ *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	return nil
}
