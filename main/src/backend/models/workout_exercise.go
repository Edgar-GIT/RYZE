package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkoutExercise corresponds to the workout_exercises table. It represents one
// exercise usage inside a program workout: "this exercise is part of this
// workout", positioned by position. A workout belongs to exactly one week of a
// trainer-owned program; ownership always derives from that chain and a
// WorkoutExercise never carries a trainer id. The active (position) slot is
// unique per workout through the stored active_workout_exercise column, so the
// same exercise may appear more than once in the same workout at different
// positions. Soft-deleted workout exercises are excluded from regular queries
// through GORM's DeletedAt handling.
type WorkoutExercise struct {
	ID               string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	ProgramWorkoutID string         `gorm:"column:program_workout_id;type:varchar(36);not null" json:"program_workout_id"`
	ExerciseID       string         `gorm:"column:exercise_id;type:varchar(36);not null" json:"exercise_id"`
	Position         int            `gorm:"column:position;not null" json:"position"`
	Exercise         *Exercise      `gorm:"foreignKey:ExerciseID;references:ID" json:"exercise"`
	CreatedAt        time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"deleted_at"`
}

func (w *WorkoutExercise) BeforeCreate(_ *gorm.DB) error {
	if w.ID == "" {
		w.ID = uuid.NewString()
	}
	return nil
}
