package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	WorkoutSessionStatusInProgress = "in_progress"
	WorkoutSessionStatusPaused     = "paused"
	WorkoutSessionStatusCompleted  = "completed"
	WorkoutSessionStatusAbandoned  = "abandoned"

	WorkoutSetStatusPending   = "pending"
	WorkoutSetStatusCompleted = "completed"
	WorkoutSetStatusSkipped   = "skipped"
)

// WorkoutSession corresponds to the workout_sessions table. It is one execution
// of one prescribed workout. The prescription is snapshotted into the session
// exercise and set rows, so later trainer edits never rewrite history. The
// unique index on active_session_user enforces at most one in-progress or
// paused session per user. Soft-deleted sessions are excluded from regular
// queries through GORM's DeletedAt handling.
type WorkoutSession struct {
	ID                 string                   `gorm:"type:char(36);primaryKey" json:"id"`
	UserID             string                   `gorm:"column:user_id;type:varchar(36);not null" json:"-"`
	ProgramID          string                   `gorm:"column:program_id;type:varchar(36);not null" json:"program_id"`
	ProgramWorkoutID   string                   `gorm:"column:program_workout_id;type:varchar(36);not null" json:"program_workout_id"`
	ProgramName        string                   `gorm:"column:program_name;type:varchar(255);not null" json:"program_name"`
	WeekNumber         int                      `gorm:"column:week_number;not null" json:"week_number"`
	WorkoutPosition    int                      `gorm:"column:workout_position;not null" json:"workout_position"`
	Status             string                   `gorm:"column:status;type:varchar(20);not null" json:"status"`
	StartedAt          time.Time                `gorm:"column:started_at;type:datetime(6);not null" json:"started_at"`
	LastResumedAt      *time.Time               `gorm:"column:last_resumed_at;type:datetime(6)" json:"-"`
	PausedAt           *time.Time               `gorm:"column:paused_at;type:datetime(6)" json:"paused_at"`
	CompletedAt        *time.Time               `gorm:"column:completed_at;type:datetime(6)" json:"completed_at"`
	AbandonedAt        *time.Time               `gorm:"column:abandoned_at;type:datetime(6)" json:"abandoned_at"`
	AccumulatedSeconds int                      `gorm:"column:accumulated_seconds;not null;default:0" json:"elapsed_seconds"`
	Exercises          []WorkoutSessionExercise `gorm:"foreignKey:WorkoutSessionID" json:"-"`
	CreatedAt          time.Time                `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt          time.Time                `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt          gorm.DeletedAt           `gorm:"column:deleted_at;type:datetime(6)" json:"-"`
}

func (w *WorkoutSession) BeforeCreate(_ *gorm.DB) error {
	if w.ID == "" {
		w.ID = uuid.NewString()
	}
	return nil
}
