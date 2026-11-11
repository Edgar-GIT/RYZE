package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MemberFeedback status values. Every submission starts pending; only approved
// feedback is ever shown publicly.
const (
	MemberFeedbackStatusPending  = "pending"
	MemberFeedbackStatusApproved = "approved"
	MemberFeedbackStatusRejected = "rejected"
)

// MemberFeedback corresponds to the member_feedback table: a star rating and a
// short message left by a member who holds (or held) a plan. DisplayName is a
// snapshot of the public name ("First L.") taken at submission, so the public
// list never needs to read account data. A member may hold at most one pending
// submission, enforced by the unique index on pending_user_id. Soft-deleted
// rows are excluded from regular queries through GORM's DeletedAt handling.
type MemberFeedback struct {
	ID          string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID      string         `gorm:"column:user_id;type:varchar(36);not null" json:"user_id"`
	Rating      int            `gorm:"column:rating;type:tinyint unsigned;not null" json:"rating"`
	Message     string         `gorm:"column:message;type:text;not null" json:"message"`
	DisplayName string         `gorm:"column:display_name;type:varchar(120);not null" json:"display_name"`
	Status      string         `gorm:"column:status;type:varchar(16);not null" json:"status"`
	ReviewedBy  *string        `gorm:"column:reviewed_by;type:varchar(40)" json:"reviewed_by"`
	ReviewedAt  *time.Time     `gorm:"column:reviewed_at;type:datetime(6)" json:"reviewed_at"`
	CreatedAt   time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"-"`
}

// TableName keeps the singular table name used by the migration.
func (MemberFeedback) TableName() string {
	return "member_feedback"
}

func (f *MemberFeedback) BeforeCreate(_ *gorm.DB) error {
	if f.ID == "" {
		f.ID = uuid.NewString()
	}
	return nil
}
