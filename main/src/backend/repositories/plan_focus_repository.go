package repositories

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// PlanFocusRow is the plan a member chose to focus on.
type PlanFocusRow struct {
	ProgramID string
	ChosenAt  time.Time
}

// PlanFocusRepository stores one chosen plan per member (absent: none).
type PlanFocusRepository interface {
	Get(ctx context.Context, userID string) (*PlanFocusRow, error)
	Set(ctx context.Context, userID, programID string) (*PlanFocusRow, error)
}

type planFocusRepository struct {
	db *gorm.DB
}

func NewPlanFocusRepository(db *gorm.DB) PlanFocusRepository {
	return &planFocusRepository{db: db}
}

func (r *planFocusRepository) Get(ctx context.Context, userID string) (*PlanFocusRow, error) {
	var rows []PlanFocusRow
	if err := r.db.WithContext(ctx).Raw(`SELECT program_id, chosen_at FROM user_plan_focus WHERE user_id = ?`, userID).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load plan focus: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

// Set records the choice now, replacing any earlier one.
func (r *planFocusRepository) Set(ctx context.Context, userID, programID string) (*PlanFocusRow, error) {
	var row PlanFocusRow
	err := r.db.WithContext(ctx).Raw(`
INSERT INTO user_plan_focus (user_id, program_id, chosen_at) VALUES (?, ?, CURRENT_TIMESTAMP)
ON CONFLICT (user_id) DO UPDATE SET program_id = EXCLUDED.program_id, chosen_at = EXCLUDED.chosen_at
RETURNING program_id, chosen_at`, userID, programID).Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("failed to save plan focus: %w", err)
	}
	return &row, nil
}
