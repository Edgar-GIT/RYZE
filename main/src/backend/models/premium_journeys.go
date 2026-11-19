package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PremiumLevel1Match records one Premium Level 1 pick: the objective and body
// data the client gave and the curated package RYZE chose for them at random.
// The body data belongs to the client and is never shown publicly.
type PremiumLevel1Match struct {
	ID        string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID    string    `gorm:"column:user_id;type:varchar(36);not null" json:"-"`
	ProgramID string    `gorm:"column:program_id;type:varchar(36);not null" json:"program_id"`
	Objective string    `gorm:"column:objective;type:varchar(32);not null" json:"objective"`
	WeightKg  float64   `gorm:"column:weight_kg;type:numeric(5,1);not null" json:"weight_kg"`
	HeightCm  float64   `gorm:"column:height_cm;type:numeric(5,1);not null" json:"height_cm"`
	Age       int       `gorm:"column:age;not null" json:"age"`
	Sex       *string   `gorm:"column:sex;type:varchar(10)" json:"sex"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (PremiumLevel1Match) TableName() string { return "premium_level_1_matches" }

func (m *PremiumLevel1Match) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
