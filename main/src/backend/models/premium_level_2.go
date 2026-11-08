package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Premium Level 2 request statuses. A purchase only starts the service: the
// request is not a delivered programme until the trainer has authored both the
// training and nutrition content and delivery has run server-side.
const (
	PremiumLevel2StatusAwaitingIntake            = "awaiting_intake"
	PremiumLevel2StatusAwaitingTrainerAssignment = "awaiting_trainer_assignment"
	PremiumLevel2StatusInProgress                = "in_progress"
	PremiumLevel2StatusReadyForDelivery          = "ready_for_delivery"
	PremiumLevel2StatusDelivered                 = "delivered"
	PremiumLevel2StatusCancelled                 = "cancelled"
)

// PremiumLevel2Request is the purchased service request for one client and one
// Premium Level 2 offer. The client is always a regular User. The trainer and
// training programme are nil until allocated/authored.
type PremiumLevel2Request struct {
	ID                string         `gorm:"type:char(36);primaryKey" json:"id"`
	UserID            string         `gorm:"column:user_id;type:varchar(36);not null" json:"user_id"`
	ProgramID         string         `gorm:"column:program_id;type:varchar(36);not null" json:"program_id"`
	TrainerID         *string        `gorm:"column:trainer_id;type:varchar(36)" json:"trainer_id"`
	TrainingProgramID *string        `gorm:"column:training_program_id;type:varchar(36)" json:"training_program_id"`
	Status            string         `gorm:"column:status;type:varchar(32);not null" json:"status"`
	IntakeVersion     int            `gorm:"column:intake_version;not null;default:0" json:"intake_version"`
	DeliveredAt       *time.Time     `gorm:"column:delivered_at;type:datetime(6)" json:"delivered_at"`
	CreatedAt         time.Time      `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;type:datetime(6)" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(6)" json:"-"`

	OfferName            string  `gorm:"column:offer_name;->" json:"offer_name"`
	OfferDescription     string  `gorm:"column:offer_description;->" json:"offer_description"`
	OfferPriceMinorUnits int64   `gorm:"column:offer_price_minor_units;->" json:"offer_price_minor_units"`
	OfferCurrency        string  `gorm:"column:offer_currency;->" json:"offer_currency"`
	ClientFirstName      string  `gorm:"column:client_first_name;->" json:"client_first_name"`
	ClientLastName       string  `gorm:"column:client_last_name;->" json:"client_last_name"`
	ClientEmail          string  `gorm:"column:client_email;->" json:"client_email"`
	TrainerUserID        *string `gorm:"column:trainer_user_id;->" json:"-"`
	TrainerFirstName     *string `gorm:"column:trainer_first_name;->" json:"trainer_first_name,omitempty"`
	TrainerLastName      *string `gorm:"column:trainer_last_name;->" json:"trainer_last_name,omitempty"`
	TrainingProgramName  *string `gorm:"column:training_program_name;->" json:"training_program_name,omitempty"`
	NutritionPlanID      *string `gorm:"column:nutrition_plan_id;->" json:"nutrition_plan_id,omitempty"`
	NutritionPlanVersion *int    `gorm:"column:nutrition_plan_version;->" json:"nutrition_plan_version,omitempty"`
	NutritionPlanStatus  *string `gorm:"column:nutrition_plan_status;->" json:"nutrition_plan_status,omitempty"`
}

func (PremiumLevel2Request) TableName() string { return "premium_level_2_requests" }

