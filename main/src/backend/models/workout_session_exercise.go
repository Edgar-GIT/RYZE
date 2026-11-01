package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkoutSessionExercise is the snapshot of one exercise prescription taken
// when a workout session starts. It carries the catalog identity for linking
// and display plus the name and instructions as they were prescribed at that
// moment, so later catalogue or trainer edits never rewrite history.
type WorkoutSessionExercise struct {
	ID               string              `gorm:"type:char(36);primaryKey" json:"id"`
	WorkoutSessionID string              `gorm:"column:workout_session_id;type:char(36);not null" json:"-"`
	ExerciseID       string              `gorm:"column:exercise_id;type:varchar(36);not null" json:"exercise_id"`
	Position         int                 `gorm:"column:position;not null" json:"position"`
	Name             string              `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Instructions     string              `gorm:"column:instructions;type:text" json:"instructions"`
	Notes            string              `gorm:"column:notes;type:text" json:"notes"`
	Sets             []WorkoutSessionSet `gorm:"foreignKey:WorkoutSessionExerciseID" json:"-"`
	CreatedAt        time.Time           `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt        time.Time           `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt        gorm.DeletedAt      `gorm:"column:deleted_at;type:datetime(6)" json:"-"`
}

func (w *WorkoutSessionExercise) BeforeCreate(_ *gorm.DB) error {
	if w.ID == "" {
		w.ID = uuid.NewString()
	}
	return nil
}
