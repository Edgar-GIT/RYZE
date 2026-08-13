package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProgramAssignment corresponds to the program_assignments table. It links one
// trainer-owned program to one trainer-managed client (a regular User); the
// client keeps being a regular User and no separate identity is ever created.
// A trainer can have at most one active assigned program per client; the rule
// is enforced by the unique index on the active_assignment generated column.
// The trainer id is stored explicitly so every query can be scoped by it, and
// is always set from the authenticated trainer context, never from the client.
// Soft-deleted assignments are excluded from regular queries through GORM's
// DeletedAt handling.
type ProgramAssignment struct {
	ID        string         `gorm:"type:char(36);primaryKey" json:"id"`
	TrainerID string         `gorm:"column:trainer_id;type:varchar(36);not null" json:"trainer_id"`
	ProgramID string         `gorm:"column:program_id;type:varchar(36);not null" json:"program_id"`
	UserID    string         `gorm:"column:user_id;type:varchar(36);not null" json:"user_id"`
	Program   Program        `gorm:"foreignKey:ProgramID;references:ID" json:"-"`
	CreatedAt time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"-"`
}

func (p *ProgramAssignment) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}
