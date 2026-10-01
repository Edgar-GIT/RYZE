package questionnaires_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

var nowFixture = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func strPtr(v string) *string     { return &v }
func intPtr(v int) *int           { return &v }
func floatPtr(v float64) *float64 { return &v }

// requirementBody serializes a requirement the way the API layer would, so a
// test can assert the response carries no intake answer content.
func requirementBody(t *testing.T, requirement any) string {
	t.Helper()

	encoded, err := json.Marshal(requirement)
	if err != nil {
		t.Fatalf("failed to serialize requirement: %v", err)
	}
	return string(encoded)
}

// containsAnswerContent reports whether a serialized payload carries intake
// answer values or field names.
func containsAnswerContent(payload string) bool {
	for _, marker := range []string{
		"body_weight_kg",
		"body_height_cm",
		"medical_conditions",
		"allergies",
		"sleep_hours",
	} {
		if strings.Contains(payload, marker) {
			return true
		}
	}
	return false
}
