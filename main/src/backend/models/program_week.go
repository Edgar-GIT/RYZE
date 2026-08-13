package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProgramWeek corresponds to the program_weeks table. It is a week slot inside
// a trainer-owned program, ordered by week_number. Weeks are the structural
// container for workouts; a week belongs to exactly one program and the active
// (week_number) slot is unique per program through the stored active_week
// column. Soft-deleted weeks are excluded from regular queries through GORM's
// DeletedAt handling.
type ProgramWeek struct {
	ID         string           `gorm:"type:varchar(36);primaryKey" json:"id"`
	ProgramID  string           `gorm:"column:program_id;type:varchar(36);not null" json:"program_id"`
	WeekNumber int              `gorm:"column:week_number;not null" json:"week_number"`
	Workouts   []ProgramWorkout `gorm:"foreignKey:ProgramWeekID" json:"-"`
	CreatedAt  time.Time        `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt  time.Time        `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt  gorm.DeletedAt   `gorm:"column:deleted_at;type:datetime(6)" json:"deleted_at"`
}

func (w *ProgramWeek) BeforeCreate(_ *gorm.DB) error {
	if w.ID == "" {
		w.ID = uuid.NewString()
	}
	return nil
}
