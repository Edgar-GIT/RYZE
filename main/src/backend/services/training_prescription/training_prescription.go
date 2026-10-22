// Package training_prescription defines the shared set-prescription input and
// validation used by every program builder. Trainer-owned and platform-owned
// programs persist the same workout_exercise_sets schema, so the accepted
// vocabulary and numeric bounds must not drift between them.
package training_prescription

import (
	"fmt"
	"unicode/utf8"
)

const (
	// MaxSetsPerExercise caps the number of prescription sets per workout
	// exercise.
	MaxSetsPerExercise = 50
	// MaxTempoLength caps the tempo notation length, matching the database
	// column.
	MaxTempoLength = 20
)

// SetTypeValues is the controlled set-type vocabulary, matching the database
// CHECK constraint and the builder's prescription editor.
var SetTypeValues = []string{"warmup", "working", "drop", "backoff", "failure"}

// SetInput carries one prescription set of a workout exercise. Set numbers are
// never accepted from the client: they are derived from the array order.
type SetInput struct {
	SetType     string
	Reps        *int
	WeightKg    *float64
	RIR         *int
	RPE         *float64
	RestSeconds *int
	Tempo       string
}

// ValidateSet checks one set against the shared prescription rules. Set type is
// mandatory because an empty value cannot satisfy the database CHECK
// constraint; every numeric field remains optional.
func ValidateSet(set *SetInput) error {
	if set == nil {
		return fmt.Errorf("set is required")
	}
	if !validSetType(set.SetType) {
		return fmt.Errorf("invalid set type")
	}
	if set.Reps != nil && *set.Reps < 1 {
		return fmt.Errorf("reps must be at least 1")
	}
	if set.WeightKg != nil && *set.WeightKg < 0 {
		return fmt.Errorf("weight cannot be negative")
	}
	if set.RIR != nil && *set.RIR < 0 {
		return fmt.Errorf("rir cannot be negative")
	}
	if set.RPE != nil && (*set.RPE < 1.0 || *set.RPE > 10.0) {
		return fmt.Errorf("rpe must be between 1 and 10")
	}
	if set.RestSeconds != nil && *set.RestSeconds < 0 {
		return fmt.Errorf("rest cannot be negative")
	}
	if utf8.RuneCountInString(set.Tempo) > MaxTempoLength {
		return fmt.Errorf("tempo exceeds the maximum length")
	}
	return nil
}

func validSetType(value string) bool {
	for _, candidate := range SetTypeValues {
		if value == candidate {
			return true
		}
	}
	return false
}
