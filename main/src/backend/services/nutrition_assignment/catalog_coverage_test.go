package nutrition_assignment

import (
	"fmt"
	"testing"

	"ryze/backend/services/nutrition_questionnaire"
)

// baseIntake returns a valid intake whose values a test may then vary.
func baseIntake() *nutrition_questionnaire.Normalized {
	return &nutrition_questionnaire.Normalized{
		SchemaVersion: 1,
		Goal:          "recomposition",
		Experience:    "intermediate",
		TrainingDays:  4,
		SessionLength: 60,
		Diet:          DietOmnivore,
		MealsPerDay:   3,
		SnacksPerDay:  1,
		CookingEffort: "moderate",
		BodyWeightKg:  80,
		BodyHeightCm:  175,
		ActivityLevel: "moderate",
		SleepHours:    7,
		StressLevel:   "moderate",
	}
}

// TestCatalogServesEveryDietAtWorstCaseMealCount is the coverage guarantee behind
// the strict-exclusion promise.
//
// Exclusion can only ever remove foods, so a catalog that is too thin would fail
// generation for a legitimate intake. This exercises the two ends of the
// supported structure — the smallest and the largest number of eating occasions —
// for every diet the questionnaire accepts, and asserts that the mandatory
// categories survive.
func TestCatalogServesEveryDietAtWorstCaseMealCount(t *testing.T) {
	diets := []string{DietOmnivore, DietFlexitarian, DietPescatarian, DietVegetarian, DietVegan, DietHalal, DietKosher}
	structures := []struct {
		meals  int
		snacks int
	}{
		{meals: 1, snacks: 0},
		{meals: 8, snacks: 6},
	}

	generator := NewDeterministicGenerator()

	for _, diet := range diets {
		for _, structure := range structures {
			name := fmt.Sprintf("%s/%dmeals+%dsnacks", diet, structure.meals, structure.snacks)
			t.Run(name, func(t *testing.T) {
				intake := baseIntake()
				intake.Diet = diet
				intake.MealsPerDay = structure.meals
				intake.SnacksPerDay = structure.snacks

				plan, err := generator.Generate(intake)
				if err != nil {
					t.Fatalf("Generate() error = %v, want a plan for every supported diet and structure", err)
				}
				if len(plan.Meals) != structure.meals+structure.snacks {
					t.Fatalf("meals = %d, want %d", len(plan.Meals), structure.meals+structure.snacks)
				}
				for _, meal := range plan.Meals {
					if len(meal.Items) == 0 {
						t.Errorf("meal %q has no foods", meal.Label)
					}
					for _, item := range meal.Items {
						if item.Quantity <= 0 {
							t.Errorf("meal %q item %q has quantity %v, want a usable amount", meal.Label, item.FoodName, item.Quantity)
						}
						if item.Unit == "" {
							t.Errorf("meal %q item %q has no unit", meal.Label, item.FoodName)
						}
					}
				}
			})
		}
	}
}

// TestCatalogServesEveryAllergenExclusion asserts the same coverage guarantee for
// the strict case: excluding every allergen a client could declare must still
// leave a servable catalog. Allergens are the only exclusions that can empty a
// category, so this is where a thin catalog would show up first.
func TestCatalogServesEveryAllergenExclusion(t *testing.T) {
	generator := NewDeterministicGenerator()

	for _, token := range AllergenTokens() {
		t.Run(token, func(t *testing.T) {
			intake := baseIntake()
			intake.Allergies = []string{token}
			intake.MealsPerDay = 8
			intake.SnacksPerDay = 6

			plan, err := generator.Generate(intake)
			if err != nil {
				t.Fatalf("Generate() error = %v, want a plan with %q excluded", err, token)
			}
			for _, meal := range plan.Meals {
				for _, item := range meal.Items {
					catalogItem, ok := CatalogItemByCode(item.CatalogCode)
					if !ok {
						t.Fatalf("meal %q references unknown catalog code %q", meal.Label, item.CatalogCode)
					}
					for _, allergen := range catalogItem.Allergens {
						if allergen == token {
							t.Errorf("meal %q served %q which contains excluded allergen %q", meal.Label, item.FoodName, token)
						}
					}
				}
			}
		})
	}
}

// TestCatalogServesEveryAllergenExclusionOnVeganDiet combines the two hardest
// constraints at once, which is where a catalog designed for either in isolation
// tends to fall short.
func TestCatalogServesEveryAllergenExclusionOnVeganDiet(t *testing.T) {
	generator := NewDeterministicGenerator()

	for _, token := range AllergenTokens() {
		t.Run(token, func(t *testing.T) {
			intake := baseIntake()
			intake.Diet = DietVegan
			intake.Allergies = []string{token}
			intake.MealsPerDay = 8
			intake.SnacksPerDay = 6

			plan, err := generator.Generate(intake)
			if err != nil {
				t.Fatalf("Generate() error = %v, want a vegan plan with %q excluded", err, token)
			}
			for _, meal := range plan.Meals {
				for _, item := range meal.Items {
					catalogItem, _ := CatalogItemByCode(item.CatalogCode)
					if !catalogItem.Vegan {
						t.Errorf("meal %q served non-vegan food %q", meal.Label, item.FoodName)
					}
				}
			}
		})
	}
}

// TestCatalogReportsEveryItem declares coverage explicitly: every catalog item
// must be servable under at least one supported diet, so no entry is dead weight.
func TestCatalogReportsEveryItem(t *testing.T) {
	seen := map[string]int{}
	for _, item := range Catalog() {
		for _, diet := range []string{DietOmnivore, DietVegetarian, DietVegan, DietHalal, DietKosher, DietPescatarian, DietFlexitarian} {
			if dietAllows(diet, item) {
				seen[item.Code]++
			}
		}
	}

	for _, item := range Catalog() {
		if seen[item.Code] == 0 {
			t.Errorf("catalog item %q is excluded under every supported diet", item.Code)
		}
	}
}
