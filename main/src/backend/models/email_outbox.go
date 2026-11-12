package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Email outbox statuses.
const (
	EmailStatusPending = "pending"
	EmailStatusSending = "sending"
	EmailStatusSent    = "sent"
	EmailStatusFailed  = "failed"
	EmailStatusSkipped = "skipped"
	EmailStatusExpired = "expired"
)

// EmailOutbox is one transactional email recorded by a business event and
// delivered by the background worker. Payload carries only template values and
// is cleared once the email reaches a final state.
type EmailOutbox struct {
	ID            string          `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID        *string         `gorm:"column:user_id;type:varchar(36)" json:"user_id"`
	Recipient     string          `gorm:"column:recipient;type:varchar(255);not null" json:"-"`
	Template      string          `gorm:"column:template;type:varchar(48);not null" json:"template"`
	Locale        string          `gorm:"column:locale;type:varchar(5);not null" json:"locale"`
	Payload       json.RawMessage `gorm:"column:payload;type:jsonb;not null" json:"-"`
	DedupeKey     *string         `gorm:"column:dedupe_key;type:varchar(160)" json:"-"`
	Status        string          `gorm:"column:status;type:varchar(16);not null" json:"status"`
	Attempts      int             `gorm:"column:attempts;not null" json:"attempts"`
	NextAttemptAt time.Time       `gorm:"column:next_attempt_at;not null" json:"next_attempt_at"`
	ExpiresAt     *time.Time      `gorm:"column:expires_at" json:"expires_at"`
	LastError     *string         `gorm:"column:last_error;type:varchar(255)" json:"last_error"`
	SentAt        *time.Time      `gorm:"column:sent_at" json:"sent_at"`
	CreatedAt     time.Time       `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time       `gorm:"column:updated_at" json:"updated_at"`
}

func (EmailOutbox) TableName() string { return "email_outbox" }

func (e *EmailOutbox) BeforeCreate(_ *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	return nil
}
