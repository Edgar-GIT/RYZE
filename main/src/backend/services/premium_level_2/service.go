package premium_level_2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

var (
	ErrRequestNotFound  = errors.New("premium level 2 request not found")
	ErrInvalidInput     = errors.New("invalid premium level 2 input")
	ErrInvalidState     = errors.New("premium level 2 request is not in the required state")
	ErrNutritionMissing = errors.New("premium level 2 nutrition not found")
)

const (
	MaxPageSize          = 100
	MaxSummaryLength     = 255
	MaxGuidanceLength    = 255
	MaxMealLabelLength   = 48
	MaxFoodNameLength    = 120
	MaxMealsPerPlan      = 8
	MinMealsPerPlan      = 3
	MaxFoodsPerMeal      = 10
	MinFoodsPerMeal      = 1
	MaxCautions          = 10
	MaxCautionLength     = 200
	MaxPlanNotesLength   = 255
	MaxHydrationLitres   = 8
	MinHydrationLitres   = 0.5
	MaxCaloriesPerFood   = 3000
	MaxMacroGramsPerFood = 500
)

var allowedDietaryPatterns = map[string]bool{
	"balanced": true, "vegetarian": true, "vegan": true, "pescatarian": true,
	"halal": true, "kosher": true, "other": true,
}

var allowedFoodUnits = map[string]bool{
	"g": true, "ml": true, "unit": true, "tbsp": true, "tsp": true,
	"cup": true, "slice": true, "serving": true,
}

var allowedMealKinds = map[string]bool{"meal": true, "snack": true}

// RequestRepository is the data surface of the request lifecycle.
type RequestRepository interface {
	CreateRequest(ctx context.Context, userID, programID string) (*models.PremiumLevel2Request, error)
	FindRequestByUserAndProgram(ctx context.Context, userID, programID string) (*models.PremiumLevel2Request, error)
	FindRequestByID(ctx context.Context, requestID string) (*models.PremiumLevel2Request, error)
	ListByUser(ctx context.Context, userID string) ([]models.PremiumLevel2Request, error)
	List(ctx context.Context, filter repositories.PremiumLevel2Filter, page, limit int) ([]models.PremiumLevel2Request, int64, error)
	CreateIntake(ctx context.Context, requestID string, answers []byte) (*models.PremiumLevel2Request, *models.PremiumLevel2Intake, error)
	FindLatestIntake(ctx context.Context, requestID string) (*models.PremiumLevel2Intake, error)
	AssignTrainer(ctx context.Context, requestID, trainerID, actorID string) (*models.PremiumLevel2Request, error)
	SetTrainingProgram(ctx context.Context, requestID, trainerID, programID string) (*models.PremiumLevel2Request, error)
	SaveNutritionPlan(ctx context.Context, requestID, trainerID string, plan *models.PremiumLevel2NutritionPlan) (*models.PremiumLevel2NutritionPlan, error)
	FindNutritionPlan(ctx context.Context, requestID string) (*models.PremiumLevel2NutritionPlan, error)
	Deliver(ctx context.Context, requestID, trainerID string) (*models.PremiumLevel2Request, error)
	RefreshReadiness(ctx context.Context, requestID string) error
	ListEvents(ctx context.Context, requestID string) ([]models.PremiumLevel2RequestEvent, error)
}

// ProgramRepository resolves the purchased offer and protects the fulfilment
// path from ever creating a request for another product family.
type ProgramRepository interface {
	FindByID(ctx context.Context, programID string) (*models.Program, error)
}

// EntitlementRepository proves the client completed the Premium Level 2
// purchase. Every client read requires it; a request alone is never enough.
type EntitlementRepository interface {
	FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.Entitlement, error)
}

