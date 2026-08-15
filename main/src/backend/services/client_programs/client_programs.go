package client_programs

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

// Exercise is the safe catalog summary exposed to the client. It carries only
// the descriptive metadata of the public exercise catalog.
type Exercise struct {
	ID            string
	Name          string
	Description   string
	TargetMuscles string
	Equipment     string
	Difficulty    string
	VideoURL      string
	ImageURL      string
}

// WorkoutExercise is the safe representation of one exercise usage inside an
// assigned workout. It carries only the position, the lifecycle timestamps and
// the safe exercise data; parent and internal identifiers are never exposed.
type WorkoutExercise struct {
	ID        string
	Position  int
	Exercise  Exercise
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Workout is the safe representation of one workout slot inside an assigned
// program week, in position order, with its safe workout exercises.
type Workout struct {
	ID        string
	Position  int
	Exercises []WorkoutExercise
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Week is the safe representation of one week slot inside the assigned program,
// in week order, with its safe workouts.
type Week struct {
	ID         string
	WeekNumber int
	Workouts   []Workout
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Program is the safe representation of the client's assigned program with its
// full active structure. It carries only public product metadata and never
// exposes the owning trainer, parent identifiers, deletion markers or any
// internal data.
type Program struct {
	ID          string
	Name        string
	Description string
	Type        string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Weeks       []Week
}

// Service implements the client-facing assigned program read flow. The
// requesting user identity always comes from the authentication context and is
// never accepted from the client. This service never knows about HTTP, Gin or
// the authentication context.
type Service interface {
	GetProgram(ctx context.Context, userID string) (*Program, error)
}

type service struct {
	programs ProgramRepository
}

func NewService(programs ProgramRepository) Service {
	return &service{programs: programs}
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
	return newProgram(program), nil
}

func newProgram(model *models.Program) *Program {
	program := &Program{
		ID:          model.ID,
		Name:        model.Name,
		Description: model.Description,
		Type:        model.Type,
		Status:      model.Status,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
		Weeks:       make([]Week, 0, len(model.Weeks)),
	}
	for i := range model.Weeks {
		program.Weeks = append(program.Weeks, newWeek(&model.Weeks[i]))
	}
	return program
}

func newWeek(model *models.ProgramWeek) Week {
	week := Week{
		ID:         model.ID,
		WeekNumber: model.WeekNumber,
		Workouts:   make([]Workout, 0, len(model.Workouts)),
		CreatedAt:  model.CreatedAt,
		UpdatedAt:  model.UpdatedAt,
	}
	for i := range model.Workouts {
		week.Workouts = append(week.Workouts, newWorkout(&model.Workouts[i]))
	}
	return week
}

func newWorkout(model *models.ProgramWorkout) Workout {
	workout := Workout{
		ID:        model.ID,
		Position:  model.Position,
		Exercises: make([]WorkoutExercise, 0, len(model.Exercises)),
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
	for i := range model.Exercises {
		workout.Exercises = append(workout.Exercises, newWorkoutExercise(&model.Exercises[i]))
	}
	return workout
}

func newWorkoutExercise(model *models.WorkoutExercise) WorkoutExercise {
	return WorkoutExercise{
		ID:        model.ID,
		Position:  model.Position,
		Exercise:  newExercise(model.Exercise),
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

func newExercise(model *models.Exercise) Exercise {
	return Exercise{
		ID:            model.ID,
		Name:          model.Name,
		Description:   model.Description,
		TargetMuscles: model.TargetMuscles,
		Equipment:     model.Equipment,
		Difficulty:    model.Difficulty,
		VideoURL:      model.VideoURL,
		ImageURL:      model.ImageURL,
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
