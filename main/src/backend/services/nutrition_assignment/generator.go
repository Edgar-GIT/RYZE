// Package nutrition_assignment generates the server-owned nutrition plan for a
// purchased Premium Level 1 program.
//
// The generator is a deterministic function of the validated intake and the
// curated food catalog: the same intake always produces the same plan, byte for
// byte. That property is what makes the pipeline idempotent — a retry after a
// failure recomputes an identical plan instead of drifting, and Test Mode
// exercises the exact code path a real payment takes.
//
// Generator is the replaceable seam. It is an interface so a future provider can
// be introduced without touching the purchase, entitlement or API layers, and so
// the current deterministic implementation can be tested in isolation with no
// external dependency and no network access.
//
// Nothing here is client-supplied and nothing is editable after generation. The
// only inputs are the stored, validated intake and the server-owned catalog.
package nutrition_assignment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"ryze/backend/models"
	"ryze/backend/services/nutrition_questionnaire"
)

// EngineVersion identifies the generation contract. It is stored on every plan
// version so a plan produced by an older contract is recognizable and can be
// regenerated deliberately rather than being silently reinterpreted.
const EngineVersion = 2

// GeneratedPlan is the server-owned nutrition plan. It mirrors the persisted
// normalised domain exactly — a header, an ordered list of meals, the foods in
// each meal and the exclusions that were applied — so what is generated is what
// is stored and what is shown, with no translation step that could drift.
type GeneratedPlan struct {
	EngineVersion int `json:"engine_version"`
	// Fingerprint is a digest of the intake and the full plan body. A plan
	// regenerated from an unchanged intake reproduces the same fingerprint, which
	// is how an unchanged intake is provably still current.
	Fingerprint string `json:"fingerprint"`

	DietaryPattern      string `json:"dietary_pattern"`
	MaintenanceCalories int    `json:"maintenance_calories"`
	TargetCalories      int    `json:"target_calories"`

	// Daily is the macro breakdown of the whole day. It is the authoritative
	// target, while the meals sum to what the selected foods actually provide.
	Daily Macros `json:"daily"`
	// Percentages are derived from Daily so the client never recomputes them.
	ProteinPercent      int `json:"protein_percent"`
	CarbohydratePercent int `json:"carbohydrate_percent"`
	FatPercent          int `json:"fat_percent"`
	// CaloriesPerGram documents the energy density constants the targets were
	// computed with, so the arithmetic behind them is auditable.
	CaloriesPerGram CaloriesPerGram `json:"calories_per_gram"`

	MealsPerDay  int `json:"meals_per_day"`
	SnacksPerDay int `json:"snacks_per_day"`

	Meals []GeneratedMeal `json:"meals"`

	Hydration Hydration `json:"hydration"`

	// Exclusions is the audit trail of what the run removed and why, using
	// controlled tokens rather than questionnaire text. It is the only record of
	// the constraints the plan honours, so a client can see what was accounted
	// for without the plan duplicating their health answers.
	Exclusions []GeneratedExclusion `json:"exclusions"`
	// UnmatchedRestrictions lists allergy and intolerance entries that resolved
	// to no known allergen token. It drives a caution rather than a silent pass,
	// because an unrecognised restriction is a reason to check labels, not proof
	// that the plan is safe.
	UnmatchedRestrictions []string `json:"-"`

	// Cautions are non-prescriptive safety notes.
	Cautions []string `json:"cautions"`

	PrepGuidance string `json:"prep_guidance"`
	Summary      string `json:"summary"`
}

// GeneratedMeal is one eating occasion in the plan.
type GeneratedMeal struct {
	Position       int             `json:"position"`
	Label          string          `json:"label"`
	Kind           string          `json:"kind"`
	PercentOfDaily int             `json:"percent_of_daily"`
	Macros         Macros          `json:"macros"`
	Notes          string          `json:"notes,omitempty"`
	Items          []GeneratedItem `json:"items"`
}

// GeneratedItem is one food in one meal, carrying the immutable snapshot that is
// persisted alongside it.
type GeneratedItem struct {
	Position         int          `json:"position"`
	CatalogCode      string       `json:"catalog_code"`
	FoodName         string       `json:"food_name"`
	Category         FoodCategory `json:"category"`
	Quantity         float64      `json:"quantity"`
	Unit             string       `json:"unit"`
	Macros           Macros       `json:"macros"`
	SubstitutionNote string       `json:"substitution_note,omitempty"`
}

// GeneratedExclusion is one recorded exclusion.
type GeneratedExclusion struct {
	ReasonCode string `json:"reason_code"`
	Token      string `json:"token"`
}

