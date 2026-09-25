package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TestSession corresponds to the test_sessions table. It records one active
// ADMIN_1 Test Mode session: the selected persona, the persona user the admin
// is browsing as, the admin identity that opened the session (so exiting always
// restores it) and a safely constrained return path. The raw bearer token is
// never stored — only its SHA-256 hash — so a captured database can never be
// replayed into an active session. Soft-deleted sessions are excluded from
// regular queries through GORM's DeletedAt handling.
type TestSession struct {
	ID            string         `gorm:"type:char(36);primaryKey" json:"id"`
	AdminIdentity string         `gorm:"column:admin_identity;type:varchar(16);not null" json:"-"`
	Persona       string         `gorm:"column:persona;type:varchar(16);not null" json:"persona"`
	PersonaUserID string         `gorm:"column:persona_user_id;type:varchar(36);not null" json:"-"`
	TokenHash     string         `gorm:"column:token_hash;type:char(64);not null;uniqueIndex" json:"-"`
	ReturnPath    string         `gorm:"column:return_path;type:varchar(512)" json:"-"`
	CreatedAt     time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"-"`
}

func (s *TestSession) BeforeCreate(_ *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	return nil
}
