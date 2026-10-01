// Package nutrition_questionnaire defines the Premium Level 1 intake contract:
// the closed set of questions a client answers before checkout, the rules that
// validate an answer, and the normalized document the server persists.
//
// The vocabulary is deliberately closed and server-owned. A client may only pick
// from the options this package declares, so an unknown or invented value can
// never reach the nutrition generator. The normalized document is the only
// shape that is stored, which is what keeps raw client input out of the
// database and out of the logs.
package nutrition_questionnaire

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// SchemaVersion identifies the intake contract version. It is persisted with
// every questionnaire so a later change to the question set can be recognized
// and migrated explicitly instead of being reinterpreted in place.
const SchemaVersion = 1

// ErrNotSubmitted indicates the owner has no stored intake for the program. It
// lives in this package so the questionnaire service and the nutrition
// generation service share the precondition without depending on each other.
var ErrNotSubmitted = errors.New("nutrition questionnaire not submitted")

// StoredIntake is a stored intake together with the identity of the row it came
// from. The questionnaire id and revision travel with the answers because a
// generated plan must record exactly which intake revision it was derived from.
type StoredIntake struct {
	// QuestionnaireID identifies the persisted questionnaire row.
	QuestionnaireID string `json:"questionnaire_id"`
	// Version is the stored intake revision, incremented on every resubmission.
	Version int `json:"version"`
	// Intake is the validated, normalized document.
	Intake *Normalized `json:"intake"`
}

// Question identifiers. These are the stable keys of the normalized document
// and are part of the persisted contract.
const (
	FieldGoal              = "goal"
	FieldExperience        = "experience"
	FieldTrainingDays      = "training_days_per_week"
	FieldSessionLength     = "session_length_minutes"
	FieldDiet              = "diet"
	FieldAllergies         = "allergies"
	FieldIntolerances      = "intolerances"
	FieldExcludedFoods     = "excluded_foods"
	FieldDislikedFoods     = "disliked_foods"
	FieldMealsPerDay       = "meals_per_day"
	FieldSnacksPerDay      = "snacks_per_day"
	FieldCookingEffort     = "cooking_effort"
	FieldBodyWeightKg      = "body_weight_kg"
	FieldBodyHeightCm      = "body_height_cm"
	FieldActivityLevel     = "activity_level"
	FieldSleepHours        = "sleep_hours"
	FieldStressLevel       = "stress_level"
	FieldMedicalConditions = "medical_conditions"
	FieldInjuries          = "injuries"
	FieldLimitations       = "limitations"
	FieldNotes             = "notes"
)

// Option vocabularies. A closed set per field keeps validation exhaustive and
// lets the generator switch on values it fully understands.
var (
	goalOptions = []string{
		"fat_loss",
		"muscle_gain",
		"strength",
		"endurance",
		"general_fitness",
		"recomposition",
		"mobility",
	}
	experienceOptions = []string{
		"beginner",
		"intermediate",
		"advanced",
	}
	dietOptions = []string{
		"omnivore",
		"flexitarian",
		"pescatarian",
		"vegetarian",
		"vegan",
		"halal",
		"kosher",
	}
	activityLevelOptions = []string{
		"sedentary",
		"light",
		"moderate",
		"high",
	}
	stressLevelOptions = []string{
		"low",
		"moderate",
		"high",
	}
	cookingEffortOptions = []string{
		"minimal",
		"moderate",
		"extensive",
	}
)

// Numeric bounds. They are exported because the API contract documents the
// accepted ranges to the client and the tests assert the same limits.
const (
	MinTrainingDaysPerWeek = 1
	MaxTrainingDaysPerWeek = 7

	MinSessionLengthMinutes = 15
	MaxSessionLengthMinutes = 180

	MinMealsPerDay  = 1
	MaxMealsPerDay  = 8
	MinSnacksPerDay = 0
	MaxSnacksPerDay = 6

	MinBodyWeightKg = 25.0
	MaxBodyWeightKg = 350.0

	MinBodyHeightCm = 100
	MaxBodyHeightCm = 260

	MinSleepHours = 0.0
	MaxSleepHours = 14.0
)

// Text field bounds. Sensitive free text is length-capped and stripped of
// control characters before it is stored, so the database never holds a
// multi-kilobyte answer and never holds terminal escape sequences.
const (
	MaxListItemLength   = 80
	MaxListItems        = 20
	MaxNotesLength      = 1000
	MaxConditionLength  = 120
	MaxConditionEntries = 10
)

