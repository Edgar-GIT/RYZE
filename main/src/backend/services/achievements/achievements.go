// Package achievements derives the member's experience points (EXP), lifter
// level and record medals from their completed workout sessions (owner
// request, 2026-11-14: "every weight record earns a medal, like Lyfta; every
// workout earns EXP towards levels such as average lifter or elite lifter").
//
// Nothing is stored: everything is recomputed from the immutable session
// history, so the numbers can never drift from what was trained and the rules
// apply to the whole history the same way.
//
// Rules:
//   - A completed session with at least one completed set earns
//     WorkoutXP, plus SetXP per completed set (at most MaxSetsCounted sets),
//     plus RecordXP per medal earned in it. Sessions with no completed set earn
//     nothing. At most SessionsPerDay sessions per UTC day earn EXP; later
//     ones that day earn none (medals still count).
//   - A medal is earned when a session's heaviest completed set of an
//     exercise beats the member's previous heaviest completed set of that
//     exercise (warm-up sets, sets without weight or reps are ignored). The
//     first time an exercise is lifted sets the record and earns no medal.
//     One medal per exercise per session at most.
package achievements

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

const (
	WorkoutXP      = 100
	SetXP          = 5
	MaxSetsCounted = 40
	RecordXP       = 50
	SessionsPerDay = 2

	// maxMedalsListed bounds the medal list of the summary; MedalCount is
	// always the full count.
	maxMedalsListed = 500
	// recordEpsilon ignores rounding noise between equal weights.
	recordEpsilon = 0.001
)

var (
	ErrInvalidInput        = errors.New("invalid achievements input")
	ErrSessionNotFound     = errors.New("workout session not found")
	ErrSessionNotCompleted = errors.New("workout session not completed")
)

// warmupSetType never counts for records.
const warmupSetType = "warmup"

// Level is one lifter level; Key is translated by the client.
type Level struct {
	Rank  int    `json:"rank"`
	Key   string `json:"key"`
	MinXP int    `json:"min_xp"`
}

// Levels are the lifter levels, in order. A typical session (about twenty
// sets) earns about 200 EXP, so Average comes after two or three weeks of
// regular training and Legendary after more than a year.
var Levels = []Level{
	{Rank: 1, Key: "rookie", MinXP: 0},
	{Rank: 2, Key: "beginner", MinXP: 500},
	{Rank: 3, Key: "average", MinXP: 1500},
	{Rank: 4, Key: "dedicated", MinXP: 4000},
	{Rank: 5, Key: "advanced", MinXP: 8000},
	{Rank: 6, Key: "elite", MinXP: 15000},
	{Rank: 7, Key: "master", MinXP: 25000},
	{Rank: 8, Key: "legend", MinXP: 40000},
}

// LevelFor returns the level reached with xp and the next one (nil at the top).
func LevelFor(xp int) (Level, *Level) {
	current := Levels[0]
	var next *Level
	for i, level := range Levels {
		if xp >= level.MinXP {
			current = level
			next = nil
			if i+1 < len(Levels) {
				following := Levels[i+1]
				next = &following
			}
		}
	}
	return current, next
}

// Medal is one weight record.
type Medal struct {
	ID             string    `json:"id"`
	SessionID      string    `json:"session_id"`
	ExerciseID     string    `json:"exercise_id"`
	ExerciseName   string    `json:"exercise_name"`
	WeightKg       float64   `json:"weight_kg"`
	Reps           int       `json:"reps"`
	PreviousBestKg float64   `json:"previous_best_kg"`
	EarnedAt       time.Time `json:"earned_at"`
}

// Breakdown explains a session's EXP.
type Breakdown struct {
	Workout int `json:"workout"`
	Sets    int `json:"sets"`
	Records int `json:"records"`
}

// SessionAward is what one completed session earned.
type SessionAward struct {
	SessionID     string    `json:"session_id"`
	CompletedAt   time.Time `json:"completed_at"`
	CompletedSets int       `json:"completed_sets"`
	XP            int       `json:"xp_earned"`
	Counted       bool      `json:"counted"`
	Breakdown     Breakdown `json:"breakdown"`
	Medals        []Medal   `json:"medals"`
	XPBefore      int       `json:"xp_before"`
	XPAfter       int       `json:"xp_after"`
}

// Summary is the member's achievements.
type Summary struct {
	XP              int
	Level           Level
	NextLevel       *Level
	WorkoutsCounted int
	MedalCount      int
	Medals          []Medal
	Sessions        []SessionAward
}

// Compute applies the rules to the completed sets (oldest session first).
func Compute(rows []repositories.CompletedSetRow) Summary {
	return ComputeFrom(0, rows)
}

