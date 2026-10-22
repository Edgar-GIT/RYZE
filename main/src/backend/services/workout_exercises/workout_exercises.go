package workout_exercises

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/training_prescription"
)

var (
	// ErrInvalidInput indicates the request identifiers or the order list were
	// malformed or incomplete.
	ErrInvalidInput = errors.New("invalid workout exercise input")
	// ErrWorkoutNotFound indicates the workout does not exist, is soft-deleted,
	// does not belong to the requested week or the owning chain is not owned by
	// the trainer.
	ErrWorkoutNotFound = errors.New("program workout not found")
	// ErrExerciseNotFound indicates the exercise does not exist or is
	// soft-deleted in the global catalog and can therefore never be assigned.
	ErrExerciseNotFound = errors.New("exercise not found")
	// ErrWorkoutExerciseNotFound indicates the workout exercise does not exist,
	// is soft-deleted, does not belong to the requested workout or the owning
	// chain is not owned by the trainer.
	ErrWorkoutExerciseNotFound = errors.New("workout exercise not found")
	// ErrReorderConflict indicates the order list is not an exact permutation
	// of the active workout exercises of the target workout.
	ErrReorderConflict = errors.New("order list mismatch")
)

// WorkoutExerciseRepository is the data-access surface required for the
// workout exercise operations. Every operation is scoped to an explicit
// trainer id; the repository never obtains it from an HTTP context.
type WorkoutExerciseRepository interface {
	AddExercise(ctx context.Context, trainerID, programID, weekID, workoutID string, workoutExercise *models.WorkoutExercise) error
	ListByWorkout(ctx context.Context, trainerID, programID, weekID, workoutID string) ([]models.WorkoutExercise, error)
	FindByIDAndWorkout(ctx context.Context, trainerID, programID, weekID, workoutID, workoutExerciseID string) (*models.WorkoutExercise, error)
	Reorder(ctx context.Context, trainerID, programID, weekID, workoutID string, orderedIDs []string) error
	UpdatePrescription(ctx context.Context, trainerID, programID, weekID, workoutID, workoutExerciseID string, instructions, notes string, sets []models.WorkoutExerciseSet) (*models.WorkoutExercise, error)
	SoftDelete(ctx context.Context, trainerID, programID, weekID, workoutID, workoutExerciseID string) error
}

// Exercise is the safe representation of the global catalog exercise assigned
// to a workout. It carries only the public descriptive metadata of the catalog
// and never exposes deletion markers or any internal data.
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

// SetInput carries one desired prescription set. Set numbers are derived from
// the array order and never accepted from the client.
type SetInput = training_prescription.SetInput

// Set is the safe representation of one prescription set.
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

