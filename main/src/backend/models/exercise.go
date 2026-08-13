package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Exercise corresponds to the exercises table. It is a global platform exercise
// library entry: every exercise exists only once and is reused across programs
// and packs. The catalog is read-only through the public API in this
// foundation; it carries only descriptive metadata. Soft-deleted exercises are
// excluded from regular queries through GORM's DeletedAt handling.
type Exercise struct {
	ID            string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	Name          string         `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Description   string         `gorm:"column:description;type:text" json:"description"`
	TargetMuscles string         `gorm:"column:target_muscles;type:varchar(255)" json:"target_muscles"`
	Equipment     string         `gorm:"column:equipment;type:varchar(255)" json:"equipment"`
	Difficulty    string         `gorm:"column:difficulty;type:varchar(50)" json:"difficulty"`
	VideoURL      string         `gorm:"column:video_url;type:varchar(500)" json:"video_url"`
	ImageURL      string         `gorm:"column:image_url;type:varchar(500)" json:"image_url"`
	CreatedAt     time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"deleted_at"`
}

func (e *Exercise) BeforeCreate(_ *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	return nil
}
