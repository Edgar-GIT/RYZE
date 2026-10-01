package nutrition_assignment

import (
	"strings"
	"testing"

	"ryze/backend/services/nutrition_questionnaire"
)

func intake(t *testing.T, mutate func(*nutrition_questionnaire.Answers)) *nutrition_questionnaire.Normalized {
	t.Helper()

	answers := nutrition_questionnaire.Answers{
		Goal:          strPtr("fat_loss"),
		Experience:    strPtr("intermediate"),
		TrainingDays:  intPtr(4),
		SessionLength: intPtr(60),
		Diet:          strPtr("omnivore"),
		MealsPerDay:   intPtr(4),
		SnacksPerDay:  intPtr(1),
		CookingEffort: strPtr("moderate"),
		BodyWeightKg:  floatPtr(80),
		BodyHeightCm:  intPtr(180),
		ActivityLevel: strPtr("moderate"),
		SleepHours:    floatPtr(7),
		StressLevel:   strPtr("low"),
	}
	if mutate != nil {
		mutate(&answers)
	}

	normalized, err := nutrition_questionnaire.Normalize(answers)
	if err != nil {
		t.Fatalf("failed to normalize intake: %v", err)
	}
	return normalized
}

func strPtr(v string) *string          { return &v }
func intPtr(v int) *int                { return &v }
func floatPtr(v float64) *float64      { return &v }
func stringsPtr(v ...string) *[]string { return &v }

func TestGenerateIsDeterministic(t *testing.T) {
	generator := NewDeterministicGenerator()
	normalized := intake(t, nil)

	first, err := generator.Generate(normalized)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}
	second, err := generator.Generate(normalized)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	firstBytes, err := first.Marshal()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	secondBytes, err := second.Marshal()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	// Determinism is the property the idempotent retry path depends on: a retry
	// after a failure must recompute an identical plan, not a drifted one.
	if string(firstBytes) != string(secondBytes) {
		t.Error("expected identical intakes to produce byte-identical plans")
	}
	if first.Fingerprint != second.Fingerprint {
		t.Error("expected identical intakes to produce identical fingerprints")
	}
}

func TestFingerprintChangesWithIntake(t *testing.T) {
	generator := NewDeterministicGenerator()

	base, err := generator.Generate(intake(t, nil))
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}
	changed, err := generator.Generate(intake(t, func(a *nutrition_questionnaire.Answers) {
		a.Goal = strPtr("muscle_gain")
	}))
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if base.Fingerprint == changed.Fingerprint {
		t.Error("expected a changed intake to produce a different fingerprint")
	}
}

func TestGenerateAppliesGoalAdjustment(t *testing.T) {
	generator := NewDeterministicGenerator()

	fatLoss, err := generator.Generate(intake(t, func(a *nutrition_questionnaire.Answers) {
		a.Goal = strPtr("fat_loss")
	}))
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}
	maintain, err := generator.Generate(intake(t, func(a *nutrition_questionnaire.Answers) {
		a.Goal = strPtr("general_fitness")
	}))
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}
	gain, err := generator.Generate(intake(t, func(a *nutrition_questionnaire.Answers) {
		a.Goal = strPtr("muscle_gain")
	}))
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if fatLoss.EnergyTargets.TargetCalories >= maintain.EnergyTargets.TargetCalories {
		t.Error("expected a fat-loss target below a maintenance target")
	}
	if gain.EnergyTargets.TargetCalories <= maintain.EnergyTargets.TargetCalories {
		t.Error("expected a muscle-gain target above a maintenance target")
	}
}

func TestGenerateRespectsMinimumEnergyFloor(t *testing.T) {
	generator := NewDeterministicGenerator()

	// A small client on the lowest supported activity level with the largest
	// deficit must still receive a conservative floor.
	plan, err := generator.Generate(intake(t, func(a *nutrition_questionnaire.Answers) {
		a.Goal = strPtr("fat_loss")
		a.BodyWeightKg = floatPtr(40)
		a.BodyHeightCm = intPtr(140)
		a.ActivityLevel = strPtr("sedentary")
	}))
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if plan.EnergyTargets.TargetCalories < minimumTargetCalories {
		t.Errorf("target calories = %d, want at least %d", plan.EnergyTargets.TargetCalories, minimumTargetCalories)
	}
}

func TestGenerateKeepsMacrosWithinEnergyBudget(t *testing.T) {
	generator := NewDeterministicGenerator()

	plan, err := generator.Generate(intake(t, func(a *nutrition_questionnaire.Answers) {
		a.Goal = strPtr("muscle_gain")
		a.Experience = strPtr("advanced")
	}))
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	target := plan.EnergyTargets.TargetCalories
	computed := plan.EnergyTargets.DailyProteinGrams*proteinKcalPerGram +
		plan.EnergyTargets.DailyCarbsGrams*carbohydrateKcalPerGram +
		plan.EnergyTargets.DailyFatGrams*fatKcalPerGram

	// Macro rounding means the total need not be exact, but it must stay close.
	difference := computed - target
	if difference < 0 {
		difference = -difference
	}
	if difference > 20 {
		t.Errorf("macro energy %d is too far from target %d", computed, target)
	}
}

