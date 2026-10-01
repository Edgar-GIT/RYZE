package nutrition_questionnaire

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func completeAnswers() Answers {
	return Answers{
		Goal:              StringPtr("fat_loss"),
		Experience:        StringPtr("intermediate"),
		TrainingDays:      IntPtr(4),
		SessionLength:     IntPtr(60),
		Diet:              StringPtr("omnivore"),
		MealsPerDay:       IntPtr(4),
		SnacksPerDay:      IntPtr(1),
		CookingEffort:     StringPtr("moderate"),
		BodyWeightKg:      FloatPtr(80),
		BodyHeightCm:      IntPtr(180),
		ActivityLevel:     StringPtr("moderate"),
		SleepHours:        FloatPtr(7),
		StressLevel:       StringPtr("low"),
		Allergies:         StringsPtr("peanuts"),
		Intolerances:      StringsPtr("lactose"),
		ExcludedFoods:     StringsPtr("pork"),
		DislikedFoods:     StringsPtr("olives"),
		MedicalConditions: StringsPtr("type 1 diabetes"),
		Injuries:          StringsPtr(),
		Limitations:       StringsPtr("no overhead pressing"),
		Notes:             StringPtr("  Prefer  fish  "),
	}
}

func TestNormalizeAcceptsCompleteSubmission(t *testing.T) {
	normalized, err := Normalize(completeAnswers())
	if err != nil {
		t.Fatalf("expected submission to be accepted, got %v", err)
	}

	if normalized.SchemaVersion != SchemaVersion {
		t.Errorf("schema version = %d, want %d", normalized.SchemaVersion, SchemaVersion)
	}
	if normalized.Goal != "fat_loss" {
		t.Errorf("goal = %q, want fat_loss", normalized.Goal)
	}
	if normalized.Notes != "Prefer  fish" {
		t.Errorf("notes = %q, want trimmed and control-stripped value", normalized.Notes)
	}
	if !normalized.HasMedicalFlags() {
		t.Error("expected medical flags to be reported for a declared condition")
	}
}

func TestNormalizeReportsEveryInvalidFieldAtOnce(t *testing.T) {
	answers := Answers{
		Goal:          StringPtr("not_a_goal"),
		TrainingDays:  IntPtr(99),
		BodyWeightKg:  FloatPtr(500),
		SessionLength: IntPtr(5),
	}

	_, err := Normalize(answers)
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("expected *ValidationError, got %v", err)
	}

	reasons := validation.FieldErrors()
	for _, field := range []string{FieldGoal, FieldTrainingDays, FieldBodyWeightKg, FieldSessionLength} {
		if _, ok := reasons[field]; !ok {
			t.Errorf("expected field %q to be reported, got %v", field, reasons)
		}
	}
	// A missing required field must also be reported rather than defaulted.
	if _, ok := reasons[FieldExperience]; !ok {
		t.Errorf("expected missing required field to be reported, got %v", reasons)
	}
}

func TestNormalizeRejectsInventedVocabulary(t *testing.T) {
	// A client-supplied value outside the closed vocabulary must never reach the
	// generator, so it is rejected rather than passed through.
	answers := completeAnswers()
	answers.Diet = StringPtr("carnivore-ish")

	if _, err := Normalize(answers); err == nil {
		t.Fatal("expected an unsupported diet value to be rejected")
	}
}

func TestNormalizeIsCaseInsensitiveAndTrimsOptions(t *testing.T) {
	answers := completeAnswers()
	answers.Goal = StringPtr("  FAT_LOSS  ")

	normalized, err := Normalize(answers)
	if err != nil {
		t.Fatalf("expected trimmed option to be accepted, got %v", err)
	}
	if normalized.Goal != "fat_loss" {
		t.Errorf("goal = %q, want normalized fat_loss", normalized.Goal)
	}
}