func (r *PremiumLevel2Request) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// PremiumLevel2Intake is one immutable revision of the detailed intake. Answers
// are sensitive and are never logged or projected into a queue listing.
type PremiumLevel2Intake struct {
	ID          string          `gorm:"type:char(36);primaryKey" json:"id"`
	RequestID   string          `gorm:"column:request_id;type:char(36);not null" json:"request_id"`
	Version     int             `gorm:"column:version;not null" json:"version"`
	Answers     json.RawMessage `gorm:"column:answers;type:json;not null" json:"answers"`
	SubmittedAt time.Time       `gorm:"column:submitted_at;type:datetime(6);not null" json:"submitted_at"`
	CreatedAt   time.Time       `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
}

func (PremiumLevel2Intake) TableName() string { return "premium_level_2_intakes" }

func (i *PremiumLevel2Intake) BeforeCreate(_ *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.NewString()
	}
	return nil
}

// PremiumLevel2NutritionPlan is the individual nutrition plan authored for one
// request. It is never shared with another client and is only returned after
// delivery to the owner, or to the allocated trainer/admin.
type PremiumLevel2NutritionPlan struct {
	ID              string                           `gorm:"type:char(36);primaryKey" json:"id"`
	RequestID       string                           `gorm:"column:request_id;type:varchar(36);not null" json:"request_id"`
	Version         int                              `gorm:"column:version;not null;default:1" json:"version"`
	Status          string                           `gorm:"column:status;type:varchar(16);not null;default:draft" json:"status"`
	DietaryPattern  string                           `gorm:"column:dietary_pattern;type:varchar(32);not null;default:balanced" json:"dietary_pattern"`
	Summary         string                           `gorm:"column:summary;type:varchar(255);not null" json:"summary"`
	PrepGuidance    string                           `gorm:"column:prep_guidance;type:varchar(255);not null" json:"prep_guidance"`
	HydrationLitres float64                          `gorm:"column:hydration_litres;type:decimal(4,1);not null" json:"hydration_litres"`
	HydrationNote   string                           `gorm:"column:hydration_note;type:varchar(255);not null" json:"hydration_note"`
	Cautions        json.RawMessage                  `gorm:"column:cautions;type:json;not null" json:"cautions"`
	Meals           []PremiumLevel2NutritionPlanMeal `gorm:"foreignKey:PlanID" json:"meals"`
	CreatedAt       time.Time                        `gorm:"column:created_at;type:datetime(6);not null" json:"created_at"`
	UpdatedAt       time.Time                        `gorm:"column:updated_at;type:datetime(6);not null" json:"updated_at"`
	DeletedAt       gorm.DeletedAt                   `gorm:"column:deleted_at;type:datetime(6)" json:"-"`
}

func (PremiumLevel2NutritionPlan) TableName() string { return "premium_level_2_nutrition_plans" }

func (p *PremiumLevel2NutritionPlan) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}

type PremiumLevel2NutritionPlanMeal struct {
	ID        string                               `gorm:"type:char(36);primaryKey" json:"id"`
	PlanID    string                               `gorm:"column:plan_id;type:char(36);not null" json:"plan_id"`
	Position  int                                  `gorm:"column:position;not null" json:"position"`
	Label     string                               `gorm:"column:label;type:varchar(48);not null" json:"label"`
	Kind      string                               `gorm:"column:kind;type:varchar(16);not null;default:meal" json:"kind"`
	Notes     *string                              `gorm:"column:notes;type:varchar(255)" json:"notes"`
	CreatedAt time.Time                            `gorm:"column:created_at;type:datetime(6);not null" json:"-"`
	Items     []PremiumLevel2NutritionPlanMealItem `gorm:"foreignKey:MealID" json:"items"`
}

func (PremiumLevel2NutritionPlanMeal) TableName() string {
	return "premium_level_2_nutrition_plan_meals"
}

func (m *PremiumLevel2NutritionPlanMeal) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	if m.Kind == "" {
		m.Kind = "meal"
	}
	return nil
}

type PremiumLevel2NutritionPlanMealItem struct {
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

func (PremiumLevel2NutritionPlanMealItem) TableName() string {
	return "premium_level_2_nutrition_plan_meal_items"
}

func (i *PremiumLevel2NutritionPlanMealItem) BeforeCreate(_ *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.NewString()
	}
	return nil
}

// PremiumLevel2RequestEvent is an append-only allocation/delivery audit entry.
type PremiumLevel2RequestEvent struct {
	ID         string    `gorm:"type:char(36);primaryKey" json:"id"`
	RequestID  string    `gorm:"column:request_id;type:char(36);not null" json:"request_id"`
	ActorKind  string    `gorm:"column:actor_kind;type:varchar(16);not null" json:"actor_kind"`
	ActorID    string    `gorm:"column:actor_id;type:varchar(36);not null" json:"actor_id"`
	EventType  string    `gorm:"column:event_type;type:varchar(32);not null" json:"event_type"`
	FromStatus *string   `gorm:"column:from_status;type:varchar(32)" json:"from_status"`
	ToStatus   *string   `gorm:"column:to_status;type:varchar(32)" json:"to_status"`
	CreatedAt  time.Time `gorm:"column:created_at;type:datetime(6)" json:"created_at"`
}

func (PremiumLevel2RequestEvent) TableName() string { return "premium_level_2_request_events" }

func (e *PremiumLevel2RequestEvent) BeforeCreate(_ *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	return nil
}
