package program_structure

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
	// ErrInvalidInput indicates the request identifiers or the order list were
	// malformed or incomplete.
	ErrInvalidInput = errors.New("invalid program structure input")
	// ErrProgramNotFound indicates the program does not exist, is soft-deleted
	// or is not owned by the trainer performing the operation.
	ErrProgramNotFound = errors.New("program not found")
	// ErrWeekNotFound indicates the week does not exist, is soft-deleted, does
	// not belong to the requested program or the program is not owned by the
	// trainer.
	ErrWeekNotFound = errors.New("program week not found")
	// ErrWorkoutNotFound indicates the workout does not exist, is soft-deleted,
	// does not belong to the requested week or the owning chain is not owned by
	// the trainer.
	ErrWorkoutNotFound = errors.New("program workout not found")
	// ErrReorderConflict indicates the order list is not an exact permutation
	// of the active entries of the target week or program.
	ErrReorderConflict = errors.New("order list mismatch")
)

// WeekRepository is the data-access surface required for the program week
// operations. Every operation is scoped to an explicit trainer id; the
// repository never obtains it from an HTTP context.
type WeekRepository interface {
	Create(ctx context.Context, trainerID, programID string, week *models.ProgramWeek) error
	ListByProgram(ctx context.Context, trainerID, programID string) ([]models.ProgramWeek, error)
	FindByIDAndProgram(ctx context.Context, trainerID, programID, weekID string) (*models.ProgramWeek, error)
	Reorder(ctx context.Context, trainerID, programID string, orderedIDs []string) error
	SoftDelete(ctx context.Context, trainerID, programID, weekID string) error
}

// WorkoutRepository is the data-access surface required for the program workout
// operations. Every operation is scoped to an explicit trainer id; the
// repository never obtains it from an HTTP context.
type WorkoutRepository interface {
	Create(ctx context.Context, trainerID, programID, weekID string, workout *models.ProgramWorkout) error
	ListByWeek(ctx context.Context, trainerID, programID, weekID string) ([]models.ProgramWorkout, error)
	FindByIDAndWeek(ctx context.Context, trainerID, programID, weekID, workoutID string) (*models.ProgramWorkout, error)
	Reorder(ctx context.Context, trainerID, programID, weekID string, orderedIDs []string) error
	SoftDelete(ctx context.Context, trainerID, programID, weekID, workoutID string) error
}

// Week is the safe representation of one program week. It carries only public
// structural metadata and never exposes deletion markers.
type Week struct {
	ID         string
	ProgramID  string
	WeekNumber int
	Workouts   []Workout
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Workout is the safe representation of one program workout. It carries only
// public structural metadata and never exposes deletion markers.
type Workout struct {
	ID            string
	ProgramWeekID string
	Position      int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Service implements the trainer-owned program structure flow (weeks and
// workouts). Authorization (which trainer may operate) is enforced by the route
// middleware; ownership is guaranteed because the trainer id always comes from
// the authenticated trainer context and is never accepted from the client. This
// service never knows about HTTP, Gin or the trainer context.
type Service interface {
	CreateWeek(ctx context.Context, trainerID, programID string) (*Week, error)
	ListWeeks(ctx context.Context, trainerID, programID string) ([]Week, error)
	GetWeek(ctx context.Context, trainerID, programID, weekID string) (*Week, error)
	ReorderWeeks(ctx context.Context, trainerID, programID string, orderedIDs []string) error
	DeleteWeek(ctx context.Context, trainerID, programID, weekID string) error

	CreateWorkout(ctx context.Context, trainerID, programID, weekID string) (*Workout, error)
	ListWorkouts(ctx context.Context, trainerID, programID, weekID string) ([]Workout, error)
	GetWorkout(ctx context.Context, trainerID, programID, weekID, workoutID string) (*Workout, error)
	ReorderWorkouts(ctx context.Context, trainerID, programID, weekID string, orderedIDs []string) error
	DeleteWorkout(ctx context.Context, trainerID, programID, weekID, workoutID string) error
}

type service struct {
	weeks    WeekRepository
	workouts WorkoutRepository
}

func NewService(weeks WeekRepository, workouts WorkoutRepository) Service {
	return &service{weeks: weeks, workouts: workouts}
}

// CreateWeek appends a new empty week to the end of one of the authenticated
// trainer's own programs. The trainer id always comes from the caller (the
// authenticated trainer context) and is never accepted from the client; the
// program must be active and owned by the trainer.
func (s *service) CreateWeek(ctx context.Context, trainerID, programID string) (*Week, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return nil, err
	}
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}

	week := &models.ProgramWeek{}
	if err := s.weeks.Create(ctx, trainerID, programID, week); err != nil {
		if errors.Is(err, repositories.ErrProgramNotFound) {
			return nil, ErrProgramNotFound
		}
		return nil, fmt.Errorf("failed to create program week: %w", err)
	}
	return newWeek(week), nil
}

// ListWeeks returns every active week of one of the authenticated trainer's own
// programs, in week order, with each week's active workouts in position order.
// A missing, soft-deleted or foreign program is indistinguishable from a program
// without weeks.
func (s *service) ListWeeks(ctx context.Context, trainerID, programID string) ([]Week, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return nil, err
	}
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}

	models, err := s.weeks.ListByProgram(ctx, trainerID, programID)
	if err != nil {
		return nil, fmt.Errorf("failed to list program weeks: %w", err)
	}

	weeks := make([]Week, 0, len(models))
	for i := range models {
		weeks = append(weeks, *newWeek(&models[i]))
	}
	return weeks, nil
}

