package premium_level_2_test

import (
	"encoding/json"
	"testing"

	"ryze/backend/services/premium_level_2"
)

func validAnswers() map[string]any {
	return map[string]any{
		"primary_goal":        "fat_loss",
		"secondary_goals":     []string{"strength"},
		"training_experience": "intermediate",
		"target_timeline":     "12_weeks",
		"motivation":          "I want to feel stronger and more confident.",
		"sessions_per_week":   4,
		"session_minutes":     60,
		"training_location":   "gym",
		"equipment":           []string{"full_gym", "dumbbells"},
		"preferred_days":      []string{"mon", "tue", "thu", "sat"},
		"height_cm":           178.5,
		"weight_kg":           82,
		"activity_level":      "moderate",
		"occupation":          "Software engineer",
		"sleep_hours":         7.5,
		"stress_level":        "moderate",
		"dietary_pattern":     "balanced",
		"meals_per_day":       4,
		"cooking_time":        "30_60",
		"budget_level":        "moderate",
		"medical_clearance":   true,
		"coaching_style":      "balanced",
		"checkin_frequency":   "weekly",
	}
}

func encoded(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return raw
}

func TestNormalizeAnswersAcceptsValidCatalog(t *testing.T) {
	normalized, fields := premium_level_2.NormalizeAnswers(encoded(t, validAnswers()))
	if len(fields) > 0 {
		t.Fatalf("unexpected validation errors: %v", fields)
	}
	var decoded map[string]any
	if err := json.Unmarshal(normalized, &decoded); err != nil {
		t.Fatalf("normalized output is not JSON: %v", err)
	}
	if decoded["primary_goal"] != "fat_loss" {
		t.Fatalf("expected normalized goal, got %#v", decoded["primary_goal"])
	}
	if _, ok := decoded["unknown"]; ok {
		t.Fatal("unknown field leaked into normalized output")
	}
}

func TestNormalizeAnswersRejectsUnknownAndMissingFields(t *testing.T) {
	answers := validAnswers()
	delete(answers, "height_cm")
	answers["client_supplied_field"] = "not allowed"
	_, fields := premium_level_2.NormalizeAnswers(encoded(t, answers))
	if fields["height_cm"] == "" {
		t.Fatalf("expected required field error, got %v", fields)
	}
	if fields["client_supplied_field"] == "" {
		t.Fatalf("expected unknown field error, got %v", fields)
	}
}

func TestNormalizeAnswersRejectsImpossibleTrainingSchedule(t *testing.T) {
	answers := validAnswers()
	answers["sessions_per_week"] = 6
	answers["preferred_days"] = []string{"mon", "wed", "fri"}
	_, fields := premium_level_2.NormalizeAnswers(encoded(t, answers))
	if fields["preferred_days"] == "" {
		t.Fatalf("expected schedule error, got %v", fields)
	}
}

func TestCatalogKeysAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, section := range premium_level_2.IntakeCatalog().Sections {
		for _, question := range section.Questions {
			if seen[question.Key] {
				t.Fatalf("duplicate question key %q", question.Key)
			}
			seen[question.Key] = true
		}
	}
}