// Answers is the raw client submission. Every field is a pointer so an omitted
// field is distinguishable from a zero value, which is what allows required-field
// validation to be exact rather than guessed.
type Answers struct {
	Goal              *string   `json:"goal"`
	Experience        *string   `json:"experience"`
	TrainingDays      *int      `json:"training_days_per_week"`
	SessionLength     *int      `json:"session_length_minutes"`
	Diet              *string   `json:"diet"`
	Allergies         *[]string `json:"allergies"`
	Intolerances      *[]string `json:"intolerances"`
	ExcludedFoods     *[]string `json:"excluded_foods"`
	DislikedFoods     *[]string `json:"disliked_foods"`
	MealsPerDay       *int      `json:"meals_per_day"`
	SnacksPerDay      *int      `json:"snacks_per_day"`
	CookingEffort     *string   `json:"cooking_effort"`
	BodyWeightKg      *float64  `json:"body_weight_kg"`
	BodyHeightCm      *int      `json:"body_height_cm"`
	ActivityLevel     *string   `json:"activity_level"`
	SleepHours        *float64  `json:"sleep_hours"`
	StressLevel       *string   `json:"stress_level"`
	MedicalConditions *[]string `json:"medical_conditions"`
	Injuries          *[]string `json:"injuries"`
	Limitations       *[]string `json:"limitations"`
	Notes             *string   `json:"notes"`
}

// Question describes one field of the intake contract. The frontend renders the
// questionnaire from the catalog the API returns, so the two can never drift
// apart.
type Question struct {
	Field      string   `json:"field"`
	Label      string   `json:"label"`
	Type       string   `json:"type"`
	Required   bool     `json:"required"`
	Options    []string `json:"options,omitempty"`
	Min        *float64 `json:"min,omitempty"`
	Max        *float64 `json:"max,omitempty"`
	MaxLength  int      `json:"max_length,omitempty"`
	MaxEntries int      `json:"max_entries,omitempty"`
	Sensitive  bool     `json:"sensitive,omitempty"`
	Help       string   `json:"help,omitempty"`
}

// Catalog returns the ordered question catalog. Order is the presentation order
// on the client; the server never depends on it for correctness.
func Catalog() []Question {
	number := func(v float64) *float64 { return &v }
	days := []string{"1", "2", "3", "4", "5", "6", "7"}

	return []Question{
		{Field: FieldGoal, Label: "What is your main goal?", Type: "select", Required: true, Options: goalOptions, Help: "This drives the calorie and protein targets."},
		{Field: FieldExperience, Label: "How would you describe your training experience?", Type: "select", Required: true, Options: experienceOptions},
		{Field: FieldTrainingDays, Label: "How many training days per week can you train?", Type: "select", Required: true, Options: days},
		{Field: FieldSessionLength, Label: "How long is a typical training session?", Type: "number", Required: true, Min: number(MinSessionLengthMinutes), Max: number(MaxSessionLengthMinutes), Help: "Minutes per session."},
		{Field: FieldDiet, Label: "Which diet best describes you?", Type: "select", Required: true, Options: dietOptions},
		{Field: FieldAllergies, Label: "Do you have any food allergies?", Type: "list", Required: false, MaxEntries: MaxListItems, MaxLength: MaxListItemLength, Sensitive: true, Help: "Allergies are treated as strict exclusions."},
		{Field: FieldIntolerances, Label: "Any food intolerances?", Type: "list", Required: false, MaxEntries: MaxListItems, MaxLength: MaxListItemLength, Sensitive: true},
		{Field: FieldExcludedFoods, Label: "Foods you strictly exclude for religious or ethical reasons?", Type: "list", Required: false, MaxEntries: MaxListItems, MaxLength: MaxListItemLength},
		{Field: FieldDislikedFoods, Label: "Foods you strongly dislike?", Type: "list", Required: false, MaxEntries: MaxListItems, MaxLength: MaxListItemLength, Help: "These are swapped for equivalents, not removed."},
		{Field: FieldMealsPerDay, Label: "How many meals per day?", Type: "number", Required: true, Min: number(MinMealsPerDay), Max: number(MaxMealsPerDay)},
		{Field: FieldSnacksPerDay, Label: "How many snacks per day?", Type: "number", Required: false, Min: number(MinSnacksPerDay), Max: number(MaxSnacksPerDay)},
		{Field: FieldCookingEffort, Label: "How much time can you spend cooking?", Type: "select", Required: true, Options: cookingEffortOptions},
		{Field: FieldBodyWeightKg, Label: "What is your body weight?", Type: "number", Required: true, Min: number(MinBodyWeightKg), Max: number(MaxBodyWeightKg), Sensitive: true, Help: "Kilograms. Used only to compute energy targets."},
		{Field: FieldBodyHeightCm, Label: "What is your height?", Type: "number", Required: true, Min: number(MinBodyHeightCm), Max: number(MaxBodyHeightCm), Sensitive: true, Help: "Centimetres."},
		{Field: FieldActivityLevel, Label: "How active are you outside of training?", Type: "select", Required: true, Options: activityLevelOptions},
		{Field: FieldSleepHours, Label: "How many hours do you sleep on average?", Type: "number", Required: true, Min: number(MinSleepHours), Max: number(MaxSleepHours)},
		{Field: FieldStressLevel, Label: "How would you rate your current stress?", Type: "select", Required: true, Options: stressLevelOptions},
		{Field: FieldMedicalConditions, Label: "Any medical conditions we should be aware of?", Type: "list", Required: false, MaxEntries: MaxConditionEntries, MaxLength: MaxConditionLength, Sensitive: true, Help: "Reviewed before your plan is generated. Not a medical assessment."},
		{Field: FieldInjuries, Label: "Any current injuries or pain?", Type: "list", Required: false, MaxEntries: MaxConditionEntries, MaxLength: MaxConditionLength, Sensitive: true},
		{Field: FieldLimitations, Label: "Any equipment or movement limitations?", Type: "list", Required: false, MaxEntries: MaxListItems, MaxLength: MaxListItemLength},
		{Field: FieldNotes, Label: "Anything else we should know?", Type: "textarea", Required: false, MaxLength: MaxNotesLength, Sensitive: true},
	}
}

