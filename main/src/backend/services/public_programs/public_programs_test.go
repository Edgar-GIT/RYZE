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
	search           func(query, programType, sortBy, order string, page, limit int) ([]models.Program, int64, error)
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

func (s *stubProgramRepo) SearchPublished(_ context.Context, query, programType, sortBy, order string, page, limit int) ([]models.Program, int64, error) {
	if s.search != nil {
		return s.search(query, programType, sortBy, order, page, limit)
	}
	if s.program == nil || s.deleted {
		return nil, 0, nil
	}
	return []models.Program{*s.program}, 1, nil
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

func TestSearchPublishedProgramsSuccess(t *testing.T) {
	var gotQuery, gotType, gotSort, gotOrder string
	var gotPage, gotLimit int
	repo := &stubProgramRepo{
		search: func(query, programType, sortBy, order string, page, limit int) ([]models.Program, int64, error) {
			gotQuery = query
			gotType = programType
			gotSort = sortBy
			gotOrder = order
			gotPage = page
			gotLimit = limit
			return []models.Program{*validProgram()}, 1, nil
		},
	}
	svc := newService(repo)

	result, err := svc.SearchPublishedPrograms(context.Background(), "strength", "premium", "name", "asc", 2, 10)
	if err != nil {
		t.Fatalf("SearchPublishedPrograms: %v", err)
	}
	if gotQuery != "strength" || gotType != "premium" || gotSort != "name" || gotOrder != "asc" {
		t.Fatalf("expected query/type/sort/order forwarded, got %q/%q/%q/%q", gotQuery, gotType, gotSort, gotOrder)
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
}

func TestSearchPublishedProgramsClampsLimit(t *testing.T) {
	var gotLimit int
	repo := &stubProgramRepo{
		search: func(_, _ string, _ string, _ string, _, limit int) ([]models.Program, int64, error) {
			gotLimit = limit
			return nil, 0, nil
		},
	}
	svc := newService(repo)

	if _, err := svc.SearchPublishedPrograms(context.Background(), "", "", "", "", 1, 99999); err != nil {
		t.Fatalf("SearchPublishedPrograms: %v", err)
	}
	if gotLimit != public_programs.MaxPageSize {
		t.Fatalf("expected limit clamped to %d, got %d", public_programs.MaxPageSize, gotLimit)
	}
}

func TestSearchPublishedProgramsRejectsInvalidPagination(t *testing.T) {
	svc := newService(&stubProgramRepo{})

	for name, args := range map[string][2]int{
		"page zero":      {0, 10},
		"page negative":  {-1, 10},
		"limit zero":     {1, 0},
		"limit negative": {1, -5},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.SearchPublishedPrograms(context.Background(), "", "", "", "", args[0], args[1]); !errors.Is(err, public_programs.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestSearchPublishedProgramsRejectsLongQuery(t *testing.T) {
	svc := newService(&stubProgramRepo{})

	longQuery := strings.Repeat("a", public_programs.MaxSearchQueryLength+1)
	_, err := svc.SearchPublishedPrograms(context.Background(), longQuery, "", "", "", 1, 10)
	if !errors.Is(err, public_programs.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for long query, got %v", err)
	}
}

func TestSearchPublishedProgramsTrimsAndAllowsMaxLengthQuery(t *testing.T) {
	var gotQuery string
	repo := &stubProgramRepo{
		search: func(query, _, _, _ string, _, _ int) ([]models.Program, int64, error) {
			gotQuery = query
			return nil, 0, nil
		},
	}
	svc := newService(repo)

	maxQuery := strings.Repeat("a", public_programs.MaxSearchQueryLength)
	_, err := svc.SearchPublishedPrograms(context.Background(), "  "+maxQuery+"  ", "", "", "", 1, 10)
	if err != nil {
		t.Fatalf("SearchPublishedPrograms: %v", err)
	}
	if gotQuery != maxQuery {
		t.Fatalf("expected trimmed query of length %d, got %q (len %d)", public_programs.MaxSearchQueryLength, gotQuery, len(gotQuery))
	}
}

func TestSearchPublishedProgramsRejectsInvalidType(t *testing.T) {
	svc := newService(&stubProgramRepo{})

	_, err := svc.SearchPublishedPrograms(context.Background(), "", "invalid_type", "", "", 1, 10)
	if !errors.Is(err, public_programs.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for invalid type, got %v", err)
	}
}

func TestSearchPublishedProgramsAllowsValidTypes(t *testing.T) {
	for _, pt := range []string{"free", "premium", "personalized"} {
		repo := &stubProgramRepo{
			search: func(query, programType, sortBy, order string, page, limit int) ([]models.Program, int64, error) {
				if programType != pt {
					t.Fatalf("expected type %q, got %q", pt, programType)
				}
				return nil, 0, nil
			},
		}
		svc := newService(repo)
		if _, err := svc.SearchPublishedPrograms(context.Background(), "", pt, "", "", 1, 10); err != nil {
			t.Fatalf("type %q: expected no error, got %v", pt, err)
		}
	}
}

func TestSearchPublishedProgramsSortFallback(t *testing.T) {
	var gotSort, gotOrder string
	repo := &stubProgramRepo{
		search: func(_, _, sortBy, order string, _, _ int) ([]models.Program, int64, error) {
			gotSort = sortBy
			gotOrder = order
			return nil, 0, nil
		},
	}
	svc := newService(repo)

	_, err := svc.SearchPublishedPrograms(context.Background(), "", "", "invalid_sort", "INVALID", 1, 10)
	if err != nil {
		t.Fatalf("SearchPublishedPrograms: %v", err)
	}
	if gotSort != "" || gotOrder != "" {
		t.Fatalf("expected invalid sort/order to fall back to empty, got sort=%q order=%q", gotSort, gotOrder)
	}
}

func TestSearchPublishedProgramsRepositoryFailure(t *testing.T) {
	repo := &stubProgramRepo{
		search: func(_, _, _, _ string, _, _ int) ([]models.Program, int64, error) {
			return nil, 0, errRepoFailure
		},
	}
	svc := newService(repo)

	_, err := svc.SearchPublishedPrograms(context.Background(), "", "", "", "", 1, 10)
	if errors.Is(err, public_programs.ErrInvalidInput) || errors.Is(err, public_programs.ErrProgramNotFound) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestSearchPublishedProgramsMapsSafeDTO(t *testing.T) {
	p := validProgram()
	p.ID = "33333333-3333-3333-3333-333333333333"
	p.Name = "Search Result"
	p.TrainerID = "44444444-4444-4444-4444-444444444444"
	repo := &stubProgramRepo{
		search: func(_, _, _, _ string, _, _ int) ([]models.Program, int64, error) {
			return []models.Program{*p}, 1, nil
		},
	}
	svc := newService(repo)

	result, err := svc.SearchPublishedPrograms(context.Background(), "", "", "", "", 1, 10)
	if err != nil {
		t.Fatalf("SearchPublishedPrograms: %v", err)
	}
	if len(result.Programs) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Programs))
	}
	prog := result.Programs[0]
	if prog.ID != p.ID || prog.Name != "Search Result" || prog.TrainerID != p.TrainerID {
		t.Fatalf("unexpected program %+v", prog)
	}
}
