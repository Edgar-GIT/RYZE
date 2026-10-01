package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NutritionQuestionnaire corresponds to the nutrition_questionnaires table. It
// stores the validated, normalized intake of a client for a Premium Level 1
// program. Exactly one active questionnaire exists per (user, program): a
// resubmission rewrites the row and increments Version, so the nutrition
// assignment can always record which intake revision it was derived from.
//
// Answers hold health and dietary information and are therefore sensitive:
// they are never logged, never returned in list or summary projections, and are
// only ever read by the authenticated owner and the server-side assignment
// generator. Soft-deleted questionnaires are excluded from regular queries
// through GORM's DeletedAt handling.
type NutritionQuestionnaire struct {
	ID        string `gorm:"type:char(36);primaryKey" json:"id"`
	UserID    string `gorm:"column:user_id;type:varchar(36);not null" json:"user_id"`
	ProgramID string `gorm:"column:program_id;type:varchar(36);not null" json:"program_id"`
	Version   int    `gorm:"column:version;not null;default:1" json:"version"`
	// Answers holds the normalized answer document. The concrete shape is owned
	// by the questionnaire service; the model only guarantees it round-trips
	// through JSON unchanged.
	Answers     json.RawMessage `gorm:"column:answers;type:json;not null" json:"answers"`
	SubmittedAt time.Time       `gorm:"column:submitted_at;type:datetime(6);not null" json:"submitted_at"`
	CreatedAt   time.Time       `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt   time.Time       `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt   gorm.DeletedAt  `gorm:"column:deleted_at;type:datetime(6)" json:"-"`
}

func (q *NutritionQuestionnaire) BeforeCreate(_ *gorm.DB) error {
	if q.ID == "" {
		q.ID = uuid.NewString()
	}
	if q.Version <= 0 {
		q.Version = 1
	}
	return nil
}
