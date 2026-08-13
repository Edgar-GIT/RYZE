package workout_exercises_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/workout_exercises"
)

const (
	trainerID         = "11111111-1111-1111-1111-111111111111"
	programID         = "22222222-2222-2222-2222-222222222222"
	weekID            = "33333333-3333-3333-3333-333333333333"
	workoutID         = "44444444-4444-4444-4444-444444444444"
	workoutExerciseID = "66666666-6666-6666-6666-666666666666"
	otherID           = "55555555-5555-5555-5555-555555555555"
	exerciseID        = "77777777-7777-7777-7777-777777777777"
	deletedExerciseID = "88888888-8888-8888-8888-888888888888"
)

var errRepoFailure = errors.New("repository failure")

// stubRepo is an in-memory fake of the workout exercise data-access surface.
// It records every identifier forwarded to the repository so tests can prove
// the service forwards the trainer context identity and never invents or
// accepts a client-supplied one.
type stubRepo struct {
	add          func(trainerID, programID, weekID, workoutID string, workoutExercise *models.WorkoutExercise) error
	list         func(trainerID, programID, weekID, workoutID string) ([]models.WorkoutExercise, error)
	find         func(trainerID, programID, weekID, workoutID, workoutExerciseID string) (*models.WorkoutExercise, error)
	reorder      func(trainerID, programID, weekID, workoutID string, orderedIDs []string) error
	softDelete   func(trainerID, programID, weekID, workoutID, workoutExerciseID string) error
	gotTrainerID string
	gotProgramID string
	gotWeekID    string
	gotWorkoutID string
	gotEntryID   string
	gotExercise  string
	gotOrder     []string
	createdEntry *models.WorkoutExercise
}

func (s *stubRepo) AddExercise(_ context.Context, trainerID, programID, weekID, workoutID string, workoutExercise *models.WorkoutExercise) error {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	s.gotWorkoutID = workoutID
	s.gotExercise = workoutExercise.ExerciseID
	if s.add != nil {
		return s.add(trainerID, programID, weekID, workoutID, workoutExercise)
	}
	workoutExercise.ID = workoutExerciseID
	workoutExercise.ProgramWorkoutID = workoutID
	workoutExercise.Position = 1
	workoutExercise.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	workoutExercise.UpdatedAt = workoutExercise.CreatedAt
	s.createdEntry = workoutExercise
	return nil
}

func (s *stubRepo) ListByWorkout(_ context.Context, trainerID, programID, weekID, workoutID string) ([]models.WorkoutExercise, error) {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	s.gotWorkoutID = workoutID
	if s.list != nil {
		return s.list(trainerID, programID, weekID, workoutID)
	}
	return nil, nil
}

func (s *stubRepo) FindByIDAndWorkout(_ context.Context, trainerID, programID, weekID, workoutID, workoutExerciseID string) (*models.WorkoutExercise, error) {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	s.gotWorkoutID = workoutID
	s.gotEntryID = workoutExerciseID
	if s.find != nil {
		return s.find(trainerID, programID, weekID, workoutID, workoutExerciseID)
	}
	return nil, repositories.ErrWorkoutExerciseNotFound
}

func (s *stubRepo) Reorder(_ context.Context, trainerID, programID, weekID, workoutID string, orderedIDs []string) error {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	s.gotWorkoutID = workoutID
	s.gotOrder = orderedIDs
	if s.reorder != nil {
		return s.reorder(trainerID, programID, weekID, workoutID, orderedIDs)
	}
	return nil
}

func (s *stubRepo) SoftDelete(_ context.Context, trainerID, programID, weekID, workoutID, workoutExerciseID string) error {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	s.gotWorkoutID = workoutID
	s.gotEntryID = workoutExerciseID
	if s.softDelete != nil {
		return s.softDelete(trainerID, programID, weekID, workoutID, workoutExerciseID)
	}
	return nil
}

func newService(repo *stubRepo) workout_exercises.Service {
	return workout_exercises.NewService(repo)
}

