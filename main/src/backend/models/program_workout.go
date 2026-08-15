package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProgramWorkout corresponds to the program_workouts table. It is one workout
// slot inside a program week, ordered by position. A workout belongs to exactly
// one week and the active (position) slot is unique per week through the stored
// active_workout column. A workout carries no name or content yet: in this
// foundation the position is its identity and future fields (exercise
// assignments, sets, reps, rest) extend this entity without structural change.
// Soft-deleted workouts are excluded from regular queries through GORM's
// DeletedAt handling.
type ProgramWorkout struct {
	ID            string            `gorm:"type:varchar(36);primaryKey" json:"id"`
	ProgramWeekID string            `gorm:"column:program_week_id;type:varchar(36);not null" json:"program_week_id"`
	Position      int               `gorm:"column:position;not null" json:"position"`
	Exercises     []WorkoutExercise `gorm:"foreignKey:ProgramWorkoutID" json:"-"`
	CreatedAt     time.Time         `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt     time.Time         `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt     gorm.DeletedAt    `gorm:"column:deleted_at;type:datetime(6)" json:"deleted_at"`
}

func (w *ProgramWorkout) BeforeCreate(_ *gorm.DB) error {
	if w.ID == "" {
		w.ID = uuid.NewString()
	}
	return nil
}
