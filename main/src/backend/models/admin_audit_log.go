package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Admin audit outcomes.
const (
	AdminAuditOutcomeSuccess = "success"
	AdminAuditOutcomeDenied  = "denied"
	AdminAuditOutcomeFailed  = "failed"
)

// AdminAuditLog is one immutable administrative audit record (see migration
// 000041). It describes an action, never its content: no request body, note,
// password or access code is ever stored.
type AdminAuditLog struct {
	ID         string    `gorm:"type:varchar(36);primaryKey"`
	AdminID    string    `gorm:"column:admin_id;type:varchar(16);not null"`
	Action     string    `gorm:"column:action;type:varchar(64);not null"`
	Method     string    `gorm:"column:method;type:varchar(8);not null"`
	Route      string    `gorm:"column:route;type:varchar(160);not null"`
	TargetID   *string   `gorm:"column:target_id;type:varchar(64)"`
	StatusCode int       `gorm:"column:status_code;not null"`
	Outcome    string    `gorm:"column:outcome;type:varchar(12);not null"`
	ClientIP   *string   `gorm:"column:client_ip;type:varchar(45)"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (AdminAuditLog) TableName() string { return "admin_audit_logs" }

func (l *AdminAuditLog) BeforeCreate(_ *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.NewString()
	}
	return nil
}

// ProgramViewStat is the view counter of one program (see migration 000042).
type ProgramViewStat struct {
	ProgramID    string     `gorm:"column:program_id;type:varchar(36);primaryKey"`
	ViewCount    int64      `gorm:"column:view_count;not null"`
	LastViewedAt *time.Time `gorm:"column:last_viewed_at"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"`
}

func (ProgramViewStat) TableName() string { return "program_view_stats" }
