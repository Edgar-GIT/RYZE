package achievements_test

import (
	"testing"
	"time"

	"ryze/backend/repositories"
	"ryze/backend/services/achievements"
)

var day = time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)

func set(session string, at time.Time, exercise string, weight float64, reps int, setType string) repositories.CompletedSetRow {
	w, r := weight, reps
	return repositories.CompletedSetRow{SessionID: session, CompletedAt: at, ExerciseID: exercise, ExerciseName: "Lift " + exercise, SetType: setType, WeightKg: &w, Reps: &r}
}

func TestFirstLiftSetsTheRecordWithoutAMedal(t *testing.T) {
	summary := achievements.Compute([]repositories.CompletedSetRow{
		set("s1", day, "bench", 60, 8, "working"),
		set("s1", day, "bench", 70, 5, "working"),
	})
	if summary.MedalCount != 0 {
		t.Fatalf("medals = %d, want 0 on the first session", summary.MedalCount)
	}
}

func TestBeatingTheRecordEarnsOneMedalPerExercise(t *testing.T) {
	next := day.Add(48 * time.Hour)
	later := day.Add(96 * time.Hour)
	summary := achievements.Compute([]repositories.CompletedSetRow{
		set("s1", day, "bench", 70, 5, "working"),
		set("s1", day, "squat", 100, 5, "working"),
		// Session 2: two heavier bench sets earn a single medal at the top
		// weight; a heavy warm-up does not count; squat only matches.
		set("s2", next, "bench", 72.5, 5, "working"),
		set("s2", next, "bench", 75, 3, "working"),
		set("s2", next, "squat", 140, 1, "warmup"),
		set("s2", next, "squat", 100, 6, "working"),
		// Session 3: lighter than the record, no medal.
		set("s3", later, "bench", 74, 5, "working"),
	})
	if summary.MedalCount != 1 {
		t.Fatalf("medals = %d (%+v), want 1", summary.MedalCount, summary.Medals)
	}
	medal := summary.Medals[0]
	if medal.ExerciseID != "bench" || medal.WeightKg != 75 || medal.Reps != 3 || medal.PreviousBestKg != 70 || medal.SessionID != "s2" {
		t.Fatalf("unexpected medal %+v", medal)
	}
}

func TestExperiencePointsAndDailyLimit(t *testing.T) {
	var rows []repositories.CompletedSetRow
	// Session a: 3 sets. Session b: 45 sets (counted as 40). Session c: the
	// third on the same UTC day earns nothing.
	for i := 0; i < 3; i++ {
		rows = append(rows, set("a", day, "row", 50, 10, "working"))
	}
	for i := 0; i < 45; i++ {
		rows = append(rows, set("b", day.Add(time.Hour), "curl", 10, 10, "working"))
	}
	rows = append(rows, set("c", day.Add(2*time.Hour), "row", 60, 10, "working"))
	summary := achievements.Compute(rows)

	want := (100 + 3*5) + (100 + 40*5)
	if summary.XP != want || summary.WorkoutsCounted != 2 {
		t.Fatalf("xp = %d over %d workouts, want %d over 2", summary.XP, summary.WorkoutsCounted, want)
	}
	third := summary.Sessions[2]
	if third.Counted || third.XP != 0 || len(third.Medals) != 1 {
		t.Fatalf("third session = %+v, want no EXP but its medal", third)
	}
	if summary.Sessions[1].XPBefore != 115 || summary.Sessions[1].XPAfter != want {
		t.Fatalf("before/after = %d/%d", summary.Sessions[1].XPBefore, summary.Sessions[1].XPAfter)
	}
}

func TestRecordBonus(t *testing.T) {
	summary := achievements.Compute([]repositories.CompletedSetRow{
		set("s1", day, "press", 40, 8, "working"),
		set("s2", day.Add(24*time.Hour), "press", 42.5, 8, "working"),
	})
	if got := summary.Sessions[1].XP; got != 100+5+50 {
		t.Fatalf("xp = %d, want 155", got)
	}
}

func TestLevels(t *testing.T) {
	cases := map[int]string{0: "rookie", 499: "rookie", 500: "beginner", 1500: "average", 14999: "advanced", 15000: "elite", 40000: "legend", 999999: "legend"}
	for xp, key := range cases {
		level, next := achievements.LevelFor(xp)
		if level.Key != key {
			t.Fatalf("LevelFor(%d) = %s, want %s", xp, level.Key, key)
		}
		if key == "legend" && next != nil {
			t.Fatalf("legend has no next level")
		}
		if key == "rookie" && (next == nil || next.Key != "beginner") {
			t.Fatalf("rookie's next level is beginner")
		}
	}
}
