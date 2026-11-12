package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User token purposes.
const (
	UserTokenPurposePasswordReset     = "password_reset"
	UserTokenPurposeEmailVerification = "email_verification"
)

// UserToken is a single-use account token (password reset or email
// verification). Only the SHA-256 of the random token is stored; the raw value
// exists only in the email link.
type UserToken struct {
	ID        string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID    string     `gorm:"column:user_id;type:varchar(36);not null" json:"user_id"`
	Purpose   string     `gorm:"column:purpose;type:varchar(32);not null" json:"purpose"`
	TokenHash string     `gorm:"column:token_hash;type:varchar(64);not null" json:"-"`
	ExpiresAt time.Time  `gorm:"column:expires_at;not null" json:"expires_at"`
	UsedAt    *time.Time `gorm:"column:used_at" json:"used_at"`
	CreatedAt time.Time  `gorm:"column:created_at" json:"created_at"`
}

func (UserToken) TableName() string { return "user_tokens" }

func (t *UserToken) BeforeCreate(_ *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	return nil
}
