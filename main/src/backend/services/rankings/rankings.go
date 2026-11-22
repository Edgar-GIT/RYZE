// Package rankings compares a member's best lifts with the whole platform
// (owner request, 2026-11-22, like Lyfta's "Lyftoff": "Bench press — you are
// in the top 1%"). Only the member's own numbers and aggregates are returned.
package rankings

import (
	"context"
	"math"

	"ryze/backend/repositories"
)

// Ranking is one exercise: the member's best lift and where it stands.
type Ranking struct {
	ExerciseID   string  `json:"exercise_id"`
	ExerciseName string  `json:"exercise_name"`
	BestE1RM     float64 `json:"best_e1rm_kg"`
	BestWeightKg float64 `json:"best_weight_kg"`
	BestReps     int     `json:"best_reps"`
	Rank         int     `json:"rank"`
	Participants int     `json:"participants"`
	// TopPercent: the member is in the top TopPercent % (1-100).
	TopPercent int `json:"top_percent"`
	// BeatsPercent: the share of the other members the member lifts more than.
	BeatsPercent int `json:"beats_percent"`
}

type Service struct {
	repo repositories.RankingRepository
}

func NewService(repo repositories.RankingRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, userID string) ([]Ranking, error) {
	rows, err := s.repo.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Ranking, 0, len(rows))
	for _, row := range rows {
		out = append(out, Score(row))
	}
	return out, nil
}

// Score turns a rank among participants into the shown percentages.
func Score(row repositories.RankingRow) Ranking {
	ranking := Ranking{
		ExerciseID: row.ExerciseID, ExerciseName: row.ExerciseName,
		BestE1RM: row.BestE1RM, BestWeightKg: row.BestWeightKg, BestReps: row.BestReps,
		Rank: row.Rank, Participants: row.Participants, TopPercent: 100,
	}
	if row.Participants <= 0 || row.Rank <= 0 {
		return ranking
	}
	ranking.TopPercent = int(math.Ceil(float64(row.Rank) * 100 / float64(row.Participants)))
	if ranking.TopPercent < 1 {
		ranking.TopPercent = 1
	}
	if row.Participants > 1 {
		ranking.BeatsPercent = int(math.Round(float64(row.Participants-row.Rank) * 100 / float64(row.Participants-1)))
	}
	return ranking
}
