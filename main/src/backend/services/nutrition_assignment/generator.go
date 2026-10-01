// Package nutrition_assignment generates the server-owned nutrition plan for a
// purchased Premium Level 1 program.
//
// The generator is a deterministic function of the validated intake: the same
// intake always produces the same configuration, byte for byte. That property is
// what makes the pipeline idempotent — a retry after a failure recomputes an
// identical plan instead of drifting, and Test Mode exercises the exact code path
// a real payment takes.
//
// Generator is the replaceable seam. It is an interface so a future provider can
// be introduced without touching the purchase, entitlement or API layers, and so
// the current deterministic implementation can be tested in isolation with no
// external dependency and no network access.
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

	"ryze/backend/services/nutrition_questionnaire"
)

// Version identifies the generation contract. It is stored on the assignment so
// a configuration produced by an older contract is recognizable and can be
// regenerated deliberately rather than being silently reinterpreted.
const Version = 1

// GeneratedPlan is the server-owned nutrition configuration. It is derived purely
// from the intake: nothing here is client-supplied, and no field is editable by
// the client after generation.
type GeneratedPlan struct {
	Version int `json:"version"`
	// Fingerprint is derived from the intake and the plan body. It lets the
	// access surface detect that the stored plan no longer matches the current
	// intake revision without comparing the whole document.
	Fingerprint string `json:"fingerprint"`

	EnergyTargets  EnergyTargets  `json:"energy_targets"`
	Macronutrients Macronutrients `json:"macronutrients"`
	MealPlan       MealPlan       `json:"meal_plan"`
	Hydration      Hydration      `json:"hydration"`
	// DietaryRules are the constraints the plan was built under. They are shown
	// back to the client so the exclusions driving the plan are visible rather
	// than implicit.
	DietaryRules DietaryRules `json:"dietary_rules"`
	// Cautions are non-prescriptive safety notes. They appear whenever the
	// intake reports a medical condition or an active injury, because the plan
	// is a general nutrition template and is not a medical or clinical
	// assessment.
	Cautions []string `json:"cautions"`
	// Summary is a short human-readable description of the plan.
	Summary string `json:"summary"`
}

// EnergyTargets are the daily energy and fluid targets. They are rounded to whole
// units so the stored plan is stable and easy to reason about.
type EnergyTargets struct {
	MaintenanceCalories int `json:"maintenance_calories"`
	TargetCalories      int `json:"target_calories"`
	DailyProteinGrams   int `json:"daily_protein_grams"`
	DailyCarbsGrams     int `json:"daily_carbs_grams"`
	DailyFatGrams       int `json:"daily_fat_grams"`
}

// Macronutrients is the daily macro split, expressed both in grams and as a
// percentage of target calories so the client can render it without recomputing.
type Macronutrients struct {
	ProteinGrams        int             `json:"protein_grams"`
	CarbohydrateGrams   int             `json:"carbohydrate_grams"`
	FatGrams            int             `json:"fat_grams"`
	FiberGrams          int             `json:"fiber_grams"`
	ProteinPercent      int             `json:"protein_percent"`
	CarbohydratePercent int             `json:"carbohydrate_percent"`
	FatPercent          int             `json:"fat_percent"`
	CaloriesPerGram     CaloriesPerGram `json:"calories_per_gram"`
}

// CaloriesPerGram documents the energy density constants the plan was computed
// with, so the arithmetic behind the targets is auditable by the client.
type CaloriesPerGram struct {
	Protein      int `json:"protein"`
	Carbohydrate int `json:"carbohydrate"`
	Fat          int `json:"fat"`
}

// MealPlan describes the daily structure derived from the intake.
type MealPlan struct {
	MealsPerDay  int        `json:"meals_per_day"`
	SnacksPerDay int        `json:"snacks_per_day"`
	Distribution []MealSlot `json:"distribution"`
	PrepGuidance string     `json:"prep_guidance"`
}

// MealSlot is one eating occasion with its share of the daily calories.
type MealSlot struct {
	Label              string `json:"label"`
	PercentOfDaily     int    `json:"percent_of_daily"`
	ApproxCalories     int    `json:"approx_calories"`
	ApproxProteinGrams int    `json:"approx_protein_grams"`
}

// Hydration is the daily fluid target.
type Hydration struct {
	DailyLitres float64 `json:"daily_litres"`
	Note        string  `json:"note"`
}

// DietaryRules records the intake constraints the plan honours.
type DietaryRules struct {
	Pattern           string   `json:"pattern"`
	ExcludedFoods     []string `json:"excluded_foods"`
	DislikedFoods     []string `json:"disliked_foods"`
	Allergies         []string `json:"allergies"`
	Intolerances      []string `json:"intolerances"`
	MedicalConditions []string `json:"medical_conditions"`
	Injuries          []string `json:"injuries"`
	Limitations       []string `json:"limitations"`
}

