package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IdentityProviderGoogle is the only external sign-in provider.
const IdentityProviderGoogle = "google"

// UserIdentity links an external sign-in account (provider + stable subject)
// to a RYZE account.
type UserIdentity struct {
	ID         string     `gorm:"type:varchar(36);primaryKey" json:"-"`
	UserID     string     `gorm:"column:user_id;type:varchar(36);not null" json:"-"`
	Provider   string     `gorm:"column:provider;type:varchar(16);not null" json:"provider"`
	Subject    string     `gorm:"column:subject;type:varchar(255);not null" json:"-"`
	Email      string     `gorm:"column:email;type:varchar(255);not null" json:"email"`
	CreatedAt  time.Time  `gorm:"column:created_at" json:"linked_at"`
	LastUsedAt *time.Time `gorm:"column:last_used_at" json:"last_used_at"`
}

func (UserIdentity) TableName() string { return "user_identities" }

func (i *UserIdentity) BeforeCreate(_ *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.NewString()
	}
	return nil
}
