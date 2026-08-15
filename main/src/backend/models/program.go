package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProgramType values describe the product category of a program.
const (
	ProgramTypeFree         = "free"
	ProgramTypePremium      = "premium"
	ProgramTypePersonalized = "personalized"
)

// ProgramStatus values describe the business state of a program. Publishing is
// only a state: it means the program is available for future consumption and
// carries no purchase, assignment or access semantics.
const (
	ProgramStatusDraft     = "draft"
	ProgramStatusPublished = "published"
)

// Program corresponds to the programs table. A program is a training or
// nutrition offer created by a trainer (trainer-owned) or, in the future, by
// the platform itself (platform-owned, trainer_id NULL). The client keeps being
// a regular User; no assignment or access relationship exists yet. Soft-deleted
// programs are excluded from regular queries through GORM's DeletedAt handling.
type Program struct {
	ID          string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	TrainerID   string         `gorm:"column:trainer_id;type:varchar(36)" json:"trainer_id"`
	Name        string         `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Description string         `gorm:"column:description;type:text" json:"description"`
	Type        string         `gorm:"column:type;type:varchar(20);not null" json:"type"`
	Status      string         `gorm:"column:status;type:varchar(20);not null" json:"status"`
	Weeks       []ProgramWeek  `gorm:"foreignKey:ProgramID" json:"-"`
	CreatedAt   time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"deleted_at"`
}

func (p *Program) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}