// Normalized is the canonical, validated document that is persisted and read
// back by the nutrition generator. It is a flat, fully-resolved struct: every
// required field is present, every optional list is non-nil (possibly empty) and
// every text value has already been trimmed and control-stripped. Marshalling it
// is deterministic because the struct field order is fixed.
type Normalized struct {
	SchemaVersion int `json:"schema_version"`

	Goal              string   `json:"goal"`
	Experience        string   `json:"experience"`
	TrainingDays      int      `json:"training_days_per_week"`
	SessionLength     int      `json:"session_length_minutes"`
	Diet              string   `json:"diet"`
	Allergies         []string `json:"allergies"`
	Intolerances      []string `json:"intolerances"`
	ExcludedFoods     []string `json:"excluded_foods"`
	DislikedFoods     []string `json:"disliked_foods"`
	MealsPerDay       int      `json:"meals_per_day"`
	SnacksPerDay      int      `json:"snacks_per_day"`
	CookingEffort     string   `json:"cooking_effort"`
	BodyWeightKg      float64  `json:"body_weight_kg"`
	BodyHeightCm      int      `json:"body_height_cm"`
	ActivityLevel     string   `json:"activity_level"`
	SleepHours        float64  `json:"sleep_hours"`
	StressLevel       string   `json:"stress_level"`
	MedicalConditions []string `json:"medical_conditions"`
	Injuries          []string `json:"injuries"`
	Limitations       []string `json:"limitations"`
	Notes             string   `json:"notes"`
}

// HasMedicalFlags reports whether the intake carries a medical condition or an
// active injury. The generated plan surfaces a caution note in this case instead
// of presenting itself as suitable for that client without review.
func (n Normalized) HasMedicalFlags() bool {
	return len(n.MedicalConditions) > 0 || len(n.Injuries) > 0
}

// FieldError names the offending field and the reason. It is part of the
// validation error so the API can point the client at the exact question, while
// never echoing the submitted value back.
type FieldError struct {
	Field  string
	Reason string
}

func (e FieldError) Error() string {
	return fmt.Sprintf("field %q: %s", e.Field, e.Reason)
}

// ValidationError aggregates every invalid field so the client can correct the
// whole submission in one pass instead of one field per round trip. It carries
// field names and reasons only, never submitted values.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	return "questionnaire validation failed"
}

