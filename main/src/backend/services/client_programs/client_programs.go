package client_programs

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"ryze/backend/config"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/program_content"
)

var (
	// ErrInvalidInput indicates the input was malformed or incomplete.
	ErrInvalidInput = errors.New("invalid client program input")
	// ErrProgramNotFound indicates the user has no active program assignment
	// or its assigned program is not readable. Both cases are deliberately
	// indistinguishable to the client.
	ErrProgramNotFound = errors.New("assigned program not found")
)

// ProgramRepository is the data-access surface required by the client program
// service. Every operation is scoped to the authenticated user id, which always
// comes from the authentication context and is never accepted from the client.
type ProgramRepository interface {
	FindAssignedProgram(ctx context.Context, userID string) (*models.Program, error)
}

// The safe structure is shared with every authenticated access surface so the
// same program can never be exposed differently depending on the endpoint.
type (
	Exercise        = program_content.Exercise
	Set             = program_content.Set
	WorkoutExercise = program_content.WorkoutExercise
	Workout         = program_content.Workout
	Week            = program_content.Week
	Program         = program_content.Program
)

// Service implements the client-facing assigned program read flow. The
// requesting user identity always comes from the authentication context and is
// never accepted from the client. This service never knows about HTTP, Gin or
// the authentication context.
type Service interface {
	GetProgram(ctx context.Context, userID string) (*Program, error)
}

type service struct {
	programs ProgramRepository
	content  *program_content.Builder
}

func NewService(programs ProgramRepository, media config.ExerciseMediaConfig) Service {
	return &service{programs: programs, content: program_content.NewBuilder(media)}
}

// GetProgram returns the full active structure of the most recent program
// assigned to the authenticated user, or a single indistinguishable not-found
// error when the user has no active assignment or the assigned program is not
// readable. The program status carries no access semantics, so a draft program
// is as readable as a published one.
func (s *service) GetProgram(ctx context.Context, userID string) (*Program, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	program, err := s.programs.FindAssignedProgram(ctx, userID)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrAssignmentNotFound), errors.Is(err, repositories.ErrProgramNotFound):
			return nil, ErrProgramNotFound
		default:
			return nil, fmt.Errorf("failed to get the assigned program: %w", err)
		}
	}
	return s.content.Build(program), nil
}

func validateUserID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid user id", ErrInvalidInput)
	}
	return nil
}