// Generator is the replaceable generation seam. Implementations receive the
// validated intake and return a plan, or a failure that is recorded and
// retryable. A Generator must be deterministic for a given intake: that is the
// contract the idempotent retry path depends on.
type Generator interface {
	Generate(intake *nutrition_questionnaire.Normalized) (*GeneratedPlan, error)
}

// DeterministicGenerator is the built-in Generator. It applies fixed,
// documented arithmetic to the intake and has no external dependency, so a plan
// can always be produced offline and in Test Mode.
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
	1: 150,
	2: 250,
	3: 350,
	4: 420,
	5: 500,
	6: 550,
	7: 600,
}

// Goal calorie adjustments applied to maintenance energy.
var (
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
		"beginner":     0.0,
		"intermediate": 0.1,
		"advanced":     0.2,
	}
	stressProteinBonus = map[string]float64{
		"low":      0.0,
		"moderate": 0.0,
		"high":     0.1,
	}
)

// Fat floors as a share of total calories, applied so a high-protein target can
// never push fat below a safe minimum.
const (
	minFatPercent    = 0.25
	fatFloorGrams    = 45
	fiberPer1000kcal = 14
)

// Generate derives the plan from the intake. The arithmetic is fully determined
// by the intake fields: identical inputs always yield identical output.
func (DeterministicGenerator) Generate(intake *nutrition_questionnaire.Normalized) (*GeneratedPlan, error) {
	if intake == nil {
		return nil, errors.New("nutrition intake is required")
	}

	bmr := mifflinStJeor(intake)
	multiplier, ok := activityMultipliers[intake.ActivityLevel]
	if !ok {
		return nil, fmt.Errorf("unsupported activity level %q", intake.ActivityLevel)
	}
	maintenance := int(math.Round(bmr * multiplier))
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

	plan := &GeneratedPlan{
		Version: Version,
		EnergyTargets: EnergyTargets{
			MaintenanceCalories: maintenance,
			TargetCalories:      target,
			DailyProteinGrams:   proteinGrams,
			DailyCarbsGrams:     carbsGrams,
			DailyFatGrams:       fatGrams,
		},
		Macronutrients: Macronutrients{
			ProteinGrams:        proteinGrams,
			CarbohydrateGrams:   carbsGrams,
			FatGrams:            fatGrams,
			FiberGrams:          fiberGrams,
			ProteinPercent:      percentOf(proteinGrams*proteinKcalPerGram, target),
			CarbohydratePercent: percentOf(carbsGrams*carbohydrateKcalPerGram, target),
			FatPercent:          percentOf(fatGrams*fatKcalPerGram, target),
			CaloriesPerGram: CaloriesPerGram{
				Protein:      proteinKcalPerGram,
				Carbohydrate: carbohydrateKcalPerGram,
				Fat:          fatKcalPerGram,
			},
		},
		MealPlan:  buildMealPlan(intake, target, proteinGrams),
		Hydration: buildHydration(intake),
		DietaryRules: DietaryRules{
			Pattern:           intake.Diet,
			ExcludedFoods:     nonNilStrings(intake.ExcludedFoods),
			DislikedFoods:     nonNilStrings(intake.DislikedFoods),
			Allergies:         nonNilStrings(intake.Allergies),
			Intolerances:      nonNilStrings(intake.Intolerances),
			MedicalConditions: nonNilStrings(intake.MedicalConditions),
			Injuries:          nonNilStrings(intake.Injuries),
			Limitations:       nonNilStrings(intake.Limitations),
		},
		Cautions: buildCautions(intake),
		Summary:  buildSummary(intake, maintenance, target),
	}

	fingerprint, err := fingerprintPlan(intake, plan)
	if err != nil {
		return nil, err
	}
	plan.Fingerprint = fingerprint

	return plan, nil
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
	// A floor keeps maintenance and weight-loss targets from collapsing into an
	// implausibly low energy target for a small client.
	if target < minimumTargetCalories {
		target = minimumTargetCalories
	}
	return target
}

// minimumTargetCalories is the lowest daily energy target the generator emits.
// It is a conservative floor for a general template, not a clinical threshold.
const minimumTargetCalories = 1200

// buildMealPlan distributes the daily energy and protein across the eating
// occasions the intake describes. The shares always sum to exactly 100: each
// snack takes a fixed slice of the daily budget and the meals share whatever
// remains, so adding a snack can never inflate the total.
func buildMealPlan(intake *nutrition_questionnaire.Normalized, targetCalories, proteinGrams int) MealPlan {
	meals := intake.MealsPerDay
	if meals <= 0 {
		meals = 3
	}
	snacks := intake.SnacksPerDay
	if snacks < 0 {
		snacks = 0
	}
	// Clamp so the meal budget can never collapse to zero or below.
	if snacks > maxSnacksInPlan {
		snacks = maxSnacksInPlan
	}

	snackPercent := snackSharePercent
	mealPercentBudget := 100 - snacks*snackPercent
	mealPercents := distributeEvenly(mealPercentBudget, meals)

	slots := make([]MealSlot, 0, meals+snacks)
	labels := mealLabels(meals)
	for i, label := range labels {
		percent := mealPercents[i]
		slots = append(slots, MealSlot{
			Label:              label,
			PercentOfDaily:     percent,
			ApproxCalories:     int(math.Round(float64(targetCalories) * float64(percent) / 100)),
			ApproxProteinGrams: int(math.Round(float64(proteinGrams) * float64(percent) / 100)),
		})
	}
	for i := 0; i < snacks; i++ {
		slots = append(slots, MealSlot{
			Label:              "Snack",
			PercentOfDaily:     snackPercent,
			ApproxCalories:     int(math.Round(float64(targetCalories) * float64(snackPercent) / 100)),
			ApproxProteinGrams: int(math.Round(float64(proteinGrams) * float64(snackPercent) / 100)),
		})
	}

	return MealPlan{
		MealsPerDay:  meals,
		SnacksPerDay: snacks,
		Distribution: slots,
		PrepGuidance: prepGuidance(intake),
	}
}