// GetWeek returns one active week of one of the authenticated trainer's own
// programs, with its active workouts in position order. The query is scoped by
// the trainer, the program and the week id, so a week that is missing,
// soft-deleted, under another program or under a foreign program is
// indistinguishable and never revealed.
func (s *service) GetWeek(ctx context.Context, trainerID, programID, weekID string) (*Week, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return nil, err
	}
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}
	if err := validateWeekID(weekID); err != nil {
		return nil, err
	}

	week, err := s.weeks.FindByIDAndProgram(ctx, trainerID, programID, weekID)
	if err != nil {
		if errors.Is(err, repositories.ErrWeekNotFound) {
			return nil, ErrWeekNotFound
		}
		return nil, fmt.Errorf("failed to load program week: %w", err)
	}
	return newWeek(week), nil
}

// ReorderWeeks replaces the week_number of every active week of one of the
// authenticated trainer's own programs with the position of its id inside
// orderedIDs. The list must be non-empty, contain only valid identifiers without
// duplicates and match every active week exactly once; the exact permutation is
// verified by the repository before any write.
func (s *service) ReorderWeeks(ctx context.Context, trainerID, programID string, orderedIDs []string) error {
	if err := validateTrainerID(trainerID); err != nil {
		return err
	}
	if err := validateProgramID(programID); err != nil {
		return err
	}
	if err := validateOrderList(orderedIDs); err != nil {
		return err
	}

	if err := s.weeks.Reorder(ctx, trainerID, programID, orderedIDs); err != nil {
		switch {
		case errors.Is(err, repositories.ErrProgramNotFound):
			return ErrProgramNotFound
		case errors.Is(err, repositories.ErrWeekReorderConflict):
			return ErrReorderConflict
		default:
			return fmt.Errorf("failed to reorder program weeks: %w", err)
		}
	}
	return nil
}

// DeleteWeek soft-deletes one of the authenticated trainer's own program weeks.
// Only the week row is touched; the program and the week's workouts are never
// deleted, and the workouts simply become unreachable through their inactive
// week.
func (s *service) DeleteWeek(ctx context.Context, trainerID, programID, weekID string) error {
	if err := validateTrainerID(trainerID); err != nil {
		return err
	}
	if err := validateProgramID(programID); err != nil {
		return err
	}
	if err := validateWeekID(weekID); err != nil {
		return err
	}

	if err := s.weeks.SoftDelete(ctx, trainerID, programID, weekID); err != nil {
		switch {
		case errors.Is(err, repositories.ErrWeekNotFound):
			return ErrWeekNotFound
		default:
			return fmt.Errorf("failed to delete program week: %w", err)
		}
	}
	return nil
}

// CreateWorkout appends a new empty workout to the end of one of the
// authenticated trainer's own program weeks. The trainer id always comes from
// the caller and is never accepted from the client; the week and its program
// must be active and owned by the trainer.
func (s *service) CreateWorkout(ctx context.Context, trainerID, programID, weekID string) (*Workout, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return nil, err
	}
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}
	if err := validateWeekID(weekID); err != nil {
		return nil, err
	}

	workout := &models.ProgramWorkout{}
	if err := s.workouts.Create(ctx, trainerID, programID, weekID, workout); err != nil {
		if errors.Is(err, repositories.ErrWeekNotFound) {
			return nil, ErrWeekNotFound
		}
		return nil, fmt.Errorf("failed to create program workout: %w", err)
	}
	return newWorkout(workout), nil
}

// ListWorkouts returns every active workout of one of the authenticated
// trainer's own program weeks, in position order. A missing, soft-deleted or
// foreign week is indistinguishable from a week without workouts.
func (s *service) ListWorkouts(ctx context.Context, trainerID, programID, weekID string) ([]Workout, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return nil, err
	}
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}
	if err := validateWeekID(weekID); err != nil {
		return nil, err
	}

	models, err := s.workouts.ListByWeek(ctx, trainerID, programID, weekID)
	if err != nil {
		return nil, fmt.Errorf("failed to list program workouts: %w", err)
	}

	workouts := make([]Workout, 0, len(models))
	for i := range models {
		workouts = append(workouts, *newWorkout(&models[i]))
	}
	return workouts, nil
}