func TestGenerateHonoursFatFloor(t *testing.T) {
	generator := NewDeterministicGenerator()

	// A high protein target must never push fat below the floor.
	plan, err := generator.Generate(intake(t, func(a *nutrition_questionnaire.Answers) {
		a.Goal = strPtr("fat_loss")
		a.Experience = strPtr("advanced")
		a.StressLevel = strPtr("high")
	}))
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if plan.EnergyTargets.DailyFatGrams < fatFloorGrams {
		t.Errorf("fat grams = %d, want at least %d", plan.EnergyTargets.DailyFatGrams, fatFloorGrams)
	}
}

func TestGenerateIncludesMedicalCautions(t *testing.T) {
	generator := NewDeterministicGenerator()

	// A plan derived from an intake reporting a medical condition must carry a
	// review note rather than presenting itself as suitable without review.
	plan, err := generator.Generate(intake(t, func(a *nutrition_questionnaire.Answers) {
		a.MedicalConditions = stringsPtr("type 1 diabetes")
		a.Allergies = stringsPtr("peanuts")
	}))
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	joined := strings.ToLower(strings.Join(plan.Cautions, " "))
	if !strings.Contains(joined, "medical condition") {
		t.Errorf("cautions = %v, want a medical review note", plan.Cautions)
	}
	if !strings.Contains(joined, "allergies") {
		t.Errorf("cautions = %v, want an allergy note", plan.Cautions)
	}
	if !strings.Contains(joined, "not medical advice") {
		t.Errorf("cautions = %v, want an explicit non-advice disclaimer", plan.Cautions)
	}
}

func TestGenerateCarriesDietaryRules(t *testing.T) {
	generator := NewDeterministicGenerator()

	plan, err := generator.Generate(intake(t, func(a *nutrition_questionnaire.Answers) {
		a.Allergies = stringsPtr("peanuts", "shellfish")
		a.ExcludedFoods = stringsPtr("pork")
		a.Diet = strPtr("vegetarian")
	}))
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if plan.DietaryRules.Pattern != "vegetarian" {
		t.Errorf("pattern = %q, want vegetarian", plan.DietaryRules.Pattern)
	}
	if len(plan.DietaryRules.Allergies) != 2 {
		t.Errorf("allergies = %v, want 2 recorded exclusions", plan.DietaryRules.Allergies)
	}
	if len(plan.DietaryRules.ExcludedFoods) != 1 {
		t.Errorf("excluded foods = %v, want 1 recorded exclusion", plan.DietaryRules.ExcludedFoods)
	}
}

func TestGenerateRejectsNilIntake(t *testing.T) {
	if _, err := NewDeterministicGenerator().Generate(nil); err == nil {
		t.Fatal("expected a nil intake to be rejected")
	}
}

func TestMealDistributionSumsToWholeBudget(t *testing.T) {
	generator := NewDeterministicGenerator()

	// Every supported combination of meals and snacks must produce a budget
	// that adds up exactly, otherwise the client renders a meal plan that does
	// not match the stated daily target.
	for meals := 1; meals <= nutrition_questionnaire.MaxMealsPerDay; meals++ {
		for snacks := 0; snacks <= nutrition_questionnaire.MaxSnacksPerDay; snacks++ {
			meals, snacks := meals, snacks
			plan, err := generator.Generate(intake(t, func(a *nutrition_questionnaire.Answers) {
				a.MealsPerDay = intPtr(meals)
				a.SnacksPerDay = intPtr(snacks)
			}))
			if err != nil {
				t.Fatalf("generation failed for %d meals/%d snacks: %v", meals, snacks, err)
			}

			total := 0
			for _, slot := range plan.MealPlan.Distribution {
				total += slot.PercentOfDaily
			}
			if total != 100 {
				t.Errorf("%d meals/%d snacks: distribution totals %d%%, want 100%%", meals, snacks, total)
			}
		}
	}
}

func TestMealSlotsUseEveryDeclaredOccasion(t *testing.T) {
	generator := NewDeterministicGenerator()

	plan, err := generator.Generate(intake(t, func(a *nutrition_questionnaire.Answers) {
		a.MealsPerDay = intPtr(3)
		a.SnacksPerDay = intPtr(2)
	}))
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(plan.MealPlan.Distribution) != 5 {
		t.Errorf("distribution has %d slots, want 3 meals plus 2 snacks", len(plan.MealPlan.Distribution))
	}
}

func TestUnmarshalRejectsUnknownPlanVersion(t *testing.T) {
	if _, err := Unmarshal([]byte(`{"version":999}`)); err == nil {
		t.Fatal("expected an unknown plan version to be rejected")
	}
}
