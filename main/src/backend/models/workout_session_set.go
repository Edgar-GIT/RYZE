package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkoutSessionSet is the snapshot of one prescribed set plus the actual
// performance recorded during the session. Prescribed and actual values are
// deliberately separate: a client may complete fewer reps or use a different
// weight without losing the original prescription.
type WorkoutSessionSet struct {
	ID                       string         `gorm:"type:char(36);primaryKey" json:"id"`
	WorkoutSessionExerciseID string         `gorm:"column:workout_session_exercise_id;type:char(36);not null" json:"-"`
	SetNumber                int            `gorm:"column:set_number;not null" json:"set_number"`
	SetType                  string         `gorm:"column:set_type;type:varchar(20);not null" json:"set_type"`
	PrescribedReps           *int           `gorm:"column:prescribed_reps" json:"prescribed_reps"`
	PrescribedWeightKg       *float64       `gorm:"column:prescribed_weight_kg;type:decimal(6,2)" json:"prescribed_weight_kg"`
	PrescribedRIR            *int           `gorm:"column:prescribed_rir" json:"prescribed_rir"`
	PrescribedRPE            *float64       `gorm:"column:prescribed_rpe;type:decimal(3,1)" json:"prescribed_rpe"`
	PrescribedRestSeconds    *int           `gorm:"column:prescribed_rest_seconds" json:"prescribed_rest_seconds"`
	PrescribedTempo          string         `gorm:"column:prescribed_tempo;type:varchar(20)" json:"prescribed_tempo"`
	ActualReps               *int           `gorm:"column:actual_reps" json:"actual_reps"`
	ActualWeightKg           *float64       `gorm:"column:actual_weight_kg;type:decimal(6,2)" json:"actual_weight_kg"`
	ActualRIR                *int           `gorm:"column:actual_rir" json:"actual_rir"`
	ActualRPE                *float64       `gorm:"column:actual_rpe;type:decimal(3,1)" json:"actual_rpe"`
	Status                   string         `gorm:"column:status;type:varchar(20);not null" json:"status"`
	CompletedAt              *time.Time     `gorm:"column:completed_at;type:datetime(6)" json:"completed_at"`
	CreatedAt                time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt                time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt                gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"-"`
}

func (w *WorkoutSessionSet) BeforeCreate(_ *gorm.DB) error {
	if w.ID == "" {
		w.ID = uuid.NewString()
	}
	return nil
}
