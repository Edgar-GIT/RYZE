package statistics

import (
	"context"
	"errors"
	"fmt"

	"ryze/backend/repositories"
)

// ErrInvalidInput indicates the input was malformed or incomplete.
var ErrInvalidInput = errors.New("invalid statistics input")

// StatisticsRepository is the data-access surface required by the statistics
// service. Every operation is scoped to the authenticated user id, which always
// comes from the authentication context and is never accepted from the client.
type StatisticsRepository interface {
	GetClientStats(ctx context.Context, userID string) (repositories.ClientStatistics, error)
}

// ClientStatisticsResponse is the safe representation of a client's workout
// statistics. It carries only derived aggregates and the current program
// reference; the owning user id, assignment id and deletion markers are never
// exposed.
type ClientStatisticsResponse struct {
	HasActiveAssignment   bool    `json:"has_active_assignment"`
	CurrentProgramName    string  `json:"current_program_name"`
	TotalExecutions       int64   `json:"total_executions"`
	UniqueWorkoutsCompleted int64 `json:"unique_workouts_completed"`
	TotalWorkoutsInProgram int64  `json:"total_workouts_in_program"`
	CompletionPercentage  float64 `json:"completion_percentage"`
	LastWorkoutDate       *string `json:"last_workout_date"`
}

// Service implements the client-facing statistics flow. The requesting user
// identity always comes from the authentication context and is never accepted
// from the client. This service never knows about HTTP, Gin or the
// authentication context.
type Service interface {
	// GetClientStatistics returns the computed workout statistics for the
	// authenticated client. The user identity always comes from the
	// authentication context.
	GetClientStatistics(ctx context.Context, userID string) (*ClientStatisticsResponse, error)
}

type service struct {
	stats StatisticsRepository
}

func NewService(stats StatisticsRepository) Service {
	return &service{stats: stats}
}

func (s *service) GetClientStatistics(ctx context.Context, userID string) (*ClientStatisticsResponse, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	raw, err := s.stats.GetClientStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to compute client statistics: %w", err)
	}

	return toResponse(raw), nil
}

func toResponse(raw repositories.ClientStatistics) *ClientStatisticsResponse {
	var pct float64
	if raw.TotalWorkoutsInProgram > 0 {
		pct = float64(raw.UniqueWorkoutsCompleted) / float64(raw.TotalWorkoutsInProgram) * 100
	}

	return &ClientStatisticsResponse{
		HasActiveAssignment:     raw.HasActiveAssignment,
		CurrentProgramName:      raw.CurrentProgramName,
		TotalExecutions:         raw.TotalExecutions,
		UniqueWorkoutsCompleted: raw.UniqueWorkoutsCompleted,
		TotalWorkoutsInProgram:  raw.TotalWorkoutsInProgram,
		CompletionPercentage:    pct,
		LastWorkoutDate:         raw.LastWorkoutDate,
	}
}

func validateUserID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	return nil
}