// GetWorkout returns one active workout of one of the authenticated trainer's
// own program weeks. The query is scoped by the trainer, the program, the week
// and the workout id, so a workout that is missing, soft-deleted, under another
// week or under a foreign program is indistinguishable and never revealed.
func (s *service) GetWorkout(ctx context.Context, trainerID, programID, weekID, workoutID string) (*Workout, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return nil, err
	}
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}
	if err := validateWeekID(weekID); err != nil {
		return nil, err
	}
	if err := validateWorkoutID(workoutID); err != nil {
		return nil, err
	}

	workout, err := s.workouts.FindByIDAndWeek(ctx, trainerID, programID, weekID, workoutID)
	if err != nil {
		if errors.Is(err, repositories.ErrWorkoutNotFound) {
			return nil, ErrWorkoutNotFound
		}
		return nil, fmt.Errorf("failed to load program workout: %w", err)
	}
	return newWorkout(workout), nil
}

// ReorderWorkouts replaces the position of every active workout of one of the
// authenticated trainer's own program weeks with the index of its id inside
// orderedIDs. The list must be non-empty, contain only valid identifiers without
// duplicates and match every active workout exactly once; the exact permutation
// is verified by the repository before any write.
func (s *service) ReorderWorkouts(ctx context.Context, trainerID, programID, weekID string, orderedIDs []string) error {
	if err := validateTrainerID(trainerID); err != nil {
		return err
	}
	if err := validateProgramID(programID); err != nil {
		return err
	}
	if err := validateWeekID(weekID); err != nil {
		return err
	}
	if err := validateOrderList(orderedIDs); err != nil {
		return err
	}

	if err := s.workouts.Reorder(ctx, trainerID, programID, weekID, orderedIDs); err != nil {
		switch {
		case errors.Is(err, repositories.ErrWeekNotFound):
			return ErrWeekNotFound
		case errors.Is(err, repositories.ErrWorkoutReorderConflict):
			return ErrReorderConflict
		default:
			return fmt.Errorf("failed to reorder program workouts: %w", err)
		}
	}
	return nil
}

// DeleteWorkout soft-deletes one of the authenticated trainer's own program
// workouts. Only the workout row is touched; the week and the program are never
// touched.
func (s *service) DeleteWorkout(ctx context.Context, trainerID, programID, weekID, workoutID string) error {
	if err := validateTrainerID(trainerID); err != nil {
		return err
	}
	if err := validateProgramID(programID); err != nil {
		return err
	}
	if err := validateWeekID(weekID); err != nil {
		return err
	}
	if err := validateWorkoutID(workoutID); err != nil {
		return err
	}

	if err := s.workouts.SoftDelete(ctx, trainerID, programID, weekID, workoutID); err != nil {
		switch {
		case errors.Is(err, repositories.ErrWorkoutNotFound):
			return ErrWorkoutNotFound
		default:
			return fmt.Errorf("failed to delete program workout: %w", err)
		}
	}
	return nil
}

func newWeek(model *models.ProgramWeek) *Week {
	workouts := make([]Workout, 0, len(model.Workouts))
	for i := range model.Workouts {
		workouts = append(workouts, *newWorkout(&model.Workouts[i]))
	}
	return &Week{
		ID:         model.ID,
		ProgramID:  model.ProgramID,
		WeekNumber: model.WeekNumber,
		Workouts:   workouts,
		CreatedAt:  model.CreatedAt,
		UpdatedAt:  model.UpdatedAt,
	}
}

func newWorkout(model *models.ProgramWorkout) *Workout {
	return &Workout{
		ID:            model.ID,
		ProgramWeekID: model.ProgramWeekID,
		Position:      model.Position,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}
}

// validateOrderList rejects empty reorder lists and lists carrying invalid or
// duplicated identifiers before any database access.
func validateOrderList(orderedIDs []string) error {
	if len(orderedIDs) == 0 {
		return fmt.Errorf("%w: the order list cannot be empty", ErrInvalidInput)
	}
	seen := make(map[string]struct{}, len(orderedIDs))
	for _, id := range orderedIDs {
		if id == "" {
			return fmt.Errorf("%w: order list contains an empty identifier", ErrInvalidInput)
		}
		if _, err := uuid.Parse(id); err != nil {
			return fmt.Errorf("%w: order list contains an invalid identifier", ErrInvalidInput)
		}
		if _, exists := seen[id]; exists {
			return fmt.Errorf("%w: order list contains duplicated identifier %q", ErrInvalidInput, id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

// validateTrainerID rejects empty and malformed identifiers before any database
// access.
func validateTrainerID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: trainer id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid trainer id", ErrInvalidInput)
	}
	return nil
}

// validateProgramID rejects empty and malformed identifiers before any database
// access.
func validateProgramID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: program id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid program id", ErrInvalidInput)
	}
	return nil
}

// validateWeekID rejects empty and malformed identifiers before any database
// access.
func validateWeekID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: week id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid week id", ErrInvalidInput)
	}
	return nil
}

// validateWorkoutID rejects empty and malformed identifiers before any database
// access.
func validateWorkoutID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: workout id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid workout id", ErrInvalidInput)
	}
	return nil
}
