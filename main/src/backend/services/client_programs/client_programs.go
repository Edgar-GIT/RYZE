package client_programs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"ryze/backend/config"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/exercise_media"
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
	ID               string
	Name             string
	Description      string
	TargetMuscles    string
	Equipment        string
	Difficulty       string
	VideoURL         string
	ImageURL         string
	MediaAttribution string
	MediaAvailable   bool
}

// Set is the safe representation of one prescribed set of an assigned workout
// exercise, in set order.
type Set struct {
	SetNumber   int
	SetType     string
	Reps        *int
	WeightKg    *float64
	RIR         *int
	RPE         *float64
	RestSeconds *int
	Tempo       string
}

// WorkoutExercise is the safe representation of one exercise usage inside an
// assigned workout. It carries the position, the trainer-owned instructions and
// notes, the set prescription, the lifecycle timestamps and the safe exercise
// data; parent and internal identifiers are never exposed.
type WorkoutExercise struct {
	ID           string
	Position     int
	Instructions string
	Notes        string
	Exercise     Exercise
	Sets         []Set
	CreatedAt    time.Time
	UpdatedAt    time.Time
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
	ID              string
	Name            string
	Description     string
	Type            string
	Status          string
	PriceMinorUnits int64
	Currency        string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Weeks           []Week
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
	media    exercise_media.Resolver
}

func NewService(programs ProgramRepository, media config.ExerciseMediaConfig) Service {
	return &service{programs: programs, media: exercise_media.New(media)}
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
	return s.newProgram(program), nil
}

func (s *service) newProgram(model *models.Program) *Program {
	program := &Program{
		ID:              model.ID,
		Name:            model.Name,
		Description:     model.Description,
		Type:            model.Type,
		Status:          model.Status,
		PriceMinorUnits: model.PriceMinorUnits,
		Currency:        model.Currency,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
		Weeks:           make([]Week, 0, len(model.Weeks)),
	}
	for i := range model.Weeks {
		program.Weeks = append(program.Weeks, s.newWeek(&model.Weeks[i]))
	}
	return program
}

func (s *service) newWeek(model *models.ProgramWeek) Week {
	week := Week{
		ID:         model.ID,
		WeekNumber: model.WeekNumber,
		Workouts:   make([]Workout, 0, len(model.Workouts)),
		CreatedAt:  model.CreatedAt,
		UpdatedAt:  model.UpdatedAt,
	}
	for i := range model.Workouts {
		week.Workouts = append(week.Workouts, s.newWorkout(&model.Workouts[i]))
	}
	return week
}

func (s *service) newWorkout(model *models.ProgramWorkout) Workout {
	workout := Workout{
		ID:        model.ID,
		Position:  model.Position,
		Exercises: make([]WorkoutExercise, 0, len(model.Exercises)),
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
	for i := range model.Exercises {
		workout.Exercises = append(workout.Exercises, s.newWorkoutExercise(&model.Exercises[i]))
	}
	return workout
}

func (s *service) newWorkoutExercise(model *models.WorkoutExercise) WorkoutExercise {
	sets := make([]Set, 0, len(model.Sets))
	for i := range model.Sets {
		set := &model.Sets[i]
		sets = append(sets, Set{
			SetNumber:   set.SetNumber,
			SetType:     set.SetType,
			Reps:        set.Reps,
			WeightKg:    set.WeightKg,
			RIR:         set.RIR,
			RPE:         set.RPE,
			RestSeconds: set.RestSeconds,
			Tempo:       set.Tempo,
		})
	}
	return WorkoutExercise{
		ID:           model.ID,
		Position:     model.Position,
		Instructions: model.Instructions,
		Notes:        model.Notes,
		Exercise:     s.newExercise(model.Exercise),
		Sets:         sets,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}

func (s *service) newExercise(model *models.Exercise) Exercise {
	if model == nil {
		return Exercise{}
	}
	media := s.media.Resolve(model.VideoURL, model.ImageURL, model.AnimationURL, model.MediaAttribution)
	return Exercise{
		ID:               model.ID,
		Name:             model.Name,
		Description:      model.Description,
		TargetMuscles:    model.TargetMuscles,
		Equipment:        model.Equipment,
		Difficulty:       model.Difficulty,
		VideoURL:         media.VideoURL,
		ImageURL:         media.ImageURL,
		MediaAttribution: media.Attribution,
		MediaAvailable:   media.Available,
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
