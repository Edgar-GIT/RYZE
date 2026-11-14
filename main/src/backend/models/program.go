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
// participates in. generic is the default, so every program that was not
// explicitly classified keeps its current behaviour.
const (
	ProgramProductTypeGeneric       = "generic"
	ProgramProductTypePremiumLevel1 = "premium_level_1"
	ProgramProductTypePremiumLevel2 = "premium_level_2"
)

// ProgramObjective values are the closed objective vocabulary of curated
// Premium Level 1 packages. The objective is server-owned catalogue metadata:
// it classifies what the package is for and can never be chosen by a buyer.
const (
	ProgramObjectiveGainMuscle           = "gain_muscle"
	ProgramObjectiveLoseFat              = "lose_fat"
	ProgramObjectiveStrengthConditioning = "strength_conditioning"
)

// ProgramObjectives is the closed list accepted for Premium Level 1 packages.
var ProgramObjectives = []string{
	ProgramObjectiveGainMuscle,
	ProgramObjectiveLoseFat,
	ProgramObjectiveStrengthConditioning,
}

// ProgramNutritionMode values persist the two Premium Level 1 delivery models.
// `curated` packages ship a predefined nutrition plan authored with the package.
// `questionnaire_generated` is the legacy model that requires an intake before
// checkout and generates a per-client plan after purchase. Generic programs
// carry NULL because nutrition is not part of their product.
const (
	ProgramNutritionModeCurated                = "curated"
	ProgramNutritionModeQuestionnaireGenerated = "questionnaire_generated"
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
	ID               string                `gorm:"type:varchar(36);primaryKey" json:"id"`
	TrainerID        string                `gorm:"column:trainer_id;type:varchar(36)" json:"trainer_id"`
	Name             string                `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Description      string                `gorm:"column:description;type:text" json:"description"`
	Type             string                `gorm:"column:type;type:varchar(20);not null" json:"type"`
	ProductType      string                `gorm:"column:product_type;type:varchar(32);not null;default:generic" json:"product_type"`
	Objective        *string               `gorm:"column:objective;type:varchar(32)" json:"objective"`
	NutritionMode    *string               `gorm:"column:nutrition_mode;type:varchar(24)" json:"nutrition_mode"`
	ContentVersion   int                   `gorm:"column:content_version;not null;default:1" json:"content_version"`
	Status           string                `gorm:"column:status;type:varchar(20);not null" json:"status"`
	Level            *string               `gorm:"column:level;type:varchar(20)" json:"level"`
	DurationWeeks    *int                  `gorm:"column:duration_weeks" json:"duration_weeks"`
	FrequencyPerWeek *int                  `gorm:"column:frequency_per_week" json:"frequency_per_week"`
	TrainingType     *string               `gorm:"column:training_type;type:varchar(40)" json:"training_type"`
	PriceMinorUnits  int64                 `gorm:"column:price_minor_units;type:bigint;not null;default:0" json:"price_minor_units"`
	Currency         string                `gorm:"column:currency;type:varchar(3);not null;default:EUR" json:"currency"`
	CoverImage       *string               `gorm:"column:cover_image;type:varchar(120)" json:"cover_image"`
	Weeks            []ProgramWeek         `gorm:"foreignKey:ProgramID" json:"-"`
	CuratedNutrition *CuratedNutritionPlan `gorm:"foreignKey:ProgramID" json:"-"`
	CreatedAt        time.Time             `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt        time.Time             `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt        gorm.DeletedAt        `gorm:"column:deleted_at;type:datetime(6)" json:"deleted_at"`
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
	if p.ContentVersion <= 0 {
		p.ContentVersion = 1
	}
	return nil
}

// Catalogue prices (owner decision, 2026-11-14): nobody sets a price. A plan's
// price follows its category only — Free is free, Premium Level 1 costs 14.99
// EUR and Premium Level 2 costs 19.49 EUR. Platform plans take the category of
// their product family; trainer plans (product family generic) take it from
// their commercial type, where premium sells at the Level 1 price and
// personalized at the Level 2 price. Administrators decide the trainer
// commission instead, never the price.
const (
	PremiumLevel1PriceMinorUnits int64 = 1499
	PremiumLevel2PriceMinorUnits int64 = 1949
)

// FamilyProgramType is the only commercial type a platform-owned plan of the
// given product family can have: generic plans are free, Level 1 and Level 2
// packages are premium.
func FamilyProgramType(productType string) string {
	switch productType {
	case ProgramProductTypePremiumLevel1, ProgramProductTypePremiumLevel2:
		return ProgramTypePremium
	default:
		return ProgramTypeFree
	}
}

// CatalogPriceMinorUnits returns the fixed catalogue price of a plan of the
// given product family and commercial type.
func CatalogPriceMinorUnits(productType, programType string) int64 {
	switch productType {
	case ProgramProductTypePremiumLevel1:
		return PremiumLevel1PriceMinorUnits
	case ProgramProductTypePremiumLevel2:
		return PremiumLevel2PriceMinorUnits
	}
	switch programType {
	case ProgramTypePremium:
		return PremiumLevel1PriceMinorUnits
	case ProgramTypePersonalized:
		return PremiumLevel2PriceMinorUnits
	default:
		return 0
	}
}