// Exclusion reason codes. These are the only values written to
// nutrition_plan_exclusions.
const (
	ExclusionAllergy       = "allergy"
	ExclusionIntolerance   = "intolerance"
	ExclusionDiet          = "diet"
	ExclusionExcludedFood  = "excluded_food"
	ExclusionDislikedFood  = "disliked_food"
	ExclusionNoAlternative = "insufficient_alternatives"
)

// CaloriesPerGram documents the energy density constants used by the macro
// arithmetic.
type CaloriesPerGram struct {
	Protein      int `json:"protein"`
	Carbohydrate int `json:"carbohydrate"`
	Fat          int `json:"fat"`
}

// Hydration is the daily fluid target.
type Hydration struct {
	DailyLitres float64 `json:"daily_litres"`
	Note        string  `json:"note"`
}

// ErrNoEligibleFood indicates the catalog cannot satisfy the intake. Generation
// fails safely in this case rather than serving a plan that breaks a declared
// allergy, intolerance, religious exclusion or diet.
var ErrNoEligibleFood = errors.New("no eligible food satisfies the submitted restrictions")

// Generator is the replaceable generation seam. Implementations receive the
// validated intake and return a plan, or a failure that is recorded and
// retryable. A Generator must be deterministic for a given intake: that is the
// contract the idempotent retry path depends on.
type Generator interface {
	Generate(intake *nutrition_questionnaire.Normalized) (*GeneratedPlan, error)
}

// DeterministicGenerator is the built-in Generator. It applies fixed, documented
// arithmetic to the intake and composes meals from the curated catalog, with no
// external dependency, so a plan can always be produced offline and in Test Mode.
type DeterministicGenerator struct{}

// NewDeterministicGenerator returns the built-in generator.
func NewDeterministicGenerator() Generator { return DeterministicGenerator{} }

// Energy density constants used by the macro arithmetic.
const (
	proteinKcalPerGram      = 4
	carbohydrateKcalPerGram = 4
	fatKcalPerGram          = 9
)

// Activity multipliers applied to the Mifflin-St Jeor baseline.
var activityMultipliers = map[string]float64{
	"sedentary": 1.2,
	"light":     1.375,
	"moderate":  1.55,
	"high":      1.725,
}

// Training-day additions applied to maintenance energy.
var trainingDayAdditions = map[int]int{
	1: 150, 2: 250, 3: 350, 4: 420, 5: 500, 6: 550, 7: 600,
}

// Goal calorie adjustments applied to maintenance energy.
const (
	goalFatLossAdjustment       = -0.15
	goalMuscleGainAdjustment    = 0.10
	goalRecompositionAdjustment = 0.0
	goalMaintenanceAdjustment   = 0.0
)

// Protein targets in grams per kilogram of body weight, selected by goal and
// narrowed by experience and stress.
var (
	proteinByGoal = map[string]float64{
		"fat_loss":        2.2,
		"muscle_gain":     1.8,
		"strength":        1.8,
		"endurance":       1.6,
		"general_fitness": 1.6,
		"recomposition":   2.0,
		"mobility":        1.4,
	}
	experienceProteinBonus = map[string]float64{
		"beginner": 0.0, "intermediate": 0.1, "advanced": 0.2,
	}
	stressProteinBonus = map[string]float64{
		"low": 0.0, "moderate": 0.0, "high": 0.1,
	}
)

// Fat floors as a share of total calories, applied so a high-protein target can
// never push fat below a safe minimum.
const (
	minFatPercent    = 0.25
	fatFloorGrams    = 45
	fiberPer1000kcal = 14
)

// minimumTargetCalories is the lowest daily energy target the generator emits.
// It is a conservative floor for a general template, not a clinical threshold.
const minimumTargetCalories = 1200

// Meal composition. Each occasion allocates its share of the daily energy across
// the meal categories in a fixed order. The weights are the target split of the
// occasion budget, not a guarantee: the final quantities are scaled to fit and
// rounded to a human-friendly increment.
const (
	weightProtein = 0.40
	weightCarbs   = 0.40
	weightProduce = 0.20
)

// Serving scaling bounds. A food is never scaled below a quarter of its standard
// serving, because a quantity too small to measure is not usable advice, and never
// far above three servings, which would distort the day's distribution.
const (
	minServingScale = 0.25
	maxServingScale = 3.0
)

