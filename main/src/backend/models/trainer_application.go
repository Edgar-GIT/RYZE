package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ApplicationStatus values describe the lifecycle of a trainer application.
const (
	ApplicationStatusPending  = "PENDING"
	ApplicationStatusApproved = "APPROVED"
	ApplicationStatusRejected = "REJECTED"
)

// TrainerApplication corresponds to the trainer_applications table. It records
// a user's request to become a trainer and its review outcome. A user may hold
// at most one active (PENDING or APPROVED) application; REJECTED applications
// stay in history so the user can apply again. Soft-deleted applications are
// excluded from regular queries through GORM's DeletedAt handling.
type TrainerApplication struct {
	ID        string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID    string         `gorm:"column:user_id;type:varchar(36);not null" json:"user_id"`
	Status    string         `gorm:"column:status;type:varchar(20);not null" json:"status"`
	User      User           `gorm:"foreignKey:UserID" json:"-"`
	CreatedAt time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"deleted_at"`
}

func (a *TrainerApplication) BeforeCreate(_ *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	return nil
}
