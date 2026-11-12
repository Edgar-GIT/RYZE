package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Notification kinds. The web app turns kind + params into text in the
// reader's language.
const (
	NotificationAccessGranted       = "access_granted"
	NotificationPL2IntakeReceived   = "pl2_intake_received"
	NotificationPL2TrainerAssigned  = "pl2_trainer_assigned"
	NotificationPL2RequestAssigned  = "pl2_request_assigned"
	NotificationPL2Delivered        = "pl2_delivered"
	NotificationSupportReply        = "support_reply"
	NotificationSupportNewMessage   = "support_new_message"
	NotificationLifetime            = 24 * time.Hour
	NotificationTombstoneRetention  = 30 * 24 * time.Hour
	NotificationMaxParamValueLength = 160
)

// Notification is a temporary in-app message for one account (see migration
// 000039 for the retention policy).
type Notification struct {
	ID        string          `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID    string          `gorm:"column:user_id;type:varchar(36);not null" json:"-"`
	Kind      string          `gorm:"column:kind;type:varchar(48);not null" json:"kind"`
	Params    json.RawMessage `gorm:"column:params;type:jsonb;not null" json:"params"`
	Link      *string         `gorm:"column:link;type:varchar(255)" json:"link"`
	DedupeKey *string         `gorm:"column:dedupe_key;type:varchar(160)" json:"-"`
	ReadAt    *time.Time      `gorm:"column:read_at" json:"read_at"`
	CreatedAt time.Time       `gorm:"column:created_at" json:"created_at"`
	ExpiresAt time.Time       `gorm:"column:expires_at;not null" json:"expires_at"`
}

func (Notification) TableName() string { return "notifications" }

func (n *Notification) BeforeCreate(_ *gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	return nil
}
