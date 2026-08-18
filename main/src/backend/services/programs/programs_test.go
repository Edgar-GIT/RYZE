package programs_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/programs"
)

const (
	trainerID      = "11111111-1111-1111-1111-111111111111"
	programID      = "22222222-2222-2222-2222-222222222222"
	otherTrainerID = "33333333-3333-3333-3333-333333333333"
)

var errRepoFailure = errors.New("repository failure")

// stubProgramRepo is an in-memory fake of the program data-access surface. It
// behaves like the real repository (create fills identifiers and timestamps,
// find/list respect ownership and soft-deletes) and records the identifiers
// passed to every operation, so tests can prove the service forwards the
// trainer context identity and never invents or accepts a client-supplied one.
type stubProgramRepo struct {
	program             *models.Program
	deleted             bool
	create              func(program *models.Program) error
	find                func(trainerID, programID string) (*models.Program, error)
	list                func(trainerID string, page, limit int) ([]models.Program, int64, error)
	update              func(trainerID, programID string, updates map[string]any) error
	publish             func(trainerID, programID string) error
	softDelete          func(trainerID, programID string) error
	createGotTrainerID  string
	updateGotTrainerID  string
	updateGotProgramID  string
	updateGotUpdates    map[string]any
	publishGotTrainerID string
	publishGotProgramID string
	deleteGotTrainerID  string
	deleteGotProgramID  string
}

func (s *stubProgramRepo) Create(_ context.Context, program *models.Program) error {
	s.createGotTrainerID = program.TrainerID
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
	s.program = program
	return nil
}

func (s *stubProgramRepo) FindByIDAndTrainer(_ context.Context, trainerID, programID string) (*models.Program, error) {
	if s.find != nil {
		return s.find(trainerID, programID)
	}
	if s.program == nil || s.deleted {
		return nil, repositories.ErrProgramNotFound
	}
	if s.program.TrainerID != trainerID || s.program.ID != programID {
		return nil, repositories.ErrProgramNotFound
	}
	return s.program, nil
}

func (s *stubProgramRepo) ListByTrainer(_ context.Context, trainerID string, page, limit int) ([]models.Program, int64, error) {
	if s.list != nil {
		return s.list(trainerID, page, limit)
	}
	if s.program == nil || s.deleted || s.program.TrainerID != trainerID {
		return nil, 0, nil
	}
	return []models.Program{*s.program}, 1, nil
}

func (s *stubProgramRepo) Update(_ context.Context, trainerID, programID string, updates map[string]any) error {
	s.updateGotTrainerID = trainerID
	s.updateGotProgramID = programID
	s.updateGotUpdates = updates
	if s.update != nil {
		return s.update(trainerID, programID, updates)
	}
	if s.program == nil || s.deleted {
		return repositories.ErrProgramNotFound
	}
	for field, value := range updates {
		switch field {
		case "name":
			s.program.Name = value.(string)
		case "description":
			s.program.Description = value.(string)
		case "type":
			s.program.Type = value.(string)
		case "status":
			s.program.Status = value.(string)
		}
	}
	return nil
}

func (s *stubProgramRepo) Publish(_ context.Context, trainerID, programID string) error {
	s.publishGotTrainerID = trainerID
	s.publishGotProgramID = programID
	if s.publish != nil {
		return s.publish(trainerID, programID)
	}
	if s.program == nil || s.deleted {
		return repositories.ErrProgramNotFound
	}
	if s.program.TrainerID != trainerID || s.program.ID != programID {
		return repositories.ErrProgramNotFound
	}
	if s.program.Status != models.ProgramStatusDraft {
		return repositories.ErrProgramNotFound
	}
	s.program.Status = models.ProgramStatusPublished
	return nil
}

func (s *stubProgramRepo) SoftDelete(_ context.Context, trainerID, programID string) error {
	s.deleteGotTrainerID = trainerID
	s.deleteGotProgramID = programID
	if s.softDelete != nil {
		return s.softDelete(trainerID, programID)
	}
	if s.program == nil || s.deleted {
		return repositories.ErrProgramNotFound
	}
	s.deleted = true
	return nil
}

