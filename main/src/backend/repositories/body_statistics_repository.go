package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MuscleSetRow is one completed set and the muscle data of its exercise, as
// the muscle recovery estimate needs it.
type MuscleSetRow struct {
	CompletedAt     time.Time
	TargetMuscles   string
	PrimaryGroup    string
	SecondaryGroups string
}

// MuscleLoadRepository reads a member's recently completed sets.
type MuscleLoadRepository interface {
	// RecentSets returns the member's completed sets since the given time,
	// from sessions that were not deleted.
	RecentSets(ctx context.Context, userID string, since time.Time) ([]MuscleSetRow, error)
}

type muscleLoadRepository struct {
	db *gorm.DB
}

func NewMuscleLoadRepository(db *gorm.DB) MuscleLoadRepository {
	return &muscleLoadRepository{db: db}
}

func (r *muscleLoadRepository) RecentSets(ctx context.Context, userID string, since time.Time) ([]MuscleSetRow, error) {
	var rows []MuscleSetRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT workout_session_sets.completed_at AS completed_at,
		       COALESCE(exercises.target_muscles, '') AS target_muscles,
		       COALESCE(exercises.primary_muscle_group, '') AS primary_group,
		       COALESCE(exercises.secondary_muscle_groups, '') AS secondary_groups
		FROM workout_session_sets
		JOIN workout_session_exercises ON workout_session_exercises.id = workout_session_sets.workout_session_exercise_id
		     AND workout_session_exercises.deleted_at IS NULL
		JOIN workout_sessions ON workout_sessions.id = workout_session_exercises.workout_session_id
		     AND workout_sessions.deleted_at IS NULL
		LEFT JOIN exercises ON exercises.id = workout_session_exercises.exercise_id
		WHERE workout_sessions.user_id = ?
		  AND workout_session_sets.status = 'completed'
		  AND workout_session_sets.deleted_at IS NULL
		  AND workout_session_sets.completed_at >= ?
		ORDER BY workout_session_sets.completed_at ASC`, userID, since).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("load recent sets: %w", err)
	}
	return rows, nil
}

// BodyMeasurementRow is one logged body measurement.
type BodyMeasurementRow struct {
	ID         string
	Kind       string
	Value      float64
	MeasuredOn time.Time
}

// BodyMeasurementRepository stores a member's body measurements: one value
// per kind and day.
type BodyMeasurementRepository interface {
	// List returns the member's measurements, oldest first, at most limit.
	List(ctx context.Context, userID string, limit int) ([]BodyMeasurementRow, error)
	// Save records the value for the kind and day, replacing the value
	// already logged for that day.
	Save(ctx context.Context, userID, kind string, value float64, measuredOn time.Time) (*BodyMeasurementRow, error)
	// Delete removes one of the member's measurements; false when the member
	// has no measurement with this id.
	Delete(ctx context.Context, userID, id string) (bool, error)
}

type bodyMeasurementRepository struct {
	db *gorm.DB
}

func NewBodyMeasurementRepository(db *gorm.DB) BodyMeasurementRepository {
	return &bodyMeasurementRepository{db: db}
}

func (r *bodyMeasurementRepository) List(ctx context.Context, userID string, limit int) ([]BodyMeasurementRow, error) {
	var rows []BodyMeasurementRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, kind, value, measured_on FROM (
			SELECT id, kind, value::float8 AS value, measured_on
			FROM body_measurements
			WHERE user_id = ?
			ORDER BY measured_on DESC, kind ASC
			LIMIT ?
		) latest
		ORDER BY measured_on ASC, kind ASC`, userID, limit).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list body measurements: %w", err)
	}
	return rows, nil
}

func (r *bodyMeasurementRepository) Save(ctx context.Context, userID, kind string, value float64, measuredOn time.Time) (*BodyMeasurementRow, error) {
	var row BodyMeasurementRow
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO body_measurements (id, user_id, kind, value, measured_on)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (user_id, kind, measured_on) DO UPDATE SET value = EXCLUDED.value
		RETURNING id, kind, value::float8 AS value, measured_on`,
		uuid.NewString(), userID, kind, value, measuredOn.Format("2006-01-02")).
		Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("save body measurement: %w", err)
	}
	return &row, nil
}

func (r *bodyMeasurementRepository) Delete(ctx context.Context, userID, id string) (bool, error) {
	result := r.db.WithContext(ctx).Exec(`DELETE FROM body_measurements WHERE id = ? AND user_id = ?`, id, userID)
	if result.Error != nil {
		return false, fmt.Errorf("delete body measurement: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}
