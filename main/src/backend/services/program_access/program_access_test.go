package program_access_test

import (
	"context"
	"errors"
	"testing"

	"ryze/backend/config"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/program_access"
)

const (
	userID    = "22222222-2222-2222-2222-222222222222"
	programID = "44444444-4444-4444-4444-444444444444"
	entID     = "99999999-9999-9999-9999-999999999999"
)

var errRepoFailure = errors.New("repository failure")

type stubEntitlements struct {
	found      bool
	err        error
	gotUser    string
	gotProgram string
}

func (s *stubEntitlements) FindActiveByUserAndProgram(_ context.Context, userID, programID string) (*models.Entitlement, error) {
	s.gotUser = userID
	s.gotProgram = programID
	if s.err != nil {
		return nil, s.err
	}
	if !s.found {
		return nil, repositories.ErrEntitlementNotFound
	}
	return &models.Entitlement{ID: entID, UserID: userID, ProgramID: programID}, nil
}

type stubAssignments struct {
	exists  bool
	err     error
	queried bool
}

func (s *stubAssignments) ExistsByUserAndProgram(_ context.Context, _, _ string) (bool, error) {
	s.queried = true
	return s.exists, s.err
}

type stubActivations struct {
	found   bool
	err     error
	queried bool
}

func (s *stubActivations) FindActiveByUserAndProgram(_ context.Context, _, _ string) (*models.FreeProgramActivation, error) {
	s.queried = true
	if s.err != nil {
		return nil, s.err
	}
	if !s.found {
		return nil, repositories.ErrFreeActivationNotFound
	}
	return &models.FreeProgramActivation{ID: "activation", UserID: userID, ProgramID: programID}, nil
}

type stubPrograms struct {
	program    *models.Program
	err        error
	gotProgram string
}

func (s *stubPrograms) FindPublishedByIDWithStructure(_ context.Context, programID string) (*models.Program, error) {
	s.gotProgram = programID
	if s.err != nil {
		return nil, s.err
	}
	if s.program == nil {
		return nil, repositories.ErrProgramNotFound
	}
	return s.program, nil
}

func newService(ents *stubEntitlements, assignments *stubAssignments, activations *stubActivations, programs *stubPrograms) program_access.Service {
	return program_access.NewService(ents, assignments, activations, programs, config.ExerciseMediaConfig{})
}

func validProgram() *models.Program {
	return &models.Program{
		ID:       programID,
		Name:     "Strength Builder",
		Type:     models.ProgramTypePremium,
		Status:   models.ProgramStatusPublished,
		Currency: "EUR",
	}
}

func TestGetProgramAccessFromEntitlement(t *testing.T) {
	ents := &stubEntitlements{found: true}
	assignments := &stubAssignments{}
	activations := &stubActivations{}
	programs := &stubPrograms{program: validProgram()}
	svc := newService(ents, assignments, activations, programs)

	detail, err := svc.GetProgramAccess(context.Background(), userID, programID)
	if err != nil {
		t.Fatalf("GetProgramAccess: %v", err)
	}
	if detail == nil || detail.ID != programID {
		t.Fatalf("unexpected detail %+v", detail)
	}
	if ents.gotUser != userID || ents.gotProgram != programID {
		t.Fatalf("unexpected entitlement scope %q/%q", ents.gotUser, ents.gotProgram)
	}
	if programs.gotProgram != programID {
		t.Fatalf("expected program %q, got %q", programID, programs.gotProgram)
	}
	if assignments.queried || activations.queried {
		t.Fatal("a granted entitlement must short-circuit the other grant sources")
	}
}

func TestGetProgramAccessFromAssignment(t *testing.T) {
	svc := newService(&stubEntitlements{}, &stubAssignments{exists: true}, &stubActivations{}, &stubPrograms{program: validProgram()})

	detail, err := svc.GetProgramAccess(context.Background(), userID, programID)
	if err != nil {
		t.Fatalf("GetProgramAccess: %v", err)
	}
	if detail == nil || detail.ID != programID {
		t.Fatalf("unexpected detail %+v", detail)
	}
}

