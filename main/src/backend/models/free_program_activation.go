package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FreeProgramActivation corresponds to the free_program_activations table. It
// records that a client activated a platform-owned Free program without a
// purchase. A Free activation is deliberately not an entitlement and not an
// assignment: it grants only the programme content, never trainer ownership or
// nutrition. At most one activation is active per user, enforced by the unique
// index on active_free_activation. Soft-deleted rows are excluded from regular
// queries through GORM's DeletedAt handling.
type FreeProgramActivation struct {
	ID          string         `gorm:"type:char(36);primaryKey" json:"id"`
	UserID      string         `gorm:"column:user_id;type:varchar(36);not null" json:"user_id"`
	ProgramID   string         `gorm:"column:program_id;type:varchar(36);not null" json:"program_id"`
	ActivatedAt time.Time      `gorm:"column:activated_at;type:datetime(6);not null" json:"activated_at"`
	ExpiresAt   *time.Time     `gorm:"column:expires_at;type:datetime(6)" json:"expires_at"`
	Program     Program        `gorm:"foreignKey:ProgramID;references:ID" json:"-"`
	CreatedAt   time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"-"`
}

func (f *FreeProgramActivation) BeforeCreate(_ *gorm.DB) error {
	if f.ID == "" {
		f.ID = uuid.NewString()
	}
	if f.ActivatedAt.IsZero() {
		f.ActivatedAt = time.Now().UTC()
	}
	return nil
}
