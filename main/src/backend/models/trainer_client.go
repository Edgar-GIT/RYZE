package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TrainerClient corresponds to the trainer_clients table. It is the persistent
// relationship between a trainer and a user (the client): a trainer manages
// many clients and a user may be a client of many trainers. The relationship
// has its own lifecycle — an active relation is unique per (trainer_id,
// user_id), removing it soft-deletes the row and reactivating it restores the
// same row. The client keeps being a regular User; no second identity is ever
// created. Soft-deleted relationships are excluded from regular queries through
// GORM's DeletedAt handling.
type TrainerClient struct {
	ID        string         `gorm:"type:char(36);primaryKey" json:"id"`
	TrainerID string         `gorm:"column:trainer_id;type:varchar(36);not null" json:"trainer_id"`
	UserID    string         `gorm:"column:user_id;type:varchar(36);not null" json:"user_id"`
	User      User           `gorm:"foreignKey:UserID" json:"-"`
	CreatedAt time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"deleted_at"`
}

func (t *TrainerClient) BeforeCreate(_ *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	return nil
}