// Generate derives the plan from the intake. Every value is determined by the
// intake fields and the catalog: identical inputs always yield identical output.
func (DeterministicGenerator) Generate(intake *nutrition_questionnaire.Normalized) (*GeneratedPlan, error) {
	if intake == nil {
		return nil, errors.New("nutrition intake is required")
	}

	daily, maintenance, err := computeDailyTargets(intake)
	if err != nil {
		return nil, err
	}

	eligible, exclusions, unmatched, err := buildEligibleFoods(intake)
	if err != nil {
		return nil, err
	}

	meals, mealsPerDay, snacksPerDay, err := composeMeals(intake, eligible, daily.TargetCalories)
	if err != nil {
		return nil, err
	}

	plan := &GeneratedPlan{
		EngineVersion:       EngineVersion,
		DietaryPattern:      intake.Diet,
		MaintenanceCalories: maintenance,
		TargetCalories:      daily.TargetCalories,
		Daily:               daily.Macros,
		ProteinPercent:      percentOf(daily.Macros.ProteinGrams*proteinKcalPerGram, daily.TargetCalories),
		CarbohydratePercent: percentOf(daily.Macros.CarbsGrams*carbohydrateKcalPerGram, daily.TargetCalories),
		FatPercent:          percentOf(daily.Macros.FatGrams*fatKcalPerGram, daily.TargetCalories),
		CaloriesPerGram: CaloriesPerGram{
			Protein:      proteinKcalPerGram,
			Carbohydrate: carbohydrateKcalPerGram,
			Fat:          fatKcalPerGram,
		},
		MealsPerDay:           mealsPerDay,
		SnacksPerDay:          snacksPerDay,
		Meals:                 meals,
		Hydration:             buildHydration(intake),
		Exclusions:            exclusions,
		UnmatchedRestrictions: unmatched,
		Cautions:              buildCautions(intake, unmatched),
		PrepGuidance:          prepGuidance(intake),
		Summary:               buildSummary(intake, maintenance, daily.TargetCalories, mealsPerDay, snacksPerDay),
	}

	fingerprint, err := fingerprintPlan(intake, plan)
	if err != nil {
		return nil, err
	}
	plan.Fingerprint = fingerprint

	return plan, nil
}

// dailyTargets is the computed daily budget plus the resolved macro split.
type dailyTargets struct {
	TargetCalories int
	Macros         Macros
}

// computeDailyTargets applies the documented energy and macro arithmetic. It is
// pure: the same intake always yields the same budget.
func computeDailyTargets(intake *nutrition_questionnaire.Normalized) (dailyTargets, int, error) {
	multiplier, ok := activityMultipliers[intake.ActivityLevel]
	if !ok {
		return dailyTargets{}, 0, fmt.Errorf("unsupported activity level %q", intake.ActivityLevel)
	}

	maintenance := int(math.Round(mifflinStJeor(intake) * multiplier))
	maintenance += trainingDayAdditions[intake.TrainingDays]
	target := applyGoal(maintenance, intake.Goal)

	proteinGrams := int(math.Round(intake.BodyWeightKg *
		(proteinByGoal[intake.Goal] + experienceProteinBonus[intake.Experience] + stressProteinBonus[intake.StressLevel])))
	// Protein is never allowed to consume the entire energy budget: cap it so
	// carbohydrate and fat still have room to carry the plan.
	proteinCapGrams := int(math.Round(float64(target) * 0.40 / proteinKcalPerGram))
	if proteinGrams > proteinCapGrams {
		proteinGrams = proteinCapGrams
	}

	remainingCalories := target - proteinGrams*proteinKcalPerGram
	fatGrams := int(math.Round(float64(remainingCalories) * minFatPercent / fatKcalPerGram))
	if fatGrams < fatFloorGrams {
		fatGrams = fatFloorGrams
	}
	carbsGrams := int(math.Round(float64(target-proteinGrams*proteinKcalPerGram-fatGrams*fatKcalPerGram) / carbohydrateKcalPerGram))
	if carbsGrams < 0 {
		carbsGrams = 0
	}

	fiberGrams := int(math.Round(float64(target) * fiberPer1000kcal / 1000))

	return dailyTargets{
		TargetCalories: target,
		Macros: Macros{
			ProteinGrams: proteinGrams,
			CarbsGrams:   carbsGrams,
			FatGrams:     fatGrams,
			FiberGrams:   fiberGrams,
		},
	}, maintenance, nil
}

// mifflinStJeor computes the resting energy estimate. Sex is not collected in
// the intake, so the equation is evaluated with the mean of the male and female
// constants, which keeps the estimate neutral and reproducible.
func mifflinStJeor(intake *nutrition_questionnaire.Normalized) float64 {
	weight := intake.BodyWeightKg
	heightCm := float64(intake.BodyHeightCm)
	age := 30.0

	male := 10*weight + 6.25*heightCm - 5*age + 5
	female := 10*weight + 6.25*heightCm - 5*age - 161
	return (male + female) / 2
}

