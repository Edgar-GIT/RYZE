package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Support conversation values.
const (
	SupportStatusOpen   = "open"
	SupportStatusClosed = "closed"

	SupportRoleClient  = "client"
	SupportRoleTrainer = "trainer"

	SupportAuthorUser  = "user"
	SupportAuthorStaff = "staff"
)

// SupportConversation is one support thread between an account and RYZE.
type SupportConversation struct {
	ID               string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID           string     `gorm:"column:user_id;type:varchar(36);not null" json:"-"`
	RequesterRole    string     `gorm:"column:requester_role;type:varchar(16);not null" json:"requester_role"`
	Subject          string     `gorm:"column:subject;type:varchar(120);not null" json:"subject"`
	Status           string     `gorm:"column:status;type:varchar(16);not null" json:"status"`
	UserLastReadSeq  int64      `gorm:"column:user_last_read_seq;not null" json:"-"`
	StaffLastReadSeq int64      `gorm:"column:staff_last_read_seq;not null" json:"-"`
	LastMessageAt    time.Time  `gorm:"column:last_message_at;not null" json:"last_message_at"`
	ClosedAt         *time.Time `gorm:"column:closed_at" json:"closed_at"`
	ClosedBy         *string    `gorm:"column:closed_by;type:varchar(64)" json:"-"`
	CreatedAt        time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (SupportConversation) TableName() string { return "support_conversations" }

func (c *SupportConversation) BeforeCreate(_ *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	return nil
}

// SupportMessage is one append-only message. Seq is assigned by the database.
type SupportMessage struct {
	ID              string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	Seq             int64     `gorm:"column:seq;->" json:"seq"`
	ConversationID  string    `gorm:"column:conversation_id;type:varchar(36);not null" json:"conversation_id"`
	AuthorKind      string    `gorm:"column:author_kind;type:varchar(8);not null" json:"author"`
	AuthorUserID    *string   `gorm:"column:author_user_id;type:varchar(36)" json:"-"`
	AuthorAdminID   *string   `gorm:"column:author_admin_id;type:varchar(64)" json:"-"`
	Body            string    `gorm:"column:body;type:text;not null" json:"body"`
	ClientMessageID *string   `gorm:"column:client_message_id;type:varchar(64)" json:"client_message_id,omitempty"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
}

func (SupportMessage) TableName() string { return "support_messages" }

func (m *SupportMessage) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
