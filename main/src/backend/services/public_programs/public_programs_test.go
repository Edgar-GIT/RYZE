package public_programs_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/public_programs"
)

const programID = "22222222-2222-2222-2222-222222222222"

var errRepoFailure = errors.New("repository failure")

// stubProgramRepo is an in-memory fake of the public program data-access
// surface. It behaves like the real repository and records the identifiers
// passed to every operation so tests can prove the service forwards the
// correct arguments.
type stubProgramRepo struct {
	program          *models.Program
	deleted          bool
	list             func(page, limit int) ([]models.Program, int64, error)
	find             func(programID string) (*models.Program, error)
	gotPage          int
	gotLimit         int
	gotFindProgramID string
}

func (s *stubProgramRepo) ListPublished(_ context.Context, page, limit int) ([]models.Program, int64, error) {
	s.gotPage = page
	s.gotLimit = limit
	if s.list != nil {
		return s.list(page, limit)
	}
	if s.program == nil || s.deleted {
		return nil, 0, nil
	}
	return []models.Program{*s.program}, 1, nil
}

func (s *stubProgramRepo) FindPublishedByID(_ context.Context, programID string) (*models.Program, error) {
	s.gotFindProgramID = programID
	if s.find != nil {
		return s.find(programID)
	}
	if s.program == nil || s.deleted {
		return nil, repositories.ErrProgramNotFound
	}
	if s.program.ID != programID {
		return nil, repositories.ErrProgramNotFound
	}
	return s.program, nil
}

func validProgram() *models.Program {
	return &models.Program{
		ID:          programID,
		TrainerID:   "11111111-1111-1111-1111-111111111111",
		Name:        "Strength Builder",
		Description: "A 12 week strength program.",
		Type:        models.ProgramTypePremium,
		Status:      models.ProgramStatusPublished,
		CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
}

func newService(repo *stubProgramRepo) public_programs.Service {
	return public_programs.NewService(repo)
}

func TestListPublishedProgramsSuccess(t *testing.T) {
	var gotPage, gotLimit int
	repo := &stubProgramRepo{
		list: func(page, limit int) ([]models.Program, int64, error) {
			gotPage = page
			gotLimit = limit
			return []models.Program{*validProgram()}, 1, nil
		},
	}
	svc := newService(repo)

	result, err := svc.ListPublishedPrograms(context.Background(), 2, 10)
	if err != nil {
		t.Fatalf("ListPublishedPrograms: %v", err)
	}
	if gotPage != 2 || gotLimit != 10 {
		t.Fatalf("expected page 2 limit 10, got %d/%d", gotPage, gotLimit)
	}
	if result.Total != 1 || len(result.Programs) != 1 {
		t.Fatalf("expected one program, got %+v", result)
	}
	if result.Page != 2 || result.Limit != 10 {
		t.Fatalf("expected page 2 limit 10 in result, got %d/%d", result.Page, result.Limit)
	}
	if result.Programs[0].Name != "Strength Builder" {
		t.Fatalf("unexpected program %+v", result.Programs[0])
	}
}

func TestListPublishedProgramsClampsLimit(t *testing.T) {
	var gotLimit int
	repo := &stubProgramRepo{
		list: func(_, limit int) ([]models.Program, int64, error) {
			gotLimit = limit
			return nil, 0, nil
		},
	}
	svc := newService(repo)

	if _, err := svc.ListPublishedPrograms(context.Background(), 1, 99999); err != nil {
		t.Fatalf("ListPublishedPrograms: %v", err)
	}
	if gotLimit != public_programs.MaxPageSize {
		t.Fatalf("expected limit clamped to %d, got %d", public_programs.MaxPageSize, gotLimit)
	}
}

func TestListPublishedProgramsRejectsInvalidInput(t *testing.T) {
	svc := newService(&stubProgramRepo{})

	for name, args := range map[string][2]int{
		"page zero":      {0, 10},
		"page negative":  {-1, 10},
		"limit zero":     {1, 0},
		"limit negative": {1, -5},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.ListPublishedPrograms(context.Background(), args[0], args[1]); !errors.Is(err, public_programs.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestListPublishedProgramsRepositoryFailure(t *testing.T) {
	repo := &stubProgramRepo{
		list: func(_, _ int) ([]models.Program, int64, error) {
			return nil, 0, errRepoFailure
		},
	}
	svc := newService(repo)

	_, err := svc.ListPublishedPrograms(context.Background(), 1, 10)
	if errors.Is(err, public_programs.ErrInvalidInput) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestGetPublishedProgramSuccess(t *testing.T) {
	var gotProgramID string
	repo := &stubProgramRepo{
		find: func(programID string) (*models.Program, error) {
			gotProgramID = programID
			return validProgram(), nil
		},
	}
	svc := newService(repo)

	program, err := svc.GetPublishedProgram(context.Background(), programID)
	if err != nil {
		t.Fatalf("GetPublishedProgram: %v", err)
	}
	if gotProgramID != programID {
		t.Fatalf("expected program id %q, got %q", programID, gotProgramID)
	}
	if program.ID != programID || program.Type != models.ProgramTypePremium {
		t.Fatalf("unexpected program %+v", program)
	}
}

func TestGetPublishedProgramRejectsInvalidIDs(t *testing.T) {
	svc := newService(&stubProgramRepo{})

	for name, id := range map[string]string{
		"empty":    "",
		"blank":    "   ",
		"not uuid": "not-a-uuid",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.GetPublishedProgram(context.Background(), id); !errors.Is(err, public_programs.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestGetPublishedProgramNotFound(t *testing.T) {
	repo := &stubProgramRepo{
		find: func(_ string) (*models.Program, error) {
			return nil, repositories.ErrProgramNotFound
		},
	}
	svc := newService(repo)

	if _, err := svc.GetPublishedProgram(context.Background(), programID); !errors.Is(err, public_programs.ErrProgramNotFound) {
		t.Fatalf("expected ErrProgramNotFound, got %v", err)
	}
}

func TestGetPublishedProgramRepositoryFailure(t *testing.T) {
	repo := &stubProgramRepo{
		find: func(_ string) (*models.Program, error) {
			return nil, errRepoFailure
		},
	}
	svc := newService(repo)

	_, err := svc.GetPublishedProgram(context.Background(), programID)
	if errors.Is(err, public_programs.ErrProgramNotFound) || errors.Is(err, public_programs.ErrInvalidInput) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestProgramNeverExposesSecrets(t *testing.T) {
	repo := &stubProgramRepo{program: validProgram()}
	svc := newService(repo)

	program, err := svc.GetPublishedProgram(context.Background(), programID)
	if err != nil {
		t.Fatalf("GetPublishedProgram: %v", err)
	}
	if program.Name == "" || program.Type == "" {
		t.Fatal("safe program fields must be present")
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