type Service interface {
	Catalog() Catalog
	FulfillPurchase(ctx context.Context, userID, programID string) error
	ListClientRequests(ctx context.Context, userID string) ([]models.PremiumLevel2Request, error)
	GetClientRequest(ctx context.Context, userID, requestID string) (*models.PremiumLevel2Request, *models.PremiumLevel2Intake, *models.PremiumLevel2NutritionPlan, error)
	SubmitIntake(ctx context.Context, userID, requestID string, answers []byte) (*models.PremiumLevel2Request, *models.PremiumLevel2Intake, error)
	GetDeliveredNutrition(ctx context.Context, userID, requestID string) (*models.PremiumLevel2Request, *models.PremiumLevel2NutritionPlan, error)
	ListAdminRequests(ctx context.Context, filter repositories.PremiumLevel2Filter, page, limit int) ([]models.PremiumLevel2Request, int64, error)
	GetAdminRequest(ctx context.Context, requestID string) (*models.PremiumLevel2Request, *models.PremiumLevel2Intake, *models.PremiumLevel2NutritionPlan, []models.PremiumLevel2RequestEvent, error)
	AssignTrainer(ctx context.Context, requestID, trainerID, adminID string) (*models.PremiumLevel2Request, error)
	ListTrainerRequests(ctx context.Context, trainerID string, filter repositories.PremiumLevel2Filter, page, limit int) ([]models.PremiumLevel2Request, int64, error)
	GetTrainerRequest(ctx context.Context, trainerID, requestID string) (*models.PremiumLevel2Request, *models.PremiumLevel2Intake, *models.PremiumLevel2NutritionPlan, []models.PremiumLevel2RequestEvent, error)
	AttachTrainingProgram(ctx context.Context, trainerID, requestID, programID string) (*models.PremiumLevel2Request, error)
	SaveNutrition(ctx context.Context, trainerID, requestID string, input NutritionInput) (*models.PremiumLevel2NutritionPlan, error)
	Deliver(ctx context.Context, trainerID, requestID string) (*models.PremiumLevel2Request, error)
}

type service struct {
	requests     RequestRepository
	programs     ProgramRepository
	entitlements EntitlementRepository
}

func NewService(requests RequestRepository, programs ProgramRepository, entitlements EntitlementRepository) Service {
	return &service{requests: requests, programs: programs, entitlements: entitlements}
}

func (s *service) Catalog() Catalog { return IntakeCatalog() }

// FulfillPurchase starts the service exactly once for a completed Premium Level
// 2 purchase. It is a no-op for every other product family so it can share the
// purchase fulfilment hook with the Premium Level 1 nutrition delivery.
func (s *service) FulfillPurchase(ctx context.Context, userID, programID string) error {
	program, err := s.programs.FindByID(ctx, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrProgramNotFound) {
			return ErrRequestNotFound
		}
		return fmt.Errorf("failed to load premium level 2 offer: %w", err)
	}
	if program.ProductType != models.ProgramProductTypePremiumLevel2 {
		return nil
	}
	if _, err := s.entitlements.FindActiveByUserAndProgram(ctx, userID, programID); err != nil {
		if errors.Is(err, repositories.ErrEntitlementNotFound) {
			return ErrRequestNotFound
		}
		return fmt.Errorf("failed to verify premium level 2 purchase: %w", err)
	}
	if _, err := s.requests.CreateRequest(ctx, userID, programID); err != nil {
		return fmt.Errorf("failed to create premium level 2 request: %w", err)
	}
	return nil
}

func (s *service) ListClientRequests(ctx context.Context, userID string) ([]models.PremiumLevel2Request, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrRequestNotFound
	}
	return s.requests.ListByUser(ctx, userID)
}

func (s *service) GetClientRequest(ctx context.Context, userID, requestID string) (*models.PremiumLevel2Request, *models.PremiumLevel2Intake, *models.PremiumLevel2NutritionPlan, error) {
	request, err := s.ownedRequest(ctx, userID, requestID)
	if err != nil {
		return nil, nil, nil, err
	}
	intake, err := s.requests.FindLatestIntake(ctx, requestID)
	if err != nil && !errors.Is(err, repositories.ErrPremiumLevel2IntakeNotFound) {
		return nil, nil, nil, fmt.Errorf("failed to load premium level 2 intake: %w", err)
	}
	if errors.Is(err, repositories.ErrPremiumLevel2IntakeNotFound) {
		intake = nil
	}
	var plan *models.PremiumLevel2NutritionPlan
	if request.Status == models.PremiumLevel2StatusDelivered {
		plan, err = s.requests.FindNutritionPlan(ctx, requestID)
		if err != nil && !errors.Is(err, repositories.ErrPremiumLevel2PlanNotFound) {
			return nil, nil, nil, fmt.Errorf("failed to load premium level 2 nutrition: %w", err)
		}
		if errors.Is(err, repositories.ErrPremiumLevel2PlanNotFound) {
			plan = nil
		}
	}
	return request, intake, plan, nil
}