// applyGoal adjusts maintenance energy to the requested goal.
func applyGoal(maintenance int, goal string) int {
	var adjustment float64
	switch goal {
	case "fat_loss":
		adjustment = goalFatLossAdjustment
	case "muscle_gain":
		adjustment = goalMuscleGainAdjustment
	case "recomposition":
		adjustment = goalRecompositionAdjustment
	default:
		adjustment = goalMaintenanceAdjustment
	}

	target := int(math.Round(float64(maintenance) * (1 + adjustment)))
	if target < minimumTargetCalories {
		target = minimumTargetCalories
	}
	return target
}

// buildEligibleFoods resolves the catalog against the intake's restrictions and
// records every exclusion it applies.
//
// The order of checks is the security-relevant part of this function. A declared
// allergy or intolerance is a strict exclusion evaluated against the allergen
// tokens each catalog food declares, so it can never be defeated by the food being
// named differently or by a spelling difference. Everything else — the diet, a
// named exclusion, a disliked food — is evaluated after, and a disliked food is
// treated as an avoidance rather than a removal because the questionnaire asks
// for swaps rather than deletions.
func buildEligibleFoods(intake *nutrition_questionnaire.Normalized) ([]FoodItem, []GeneratedExclusion, []string, error) {
	allergyTokens, unmatchedAllergies := classifyAllergens(intake.Allergies)
	intoleranceTokens, unmatchedIntolerances := classifyAllergens(intake.Intolerances)

	strictTokens := map[string]bool{}
	for _, token := range append(append([]string{}, allergyTokens...), intoleranceTokens...) {
		strictTokens[token] = true
	}

	// Name-based exclusions apply to every free-text restriction field, so an
	// allergy named in plain words also excludes a food called by that name even
	// when no allergen token matched.
	exclusionWords := restrictionWords(append(append(
		append([]string{}, intake.Allergies...),
		intake.Intolerances...),
		intake.ExcludedFoods...))
	dislikedWords := restrictionWords(intake.DislikedFoods)

	exclusions := make([]GeneratedExclusion, 0, 16)
	recorded := map[string]bool{}
	record := func(reason, token string) {
		key := reason + "\x00" + token
		if token == "" || recorded[key] {
			return
		}
		recorded[key] = true
		exclusions = append(exclusions, GeneratedExclusion{ReasonCode: reason, Token: token})
	}

	// Every resolved strict token is recorded once, whether or not it happened to
	// match a food, so the audit trail shows what the run honoured.
	for _, token := range allergyTokens {
		record(ExclusionAllergy, token)
	}
	for _, token := range intoleranceTokens {
		record(ExclusionIntolerance, token)
	}
	for _, entry := range intake.ExcludedFoods {
		record(ExclusionExcludedFood, strings.Join(foodWords(entry), " "))
	}
	for _, entry := range intake.DislikedFoods {
		record(ExclusionDislikedFood, strings.Join(foodWords(entry), " "))
	}
	record(ExclusionDiet, intake.Diet)

	eligible := make([]FoodItem, 0, len(catalog))
	for _, item := range catalog {
		if !dietAllows(intake.Diet, item) {
			continue
		}
		if intersects(item.Allergens, strictTokens) {
			continue
		}
		if matchesRestriction(item, exclusionWords) {
			continue
		}
		if group, excluded := matchesGroup(item, intake.ExcludedFoods); excluded {
			record(ExclusionExcludedFood, group)
			continue
		}
		if matchesRestriction(item, dislikedWords) {
			// Disliked is an avoidance, not a removal: the food stays eligible so
			// it can still be offered as a substitution elsewhere in the day.
			continue
		}
		if !item.ScalesToServings() {
			continue
		}
		eligible = append(eligible, item)
	}

	if len(eligible) == 0 {
		return nil, nil, nil, ErrNoEligibleFood
	}

	sort.Slice(eligible, func(i, j int) bool { return eligible[i].Code < eligible[j].Code })
	return eligible, exclusions, append(unmatchedAllergies, unmatchedIntolerances...), nil
}

// dietAllows reports whether a food is compatible with a declared diet. An
// unrecognised diet is treated as omnivore, which is the only safe direction:
// the questionnaire validates the value, and refusing to generate for an unknown
// token would be a worse failure than composing a plan without a restriction the
// server cannot interpret.
func dietAllows(diet string, item FoodItem) bool {
	switch diet {
	case DietVegan:
		return item.Vegan
	case DietVegetarian:
		return item.Vegetarian || item.Vegan
	case DietPescatarian:
		return item.Pescatarian || item.Vegetarian || item.Vegan
	case DietHalal:
		return item.Halal
	case DietKosher:
		return item.Kosher
	case DietFlexitarian, DietOmnivore:
		return item.Omnivore || item.Flexitarian
	default:
		return item.Omnivore || item.Flexitarian || item.Vegetarian || item.Vegan
	}
}

