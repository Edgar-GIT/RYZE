package program_access_test

import (
	"context"
	"errors"
	"testing"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/program_access"
	"ryze/backend/services/public_programs"
)

const (
	userID    = "22222222-2222-2222-2222-222222222222"
	programID = "44444444-4444-4444-4444-444444444444"
	entID     = "99999999-9999-9999-9999-999999999999"
)

var errRepoFailure = errors.New("repository failure")

// stubrEntitlements fakes the entitlement gate. It records the forwarded user
// and program ids so tests can prove the service scopes the check to the
// authentication-context identity and to the requested program.
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

// stubPrograms fakes the published program reader.
type stubPrograms struct {
	detail     *public_programs.ProgramDetail
	err        error
	gotProgram string
}

func (s *stubPrograms) GetPublishedProgram(_ context.Context, programID string) (*public_programs.ProgramDetail, error) {
	s.gotProgram = programID
	if s.err != nil {
		return nil, s.err
	}
	if s.detail == nil {
		return nil, public_programs.ErrProgramNotFound
	}
	return s.detail, nil
}

func newService(ents *stubEntitlements, programs *stubPrograms) program_access.Service {
	return program_access.NewService(ents, programs)
}

func validDetail() *public_programs.ProgramDetail {
	return &public_programs.ProgramDetail{
		Program: public_programs.Program{
			ID:       programID,
			Name:     "Strength Builder",
			Type:     models.ProgramTypePremium,
			Status:   models.ProgramStatusPublished,
			Currency: "EUR",
		},
		Weeks: []public_programs.Week{},
	}
}

func TestGetProgramAccessSuccess(t *testing.T) {
	ents := &stubEntitlements{found: true}
	programs := &stubPrograms{detail: validDetail()}
	svc := newService(ents, programs)

	detail, err := svc.GetProgramAccess(context.Background(), userID, programID)
	if err != nil {
		t.Fatalf("GetProgramAccess: %v", err)
	}
	if detail == nil || detail.ID != programID || detail.Name != "Strength Builder" {
		t.Fatalf("unexpected detail %+v", detail)
	}
	if ents.gotUser != userID {
		t.Fatalf("expected scope %q, got %q", userID, ents.gotUser)
	}
	if ents.gotProgram != programID {
		t.Fatalf("expected program %q, got %q", programID, ents.gotProgram)
	}
	if programs.gotProgram != programID {
		t.Fatalf("expected program read %q, got %q", programID, programs.gotProgram)
	}
}

func TestGetProgramAccessNoEntitlement(t *testing.T) {
	ents := &stubEntitlements{found: false}
	programs := &stubPrograms{detail: validDetail()}
	svc := newService(ents, programs)

	_, err := svc.GetProgramAccess(context.Background(), userID, programID)
	if !errors.Is(err, program_access.ErrProgramNotAccessible) {
		t.Fatalf("expected ErrProgramNotAccessible, got %v", err)
	}
	// The program must never be loaded without an entitlement.
	if programs.gotProgram != "" {
		t.Fatalf("program must not be loaded without an entitlement, got read for %q", programs.gotProgram)
	}
}

func TestGetProgramAccessProgramUnavailable(t *testing.T) {
	ents := &stubEntitlements{found: true}

	for name, programs := range map[string]*stubPrograms{
		"not found": {err: public_programs.ErrProgramNotFound},
	} {
		t.Run(name, func(t *testing.T) {
			svc := newService(ents, programs)
			_, err := svc.GetProgramAccess(context.Background(), userID, programID)
			if !errors.Is(err, program_access.ErrProgramNotAccessible) {
				t.Fatalf("expected ErrProgramNotAccessible, got %v", err)
			}
		})
	}
}

func TestGetProgramAccessInvalidInput(t *testing.T) {
	svc := newService(&stubEntitlements{found: true}, &stubPrograms{detail: validDetail()})

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

func TestGetProgramAccessEntitlementFailure(t *testing.T) {
	svc := newService(&stubEntitlements{err: errRepoFailure}, &stubPrograms{detail: validDetail()})

	_, err := svc.GetProgramAccess(context.Background(), userID, programID)
	if err == nil || errors.Is(err, program_access.ErrInvalidInput) || errors.Is(err, program_access.ErrProgramNotAccessible) {
		t.Fatalf("expected internal failure to be hidden, got %v", err)
	}
	if !errors.Is(err, errRepoFailure) {
		t.Fatalf("expected the repository failure to stay wrapped, got %v", err)
	}
}

func TestGetProgramAccessProgramFailure(t *testing.T) {
	svc := newService(&stubEntitlements{found: true}, &stubPrograms{err: errRepoFailure})

	_, err := svc.GetProgramAccess(context.Background(), userID, programID)
	if err == nil || errors.Is(err, program_access.ErrInvalidInput) || errors.Is(err, program_access.ErrProgramNotAccessible) {
		t.Fatalf("expected internal failure to be hidden, got %v", err)
	}
	if !errors.Is(err, errRepoFailure) {
		t.Fatalf("expected the repository failure to stay wrapped, got %v", err)
	}
}