func (s *service) SubmitIntake(ctx context.Context, userID, requestID string, answers []byte) (*models.PremiumLevel2Request, *models.PremiumLevel2Intake, error) {
	request, err := s.ownedRequest(ctx, userID, requestID)
	if err != nil {
		return nil, nil, err
	}
	if request.Status != models.PremiumLevel2StatusAwaitingIntake && request.Status != models.PremiumLevel2StatusAwaitingTrainerAssignment {
		return nil, nil, ErrInvalidState
	}
	normalized, fields := NormalizeAnswers(answers)
	if len(fields) > 0 {
		return nil, nil, &ValidationError{Fields: fields}
	}
	updated, intake, err := s.requests.CreateIntake(ctx, requestID, normalized)
	if err != nil {
		if errors.Is(err, repositories.ErrPremiumLevel2InvalidState) {
			return nil, nil, ErrInvalidState
		}
		if errors.Is(err, repositories.ErrPremiumLevel2RequestNotFound) {
			return nil, nil, ErrRequestNotFound
		}
		return nil, nil, fmt.Errorf("failed to submit premium level 2 intake: %w", err)
	}
	return updated, intake, nil
}

func (s *service) GetDeliveredNutrition(ctx context.Context, userID, requestID string) (*models.PremiumLevel2Request, *models.PremiumLevel2NutritionPlan, error) {
	request, err := s.ownedRequest(ctx, userID, requestID)
	if err != nil {
		return nil, nil, err
	}
	if request.Status != models.PremiumLevel2StatusDelivered {
		return nil, nil, ErrNutritionMissing
	}
	plan, err := s.requests.FindNutritionPlan(ctx, requestID)
	if err != nil {
		if errors.Is(err, repositories.ErrPremiumLevel2PlanNotFound) {
			return nil, nil, ErrNutritionMissing
		}
		return nil, nil, fmt.Errorf("failed to load delivered nutrition: %w", err)
	}
	if plan.Status != "delivered" {
		return nil, nil, ErrNutritionMissing
	}
	return request, plan, nil
}

func (s *service) ListAdminRequests(ctx context.Context, filter repositories.PremiumLevel2Filter, page, limit int) ([]models.PremiumLevel2Request, int64, error) {
	page, limit = normalizePage(page, limit)
	return s.requests.List(ctx, filter, page, limit)
}

func (s *service) GetAdminRequest(ctx context.Context, requestID string) (*models.PremiumLevel2Request, *models.PremiumLevel2Intake, *models.PremiumLevel2NutritionPlan, []models.PremiumLevel2RequestEvent, error) {
	return s.detail(ctx, requestID)
}

func (s *service) AssignTrainer(ctx context.Context, requestID, trainerID, adminID string) (*models.PremiumLevel2Request, error) {
	if strings.TrimSpace(trainerID) == "" || strings.TrimSpace(adminID) == "" {
		return nil, ErrInvalidInput
	}
	request, err := s.requests.AssignTrainer(ctx, requestID, trainerID, adminID)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrPremiumLevel2RequestNotFound):
			return nil, ErrRequestNotFound
		case errors.Is(err, repositories.ErrPremiumLevel2InvalidState):
			return nil, ErrInvalidState
		default:
			return nil, fmt.Errorf("failed to assign trainer: %w", err)
		}
	}
	return request, nil
}

func (s *service) ListTrainerRequests(ctx context.Context, trainerID string, filter repositories.PremiumLevel2Filter, page, limit int) ([]models.PremiumLevel2Request, int64, error) {
	filter.TrainerID = trainerID
	page, limit = normalizePage(page, limit)
	return s.requests.List(ctx, filter, page, limit)
}

