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
	CatalogVersion     = 2
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

func labeledOpts(labels map[string]string, values ...string) []Option {
	result := make([]Option, 0, len(values))
	for _, value := range values {
		result = append(result, Option{Value: value, Label: labels[value]})
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
	case "posture_mobility":
		return "Posture and mobility"
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
		return "Moderate"
	case "high":
		return "High"
	case "very_high":
		return "Very high"
	case "very_low":
		return "Very low"
	case "low":
		return "Low"
	case "poor":
		return "Poor"
	case "fair":
		return "Fair"
	case "good":
		return "Good"
	case "excellent":
		return "Excellent"
	case "needs_improvement":
		return "Needs improvement"
	case "high_protein":
		return "High in protein"
	case "high_fast_food":
		return "High in fast food"
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
	case "never":
		return "Never"
	case "under_6_months":
		return "Less than 6 months"
	case "6_12_months":
		return "6-12 months"
	case "1_2_years":
		return "1-2 years"
	case "2_5_years":
		return "2-5 years"
	case "5_plus_years":
		return "5 or more years"
	case "female":
		return "Female"
	case "male":
		return "Male"
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
			Key:         "personal",
			Title:       "Personal details",
			Description: "These details identify you to your trainer and help shape the plan around your life.",
			Questions: []Question{
				{Key: "full_name", Label: "Full name", Type: QuestionTypeText, Required: true, MaxLength: MaxShortTextLength},
				{Key: "date_of_birth", Label: "Date of birth", Type: QuestionTypeText, Required: true, MaxLength: 10, Help: "Use the format DD/MM/YYYY."},
				{Key: "sex", Label: "Sex", Type: QuestionTypeSelect, Required: true, Options: opts("female", "male", "prefer_not_to_say")},
				{Key: "occupation", Label: "Occupation", Type: QuestionTypeText, Required: true, MaxLength: MaxShortTextLength},
				{Key: "work_schedule", Label: "Work schedule", Type: QuestionTypeText, Required: true, MaxLength: 200, Help: "For example fixed hours, shifts or the times you usually work."},
				{Key: "social_media", Label: "Social media", Type: QuestionTypeText, Required: false, MaxLength: 200, Help: "Optional: Instagram, Facebook or TikTok, if you want your trainer to know."},
			},
		},
		{
			Key:         "goals",
			Title:       "Goals",
			Description: "Tell your trainer exactly what you want to achieve.",
			Questions: []Question{
				{Key: "primary_goal", Label: "Primary goal", Type: QuestionTypeSelect, Required: true, Options: opts("fat_loss", "muscle_gain", "strength", "recomposition", "athletic_performance", "general_health", "posture_mobility", "other")},
				{Key: "primary_goal_other", Label: "If you chose Other, describe your main goal", Type: QuestionTypeText, Required: false, MaxLength: 200},
				{Key: "secondary_goals", Label: "Secondary goals", Type: QuestionTypeMultiSelect, Required: false, Options: opts("fat_loss", "muscle_gain", "strength", "recomposition", "athletic_performance", "general_health", "posture_mobility", "other")},
				{Key: "specific_goal", Label: "What exactly do you want to achieve?", Type: QuestionTypeTextarea, Required: true, MaxLength: 1000, Help: "For example: lose 5 kg, build lower-body muscle or improve posture."},
				{Key: "target_timeline", Label: "Target timeline", Type: QuestionTypeSelect, Required: false, Options: opts("8_weeks", "12_weeks", "16_weeks", "24_weeks", "no_deadline")},
				{Key: "motivation", Label: "Why does this goal matter to you?", Type: QuestionTypeTextarea, Required: true, MaxLength: 1000},
			},
		},
		{
			Key:         "training",
			Title:       "Training history",
			Description: "Your trainer needs to know how, where and when you can train.",
			Questions: []Question{
				{Key: "training_experience", Label: "Training experience", Type: QuestionTypeSelect, Required: true, Options: opts("beginner", "intermediate", "advanced")},
				{Key: "training_history_duration", Label: "How long have you trained for?", Type: QuestionTypeSelect, Required: true, Options: opts("never", "under_6_months", "6_12_months", "1_2_years", "2_5_years", "5_plus_years")},
				{Key: "currently_training", Label: "Are you currently training?", Type: QuestionTypeBoolean, Required: true},
				{Key: "sessions_per_week", Label: "Training sessions per week", Type: QuestionTypeNumber, Required: true, Min: float(1), Max: float(7), Step: float(1)},
				{Key: "session_minutes", Label: "Minutes per session", Type: QuestionTypeNumber, Required: true, Min: float(20), Max: float(180), Step: float(5)},
				{Key: "training_location", Label: "Where will you train?", Type: QuestionTypeSelect, Required: true, Options: opts("gym", "home", "hybrid", "outdoors")},
				{Key: "equipment", Label: "Available equipment", Type: QuestionTypeMultiSelect, Required: false, Options: opts("full_gym", "dumbbells", "barbell_rack", "resistance_bands", "pull_up_bar", "cardio_machines", "kettlebells", "bodyweight_only")},
				{Key: "preferred_days", Label: "Preferred training days", Type: QuestionTypeMultiSelect, Required: true, Options: opts("mon", "tue", "wed", "thu", "fri", "sat", "sun")},
				{Key: "split_preference", Label: "Preferred training split", Type: QuestionTypeText, Required: false, MaxLength: 200, Help: "Leave blank if you have no preference."},
				{Key: "current_training", Label: "Current training routine", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
				{Key: "liked_exercises", Label: "Exercises you enjoy", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
				{Key: "disliked_exercises", Label: "Exercises you dislike or avoid", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000, Help: "Include anything that feels uncomfortable."},
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
				{Key: "activity_level", Label: "Daily activity level", Type: QuestionTypeSelect, Required: true, Options: labeledOpts(map[string]string{"sedentary": "Sedentary", "light": "Lightly active", "moderate": "Moderately active", "high": "Very active", "very_high": "Extremely active"}, "sedentary", "light", "moderate", "high", "very_high")},
				{Key: "daily_steps", Label: "Average daily steps", Type: QuestionTypeNumber, Required: false, Min: float(0), Max: float(50000), Step: float(500)},
				{Key: "sleep_hours", Label: "Average sleep per night (hours)", Type: QuestionTypeNumber, Required: true, Min: float(3), Max: float(14), Step: float(0.5)},
				{Key: "sleep_quality", Label: "How would you rate your sleep?", Type: QuestionTypeSelect, Required: true, Options: opts("poor", "fair", "good", "excellent")},
				{Key: "stress_level", Label: "Stress level", Type: QuestionTypeSelect, Required: true, Options: opts("very_low", "low", "moderate", "high", "very_high"), Help: "From 1 (very low) to 5 (very high)."},
				{Key: "alcohol", Label: "Do you drink alcohol?", Type: QuestionTypeBoolean, Required: true},
				{Key: "alcohol_frequency", Label: "How often do you drink alcohol?", Type: QuestionTypeText, Required: false, MaxLength: 200},
				{Key: "caffeine", Label: "Do you have caffeine?", Type: QuestionTypeBoolean, Required: true, Help: "For example coffee, cola or pre-workout."},
				{Key: "caffeine_frequency", Label: "How often do you have caffeine?", Type: QuestionTypeText, Required: false, MaxLength: 200},
				{Key: "schedule_notes", Label: "Schedule or recovery notes", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
			},
		},
		{
			Key:         "nutrition",
			Title:       "Nutrition",
			Description: "Your trainer will build an individual plan around these answers.",
			Questions: []Question{
				{Key: "dietary_pattern", Label: "Dietary pattern", Type: QuestionTypeSelect, Required: true, Options: opts("balanced", "vegetarian", "vegan", "pescatarian", "halal", "kosher", "other")},
				{Key: "current_nutrition_quality", Label: "How would you describe your current nutrition?", Type: QuestionTypeSelect, Required: true, Options: opts("balanced", "needs_improvement", "high_protein", "high_fast_food")},
				{Key: "meals_per_day", Label: "Meals per day", Type: QuestionTypeNumber, Required: true, Min: float(2), Max: float(8), Step: float(1)},
				{Key: "allergies", Label: "Allergies", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000, Help: "Write none if not applicable."},
				{Key: "intolerances", Label: "Intolerances", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000, Help: "Write none if not applicable."},
				{Key: "preferred_foods", Label: "Foods you enjoy", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
				{Key: "disliked_foods", Label: "Foods you dislike", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
				{Key: "cooking_time", Label: "Time available to cook", Type: QuestionTypeSelect, Required: true, Options: opts("minimal", "20_30", "30_60", "60_plus")},
				{Key: "budget_level", Label: "Food budget", Type: QuestionTypeSelect, Required: true, Options: opts("low", "moderate", "flexible")},
				{Key: "water_litres", Label: "Water per day (litres)", Type: QuestionTypeNumber, Required: false, Min: float(0.5), Max: float(10), Step: float(0.1)},
				{Key: "supplements", Label: "Supplements you use", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
				{Key: "nutrition_notes", Label: "Anything else about your nutrition", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
			},
		},
		{
			Key:         "health",
			Title:       "Health",
			Description: "Health answers are private and only used to keep your plan safe.",
			Questions: []Question{
				{Key: "has_medical_conditions", Label: "Do you have a diagnosed medical condition?", Type: QuestionTypeBoolean, Required: true},
				{Key: "medical_conditions", Label: "Medical conditions", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000, Help: "Write none if not applicable."},
				{Key: "has_injury_history", Label: "Have you had a relevant injury?", Type: QuestionTypeBoolean, Required: true},
				{Key: "injuries", Label: "Injuries or pain", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000, Help: "Describe anything that limits your training. Write none if not applicable."},
				{Key: "takes_medication", Label: "Are you taking any medication?", Type: QuestionTypeBoolean, Required: true},
				{Key: "medications", Label: "Medications", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
				{Key: "has_movement_limitations", Label: "Do you have any movement restriction or pain when training?", Type: QuestionTypeBoolean, Required: true},
				{Key: "movement_limitations", Label: "Movement limitations", Type: QuestionTypeTextarea, Required: false, MaxLength: 1000},
				{Key: "medical_clearance", Label: "Has a doctor cleared you for exercise?", Type: QuestionTypeBoolean, Required: true},
				{Key: "pregnancy_status", Label: "Pregnancy status", Type: QuestionTypeSelect, Required: false, Options: opts("not_pregnant", "pregnant", "breastfeeding", "prefer_not_to_say")},
				{Key: "smoking", Label: "Do you smoke?", Type: QuestionTypeBoolean, Required: true},
				{Key: "smoking_frequency", Label: "How often do you smoke?", Type: QuestionTypeText, Required: false, MaxLength: 200},
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
