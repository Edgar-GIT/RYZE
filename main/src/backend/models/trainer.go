package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Trainer corresponds to the trainers table. A trainer is the professional
// profile linked to exactly one user account: the User owns authentication,
// contact and profile data while the Trainer owns the trainer-specific
// identity. Soft-deleted trainers are excluded from regular queries through
// GORM's DeletedAt handling.
type Trainer struct {
	ID        string         `gorm:"type:char(36);primaryKey" json:"id"`
	UserID    string         `gorm:"column:user_id;type:varchar(36);not null" json:"user_id"`
	User      User           `gorm:"foreignKey:UserID" json:"-"`
	CreatedAt time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"deleted_at"`
}

func (t *Trainer) BeforeCreate(_ *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	return nil
}
