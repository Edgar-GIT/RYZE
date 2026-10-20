package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Publication lifecycle of a catalogue entry. Only published entries are
// served by the public read API. Draft and archived entries stay in the
// catalogue for administrators and keep existing program references intact.
const (
	ExerciseStatusDraft     = "draft"
	ExerciseStatusPublished = "published"
	ExerciseStatusArchived  = "archived"
)

// Exercise corresponds to the exercises table. It is one entry of the global
// exercise library: every exercise exists only once and is reused across
// programs. The catalogue carries descriptive metadata, optional source
// identity (Source+SourceID link an imported record so re-imports are
// idempotent), media references and a publication lifecycle. Status is
// 'published' by default so pre-existing rows and program references keep their
// visibility; 'draft' and 'archived' rows are never served by the public API.
// OwnerTrainerID is reserved for future trainer-owned content and is always
// NULL for platform-owned exercises in this version. Soft-deleted exercises are
// excluded from regular queries through GORM's DeletedAt handling.
type Exercise struct {
	ID                    string          `gorm:"type:varchar(36);primaryKey" json:"id"`
	Name                  string          `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Source                string          `gorm:"column:source;type:varchar(40)" json:"source"`
	SourceID              *string         `gorm:"column:source_id;type:varchar(40)" json:"source_id"`
	BodyPart              *string         `gorm:"column:body_part;type:varchar(40)" json:"body_part"`
	Description           string          `gorm:"column:description;type:text" json:"description"`
	Instructions          string          `gorm:"column:instructions;type:text" json:"instructions"`
	InstructionSteps      json.RawMessage `gorm:"column:instruction_steps;type:json" json:"instruction_steps"`
	TargetMuscles         string          `gorm:"column:target_muscles;type:varchar(255)" json:"target_muscles"`
	PrimaryMuscleGroup    string          `gorm:"column:primary_muscle_group;type:varchar(80)" json:"primary_muscle_group"`
	SecondaryMuscleGroups string          `gorm:"column:secondary_muscle_groups;type:varchar(255)" json:"secondary_muscle_groups"`
	Equipment             string          `gorm:"column:equipment;type:varchar(255)" json:"equipment"`
	Difficulty            string          `gorm:"column:difficulty;type:varchar(50)" json:"difficulty"`
	MovementCategory      string          `gorm:"column:movement_category;type:varchar(60)" json:"movement_category"`
	VideoURL              string          `gorm:"column:video_url;type:varchar(500)" json:"video_url"`
	ImageURL              string          `gorm:"column:image_url;type:varchar(500)" json:"image_url"`
	AnimationURL          string          `gorm:"column:animation_url;type:varchar(500)" json:"animation_url"`
	ThreeDKey             *string         `gorm:"column:three_d_key;type:varchar(64)" json:"three_d_key"`
	SourceMediaID         string          `gorm:"column:source_media_id;type:varchar(40)" json:"source_media_id"`
	MediaAttribution      string          `gorm:"column:media_attribution;type:varchar(255)" json:"media_attribution"`
	Status                string          `gorm:"column:status;type:varchar(20);not null;default:published" json:"status"`
	OwnerTrainerID        *string         `gorm:"column:owner_trainer_id;type:varchar(36)" json:"owner_trainer_id"`
	CreatedAt             time.Time       `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt             time.Time       `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt             gorm.DeletedAt  `gorm:"column:deleted_at;type:datetime(6)" json:"deleted_at"`
}

func (e *Exercise) BeforeCreate(_ *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	return nil
}
