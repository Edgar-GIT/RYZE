// Package curated_nutrition owns the validated shape of the predefined
// nutrition plan bundled with a curated Premium Level 1 package.
//
// The plan is package content, not client data: it is authored by the platform,
// stored once per program and delivered unchanged to every entitled buyer. All
// validation lives here so the administration path and the publication gate use
// exactly the same rules.
package curated_nutrition

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"ryze/backend/models"
)

var ErrInvalidInput = errors.New("invalid curated nutrition input")

const (
	MaxMeals          = 14
	MaxItemsPerMeal   = 30
	MaxLabelLength    = 48
	MaxSummaryLength  = 255
	MaxGuidanceLength = 255
	MaxNoteLength     = 255
	MaxFoodNameLength = 120
	MaxCategoryLength = 32
	MaxUnitLength     = 24
	MaxCautions       = 10
	MaxCautionLength  = 160
	MaxQuantity       = 99999
	MaxMacroGrams     = 10000
	MaxCalories       = 10000
	MaxHydration      = 20
)

// PlanInput is the editable content of a curated package plan. Numeric fields
// are already domain values; the handler is responsible only for decoding and
// basic transport-level validation.
type PlanInput struct {
	DietaryPattern  string
	Summary         string
	PrepGuidance    string
	HydrationLitres float64
	HydrationNote   string
	Cautions        []string
	Meals           []MealInput
}

type MealInput struct {
	Label string
	Kind  string
	Notes string
	Items []ItemInput
}

type ItemInput struct {
	FoodName          string
	Category          string
	Quantity          float64
	Unit              string
	Calories          int
	ProteinGrams      int
	CarbohydrateGrams int
	FatGrams          int
	FiberGrams        int
	SubstitutionNote  string
}

// Plan is the normalized, client-safe representation returned by admin reads.
type Plan struct {
	DietaryPattern  string   `json:"dietary_pattern"`
	Summary         string   `json:"summary"`
	PrepGuidance    string   `json:"prep_guidance"`
	HydrationLitres float64  `json:"hydration_litres"`
	HydrationNote   string   `json:"hydration_note"`
	Cautions        []string `json:"cautions"`
	Meals           []Meal   `json:"meals"`
}

type Meal struct {
	Position int    `json:"position"`
	Label    string `json:"label"`
	Kind     string `json:"kind"`
	Notes    string `json:"notes,omitempty"`
	Items    []Item `json:"items"`
}

type Item struct {
	Position          int     `json:"position"`
	FoodName          string  `json:"food_name"`
	Category          string  `json:"category"`
	Quantity          float64 `json:"quantity"`
	Unit              string  `json:"unit"`
	Calories          int     `json:"calories"`
	ProteinGrams      int     `json:"protein_grams"`
	CarbohydrateGrams int     `json:"carbohydrate_grams"`
	FatGrams          int     `json:"fat_grams"`
	FiberGrams        int     `json:"fiber_grams"`
	SubstitutionNote  string  `json:"substitution_note,omitempty"`
}

// Validate normalizes and validates a plan. It returns a defensive copy so the
// caller cannot preserve untrimmed values by retaining its input.
func Validate(input PlanInput) (*Plan, error) {
	plan := &Plan{
		DietaryPattern:  strings.TrimSpace(input.DietaryPattern),
		Summary:         strings.TrimSpace(input.Summary),
		PrepGuidance:    strings.TrimSpace(input.PrepGuidance),
		HydrationLitres: input.HydrationLitres,
		HydrationNote:   strings.TrimSpace(input.HydrationNote),
		Cautions:        make([]string, 0, len(input.Cautions)),
		Meals:           make([]Meal, 0, len(input.Meals)),
	}
	if plan.DietaryPattern == "" {
		plan.DietaryPattern = "balanced"
	}
	if len([]rune(plan.DietaryPattern)) > 32 {
		return nil, invalid("dietary pattern is too long")
	}
	if plan.Summary == "" || len([]rune(plan.Summary)) > MaxSummaryLength {
		return nil, invalid("summary is required and must be at most 255 characters")
	}
	if plan.PrepGuidance == "" || len([]rune(plan.PrepGuidance)) > MaxGuidanceLength {
		return nil, invalid("preparation guidance is required and must be at most 255 characters")
	}
	if plan.HydrationLitres <= 0 || plan.HydrationLitres > MaxHydration {
		return nil, invalid("hydration must be between 0.1 and 20 litres")
	}
	if plan.HydrationNote == "" || len([]rune(plan.HydrationNote)) > MaxNoteLength {
		return nil, invalid("hydration note is required and must be at most 255 characters")
	}
	if len(input.Cautions) > MaxCautions {
		return nil, invalid("a plan is capped at 10 cautions")
	}
	seenCautions := make(map[string]struct{}, len(input.Cautions))
	for _, caution := range input.Cautions {
		caution = strings.TrimSpace(caution)
		if caution == "" {
			continue
		}
		if len([]rune(caution)) > MaxCautionLength {
			return nil, invalid("a caution is too long")
		}
		if _, exists := seenCautions[caution]; exists {
			continue
		}
		seenCautions[caution] = struct{}{}
		plan.Cautions = append(plan.Cautions, caution)
	}

	if len(input.Meals) == 0 {
		return nil, invalid("a curated plan requires at least one meal")
	}
	if len(input.Meals) > MaxMeals {
		return nil, invalid("a curated plan is capped at 14 meals")
	}

	totalCalories := 0
	for i := range input.Meals {
		source := &input.Meals[i]
		label := strings.TrimSpace(source.Label)
		kind := strings.TrimSpace(source.Kind)
		if label == "" || len([]rune(label)) > MaxLabelLength {
			return nil, invalid(fmt.Sprintf("meal %d needs a label of at most 48 characters", i+1))
		}
		if kind == "" {
			kind = models.NutritionPlanMealKindMeal
		}
		if kind != models.NutritionPlanMealKindMeal && kind != models.NutritionPlanMealKindSnack {
			return nil, invalid(fmt.Sprintf("meal %d has an unsupported kind", i+1))
		}
		notes := strings.TrimSpace(source.Notes)
		if len([]rune(notes)) > MaxNoteLength {
			return nil, invalid(fmt.Sprintf("meal %d notes are too long", i+1))
		}
		if len(source.Items) == 0 {
			return nil, invalid(fmt.Sprintf("meal %d needs at least one food", i+1))
		}
		if len(source.Items) > MaxItemsPerMeal {
			return nil, invalid(fmt.Sprintf("meal %d is capped at 30 foods", i+1))
		}

		meal := Meal{
			Position: i + 1,
			Label:    label,
			Kind:     kind,
			Notes:    notes,
			Items:    make([]Item, 0, len(source.Items)),
		}
		for j := range source.Items {
			item, itemCalories, err := validateItem(&source.Items[j], i+1, j+1)
			if err != nil {
				return nil, err
			}
			totalCalories += itemCalories
			meal.Items = append(meal.Items, item)
		}
		plan.Meals = append(plan.Meals, meal)
	}

	if totalCalories <= 0 {
		return nil, invalid("the plan must provide a positive daily energy total")
	}
	return plan, nil
}