// FieldErrors returns the per-field reasons keyed by field name, which is the
// shape the API contract documents for a rejected submission.
func (e *ValidationError) FieldErrors() map[string]string {
	out := make(map[string]string, len(e.Fields))
	for _, field := range e.Fields {
		out[field.Field] = field.Reason
	}
	return out
}

// Normalize validates a raw submission and converts it into the canonical
// document. It returns a *ValidationError listing every problem, so a malformed
// submission is rejected in full rather than partially accepted.
//
// Rejection is always a validation error and never a partially-stored intake:
// nothing is persisted unless the whole document is valid.
func Normalize(raw Answers) (*Normalized, error) {
	invalid := &ValidationError{}

	normalized := &Normalized{
		SchemaVersion:     SchemaVersion,
		Allergies:         []string{},
		Intolerances:      []string{},
		ExcludedFoods:     []string{},
		DislikedFoods:     []string{},
		MedicalConditions: []string{},
		Injuries:          []string{},
		Limitations:       []string{},
	}

	normalized.Goal = requiredOption(invalid, FieldGoal, raw.Goal, goalOptions)
	normalized.Experience = requiredOption(invalid, FieldExperience, raw.Experience, experienceOptions)
	normalized.Diet = requiredOption(invalid, FieldDiet, raw.Diet, dietOptions)
	normalized.ActivityLevel = requiredOption(invalid, FieldActivityLevel, raw.ActivityLevel, activityLevelOptions)
	normalized.StressLevel = requiredOption(invalid, FieldStressLevel, raw.StressLevel, stressLevelOptions)
	normalized.CookingEffort = requiredOption(invalid, FieldCookingEffort, raw.CookingEffort, cookingEffortOptions)

	// Training days are collected as a select on the client but stored as a
	// count, so it is validated as a bounded integer instead of an option.
	normalized.TrainingDays = requiredInt(invalid, FieldTrainingDays, raw.TrainingDays, MinTrainingDaysPerWeek, MaxTrainingDaysPerWeek)
	normalized.SessionLength = requiredInt(invalid, FieldSessionLength, raw.SessionLength, MinSessionLengthMinutes, MaxSessionLengthMinutes)
	normalized.MealsPerDay = requiredInt(invalid, FieldMealsPerDay, raw.MealsPerDay, MinMealsPerDay, MaxMealsPerDay)
	normalized.SnacksPerDay = optionalInt(invalid, FieldSnacksPerDay, raw.SnacksPerDay, MinSnacksPerDay, MaxSnacksPerDay)
	normalized.BodyHeightCm = requiredInt(invalid, FieldBodyHeightCm, raw.BodyHeightCm, MinBodyHeightCm, MaxBodyHeightCm)

	normalized.BodyWeightKg = requiredFloat(invalid, FieldBodyWeightKg, raw.BodyWeightKg, MinBodyWeightKg, MaxBodyWeightKg)
	normalized.SleepHours = requiredFloat(invalid, FieldSleepHours, raw.SleepHours, MinSleepHours, MaxSleepHours)

	normalized.Allergies = optionalList(invalid, FieldAllergies, raw.Allergies, MaxListItems, MaxListItemLength)
	normalized.Intolerances = optionalList(invalid, FieldIntolerances, raw.Intolerances, MaxListItems, MaxListItemLength)
	normalized.ExcludedFoods = optionalList(invalid, FieldExcludedFoods, raw.ExcludedFoods, MaxListItems, MaxListItemLength)
	normalized.DislikedFoods = optionalList(invalid, FieldDislikedFoods, raw.DislikedFoods, MaxListItems, MaxListItemLength)
	normalized.MedicalConditions = optionalList(invalid, FieldMedicalConditions, raw.MedicalConditions, MaxConditionEntries, MaxConditionLength)
	normalized.Injuries = optionalList(invalid, FieldInjuries, raw.Injuries, MaxConditionEntries, MaxConditionLength)
	normalized.Limitations = optionalList(invalid, FieldLimitations, raw.Limitations, MaxListItems, MaxListItemLength)

	normalized.Notes = optionalText(invalid, FieldNotes, raw.Notes, MaxNotesLength)

	if len(invalid.Fields) > 0 {
		return nil, invalid
	}
	return normalized, nil
}

