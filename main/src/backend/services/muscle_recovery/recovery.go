// Package muscle_recovery estimates how recovered each muscle group is from
// the sets a member completed in the last seven days (owner request,
// 2026-11-20, modelled on Lyfta's and Fitbod's muscle recovery).
//
// The estimate is deliberately simple, and documented in
// docs/INSTRUCTIONS/api.MD:
//   - each completed set loads the exercise's primary muscle with one set and
//     each secondary muscle with half a set;
//   - that load fades linearly to nothing over the muscle's recovery window:
//     72 hours for the large muscles, 48 hours for the others;
//   - ten sets of load still present means 0% recovered;
//   - 85% or more is "ready", 50% to 84% "recovering", below 50% "fatigued".
package muscle_recovery

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"ryze/backend/repositories"
)

// The muscle groups of the body map, in display order.
const (
	Chest      = "chest"
	Shoulders  = "shoulders"
	Biceps     = "biceps"
	Triceps    = "triceps"
	Forearms   = "forearms"
	Abs        = "abs"
	Obliques   = "obliques"
	Traps      = "traps"
	Lats       = "lats"
	UpperBack  = "upper_back"
	LowerBack  = "lower_back"
	Glutes     = "glutes"
	Quads      = "quads"
	Hamstrings = "hamstrings"
	Adductors  = "adductors"
	Calves     = "calves"
)

// Muscles lists every muscle group of the estimate.
var Muscles = []string{Chest, Shoulders, Biceps, Triceps, Forearms, Abs, Obliques, Traps, Lats, UpperBack, LowerBack, Glutes, Quads, Hamstrings, Adductors, Calves}

const (
	// Window is how far back completed sets count.
	Window = 7 * 24 * time.Hour
	// fatigueCapacity is the load (in sets) that means 0% recovered.
	fatigueCapacity = 10.0
	secondaryWeight = 0.5
	largeRecovery   = 72 * time.Hour
	smallRecovery   = 48 * time.Hour
	readyFrom       = 85
	recoveringFrom  = 50

	StatusReady      = "ready"
	StatusRecovering = "recovering"
	StatusFatigued   = "fatigued"
)

var largeMuscles = map[string]bool{Chest: true, Lats: true, UpperBack: true, LowerBack: true, Glutes: true, Quads: true, Hamstrings: true, Adductors: true}

// aliases maps the muscle names used by the exercise catalogue (the starter
// catalogue and the imported dataset) to the muscle groups of the map.
// Names that are not muscles (for example "cardiovascular system") are
// ignored.
var aliases = map[string]string{
	"chest": Chest, "pectorals": Chest, "pecs": Chest, "upper chest": Chest, "lower chest": Chest, "serratus anterior": Chest,
	"shoulders": Shoulders, "delts": Shoulders, "deltoids": Shoulders, "rear deltoids": Shoulders, "rear delts": Shoulders,
	"front deltoids": Shoulders, "side deltoids": Shoulders, "rotator cuff": Shoulders,
	"biceps": Biceps, "brachialis": Biceps,
	"triceps":  Triceps,
	"forearms": Forearms, "wrist flexors": Forearms, "wrist extensors": Forearms, "wrists": Forearms, "hands": Forearms,
	"grip muscles": Forearms, "grip": Forearms,
	"abs": Abs, "abdominals": Abs, "core": Abs, "lower abs": Abs, "upper abs": Abs, "rectus abdominis": Abs,
	"obliques": Obliques,
	"traps":    Traps, "trapezius": Traps, "levator scapulae": Traps, "neck": Traps, "sternocleidomastoid": Traps,
	"lats": Lats, "latissimus dorsi": Lats,
	"upper back": UpperBack, "back": UpperBack, "middle back": UpperBack, "rhomboids": UpperBack,
	"lower back": LowerBack, "spine": LowerBack, "erector spinae": LowerBack,
	"glutes": Glutes, "abductors": Glutes, "hip abductors": Glutes,
	"quads": Quads, "quadriceps": Quads, "hip flexors": Quads,
	"hamstrings": Hamstrings,
	"adductors":  Adductors, "inner thighs": Adductors, "groin": Adductors, "hip adductors": Adductors,
	"calves": Calves, "soleus": Calves, "gastrocnemius": Calves, "ankles": Calves, "ankle stabilizers": Calves,
	"shins": Calves, "tibialis": Calves, "feet": Calves,
}

// ErrInvalidUser is returned for an empty user id.
var ErrInvalidUser = errors.New("muscle recovery: user id required")

// Muscle is the estimate for one muscle group.
type Muscle struct {
	Muscle string `json:"muscle"`
	// Recovery is 0 to 100.
	Recovery int    `json:"recovery"`
	Status   string `json:"status"`
	// WeeklySets is the load of the last seven days, in sets (secondary
	// work counts half).
	WeeklySets    float64    `json:"weekly_sets"`
	LastTrainedAt *time.Time `json:"last_trained_at"`
	// RecoveredAt is when the muscle is estimated fully recovered; null when
	// it already is.
	RecoveredAt *time.Time `json:"recovered_at"`
}

