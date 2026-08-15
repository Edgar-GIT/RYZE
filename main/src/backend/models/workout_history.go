package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkoutHistory corresponds to the workout_history table. It is the append-only
// execution log of one workout of the user's currently assigned program: each
// row records that the authenticated user completed a specific workout
// (program_workout) at a specific moment. A workout may be completed more than
// once, so repeated executions are preserved as separate rows and no unique
// constraint exists on (user_id, program_workout_id). The user id is always set
// from the authentication context, never from the client. Soft-deleted rows are
// excluded from regular queries through GORM's DeletedAt handling.
type WorkoutHistory struct {
	ID               string         `gorm:"type:char(36);primaryKey" json:"id"`
	UserID           string         `gorm:"column:user_id;type:varchar(36);not null" json:"-"`
	ProgramWorkoutID string         `gorm:"column:program_workout_id;type:varchar(36);not null" json:"program_workout_id"`
	CompletedAt      time.Time      `gorm:"column:completed_at;type:datetime(6);not null" json:"completed_at"`
	CreatedAt        time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"-"`
}

func (w *WorkoutHistory) BeforeCreate(_ *gorm.DB) error {
	if w.ID == "" {
		w.ID = uuid.NewString()
	}
	if w.CompletedAt.IsZero() {
		w.CompletedAt = time.Now().UTC()
	}
	return nil
}

func (WorkoutHistory) TableName() string {
	return "workout_history"
}