// intersects reports whether any of a food's allergen tokens is strictly
// excluded.
func intersects(tokens []string, excluded map[string]bool) bool {
	for _, token := range tokens {
		if excluded[token] {
			return true
		}
	}
	return false
}

// Meal structure. A snack takes a fixed slice of the daily budget and the meals
// share the remainder, so adding a snack can never inflate the total. When the
// requested snack count would claim too much of the day, the per-snack share is
// reduced instead of dropping snacks, which keeps the client's declared structure
// while guaranteeing the meals keep the majority of the energy.
const (
	snackSharePercent    = 10
	maxSnackSharePercent = 50
	minSnackSharePercent = 5
	minMealSharePercent  = 50
)

// composeMeals distributes the daily energy across the eating occasions the
// intake asked for and fills each one from the eligible foods.
func composeMeals(intake *nutrition_questionnaire.Normalized, eligible []FoodItem, targetCalories int) ([]GeneratedMeal, int, int, error) {
	meals, snacks := mealStructure(intake)

	byCategory := groupByCategory(eligible)

	snackShare := snackSharePercent
	if snacks > 0 && snacks*snackShare > maxSnackSharePercent {
		snackShare = maxSnackSharePercent / snacks
		if snackShare < minSnackSharePercent {
			snackShare = minSnackSharePercent
		}
	}

	mealBudget := 100 - snacks*snackShare
	mealPercents := distributeEvenly(mealBudget, meals)
	labels := mealLabels(meals)

	// A category that can never be filled makes a plan unusable, so this is
	// detected before any meal is built rather than half way through.
	for _, required := range []FoodCategory{FoodCategoryProtein, FoodCategoryCarbohydrate} {
		if len(byCategory[required]) == 0 {
			return nil, 0, 0, fmt.Errorf("%w: no eligible %s", ErrNoEligibleFood, required)
		}
	}

	occasions := make([]GeneratedMeal, 0, meals+snacks)
	for index, label := range labels {
		meal, err := composeMeal(label, models.NutritionPlanMealKindMeal, index+1, mealPercents[index], targetCalories, byCategory)
		if err != nil {
			return nil, 0, 0, err
		}
		occasions = append(occasions, meal)
	}
	for index := 0; index < snacks; index++ {
		label := "Snack"
		if snacks > 1 {
			label = fmt.Sprintf("Snack %d", index+1)
		}
		// Snacks continue the meal numbering instead of restarting it. Positions
		// are unique within a plan, so a snack restarting at one would collide
		// with the first meal and the plan could not be stored at all.
		meal, err := composeSnack(label, len(occasions)+1, snackShare, targetCalories, byCategory)
		if err != nil {
			return nil, 0, 0, err
		}
		occasions = append(occasions, meal)
	}

	return occasions, meals, snacks, nil
}

// mealStructure resolves the number of meals and snacks from the intake.
//
// The questionnaire already bounds both fields, so the values are trusted and
// only the defensive bounds are applied here: a stored value outside the supported
// range would otherwise produce a plan whose percentages do not describe it.
func mealStructure(intake *nutrition_questionnaire.Normalized) (int, int) {
	meals := intake.MealsPerDay
	if meals < 1 || meals > nutrition_questionnaire.MaxMealsPerDay {
		meals = 3
	}
	snacks := intake.SnacksPerDay
	if snacks < 0 || snacks > nutrition_questionnaire.MaxSnacksPerDay {
		snacks = 0
	}
	return meals, snacks
}

// groupByCategory indexes the eligible foods by the role they fill.
func groupByCategory(eligible []FoodItem) map[FoodCategory][]FoodItem {
	grouped := map[FoodCategory][]FoodItem{}
	for _, item := range eligible {
		grouped[item.Category] = append(grouped[item.Category], item)
	}
	return grouped
}

