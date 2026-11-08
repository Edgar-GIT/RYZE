package premium_level_2

// Question types understood by the client renderer and by the server-side
// validator. They are deliberately generic so the intake can evolve without a
// client release.
const (
	QuestionTypeText        = "text"
	QuestionTypeTextarea    = "textarea"
	QuestionTypeNumber      = "number"
	QuestionTypeSelect      = "select"
	QuestionTypeMultiSelect = "multiselect"
	QuestionTypeBoolean     = "boolean"
)

const (
	CatalogVersion     = 1
	MaxAnswerLength    = 4000
	MaxShortTextLength = 160
)

// Option is one closed answer choice.
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Question is one server-declared intake field. Min/Max/Step only apply to
// number questions; Options only apply to select and multiselect questions.
type Question struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Help        string   `json:"help,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Options     []Option `json:"options,omitempty"`
	Min         *float64 `json:"min,omitempty"`
	Max         *float64 `json:"max,omitempty"`
	Step        *float64 `json:"step,omitempty"`
	MaxLength   int      `json:"max_length,omitempty"`
}

// Section groups related questions in the client form.
type Section struct {
	Key         string     `json:"key"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Questions   []Question `json:"questions"`
}

// Catalog is the complete, versioned Premium Level 2 intake definition.
type Catalog struct {
	Version  int       `json:"version"`
	Sections []Section `json:"sections"`
}

func float(v float64) *float64 { return &v }

func opts(values ...string) []Option {
	result := make([]Option, 0, len(values))
	for _, value := range values {
		result = append(result, Option{Value: value, Label: humanize(value)})
	}
	return result
}

func humanize(value string) string {
	switch value {
	case "fat_loss":
		return "Fat loss"
	case "muscle_gain":
		return "Muscle gain"
	case "recomposition":
		return "Body recomposition"
	case "athletic_performance":
		return "Athletic performance"
	case "general_health":
		return "General health"
	case "strength":
		return "Strength"
	case "beginner":
		return "Beginner"
	case "intermediate":
		return "Intermediate"
	case "advanced":
		return "Advanced"
	case "no_deadline":
		return "No fixed deadline"
	case "8_weeks":
		return "8 weeks"
	case "12_weeks":
		return "12 weeks"
	case "16_weeks":
		return "16 weeks"
	case "24_weeks":
		return "24 weeks"
	case "gym":
		return "Full gym"
	case "home":
		return "Home"
	case "hybrid":
		return "Home and gym"
	case "outdoors":
		return "Outdoors"
	case "full_gym":
		return "Full gym equipment"
	case "dumbbells":
		return "Dumbbells"
	case "barbell_rack":
		return "Barbell and rack"
	case "resistance_bands":
		return "Resistance bands"
	case "pull_up_bar":
		return "Pull-up bar"
	case "cardio_machines":
		return "Cardio machines"
	case "kettlebells":
		return "Kettlebells"
	case "bodyweight_only":
		return "Bodyweight only"
	case "mon", "tue", "wed", "thu", "fri", "sat", "sun":
		return map[string]string{"mon": "Monday", "tue": "Tuesday", "wed": "Wednesday", "thu": "Thursday", "fri": "Friday", "sat": "Saturday", "sun": "Sunday"}[value]
	case "sedentary":
		return "Sedentary"
	case "light":
		return "Lightly active"
	case "moderate":
		return "Moderately active"
	case "high":
		return "Very active"
	case "very_high":
		return "Extremely active"
	case "low":
		return "Low"
	case "balanced":
		return "Balanced"
	case "vegetarian":
		return "Vegetarian"
	case "vegan":
		return "Vegan"
	case "pescatarian":
		return "Pescatarian"
	case "halal":
		return "Halal"
	case "kosher":
		return "Kosher"
	case "other":
		return "Other"
	case "minimal":
		return "Minimal preparation"
	case "20_30":
		return "20-30 minutes"
	case "30_60":
		return "30-60 minutes"
	case "60_plus":
		return "More than 60 minutes"
	case "flexible":
		return "Flexible"
	case "direct":
		return "Direct and challenging"
	case "supportive":
		return "Supportive and encouraging"
	case "weekly":
		return "Weekly"
	case "biweekly":
		return "Every two weeks"
	case "monthly":
		return "Monthly"
	case "not_pregnant":
		return "Not pregnant"
	case "pregnant":
		return "Pregnant"
	case "breastfeeding":
		return "Breastfeeding"
	case "prefer_not_to_say":
		return "Prefer not to say"
	default:
		return value
	}
}

