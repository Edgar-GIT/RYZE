package curated_nutrition_test

import (
	"errors"
	"strings"
	"testing"

	"ryze/backend/models"
	"ryze/backend/services/curated_nutrition"
)

func validInput() curated_nutrition.PlanInput {
	return curated_nutrition.PlanInput{
		DietaryPattern:  " balanced ",
		Summary:         " A complete day of meals. ",
		PrepGuidance:    " Cook once and portion. ",
		HydrationLitres: 2.5,
		HydrationNote:   " Spread water through the day. ",
		Cautions:        []string{" Drink water ", "Drink water", ""},
		Meals: []curated_nutrition.MealInput{
			{
				Label: " Breakfast ",
				Kind:  models.NutritionPlanMealKindMeal,
				Notes: " Quick ",
				Items: []curated_nutrition.ItemInput{
					{
						FoodName:          " Oats ",
						Category:          " Grain ",
						Quantity:          80,
						Unit:              " g ",
						Calories:          300,
						ProteinGrams:      10,
						CarbohydrateGrams: 50,
						FatGrams:          6,
						FiberGrams:        8,
						SubstitutionNote:  " Rice ",
					},
				},
			},
		},
	}
}

func TestValidateNormalizesAndPositionsContent(t *testing.T) {
	plan, err := curated_nutrition.Validate(validInput())
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if plan.DietaryPattern != "balanced" || plan.Summary != "A complete day of meals." {
		t.Fatalf("expected trimmed metadata, got %+v", plan)
	}
	if len(plan.Cautions) != 1 || plan.Cautions[0] != "Drink water" {
		t.Fatalf("expected deduplicated cautions, got %+v", plan.Cautions)
	}
	if len(plan.Meals) != 1 || plan.Meals[0].Position != 1 {
		t.Fatalf("expected one positioned meal, got %+v", plan.Meals)
	}
	if plan.Meals[0].Items[0].Position != 1 || plan.Meals[0].Items[0].FoodName != "Oats" {
		t.Fatalf("expected one positioned item, got %+v", plan.Meals[0].Items)
	}
}

func TestValidateRejectsIncompletePlans(t *testing.T) {
	tests := map[string]func(*curated_nutrition.PlanInput){
		"missing summary":    func(input *curated_nutrition.PlanInput) { input.Summary = " " },
		"invalid hydration":  func(input *curated_nutrition.PlanInput) { input.HydrationLitres = 0 },
		"missing meals":      func(input *curated_nutrition.PlanInput) { input.Meals = nil },
		"missing meal label": func(input *curated_nutrition.PlanInput) { input.Meals[0].Label = "" },
		"unsupported kind":   func(input *curated_nutrition.PlanInput) { input.Meals[0].Kind = "dessert" },
		"missing items":      func(input *curated_nutrition.PlanInput) { input.Meals[0].Items = nil },
		"invalid quantity":   func(input *curated_nutrition.PlanInput) { input.Meals[0].Items[0].Quantity = 0 },
		"negative macros":    func(input *curated_nutrition.PlanInput) { input.Meals[0].Items[0].ProteinGrams = -1 },
		"zero calories":      func(input *curated_nutrition.PlanInput) { input.Meals[0].Items[0].Calories = 0 },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			input := validInput()
			mutate(&input)
			if _, err := curated_nutrition.Validate(input); !errors.Is(err, curated_nutrition.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestModelRoundTripPreservesPlan(t *testing.T) {
	plan, err := curated_nutrition.Validate(validInput())
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	stored, err := curated_nutrition.ToModel(plan, "program-id", 0)
	if err != nil {
		t.Fatalf("ToModel: %v", err)
	}
	if stored.Version != 1 || stored.ProgramID != "program-id" {
		t.Fatalf("unexpected stored metadata %+v", stored)
	}
	projected := curated_nutrition.FromModel(stored)
	if projected == nil || len(projected.Meals) != 1 || projected.Meals[0].Items[0].FoodName != "Oats" {
		t.Fatalf("unexpected projected plan %+v", projected)
	}
	if err := curated_nutrition.ValidateModel(stored); err != nil {
		t.Fatalf("ValidateModel: %v", err)
	}
	if !strings.Contains(string(stored.Cautions), "Drink water") {
		t.Fatalf("expected encoded cautions, got %q", stored.Cautions)
	}
}

func TestToModelRejectsNilPlan(t *testing.T) {
	if _, err := curated_nutrition.ToModel(nil, "program-id", 0); !errors.Is(err, curated_nutrition.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