func validateItem(source *ItemInput, mealPosition, itemPosition int) (Item, int, error) {
	foodName := strings.TrimSpace(source.FoodName)
	category := strings.TrimSpace(source.Category)
	unit := strings.TrimSpace(source.Unit)
	if foodName == "" || len([]rune(foodName)) > MaxFoodNameLength {
		return Item{}, 0, invalid(fmt.Sprintf("meal %d food %d needs a name of at most 120 characters", mealPosition, itemPosition))
	}
	if category == "" || len([]rune(category)) > MaxCategoryLength {
		return Item{}, 0, invalid(fmt.Sprintf("meal %d food %d needs a category", mealPosition, itemPosition))
	}
	if unit == "" || len([]rune(unit)) > MaxUnitLength {
		return Item{}, 0, invalid(fmt.Sprintf("meal %d food %d needs a unit", mealPosition, itemPosition))
	}
	if source.Quantity <= 0 || source.Quantity > MaxQuantity {
		return Item{}, 0, invalid(fmt.Sprintf("meal %d food %d has an invalid quantity", mealPosition, itemPosition))
	}
	if source.Calories < 0 || source.Calories > MaxCalories ||
		source.ProteinGrams < 0 || source.ProteinGrams > MaxMacroGrams ||
		source.CarbohydrateGrams < 0 || source.CarbohydrateGrams > MaxMacroGrams ||
		source.FatGrams < 0 || source.FatGrams > MaxMacroGrams ||
		source.FiberGrams < 0 || source.FiberGrams > MaxMacroGrams {
		return Item{}, 0, invalid(fmt.Sprintf("meal %d food %d has invalid nutrition values", mealPosition, itemPosition))
	}
	substitution := strings.TrimSpace(source.SubstitutionNote)
	if len([]rune(substitution)) > MaxNoteLength {
		return Item{}, 0, invalid(fmt.Sprintf("meal %d food %d substitution note is too long", mealPosition, itemPosition))
	}

	return Item{
		Position:          itemPosition,
		FoodName:          foodName,
		Category:          category,
		Quantity:          source.Quantity,
		Unit:              unit,
		Calories:          source.Calories,
		ProteinGrams:      source.ProteinGrams,
		CarbohydrateGrams: source.CarbohydrateGrams,
		FatGrams:          source.FatGrams,
		FiberGrams:        source.FiberGrams,
		SubstitutionNote:  substitution,
	}, source.Calories, nil
}