// requiredOption validates a required closed-vocabulary field.
func requiredOption(invalid *ValidationError, field string, value *string, allowed []string) string {
	if value == nil {
		invalid.Fields = append(invalid.Fields, FieldError{Field: field, Reason: "required"})
		return ""
	}
	trimmed := strings.ToLower(strings.TrimSpace(*value))
	if !contains(allowed, trimmed) {
		invalid.Fields = append(invalid.Fields, FieldError{Field: field, Reason: "unsupported value"})
		return ""
	}
	return trimmed
}

// requiredInt validates a required bounded integer.
func requiredInt(invalid *ValidationError, field string, value *int, min, max int) int {
	if value == nil {
		invalid.Fields = append(invalid.Fields, FieldError{Field: field, Reason: "required"})
		return 0
	}
	if *value < min || *value > max {
		invalid.Fields = append(invalid.Fields, FieldError{Field: field, Reason: "out of range"})
		return 0
	}
	return *value
}

// optionalInt validates an optional bounded integer, defaulting to zero.
func optionalInt(invalid *ValidationError, field string, value *int, min, max int) int {
	if value == nil {
		return 0
	}
	if *value < min || *value > max {
		invalid.Fields = append(invalid.Fields, FieldError{Field: field, Reason: "out of range"})
		return 0
	}
	return *value
}

// requiredFloat validates a required bounded number.
func requiredFloat(invalid *ValidationError, field string, value *float64, min, max float64) float64 {
	if value == nil {
		invalid.Fields = append(invalid.Fields, FieldError{Field: field, Reason: "required"})
		return 0
	}
	if *value < min || *value > max {
		invalid.Fields = append(invalid.Fields, FieldError{Field: field, Reason: "out of range"})
		return 0
	}
	return *value
}

// optionalList validates an optional list of free-text entries: blank entries
// are dropped, duplicates are collapsed, entries are trimmed and stripped of
// control characters, and both the count and the per-entry length are capped.
func optionalList(invalid *ValidationError, field string, values *[]string, maxEntries, maxLength int) []string {
	if values == nil {
		return []string{}
	}
	if len(*values) > maxEntries {
		invalid.Fields = append(invalid.Fields, FieldError{Field: field, Reason: "too many entries"})
		return []string{}
	}

	out := make([]string, 0, len(*values))
	seen := make(map[string]struct{}, len(*values))
	for _, entry := range *values {
		cleaned := sanitizeText(entry)
		if cleaned == "" {
			continue
		}
		if len([]rune(cleaned)) > maxLength {
			invalid.Fields = append(invalid.Fields, FieldError{Field: field, Reason: "entry too long"})
			return []string{}
		}
		key := strings.ToLower(cleaned)
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, cleaned)
	}
	sort.Strings(out)
	return out
}

// optionalText validates an optional free-text field and returns it cleaned. An
// empty result is stored as an empty string rather than null, so the persisted
// document always has the same shape.
func optionalText(invalid *ValidationError, field string, value *string, maxLength int) string {
	if value == nil {
		return ""
	}
	cleaned := sanitizeText(*value)
	if len([]rune(cleaned)) > maxLength {
		invalid.Fields = append(invalid.Fields, FieldError{Field: field, Reason: "too long"})
		return ""
	}
	return cleaned
}

// sanitizeText trims surrounding whitespace and removes control characters so a
// stored answer can never contain a terminal escape sequence or a stray
// newline that would corrupt downstream rendering.
func sanitizeText(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for _, r := range value {
		switch {
		case r == '\t':
			builder.WriteRune(' ')
		case unicode.IsControl(r):
			continue
		default:
			builder.WriteRune(r)
		}
	}
	return strings.TrimSpace(builder.String())
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// Marshal renders the normalized document for persistence. Marshalling a fixed
// struct yields a stable key order, so the stored bytes are byte-identical for
// equal inputs, which is a precondition for the deterministic generator.
func (n Normalized) Marshal() ([]byte, error) {
	encoded, err := json.Marshal(n)
	if err != nil {
		return nil, fmt.Errorf("failed to encode normalized questionnaire: %w", err)
	}
	return encoded, nil
}

// Unmarshal decodes a persisted normalized document back into its struct form.
func Unmarshal(raw []byte) (*Normalized, error) {
	var normalized Normalized
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return nil, fmt.Errorf("failed to decode stored questionnaire: %w", err)
	}
	// A stored document from an older contract version is not reinterpreted in
	// place; it must be migrated explicitly.
	if normalized.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("unsupported questionnaire schema version %d", normalized.SchemaVersion)
	}
	return &normalized, nil
}
