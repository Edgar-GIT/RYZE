package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User corresponds to the users table. Soft-deleted users are excluded from
// regular queries through GORM's DeletedAt handling.
type User struct {
	ID             string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	Email          string    `gorm:"type:varchar(255);uniqueIndex:uq_users_email_deleted_at" json:"email"`
	PasswordHash   string    `gorm:"column:password_hash;type:varchar(255)" json:"-"`
	FirstName      string    `gorm:"column:first_name;type:varchar(100)" json:"first_name"`
	LastName       string    `gorm:"column:last_name;type:varchar(100)" json:"last_name"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`
	SessionVersion int       `gorm:"column:session_version;not null;default:0" json:"-"`
	// TestPersona marks the account as a Test Mode persona owned by the
	// platform. It is set exclusively by the Test Mode provisioning service and
	// is never accepted from a client, registration or administrator payload.
	TestPersona *string `gorm:"column:test_persona;type:varchar(16)" json:"-"`
	// EmailVerifiedAt is set when the owner follows an email verification
	// link. It is informational and never gates sign-in.
	EmailVerifiedAt *time.Time `gorm:"column:email_verified_at" json:"email_verified_at"`
	// PreferredLanguage is the language RYZE writes emails in ("en" or "pt");
	// nil falls back to English.
	PreferredLanguage *string `gorm:"column:preferred_language;type:varchar(5)" json:"preferred_language"`
	// PasswordSet reports whether the owner knows a password for the account.
	// Accounts created through Google Sign-In start with an unusable random
	// hash and false until a password is set through the reset link. The
	// database default (true) applies to every password registration.
	PasswordSet bool           `gorm:"column:password_set;not null;default:true" json:"password_set"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at" json:"deleted_at"`
	// Sex ("female" or "male") is asked when the member first opens the
	// training plans and in the Premium Level 1 journey; nil until answered.
	// PlanAudienceFilter records whether the member prefers to see the plans
	// made for their sex first; nil means the question was not answered yet.
	Sex                *string `gorm:"column:sex;type:varchar(10)" json:"sex"`
	PlanAudienceFilter *bool   `gorm:"column:plan_audience_filter" json:"plan_audience_filter"`
	// Username is the account's unique public reference (@username, migration
	// 000049): lowercase letters, digits, dots and underscores, 3-30
	// characters. Left empty on create, the database generates one from the
	// email (default:null keeps it out of the INSERT and reads it back).
	Username string `gorm:"column:username;type:varchar(30);default:null" json:"username"`
	// UsernameChosen: the person chose the @username (registration, a change
	// or a confirmation); false while it is a generated placeholder.
	UsernameChosen bool `gorm:"column:username_chosen;not null;default:false" json:"-"`
	// DisplayName is the name people see (nil: the registered first and last
	// name). AvatarKey is the profile photo under the public uploads.
	DisplayName *string `gorm:"column:display_name;type:varchar(40)" json:"display_name"`
	AvatarKey   *string `gorm:"column:avatar_key;type:varchar(80)" json:"-"`
	// DisplayNameChangedAt: when the nickname last changed (once every 7
	// days, migration 000062).
	DisplayNameChangedAt *time.Time `gorm:"column:display_name_changed_at" json:"-"`
}

// PublicName is the name other people see: the display name, else the
// registered first and last name.
func (u *User) PublicName() string {
	// The nickname, else the @username (owner decision, 2026-11-21: never the
	// email, nor the registered first and last name).
	if u.DisplayName != nil && *u.DisplayName != "" {
		return *u.DisplayName
	}
	return u.Username
}

func (u *User) BeforeCreate(_ *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	return nil
}