// WorkoutExercise is the safe representation of one exercise usage inside a
// program workout. It carries the public structural metadata, the per-assignment
// instructions and notes, the set prescription and the safe catalog data of the
// assigned exercise. Deletion markers and any internal data are never exposed.
type WorkoutExercise struct {
	ID               string
	ProgramWorkoutID string
	Position         int
	Instructions     string
	Notes            string
	Exercise         Exercise
	Sets             []Set
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Service implements the trainer-owned workout exercise flow. Authorization
// (which trainer may operate) is enforced by the route middleware; ownership
// is guaranteed because the trainer id always comes from the authenticated
// trainer context and is never accepted from the client. This service never
// knows about HTTP, Gin or the trainer context.
type Service interface {
	AddExercise(ctx context.Context, trainerID, programID, weekID, workoutID, exerciseID string) (*WorkoutExercise, error)
	ListExercises(ctx context.Context, trainerID, programID, weekID, workoutID string) ([]WorkoutExercise, error)
	GetExercise(ctx context.Context, trainerID, programID, weekID, workoutID, workoutExerciseID string) (*WorkoutExercise, error)
	UpdatePrescription(ctx context.Context, trainerID, programID, weekID, workoutID, workoutExerciseID string, instructions, notes string, sets []SetInput) (*WorkoutExercise, error)
	ReorderExercises(ctx context.Context, trainerID, programID, weekID, workoutID string, orderedIDs []string) error
	RemoveExercise(ctx context.Context, trainerID, programID, weekID, workoutID, workoutExerciseID string) error
}

type service struct {
	exercises WorkoutExerciseRepository
}

func NewService(exercises WorkoutExerciseRepository) Service {
	return &service{exercises: exercises}
}

// AddExercise assigns one active catalog exercise to the end of one of the
// authenticated trainer's own program workouts. The trainer id always comes
// from the caller and is never accepted from the client; the workout and its
// owning chain must be active and owned by the trainer, and the exercise must
// exist and be active in the global catalog.
func (s *service) AddExercise(ctx context.Context, trainerID, programID, weekID, workoutID, exerciseID string) (*WorkoutExercise, error) {
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
	if err := validateExerciseID(exerciseID); err != nil {
		return nil, err
	}

	workoutExercise := &models.WorkoutExercise{ExerciseID: exerciseID}
	if err := s.exercises.AddExercise(ctx, trainerID, programID, weekID, workoutID, workoutExercise); err != nil {
		switch {
		case errors.Is(err, repositories.ErrWorkoutNotFound):
			return nil, ErrWorkoutNotFound
		case errors.Is(err, repositories.ErrExerciseNotFound):
			return nil, ErrExerciseNotFound
		default:
			return nil, fmt.Errorf("failed to add workout exercise: %w", err)
		}
	}
	return newWorkoutExercise(workoutExercise), nil
}

// ListExercises returns every active workout exercise of one of the
// authenticated trainer's own program workouts, in position order, each with
// its safe exercise data. A missing, soft-deleted or foreign workout is
// indistinguishable from a workout without exercises.
func (s *service) ListExercises(ctx context.Context, trainerID, programID, weekID, workoutID string) ([]WorkoutExercise, error) {
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

	models, err := s.exercises.ListByWorkout(ctx, trainerID, programID, weekID, workoutID)
	if err != nil {
		return nil, fmt.Errorf("failed to list workout exercises: %w", err)
	}

	exercises := make([]WorkoutExercise, 0, len(models))
	for i := range models {
		exercises = append(exercises, *newWorkoutExercise(&models[i]))
	}
	return exercises, nil
}

// GetExercise returns one active workout exercise of one of the authenticated
// trainer's own program workouts, with its safe exercise data. The query is
// scoped by the trainer, the program, the week, the workout and the workout
// exercise id, so a workout exercise that is missing, soft-deleted, under
// another workout or under a foreign program is indistinguishable and never
// revealed.
func (s *service) GetExercise(ctx context.Context, trainerID, programID, weekID, workoutID, workoutExerciseID string) (*WorkoutExercise, error) {
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
	if err := validateWorkoutExerciseID(workoutExerciseID); err != nil {
		return nil, err
	}

	workoutExercise, err := s.exercises.FindByIDAndWorkout(ctx, trainerID, programID, weekID, workoutID, workoutExerciseID)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrWorkoutExerciseNotFound):
			return nil, ErrWorkoutExerciseNotFound
		default:
			return nil, fmt.Errorf("failed to load workout exercise: %w", err)
		}
	}
	return newWorkoutExercise(workoutExercise), nil
}

// UpdatePrescription replaces the per-assignment instructions, notes and full
// set prescription of one of the authenticated trainer's own workout exercises.
// The assignment and its complete owning chain are verified by the repository
// before any write, so a foreign or soft-deleted entry is indistinguishable
// from a missing one. Set numbers are derived from the array order.
func (s *service) UpdatePrescription(ctx context.Context, trainerID, programID, weekID, workoutID, workoutExerciseID, instructions, notes string, sets []SetInput) (*WorkoutExercise, error) {
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
	if err := validateWorkoutExerciseID(workoutExerciseID); err != nil {
		return nil, err
	}
	if err := validatePrescription(sets); err != nil {
		return nil, err
	}

	setModels := make([]models.WorkoutExerciseSet, 0, len(sets))
	for i := range sets {
		setModels = append(setModels, models.WorkoutExerciseSet{
			SetNumber:   i + 1,
			SetType:     sets[i].SetType,
			Reps:        sets[i].Reps,
			WeightKg:    sets[i].WeightKg,
			RIR:         sets[i].RIR,
			RPE:         sets[i].RPE,
			RestSeconds: sets[i].RestSeconds,
			Tempo:       sets[i].Tempo,
		})
	}

	updated, err := s.exercises.UpdatePrescription(ctx, trainerID, programID, weekID, workoutID, workoutExerciseID, instructions, notes, setModels)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrWorkoutExerciseNotFound):
			return nil, ErrWorkoutExerciseNotFound
		default:
			return nil, fmt.Errorf("failed to update workout exercise prescription: %w", err)
		}
	}
	return newWorkoutExercise(updated), nil
}

