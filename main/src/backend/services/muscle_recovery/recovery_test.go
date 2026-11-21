package muscle_recovery

import (
	"reflect"
	"testing"
	"time"

	"ryze/backend/repositories"
)

func TestRegionsReadsBothCatalogues(t *testing.T) {
	// Imported dataset: the target is the real primary muscle.
	primary, secondary := Regions("pectorals", "triceps", "shoulders, forearms")
	if !reflect.DeepEqual(primary, []string{Chest}) || !reflect.DeepEqual(secondary, []string{Triceps, Shoulders, Forearms}) {
		t.Fatalf("dataset: %v %v", primary, secondary)
	}
	// Starter catalogue: targets list the primary first.
	primary, secondary = Regions("Chest, Triceps, Shoulders", "Chest", "Triceps, Shoulders")
	if !reflect.DeepEqual(primary, []string{Chest}) || !reflect.DeepEqual(secondary, []string{Triceps, Shoulders}) {
		t.Fatalf("starter: %v %v", primary, secondary)
	}
	// No known target: the muscle group is the primary one.
	primary, _ = Regions("cardiovascular system", "quadriceps", "")
	if !reflect.DeepEqual(primary, []string{Quads}) {
		t.Fatalf("fallback: %v", primary)
	}
}

func find(report []Muscle, name string) Muscle {
	for _, muscle := range report {
		if muscle.Muscle == name {
			return muscle
		}
	}
	return Muscle{}
}

func TestEstimateFadesLoadOverTheRecoveryWindow(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	bench := func(at time.Time) repositories.MuscleSetRow {
		return repositories.MuscleSetRow{CompletedAt: at, TargetMuscles: "pectorals", PrimaryGroup: "triceps"}
	}
	var sets []repositories.MuscleSetRow
	for range 10 {
		sets = append(sets, bench(now.Add(-36*time.Hour)))
	}
	sets = append(sets, bench(now.Add(-8*24*time.Hour))) // outside the window

	report := Estimate(now, sets)
	if len(report) != len(Muscles) {
		t.Fatalf("expected every muscle, got %d", len(report))
	}
	chest := find(report, Chest)
	// 10 sets, half of the 72 h window gone: 5 sets left of 10 = 50%.
	if chest.Recovery != 50 || chest.Status != StatusRecovering || chest.WeeklySets != 10 {
		t.Fatalf("chest = %+v", chest)
	}
	if chest.RecoveredAt == nil || !chest.RecoveredAt.Equal(now.Add(36*time.Hour)) {
		t.Fatalf("chest recovered at = %v", chest.RecoveredAt)
	}
	triceps := find(report, Triceps)
	// 10 half sets, 36 of 48 h gone: 1.25 sets left = 88%.
	if triceps.Recovery != 88 || triceps.Status != StatusReady || triceps.WeeklySets != 5 {
		t.Fatalf("triceps = %+v", triceps)
	}
	quads := find(report, Quads)
	if quads.Recovery != 100 || quads.Status != StatusReady || quads.LastTrainedAt != nil || quads.RecoveredAt != nil {
		t.Fatalf("quads = %+v", quads)
	}
}
