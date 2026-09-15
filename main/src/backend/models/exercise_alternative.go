package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ExerciseAlternative corresponds to the exercise_alternatives table. It is one
// directed link between two entries of the global exercise library: the
// exercise "exercise_id" can be swapped for or compared with
// "alternative_exercise_id". Links are platform-owned and never cascade in
// either direction, so soft-deleting an exercise keeps links of other entries
// readable through the stored foreign keys.
type ExerciseAlternative struct {
	ID                    string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	ExerciseID            string         `gorm:"column:exercise_id;type:varchar(36);not null" json:"exercise_id"`
	AlternativeExerciseID string         `gorm:"column:alternative_exercise_id;type:varchar(36);not null" json:"alternative_exercise_id"`
	CreatedAt             time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt             time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"deleted_at"`
}

func (a *ExerciseAlternative) BeforeCreate(_ *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	return nil
}