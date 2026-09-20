package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkoutExercise corresponds to the workout_exercises table. It is one
// exercise usage inside a program workout carrying the per-assignment
// prescription sets. A workout exercise belongs to exactly one workout; the
// active (position) slot is unique per workout through the stored
// active_workout_exercise column. Soft-deleted sets are excluded from regular
// queries through GORM's DeletedAt handling.
type WorkoutExerciseSet struct {
	ID                string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	WorkoutExerciseID string         `gorm:"column:workout_exercise_id;type:varchar(36);not null" json:"workout_exercise_id"`
	SetNumber         int            `gorm:"column:set_number;not null" json:"set_number"`
	Reps              *int           `gorm:"column:reps" json:"reps"`
	WeightKg          *float64       `gorm:"column:weight_kg;type:decimal(6,2)" json:"weight_kg"`
	RIR               *int           `gorm:"column:rir" json:"rir"`
	RPE               *float64       `gorm:"column:rpe;type:decimal(3,1)" json:"rpe"`
	RestSeconds       *int           `gorm:"column:rest_seconds" json:"rest_seconds"`
	Tempo             string         `gorm:"column:tempo;type:varchar(20)" json:"tempo"`
	SetType           string         `gorm:"column:set_type;type:varchar(20)" json:"set_type"`
	CreatedAt         time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"deleted_at"`
}

func (w *WorkoutExerciseSet) BeforeCreate(_ *gorm.DB) error {
	if w.ID == "" {
		w.ID = uuid.NewString()
	}
	return nil
}
