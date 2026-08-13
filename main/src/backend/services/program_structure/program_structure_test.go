package program_structure_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/program_structure"
)

const (
	trainerID = "11111111-1111-1111-1111-111111111111"
	programID = "22222222-2222-2222-2222-222222222222"
	weekID    = "33333333-3333-3333-3333-333333333333"
	workoutID = "44444444-4444-4444-4444-444444444444"
	otherID   = "55555555-5555-5555-5555-555555555555"
)

var errRepoFailure = errors.New("repository failure")

// stubWeekRepo is an in-memory fake of the week data-access surface. It
// records every identifier forwarded to the repository so tests can prove the
// service forwards the trainer context identity and never invents or accepts a
// client-supplied one.
type stubWeekRepo struct {
	create       func(trainerID, programID string, week *models.ProgramWeek) error
	list         func(trainerID, programID string) ([]models.ProgramWeek, error)
	find         func(trainerID, programID, weekID string) (*models.ProgramWeek, error)
	reorder      func(trainerID, programID string, orderedIDs []string) error
	softDelete   func(trainerID, programID, weekID string) error
	gotTrainerID string
	gotProgramID string
	gotWeekID    string
	gotOrder     []string
	createdWeek  *models.ProgramWeek
}

func (s *stubWeekRepo) Create(_ context.Context, trainerID, programID string, week *models.ProgramWeek) error {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	if s.create != nil {
		return s.create(trainerID, programID, week)
	}
	week.ID = weekID
	week.ProgramID = programID
	week.WeekNumber = 1
	week.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	week.UpdatedAt = week.CreatedAt
	s.createdWeek = week
	return nil
}

func (s *stubWeekRepo) ListByProgram(_ context.Context, trainerID, programID string) ([]models.ProgramWeek, error) {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	if s.list != nil {
		return s.list(trainerID, programID)
	}
	return nil, nil
}

func (s *stubWeekRepo) FindByIDAndProgram(_ context.Context, trainerID, programID, weekID string) (*models.ProgramWeek, error) {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	if s.find != nil {
		return s.find(trainerID, programID, weekID)
	}
	return nil, repositories.ErrWeekNotFound
}

func (s *stubWeekRepo) Reorder(_ context.Context, trainerID, programID string, orderedIDs []string) error {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotOrder = orderedIDs
	if s.reorder != nil {
		return s.reorder(trainerID, programID, orderedIDs)
	}
	return nil
}

func (s *stubWeekRepo) SoftDelete(_ context.Context, trainerID, programID, weekID string) error {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	if s.softDelete != nil {
		return s.softDelete(trainerID, programID, weekID)
	}
	return nil
}

// stubWorkoutRepo is an in-memory fake of the workout data-access surface with
// the same identity-forwarding guarantees as stubWeekRepo.
type stubWorkoutRepo struct {
	create       func(trainerID, programID, weekID string, workout *models.ProgramWorkout) error
	list         func(trainerID, programID, weekID string) ([]models.ProgramWorkout, error)
	find         func(trainerID, programID, weekID, workoutID string) (*models.ProgramWorkout, error)
	reorder      func(trainerID, programID, weekID string, orderedIDs []string) error
	softDelete   func(trainerID, programID, weekID, workoutID string) error
	gotTrainerID string
	gotProgramID string
	gotWeekID    string
	gotWorkoutID string
	gotOrder     []string
}

func (s *stubWorkoutRepo) Create(_ context.Context, trainerID, programID, weekID string, workout *models.ProgramWorkout) error {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	if s.create != nil {
		return s.create(trainerID, programID, weekID, workout)
	}
	workout.ID = workoutID
	workout.ProgramWeekID = weekID
	workout.Position = 1
	workout.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	workout.UpdatedAt = workout.CreatedAt
	return nil
}

func (s *stubWorkoutRepo) ListByWeek(_ context.Context, trainerID, programID, weekID string) ([]models.ProgramWorkout, error) {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	if s.list != nil {
		return s.list(trainerID, programID, weekID)
	}
	return nil, nil
}