// composeMeal fills one main meal from the occasion's calorie budget.
func composeMeal(label, kind string, position, percent, targetCalories int, byCategory map[FoodCategory][]FoodItem) (GeneratedMeal, error) {
	budget := occasionBudget(targetCalories, percent)

	meal := GeneratedMeal{
		Position:       position,
		Label:          label,
		Kind:           kind,
		PercentOfDaily: percent,
		Notes:          mealNotes(kind),
	}

	remaining := budget
	sequence := []struct {
		category FoodCategory
		weight   float64
	}{
		{FoodCategoryProtein, weightProtein},
		{FoodCategoryCarbohydrate, weightCarbs},
		{FoodCategoryVegetable, weightProduce},
		{FoodCategoryFruit, weightProduce},
	}

	used := map[string]bool{}
	for _, step := range sequence {
		share := int(math.Round(float64(budget) * step.weight))
		if share > remaining {
			share = remaining
		}
		item, ok := chooseItem(byCategory[step.category], share, used)
		if !ok {
			continue
		}
		meal.Items = append(meal.Items, buildItem(len(meal.Items)+1, item, share, byCategory[step.category], used))
		used[item.Code] = true
		remaining -= meal.Items[len(meal.Items)-1].Macros.Calories
	}

	if fat, ok := chooseItem(byCategory[FoodCategoryFat], remaining, used); ok && remaining >= minSnackSharePercent*6 {
		meal.Items = append(meal.Items, buildItem(len(meal.Items)+1, fat, remaining, byCategory[FoodCategoryFat], used))
		used[fat.Code] = true
	}

	if len(meal.Items) == 0 {
		return GeneratedMeal{}, fmt.Errorf("%w: %s could not be composed", ErrNoEligibleFood, label)
	}

	meal.Macros = sumItems(meal.Items)
	return meal, nil
}

// composeSnack fills a snack from its smaller budget. A snack is deliberately
// simpler than a meal: a protein and a produce item, with fat only when the
// budget is large enough to carry it.
func composeSnack(label string, position, percent, targetCalories int, byCategory map[FoodCategory][]FoodItem) (GeneratedMeal, error) {
	budget := occasionBudget(targetCalories, percent)

	meal := GeneratedMeal{
		Position:       position,
		Label:          label,
		Kind:           models.NutritionPlanMealKindSnack,
		PercentOfDaily: percent,
		Notes:          mealNotes("snack"),
	}

	remaining := budget
	sequence := []struct {
		category FoodCategory
		weight   float64
	}{
		{FoodCategoryProtein, 0.5},
		{FoodCategoryFruit, 0.3},
		{FoodCategoryVegetable, 0.2},
	}

	used := map[string]bool{}
	for _, step := range sequence {
		share := int(math.Round(float64(budget) * step.weight))
		if share > remaining {
			share = remaining
		}
		item, ok := chooseItem(byCategory[step.category], share, used)
		if !ok {
			continue
		}
		meal.Items = append(meal.Items, buildItem(len(meal.Items)+1, item, share, byCategory[step.category], used))
		used[item.Code] = true
		remaining -= meal.Items[len(meal.Items)-1].Macros.Calories
	}

	if fat, ok := chooseItem(byCategory[FoodCategoryFat], remaining, used); ok && remaining >= 45 {
		meal.Items = append(meal.Items, buildItem(len(meal.Items)+1, fat, remaining, byCategory[FoodCategoryFat], used))
	}

	if len(meal.Items) == 0 {
		return GeneratedMeal{}, fmt.Errorf("%w: %s could not be composed", ErrNoEligibleFood, label)
	}

	meal.Macros = sumItems(meal.Items)
	return meal, nil
}

// mealNotes gives the client the intent of the occasion rather than a bare label.
func mealNotes(kind string) string {
	if kind == "snack" {
		return "Kept deliberately small to hold the daily calorie target steady."
	}
	return "Sized to this occasion's share of the day's energy."
}

// occasionBudget converts a percentage share of the daily target into calories.
func occasionBudget(targetCalories, percent int) int {
	return int(math.Round(float64(targetCalories) * float64(percent) / 100))
}

// chooseItem picks the eligible food in a category whose standard serving best
// fits the requested calorie share, and is not already used in this meal.
//
// Selection is deterministic: the category slice is sorted by code and a strict
// improvement is required to displace the incumbent, so ties resolve to the
// lexicographically first code on every run.
func chooseItem(candidates []FoodItem, targetCalories int, used map[string]bool) (FoodItem, bool) {
	if targetCalories <= 0 {
		targetCalories = 1
	}

	var (
		best     FoodItem
		found    bool
		bestDiff = math.MaxInt
	)
	for _, candidate := range candidates {
		if used[candidate.Code] || !candidate.ScalesToServings() {
			continue
		}
		diff := abs(candidate.Calories - targetCalories)
		if !found || diff < bestDiff {
			best, bestDiff, found = candidate, diff, true
		}
	}
	return best, found
}

// buildItem scales a chosen food to the requested calorie share and records the
// result as the immutable snapshot that gets persisted.
func buildItem(position int, item FoodItem, targetCalories int, alternatives []FoodItem, used map[string]bool) GeneratedItem {
	quantity := scaleForCalories(item, targetCalories)

	return GeneratedItem{
		Position:    position,
		CatalogCode: item.Code,
		FoodName:    item.Name,
		Category:    item.Category,
		Quantity:    quantity,
		Unit:        item.Unit,
		Macros:      item.MacrosAtQuantity(quantity),
		// A substitution note is only meaningful where an equivalent the client
		// could actually use exists, so it is omitted rather than filled with a
		// food that has already appeared in this meal.
		SubstitutionNote: substitutionNote(item, alternatives, used),
	}
}

