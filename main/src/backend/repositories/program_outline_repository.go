package repositories

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// ProgramOutlineDay is one training day of a programme outline: its order,
// title and focus, and how many exercises it has. The exercises themselves
// (names, sets, prescriptions) are never part of an outline.
type ProgramOutlineDay struct {
	WeekNumber    int
	Position      int
	Name          string
	Focus         *string
	ExerciseCount int
}

// ProgramOutlineRepository reads the shape of a programme's structure.
type ProgramOutlineRepository interface {
	// Outline returns the programme's active training days in week and day
	// order. A programme without weeks returns no days.
	Outline(ctx context.Context, programID string) ([]ProgramOutlineDay, error)
}

type programOutlineRepository struct {
	db *gorm.DB
}

func NewProgramOutlineRepository(db *gorm.DB) ProgramOutlineRepository {
	return &programOutlineRepository{db: db}
}

func (r *programOutlineRepository) Outline(ctx context.Context, programID string) ([]ProgramOutlineDay, error) {
	var rows []struct {
		WeekNumber    int
		Position      int
		Name          string
		Focus         *string
		ExerciseCount int
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT program_weeks.week_number AS week_number,
		       program_workouts.position AS position,
		       program_workouts.name AS name,
		       program_workouts.focus AS focus,
		       COUNT(workout_exercises.id) AS exercise_count
		FROM program_weeks
		JOIN program_workouts ON program_workouts.program_week_id = program_weeks.id
		     AND program_workouts.deleted_at IS NULL
		LEFT JOIN workout_exercises ON workout_exercises.program_workout_id = program_workouts.id
		     AND workout_exercises.deleted_at IS NULL
		WHERE program_weeks.program_id = ? AND program_weeks.deleted_at IS NULL
		GROUP BY program_weeks.week_number, program_workouts.id, program_workouts.position,
		         program_workouts.name, program_workouts.focus
		ORDER BY program_weeks.week_number ASC, program_workouts.position ASC`, programID).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to read programme outline: %w", err)
	}
	days := make([]ProgramOutlineDay, 0, len(rows))
	for _, row := range rows {
		days = append(days, ProgramOutlineDay(row))
	}
	return days, nil
}