// ToModel maps a validated plan onto the persisted aggregate. The caller owns
// the transaction that inserts the returned tree.
func ToModel(plan *Plan, programID string, previousVersion int) (*models.CuratedNutritionPlan, error) {
	if plan == nil {
		return nil, fmt.Errorf("%w: plan is required", ErrInvalidInput)
	}
	cautions, err := json.Marshal(plan.Cautions)
	if err != nil {
		return nil, fmt.Errorf("failed to encode curated nutrition cautions: %w", err)
	}

	version := previousVersion + 1
	if version <= 0 {
		version = 1
	}
	stored := &models.CuratedNutritionPlan{
		ProgramID:       programID,
		Version:         version,
		DietaryPattern:  plan.DietaryPattern,
		Summary:         plan.Summary,
		PrepGuidance:    plan.PrepGuidance,
		HydrationLitres: plan.HydrationLitres,
		HydrationNote:   plan.HydrationNote,
		Cautions:        cautions,
		Meals:           make([]models.CuratedNutritionPlanMeal, 0, len(plan.Meals)),
	}
	for _, meal := range plan.Meals {
		storedMeal := models.CuratedNutritionPlanMeal{
			Position: meal.Position,
			Label:    meal.Label,
			Kind:     meal.Kind,
			Items:    make([]models.CuratedNutritionPlanMealItem, 0, len(meal.Items)),
		}
		if meal.Notes != "" {
			notes := meal.Notes
			storedMeal.Notes = &notes
		}
		for _, item := range meal.Items {
			storedItem := models.CuratedNutritionPlanMealItem{
				Position:          item.Position,
				FoodName:          item.FoodName,
				Category:          item.Category,
				Quantity:          item.Quantity,
				Unit:              item.Unit,
				Calories:          item.Calories,
				ProteinGrams:      item.ProteinGrams,
				CarbohydrateGrams: item.CarbohydrateGrams,
				FatGrams:          item.FatGrams,
				FiberGrams:        item.FiberGrams,
			}
			if item.SubstitutionNote != "" {
				note := item.SubstitutionNote
				storedItem.SubstitutionNote = &note
			}
			storedMeal.Items = append(storedMeal.Items, storedItem)
		}
		stored.Meals = append(stored.Meals, storedMeal)
	}
	return stored, nil
}

// FromModel projects a persisted plan into the admin-safe representation.
func FromModel(stored *models.CuratedNutritionPlan) *Plan {
	if stored == nil {
		return nil
	}
	plan := &Plan{
		DietaryPattern:  stored.DietaryPattern,
		Summary:         stored.Summary,
		PrepGuidance:    stored.PrepGuidance,
		HydrationLitres: stored.HydrationLitres,
		HydrationNote:   stored.HydrationNote,
		Cautions:        []string{},
		Meals:           make([]Meal, 0, len(stored.Meals)),
	}
	if len(stored.Cautions) > 0 {
		_ = json.Unmarshal(stored.Cautions, &plan.Cautions)
	}
	for _, meal := range stored.Meals {
		projected := Meal{
			Position: meal.Position,
			Label:    meal.Label,
			Kind:     meal.Kind,
			Items:    make([]Item, 0, len(meal.Items)),
		}
		if meal.Notes != nil {
			projected.Notes = *meal.Notes
		}
		for _, item := range meal.Items {
			projectedItem := Item{
				Position:          item.Position,
				FoodName:          item.FoodName,
				Category:          item.Category,
				Quantity:          item.Quantity,
				Unit:              item.Unit,
				Calories:          item.Calories,
				ProteinGrams:      item.ProteinGrams,
				CarbohydrateGrams: item.CarbohydrateGrams,
				FatGrams:          item.FatGrams,
				FiberGrams:        item.FiberGrams,
			}
			if item.SubstitutionNote != nil {
				projectedItem.SubstitutionNote = *item.SubstitutionNote
			}
			projected.Items = append(projected.Items, projectedItem)
		}
		plan.Meals = append(plan.Meals, projected)
	}
	return plan
}

// ValidateModel re-runs the authoring validation against stored content. It is
// used by the publication gate so a package can never be published incomplete,
// even if a row was written by an older or faulty path.
func ValidateModel(stored *models.CuratedNutritionPlan) error {
	plan := FromModel(stored)
	if plan == nil {
		return fmt.Errorf("%w: a curated package requires a nutrition plan", ErrInvalidInput)
	}
	input := PlanInput{
		DietaryPattern:  plan.DietaryPattern,
		Summary:         plan.Summary,
		PrepGuidance:    plan.PrepGuidance,
		HydrationLitres: plan.HydrationLitres,
		HydrationNote:   plan.HydrationNote,
		Cautions:        plan.Cautions,
		Meals:           make([]MealInput, 0, len(plan.Meals)),
	}
	for _, meal := range plan.Meals {
		mealInput := MealInput{Label: meal.Label, Kind: meal.Kind, Notes: meal.Notes, Items: make([]ItemInput, 0, len(meal.Items))}
		for _, item := range meal.Items {
			mealInput.Items = append(mealInput.Items, ItemInput{
				FoodName:          item.FoodName,
				Category:          item.Category,
				Quantity:          item.Quantity,
				Unit:              item.Unit,
				Calories:          item.Calories,
				ProteinGrams:      item.ProteinGrams,
				CarbohydrateGrams: item.CarbohydrateGrams,
				FatGrams:          item.FatGrams,
				FiberGrams:        item.FiberGrams,
				SubstitutionNote:  item.SubstitutionNote,
			})
		}
		input.Meals = append(input.Meals, mealInput)
	}
	_, err := Validate(input)
	return err
}

func invalid(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidInput, message)
}
