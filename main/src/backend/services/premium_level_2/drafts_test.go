package premium_level_2_test

import (
	"encoding/json"
	"testing"

	"ryze/backend/services/premium_level_2"
)

// TestNormalizePartialAnswers covers the questionnaire saved before payment:
// valid answers are kept, invalid and unknown ones are reported (and never
// stored), and every required question not answered yet is listed.
func TestNormalizePartialAnswers(t *testing.T) {
	raw := []byte(`{"primary_goal":"muscle_gain","sessions_per_week":42,"favourite_colour":"blue"}`)
	normalized, invalid, missing := premium_level_2.NormalizePartialAnswers(raw)

	var kept map[string]any
	if err := json.Unmarshal(normalized, &kept); err != nil {
		t.Fatalf("normalized answers are not JSON: %v", err)
	}
	if kept["primary_goal"] != "muscle_gain" {
		t.Fatalf("valid answer not kept: %v", kept)
	}
	if _, ok := kept["sessions_per_week"]; ok {
		t.Fatal("an out-of-range answer must not be stored")
	}
	if _, ok := kept["favourite_colour"]; ok {
		t.Fatal("an unknown field must not be stored")
	}
	if invalid["sessions_per_week"] == "" || invalid["favourite_colour"] == "" {
		t.Fatalf("invalid answers not reported: %v", invalid)
	}
	if len(missing) == 0 {
		t.Fatal("required questions not answered yet must be listed")
	}
	for _, key := range missing {
		if key == "primary_goal" {
			t.Fatal("an answered question must not be listed as missing")
		}
	}

	empty, emptyInvalid, emptyMissing := premium_level_2.NormalizePartialAnswers(nil)
	if string(empty) != "{}" || len(emptyInvalid) != 0 || len(emptyMissing) == 0 {
		t.Fatalf("an empty draft is valid but incomplete: %s %v %v", empty, emptyInvalid, emptyMissing)
	}
}
