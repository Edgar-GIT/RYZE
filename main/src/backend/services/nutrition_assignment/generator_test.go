package nutrition_assignment

import (
	"strconv"
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

	firstFingerprint, err := fingerprintPlan(normalized, first)
	if err != nil {
		t.Fatalf("fingerprint failed: %v", err)
	}
	secondFingerprint, err := fingerprintPlan(normalized, second)
	if err != nil {
		t.Fatalf("fingerprint failed: %v", err)
	}

	// Determinism is the property the idempotent retry path depends on: a retry
	// after a failure must recompute an identical plan, not a drifted one.
	if firstFingerprint != secondFingerprint {
		t.Error("expected identical intakes to produce identical fingerprints")
	}
	if first.Fingerprint != second.Fingerprint {
		t.Error("expected identical intakes to report identical fingerprints")
	}
	if mealsFingerprint(first) != mealsFingerprint(second) {
		t.Error("expected identical intakes to produce identical meals")
	}
}

// mealsFingerprint reduces a plan's meals and foods to a comparable string so a
// determinism failure points at the composition rather than only the digest.
func mealsFingerprint(plan *GeneratedPlan) string {
	var builder strings.Builder
	for _, meal := range plan.Meals {
		builder.WriteString(meal.Label)
		for _, item := range meal.Items {
			builder.WriteString("|")
			builder.WriteString(item.CatalogCode)
			builder.WriteString("|")
			builder.WriteString(item.Unit)
			builder.WriteString("|")
			builder.WriteString(strconv.FormatFloat(item.Quantity, 'f', -1, 64))
		}
		builder.WriteString(";")
	}
	return builder.String()
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

	if fatLoss.TargetCalories >= maintain.TargetCalories {
		t.Error("expected a fat-loss target below a maintenance target")
	}
	if gain.TargetCalories <= maintain.TargetCalories {
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

	if plan.TargetCalories < minimumTargetCalories {
		t.Errorf("target calories = %d, want at least %d", plan.TargetCalories, minimumTargetCalories)
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

	target := plan.TargetCalories
	computed := plan.Daily.ProteinGrams*proteinKcalPerGram +
		plan.Daily.CarbsGrams*carbohydrateKcalPerGram +
		plan.Daily.FatGrams*fatKcalPerGram

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

	if plan.Daily.FatGrams < fatFloorGrams {
		t.Errorf("fat grams = %d, want at least %d", plan.Daily.FatGrams, fatFloorGrams)
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

// TestGenerateStatesReligiousPatternCaveat asserts that a plan claiming to apply
// halal or kosher never implies a certification the server cannot verify.
func TestGenerateStatesReligiousPatternCaveat(t *testing.T) {
	generator := NewDeterministicGenerator()

	for _, diet := range []string{DietHalal, DietKosher} {
		t.Run(diet, func(t *testing.T) {
			plan, err := generator.Generate(intake(t, func(a *nutrition_questionnaire.Answers) {
				a.Diet = strPtr(diet)
			}))
			if err != nil {
				t.Fatalf("generation failed: %v", err)
			}

			joined := strings.ToLower(strings.Join(plan.Cautions, " "))
			if !strings.Contains(joined, "certification") {
				t.Errorf("cautions = %v, want a certification caveat for %s", plan.Cautions, diet)
			}
		})
	}
}

// TestGenerateCarriesDietaryConstraintsAsExclusions asserts the constraints a plan
// honours are delivered as the exclusion audit rather than as a second copy of the
// client's health answers. The pattern is carried on the plan itself, and every
// removed food is recorded with the reason it was removed.
func TestGenerateCarriesDietaryConstraintsAsExclusions(t *testing.T) {
	generator := NewDeterministicGenerator()

	plan, err := generator.Generate(intake(t, func(a *nutrition_questionnaire.Answers) {
		a.Allergies = stringsPtr("peanuts", "shellfish")
		a.ExcludedFoods = stringsPtr("pork")
		a.Diet = strPtr("vegetarian")
	}))
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if plan.DietaryPattern != "vegetarian" {
		t.Errorf("pattern = %q, want vegetarian", plan.DietaryPattern)
	}

	byReason := map[string][]string{}
	for _, exclusion := range plan.Exclusions {
		byReason[exclusion.ReasonCode] = append(byReason[exclusion.ReasonCode], exclusion.Token)
	}
	if got := len(byReason["allergy"]); got != 2 {
		t.Errorf("allergy exclusions = %v, want 2 recorded", byReason["allergy"])
	}
	if got := len(byReason["excluded_food"]); got != 1 {
		t.Errorf("excluded food exclusions = %v, want 1 recorded", byReason["excluded_food"])
	}
}

// TestGenerateWarnsAboutUnrecognisedRestrictions asserts a restriction the engine
// cannot interpret produces an explicit caution. Silently accepting a word the
// catalog does not know would present an unfiltered plan as a filtered one, which
// is the dangerous failure mode for an allergy.
func TestGenerateWarnsAboutUnrecognisedRestrictions(t *testing.T) {
	generator := NewDeterministicGenerator()

	plan, err := generator.Generate(intake(t, func(a *nutrition_questionnaire.Answers) {
		a.Allergies = stringsPtr("qwoikj")
	}))
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(plan.UnmatchedRestrictions) != 1 {
		t.Fatalf("unmatched restrictions = %v, want the unrecognised entry", plan.UnmatchedRestrictions)
	}

	joined := strings.Join(plan.Cautions, " ")
	if !strings.Contains(joined, "could not be matched") {
		t.Errorf("cautions = %v, want a caution about unrecognised restrictions", plan.Cautions)
	}
	if strings.Contains(joined, "qwoikj") {
		t.Error("cautions must not echo the client's own health text")
	}
}

// TestGenerateRecordsExclusionsAsControlledTokens asserts the audit trail stores
// the normalised vocabulary rather than the client's free text, so a stored
// exclusion can never duplicate a health answer verbatim.
func TestGenerateRecordsExclusionsAsControlledTokens(t *testing.T) {
	generator := NewDeterministicGenerator()

	plan, err := generator.Generate(intake(t, func(a *nutrition_questionnaire.Answers) {
		a.Allergies = stringsPtr("Peanuts, severe")
	}))
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	found := false
	for _, exclusion := range plan.Exclusions {
		if exclusion.ReasonCode != ExclusionAllergy {
			continue
		}
		found = true
		if exclusion.Token != AllergenPeanut {
			t.Errorf("allergy exclusion token = %q, want %q", exclusion.Token, AllergenPeanut)
		}
	}
	if !found {
		t.Errorf("exclusions = %v, want a recorded peanut allergy", plan.Exclusions)
	}
}

func TestGenerateRejectsNilIntake(t *testing.T) {
	if _, err := NewDeterministicGenerator().Generate(nil); err == nil {
		t.Fatal("expected a nil intake to be rejected")
	}
}

// TestGenerateRejectsUnsupportedActivityLevel asserts a value the engine cannot
// interpret fails closed rather than silently falling back to a default that the
// client never asked for.
func TestGenerateRejectsUnsupportedActivityLevel(t *testing.T) {
	normalized := intake(t, nil)
	normalized.ActivityLevel = "unknown"

	if _, err := NewDeterministicGenerator().Generate(normalized); err == nil {
		t.Fatal("expected an unsupported activity level to fail generation")
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
			for _, meal := range plan.Meals {
				total += meal.PercentOfDaily
			}
			if total != 100 {
				t.Errorf("%d meals/%d snacks: distribution totals %d%%, want 100%%", meals, snacks, total)
			}
		}
	}
}

// TestMealsAndSnacksKeepTheMajorityOfTheDay asserts the declared structure is
// honoured even at the maximum snack count: the meals must still hold the
// majority of the energy rather than being squeezed out by the snacks.
func TestMealsAndSnacksKeepTheMajorityOfTheDay(t *testing.T) {
	generator := NewDeterministicGenerator()

	plan, err := generator.Generate(intake(t, func(a *nutrition_questionnaire.Answers) {
		a.MealsPerDay = intPtr(1)
		a.SnacksPerDay = intPtr(nutrition_questionnaire.MaxSnacksPerDay)
	}))
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	mealPercent := 0
	for _, meal := range plan.Meals {
		if meal.Kind == "meal" {
			mealPercent += meal.PercentOfDaily
		}
	}
	if mealPercent < minMealSharePercent {
		t.Errorf("meals hold %d%% of the day, want at least %d%%", mealPercent, minMealSharePercent)
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

	if len(plan.Meals) != 5 {
		t.Errorf("plan has %d meals, want 3 meals plus 2 snacks", len(plan.Meals))
	}

	snacks := 0
	for _, meal := range plan.Meals {
		if meal.Kind == "snack" {
			snacks++
		}
	}
	if snacks != 2 {
		t.Errorf("plan has %d snacks, want 2", snacks)
	}
}

// TestMealPositionsAreSequential asserts the stored order is exactly what the
// client renders, with no gaps or duplicates for the unique per-plan position
// constraint to reject.
func TestMealPositionsAreSequential(t *testing.T) {
	generator := NewDeterministicGenerator()

	plan, err := generator.Generate(intake(t, func(a *nutrition_questionnaire.Answers) {
		a.MealsPerDay = intPtr(4)
		a.SnacksPerDay = intPtr(2)
	}))
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	for index, meal := range plan.Meals {
		if meal.Position != index+1 {
			t.Errorf("meal %q position = %d, want %d", meal.Label, meal.Position, index+1)
		}
		for itemIndex, item := range meal.Items {
			if item.Position != itemIndex+1 {
				t.Errorf("meal %q item %q position = %d, want %d", meal.Label, item.FoodName, item.Position, itemIndex+1)
			}
		}
	}
}

// TestMealTotalsMatchTheirFoods asserts the denormalised meal totals agree with
// the foods they were written from, because they are stored independently.
func TestMealTotalsMatchTheirFoods(t *testing.T) {
	generator := NewDeterministicGenerator()

	plan, err := generator.Generate(intake(t, nil))
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	for _, meal := range plan.Meals {
		if got, want := meal.Macros, sumItems(meal.Items); got != want {
			t.Errorf("meal %q totals = %+v, want %+v from its foods", meal.Label, got, want)
		}
	}
}

// TestGeneratedQuantitiesAreMeasurable asserts every food carries a quantity a
// person can actually weigh or measure, rather than a precise-looking number
// derived from a division.
func TestGeneratedQuantitiesAreMeasurable(t *testing.T) {
	generator := NewDeterministicGenerator()

	plan, err := generator.Generate(intake(t, nil))
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	for _, meal := range plan.Meals {
		for _, item := range meal.Items {
			catalogItem, ok := CatalogItemByCode(item.CatalogCode)
			if !ok {
				t.Fatalf("meal %q references unknown catalog code %q", meal.Label, item.CatalogCode)
			}
			if item.Unit != catalogItem.Unit {
				t.Errorf("meal %q item %q unit = %q, want %q", meal.Label, item.FoodName, item.Unit, catalogItem.Unit)
			}

			switch item.Unit {
			case "g", "ml":
				if remainder := int(item.Quantity) % 5; remainder != 0 {
					t.Errorf("meal %q item %q quantity = %v, want a multiple of 5 %s", meal.Label, item.FoodName, item.Quantity, item.Unit)
				}
			}
		}
	}
}
