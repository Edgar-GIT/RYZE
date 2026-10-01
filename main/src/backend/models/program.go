package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProgramType values describe the product category of a program.
const (
	ProgramTypeFree         = "free"
	ProgramTypePremium      = "premium"
	ProgramTypePersonalized = "personalized"
)

// ProgramProductType values describe which product family a program belongs to.
// It is orthogonal to ProgramType: the latter is the commercial/pricing category
// (free, premium, personalized) and is left untouched for compatibility, while
// the product type decides which parts of the purchase lifecycle the program
// participates in. A premium_level_1 program requires a submitted nutrition
// questionnaire before checkout and receives a server-owned nutrition assignment
// once the purchase is verified. generic is the default, so every program that
// was not explicitly classified keeps its current behaviour.
const (
	ProgramProductTypeGeneric       = "generic"
	ProgramProductTypePremiumLevel1 = "premium_level_1"
)

// ProgramStatus values describe the business state of a program. Publishing is
// only a state: it means the program is available for future consumption and
// carries no purchase, assignment or access semantics.
const (
	ProgramStatusDraft     = "draft"
	ProgramStatusPublished = "published"
)

// ProgramCurrency is the ISO 4217 currency code accepted for program pricing.
// Only EUR is supported initially; additional currencies can be added later.
type ProgramCurrency string

const (
	ProgramCurrencyEUR ProgramCurrency = "EUR"
)

// Program corresponds to the programs table. A program is a training or
// nutrition offer created by a trainer (trainer-owned) or, in the future, by
// the platform itself (platform-owned, trainer_id NULL). The client keeps being
// a regular User; no assignment or access relationship exists yet. Soft-deleted
// programs are excluded from regular queries through GORM's DeletedAt handling.
type Program struct {
	ID               string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	TrainerID        string         `gorm:"column:trainer_id;type:varchar(36)" json:"trainer_id"`
	Name             string         `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Description      string         `gorm:"column:description;type:text" json:"description"`
	Type             string         `gorm:"column:type;type:varchar(20);not null" json:"type"`
	ProductType      string         `gorm:"column:product_type;type:varchar(32);not null;default:generic" json:"product_type"`
	Status           string         `gorm:"column:status;type:varchar(20);not null" json:"status"`
	Level            *string        `gorm:"column:level;type:varchar(20)" json:"level"`
	DurationWeeks    *int           `gorm:"column:duration_weeks" json:"duration_weeks"`
	FrequencyPerWeek *int           `gorm:"column:frequency_per_week" json:"frequency_per_week"`
	TrainingType     *string        `gorm:"column:training_type;type:varchar(40)" json:"training_type"`
	PriceMinorUnits  int64          `gorm:"column:price_minor_units;type:bigint;not null;default:0" json:"price_minor_units"`
	Currency         string         `gorm:"column:currency;type:varchar(3);not null;default:EUR" json:"currency"`
	Weeks            []ProgramWeek  `gorm:"foreignKey:ProgramID" json:"-"`
	CreatedAt        time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"deleted_at"`
}

func (p *Program) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	// Every creation path that does not classify the product explicitly lands on
	// the generic family. Setting it here rather than relying on the column
	// default keeps the in-memory value authoritative, so a program is never
	// read back with an empty product type.
	if p.ProductType == "" {
		p.ProductType = ProgramProductTypeGeneric
	}
	return nil
}
