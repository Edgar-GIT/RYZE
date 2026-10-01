package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NutritionPlanStatus values describe the lifecycle of a generated plan version.
//
// A plan version is immutable once written: nothing in a completed plan is ever
// updated in place. Regenerating an assignment therefore inserts the next
// version and marks the previous one superseded, which is what guarantees a
// delivered plan is never silently rewritten and that the history of what a
// client was shown is always recoverable.
const (
	NutritionPlanStatusActive     = "active"
	NutritionPlanStatusSuperseded = "superseded"
)

// NutritionPlanMealKind values classify an eating occasion within a plan. Meals
// carry the main energy distribution; snacks are the smaller slots the intake
// asked for.
const (
	NutritionPlanMealKindMeal  = "meal"
	NutritionPlanMealKindSnack = "snack"
)

// NutritionPlan corresponds to the nutrition_plans table: the server-owned
// nutrition plan generated for a purchased Premium Level 1 program from the
// owner's questionnaire, stored as a normalised domain rather than an opaque
// document.
//
// A plan belongs to exactly one NutritionAssignment and is scoped by UserID and
// ProgramID as well, so an owner-scoped read never has to join through another
// table before it can be filtered by owner. The meal, meal item and exclusion
// rows hang off PlanID; this row carries the daily totals and the provenance of
// the intake the plan was derived from.
//
// Version is the assignment revision this plan realises, and the pair
// (AssignmentID, Version) is unique, so a run can never produce two plans for the
// same revision. The database additionally enforces at most one active plan per
// assignment, so concurrency is resolved by the storage engine rather than by
// hopeful application logic.
//
// QuestionnaireVersion pins the intake revision the plan was derived from, which
// is what lets the access surface detect a stale plan after a questionnaire
// resubmission instead of serving a plan that no longer matches the intake.
//
// EngineVersion records the generation contract that produced the plan. A plan
// written by an older engine is recognised rather than silently reinterpreted,
// and Fingerprint is the deterministic digest of the intake and the plan body,
// so an unchanged intake is provably still current.
type NutritionPlan struct {
	ID                   string `gorm:"type:char(36);primaryKey" json:"id"`
	AssignmentID         string `gorm:"column:assignment_id;type:varchar(36);not null" json:"assignment_id"`
	UserID               string `gorm:"column:user_id;type:varchar(36);not null" json:"user_id"`
	ProgramID            string `gorm:"column:program_id;type:varchar(36);not null" json:"program_id"`
	QuestionnaireID      string `gorm:"column:questionnaire_id;type:char(36);not null" json:"questionnaire_id"`
	QuestionnaireVersion int    `gorm:"column:questionnaire_version;not null" json:"questionnaire_version"`
	Version              int    `gorm:"column:version;not null" json:"version"`
	Status               string `gorm:"column:status;type:varchar(16);not null;default:active" json:"status"`
	EngineVersion        int    `gorm:"column:engine_version;not null" json:"engine_version"`
	Fingerprint          string `gorm:"column:fingerprint;type:varchar(64);not null" json:"fingerprint"`

	DietaryPattern      string  `gorm:"column:dietary_pattern;type:varchar(32);not null" json:"dietary_pattern"`
	MaintenanceCalories int     `gorm:"column:maintenance_calories;not null" json:"maintenance_calories"`
	TargetCalories      int     `gorm:"column:target_calories;not null" json:"target_calories"`
	DailyProteinGrams   int     `gorm:"column:daily_protein_grams;not null" json:"daily_protein_grams"`
	DailyCarbsGrams     int     `gorm:"column:daily_carbohydrate_grams;not null" json:"daily_carbohydrate_grams"`
	DailyFatGrams       int     `gorm:"column:daily_fat_grams;not null" json:"daily_fat_grams"`
	DailyFiberGrams     int     `gorm:"column:daily_fiber_grams;not null" json:"daily_fiber_grams"`
	ProteinPercent      int     `gorm:"column:protein_percent;not null" json:"protein_percent"`
	CarbohydratePercent int     `gorm:"column:carbohydrate_percent;not null" json:"carbohydrate_percent"`
	FatPercent          int     `gorm:"column:fat_percent;not null" json:"fat_percent"`
	MealsPerDay         int     `gorm:"column:meals_per_day;not null" json:"meals_per_day"`
	SnacksPerDay        int     `gorm:"column:snacks_per_day;not null" json:"snacks_per_day"`
	HydrationLitres     float64 `gorm:"column:hydration_litres;type:decimal(4,1);not null" json:"hydration_litres"`
	HydrationNote       string  `gorm:"column:hydration_note;type:varchar(255);not null" json:"hydration_note"`
	Summary             string  `gorm:"column:summary;type:varchar(255);not null" json:"summary"`
	PrepGuidance        string  `gorm:"column:prep_guidance;type:varchar(255);not null" json:"prep_guidance"`

	// Cautions are the non-prescriptive safety notes shown with the plan. They
	// are a small controlled list rather than a document, so they are stored as a
	// JSON array; they are never questionnaire content.
	Cautions json.RawMessage `gorm:"column:cautions;type:json;not null" json:"cautions"`

	GeneratedAt time.Time      `gorm:"column:generated_at;type:datetime(6);not null" json:"generated_at"`
	CreatedAt   time.Time      `gorm:"column:created_at;type:datetime(6);not null" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;type:datetime(6);not null" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"-"`

	Meals      []NutritionPlanMeal      `gorm:"-" json:"meals,omitempty"`
	Exclusions []NutritionPlanExclusion `gorm:"-" json:"exclusions,omitempty"`
}

