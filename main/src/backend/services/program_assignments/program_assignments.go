package program_assignments

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

var (
	// ErrInvalidInput indicates the assignment input was malformed or
	// incomplete.
	ErrInvalidInput = errors.New("invalid program assignment input")
	// ErrClientRelationNotFound indicates the trainer has no active
	// relationship with the client user, so no assignment can ever be created,
	// listed, read or removed for that pair.
	ErrClientRelationNotFound = errors.New("trainer-client relationship not found")
	// ErrProgramNotFound indicates the program does not exist, is soft-deleted
	// or is not owned by the trainer.
	ErrProgramNotFound = errors.New("program not found")
	// ErrAssignmentNotFound indicates the program assignment does not exist, is
	// soft-deleted, belongs to another client or the owning relationship is not
	// active.
	ErrAssignmentNotFound = errors.New("program assignment not found")
	// ErrAssignmentAlreadyActive indicates the trainer already assigned an
	// active program to this client. A trainer can have at most one active
	// assigned program per client.
	ErrAssignmentAlreadyActive = errors.New("program already assigned to this client")
)

// ProgramAssignmentRepository is the data-access surface required by the
// program assignment service. Every operation is scoped to an explicit trainer
// id; the repository never obtains it from an HTTP context.
type ProgramAssignmentRepository interface {
	Create(ctx context.Context, trainerID, userID, programID string, assignment *models.ProgramAssignment) error
	ListByClient(ctx context.Context, trainerID, userID string) ([]models.ProgramAssignment, error)
	FindByIDAndClient(ctx context.Context, trainerID, userID, assignmentID string) (*models.ProgramAssignment, error)
	SoftDelete(ctx context.Context, trainerID, userID, assignmentID string) error
}

// Program is the safe representation of the assigned program. It carries only
// the public product metadata and never exposes deletion markers or any
// internal data.
type Program struct {
	ID          string
	TrainerID   string
	Name        string
	Description string
	Type        string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Assignment is the safe representation of one trainer → client program
// assignment. It carries only the relationship identity, the safe program data
// and the lifecycle timestamps; deletion markers and any internal data are
// never exposed.
type Assignment struct {
	ID        string
	TrainerID string
	ProgramID string
	UserID    string
	Program   Program
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Service implements the trainer-owned client program assignment flow.
// Authorization (which trainer may operate) is enforced by the route
// middleware; ownership is guaranteed because the trainer id always comes from
// the authenticated trainer context and is never accepted from the client.
// This service never knows about HTTP, Gin or the trainer context.
type Service interface {
	AssignProgram(ctx context.Context, trainerID, userID, programID string) (*Assignment, error)
	ListClientPrograms(ctx context.Context, trainerID, userID string) ([]Assignment, error)
	RemoveAssignment(ctx context.Context, trainerID, userID, assignmentID string) error
}

type service struct {
	assignments ProgramAssignmentRepository
}

func NewService(assignments ProgramAssignmentRepository) Service {
	return &service{assignments: assignments}
}

// AssignProgram assigns one of the trainer's own active programs to one of the
// trainer's active clients. The trainer id always comes from the caller and is
// never accepted from the client. The client must be an active client of the
// trainer, the program must be active and owned by the trainer, and the
// trainer can never assign a program to a client who already holds an active
// assignment.
func (s *service) AssignProgram(ctx context.Context, trainerID, userID, programID string) (*Assignment, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}

	assignment := &models.ProgramAssignment{}
	if err := s.assignments.Create(ctx, trainerID, userID, programID, assignment); err != nil {
		switch {
		case errors.Is(err, repositories.ErrClientRelationNotFound):
			return nil, ErrClientRelationNotFound
		case errors.Is(err, repositories.ErrProgramNotFound):
			return nil, ErrProgramNotFound
		case errors.Is(err, repositories.ErrAssignmentAlreadyActive):
			return nil, ErrAssignmentAlreadyActive
		default:
			return nil, fmt.Errorf("failed to assign program: %w", err)
		}
	}
	return newAssignment(assignment), nil
}

// ListClientPrograms returns every active program assignment of one of the
// authenticated trainer's active clients, in assignment order, each with its
// safe program data. A client that does not exist, is not managed by the
// trainer or whose relationship was soft-deleted is indistinguishable from a
// client without assignments.
func (s *service) ListClientPrograms(ctx context.Context, trainerID, userID string) ([]Assignment, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	models, err := s.assignments.ListByClient(ctx, trainerID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list client programs: %w", err)
	}

	assignments := make([]Assignment, 0, len(models))
	for i := range models {
		assignments = append(assignments, *newAssignment(&models[i]))
	}
	return assignments, nil
}

// RemoveAssignment soft-deletes one of the authenticated trainer's own client
// assignments. Only the assignment row is touched; the program and the
// trainer-client relationship are never touched. An assignment that is missing,
// soft-deleted, belongs to another client or whose relationship is not active
// is indistinguishable and never revealed.
func (s *service) RemoveAssignment(ctx context.Context, trainerID, userID, assignmentID string) error {
	if err := validateTrainerID(trainerID); err != nil {
		return err
	}
	if err := validateUserID(userID); err != nil {
		return err
	}
	if err := validateAssignmentID(assignmentID); err != nil {
		return err
	}

	if err := s.assignments.SoftDelete(ctx, trainerID, userID, assignmentID); err != nil {
		switch {
		case errors.Is(err, repositories.ErrAssignmentNotFound):
			return ErrAssignmentNotFound
		default:
			return fmt.Errorf("failed to remove program assignment: %w", err)
		}
	}
	return nil
}

func newAssignment(model *models.ProgramAssignment) *Assignment {
	return &Assignment{
		ID:        model.ID,
		TrainerID: model.TrainerID,
		ProgramID: model.ProgramID,
		UserID:    model.UserID,
		Program: Program{
			ID:          model.Program.ID,
			TrainerID:   model.Program.TrainerID,
			Name:        model.Program.Name,
			Description: model.Program.Description,
			Type:        model.Program.Type,
			Status:      model.Program.Status,
			CreatedAt:   model.Program.CreatedAt,
			UpdatedAt:   model.Program.UpdatedAt,
		},
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
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

func validateTrainerID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: trainer id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid trainer id", ErrInvalidInput)
	}
	return nil
}

func validateProgramID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: program id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid program id", ErrInvalidInput)
	}
	return nil
}

func validateAssignmentID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: assignment id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid assignment id", ErrInvalidInput)
	}
	return nil
}
