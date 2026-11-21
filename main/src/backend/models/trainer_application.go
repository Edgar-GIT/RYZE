package models

import (
	"encoding/json"
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
	ID     string `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID string `gorm:"column:user_id;type:varchar(36);not null" json:"user_id"`
	Status string `gorm:"column:status;type:varchar(20);not null" json:"status"`
	// The applicant's details (migration 000050); empty on applications made
	// before the complete form and on administrator promotions.
	FullName        *string         `gorm:"column:full_name;type:varchar(120)" json:"full_name"`
	BirthDate       *time.Time      `gorm:"column:birth_date;type:date" json:"birth_date"`
	City            *string         `gorm:"column:city;type:varchar(80)" json:"city"`
	Profession      *string         `gorm:"column:profession;type:varchar(120)" json:"profession"`
	YearsExperience *int            `gorm:"column:years_experience" json:"years_experience"`
	WorkHistory     json.RawMessage `gorm:"column:work_history;type:jsonb;not null;default:'[]'" json:"work_history"`
	Education       json.RawMessage `gorm:"column:education;type:jsonb;not null;default:'[]'" json:"education"`
	About           *string         `gorm:"column:about;type:text" json:"about"`
	User            User            `gorm:"foreignKey:UserID" json:"-"`
	CreatedAt       time.Time       `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt       gorm.DeletedAt  `gorm:"column:deleted_at;type:datetime(6)" json:"deleted_at"`
}

func (a *TrainerApplication) BeforeCreate(_ *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	return nil
}