// Report is the estimate for every muscle group.
type Report struct {
	WindowDays  int       `json:"window_days"`
	GeneratedAt time.Time `json:"generated_at"`
	Muscles     []Muscle  `json:"muscles"`
}

// Regions returns the muscle groups an exercise trains. The first target
// muscle is the primary one; the other targets and the catalogue's muscle
// groups are secondary. (The imported dataset stores the real target in
// target_muscles and a secondary muscle in primary_muscle_group, the starter
// catalogue stores both consistently; this rule reads both correctly.) When
// no target is a known muscle, the primary muscle group is used instead.
func Regions(target, primaryGroup, secondaryGroups string) (primary, secondary []string) {
	targets := splitNames(target)
	var primaryNames, secondaryNames []string
	if len(targets) > 0 && aliases[targets[0]] != "" {
		primaryNames = targets[:1]
		secondaryNames = append(append(targets[1:], splitNames(primaryGroup)...), splitNames(secondaryGroups)...)
	} else {
		primaryNames = splitNames(primaryGroup)
		secondaryNames = append(targets, splitNames(secondaryGroups)...)
	}
	seen := map[string]bool{}
	for _, name := range primaryNames {
		if region := aliases[name]; region != "" && !seen[region] {
			seen[region] = true
			primary = append(primary, region)
		}
	}
	for _, name := range secondaryNames {
		if region := aliases[name]; region != "" && !seen[region] {
			seen[region] = true
			secondary = append(secondary, region)
		}
	}
	return primary, secondary
}

func splitNames(value string) []string {
	parts := strings.FieldsFunc(strings.ToLower(value), func(r rune) bool { return r == ',' || r == ';' || r == '/' })
	names := make([]string, 0, len(parts))
	for _, part := range parts {
		if name := strings.Join(strings.Fields(part), " "); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func recoveryWindow(muscle string) time.Duration {
	if largeMuscles[muscle] {
		return largeRecovery
	}
	return smallRecovery
}

// Estimate computes the report at now from the completed sets.
func Estimate(now time.Time, sets []repositories.MuscleSetRow) []Muscle {
	type state struct {
		fatigue, weekly float64
		last, recovered time.Time
	}
	states := map[string]*state{}
	for _, muscle := range Muscles {
		states[muscle] = &state{}
	}
	since := now.Add(-Window)
	load := func(muscle string, weight float64, at time.Time) {
		current := states[muscle]
		window := recoveryWindow(muscle)
		elapsed := now.Sub(at)
		if elapsed < 0 {
			elapsed = 0
		}
		current.weekly += weight
		if at.After(current.last) {
			current.last = at
		}
		if remaining := 1 - float64(elapsed)/float64(window); remaining > 0 {
			current.fatigue += weight * remaining
			if done := at.Add(window); done.After(current.recovered) {
				current.recovered = done
			}
		}
	}
	for _, set := range sets {
		if set.CompletedAt.Before(since) {
			continue
		}
		primary, secondary := Regions(set.TargetMuscles, set.PrimaryGroup, set.SecondaryGroups)
		for _, muscle := range primary {
			load(muscle, 1, set.CompletedAt)
		}
		for _, muscle := range secondary {
			load(muscle, secondaryWeight, set.CompletedAt)
		}
	}
	report := make([]Muscle, 0, len(Muscles))
	for _, name := range Muscles {
		current := states[name]
		recovery := int(math.Round(100 * (1 - math.Min(1, current.fatigue/fatigueCapacity))))
		muscle := Muscle{Muscle: name, Recovery: recovery, WeeklySets: math.Round(current.weekly*10) / 10}
		switch {
		case recovery >= readyFrom:
			muscle.Status = StatusReady
		case recovery >= recoveringFrom:
			muscle.Status = StatusRecovering
		default:
			muscle.Status = StatusFatigued
		}
		if !current.last.IsZero() {
			last := current.last.UTC()
			muscle.LastTrainedAt = &last
		}
		if recovery < 100 && current.recovered.After(now) {
			recovered := current.recovered.UTC()
			muscle.RecoveredAt = &recovered
		}
		report = append(report, muscle)
	}
	return report
}

// SetReader reads a member's recently completed sets.
type SetReader interface {
	RecentSets(ctx context.Context, userID string, since time.Time) ([]repositories.MuscleSetRow, error)
}

// Service serves the estimate of the authenticated member.
type Service struct {
	sets SetReader
	now  func() time.Time
}

func NewService(sets SetReader) *Service {
	return &Service{sets: sets, now: time.Now}
}

// Recovery returns the member's estimate now.
func (s *Service) Recovery(ctx context.Context, userID string) (*Report, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrInvalidUser
	}
	now := s.now().UTC()
	sets, err := s.sets.RecentSets(ctx, userID, now.Add(-Window))
	if err != nil {
		return nil, fmt.Errorf("muscle recovery: %w", err)
	}
	return &Report{WindowDays: int(Window / (24 * time.Hour)), GeneratedAt: now, Muscles: Estimate(now, sets)}, nil
}