func (s *service) GetTrainerRequest(ctx context.Context, trainerID, requestID string) (*models.PremiumLevel2Request, *models.PremiumLevel2Intake, *models.PremiumLevel2NutritionPlan, []models.PremiumLevel2RequestEvent, error) {
	request, err := s.requests.FindRequestByID(ctx, requestID)
	if err != nil {
		if errors.Is(err, repositories.ErrPremiumLevel2RequestNotFound) {
			return nil, nil, nil, nil, ErrRequestNotFound
		}
		return nil, nil, nil, nil, fmt.Errorf("failed to load request: %w", err)
	}
	if request.TrainerID == nil || *request.TrainerID != trainerID {
		return nil, nil, nil, nil, ErrRequestNotFound
	}
	return s.detail(ctx, requestID)
}

func (s *service) AttachTrainingProgram(ctx context.Context, trainerID, requestID, programID string) (*models.PremiumLevel2Request, error) {
	if strings.TrimSpace(programID) == "" {
		return nil, ErrInvalidInput
	}
	request, err := s.requests.SetTrainingProgram(ctx, requestID, trainerID, programID)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrPremiumLevel2RequestNotFound), errors.Is(err, repositories.ErrProgramNotFound):
			return nil, ErrRequestNotFound
		case errors.Is(err, repositories.ErrPremiumLevel2InvalidState):
			return nil, ErrInvalidState
		default:
			return nil, fmt.Errorf("failed to attach training programme: %w", err)
		}
	}
	return request, nil
}

func (s *service) SaveNutrition(ctx context.Context, trainerID, requestID string, input NutritionInput) (*models.PremiumLevel2NutritionPlan, error) {
	plan, err := buildNutritionPlan(input)
	if err != nil {
		return nil, err
	}
	saved, err := s.requests.SaveNutritionPlan(ctx, requestID, trainerID, plan)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrPremiumLevel2RequestNotFound):
			return nil, ErrRequestNotFound
		case errors.Is(err, repositories.ErrPremiumLevel2InvalidState):
			return nil, ErrInvalidState
		default:
			return nil, fmt.Errorf("failed to save nutrition plan: %w", err)
		}
	}
	return saved, nil
}

func (s *service) Deliver(ctx context.Context, trainerID, requestID string) (*models.PremiumLevel2Request, error) {
	request, err := s.requests.Deliver(ctx, requestID, trainerID)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrPremiumLevel2RequestNotFound):
			return nil, ErrRequestNotFound
		case errors.Is(err, repositories.ErrPremiumLevel2InvalidState):
			return nil, ErrInvalidState
		default:
			return nil, fmt.Errorf("failed to deliver premium level 2 request: %w", err)
		}
	}
	return request, nil
}

func (s *service) ownedRequest(ctx context.Context, userID, requestID string) (*models.PremiumLevel2Request, error) {
	request, err := s.requests.FindRequestByID(ctx, requestID)
	if err != nil {
		if errors.Is(err, repositories.ErrPremiumLevel2RequestNotFound) {
			return nil, ErrRequestNotFound
		}
		return nil, fmt.Errorf("failed to load request: %w", err)
	}
	if request.UserID != userID {
		return nil, ErrRequestNotFound
	}
	if _, err := s.entitlements.FindActiveByUserAndProgram(ctx, userID, request.ProgramID); err != nil {
		if errors.Is(err, repositories.ErrEntitlementNotFound) {
			return nil, ErrRequestNotFound
		}
		return nil, fmt.Errorf("failed to verify purchase: %w", err)
	}
	return request, nil
}

