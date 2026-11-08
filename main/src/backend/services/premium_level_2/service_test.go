package premium_level_2_test

import (
	"errors"
	"testing"

	"ryze/backend/services/premium_level_2"
)

func validNutritionInput() premium_level_2.NutritionInput {
	return premium_level_2.NutritionInput{
		DietaryPattern:  "balanced",
		Summary:         "A balanced plan built around the client's training week.",
		PrepGuidance:    "Cook grains in batches and keep protein portions ready.",
		HydrationLitres: 2.5,
		HydrationNote:   "Spread water across the day.",
		Cautions:        []string{"Stop if you feel dizzy."},
		Meals: []premium_level_2.NutritionMeal{
			{Label: "Breakfast", Kind: "meal", Foods: []premium_level_2.NutritionFood{{Name: "Oats", Category: "grain", Quantity: 80, Unit: "g", Calories: 300, ProteinGrams: 10, CarbohydrateGrams: 50, FatGrams: 6, FiberGrams: 8}}},
			{Label: "Lunch", Kind: "meal", Foods: []premium_level_2.NutritionFood{{Name: "Chicken and rice", Category: "meal", Quantity: 1, Unit: "serving", Calories: 600, ProteinGrams: 45, CarbohydrateGrams: 70, FatGrams: 12, FiberGrams: 5}}},
			{Label: "Dinner", Kind: "meal", Foods: []premium_level_2.NutritionFood{{Name: "Salmon and potatoes", Category: "meal", Quantity: 1, Unit: "serving", Calories: 650, ProteinGrams: 40, CarbohydrateGrams: 60, FatGrams: 25, FiberGrams: 7}}},
		},
	}
}

func TestSaveNutritionRejectsIncompletePlanBeforePersistence(t *testing.T) {
	svc := premium_level_2.NewService(nil, nil, nil)
	_, err := svc.SaveNutrition(t.Context(), "trainer", "request", premium_level_2.NutritionInput{})
	var validation *premium_level_2.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