func validProgram() *models.Program {
	return &models.Program{
		ID:          programID,
		TrainerID:   trainerID,
		Name:        "Strength Builder",
		Description: "A 12 week strength program.",
		Type:        models.ProgramTypePremium,
		Status:      models.ProgramStatusDraft,
		CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
}

func newService(repo *stubProgramRepo) programs.Service {
	return programs.NewService(repo)
}

func TestCreateProgramSuccess(t *testing.T) {
	repo := &stubProgramRepo{}
	svc := newService(repo)

	program, err := svc.CreateProgram(context.Background(), trainerID, programs.CreateProgramInput{
		Name:        "  Strength Builder  ",
		Description: "A 12 week strength program.",
		Type:        models.ProgramTypePremium,
		Status:      models.ProgramStatusPublished,
	})
	if err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}
	if repo.createGotTrainerID != trainerID {
		t.Fatalf("expected trainer id %q, got %q", trainerID, repo.createGotTrainerID)
	}
	if program.Name != "Strength Builder" {
		t.Fatalf("expected trimmed name, got %q", program.Name)
	}
	if program.Type != models.ProgramTypePremium || program.Status != models.ProgramStatusPublished {
		t.Fatalf("unexpected program %+v", program)
	}
	if program.ID == "" || program.CreatedAt.IsZero() {
		t.Fatal("expected id and timestamps")
	}
}

func TestCreateProgramDefaultsStatusToDraft(t *testing.T) {
	repo := &stubProgramRepo{}
	svc := newService(repo)

	program, err := svc.CreateProgram(context.Background(), trainerID, programs.CreateProgramInput{
		Name: "Free Program",
		Type: models.ProgramTypeFree,
	})
	if err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}
	if program.Status != models.ProgramStatusDraft {
		t.Fatalf("expected default status draft, got %q", program.Status)
	}
}

