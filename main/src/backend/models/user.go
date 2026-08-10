package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User corresponds to the users table. Soft-deleted users are excluded from
// regular queries through GORM's DeletedAt handling.
type User struct {
	ID             string         `gorm:"type:char(36);primaryKey" json:"id"`
	Email          string         `gorm:"type:varchar(255);uniqueIndex:uq_users_email_deleted_at" json:"email"`
	PasswordHash   string         `gorm:"column:password_hash;type:varchar(255)" json:"-"`
	FirstName      string         `gorm:"column:first_name;type:varchar(100)" json:"first_name"`
	LastName       string         `gorm:"column:last_name;type:varchar(100)" json:"last_name"`
	CreatedAt      time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	SessionVersion int            `gorm:"column:session_version;not null;default:0" json:"-"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"deleted_at"`
}

func (u *User) BeforeCreate(_ *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	return nil
}