// scaleForCalories returns the quantity, in the food's own unit, whose energy
// best matches a calorie target.
//
// Rounding is by unit: mass and volume round to the nearest five, which is the
// precision a kitchen scale and a measuring jug support, while any other unit
// rounds to the nearest half. Without this the plan would advise quantities like
// "137 g", which is precise on paper and useless in practice.
func scaleForCalories(item FoodItem, targetCalories int) float64 {
	if item.Calories <= 0 {
		return item.Serving
	}

	servings := float64(targetCalories) / float64(item.Calories)
	servings = math.Max(minServingScale, math.Min(maxServingScale, servings))

	quantity := roundTo(servings*item.Serving, unitIncrement(item))

	// A quantity below the smallest measurable step is not usable advice, so the
	// result is floored at the smallest amount a person could weigh out, itself
	// rounded up onto the same grid to keep every quantity measurable.
	if floor := ceilTo(math.Max(item.Serving*minServingScale, unitIncrement(item)), unitIncrement(item)); quantity < floor {
		quantity = floor
	}
	return quantity
}

// unitIncrement returns the measurement step for a food's unit.
func unitIncrement(item FoodItem) float64 {
	switch item.Unit {
	case "g", "ml":
		return 5
	default:
		return 0.5
	}
}

// roundTo rounds a value to the nearest increment.
func roundTo(value, increment float64) float64 {
	if increment <= 0 {
		return value
	}
	return math.Round(value/increment) * increment
}

// ceilTo rounds a value up to the nearest increment.
func ceilTo(value, increment float64) float64 {
	if increment <= 0 {
		return value
	}
	return math.Ceil(value/increment) * increment
}

// substitutionNote names an equivalent the client may swap in. It only reports a
// food that is eligible for this client and is not already in the meal, so the
// note can never point at something the client excluded.
func substitutionNote(item FoodItem, alternatives []FoodItem, used map[string]bool) string {
	for _, candidate := range alternatives {
		if candidate.Code == item.Code || used[candidate.Code] || candidate.Category != item.Category {
			continue
		}
		return fmt.Sprintf("Can be swapped for %s.", candidate.Name)
	}
	return ""
}

// sumItems totals the macros of a meal's foods.
func sumItems(items []GeneratedItem) Macros {
	total := Macros{}
	for _, item := range items {
		total = total.Add(item.Macros)
	}
	return total
}

// distributeEvenly splits a percentage budget across a number of slots so the
// shares always add up to exactly the requested total. The remainder is handed
// out one point at a time, which keeps the distribution stable and exact.
func distributeEvenly(total, parts int) []int {
	if parts <= 0 {
		return nil
	}
	base := total / parts
	remainder := total - base*parts

	shares := make([]int, parts)
	for i := range shares {
		shares[i] = base
		if i < remainder {
			shares[i]++
		}
	}
	return shares
}

// mealLabels returns stable, ordered labels for a meal count within the
// supported range. The labels describe main meals only and deliberately never say
// "snack": snacks are the separate occasions declared by the intake, and reusing
// the word here would leave a client with two kinds of snack it cannot tell apart.
func mealLabels(meals int) []string {
	switch meals {
	case 1:
		return []string{"Daily plate"}
	case 2:
		return []string{"Breakfast", "Dinner"}
	case 3:
		return []string{"Breakfast", "Lunch", "Dinner"}
	case 4:
		return []string{"Breakfast", "Lunch", "Dinner", "Supper"}
	case 5:
		return []string{"Breakfast", "Brunch", "Lunch", "Dinner", "Supper"}
	case 6:
		return []string{"Breakfast", "Brunch", "Lunch", "Afternoon tea", "Dinner", "Supper"}
	case 7:
		return []string{"Breakfast", "Brunch", "Mid-morning", "Lunch", "Afternoon tea", "Dinner", "Supper"}
	case 8:
		return []string{"Breakfast", "Brunch", "Mid-morning", "Lunch", "Afternoon tea", "Evening meal", "Supper", "Late supper"}
	default:
		return []string{"Meal 1", "Meal 2", "Meal 3"}
	}
}