func TestCreateProgramRejectsInvalidInput(t *testing.T) {
	svc := newService(&stubProgramRepo{})

	cases := []struct {
		name  string
		input programs.CreateProgramInput
	}{
		{name: "empty trainer", input: programs.CreateProgramInput{Name: "P", Type: models.ProgramTypeFree}},
		{name: "bad trainer", input: programs.CreateProgramInput{Name: "P", Type: models.ProgramTypeFree}},
		{name: "empty name", input: programs.CreateProgramInput{Name: "   ", Type: models.ProgramTypeFree}},
		{name: "name too long", input: programs.CreateProgramInput{Name: strings.Repeat("a", programs.MaxNameLength+1), Type: models.ProgramTypeFree}},
		{name: "description too long", input: programs.CreateProgramInput{Name: "P", Description: strings.Repeat("a", programs.MaxDescriptionLength+1), Type: models.ProgramTypeFree}},
		{name: "invalid type", input: programs.CreateProgramInput{Name: "P", Type: "random"}},
		{name: "invalid status", input: programs.CreateProgramInput{Name: "P", Type: models.ProgramTypeFree, Status: "archived"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			trainer := trainerID
			if tc.name == "empty trainer" || tc.name == "bad trainer" {
				trainer = ""
				if tc.name == "bad trainer" {
					trainer = "not-a-uuid"
				}
			}
			if _, err := svc.CreateProgram(context.Background(), trainer, tc.input); !errors.Is(err, programs.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestCreateProgramRepositoryFailure(t *testing.T) {
	repo := &stubProgramRepo{
		create: func(_ *models.Program) error {
			return errRepoFailure
		},
	}
	svc := newService(repo)

	_, err := svc.CreateProgram(context.Background(), trainerID, programs.CreateProgramInput{
		Name: "P",
		Type: models.ProgramTypeFree,
	})
	if errors.Is(err, programs.ErrInvalidInput) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestListProgramsSuccess(t *testing.T) {
	var gotTrainerID string
	repo := &stubProgramRepo{
		list: func(trainerID string, page, limit int) ([]models.Program, int64, error) {
			gotTrainerID = trainerID
			if page != 2 || limit != 10 {
				t.Fatalf("expected page 2 limit 10, got %d/%d", page, limit)
			}
			return []models.Program{*validProgram()}, 1, nil
		},
	}
	svc := newService(repo)

	result, err := svc.ListPrograms(context.Background(), trainerID, 2, 10)
	if err != nil {
		t.Fatalf("ListPrograms: %v", err)
	}
	if gotTrainerID != trainerID {
		t.Fatalf("expected trainer id %q, got %q", trainerID, gotTrainerID)
	}
	if result.Total != 1 || len(result.Programs) != 1 {
		t.Fatalf("expected one program, got %+v", result)
	}
	if result.Page != 2 || result.Limit != 10 {
		t.Fatalf("expected page 2 limit 10, got %d/%d", result.Page, result.Limit)
	}
	if result.Programs[0].Name != "Strength Builder" {
		t.Fatalf("unexpected program %+v", result.Programs[0])
	}
}

func TestListProgramsClampsLimit(t *testing.T) {
	var gotLimit int
	repo := &stubProgramRepo{
		list: func(_ string, _, limit int) ([]models.Program, int64, error) {
			gotLimit = limit
			return nil, 0, nil
		},
	}
	svc := newService(repo)

	if _, err := svc.ListPrograms(context.Background(), trainerID, 1, 99999); err != nil {
		t.Fatalf("ListPrograms: %v", err)
	}
	if gotLimit != programs.MaxPageSize {
		t.Fatalf("expected limit clamped to %d, got %d", programs.MaxPageSize, gotLimit)
	}
}

func TestListProgramsRejectsInvalidInput(t *testing.T) {
	svc := newService(&stubProgramRepo{})

	for name, args := range map[string][2]int{
		"page zero":      {0, 10},
		"page negative":  {-1, 10},
		"limit zero":     {1, 0},
		"limit negative": {1, -5},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.ListPrograms(context.Background(), trainerID, args[0], args[1]); !errors.Is(err, programs.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}

	if _, err := svc.ListPrograms(context.Background(), "", 1, 10); !errors.Is(err, programs.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for empty trainer, got %v", err)
	}
}

func TestListProgramsRepositoryFailure(t *testing.T) {
	repo := &stubProgramRepo{
		list: func(_ string, _, _ int) ([]models.Program, int64, error) {
			return nil, 0, errRepoFailure
		},
	}
	svc := newService(repo)

	_, err := svc.ListPrograms(context.Background(), trainerID, 1, 10)
	if errors.Is(err, programs.ErrInvalidInput) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestGetProgramSuccess(t *testing.T) {
	var gotTrainerID, gotProgramID string
	repo := &stubProgramRepo{
		find: func(trainerID, programID string) (*models.Program, error) {
			gotTrainerID = trainerID
			gotProgramID = programID
			return validProgram(), nil
		},
	}
	svc := newService(repo)

	program, err := svc.GetProgram(context.Background(), trainerID, programID)
	if err != nil {
		t.Fatalf("GetProgram: %v", err)
	}
	if gotTrainerID != trainerID || gotProgramID != programID {
		t.Fatalf("expected identifiers %q/%q, got %q/%q", trainerID, programID, gotTrainerID, gotProgramID)
	}
	if program.ID != programID || program.Type != models.ProgramTypePremium {
		t.Fatalf("unexpected program %+v", program)
	}
}

func TestGetProgramRejectsInvalidIDs(t *testing.T) {
	svc := newService(&stubProgramRepo{})

	for name, ids := range map[string][2]string{
		"empty trainer": {"", programID},
		"bad trainer":   {"not-a-uuid", programID},
		"empty program": {trainerID, ""},
		"bad program":   {trainerID, "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.GetProgram(context.Background(), ids[0], ids[1]); !errors.Is(err, programs.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestGetProgramNotFound(t *testing.T) {
	repo := &stubProgramRepo{
		find: func(_, _ string) (*models.Program, error) {
			return nil, repositories.ErrProgramNotFound
		},
	}
	svc := newService(repo)

	if _, err := svc.GetProgram(context.Background(), trainerID, programID); !errors.Is(err, programs.ErrProgramNotFound) {
		t.Fatalf("expected ErrProgramNotFound, got %v", err)
	}
}

func TestGetProgramRepositoryFailure(t *testing.T) {
	repo := &stubProgramRepo{
		find: func(_, _ string) (*models.Program, error) {
			return nil, errRepoFailure
		},
	}
	svc := newService(repo)

	_, err := svc.GetProgram(context.Background(), trainerID, programID)
	if errors.Is(err, programs.ErrProgramNotFound) || errors.Is(err, programs.ErrInvalidInput) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestUpdateProgramSuccess(t *testing.T) {
	repo := &stubProgramRepo{program: validProgram()}
	svc := newService(repo)

	name := "Hypertrophy 101"
	status := models.ProgramStatusPublished
	program, err := svc.UpdateProgram(context.Background(), trainerID, programID, programs.UpdateProgramInput{
		Name:   &name,
		Status: &status,
	})
	if err != nil {
		t.Fatalf("UpdateProgram: %v", err)
	}
	if repo.updateGotTrainerID != trainerID || repo.updateGotProgramID != programID {
		t.Fatalf("expected update scope %q/%q, got %q/%q", trainerID, programID, repo.updateGotTrainerID, repo.updateGotProgramID)
	}
	if repo.updateGotUpdates["name"] != name {
		t.Fatalf("expected name update, got %v", repo.updateGotUpdates)
	}
	if _, present := repo.updateGotUpdates["trainer_id"]; present {
		t.Fatal("trainer_id must never be updatable")
	}
	if program.Name != name || program.Status != status {
		t.Fatalf("unexpected program %+v", program)
	}
}

func TestUpdateProgramRejectsInvalidIDs(t *testing.T) {
	svc := newService(&stubProgramRepo{})

	for tcName, ids := range map[string][2]string{
		"empty trainer": {"", programID},
		"bad trainer":   {"not-a-uuid", programID},
		"empty program": {trainerID, ""},
		"bad program":   {trainerID, "not-a-uuid"},
	} {
		t.Run(tcName, func(t *testing.T) {
			newName := "New Name"
			if _, err := svc.UpdateProgram(context.Background(), ids[0], ids[1], programs.UpdateProgramInput{Name: &newName}); !errors.Is(err, programs.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestUpdateProgramEmptyUpdate(t *testing.T) {
	svc := newService(&stubProgramRepo{program: validProgram()})

	if _, err := svc.UpdateProgram(context.Background(), trainerID, programID, programs.UpdateProgramInput{}); !errors.Is(err, programs.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for empty update, got %v", err)
	}
}

func TestUpdateProgramInvalidField(t *testing.T) {
	svc := newService(&stubProgramRepo{program: validProgram()})

	badType := "random"
	if _, err := svc.UpdateProgram(context.Background(), trainerID, programID, programs.UpdateProgramInput{Type: &badType}); !errors.Is(err, programs.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for invalid type, got %v", err)
	}
}

func TestUpdateProgramNotFound(t *testing.T) {
	repo := &stubProgramRepo{
		find: func(_, _ string) (*models.Program, error) {
			return nil, repositories.ErrProgramNotFound
		},
	}
	svc := newService(repo)

	name := "New Name"
	if _, err := svc.UpdateProgram(context.Background(), trainerID, programID, programs.UpdateProgramInput{Name: &name}); !errors.Is(err, programs.ErrProgramNotFound) {
		t.Fatalf("expected ErrProgramNotFound, got %v", err)
	}
	if repo.updateGotProgramID != "" {
		t.Fatalf("no update may run for a missing program, got %q", repo.updateGotProgramID)
	}
}

func TestUpdateProgramRepositoryFailure(t *testing.T) {
	repo := &stubProgramRepo{
		program: validProgram(),
		update: func(_, _ string, _ map[string]any) error {
			return errRepoFailure
		},
	}
	svc := newService(repo)

	name := "New Name"
	_, err := svc.UpdateProgram(context.Background(), trainerID, programID, programs.UpdateProgramInput{Name: &name})
	if errors.Is(err, programs.ErrProgramNotFound) || errors.Is(err, programs.ErrInvalidInput) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestDeleteProgramSuccess(t *testing.T) {
	repo := &stubProgramRepo{program: validProgram()}
	svc := newService(repo)

	if err := svc.DeleteProgram(context.Background(), trainerID, programID); err != nil {
		t.Fatalf("DeleteProgram: %v", err)
	}
	if repo.deleteGotTrainerID != trainerID || repo.deleteGotProgramID != programID {
		t.Fatalf("expected soft delete on %q/%q, got %q/%q", trainerID, programID, repo.deleteGotTrainerID, repo.deleteGotProgramID)
	}
}

func TestDeleteProgramNotFound(t *testing.T) {
	repo := &stubProgramRepo{
		softDelete: func(_, _ string) error {
			return repositories.ErrProgramNotFound
		},
	}
	svc := newService(repo)

	if err := svc.DeleteProgram(context.Background(), trainerID, programID); !errors.Is(err, programs.ErrProgramNotFound) {
		t.Fatalf("expected ErrProgramNotFound, got %v", err)
	}
}

func TestDeleteProgramRejectsInvalidIDs(t *testing.T) {
	svc := newService(&stubProgramRepo{})

	for name, ids := range map[string][2]string{
		"empty trainer": {"", programID},
		"bad program":   {trainerID, "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := svc.DeleteProgram(context.Background(), ids[0], ids[1]); !errors.Is(err, programs.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestDeleteProgramRepositoryFailure(t *testing.T) {
	repo := &stubProgramRepo{
		softDelete: func(_, _ string) error {
			return errRepoFailure
		},
	}
	svc := newService(repo)

	err := svc.DeleteProgram(context.Background(), trainerID, programID)
	if errors.Is(err, programs.ErrProgramNotFound) || errors.Is(err, programs.ErrInvalidInput) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestPublishProgramSuccess(t *testing.T) {
	draftProgram := validProgram()
	draftProgram.Status = models.ProgramStatusDraft
	repo := &stubProgramRepo{program: draftProgram}
	svc := newService(repo)

	program, err := svc.PublishProgram(context.Background(), trainerID, programID)
	if err != nil {
		t.Fatalf("PublishProgram: %v", err)
	}
	if program.Status != models.ProgramStatusPublished {
		t.Fatalf("expected published, got %q", program.Status)
	}
	if repo.publishGotTrainerID != trainerID || repo.publishGotProgramID != programID {
		t.Fatalf("expected publish on %q/%q, got %q/%q", trainerID, programID, repo.publishGotTrainerID, repo.publishGotProgramID)
	}
}

func TestPublishProgramAlreadyPublished(t *testing.T) {
	publishedProgram := validProgram()
	publishedProgram.Status = models.ProgramStatusPublished
	repo := &stubProgramRepo{
		program: publishedProgram,
		publish: func(_, _ string) error {
			return repositories.ErrProgramNotFound
		},
	}
	svc := newService(repo)

	_, err := svc.PublishProgram(context.Background(), trainerID, programID)
	if !errors.Is(err, programs.ErrProgramAlreadyPublished) {
		t.Fatalf("expected ErrProgramAlreadyPublished, got %v", err)
	}
}

func TestPublishProgramNotFound(t *testing.T) {
	repo := &stubProgramRepo{
		publish: func(_, _ string) error {
			return repositories.ErrProgramNotFound
		},
		find: func(_, _ string) (*models.Program, error) {
			return nil, repositories.ErrProgramNotFound
		},
	}
	svc := newService(repo)

	_, err := svc.PublishProgram(context.Background(), trainerID, programID)
	if !errors.Is(err, programs.ErrProgramNotFound) {
		t.Fatalf("expected ErrProgramNotFound, got %v", err)
	}
}

func TestPublishProgramRejectsInvalidIDs(t *testing.T) {
	svc := newService(&stubProgramRepo{})

	for name, ids := range map[string][2]string{
		"empty trainer": {"", programID},
		"bad program":   {trainerID, "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := svc.PublishProgram(context.Background(), ids[0], ids[1])
			if !errors.Is(err, programs.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestPublishProgramRepositoryFailure(t *testing.T) {
	repo := &stubProgramRepo{
		publish: func(_, _ string) error {
			return errRepoFailure
		},
	}
	svc := newService(repo)

	_, err := svc.PublishProgram(context.Background(), trainerID, programID)
	if errors.Is(err, programs.ErrProgramNotFound) || errors.Is(err, programs.ErrInvalidInput) || errors.Is(err, programs.ErrProgramAlreadyPublished) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestProgramNeverExposesSecrets(t *testing.T) {
	repo := &stubProgramRepo{program: validProgram()}
	svc := newService(repo)

	program, err := svc.GetProgram(context.Background(), trainerID, programID)
	if err != nil {
		t.Fatalf("GetProgram: %v", err)
	}
	if program.Name == "" || program.Type == "" {
		t.Fatal("safe program fields must be present")
	}
	if program.TrainerID != trainerID {
		t.Fatalf("unexpected trainer for a trainer-owned program, got %q", program.TrainerID)
	}

	// The Program struct is the only shape this service ever returns. Structural
	// guarantee: it carries metadata only and no sensitive or internal field can
	// reach the caller.
	typ := reflect.TypeOf(*program)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i).Name
		for _, sensitive := range []string{"password", "token", "secret", "session", "deleted"} {
			if strings.Contains(strings.ToLower(field), sensitive) {
				t.Fatalf("Program must not expose %q", field)
			}
		}
	}
}
