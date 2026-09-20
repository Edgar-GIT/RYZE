package generic_programs_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"ryze/backend/config"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/generic_programs"
)

const (
	programID   = "22222222-2222-2222-2222-222222222222"
	exerciseID1 = "44444444-4444-4444-4444-444444444444"
	exerciseID2 = "55555555-5555-5555-5555-555555555555"
)

var errRepoFailure = errors.New("repository failure")

// stubGenericRepo is an in-memory fake of the generic program data-access
// surface. It behaves like the real repository (create fills identifiers and
// timestamps, find/list respect soft-deletes) and records every identifier
// passed to each operation so tests can prove the service never forwards,
// invents or accepts a trainer id.
type stubGenericRepo struct {
	program          *models.Program
	deleted          bool
	create           func(program *models.Program) error
	find             func(programID string) (*models.Program, error)
	search           func(filter repositories.GenericProgramFilter, page, limit int) ([]models.Program, int64, error)
	update           func(programID string, program *models.Program) error
	softDelete       func(programID string) error
	publish          func(programID string) error
	createTrainerID  string
	updateProgramID  string
	deleteProgramID  string
	publishProgramID string
}

func (s *stubGenericRepo) CreateFull(_ context.Context, program *models.Program) error {
	s.createTrainerID = program.TrainerID
	if s.create != nil {
		return s.create(program)
	}
	if program.ID == "" {
		program.ID = programID
	}
	if program.CreatedAt.IsZero() {
		program.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if program.UpdatedAt.IsZero() {
		program.UpdatedAt = program.CreatedAt
	}
	fillNestedIDs(program)
	s.program = program
	return nil
}

func (s *stubGenericRepo) UpdateFull(_ context.Context, programID string, program *models.Program) error {
	s.updateProgramID = programID
	if s.update != nil {
		return s.update(programID, program)
	}
	if s.program == nil || s.deleted {
		return repositories.ErrGenericProgramNotFound
	}
	fillNestedIDs(program)
	program.ID = programID
	program.CreatedAt = s.program.CreatedAt
	s.program = program
	return nil
}

func (s *stubGenericRepo) FindByID(_ context.Context, reqID string) (*models.Program, error) {
	if s.find != nil {
		return s.find(reqID)
	}
	if s.program == nil || s.deleted || s.program.ID != reqID {
		return nil, repositories.ErrGenericProgramNotFound
	}
	return s.program, nil
}

func (s *stubGenericRepo) Search(_ context.Context, filter repositories.GenericProgramFilter, page, limit int) ([]models.Program, int64, error) {
	if s.search != nil {
		return s.search(filter, page, limit)
	}
	if s.program == nil || s.deleted {
		return nil, 0, nil
	}
	return []models.Program{*s.program}, 1, nil
}

func (s *stubGenericRepo) SoftDelete(_ context.Context, programID string) error {
	s.deleteProgramID = programID
	if s.softDelete != nil {
		return s.softDelete(programID)
	}
	if s.program == nil || s.deleted || s.program.ID != programID {
		return repositories.ErrGenericProgramNotFound
	}
	s.deleted = true
	return nil
}

func (s *stubGenericRepo) Publish(_ context.Context, programID string) error {
	s.publishProgramID = programID
	if s.publish != nil {
		return s.publish(programID)
	}
	if s.program == nil || s.deleted || s.program.ID != programID || s.program.Status != models.ProgramStatusDraft {
		return repositories.ErrGenericProgramNotFound
	}
	s.program.Status = models.ProgramStatusPublished
	return nil
}

func fillNestedIDs(program *models.Program) {
	for i := range program.Weeks {
		week := &program.Weeks[i]
		if week.ID == "" {
			week.ID = fmt.Sprintf("week-%d", i+1)
		}
		for k := range week.Workouts {
			workout := &week.Workouts[k]
			if workout.ID == "" {
				workout.ID = fmt.Sprintf("workout-%d-%d", i+1, k+1)
			}
			for j := range workout.Exercises {
				exercise := &workout.Exercises[j]
				if exercise.ID == "" {
					exercise.ID = fmt.Sprintf("wexercise-%d-%d-%d", i+1, k+1, j+1)
				}
				for l := range exercise.Sets {
					if exercise.Sets[l].ID == "" {
						exercise.Sets[l].ID = fmt.Sprintf("set-%d", l+1)
					}
				}
			}
		}
	}
}

// validProgramInput returns a structurally valid generic program input with one
// week, one workout, one exercise and one set.
func validProgramInput() generic_programs.ProgramInput {
	reps := 10
	weight := 100.0
	rir := 1
	rpe := 8.0
	rest := 90
	return generic_programs.ProgramInput{
		Name:             "  Hypertrophy 101  ",
		Description:      "A 12 week hypertrophy program.",
		Type:             models.ProgramTypePremium,
		Level:            "Intermediate",
		DurationWeeks:    12,
		FrequencyPerWeek: 4,
		PriceMinorUnits:  1449,
		Currency:         "EUR",
		Weeks: []generic_programs.WeekInput{
			{
				WeekNumber: 1,
				Workouts: []generic_programs.WorkoutInput{
					{
						Position: 1,
						Exercises: []generic_programs.ExerciseInput{
							{
								ExerciseID:   exerciseID1,
								Instructions: "Keep your back straight.",
								Notes:        "Focus on control.",
								Prescription: []generic_programs.SetInput{
									{
										SetType:     "working",
										Reps:        &reps,
										WeightKg:    &weight,
										RIR:         &rir,
										RPE:         &rpe,
										RestSeconds: &rest,
										Tempo:       "2010",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// validProgramModel returns an already-created generic program model matching
// the shape the repository would persist.
func validProgramModel() *models.Program {
	reps := 10
	weight := 100.0
	rir := 1
	rpe := 8.0
	rest := 90
	level := "Intermediate"
	duration := 12
	frequency := 4
	return &models.Program{
		ID:               programID,
		Name:             "Hypertrophy 101",
		Description:      "A 12 week hypertrophy program.",
		Type:             models.ProgramTypePremium,
		Status:           models.ProgramStatusDraft,
		Level:            &level,
		DurationWeeks:    &duration,
		FrequencyPerWeek: &frequency,
		PriceMinorUnits:  1449,
		Currency:         "EUR",
		CreatedAt:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Weeks: []models.ProgramWeek{
			{
				ID:         "week-1",
				ProgramID:  programID,
				WeekNumber: 1,
				Workouts: []models.ProgramWorkout{
					{
						ID:            "workout-1-1",
						ProgramWeekID: "week-1",
						Position:      1,
						Exercises: []models.WorkoutExercise{
							{
								ID:               "wexercise-1-1-1",
								ProgramWorkoutID: "workout-1-1",
								Position:         1,
								ExerciseID:       exerciseID1,
								Instructions:     "Keep your back straight.",
								Notes:            "Focus on control.",
								Sets: []models.WorkoutExerciseSet{
									{
										ID:                "set-1",
										WorkoutExerciseID: "wexercise-1-1-1",
										SetNumber:         1,
										SetType:           "working",
										Reps:              &reps,
										WeightKg:          &weight,
										RIR:               &rir,
										RPE:               &rpe,
										RestSeconds:       &rest,
										Tempo:             "2010",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func newService(repo *stubGenericRepo) generic_programs.Service {
	return generic_programs.NewService(repo, config.PricingConfig{MinProgramPriceMinorUnits: 100})
}

func TestCreateProgramSuccess(t *testing.T) {
	repo := &stubGenericRepo{}
	svc := newService(repo)

	program, err := svc.CreateProgram(context.Background(), validProgramInput())
	if err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}
	if repo.createTrainerID != "" {
		t.Fatalf("generic programs must never carry a trainer id, got %q", repo.createTrainerID)
	}
	if program.ID != programID {
		t.Fatalf("expected program id %q, got %q", programID, program.ID)
	}
	if program.Name != "Hypertrophy 101" {
		t.Fatalf("expected trimmed name, got %q", program.Name)
	}
	if program.Level == nil || *program.Level != "Intermediate" {
		t.Fatalf("expected level Intermediate, got %v", program.Level)
	}
	if program.DurationWeeks == nil || *program.DurationWeeks != 12 {
		t.Fatalf("expected duration 12, got %v", program.DurationWeeks)
	}
	if program.FrequencyPerWeek == nil || *program.FrequencyPerWeek != 4 {
		t.Fatalf("expected frequency 4, got %v", program.FrequencyPerWeek)
	}
	if len(program.Weeks) != 1 || len(program.Weeks[0].Workouts) != 1 ||
		len(program.Weeks[0].Workouts[0].Exercises) != 1 ||
		len(program.Weeks[0].Workouts[0].Exercises[0].Sets) != 1 {
		t.Fatalf("expected full nested structure, got %+v", program.Weeks)
	}
	if program.Weeks[0].WeekNumber != 1 || program.Weeks[0].Workouts[0].Position != 1 ||
		program.Weeks[0].Workouts[0].Exercises[0].Position != 1 ||
		program.Weeks[0].Workouts[0].Exercises[0].Sets[0].SetNumber != 1 {
		t.Fatalf("expected structural keys starting at 1, got %+v", program.Weeks)
	}
	if program.CreatedAt.IsZero() {
		t.Fatal("expected timestamps")
	}
}

func TestCreateProgramDefaults(t *testing.T) {
	repo := &stubGenericRepo{}
	svc := newService(repo)

	input := validProgramInput()
	input.Status = ""
	input.Currency = ""
	input.Level = ""
	input.DurationWeeks = 0
	input.FrequencyPerWeek = 0

	program, err := svc.CreateProgram(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}
	if program.Status != models.ProgramStatusDraft {
		t.Fatalf("expected default status draft, got %q", program.Status)
	}
	if program.Currency != "EUR" {
		t.Fatalf("expected default currency EUR, got %q", program.Currency)
	}
	if program.Level != nil {
		t.Fatalf("expected nil level when unset, got %v", program.Level)
	}
	if program.DurationWeeks != nil || program.FrequencyPerWeek != nil {
		t.Fatalf("expected nil duration/frequency when unset, got %v/%v", program.DurationWeeks, program.FrequencyPerWeek)
	}
}

func TestCreateProgramRejectsInvalidInput(t *testing.T) {
	svc := newService(&stubGenericRepo{})

	cases := []struct {
		name   string
		mutate func(i *generic_programs.ProgramInput)
	}{
		{name: "empty name", mutate: func(i *generic_programs.ProgramInput) { i.Name = "   " }},
		{name: "name too long", mutate: func(i *generic_programs.ProgramInput) { i.Name = strings.Repeat("a", generic_programs.MaxNameLength+1) }},
		{name: "description too long", mutate: func(i *generic_programs.ProgramInput) {
			i.Description = strings.Repeat("a", generic_programs.MaxDescriptionLength+1)
		}},
		{name: "invalid type", mutate: func(i *generic_programs.ProgramInput) { i.Type = "random" }},
		{name: "invalid status", mutate: func(i *generic_programs.ProgramInput) { i.Status = "archived" }},
		{name: "invalid currency", mutate: func(i *generic_programs.ProgramInput) { i.Currency = "USD" }},
		{name: "free with price", mutate: func(i *generic_programs.ProgramInput) { i.Type = models.ProgramTypeFree; i.PriceMinorUnits = 100 }},
		{name: "paid below minimum", mutate: func(i *generic_programs.ProgramInput) { i.PriceMinorUnits = 50 }},
		{name: "invalid level", mutate: func(i *generic_programs.ProgramInput) { i.Level = "Expert" }},
		{name: "negative duration", mutate: func(i *generic_programs.ProgramInput) { i.DurationWeeks = -1 }},
		{name: "duration too long", mutate: func(i *generic_programs.ProgramInput) { i.DurationWeeks = generic_programs.MaxWeeksPerProgram + 1 }},
		{name: "frequency too high", mutate: func(i *generic_programs.ProgramInput) { i.FrequencyPerWeek = generic_programs.MaxWorkoutsPerWeek + 1 }},
		{name: "no weeks", mutate: func(i *generic_programs.ProgramInput) { i.Weeks = nil }},
		{name: "week number not contiguous", mutate: func(i *generic_programs.ProgramInput) { i.Weeks[0].WeekNumber = 2 }},
		{name: "workout position not contiguous", mutate: func(i *generic_programs.ProgramInput) { i.Weeks[0].Workouts[0].Position = 0 }},
		{name: "too many weeks", mutate: func(i *generic_programs.ProgramInput) {
			for w := 1; w <= generic_programs.MaxWeeksPerProgram+1; w++ {
				i.Weeks = append(i.Weeks, i.Weeks[0])
			}
		}},
		{name: "empty exercise id", mutate: func(i *generic_programs.ProgramInput) { i.Weeks[0].Workouts[0].Exercises[0].ExerciseID = "" }},
		{name: "invalid exercise id", mutate: func(i *generic_programs.ProgramInput) { i.Weeks[0].Workouts[0].Exercises[0].ExerciseID = "not-a-uuid" }},
		{name: "invalid set type", mutate: func(i *generic_programs.ProgramInput) {
			i.Weeks[0].Workouts[0].Exercises[0].Prescription[0].SetType = "max"
		}},
		{name: "zero reps", mutate: func(i *generic_programs.ProgramInput) {
			r := 0
			i.Weeks[0].Workouts[0].Exercises[0].Prescription[0].Reps = &r
		}},
		{name: "negative weight", mutate: func(i *generic_programs.ProgramInput) {
			w := -1.0
			i.Weeks[0].Workouts[0].Exercises[0].Prescription[0].WeightKg = &w
		}},
		{name: "rpe out of range", mutate: func(i *generic_programs.ProgramInput) {
			rpe := 11.0
			i.Weeks[0].Workouts[0].Exercises[0].Prescription[0].RPE = &rpe
		}},
		{name: "tempo too long", mutate: func(i *generic_programs.ProgramInput) {
			i.Weeks[0].Workouts[0].Exercises[0].Prescription[0].Tempo = strings.Repeat("1", generic_programs.MaxTempoLength+1)
		}},
		{name: "too many workouts in a week", mutate: func(i *generic_programs.ProgramInput) {
			for w := 1; w <= generic_programs.MaxWorkoutsPerWeek+1; w++ {
				i.Weeks[0].Workouts = append(i.Weeks[0].Workouts, i.Weeks[0].Workouts[0])
			}
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := validProgramInput()
			tc.mutate(&input)
			if _, err := svc.CreateProgram(context.Background(), input); !errors.Is(err, generic_programs.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestCreateProgramRejectsDuplicateExercise(t *testing.T) {
	svc := newService(&stubGenericRepo{})

	input := validProgramInput()
	input.Weeks[0].Workouts[0].Exercises = append(input.Weeks[0].Workouts[0].Exercises, input.Weeks[0].Workouts[0].Exercises[0])

	if _, err := svc.CreateProgram(context.Background(), input); !errors.Is(err, generic_programs.ErrDuplicateExercise) {
		t.Fatalf("expected ErrDuplicateExercise, got %v", err)
	}
}

func TestCreateProgramAllowsExerciseAcrossDifferentWorkouts(t *testing.T) {
	repo := &stubGenericRepo{}
	svc := newService(repo)

	input := validProgramInput()
	input.Weeks[0].Workouts = append(input.Weeks[0].Workouts, generic_programs.WorkoutInput{
		Position: 2,
		Exercises: []generic_programs.ExerciseInput{
			{ExerciseID: exerciseID1},
		},
	})

	program, err := svc.CreateProgram(context.Background(), input)
	if err != nil {
		t.Fatalf("same exercise in different workouts must be allowed, got %v", err)
	}
	if len(program.Weeks[0].Workouts) != 2 {
		t.Fatalf("expected two workouts, got %d", len(program.Weeks[0].Workouts))
	}
}

func TestCreateProgramExerciseNotFound(t *testing.T) {
	repo := &stubGenericRepo{
		create: func(_ *models.Program) error {
			return repositories.ErrExerciseNotFound
		},
	}
	svc := newService(repo)

	if _, err := svc.CreateProgram(context.Background(), validProgramInput()); !errors.Is(err, generic_programs.ErrExerciseNotFound) {
		t.Fatalf("expected ErrExerciseNotFound, got %v", err)
	}
}

func TestCreateProgramRepositoryFailure(t *testing.T) {
	repo := &stubGenericRepo{
		create: func(_ *models.Program) error {
			return errRepoFailure
		},
	}
	svc := newService(repo)

	_, err := svc.CreateProgram(context.Background(), validProgramInput())
	if errors.Is(err, generic_programs.ErrInvalidInput) || errors.Is(err, generic_programs.ErrExerciseNotFound) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestListProgramsSuccess(t *testing.T) {
	var gotFilter repositories.GenericProgramFilter
	repo := &stubGenericRepo{
		search: func(filter repositories.GenericProgramFilter, page, limit int) ([]models.Program, int64, error) {
			gotFilter = filter
			if page != 2 || limit != 10 {
				t.Fatalf("expected page 2 limit 10, got %d/%d", page, limit)
			}
			return []models.Program{*validProgramModel()}, 1, nil
		},
	}
	svc := newService(repo)

	result, err := svc.ListPrograms(context.Background(), generic_programs.ProgramFilter{
		Query:            " hypertrophy ",
		Type:             models.ProgramTypePremium,
		Level:            "Intermediate",
		DurationWeeks:    12,
		FrequencyPerWeek: 4,
	}, 2, 10)
	if err != nil {
		t.Fatalf("ListPrograms: %v", err)
	}
	if gotFilter.Query != "hypertrophy" {
		t.Fatalf("expected trimmed query, got %q", gotFilter.Query)
	}
	if gotFilter.DurationWeeks != 12 || gotFilter.FrequencyPerWeek != 4 {
		t.Fatalf("expected filters forwarded, got %+v", gotFilter)
	}
	if result.Total != 1 || len(result.Programs) != 1 || result.Page != 2 || result.Limit != 10 {
		t.Fatalf("unexpected result %+v", result)
	}
	if result.Programs[0].Name != "Hypertrophy 101" {
		t.Fatalf("unexpected program %+v", result.Programs[0])
	}
}

func TestListProgramsClampsLimit(t *testing.T) {
	var gotLimit int
	repo := &stubGenericRepo{
		search: func(_ repositories.GenericProgramFilter, _, limit int) ([]models.Program, int64, error) {
			gotLimit = limit
			return nil, 0, nil
		},
	}
	svc := newService(repo)

	if _, err := svc.ListPrograms(context.Background(), generic_programs.ProgramFilter{}, 1, 99999); err != nil {
		t.Fatalf("ListPrograms: %v", err)
	}
	if gotLimit != generic_programs.MaxPageSize {
		t.Fatalf("expected limit clamped to %d, got %d", generic_programs.MaxPageSize, gotLimit)
	}
}

func TestListProgramsRejectsInvalidInput(t *testing.T) {
	svc := newService(&stubGenericRepo{})

	for name, args := range map[string][2]int{
		"page zero":      {0, 10},
		"page negative":  {-1, 10},
		"limit zero":     {1, 0},
		"limit negative": {1, -5},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.ListPrograms(context.Background(), generic_programs.ProgramFilter{}, args[0], args[1]); !errors.Is(err, generic_programs.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}

	if _, err := svc.ListPrograms(context.Background(), generic_programs.ProgramFilter{Query: strings.Repeat("a", generic_programs.MaxSearchLength+1)}, 1, 10); !errors.Is(err, generic_programs.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for long search, got %v", err)
	}

	if _, err := svc.ListPrograms(context.Background(), generic_programs.ProgramFilter{Level: "Expert"}, 1, 10); !errors.Is(err, generic_programs.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for invalid level filter, got %v", err)
	}

	if _, err := svc.ListPrograms(context.Background(), generic_programs.ProgramFilter{Type: "random"}, 1, 10); !errors.Is(err, generic_programs.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for invalid type filter, got %v", err)
	}

	if _, err := svc.ListPrograms(context.Background(), generic_programs.ProgramFilter{DurationWeeks: -1}, 1, 10); !errors.Is(err, generic_programs.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for negative duration filter, got %v", err)
	}
}

func TestListProgramsRepositoryFailure(t *testing.T) {
	repo := &stubGenericRepo{
		search: func(_ repositories.GenericProgramFilter, _, _ int) ([]models.Program, int64, error) {
			return nil, 0, errRepoFailure
		},
	}
	svc := newService(repo)

	_, err := svc.ListPrograms(context.Background(), generic_programs.ProgramFilter{}, 1, 10)
	if errors.Is(err, generic_programs.ErrInvalidInput) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestGetProgramSuccess(t *testing.T) {
	repo := &stubGenericRepo{program: validProgramModel()}
	svc := newService(repo)

	program, err := svc.GetProgram(context.Background(), programID)
	if err != nil {
		t.Fatalf("GetProgram: %v", err)
	}
	if program.ID != programID || program.Type != models.ProgramTypePremium {
		t.Fatalf("unexpected program %+v", program)
	}
	if len(program.Weeks) != 1 || len(program.Weeks[0].Workouts[0].Exercises[0].Sets) != 1 {
		t.Fatalf("expected full structure, got %+v", program.Weeks)
	}
	if program.Weeks[0].Workouts[0].Exercises[0].Exercise.ID != exerciseID1 {
		t.Fatalf("expected exercise metadata, got %+v", program.Weeks[0].Workouts[0].Exercises[0])
	}
}

func TestGetProgramNotFound(t *testing.T) {
	repo := &stubGenericRepo{
		find: func(_ string) (*models.Program, error) {
			return nil, repositories.ErrGenericProgramNotFound
		},
	}
	svc := newService(repo)

	if _, err := svc.GetProgram(context.Background(), programID); !errors.Is(err, generic_programs.ErrProgramNotFound) {
		t.Fatalf("expected ErrProgramNotFound, got %v", err)
	}
}

func TestGetProgramRejectsInvalidID(t *testing.T) {
	svc := newService(&stubGenericRepo{})

	for name, id := range map[string]string{
		"empty":      "",
		"not a uuid": "not-a-uuid",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.GetProgram(context.Background(), id); !errors.Is(err, generic_programs.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestGetProgramRepositoryFailure(t *testing.T) {
	repo := &stubGenericRepo{
		find: func(_ string) (*models.Program, error) {
			return nil, errRepoFailure
		},
	}
	svc := newService(repo)

	_, err := svc.GetProgram(context.Background(), programID)
	if errors.Is(err, generic_programs.ErrProgramNotFound) || errors.Is(err, generic_programs.ErrInvalidInput) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestUpdateProgramSuccess(t *testing.T) {
	repo := &stubGenericRepo{program: validProgramModel()}
	svc := newService(repo)

	input := validProgramInput()
	input.Name = "  Advanced Power  "
	input.Level = "Advanced"
	program, err := svc.UpdateProgram(context.Background(), programID, input)
	if err != nil {
		t.Fatalf("UpdateProgram: %v", err)
	}
	if repo.updateProgramID != programID {
		t.Fatalf("expected update on %q, got %q", programID, repo.updateProgramID)
	}
	if program.Name != "Advanced Power" {
		t.Fatalf("expected updated name, got %q", program.Name)
	}
	if program.Level == nil || *program.Level != "Advanced" {
		t.Fatalf("expected updated level, got %v", program.Level)
	}
}

func TestUpdateProgramNotFound(t *testing.T) {
	repo := &stubGenericRepo{}
	svc := newService(repo)

	// UpdateFull is a single atomic verify-and-mutate operation: the repository
	// locks and checks the program inside the same transaction that applies the
	// structure. A missing program surfaces as ErrProgramNotFound without any
	// mutation being persisted.
	if _, err := svc.UpdateProgram(context.Background(), programID, validProgramInput()); !errors.Is(err, generic_programs.ErrProgramNotFound) {
		t.Fatalf("expected ErrProgramNotFound, got %v", err)
	}
	if repo.program != nil {
		t.Fatal("no program may be persisted for a missing program")
	}
}

func TestUpdateProgramRejectsInvalidIDs(t *testing.T) {
	svc := newService(&stubGenericRepo{})

	for name, id := range map[string]string{
		"empty":      "",
		"not a uuid": "not-a-uuid",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.UpdateProgram(context.Background(), id, validProgramInput()); !errors.Is(err, generic_programs.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestUpdateProgramExerciseNotFound(t *testing.T) {
	repo := &stubGenericRepo{
		program: validProgramModel(),
		update: func(_ string, _ *models.Program) error {
			return repositories.ErrExerciseNotFound
		},
	}
	svc := newService(repo)

	if _, err := svc.UpdateProgram(context.Background(), programID, validProgramInput()); !errors.Is(err, generic_programs.ErrExerciseNotFound) {
		t.Fatalf("expected ErrExerciseNotFound, got %v", err)
	}
}

func TestUpdateProgramRepositoryFailure(t *testing.T) {
	repo := &stubGenericRepo{
		program: validProgramModel(),
		update: func(_ string, _ *models.Program) error {
			return errRepoFailure
		},
	}
	svc := newService(repo)

	_, err := svc.UpdateProgram(context.Background(), programID, validProgramInput())
	if errors.Is(err, generic_programs.ErrProgramNotFound) || errors.Is(err, generic_programs.ErrInvalidInput) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestPublishProgramSuccess(t *testing.T) {
	repo := &stubGenericRepo{program: validProgramModel()}
	svc := newService(repo)

	program, err := svc.PublishProgram(context.Background(), programID)
	if err != nil {
		t.Fatalf("PublishProgram: %v", err)
	}
	if program.Status != models.ProgramStatusPublished {
		t.Fatalf("expected published, got %q", program.Status)
	}
	if repo.publishProgramID != programID {
		t.Fatalf("expected publish on %q, got %q", programID, repo.publishProgramID)
	}
}

func TestPublishProgramAlreadyPublished(t *testing.T) {
	published := validProgramModel()
	published.Status = models.ProgramStatusPublished
	repo := &stubGenericRepo{
		program: published,
		publish: func(_ string) error {
			return repositories.ErrGenericProgramNotFound
		},
	}
	svc := newService(repo)

	if _, err := svc.PublishProgram(context.Background(), programID); !errors.Is(err, generic_programs.ErrProgramAlreadyPublished) {
		t.Fatalf("expected ErrProgramAlreadyPublished, got %v", err)
	}
}

func TestPublishProgramNotFound(t *testing.T) {
	repo := &stubGenericRepo{
		publish: func(_ string) error {
			return repositories.ErrGenericProgramNotFound
		},
		find: func(_ string) (*models.Program, error) {
			return nil, repositories.ErrGenericProgramNotFound
		},
	}
	svc := newService(repo)

	if _, err := svc.PublishProgram(context.Background(), programID); !errors.Is(err, generic_programs.ErrProgramNotFound) {
		t.Fatalf("expected ErrProgramNotFound, got %v", err)
	}
}

func TestPublishProgramRejectsInvalidIDs(t *testing.T) {
	svc := newService(&stubGenericRepo{})

	for name, id := range map[string]string{
		"empty":      "",
		"not a uuid": "not-a-uuid",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.PublishProgram(context.Background(), id); !errors.Is(err, generic_programs.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestPublishProgramRepositoryFailure(t *testing.T) {
	repo := &stubGenericRepo{
		publish: func(_ string) error {
			return errRepoFailure
		},
	}
	svc := newService(repo)

	_, err := svc.PublishProgram(context.Background(), programID)
	if errors.Is(err, generic_programs.ErrProgramNotFound) || errors.Is(err, generic_programs.ErrInvalidInput) || errors.Is(err, generic_programs.ErrProgramAlreadyPublished) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestDeleteProgramSuccess(t *testing.T) {
	repo := &stubGenericRepo{program: validProgramModel()}
	svc := newService(repo)

	if err := svc.DeleteProgram(context.Background(), programID); err != nil {
		t.Fatalf("DeleteProgram: %v", err)
	}
	if repo.deleteProgramID != programID {
		t.Fatalf("expected soft delete on %q, got %q", programID, repo.deleteProgramID)
	}
}

func TestDeleteProgramNotFound(t *testing.T) {
	repo := &stubGenericRepo{
		softDelete: func(_ string) error {
			return repositories.ErrGenericProgramNotFound
		},
	}
	svc := newService(repo)

	if err := svc.DeleteProgram(context.Background(), programID); !errors.Is(err, generic_programs.ErrProgramNotFound) {
		t.Fatalf("expected ErrProgramNotFound, got %v", err)
	}
}

func TestDeleteProgramRejectsInvalidIDs(t *testing.T) {
	svc := newService(&stubGenericRepo{})

	for name, id := range map[string]string{
		"empty":      "",
		"not a uuid": "not-a-uuid",
	} {
		t.Run(name, func(t *testing.T) {
			if err := svc.DeleteProgram(context.Background(), id); !errors.Is(err, generic_programs.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestDeleteProgramRepositoryFailure(t *testing.T) {
	repo := &stubGenericRepo{
		softDelete: func(_ string) error {
			return errRepoFailure
		},
	}
	svc := newService(repo)

	err := svc.DeleteProgram(context.Background(), programID)
	if errors.Is(err, generic_programs.ErrProgramNotFound) || errors.Is(err, generic_programs.ErrInvalidInput) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestNeverExposesSecrets(t *testing.T) {
	repo := &stubGenericRepo{program: validProgramModel()}
	svc := newService(repo)

	program, err := svc.GetProgram(context.Background(), programID)
	if err != nil {
		t.Fatalf("GetProgram: %v", err)
	}
	if program.Name == "" || program.Type == "" {
		t.Fatal("safe program fields must be present")
	}

	for _, shape := range []any{generic_programs.Program{}, generic_programs.ProgramDetail{}, program.Weeks[0].Workouts[0].Exercises[0]} {
		typ := reflect.TypeOf(shape)
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i).Name
			for _, sensitive := range []string{"password", "token", "secret", "session", "deleted", "trainer"} {
				if strings.Contains(strings.ToLower(field), sensitive) {
					t.Fatalf("%s must not expose %q", typ.Name(), field)
				}
			}
		}
	}
}
