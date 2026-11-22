package rankings_test

import (
	"testing"

	"ryze/backend/repositories"
	"ryze/backend/services/rankings"
)

func TestScore(t *testing.T) {
	cases := []struct {
		rank, participants, top, beats int
	}{
		{1, 200, 1, 100},
		{2, 200, 1, 99},
		{50, 100, 50, 51},
		{100, 100, 100, 0},
		{1, 1, 100, 0},
	}
	for _, c := range cases {
		got := rankings.Score(repositories.RankingRow{Rank: c.rank, Participants: c.participants})
		if got.TopPercent != c.top || got.BeatsPercent != c.beats {
			t.Errorf("rank %d of %d: got top %d beats %d, want %d and %d", c.rank, c.participants, got.TopPercent, got.BeatsPercent, c.top, c.beats)
		}
	}
}