func validModel() *models.WorkoutExercise {
	return &models.WorkoutExercise{
		ID:               workoutExerciseID,
		ProgramWorkoutID: workoutID,
		Position:         1,
		Exercise: &models.Exercise{
			ID:            exerciseID,
			Name:          "Barbell Squat",
			Description:   "Legs",
			TargetMuscles: "Quads, Glutes",
			Equipment:     "Barbell",
			Difficulty:    "Intermediate",
			VideoURL:      "https://media.ryze.local/videos/squat.mp4",
			ImageURL:      "https://media.ryze.local/images/squat.jpg",
		},
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestAddExerciseSuccess(t *testing.T) {
	repo := &stubRepo{}
	svc := newService(repo)

	entry, err := svc.AddExercise(context.Background(), trainerID, programID, weekID, workoutID, exerciseID)
	if err != nil {
		t.Fatalf("AddExercise: %v", err)
	}
	if repo.gotTrainerID != trainerID || repo.gotProgramID != programID || repo.gotWeekID != weekID || repo.gotWorkoutID != workoutID {
		t.Fatalf("expected scope %q/%q/%q/%q, got %q/%q/%q/%q", trainerID, programID, weekID, workoutID, repo.gotTrainerID, repo.gotProgramID, repo.gotWeekID, repo.gotWorkoutID)
	}
	if repo.gotExercise != exerciseID {
		t.Fatalf("expected exercise %q, got %q", exerciseID, repo.gotExercise)
	}
	if entry.ID == "" || entry.Position != 1 || entry.ProgramWorkoutID != workoutID {
		t.Fatalf("unexpected entry %+v", entry)
	}
}

func TestAddExerciseInvalidInput(t *testing.T) {
	svc := newService(&stubRepo{})

	for name, ids := range map[string][5]string{
		"empty trainer":  {"", programID, weekID, workoutID, exerciseID},
		"bad trainer":    {"not-a-uuid", programID, weekID, workoutID, exerciseID},
		"empty program":  {trainerID, "", weekID, workoutID, exerciseID},
		"bad program":    {trainerID, "not-a-uuid", weekID, workoutID, exerciseID},
		"empty week":     {trainerID, programID, "", workoutID, exerciseID},
		"bad week":       {trainerID, programID, "not-a-uuid", workoutID, exerciseID},
		"empty workout":  {trainerID, programID, weekID, "", exerciseID},
		"bad workout":    {trainerID, programID, weekID, "not-a-uuid", exerciseID},
		"empty exercise": {trainerID, programID, weekID, workoutID, ""},
		"bad exercise":   {trainerID, programID, weekID, workoutID, "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.AddExercise(context.Background(), ids[0], ids[1], ids[2], ids[3], ids[4]); !errors.Is(err, workout_exercises.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestAddExerciseWorkoutNotFound(t *testing.T) {
	repo := &stubRepo{
		add: func(_, _, _, _ string, _ *models.WorkoutExercise) error {
			return repositories.ErrWorkoutNotFound
		},
	}
	svc := newService(repo)

	if _, err := svc.AddExercise(context.Background(), trainerID, programID, weekID, workoutID, exerciseID); !errors.Is(err, workout_exercises.ErrWorkoutNotFound) {
		t.Fatalf("expected ErrWorkoutNotFound, got %v", err)
	}
}

func TestAddExerciseExerciseNotFound(t *testing.T) {
	repo := &stubRepo{
		add: func(_, _, _, _ string, _ *models.WorkoutExercise) error {
			return repositories.ErrExerciseNotFound
		},
	}
	svc := newService(repo)

	if _, err := svc.AddExercise(context.Background(), trainerID, programID, weekID, workoutID, deletedExerciseID); !errors.Is(err, workout_exercises.ErrExerciseNotFound) {
		t.Fatalf("expected ErrExerciseNotFound, got %v", err)
	}
}

func TestAddExerciseRepositoryFailure(t *testing.T) {
	repo := &stubRepo{
		add: func(_, _, _, _ string, _ *models.WorkoutExercise) error {
			return errRepoFailure
		},
	}
	svc := newService(repo)

	_, err := svc.AddExercise(context.Background(), trainerID, programID, weekID, workoutID, exerciseID)
	if errors.Is(err, workout_exercises.ErrWorkoutNotFound) || errors.Is(err, workout_exercises.ErrExerciseNotFound) || errors.Is(err, workout_exercises.ErrInvalidInput) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestListExercisesSuccess(t *testing.T) {
	repo := &stubRepo{
		list: func(_, _, _, _ string) ([]models.WorkoutExercise, error) {
			return []models.WorkoutExercise{*validModel()}, nil
		},
	}
	svc := newService(repo)

	result, err := svc.ListExercises(context.Background(), trainerID, programID, weekID, workoutID)
	if err != nil {
		t.Fatalf("ListExercises: %v", err)
	}
	if repo.gotTrainerID != trainerID || repo.gotProgramID != programID || repo.gotWeekID != weekID || repo.gotWorkoutID != workoutID {
		t.Fatalf("expected scope %q/%q/%q/%q, got %q/%q/%q/%q", trainerID, programID, weekID, workoutID, repo.gotTrainerID, repo.gotProgramID, repo.gotWeekID, repo.gotWorkoutID)
	}
	if len(result) != 1 {
		t.Fatalf("expected one entry, got %+v", result)
	}
	if result[0].ID != workoutExerciseID || result[0].Position != 1 || result[0].ProgramWorkoutID != workoutID {
		t.Fatalf("unexpected entry %+v", result[0])
	}
	if result[0].Exercise.ID != exerciseID || result[0].Exercise.Name != "Barbell Squat" {
		t.Fatalf("unexpected embedded exercise %+v", result[0].Exercise)
	}
}

func TestListExercisesInvalidInput(t *testing.T) {
	svc := newService(&stubRepo{})

	for name, ids := range map[string][4]string{
		"empty trainer": {"", programID, weekID, workoutID},
		"bad trainer":   {"not-a-uuid", programID, weekID, workoutID},
		"empty program": {trainerID, "", weekID, workoutID},
		"bad program":   {trainerID, "not-a-uuid", weekID, workoutID},
		"empty week":    {trainerID, programID, "", workoutID},
		"empty workout": {trainerID, programID, weekID, ""},
		"bad workout":   {trainerID, programID, weekID, "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.ListExercises(context.Background(), ids[0], ids[1], ids[2], ids[3]); !errors.Is(err, workout_exercises.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestListExercisesRepositoryFailure(t *testing.T) {
	repo := &stubRepo{
		list: func(_, _, _, _ string) ([]models.WorkoutExercise, error) {
			return nil, errRepoFailure
		},
	}
	svc := newService(repo)

	_, err := svc.ListExercises(context.Background(), trainerID, programID, weekID, workoutID)
	if errors.Is(err, workout_exercises.ErrInvalidInput) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestGetExerciseSuccess(t *testing.T) {
	repo := &stubRepo{
		find: func(_, _, _, _, _ string) (*models.WorkoutExercise, error) {
			return validModel(), nil
		},
	}
	svc := newService(repo)

	entry, err := svc.GetExercise(context.Background(), trainerID, programID, weekID, workoutID, workoutExerciseID)
	if err != nil {
		t.Fatalf("GetExercise: %v", err)
	}
	if repo.gotTrainerID != trainerID || repo.gotProgramID != programID || repo.gotWeekID != weekID || repo.gotWorkoutID != workoutID || repo.gotEntryID != workoutExerciseID {
		t.Fatalf("expected scope %q/%q/%q/%q/%q, got %q/%q/%q/%q/%q", trainerID, programID, weekID, workoutID, workoutExerciseID, repo.gotTrainerID, repo.gotProgramID, repo.gotWeekID, repo.gotWorkoutID, repo.gotEntryID)
	}
	if entry.ID != workoutExerciseID {
		t.Fatalf("unexpected entry %+v", entry)
	}
}

func TestGetExerciseNotFound(t *testing.T) {
	svc := newService(&stubRepo{})

	if _, err := svc.GetExercise(context.Background(), trainerID, programID, weekID, workoutID, workoutExerciseID); !errors.Is(err, workout_exercises.ErrWorkoutExerciseNotFound) {
		t.Fatalf("expected ErrWorkoutExerciseNotFound, got %v", err)
	}
}

func TestGetExerciseInvalidInput(t *testing.T) {
	svc := newService(&stubRepo{})

	for name, ids := range map[string][5]string{
		"empty trainer": {"", programID, weekID, workoutID, workoutExerciseID},
		"empty program": {trainerID, "", weekID, workoutID, workoutExerciseID},
		"empty week":    {trainerID, programID, "", workoutID, workoutExerciseID},
		"empty workout": {trainerID, programID, weekID, "", workoutExerciseID},
		"empty entry":   {trainerID, programID, weekID, workoutID, ""},
		"bad entry":     {trainerID, programID, weekID, workoutID, "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.GetExercise(context.Background(), ids[0], ids[1], ids[2], ids[3], ids[4]); !errors.Is(err, workout_exercises.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestReorderExercisesSuccess(t *testing.T) {
	repo := &stubRepo{}
	svc := newService(repo)

	order := []string{workoutExerciseID, otherID}
	if err := svc.ReorderExercises(context.Background(), trainerID, programID, weekID, workoutID, order); err != nil {
		t.Fatalf("ReorderExercises: %v", err)
	}
	if repo.gotTrainerID != trainerID || repo.gotProgramID != programID || repo.gotWeekID != weekID || repo.gotWorkoutID != workoutID {
		t.Fatalf("expected scope %q/%q/%q/%q, got %q/%q/%q/%q", trainerID, programID, weekID, workoutID, repo.gotTrainerID, repo.gotProgramID, repo.gotWeekID, repo.gotWorkoutID)
	}
	if len(repo.gotOrder) != 2 || repo.gotOrder[0] != workoutExerciseID || repo.gotOrder[1] != otherID {
		t.Fatalf("expected order %v, got %v", order, repo.gotOrder)
	}
}

func TestReorderExercisesInvalidOrder(t *testing.T) {
	svc := newService(&stubRepo{})

	for name, order := range map[string][]string{
		"empty":          {},
		"empty id":       {""},
		"bad id":         {"not-a-uuid"},
		"duplicate":      {workoutExerciseID, workoutExerciseID},
		"duplicate pair": {workoutExerciseID, otherID, workoutExerciseID},
	} {
		t.Run(name, func(t *testing.T) {
			if err := svc.ReorderExercises(context.Background(), trainerID, programID, weekID, workoutID, order); !errors.Is(err, workout_exercises.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestReorderExercisesConflict(t *testing.T) {
	repo := &stubRepo{
		reorder: func(_, _, _, _ string, _ []string) error {
			return repositories.ErrWorkoutExerciseReorderConflict
		},
	}
	svc := newService(repo)

	if err := svc.ReorderExercises(context.Background(), trainerID, programID, weekID, workoutID, []string{workoutExerciseID, otherID}); !errors.Is(err, workout_exercises.ErrReorderConflict) {
		t.Fatalf("expected ErrReorderConflict, got %v", err)
	}
}

func TestReorderExercisesWorkoutNotFound(t *testing.T) {
	repo := &stubRepo{
		reorder: func(_, _, _, _ string, _ []string) error {
			return repositories.ErrWorkoutNotFound
		},
	}
	svc := newService(repo)

	if err := svc.ReorderExercises(context.Background(), trainerID, programID, weekID, workoutID, []string{workoutExerciseID, otherID}); !errors.Is(err, workout_exercises.ErrWorkoutNotFound) {
		t.Fatalf("expected ErrWorkoutNotFound, got %v", err)
	}
}

func TestRemoveExerciseSuccess(t *testing.T) {
	repo := &stubRepo{}
	svc := newService(repo)

	if err := svc.RemoveExercise(context.Background(), trainerID, programID, weekID, workoutID, workoutExerciseID); err != nil {
		t.Fatalf("RemoveExercise: %v", err)
	}
	if repo.gotTrainerID != trainerID || repo.gotProgramID != programID || repo.gotWeekID != weekID || repo.gotWorkoutID != workoutID || repo.gotEntryID != workoutExerciseID {
		t.Fatalf("expected scope %q/%q/%q/%q/%q, got %q/%q/%q/%q/%q", trainerID, programID, weekID, workoutID, workoutExerciseID, repo.gotTrainerID, repo.gotProgramID, repo.gotWeekID, repo.gotWorkoutID, repo.gotEntryID)
	}
}

func TestRemoveExerciseNotFound(t *testing.T) {
	repo := &stubRepo{
		softDelete: func(_, _, _, _, _ string) error {
			return repositories.ErrWorkoutExerciseNotFound
		},
	}
	svc := newService(repo)

	if err := svc.RemoveExercise(context.Background(), trainerID, programID, weekID, workoutID, workoutExerciseID); !errors.Is(err, workout_exercises.ErrWorkoutExerciseNotFound) {
		t.Fatalf("expected ErrWorkoutExerciseNotFound, got %v", err)
	}
}

func TestWorkoutExercisesNeverExposeSensitiveData(t *testing.T) {
	repo := &stubRepo{
		find: func(_, _, _, _, _ string) (*models.WorkoutExercise, error) {
			return validModel(), nil
		},
	}
	svc := newService(repo)

	entry, err := svc.GetExercise(context.Background(), trainerID, programID, weekID, workoutID, workoutExerciseID)
	if err != nil {
		t.Fatalf("GetExercise: %v", err)
	}
	if entry.ID == "" || entry.ProgramWorkoutID == "" || entry.Exercise.ID == "" || entry.Exercise.Name == "" {
		t.Fatal("safe struct fields must be present")
	}

	for _, value := range []reflect.Type{reflect.TypeOf(*entry), reflect.TypeOf(entry.Exercise)} {
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i).Name
			for _, sensitive := range []string{"password", "token", "secret", "session", "deleted"} {
				if strings.Contains(strings.ToLower(field), sensitive) {
					t.Fatalf("%s must not expose %q", value.Name(), field)
				}
			}
		}
	}
}