func (p *NutritionPlan) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.Status == "" {
		p.Status = NutritionPlanStatusActive
	}
	if p.Version <= 0 {
		p.Version = 1
	}
	return nil
}

// NutritionPlanMeal corresponds to the nutrition_plan_meals table: one eating
// occasion in a plan, carrying its own share of the daily energy.
//
// The macro columns are a denormalised total of the meal's items. They exist so
// a meal can be rendered without aggregating its children on every read, and they
// are written from the items in the same transaction so they cannot disagree.
type NutritionPlanMeal struct {
	ID                string    `gorm:"type:char(36);primaryKey" json:"id"`
	PlanID            string    `gorm:"column:plan_id;type:char(36);not null" json:"plan_id"`
	Position          int       `gorm:"column:position;not null" json:"position"`
	Label             string    `gorm:"column:label;type:varchar(48);not null" json:"label"`
	Kind              string    `gorm:"column:kind;type:varchar(16);not null" json:"kind"`
	PercentOfDaily    int       `gorm:"column:percent_of_daily;not null" json:"percent_of_daily"`
	Calories          int       `gorm:"column:calories;not null" json:"calories"`
	ProteinGrams      int       `gorm:"column:protein_grams;not null" json:"protein_grams"`
	CarbohydrateGrams int       `gorm:"column:carbohydrate_grams;not null" json:"carbohydrate_grams"`
	FatGrams          int       `gorm:"column:fat_grams;not null" json:"fat_grams"`
	Notes             *string   `gorm:"column:notes;type:varchar(255)" json:"notes"`
	CreatedAt         time.Time `gorm:"column:created_at;type:datetime(6);not null" json:"-"`

	Items []NutritionPlanMealItem `gorm:"-" json:"items"`
}

func (m *NutritionPlanMeal) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	if m.Kind == "" {
		m.Kind = NutritionPlanMealKindMeal
	}
	return nil
}

// NutritionPlanMealItem corresponds to the nutrition_plan_meal_items table: one
// food in one meal of one plan version.
//
// The food columns are an immutable snapshot taken at generation time rather than
// a reference to a mutable catalog row. CatalogCode records provenance so the
// food can be traced back to the generator's catalog, while FoodName, Category,
// Quantity and Unit record exactly what the client was shown. Because nothing is
// re-read from a catalog that may later change, a delivered plan is byte-stable
// for ever.
//
// SubstitutionNote is present only where the generator had to swap the item: it
// names the equivalent the client may use instead, so a swap is visible and
// actionable rather than silent.
type NutritionPlanMealItem struct {
	ID                string    `gorm:"type:char(36);primaryKey" json:"id"`
	PlanID            string    `gorm:"column:plan_id;type:char(36);not null" json:"plan_id"`
	MealID            string    `gorm:"column:meal_id;type:char(36);not null" json:"meal_id"`
	Position          int       `gorm:"column:position;not null" json:"position"`
	CatalogCode       string    `gorm:"column:catalog_code;type:varchar(64);not null" json:"catalog_code"`
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

func (i *NutritionPlanMealItem) BeforeCreate(_ *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.NewString()
	}
	return nil
}

// NutritionPlanExclusion corresponds to the nutrition_plan_exclusions table: the
// audit trail of what a generation run excluded and why.
//
// This is what makes the dietary guarantee inspectable rather than implicit. Every
// strict exclusion the run applied is recorded against the plan version that
// applied it, so a question of the form "why is this not in my plan" always has an
// answer in the data.
//
// ReasonCode is a controlled token and Token is the normalised vocabulary entry
// that matched, never raw questionnaire text: an allergy a client typed is
// recorded as the allergen it resolved to, so no free-text health answer is ever
// duplicated into this table.
type NutritionPlanExclusion struct {
	ID         string    `gorm:"type:char(36);primaryKey" json:"id"`
	PlanID     string    `gorm:"column:plan_id;type:char(36);not null" json:"plan_id"`
	ReasonCode string    `gorm:"column:reason_code;type:varchar(32);not null" json:"reason_code"`
	Token      string    `gorm:"column:token;type:varchar(120);not null" json:"token"`
	CreatedAt  time.Time `gorm:"column:created_at;type:datetime(6);not null" json:"-"`
}

func (e *NutritionPlanExclusion) BeforeCreate(_ *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	return nil
}