// snackSharePercent is the share of the daily budget one snack occupies.
const snackSharePercent = 10

// maxSnacksInPlan bounds how much of the daily budget snacks may claim, leaving
// a meaningful share for the meals themselves.
const maxSnacksInPlan = 4

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
// supported range.
func mealLabels(meals int) []string {
	switch meals {
	case 1:
		return []string{"Daily plate"}
	case 2:
		return []string{"Breakfast", "Dinner"}
	case 3:
		return []string{"Breakfast", "Lunch", "Dinner"}
	case 4:
		return []string{"Breakfast", "Lunch", "Snack", "Dinner"}
	case 5:
		return []string{"Breakfast", "Morning snack", "Lunch", "Afternoon snack", "Dinner"}
	case 6:
		return []string{"Breakfast", "Morning snack", "Lunch", "Afternoon snack", "Dinner", "Evening snack"}
	case 7:
		return []string{"Breakfast", "Morning snack", "Lunch", "Afternoon snack", "Dinner", "Evening snack", "Supper"}
	case 8:
		return []string{"Breakfast", "Morning snack", "Lunch", "Afternoon snack", "Dinner", "Evening snack", "Supper", "Late snack"}
	default:
		// A fixed fallback keeps generation total for any stored value; the
		// validator already bounds this field.
		return []string{"Meal 1", "Meal 2", "Meal 3"}
	}
}

// prepGuidance derives batch-cooking guidance from the declared cooking effort.
func prepGuidance(intake *nutrition_questionnaire.Normalized) string {
	switch intake.CookingEffort {
	case "minimal":
		return "Favour ready-made and minimally processed staples. Batch-coach one protein source and one carbohydrate source per week, and keep frozen vegetables as the default vegetable."
	case "extensive":
		return "Cook fresh most days. Batch-coach two protein sources and two carbohydrate sources mid-week, and prep vegetables in a single session."
	default:
		return "Batch-coach one protein source and one carbohydrate source per week, and prep vegetables in a single session."
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
func buildCautions(intake *nutrition_questionnaire.Normalized) []string {
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
		cautions = append(cautions, "You reported allergies or intolerances. Confirm every ingredient label yourself; this plan cannot verify product contents.")
	}
	if intake.SleepHours < 6 {
		cautions = append(cautions, "Your reported sleep is below six hours. Sleep has a large effect on recovery and on how your body partitions energy; treat it as a priority alongside this plan.")
	}
	return cautions
}

// buildSummary renders a short, stable description of the plan.
func buildSummary(intake *nutrition_questionnaire.Normalized, maintenance, target int) string {
	return fmt.Sprintf(
		"%s pattern with a %d kcal maintenance estimate, targeting %d kcal per day, distributed across %d meals and %d snacks.",
		strings.ToUpper(intake.Diet[:1])+intake.Diet[1:],
		maintenance,
		target,
		intake.MealsPerDay,
		intake.SnacksPerDay,
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

// Marshal renders the plan for persistence. Marshalling a fixed struct yields a
// stable key order, so equal plans are stored byte-identical.
func (p *GeneratedPlan) Marshal() ([]byte, error) {
	encoded, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to encode nutrition plan: %w", err)
	}
	return encoded, nil
}

// Unmarshal decodes a stored plan back into its struct form.
func Unmarshal(raw []byte) (*GeneratedPlan, error) {
	var plan GeneratedPlan
	if err := json.Unmarshal(raw, &plan); err != nil {
		return nil, fmt.Errorf("failed to decode nutrition plan: %w", err)
	}
	if plan.Version != Version {
		return nil, fmt.Errorf("unsupported nutrition plan version %d", plan.Version)
	}
	return &plan, nil
}

// percentOf returns value as a whole percentage of total, guarding against a
// zero total.
func percentOf(value, total int) int {
	if total <= 0 {
		return 0
	}
	return int(math.Round(float64(value) / float64(total) * 100))
}

// nonNilStrings returns a non-nil copy so an empty list is serialized as an
// empty array rather than null.
func nonNilStrings(values []string) []string {
	out := append([]string{}, values...)
	sort.Strings(out)
	return out
}