func (s *service) detail(ctx context.Context, requestID string) (*models.PremiumLevel2Request, *models.PremiumLevel2Intake, *models.PremiumLevel2NutritionPlan, []models.PremiumLevel2RequestEvent, error) {
	request, err := s.requests.FindRequestByID(ctx, requestID)
	if err != nil {
		if errors.Is(err, repositories.ErrPremiumLevel2RequestNotFound) {
			return nil, nil, nil, nil, ErrRequestNotFound
		}
		return nil, nil, nil, nil, fmt.Errorf("failed to load request: %w", err)
	}
	intake, err := s.requests.FindLatestIntake(ctx, requestID)
	if err != nil && !errors.Is(err, repositories.ErrPremiumLevel2IntakeNotFound) {
		return nil, nil, nil, nil, fmt.Errorf("failed to load intake: %w", err)
	}
	if errors.Is(err, repositories.ErrPremiumLevel2IntakeNotFound) {
		intake = nil
	}
	plan, err := s.requests.FindNutritionPlan(ctx, requestID)
	if err != nil && !errors.Is(err, repositories.ErrPremiumLevel2PlanNotFound) {
		return nil, nil, nil, nil, fmt.Errorf("failed to load nutrition: %w", err)
	}
	if errors.Is(err, repositories.ErrPremiumLevel2PlanNotFound) {
		plan = nil
	}
	events, err := s.requests.ListEvents(ctx, requestID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return request, intake, plan, events, nil
}

func normalizePage(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	return page, limit
}

// NutritionInput is the trainer-authored plan submitted for one request.
type NutritionInput struct {
	DietaryPattern  string          `json:"dietary_pattern"`
	Summary         string          `json:"summary"`
	PrepGuidance    string          `json:"prep_guidance"`
	HydrationLitres float64         `json:"hydration_litres"`
	HydrationNote   string          `json:"hydration_note"`
	Cautions        []string        `json:"cautions"`
	Meals           []NutritionMeal `json:"meals"`
}

type NutritionMeal struct {
	Label string          `json:"label"`
	Kind  string          `json:"kind"`
	Notes string          `json:"notes"`
	Foods []NutritionFood `json:"foods"`
}

type NutritionFood struct {
	Name              string  `json:"name"`
	Category          string  `json:"category"`
	Quantity          float64 `json:"quantity"`
	Unit              string  `json:"unit"`
	Calories          int     `json:"calories"`
	ProteinGrams      int     `json:"protein_grams"`
	CarbohydrateGrams int     `json:"carbohydrate_grams"`
	FatGrams          int     `json:"fat_grams"`
	FiberGrams        int     `json:"fiber_grams"`
	SubstitutionNote  string  `json:"substitution_note"`
}

func buildNutritionPlan(input NutritionInput) (*models.PremiumLevel2NutritionPlan, error) {
	fields := map[string]string{}
	input.DietaryPattern = strings.TrimSpace(input.DietaryPattern)
	input.Summary = strings.TrimSpace(input.Summary)
	input.PrepGuidance = strings.TrimSpace(input.PrepGuidance)
	input.HydrationNote = strings.TrimSpace(input.HydrationNote)
	if !allowedDietaryPatterns[input.DietaryPattern] {
		fields["dietary_pattern"] = "Choose a supported dietary pattern."
	}
	if input.Summary == "" || utf8.RuneCountInString(input.Summary) > MaxSummaryLength {
		fields["summary"] = "A summary of up to 255 characters is required."
	}
	if input.PrepGuidance == "" || utf8.RuneCountInString(input.PrepGuidance) > MaxGuidanceLength {
		fields["prep_guidance"] = "Preparation guidance of up to 255 characters is required."
	}
	if input.HydrationLitres < MinHydrationLitres || input.HydrationLitres > MaxHydrationLitres {
		fields["hydration_litres"] = "Hydration must be between 0.5 and 8 litres."
	}
	if utf8.RuneCountInString(input.HydrationNote) > MaxPlanNotesLength {
		fields["hydration_note"] = "Hydration note must be at most 255 characters."
	}
	if len(input.Cautions) > MaxCautions {
		fields["cautions"] = "At most 10 cautions are allowed."
	}
	cautions := make([]string, 0, len(input.Cautions))
	for _, caution := range input.Cautions {
		caution = strings.TrimSpace(caution)
		if caution == "" {
			continue
		}
		if utf8.RuneCountInString(caution) > MaxCautionLength {
			fields["cautions"] = "Each caution must be at most 200 characters."
			break
		}
		cautions = append(cautions, caution)
	}
	if len(input.Meals) < MinMealsPerPlan || len(input.Meals) > MaxMealsPerPlan {
		fields["meals"] = "A plan must contain between 3 and 8 meals."
	}
	meals := make([]models.PremiumLevel2NutritionPlanMeal, 0, len(input.Meals))
	totalCalories := 0
	for mealIndex, source := range input.Meals {
		label := strings.TrimSpace(source.Label)
		kind := strings.TrimSpace(source.Kind)
		notes := strings.TrimSpace(source.Notes)
		if kind == "" {
			kind = "meal"
		}
		if label == "" || utf8.RuneCountInString(label) > MaxMealLabelLength {
			fields[fmt.Sprintf("meals.%d.label", mealIndex)] = "A meal label of up to 48 characters is required."
		}
		if !allowedMealKinds[kind] {
			fields[fmt.Sprintf("meals.%d.kind", mealIndex)] = "Meal type must be meal or snack."
		}
		if utf8.RuneCountInString(notes) > MaxPlanNotesLength {
			fields[fmt.Sprintf("meals.%d.notes", mealIndex)] = "Meal notes must be at most 255 characters."
		}
		if len(source.Foods) < MinFoodsPerMeal || len(source.Foods) > MaxFoodsPerMeal {
			fields[fmt.Sprintf("meals.%d.foods", mealIndex)] = "Each meal must contain between 1 and 10 foods."
		}
		meal := models.PremiumLevel2NutritionPlanMeal{Position: mealIndex + 1, Label: label, Kind: kind}
		if notes != "" {
			meal.Notes = &notes
		}
		meal.Items = make([]models.PremiumLevel2NutritionPlanMealItem, 0, len(source.Foods))
		for foodIndex, food := range source.Foods {
			key := fmt.Sprintf("meals.%d.foods.%d", mealIndex, foodIndex)
			food.Name = strings.TrimSpace(food.Name)
			food.Category = strings.TrimSpace(food.Category)
			food.Unit = strings.TrimSpace(food.Unit)
			food.SubstitutionNote = strings.TrimSpace(food.SubstitutionNote)
			if food.Name == "" || utf8.RuneCountInString(food.Name) > MaxFoodNameLength {
				fields[key+".name"] = "A food name of up to 120 characters is required."
			}
			if food.Category == "" || utf8.RuneCountInString(food.Category) > 32 {
				fields[key+".category"] = "A food category is required."
			}
			if food.Quantity <= 0 || food.Quantity > 100000 {
				fields[key+".quantity"] = "Quantity must be greater than zero."
			}
			if !allowedFoodUnits[food.Unit] {
				fields[key+".unit"] = "Choose a supported unit."
			}
			if food.Calories < 0 || food.Calories > MaxCaloriesPerFood {
				fields[key+".calories"] = "Calories are out of range."
			}
			if food.ProteinGrams < 0 || food.ProteinGrams > MaxMacroGramsPerFood ||
				food.CarbohydrateGrams < 0 || food.CarbohydrateGrams > MaxMacroGramsPerFood ||
				food.FatGrams < 0 || food.FatGrams > MaxMacroGramsPerFood ||
				food.FiberGrams < 0 || food.FiberGrams > MaxMacroGramsPerFood {
				fields[key+".macros"] = "Macronutrients are out of range."
			}
			if utf8.RuneCountInString(food.SubstitutionNote) > MaxPlanNotesLength {
				fields[key+".substitution_note"] = "Substitution note must be at most 255 characters."
			}
			totalCalories += food.Calories
			item := models.PremiumLevel2NutritionPlanMealItem{
				Position:          foodIndex + 1,
				FoodName:          food.Name,
				Category:          food.Category,
				Quantity:          food.Quantity,
				Unit:              food.Unit,
				Calories:          food.Calories,
				ProteinGrams:      food.ProteinGrams,
				CarbohydrateGrams: food.CarbohydrateGrams,
				FatGrams:          food.FatGrams,
				FiberGrams:        food.FiberGrams,
			}
			if food.SubstitutionNote != "" {
				item.SubstitutionNote = &food.SubstitutionNote
			}
			meal.Items = append(meal.Items, item)
		}
		meals = append(meals, meal)
	}
	if totalCalories <= 0 {
		fields["meals"] = "The plan must provide calories."
	}
	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}
	cautionsJSON, err := json.Marshal(cautions)
	if err != nil {
		return nil, fmt.Errorf("failed to encode cautions: %w", err)
	}
	now := time.Now()
	return &models.PremiumLevel2NutritionPlan{
		DietaryPattern:  input.DietaryPattern,
		Summary:         input.Summary,
		PrepGuidance:    input.PrepGuidance,
		HydrationLitres: input.HydrationLitres,
		HydrationNote:   input.HydrationNote,
		Cautions:        cautionsJSON,
		Meals:           meals,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}
