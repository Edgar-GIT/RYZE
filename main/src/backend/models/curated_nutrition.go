package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CuratedNutritionPlan corresponds to the curated_nutrition_plans table: the
// server-owned nutrition plan bundled with a curated Premium Level 1 package.
//
// Unlike a legacy generated plan, a curated plan is not derived from a client
// intake. It belongs to the package, not to a buyer, and every buyer of that
// package receives the same content. The plan is therefore a template aggregate:
// its meals and foods are authored by the platform and projected to each
// entitled client by the nutrition read surface.
type CuratedNutritionPlan struct {
	ID              string          `gorm:"type:char(36);primaryKey" json:"id"`
	ProgramID       string          `gorm:"column:program_id;type:char(36);not null;uniqueIndex" json:"program_id"`
	Version         int             `gorm:"column:version;not null;default:1" json:"version"`
	DietaryPattern  string          `gorm:"column:dietary_pattern;type:varchar(32);not null;default:balanced" json:"dietary_pattern"`
	Summary         string          `gorm:"column:summary;type:varchar(255);not null" json:"summary"`
	PrepGuidance    string          `gorm:"column:prep_guidance;type:varchar(255);not null" json:"prep_guidance"`
	HydrationLitres float64         `gorm:"column:hydration_litres;type:decimal(4,1);not null" json:"hydration_litres"`
	HydrationNote   string          `gorm:"column:hydration_note;type:varchar(255);not null" json:"hydration_note"`
	Cautions        json.RawMessage `gorm:"column:cautions;type:json;not null" json:"cautions"`

	Meals     []CuratedNutritionPlanMeal `gorm:"foreignKey:PlanID" json:"-"`
	CreatedAt time.Time                  `gorm:"column:created_at;type:datetime(6);not null" json:"created_at"`
	UpdatedAt time.Time                  `gorm:"column:updated_at;type:datetime(6);not null" json:"updated_at"`
	DeletedAt gorm.DeletedAt             `gorm:"column:deleted_at;type:datetime(6)" json:"-"`
}

func (p *CuratedNutritionPlan) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.Version <= 0 {
		p.Version = 1
	}
	return nil
}

// CuratedNutritionPlanMeal is one eating occasion in a curated package plan.
type CuratedNutritionPlanMeal struct {
	ID        string    `gorm:"type:char(36);primaryKey" json:"id"`
	PlanID    string    `gorm:"column:plan_id;type:char(36);not null" json:"plan_id"`
	Position  int       `gorm:"column:position;not null" json:"position"`
	Label     string    `gorm:"column:label;type:varchar(48);not null" json:"label"`
	Kind      string    `gorm:"column:kind;type:varchar(16);not null;default:meal" json:"kind"`
	Notes     *string   `gorm:"column:notes;type:varchar(255)" json:"notes"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime(6);not null" json:"-"`

	Items []CuratedNutritionPlanMealItem `gorm:"foreignKey:MealID" json:"-"`
}

func (m *CuratedNutritionPlanMeal) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	if m.Kind == "" {
		m.Kind = "meal"
	}
	return nil
}

// CuratedNutritionPlanMealItem is one food in one meal of a curated package
// plan. The macronutrient values are authored content, not a snapshot of a
// generated catalog lookup.
type CuratedNutritionPlanMealItem struct {
	ID                string    `gorm:"type:char(36);primaryKey" json:"id"`
	PlanID            string    `gorm:"column:plan_id;type:char(36);not null" json:"plan_id"`
	MealID            string    `gorm:"column:meal_id;type:char(36);not null" json:"meal_id"`
	Position          int       `gorm:"column:position;not null" json:"position"`
	FoodName          string    `gorm:"column:food_name;type:varchar(120);not null" json:"food_name"`
	Category          string    `gorm:"column:category;type:varchar(32);not null" json:"category"`
	Quantity          float64   `gorm:"column:quantity;type:decimal(7,2);not null" json:"quantity"`
	Unit              string    `gorm:"column:unit;type:varchar(24);not null" json:"unit"`
	Calories          int       `gorm:"column:calories;not null" json:"calories"`
	ProteinGrams      int       `gorm:"column:protein_grams;not null" json:"protein_grams"`
	CarbohydrateGrams int       `gorm:"column:carbohydrate_grams;not null" json:"carbohydrate_grams"`
	FatGrams          int       `gorm:"column:fat_grams;not null" json:"fat_grams"`
	FiberGrams        int       `gorm:"column:fiber_grams;not null" json:"fiber_grams"`
	SubstitutionNote  *string   `gorm:"column:substitution_note;type:varchar(255)" json:"substitution_note"`
	CreatedAt         time.Time `gorm:"column:created_at;type:datetime(6);not null" json:"-"`
}

func (i *CuratedNutritionPlanMealItem) BeforeCreate(_ *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.NewString()
	}
	return nil
}