func TestNormalizeCollapsesDuplicateAndBlankListEntries(t *testing.T) {
	answers := completeAnswers()
	answers.DislikedFoods = StringsPtr("olives", "  ", "OLIVES", "kale")

	normalized, err := Normalize(answers)
	if err != nil {
		t.Fatalf("expected submission to be accepted, got %v", err)
	}
	if len(normalized.DislikedFoods) != 2 {
		t.Fatalf("disliked foods = %v, want 2 deduplicated entries", normalized.DislikedFoods)
	}
	// Entries are sorted so the persisted document is stable.
	if normalized.DislikedFoods[0] != "kale" || normalized.DislikedFoods[1] != "olives" {
		t.Errorf("disliked foods = %v, want sorted entries", normalized.DislikedFoods)
	}
}

func TestNormalizeEnforcesListBounds(t *testing.T) {
	t.Run("too many entries", func(t *testing.T) {
		answers := completeAnswers()
		over := make([]string, MaxListItems+1)
		for i := range over {
			over[i] = "item"
		}
		answers.Allergies = &over

		if _, err := Normalize(answers); err == nil {
			t.Fatal("expected too many entries to be rejected")
		}
	})

	t.Run("entry too long", func(t *testing.T) {
		answers := completeAnswers()
		answers.Allergies = StringsPtr(strings.Repeat("a", MaxListItemLength+1))

		if _, err := Normalize(answers); err == nil {
			t.Fatal("expected an over-long entry to be rejected")
		}
	})
}

func TestNormalizeStripsControlCharacters(t *testing.T) {
	answers := completeAnswers()
	// A terminal escape sequence must never reach the database.
	answers.Notes = StringPtr("hello\x1b[31mred\x07")

	normalized, err := Normalize(answers)
	if err != nil {
		t.Fatalf("expected submission to be accepted, got %v", err)
	}
	if strings.ContainsAny(normalized.Notes, "\x1b\x07") {
		t.Errorf("notes = %q, want control characters removed", normalized.Notes)
	}
}

func TestNormalizeOptionalListsDefaultToEmptyNotNull(t *testing.T) {
	answers := completeAnswers()
	answers.Injuries = nil
	answers.Limitations = nil

	normalized, err := Normalize(answers)
	if err != nil {
		t.Fatalf("expected submission to be accepted, got %v", err)
	}
	if normalized.Injuries == nil || normalized.Limitations == nil {
		t.Fatal("expected omitted lists to normalize to empty slices")
	}

	encoded, err := normalized.Marshal()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(encoded), "null") {
		t.Errorf("encoded document %s must not contain null list values", encoded)
	}
}

func TestMarshalIsDeterministic(t *testing.T) {
	normalized, err := Normalize(completeAnswers())
	if err != nil {
		t.Fatalf("normalization failed: %v", err)
	}

	first, err := normalized.Marshal()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	second, err := normalized.Marshal()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(first) != string(second) {
		t.Error("expected identical inputs to produce byte-identical documents")
	}
}

func TestRoundTripPreservesDocument(t *testing.T) {
	normalized, err := Normalize(completeAnswers())
	if err != nil {
		t.Fatalf("normalization failed: %v", err)
	}
	encoded, err := normalized.Marshal()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	decoded, err := Unmarshal(encoded)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	again, err := decoded.Marshal()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(encoded) != string(again) {
		t.Error("expected a round trip to be lossless")
	}
}

func TestUnmarshalRejectsUnknownSchemaVersion(t *testing.T) {
	// A document written by a different contract version must not be
	// reinterpreted in place.
	payload, err := json.Marshal(map[string]any{"schema_version": SchemaVersion + 1})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	if _, err := Unmarshal(payload); err == nil {
		t.Fatal("expected an unknown schema version to be rejected")
	}
}

func TestCatalogDescribesEveryNormalizedField(t *testing.T) {
	catalog := Catalog()
	described := make(map[string]struct{}, len(catalog))
	for _, question := range catalog {
		if _, duplicate := described[question.Field]; duplicate {
			t.Errorf("field %q appears twice in the catalog", question.Field)
		}
		described[question.Field] = struct{}{}
	}

	// The client renders the questionnaire from the catalog, so every persisted
	// field must be described by it.
	encoded, err := Normalized{}.Marshal()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	for field := range document {
		if field == "schema_version" {
			continue
		}
		if _, ok := described[field]; !ok {
			t.Errorf("normalized field %q is missing from the catalog", field)
		}
	}
}