func TestGetProgramAccessFromFreeActivation(t *testing.T) {
	free := validProgram()
	free.Type = models.ProgramTypeFree
	free.ProductType = models.ProgramProductTypeGeneric
	svc := newService(&stubEntitlements{}, &stubAssignments{}, &stubActivations{found: true}, &stubPrograms{program: free})

	detail, err := svc.GetProgramAccess(context.Background(), userID, programID)
	if err != nil {
		t.Fatalf("GetProgramAccess: %v", err)
	}
	if detail == nil || detail.ID != programID {
		t.Fatalf("unexpected detail %+v", detail)
	}
}

func TestFreeActivationCannotOpenPaidProgram(t *testing.T) {
	svc := newService(&stubEntitlements{}, &stubAssignments{}, &stubActivations{found: true}, &stubPrograms{program: validProgram()})

	if _, err := svc.GetProgramAccess(context.Background(), userID, programID); !errors.Is(err, program_access.ErrProgramNotAccessible) {
		t.Fatalf("expected ErrProgramNotAccessible, got %v", err)
	}
}

func TestGetProgramAccessNoGrantDoesNotLoadProgram(t *testing.T) {
	programs := &stubPrograms{program: validProgram()}
	svc := newService(&stubEntitlements{}, &stubAssignments{}, &stubActivations{}, programs)

	if _, err := svc.GetProgramAccess(context.Background(), userID, programID); !errors.Is(err, program_access.ErrProgramNotAccessible) {
		t.Fatalf("expected ErrProgramNotAccessible, got %v", err)
	}
	if programs.gotProgram != "" {
		t.Fatalf("program must not be loaded without a grant, got read for %q", programs.gotProgram)
	}
}

func TestGetProgramAccessProgramUnavailable(t *testing.T) {
	svc := newService(&stubEntitlements{found: true}, &stubAssignments{}, &stubActivations{}, &stubPrograms{err: repositories.ErrProgramNotFound})

	if _, err := svc.GetProgramAccess(context.Background(), userID, programID); !errors.Is(err, program_access.ErrProgramNotAccessible) {
		t.Fatalf("expected ErrProgramNotAccessible, got %v", err)
	}
}

func TestGetProgramAccessInvalidInput(t *testing.T) {
	svc := newService(&stubEntitlements{found: true}, &stubAssignments{}, &stubActivations{}, &stubPrograms{program: validProgram()})

	cases := map[string][2]string{
		"empty user":    {"", programID},
		"bad user":      {"not-a-uuid", programID},
		"empty program": {userID, ""},
		"bad program":   {userID, "not-a-uuid"},
		"both empty":    {"", ""},
	}
	for name, pair := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.GetProgramAccess(context.Background(), pair[0], pair[1]); !errors.Is(err, program_access.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestGetProgramAccessGrantFailure(t *testing.T) {
	svc := newService(&stubEntitlements{err: errRepoFailure}, &stubAssignments{}, &stubActivations{}, &stubPrograms{program: validProgram()})

	_, err := svc.GetProgramAccess(context.Background(), userID, programID)
	if err == nil || errors.Is(err, program_access.ErrInvalidInput) || errors.Is(err, program_access.ErrProgramNotAccessible) {
		t.Fatalf("expected internal failure to be hidden, got %v", err)
	}
	if !errors.Is(err, errRepoFailure) {
		t.Fatalf("expected the repository failure to stay wrapped, got %v", err)
	}
}

func TestResolveGrantReportsSource(t *testing.T) {
	svc := newService(&stubEntitlements{found: true}, &stubAssignments{}, &stubActivations{}, &stubPrograms{program: validProgram()})

	source, err := svc.ResolveGrant(context.Background(), userID, programID)
	if err != nil {
		t.Fatalf("ResolveGrant: %v", err)
	}
	if source != program_access.GrantSourceEntitlement {
		t.Fatalf("expected entitlement source, got %q", source)
	}
}
