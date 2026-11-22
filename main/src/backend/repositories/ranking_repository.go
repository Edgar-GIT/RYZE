package repositories

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// RankingRow is how a member's best lift on one exercise compares with every
// member's best on it.
type RankingRow struct {
	ExerciseID   string
	ExerciseName string
	BestE1RM     float64
	BestWeightKg float64
	BestReps     int
	Rank         int
	Participants int
}

// RankingRepository compares lifts across the platform (owner request,
// 2026-11-22, like Lyfta's "Lyftoff"). Only aggregates leave the database:
// never another member's identity or numbers.
type RankingRepository interface {
	ListForUser(ctx context.Context, userID string) ([]RankingRow, error)
}

type rankingRepository struct {
	db *gorm.DB
}

func NewRankingRepository(db *gorm.DB) RankingRepository {
	return &rankingRepository{db: db}
}

// ListForUser ranks the member on every exercise they completed a working set
// of. A lift is scored by its estimated one-rep max (Epley, reps capped at 12);
// warm-up sets, sets without weight or reps, abandoned workouts and deleted
// accounts do not count. Rank 1 is the strongest; ties share a rank.
func (r *rankingRepository) ListForUser(ctx context.Context, userID string) ([]RankingRow, error) {
	var rows []RankingRow
	err := r.db.WithContext(ctx).Raw(`
WITH lifts AS (
    SELECT s.user_id,
           e.exercise_id,
           e.name AS exercise_name,
           ws.actual_weight_kg AS weight,
           ws.actual_reps AS reps,
           ws.actual_weight_kg * (1 + LEAST(ws.actual_reps, 12) / 30.0) AS e1rm
    FROM workout_sessions s
    JOIN users u ON u.id = s.user_id AND u.deleted_at IS NULL
    JOIN workout_session_exercises e ON e.workout_session_id = s.id AND e.deleted_at IS NULL
    JOIN workout_session_sets ws ON ws.workout_session_exercise_id = e.id AND ws.deleted_at IS NULL
    WHERE s.status = 'completed' AND s.deleted_at IS NULL
      AND ws.status = 'completed' AND ws.set_type <> 'warmup'
      AND ws.actual_weight_kg > 0 AND ws.actual_reps >= 1
      AND e.exercise_id IN (
          SELECT e2.exercise_id
          FROM workout_sessions s2
          JOIN workout_session_exercises e2 ON e2.workout_session_id = s2.id AND e2.deleted_at IS NULL
          WHERE s2.user_id = ? AND s2.status = 'completed' AND s2.deleted_at IS NULL
      )
),
best AS (
    SELECT DISTINCT ON (user_id, exercise_id) user_id, exercise_id, exercise_name, weight, reps, e1rm
    FROM lifts
    ORDER BY user_id, exercise_id, e1rm DESC, weight DESC
),
ranked AS (
    SELECT user_id, exercise_id, exercise_name, weight, reps, e1rm,
           RANK() OVER (PARTITION BY exercise_id ORDER BY e1rm DESC) AS rank,
           COUNT(*) OVER (PARTITION BY exercise_id) AS participants
    FROM best
)
SELECT exercise_id, exercise_name, ROUND(e1rm::numeric, 1)::float8 AS best_e1rm, weight::float8 AS best_weight_kg,
       reps AS best_reps, rank, participants
FROM ranked
WHERE user_id = ?
ORDER BY participants DESC, rank ASC, exercise_name ASC`, userID, userID).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to rank lifts: %w", err)
	}
	return rows, nil
}