// ComputeFrom applies the rules starting from base EXP (the sum of the
// member's EXP adjustments; never below zero).
func ComputeFrom(base int, rows []repositories.CompletedSetRow) Summary {
	type exerciseBest struct {
		weight float64
		reps   int
		name   string
	}
	best := map[string]float64{}
	var sessions []SessionAward
	var medals []Medal
	perDay := map[string]int{}
	total := max(base, 0)
	counted := 0

	for start := 0; start < len(rows); {
		end := start
		for end < len(rows) && rows[end].SessionID == rows[start].SessionID {
			end++
		}
		group := rows[start:end]
		start = end

		award := SessionAward{SessionID: group[0].SessionID, CompletedAt: group[0].CompletedAt, Medals: []Medal{}}
		sessionBest := map[string]*exerciseBest{}
		order := []string{}
		for _, row := range group {
			award.CompletedSets++
			if strings.EqualFold(row.SetType, warmupSetType) || row.WeightKg == nil || row.Reps == nil || *row.WeightKg <= 0 || *row.Reps < 1 {
				continue
			}
			current, ok := sessionBest[row.ExerciseID]
			if !ok {
				current = &exerciseBest{name: row.ExerciseName}
				sessionBest[row.ExerciseID] = current
				order = append(order, row.ExerciseID)
			}
			if *row.WeightKg > current.weight+recordEpsilon || (*row.WeightKg > current.weight-recordEpsilon && *row.Reps > current.reps) {
				current.weight = *row.WeightKg
				current.reps = *row.Reps
			}
		}
		for _, exerciseID := range order {
			lifted := sessionBest[exerciseID]
			previous, seen := best[exerciseID]
			if seen && lifted.weight > previous+recordEpsilon {
				award.Medals = append(award.Medals, Medal{
					ID:             award.SessionID + ":" + exerciseID,
					SessionID:      award.SessionID,
					ExerciseID:     exerciseID,
					ExerciseName:   lifted.name,
					WeightKg:       lifted.weight,
					Reps:           lifted.reps,
					PreviousBestKg: previous,
					EarnedAt:       award.CompletedAt,
				})
			}
			if !seen || lifted.weight > previous {
				best[exerciseID] = lifted.weight
			}
		}

		day := award.CompletedAt.UTC().Format("2006-01-02")
		award.XPBefore = total
		if perDay[day] < SessionsPerDay {
			perDay[day]++
			sets := award.CompletedSets
			if sets > MaxSetsCounted {
				sets = MaxSetsCounted
			}
			award.Counted = true
			award.Breakdown = Breakdown{Workout: WorkoutXP, Sets: sets * SetXP, Records: len(award.Medals) * RecordXP}
			award.XP = award.Breakdown.Workout + award.Breakdown.Sets + award.Breakdown.Records
			total += award.XP
			counted++
		}
		award.XPAfter = total
		medals = append(medals, award.Medals...)
		sessions = append(sessions, award)
	}

	level, next := LevelFor(total)
	newestFirst := make([]Medal, len(medals))
	for i := range medals {
		newestFirst[len(medals)-1-i] = medals[i]
	}
	listed := newestFirst
	if len(listed) > maxMedalsListed {
		listed = listed[:maxMedalsListed]
	}
	return Summary{
		XP:              total,
		Level:           level,
		NextLevel:       next,
		WorkoutsCounted: counted,
		MedalCount:      len(medals),
		Medals:          listed,
		Sessions:        sessions,
	}
}

// XPAdjustmentReader is implemented by repositories that store EXP
// adjustments (optional, so test fakes keep compiling).
type XPAdjustmentReader interface {
	XPAdjustment(ctx context.Context, userID string) (int, error)
}

type Service interface {
	Summary(ctx context.Context, userID string) (*Summary, error)
	// Session returns what one completed session earned; a completed
	// session without completed sets earned nothing.
	Session(ctx context.Context, userID, sessionID string) (*SessionAward, error)
}

type service struct {
	repo repositories.AchievementsRepository
}

func NewService(repo repositories.AchievementsRepository) Service {
	return &service{repo: repo}
}

func (s *service) Summary(ctx context.Context, userID string) (*Summary, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrInvalidInput
	}
	rows, err := s.repo.ListCompletedSets(ctx, userID)
	if err != nil {
		return nil, err
	}
	base := 0
	if adjustments, ok := s.repo.(XPAdjustmentReader); ok {
		if base, err = adjustments.XPAdjustment(ctx, userID); err != nil {
			return nil, err
		}
	}
	summary := ComputeFrom(base, rows)
	return &summary, nil
}

func (s *service) Session(ctx context.Context, userID, sessionID string) (*SessionAward, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(sessionID) == "" {
		return nil, ErrSessionNotFound
	}
	status, err := s.repo.SessionStatus(ctx, userID, sessionID)
	if err != nil {
		if errors.Is(err, repositories.ErrAchievementSessionNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	if status != models.WorkoutSessionStatusCompleted {
		return nil, ErrSessionNotCompleted
	}
	summary, err := s.Summary(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to compute achievements: %w", err)
	}
	for i := range summary.Sessions {
		if summary.Sessions[i].SessionID == sessionID {
			award := summary.Sessions[i]
			return &award, nil
		}
	}
	return &SessionAward{SessionID: sessionID, Medals: []Medal{}, XPBefore: summary.XP, XPAfter: summary.XP}, nil
}
