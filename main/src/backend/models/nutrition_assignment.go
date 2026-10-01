package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NutritionAssignmentStatus values describe the observable state machine of a
// server-owned nutrition plan. A purchase never implies a finished plan, so the
// client renders a plan only for a completed assignment. pending is the state a
// verified purchase creates; processing and failed are the transient and
// recoverable states of the deterministic generation step.
const (
	NutritionAssignmentStatusPending    = "pending"
	NutritionAssignmentStatusProcessing = "processing"
	NutritionAssignmentStatusCompleted  = "completed"
	NutritionAssignmentStatusFailed     = "failed"
)

// NutritionAssignment corresponds to the nutrition_assignments table. It is the
// server-owned nutrition configuration generated for a purchased Premium Level 1
// program from the owner's questionnaire.
//
// Generation is deterministic and idempotent, so a failed or interrupted run is
// recovered in place by retrying rather than by creating a second plan: at most
// one active assignment exists per (user, program). QuestionnaireVersion pins
// the intake revision the configuration was derived from, which lets the access
// surface detect an outdated plan after a questionnaire resubmission instead of
// silently serving a stale configuration.
//
// FailureReason is internal diagnostics for administrators. It is never exposed
// to the client and never logged together with questionnaire content.
type NutritionAssignment struct {
	ID                   string `gorm:"type:char(36);primaryKey" json:"id"`
	UserID               string `gorm:"column:user_id;type:varchar(36);not null" json:"user_id"`
	ProgramID            string `gorm:"column:program_id;type:varchar(36);not null" json:"program_id"`
	QuestionnaireID      string `gorm:"column:questionnaire_id;type:char(36);not null" json:"questionnaire_id"`
	QuestionnaireVersion int    `gorm:"column:questionnaire_version;not null" json:"questionnaire_version"`
	Status               string `gorm:"column:status;type:varchar(20);not null;default:pending" json:"status"`
	Version              int    `gorm:"column:version;not null;default:1" json:"version"`
	// Configuration is the generated plan snapshot. It is only populated for a
	// completed assignment; the concrete shape is owned by the assignment
	// service.
	Configuration json.RawMessage `gorm:"column:configuration;type:json" json:"configuration"`
	FailureReason *string         `gorm:"column:failure_reason;type:varchar(255)" json:"-"`
	GeneratedAt   *time.Time      `gorm:"column:generated_at;type:datetime(6)" json:"generated_at"`
	CreatedAt     time.Time       `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt     time.Time       `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt     gorm.DeletedAt  `gorm:"column:deleted_at;type:datetime(6)" json:"-"`
}

func (a *NutritionAssignment) BeforeCreate(_ *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if a.Version <= 0 {
		a.Version = 1
	}
	if a.Status == "" {
		a.Status = NutritionAssignmentStatusPending
	}
	return nil
}
