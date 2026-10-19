package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Trainer application document kinds. Values are stored verbatim and every
// upload is validated against them.
const (
	DocumentKindCertification  = "certification"
	DocumentKindIdentification = "identification"
	DocumentKindInsurance      = "insurance"
	DocumentKindOther          = "other"
)

// TrainerApplicationDocument is one uploaded piece of evidence for a trainer
// application. The binary content lives in the private evidence store outside
// the database; this row keeps the validated metadata and the opaque storage
// key. ContentType is always one of the allow-listed validated types and
// ByteSize is the number of stored bytes, so a row can never describe an
// unvalidated payload. Soft-deleted rows are preserved so a withdrawn upload
// never destroys the review trail; the file itself is kept on disk.
type TrainerApplicationDocument struct {
	ID               string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	ApplicationID    string         `gorm:"column:application_id;type:varchar(36);not null" json:"application_id"`
	Kind             string         `gorm:"column:kind;type:varchar(40);not null" json:"kind"`
	OriginalFilename string         `gorm:"column:original_filename;type:varchar(255);not null" json:"original_filename"`
	StorageKey       string         `gorm:"column:storage_key;type:varchar(255);not null" json:"storage_key"`
	ContentType      string         `gorm:"column:content_type;type:varchar(100);not null" json:"content_type"`
	ByteSize         int64          `gorm:"column:byte_size;type:bigint unsigned;not null" json:"byte_size"`
	SHA256           string         `gorm:"column:sha256;type:char(64);not null" json:"sha256"`
	CreatedAt        time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"deleted_at"`
}

func (d *TrainerApplicationDocument) BeforeCreate(_ *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	return nil
}

// TrainerApplicationDecision is one append-only reviewer decision. It records
// the reviewing administrator identity (the configured ADMIN_1/ADMIN_2 id, not
// a database user), the status transition and an optional reason. Decision
// rows are never updated or soft-deleted: a review decision is an audit fact.
type TrainerApplicationDecision struct {
	ID            string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	ApplicationID string    `gorm:"column:application_id;type:varchar(36);not null" json:"application_id"`
	AdminID       string    `gorm:"column:admin_id;type:varchar(40);not null" json:"admin_id"`
	FromStatus    string    `gorm:"column:from_status;type:varchar(20);not null" json:"from_status"`
	ToStatus      string    `gorm:"column:to_status;type:varchar(20);not null" json:"to_status"`
	Reason        string    `gorm:"column:reason;type:varchar(500)" json:"reason"`
	CreatedAt     time.Time `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
}

func (d *TrainerApplicationDecision) BeforeCreate(_ *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	return nil
}