// prepGuidance derives batch-cooking guidance from the declared cooking effort.
func prepGuidance(intake *nutrition_questionnaire.Normalized) string {
	switch intake.CookingEffort {
	case "minimal":
		return "Favour ready-made and minimally processed staples. Batch-cook one protein source and one carbohydrate source per week, and keep frozen vegetables as the default vegetable."
	case "extensive":
		return "Cook fresh most days. Batch-cook two protein sources and two carbohydrate sources mid-week, and prep vegetables in a single session."
	default:
		return "Batch-cook one protein source and one carbohydrate source per week, and prep vegetables in a single session."
	}
}

// buildHydration derives the daily fluid target. It is scaled by training load
// and capped so it stays a general template rather than a clinical prescription.
func buildHydration(intake *nutrition_questionnaire.Normalized) Hydration {
	litres := 2.0 + 0.3*float64(intake.TrainingDays)
	if litres > 4.0 {
		litres = 4.0
	}
	return Hydration{
		DailyLitres: math.Round(litres*10) / 10,
		Note:        "Adjust to thirst, climate and your clinician's advice. Fluid needs rise on hotter days and long training sessions.",
	}
}

// buildCautions returns the non-prescriptive safety notes for the intake. A plan
// derived from an intake that reports a medical condition or an active injury is
// always accompanied by a review note, because the generator is a general
// template and not a medical assessment.
func buildCautions(intake *nutrition_questionnaire.Normalized, unmatchedRestrictions []string) []string {
	cautions := []string{
		"This plan is a general nutrition template generated from your questionnaire. It is not medical advice, a diagnosis, or a substitute for professional care.",
	}
	if len(intake.MedicalConditions) > 0 {
		cautions = append(cautions, "You reported a medical condition. Review this plan with a qualified professional before making significant changes to your diet.")
	}
	if len(intake.Injuries) > 0 {
		cautions = append(cautions, "You reported a current injury or pain. Adjust your training volume accordingly and seek assessment if the pain persists.")
	}
	if len(intake.Allergies) > 0 || len(intake.Intolerances) > 0 {
		cautions = append(cautions, "You reported allergies or intolerances. Every item in this plan was selected from a closed catalog that declares its allergens, but no system can verify a product's manufacturing or cross-contamination: check every ingredient label yourself.")
	}
	if len(unmatchedRestrictions) > 0 {
		// The matched restrictions are excluded from the catalog; an unmatched
		// one is not. Saying so plainly is safer than implying the plan covered
		// something the engine did not understand. The entries themselves are on
		// purpose not echoed: they are the client's own health text.
		cautions = append(cautions, "Some of the allergies or intolerances you entered could not be matched to a known ingredient. They were not used to filter this plan, so review every item and its label before eating it.")
	}
	switch intake.Diet {
	case DietHalal:
		cautions = append(cautions, "Your halal pattern is applied from ingredient composition alone. Confirm certification and sourcing, particularly for dairy and meat products.")
	case DietKosher:
		cautions = append(cautions, "Your kosher pattern is applied from ingredient composition alone. Confirm certification, rennet sourcing and the separation of dairy and meat.")
	}
	if intake.SleepHours < 6 {
		cautions = append(cautions, "Your reported sleep is below six hours. Sleep has a large effect on recovery and on how your body partitions energy; treat it as a priority alongside this plan.")
	}
	return cautions
}

// buildSummary renders a short, stable description of the plan.
func buildSummary(intake *nutrition_questionnaire.Normalized, maintenance, target, meals, snacks int) string {
	pattern := intake.Diet
	if pattern != "" {
		pattern = strings.ToUpper(pattern[:1]) + pattern[1:]
	}
	return fmt.Sprintf(
		"%s pattern with a %d kcal maintenance estimate, targeting %d kcal per day across %d meals and %d snacks.",
		pattern, maintenance, target, meals, snacks,
	)
}

// fingerprintPlan derives a stable digest of the plan body together with the
// intake it came from. Including the intake means a plan regenerated from an
// unchanged intake reproduces the same fingerprint, while a plan derived from a
// changed intake does not.
func fingerprintPlan(intake *nutrition_questionnaire.Normalized, plan *GeneratedPlan) (string, error) {
	body := *plan
	body.Fingerprint = ""

	encodedPlan, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to encode nutrition plan: %w", err)
	}
	encodedIntake, err := intake.Marshal()
	if err != nil {
		return "", fmt.Errorf("failed to encode nutrition intake: %w", err)
	}

	digest := sha256.New()
	digest.Write(encodedPlan)
	digest.Write(encodedIntake)
	return hex.EncodeToString(digest.Sum(nil)), nil
}

// percentOf returns value as a whole percentage of total, guarding against a
// zero total.
func percentOf(value, total int) int {
	if total <= 0 {
		return 0
	}
	return int(math.Round(float64(value) / float64(total) * 100))
}

// abs returns the absolute value of an int.
func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