// IntakeCatalog returns a fresh copy of the server-owned intake definition.
func IntakeCatalog() Catalog {
	return Catalog{Version: CatalogVersion, Sections: []Section{
		{
			Key:         "goals",
			Title:       "Goals",
			Description: "Tell your trainer what you want to achieve and how experienced you are.",
			Questions: []Question{
				{Key: "primary_goal", Label: "Primary goal", Type: QuestionTypeSelect, Required: true, Options: opts("fat_loss", "muscle_gain", "strength", "recomposition", "athletic_performance", "general_health")},
				{Key: "secondary_goals", Label: "Secondary goals", Type: QuestionTypeMultiSelect, Required: false, Options: opts("fat_loss", "muscle_gain", "strength", "recomposition", "athletic_performance", "general_health")},
				{Key: "training_experience", Label: "Training experience", Type: QuestionTypeSelect, Required: true, Options: opts("beginner", "intermediate", "advanced")},
				{Key: "target_timeline", Label: "Target timeline", Type: QuestionTypeSelect, Required: true, Options: opts("8_weeks", "12_weeks", "16_weeks", "24_weeks", "no_deadline")},
				{Key: "motivation", Label: "Why does this goal matter to you?", Type: QuestionTypeTextarea, Required: true, MaxLength: 1000},
			},
		},
		{
			Key:         "training",
			Title:       "Training",
			Description: "Your trainer needs to know how, where and when you can train.",
			Questions: []Question{
				{Key: "sessions_per_week", Label: "Training sessions per week", Type: QuestionTypeNumber, Required: true, Min: float(1), Max: float(7), Step: float(1)},
				{Key: "session_minutes", Label: "Minutes per session", Type: QuestionTypeNumber, Required: true, Min: float(20), Max: float(180), Step: float(5)},
				{Key: "training_location", Label: "Where will you train?", Type: QuestionTypeSelect, Required: true, Options: opts("gym", "home", "hybrid", "outdoors")},
				{Key: "equipment", Label: "Available equipment", Type: QuestionTypeMultiSelect, Required: false, Options: opts("full_gym", "dumbbells", "barbell_rack", "resistance_bands", "pull_up_bar", "cardio_machines", "kettlebells", "bodyweight_only")},
				{Key: "preferred_days", Label: "Preferred training days", Type: QuestionTypeMultiSelect, Required: true, Options: opts("mon", "tue", "wed", "thu", "fri", "sat", "sun")},
				{Key: "current_training", Label: "Current training routine", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
				{Key: "injuries", Label: "Injuries or pain", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000, Help: "Describe anything that limits your training. Write none if not applicable."},
				{Key: "movement_limitations", Label: "Movement limitations", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
				{Key: "exercise_preferences", Label: "Exercises you enjoy or want to avoid", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
			},
		},
		{
			Key:         "body",
			Title:       "Body measurements",
			Description: "Used to calibrate training and nutrition. Optional fields can be left blank.",
			Questions: []Question{
				{Key: "height_cm", Label: "Height (cm)", Type: QuestionTypeNumber, Required: true, Min: float(120), Max: float(230), Step: float(0.1)},
				{Key: "weight_kg", Label: "Weight (kg)", Type: QuestionTypeNumber, Required: true, Min: float(35), Max: float(250), Step: float(0.1)},
				{Key: "goal_weight_kg", Label: "Goal weight (kg)", Type: QuestionTypeNumber, Required: false, Min: float(35), Max: float(250), Step: float(0.1)},
				{Key: "body_fat_percent", Label: "Body fat (%)", Type: QuestionTypeNumber, Required: false, Min: float(3), Max: float(60), Step: float(0.1)},
				{Key: "waist_cm", Label: "Waist (cm)", Type: QuestionTypeNumber, Required: false, Min: float(40), Max: float(200), Step: float(0.1)},
				{Key: "body_notes", Label: "Anything else about your body or measurements", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
			},
		},
		{
			Key:         "lifestyle",
			Title:       "Lifestyle",
			Description: "Recovery and daily activity shape the plan as much as training does.",
			Questions: []Question{
				{Key: "occupation", Label: "Occupation", Type: QuestionTypeText, Required: true, MaxLength: MaxShortTextLength},
				{Key: "activity_level", Label: "Daily activity level", Type: QuestionTypeSelect, Required: true, Options: opts("sedentary", "light", "moderate", "high", "very_high")},
				{Key: "sleep_hours", Label: "Average sleep per night (hours)", Type: QuestionTypeNumber, Required: true, Min: float(3), Max: float(14), Step: float(0.5)},
				{Key: "stress_level", Label: "Stress level", Type: QuestionTypeSelect, Required: true, Options: opts("low", "moderate", "high", "very_high")},
				{Key: "daily_steps", Label: "Average daily steps", Type: QuestionTypeNumber, Required: false, Min: float(0), Max: float(50000), Step: float(500)},
				{Key: "schedule_notes", Label: "Schedule or recovery notes", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
			},
		},
		{
			Key:         "nutrition",
			Title:       "Nutrition",
			Description: "Your trainer will build an individual plan around these answers.",
			Questions: []Question{
				{Key: "dietary_pattern", Label: "Dietary pattern", Type: QuestionTypeSelect, Required: true, Options: opts("balanced", "vegetarian", "vegan", "pescatarian", "halal", "kosher", "other")},
				{Key: "meals_per_day", Label: "Meals per day", Type: QuestionTypeNumber, Required: true, Min: float(2), Max: float(8), Step: float(1)},
				{Key: "allergies", Label: "Allergies", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000, Help: "Write none if not applicable."},
				{Key: "intolerances", Label: "Intolerances", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
				{Key: "disliked_foods", Label: "Foods you dislike", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
				{Key: "preferred_foods", Label: "Foods you enjoy", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
				{Key: "cooking_time", Label: "Time available to cook", Type: QuestionTypeSelect, Required: true, Options: opts("minimal", "20_30", "30_60", "60_plus")},
				{Key: "budget_level", Label: "Food budget", Type: QuestionTypeSelect, Required: true, Options: opts("low", "moderate", "flexible")},
				{Key: "supplements", Label: "Supplements you use", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
				{Key: "nutrition_notes", Label: "Anything else about your nutrition", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
			},
		},
		{
			Key:         "health",
			Title:       "Health",
			Description: "Health answers are private and only used to keep your plan safe.",
			Questions: []Question{
				{Key: "medical_conditions", Label: "Medical conditions", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000, Help: "Write none if not applicable."},
				{Key: "medications", Label: "Medications", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
				{Key: "medical_clearance", Label: "Has a doctor cleared you for exercise?", Type: QuestionTypeBoolean, Required: true},
				{Key: "pregnancy_status", Label: "Pregnancy status", Type: QuestionTypeSelect, Required: false, Options: opts("not_pregnant", "pregnant", "breastfeeding", "prefer_not_to_say")},
				{Key: "eating_disorder_history", Label: "History of an eating disorder", Type: QuestionTypeBoolean, Required: false},
				{Key: "health_notes", Label: "Other health information", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
			},
		},
		{
			Key:         "preferences",
			Title:       "Coaching preferences",
			Description: "Tell your trainer how you prefer to work together.",
			Questions: []Question{
				{Key: "coaching_style", Label: "Preferred coaching style", Type: QuestionTypeSelect, Required: true, Options: opts("direct", "supportive", "balanced")},
				{Key: "checkin_frequency", Label: "Preferred check-in frequency", Type: QuestionTypeSelect, Required: true, Options: opts("weekly", "biweekly", "monthly")},
				{Key: "previous_coaching", Label: "Previous coaching experience", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
				{Key: "additional_notes", Label: "Anything else your trainer should know", Type: QuestionTypeTextarea, Required: false, MaxLength: 2000},
			},
		},
	}}
}

func questionByKey(catalog Catalog) map[string]Question {
	questions := map[string]Question{}
	for _, section := range catalog.Sections {
		for _, question := range section.Questions {
			questions[question.Key] = question
		}
	}
	return questions
}