// ReorderExercises replaces the position of every active workout exercise of
// one of the authenticated trainer's own program workouts with the index of
// its id inside orderedIDs. The list must be non-empty, contain only valid
// identifiers without duplicates and match every active workout exercise
// exactly once; the exact permutation is verified by the repository before any
// write.
func (s *service) ReorderExercises(ctx context.Context, trainerID, programID, weekID, workoutID string, orderedIDs []string) error {
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
	if err := validateOrderList(orderedIDs); err != nil {
		return err
	}

	if err := s.exercises.Reorder(ctx, trainerID, programID, weekID, workoutID, orderedIDs); err != nil {
		switch {
		case errors.Is(err, repositories.ErrWorkoutNotFound):
			return ErrWorkoutNotFound
		case errors.Is(err, repositories.ErrWorkoutExerciseReorderConflict):
			return ErrReorderConflict
		default:
			return fmt.Errorf("failed to reorder workout exercises: %w", err)
		}
	}
	return nil
}

// RemoveExercise soft-deletes one of the authenticated trainer's own workout
// exercises. Only the workout exercise row is touched; the workout, the week,
// the program and the catalog exercise are never touched.
func (s *service) RemoveExercise(ctx context.Context, trainerID, programID, weekID, workoutID, workoutExerciseID string) error {
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
	if err := validateWorkoutExerciseID(workoutExerciseID); err != nil {
		return err
	}

	if err := s.exercises.SoftDelete(ctx, trainerID, programID, weekID, workoutID, workoutExerciseID); err != nil {
		switch {
		case errors.Is(err, repositories.ErrWorkoutExerciseNotFound):
			return ErrWorkoutExerciseNotFound
		default:
			return fmt.Errorf("failed to remove workout exercise: %w", err)
		}
	}
	return nil
}

func newWorkoutExercise(model *models.WorkoutExercise) *WorkoutExercise {
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

	result := &WorkoutExercise{
		ID:               model.ID,
		ProgramWorkoutID: model.ProgramWorkoutID,
		Position:         model.Position,
		Instructions:     model.Instructions,
		Notes:            model.Notes,
		Sets:             sets,
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
	}
	if model.Exercise != nil {
		result.Exercise = Exercise{
			ID:            model.Exercise.ID,
			Name:          model.Exercise.Name,
			Description:   model.Exercise.Description,
			TargetMuscles: model.Exercise.TargetMuscles,
			Equipment:     model.Exercise.Equipment,
			Difficulty:    model.Exercise.Difficulty,
			VideoURL:      model.Exercise.VideoURL,
			ImageURL:      model.Exercise.ImageURL,
		}
	}
	return result
}

func validatePrescription(sets []SetInput) error {
	if len(sets) > training_prescription.MaxSetsPerExercise {
		return fmt.Errorf("%w: an exercise is capped at %d sets", ErrInvalidInput, training_prescription.MaxSetsPerExercise)
	}
	for i := range sets {
		if err := training_prescription.ValidateSet(&sets[i]); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidInput, err)
		}
	}
	return nil
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

func validateWeekID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: week id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid week id", ErrInvalidInput)
	}
	return nil
}

func validateWorkoutID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: workout id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid workout id", ErrInvalidInput)
	}
	return nil
}

func validateWorkoutExerciseID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: workout exercise id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid workout exercise id", ErrInvalidInput)
	}
	return nil
}

func validateExerciseID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: exercise id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid exercise id", ErrInvalidInput)
	}
	return nil
}