func (s *stubWorkoutRepo) FindByIDAndWeek(_ context.Context, trainerID, programID, weekID, workoutID string) (*models.ProgramWorkout, error) {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	s.gotWorkoutID = workoutID
	if s.find != nil {
		return s.find(trainerID, programID, weekID, workoutID)
	}
	return nil, repositories.ErrWorkoutNotFound
}

func (s *stubWorkoutRepo) Reorder(_ context.Context, trainerID, programID, weekID string, orderedIDs []string) error {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	s.gotOrder = orderedIDs
	if s.reorder != nil {
		return s.reorder(trainerID, programID, weekID, orderedIDs)
	}
	return nil
}

func (s *stubWorkoutRepo) SoftDelete(_ context.Context, trainerID, programID, weekID, workoutID string) error {
	s.gotTrainerID = trainerID
	s.gotProgramID = programID
	s.gotWeekID = weekID
	s.gotWorkoutID = workoutID
	if s.softDelete != nil {
		return s.softDelete(trainerID, programID, weekID, workoutID)
	}
	return nil
}

func newService(weeks *stubWeekRepo, workouts *stubWorkoutRepo) program_structure.Service {
	return program_structure.NewService(weeks, workouts)
}

func validWeek() *models.ProgramWeek {
	return &models.ProgramWeek{
		ID:         weekID,
		ProgramID:  programID,
		WeekNumber: 1,
		Workouts:   []models.ProgramWorkout{*validWorkout()},
		CreatedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func validWorkout() *models.ProgramWorkout {
	return &models.ProgramWorkout{
		ID:            workoutID,
		ProgramWeekID: weekID,
		Position:      1,
		CreatedAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestCreateWeekSuccess(t *testing.T) {
	weeks := &stubWeekRepo{}
	svc := newService(weeks, &stubWorkoutRepo{})

	week, err := svc.CreateWeek(context.Background(), trainerID, programID)
	if err != nil {
		t.Fatalf("CreateWeek: %v", err)
	}
	if weeks.gotTrainerID != trainerID || weeks.gotProgramID != programID {
		t.Fatalf("expected scope %q/%q, got %q/%q", trainerID, programID, weeks.gotTrainerID, weeks.gotProgramID)
	}
	if week.ID == "" || week.WeekNumber != 1 || week.ProgramID != programID {
		t.Fatalf("unexpected week %+v", week)
	}
	if week.CreatedAt.IsZero() {
		t.Fatal("expected timestamps")
	}
}

func TestCreateWeekInvalidInput(t *testing.T) {
	svc := newService(&stubWeekRepo{}, &stubWorkoutRepo{})

	for name, ids := range map[string][2]string{
		"empty trainer": {"", programID},
		"bad trainer":   {"not-a-uuid", programID},
		"empty program": {trainerID, ""},
		"bad program":   {trainerID, "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.CreateWeek(context.Background(), ids[0], ids[1]); !errors.Is(err, program_structure.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestCreateWeekProgramNotFound(t *testing.T) {
	weeks := &stubWeekRepo{
		create: func(_, _ string, _ *models.ProgramWeek) error {
			return repositories.ErrProgramNotFound
		},
	}
	svc := newService(weeks, &stubWorkoutRepo{})

	if _, err := svc.CreateWeek(context.Background(), trainerID, programID); !errors.Is(err, program_structure.ErrProgramNotFound) {
		t.Fatalf("expected ErrProgramNotFound, got %v", err)
	}
}

func TestCreateWeekRepositoryFailure(t *testing.T) {
	weeks := &stubWeekRepo{
		create: func(_, _ string, _ *models.ProgramWeek) error {
			return errRepoFailure
		},
	}
	svc := newService(weeks, &stubWorkoutRepo{})

	_, err := svc.CreateWeek(context.Background(), trainerID, programID)
	if errors.Is(err, program_structure.ErrProgramNotFound) || errors.Is(err, program_structure.ErrInvalidInput) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestListWeeksSuccess(t *testing.T) {
	weeks := &stubWeekRepo{
		list: func(_, _ string) ([]models.ProgramWeek, error) {
			return []models.ProgramWeek{*validWeek()}, nil
		},
	}
	svc := newService(weeks, &stubWorkoutRepo{})

	result, err := svc.ListWeeks(context.Background(), trainerID, programID)
	if err != nil {
		t.Fatalf("ListWeeks: %v", err)
	}
	if weeks.gotTrainerID != trainerID || weeks.gotProgramID != programID {
		t.Fatalf("expected scope %q/%q, got %q/%q", trainerID, programID, weeks.gotTrainerID, weeks.gotProgramID)
	}
	if len(result) != 1 {
		t.Fatalf("expected one week, got %+v", result)
	}
	if result[0].ID != weekID || result[0].WeekNumber != 1 {
		t.Fatalf("unexpected week %+v", result[0])
	}
	if len(result[0].Workouts) != 1 || result[0].Workouts[0].ID != workoutID || result[0].Workouts[0].Position != 1 {
		t.Fatalf("unexpected nested workouts %+v", result[0].Workouts)
	}
}

func TestListWeeksInvalidInput(t *testing.T) {
	svc := newService(&stubWeekRepo{}, &stubWorkoutRepo{})

	for name, ids := range map[string][2]string{
		"empty trainer": {"", programID},
		"bad program":   {trainerID, "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.ListWeeks(context.Background(), ids[0], ids[1]); !errors.Is(err, program_structure.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestListWeeksRepositoryFailure(t *testing.T) {
	weeks := &stubWeekRepo{
		list: func(_, _ string) ([]models.ProgramWeek, error) {
			return nil, errRepoFailure
		},
	}
	svc := newService(weeks, &stubWorkoutRepo{})

	_, err := svc.ListWeeks(context.Background(), trainerID, programID)
	if errors.Is(err, program_structure.ErrInvalidInput) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestGetWeekSuccess(t *testing.T) {
	weeks := &stubWeekRepo{
		find: func(_, _, _ string) (*models.ProgramWeek, error) {
			return validWeek(), nil
		},
	}
	svc := newService(weeks, &stubWorkoutRepo{})

	week, err := svc.GetWeek(context.Background(), trainerID, programID, weekID)
	if err != nil {
		t.Fatalf("GetWeek: %v", err)
	}
	if weeks.gotTrainerID != trainerID || weeks.gotProgramID != programID || weeks.gotWeekID != weekID {
		t.Fatalf("expected scope %q/%q/%q, got %q/%q/%q", trainerID, programID, weekID, weeks.gotTrainerID, weeks.gotProgramID, weeks.gotWeekID)
	}
	if week.ID != weekID {
		t.Fatalf("unexpected week %+v", week)
	}
}

func TestGetWeekNotFound(t *testing.T) {
	svc := newService(&stubWeekRepo{}, &stubWorkoutRepo{})

	if _, err := svc.GetWeek(context.Background(), trainerID, programID, weekID); !errors.Is(err, program_structure.ErrWeekNotFound) {
		t.Fatalf("expected ErrWeekNotFound, got %v", err)
	}
}

func TestGetWeekInvalidInput(t *testing.T) {
	svc := newService(&stubWeekRepo{}, &stubWorkoutRepo{})

	for name, ids := range map[string][3]string{
		"empty trainer": {"", programID, weekID},
		"empty program": {trainerID, "", weekID},
		"empty week":    {trainerID, programID, ""},
		"bad week":      {trainerID, programID, "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.GetWeek(context.Background(), ids[0], ids[1], ids[2]); !errors.Is(err, program_structure.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestReorderWeeksSuccess(t *testing.T) {
	weeks := &stubWeekRepo{}
	svc := newService(weeks, &stubWorkoutRepo{})

	order := []string{weekID, otherID}
	if err := svc.ReorderWeeks(context.Background(), trainerID, programID, order); err != nil {
		t.Fatalf("ReorderWeeks: %v", err)
	}
	if weeks.gotTrainerID != trainerID || weeks.gotProgramID != programID {
		t.Fatalf("expected scope %q/%q, got %q/%q", trainerID, programID, weeks.gotTrainerID, weeks.gotProgramID)
	}
	if len(weeks.gotOrder) != 2 || weeks.gotOrder[0] != weekID || weeks.gotOrder[1] != otherID {
		t.Fatalf("expected order %v, got %v", order, weeks.gotOrder)
	}
}

func TestReorderWeeksInvalidOrder(t *testing.T) {
	svc := newService(&stubWeekRepo{}, &stubWorkoutRepo{})

	for name, order := range map[string][]string{
		"empty":          {},
		"empty id":       {""},
		"bad id":         {"not-a-uuid"},
		"duplicate":      {weekID, weekID},
		"duplicate pair": {weekID, otherID, weekID},
	} {
		t.Run(name, func(t *testing.T) {
			if err := svc.ReorderWeeks(context.Background(), trainerID, programID, order); !errors.Is(err, program_structure.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestReorderWeeksConflict(t *testing.T) {
	weeks := &stubWeekRepo{
		reorder: func(_, _ string, _ []string) error {
			return repositories.ErrWeekReorderConflict
		},
	}
	svc := newService(weeks, &stubWorkoutRepo{})

	if err := svc.ReorderWeeks(context.Background(), trainerID, programID, []string{weekID, otherID}); !errors.Is(err, program_structure.ErrReorderConflict) {
		t.Fatalf("expected ErrReorderConflict, got %v", err)
	}
}

func TestReorderWeeksProgramNotFound(t *testing.T) {
	weeks := &stubWeekRepo{
		reorder: func(_, _ string, _ []string) error {
			return repositories.ErrProgramNotFound
		},
	}
	svc := newService(weeks, &stubWorkoutRepo{})

	if err := svc.ReorderWeeks(context.Background(), trainerID, programID, []string{weekID, otherID}); !errors.Is(err, program_structure.ErrProgramNotFound) {
		t.Fatalf("expected ErrProgramNotFound, got %v", err)
	}
}

func TestDeleteWeekSuccess(t *testing.T) {
	weeks := &stubWeekRepo{}
	svc := newService(weeks, &stubWorkoutRepo{})

	if err := svc.DeleteWeek(context.Background(), trainerID, programID, weekID); err != nil {
		t.Fatalf("DeleteWeek: %v", err)
	}
	if weeks.gotTrainerID != trainerID || weeks.gotProgramID != programID || weeks.gotWeekID != weekID {
		t.Fatalf("expected scope %q/%q/%q, got %q/%q/%q", trainerID, programID, weekID, weeks.gotTrainerID, weeks.gotProgramID, weeks.gotWeekID)
	}
}

func TestDeleteWeekNotFound(t *testing.T) {
	weeks := &stubWeekRepo{
		softDelete: func(_, _, _ string) error {
			return repositories.ErrWeekNotFound
		},
	}
	svc := newService(weeks, &stubWorkoutRepo{})

	if err := svc.DeleteWeek(context.Background(), trainerID, programID, weekID); !errors.Is(err, program_structure.ErrWeekNotFound) {
		t.Fatalf("expected ErrWeekNotFound, got %v", err)
	}
}

func TestCreateWorkoutSuccess(t *testing.T) {
	workouts := &stubWorkoutRepo{}
	svc := newService(&stubWeekRepo{}, workouts)

	workout, err := svc.CreateWorkout(context.Background(), trainerID, programID, weekID)
	if err != nil {
		t.Fatalf("CreateWorkout: %v", err)
	}
	if workouts.gotTrainerID != trainerID || workouts.gotProgramID != programID || workouts.gotWeekID != weekID {
		t.Fatalf("expected scope %q/%q/%q, got %q/%q/%q", trainerID, programID, weekID, workouts.gotTrainerID, workouts.gotProgramID, workouts.gotWeekID)
	}
	if workout.ID == "" || workout.Position != 1 || workout.ProgramWeekID != weekID {
		t.Fatalf("unexpected workout %+v", workout)
	}
}

func TestCreateWorkoutWeekNotFound(t *testing.T) {
	workouts := &stubWorkoutRepo{
		create: func(_, _, _ string, _ *models.ProgramWorkout) error {
			return repositories.ErrWeekNotFound
		},
	}
	svc := newService(&stubWeekRepo{}, workouts)

	if _, err := svc.CreateWorkout(context.Background(), trainerID, programID, weekID); !errors.Is(err, program_structure.ErrWeekNotFound) {
		t.Fatalf("expected ErrWeekNotFound, got %v", err)
	}
}

func TestCreateWorkoutInvalidInput(t *testing.T) {
	svc := newService(&stubWeekRepo{}, &stubWorkoutRepo{})

	for name, ids := range map[string][3]string{
		"empty trainer": {"", programID, weekID},
		"empty program": {trainerID, "", weekID},
		"empty week":    {trainerID, programID, ""},
		"bad week":      {trainerID, programID, "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.CreateWorkout(context.Background(), ids[0], ids[1], ids[2]); !errors.Is(err, program_structure.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestListWorkoutsSuccess(t *testing.T) {
	workouts := &stubWorkoutRepo{
		list: func(_, _, _ string) ([]models.ProgramWorkout, error) {
			return []models.ProgramWorkout{*validWorkout()}, nil
		},
	}
	svc := newService(&stubWeekRepo{}, workouts)

	result, err := svc.ListWorkouts(context.Background(), trainerID, programID, weekID)
	if err != nil {
		t.Fatalf("ListWorkouts: %v", err)
	}
	if workouts.gotTrainerID != trainerID || workouts.gotProgramID != programID || workouts.gotWeekID != weekID {
		t.Fatalf("expected scope %q/%q/%q, got %q/%q/%q", trainerID, programID, weekID, workouts.gotTrainerID, workouts.gotProgramID, workouts.gotWeekID)
	}
	if len(result) != 1 || result[0].ID != workoutID || result[0].Position != 1 {
		t.Fatalf("unexpected workouts %+v", result)
	}
}

func TestGetWorkoutSuccess(t *testing.T) {
	workouts := &stubWorkoutRepo{
		find: func(_, _, _, _ string) (*models.ProgramWorkout, error) {
			return validWorkout(), nil
		},
	}
	svc := newService(&stubWeekRepo{}, workouts)

	workout, err := svc.GetWorkout(context.Background(), trainerID, programID, weekID, workoutID)
	if err != nil {
		t.Fatalf("GetWorkout: %v", err)
	}
	if workouts.gotWorkoutID != workoutID {
		t.Fatalf("expected workout id %q, got %q", workoutID, workouts.gotWorkoutID)
	}
	if workout.ID != workoutID {
		t.Fatalf("unexpected workout %+v", workout)
	}
}

func TestGetWorkoutNotFound(t *testing.T) {
	svc := newService(&stubWeekRepo{}, &stubWorkoutRepo{})

	if _, err := svc.GetWorkout(context.Background(), trainerID, programID, weekID, workoutID); !errors.Is(err, program_structure.ErrWorkoutNotFound) {
		t.Fatalf("expected ErrWorkoutNotFound, got %v", err)
	}
}

func TestGetWorkoutInvalidInput(t *testing.T) {
	svc := newService(&stubWeekRepo{}, &stubWorkoutRepo{})

	for name, ids := range map[string][4]string{
		"empty trainer": {"", programID, weekID, workoutID},
		"empty program": {trainerID, "", weekID, workoutID},
		"empty week":    {trainerID, programID, "", workoutID},
		"empty workout": {trainerID, programID, weekID, ""},
		"bad workout":   {trainerID, programID, weekID, "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.GetWorkout(context.Background(), ids[0], ids[1], ids[2], ids[3]); !errors.Is(err, program_structure.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestReorderWorkoutsSuccess(t *testing.T) {
	workouts := &stubWorkoutRepo{}
	svc := newService(&stubWeekRepo{}, workouts)

	order := []string{workoutID, otherID}
	if err := svc.ReorderWorkouts(context.Background(), trainerID, programID, weekID, order); err != nil {
		t.Fatalf("ReorderWorkouts: %v", err)
	}
	if workouts.gotTrainerID != trainerID || workouts.gotProgramID != programID || workouts.gotWeekID != weekID {
		t.Fatalf("expected scope %q/%q/%q, got %q/%q/%q", trainerID, programID, weekID, workouts.gotTrainerID, workouts.gotProgramID, workouts.gotWeekID)
	}
	if len(workouts.gotOrder) != 2 || workouts.gotOrder[0] != workoutID || workouts.gotOrder[1] != otherID {
		t.Fatalf("expected order %v, got %v", order, workouts.gotOrder)
	}
}

func TestReorderWorkoutsInvalidOrder(t *testing.T) {
	svc := newService(&stubWeekRepo{}, &stubWorkoutRepo{})

	for name, order := range map[string][]string{
		"empty":     {},
		"bad id":    {"not-a-uuid"},
		"duplicate": {workoutID, workoutID},
	} {
		t.Run(name, func(t *testing.T) {
			if err := svc.ReorderWorkouts(context.Background(), trainerID, programID, weekID, order); !errors.Is(err, program_structure.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestReorderWorkoutsConflict(t *testing.T) {
	workouts := &stubWorkoutRepo{
		reorder: func(_, _, _ string, _ []string) error {
			return repositories.ErrWorkoutReorderConflict
		},
	}
	svc := newService(&stubWeekRepo{}, workouts)

	if err := svc.ReorderWorkouts(context.Background(), trainerID, programID, weekID, []string{workoutID, otherID}); !errors.Is(err, program_structure.ErrReorderConflict) {
		t.Fatalf("expected ErrReorderConflict, got %v", err)
	}
}

func TestReorderWorkoutsWeekNotFound(t *testing.T) {
	workouts := &stubWorkoutRepo{
		reorder: func(_, _, _ string, _ []string) error {
			return repositories.ErrWeekNotFound
		},
	}
	svc := newService(&stubWeekRepo{}, workouts)

	if err := svc.ReorderWorkouts(context.Background(), trainerID, programID, weekID, []string{workoutID, otherID}); !errors.Is(err, program_structure.ErrWeekNotFound) {
		t.Fatalf("expected ErrWeekNotFound, got %v", err)
	}
}

func TestDeleteWorkoutSuccess(t *testing.T) {
	workouts := &stubWorkoutRepo{}
	svc := newService(&stubWeekRepo{}, workouts)

	if err := svc.DeleteWorkout(context.Background(), trainerID, programID, weekID, workoutID); err != nil {
		t.Fatalf("DeleteWorkout: %v", err)
	}
	if workouts.gotWorkoutID != workoutID {
		t.Fatalf("expected workout id %q, got %q", workoutID, workouts.gotWorkoutID)
	}
}

func TestDeleteWorkoutNotFound(t *testing.T) {
	workouts := &stubWorkoutRepo{
		softDelete: func(_, _, _, _ string) error {
			return repositories.ErrWorkoutNotFound
		},
	}
	svc := newService(&stubWeekRepo{}, workouts)

	if err := svc.DeleteWorkout(context.Background(), trainerID, programID, weekID, workoutID); !errors.Is(err, program_structure.ErrWorkoutNotFound) {
		t.Fatalf("expected ErrWorkoutNotFound, got %v", err)
	}
}

func TestProgramStructureNeverExposesSecrets(t *testing.T) {
	weeks := &stubWeekRepo{
		find: func(_, _, _ string) (*models.ProgramWeek, error) {
			return validWeek(), nil
		},
	}
	workouts := &stubWorkoutRepo{
		find: func(_, _, _, _ string) (*models.ProgramWorkout, error) {
			return validWorkout(), nil
		},
	}
	svc := newService(weeks, workouts)

	week, err := svc.GetWeek(context.Background(), trainerID, programID, weekID)
	if err != nil {
		t.Fatalf("GetWeek: %v", err)
	}
	workout, err := svc.GetWorkout(context.Background(), trainerID, programID, weekID, workoutID)
	if err != nil {
		t.Fatalf("GetWorkout: %v", err)
	}
	if week.ID == "" || week.ProgramID == "" || workout.ID == "" {
		t.Fatal("safe struct fields must be present")
	}

	for _, value := range []reflect.Type{reflect.TypeOf(*week), reflect.TypeOf(*workout)} {
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
