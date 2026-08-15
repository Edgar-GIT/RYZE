package client_programs_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/client_programs"
)

const (
	trainerID         = "11111111-1111-1111-1111-111111111111"
	userID            = "22222222-2222-2222-2222-222222222222"
	programID         = "44444444-4444-4444-4444-444444444444"
	weekID            = "55555555-5555-5555-5555-555555555555"
	workoutID         = "66666666-6666-6666-6666-666666666666"
	workoutExerciseID = "77777777-7777-7777-7777-777777777777"
	exerciseID        = "88888888-8888-8888-8888-888888888888"
)

var errRepoFailure = errors.New("repository failure")

// stubRepo is an in-memory fake of the client program data-access surface. It
// records the user id forwarded to the repository so tests can prove the
// service forwards the authentication-context identity and never invents one.
type stubRepo struct {
	program *models.Program
	err     error
	gotUser string
}

func (s *stubRepo) FindAssignedProgram(_ context.Context, userID string) (*models.Program, error) {
	s.gotUser = userID
	if s.err != nil {
		return nil, s.err
	}
	return s.program, nil
}

func newService(repo *stubRepo) client_programs.Service {
	return client_programs.NewService(repo)
}

func validModel() *models.Program {
	return &models.Program{
		ID:          programID,
		TrainerID:   trainerID,
		Name:        "Strength Builder",
		Description: "Progressive strength program",
		Type:        models.ProgramTypePremium,
		Status:      models.ProgramStatusDraft,
		CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		Weeks: []models.ProgramWeek{
			{
				ID:         weekID,
				ProgramID:  programID,
				WeekNumber: 1,
				CreatedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				Workouts: []models.ProgramWorkout{
					{
						ID:            workoutID,
						ProgramWeekID: weekID,
						Position:      1,
						CreatedAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
						Exercises: []models.WorkoutExercise{
							{
								ID:               workoutExerciseID,
								ProgramWorkoutID: workoutID,
								ExerciseID:       exerciseID,
								Position:         1,
								CreatedAt:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
								UpdatedAt:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
								Exercise: &models.Exercise{
									ID:            exerciseID,
									Name:          "Bench Press",
									Description:   "A chest press",
									TargetMuscles: "Chest",
									Equipment:     "Barbell",
									Difficulty:    "Intermediate",
									VideoURL:      "https://example.com/video",
									ImageURL:      "https://example.com/image",
									CreatedAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
									UpdatedAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestGetProgramSuccess(t *testing.T) {
	repo := &stubRepo{program: validModel()}
	svc := newService(repo)

	program, err := svc.GetProgram(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetProgram: %v", err)
	}
	if repo.gotUser != userID {
		t.Fatalf("expected scope %q, got %q", userID, repo.gotUser)
	}
	if program.ID != programID || program.Name != "Strength Builder" {
		t.Fatalf("unexpected program %+v", program)
	}
	if len(program.Weeks) != 1 || program.Weeks[0].ID != weekID || program.Weeks[0].WeekNumber != 1 {
		t.Fatalf("unexpected weeks %+v", program.Weeks)
	}
	if len(program.Weeks[0].Workouts) != 1 || program.Weeks[0].Workouts[0].ID != workoutID || program.Weeks[0].Workouts[0].Position != 1 {
		t.Fatalf("unexpected workouts %+v", program.Weeks[0].Workouts)
	}
	if len(program.Weeks[0].Workouts[0].Exercises) != 1 || program.Weeks[0].Workouts[0].Exercises[0].ID != workoutExerciseID {
		t.Fatalf("unexpected workout exercises %+v", program.Weeks[0].Workouts[0].Exercises)
	}
	exercise := program.Weeks[0].Workouts[0].Exercises[0].Exercise
	if exercise.ID != exerciseID || exercise.Name != "Bench Press" {
		t.Fatalf("unexpected exercise %+v", exercise)
	}
}

func TestGetProgramEmptyStructure(t *testing.T) {
	repo := &stubRepo{program: &models.Program{
		ID:        programID,
		Name:      "Empty Program",
		Type:      models.ProgramTypeFree,
		Status:    models.ProgramStatusPublished,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}}
	svc := newService(repo)

	program, err := svc.GetProgram(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetProgram: %v", err)
	}
	if program.Weeks == nil || len(program.Weeks) != 0 {
		t.Fatalf("expected an empty non-nil week list, got %#v", program.Weeks)
	}
}

func TestGetProgramInvalidInput(t *testing.T) {
	svc := newService(&stubRepo{})

	cases := map[string]string{
		"empty user": "",
		"bad user":   "not-a-uuid",
	}
	for name, id := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.GetProgram(context.Background(), id); !errors.Is(err, client_programs.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestGetProgramCollapsesNotFound(t *testing.T) {
	for name, err := range map[string]error{
		"no assignment":   repositories.ErrAssignmentNotFound,
		"deleted program": repositories.ErrProgramNotFound,
	} {
		t.Run(name, func(t *testing.T) {
			svc := newService(&stubRepo{err: err})
			if _, err := svc.GetProgram(context.Background(), userID); !errors.Is(err, client_programs.ErrProgramNotFound) {
				t.Fatalf("expected ErrProgramNotFound, got %v", err)
			}
		})
	}
}

func TestGetProgramRepoFailureNotExposed(t *testing.T) {
	svc := newService(&stubRepo{err: errRepoFailure})

	_, err := svc.GetProgram(context.Background(), userID)
	if err == nil || errors.Is(err, client_programs.ErrInvalidInput) || errors.Is(err, client_programs.ErrProgramNotFound) {
		t.Fatalf("expected an internal failure to be hidden, got %v", err)
	}
	if !errors.Is(err, errRepoFailure) {
		t.Fatalf("expected the repository failure to stay wrapped for logs, got %v", err)
	}
}
