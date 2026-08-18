package program_assignments_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/program_assignments"
)

const (
	trainerID     = "11111111-1111-1111-1111-111111111111"
	clientID      = "22222222-2222-2222-2222-222222222222"
	otherClientID = "33333333-3333-3333-3333-333333333333"
	programID     = "44444444-4444-4444-4444-444444444444"
	otherProgram  = "55555555-5555-5555-5555-555555555555"
	assignmentID  = "66666666-6666-6666-6666-666666666666"
)

var errRepoFailure = errors.New("repository failure")

// stubRepo is an in-memory fake of the program assignment data-access surface.
// It records every identifier forwarded to the repository so tests can prove
// the service forwards the trainer context identity and never invents or
// accepts a client-supplied one.
type stubRepo struct {
	create       func(trainerID, userID, programID string, assignment *models.ProgramAssignment) error
	list         func(trainerID, userID string) ([]models.ProgramAssignment, error)
	find         func(trainerID, userID, assignmentID string) (*models.ProgramAssignment, error)
	softDelete   func(trainerID, userID, assignmentID string) error
	gotTrainerID string
	gotUserID    string
	gotProgramID string
	gotEntryID   string
	createdEntry *models.ProgramAssignment
}

func (s *stubRepo) Create(_ context.Context, trainerID, userID, programID string, assignment *models.ProgramAssignment) error {
	s.gotTrainerID = trainerID
	s.gotUserID = userID
	s.gotProgramID = programID
	if s.create != nil {
		return s.create(trainerID, userID, programID, assignment)
	}
	assignment.ID = assignmentID
	assignment.TrainerID = trainerID
	assignment.ProgramID = programID
	assignment.UserID = userID
	assignment.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	assignment.UpdatedAt = assignment.CreatedAt
	assignment.Program = validProgram()
	s.createdEntry = assignment
	return nil
}

func (s *stubRepo) ListByClient(_ context.Context, trainerID, userID string) ([]models.ProgramAssignment, error) {
	s.gotTrainerID = trainerID
	s.gotUserID = userID
	if s.list != nil {
		return s.list(trainerID, userID)
	}
	return nil, nil
}

func (s *stubRepo) FindByIDAndClient(_ context.Context, trainerID, userID, assignmentID string) (*models.ProgramAssignment, error) {
	s.gotTrainerID = trainerID
	s.gotUserID = userID
	s.gotEntryID = assignmentID
	if s.find != nil {
		return s.find(trainerID, userID, assignmentID)
	}
	return nil, repositories.ErrAssignmentNotFound
}

func (s *stubRepo) SoftDelete(_ context.Context, trainerID, userID, assignmentID string) error {
	s.gotTrainerID = trainerID
	s.gotUserID = userID
	s.gotEntryID = assignmentID
	if s.softDelete != nil {
		return s.softDelete(trainerID, userID, assignmentID)
	}
	return nil
}

func newService(repo *stubRepo) program_assignments.Service {
	return program_assignments.NewService(repo)
}

func validProgram() models.Program {
	return models.Program{
		ID:          programID,
		TrainerID:   trainerID,
		Name:        "Strength Builder",
		Description: "Progressive strength program",
		Type:        models.ProgramTypePremium,
		Status:      models.ProgramStatusDraft,
		CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func validModel() *models.ProgramAssignment {
	return &models.ProgramAssignment{
		ID:        assignmentID,
		TrainerID: trainerID,
		ProgramID: programID,
		UserID:    clientID,
		Program:   validProgram(),
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestAssignProgramSuccess(t *testing.T) {
	repo := &stubRepo{}
	svc := newService(repo)

	assignment, err := svc.AssignProgram(context.Background(), trainerID, clientID, programID)
	if err != nil {
		t.Fatalf("AssignProgram: %v", err)
	}
	if repo.gotTrainerID != trainerID || repo.gotUserID != clientID || repo.gotProgramID != programID {
		t.Fatalf("expected scope %q/%q/%q, got %q/%q/%q", trainerID, clientID, programID, repo.gotTrainerID, repo.gotUserID, repo.gotProgramID)
	}
	if assignment.ID != assignmentID || assignment.TrainerID != trainerID || assignment.ProgramID != programID || assignment.UserID != clientID {
		t.Fatalf("unexpected assignment %+v", assignment)
	}
	if assignment.Program.ID != programID || assignment.Program.Name != "Strength Builder" || assignment.Program.TrainerID != trainerID {
		t.Fatalf("expected safe program embedded, got %+v", assignment.Program)
	}
}

func TestAssignProgramInvalidInput(t *testing.T) {
	svc := newService(&stubRepo{})

	cases := map[string][3]string{
		"empty trainer": {"", clientID, programID},
		"bad trainer":   {"not-a-uuid", clientID, programID},
		"empty client":  {trainerID, "", programID},
		"bad client":    {trainerID, "not-a-uuid", programID},
		"empty program": {trainerID, clientID, ""},
		"bad program":   {trainerID, clientID, "not-a-uuid"},
	}
	for name, ids := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.AssignProgram(context.Background(), ids[0], ids[1], ids[2]); !errors.Is(err, program_assignments.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestAssignProgramClientRelationNotFound(t *testing.T) {
	repo := &stubRepo{
		create: func(_, _, _ string, _ *models.ProgramAssignment) error {
			return repositories.ErrClientRelationNotFound
		},
	}
	svc := newService(repo)

	if _, err := svc.AssignProgram(context.Background(), trainerID, clientID, programID); !errors.Is(err, program_assignments.ErrClientRelationNotFound) {
		t.Fatalf("expected ErrClientRelationNotFound, got %v", err)
	}
}

func TestAssignProgramProgramNotFound(t *testing.T) {
	repo := &stubRepo{
		create: func(_, _, _ string, _ *models.ProgramAssignment) error {
			return repositories.ErrProgramNotFound
		},
	}
	svc := newService(repo)

	if _, err := svc.AssignProgram(context.Background(), trainerID, clientID, programID); !errors.Is(err, program_assignments.ErrProgramNotFound) {
		t.Fatalf("expected ErrProgramNotFound, got %v", err)
	}
}

func TestAssignProgramAlreadyActive(t *testing.T) {
	repo := &stubRepo{
		create: func(_, _, _ string, _ *models.ProgramAssignment) error {
			return repositories.ErrAssignmentAlreadyActive
		},
	}
	svc := newService(repo)

	if _, err := svc.AssignProgram(context.Background(), trainerID, clientID, programID); !errors.Is(err, program_assignments.ErrAssignmentAlreadyActive) {
		t.Fatalf("expected ErrAssignmentAlreadyActive, got %v", err)
	}
}

func TestAssignProgramRepoFailureNotExposed(t *testing.T) {
	repo := &stubRepo{
		create: func(_, _, _ string, _ *models.ProgramAssignment) error {
			return errRepoFailure
		},
	}
	svc := newService(repo)

	_, err := svc.AssignProgram(context.Background(), trainerID, clientID, programID)
	if err == nil || errors.Is(err, program_assignments.ErrInvalidInput) {
		t.Fatalf("expected an internal failure to be wrapped and hidden, got %v", err)
	}
	if !errors.Is(err, errRepoFailure) {
		t.Fatalf("expected the repository failure to stay wrapped for logs, got %v", err)
	}
}

func TestListClientProgramsSuccess(t *testing.T) {
	repo := &stubRepo{
		list: func(_, _ string) ([]models.ProgramAssignment, error) {
			return []models.ProgramAssignment{*validModel()}, nil
		},
	}
	svc := newService(repo)

	assignments, err := svc.ListClientPrograms(context.Background(), trainerID, clientID)
	if err != nil {
		t.Fatalf("ListClientPrograms: %v", err)
	}
	if repo.gotTrainerID != trainerID || repo.gotUserID != clientID {
		t.Fatalf("expected scope %q/%q, got %q/%q", trainerID, clientID, repo.gotTrainerID, repo.gotUserID)
	}
	if len(assignments) != 1 {
		t.Fatalf("expected 1 assignment, got %d", len(assignments))
	}
	if assignments[0].ID != assignmentID || assignments[0].Program.ID != programID {
		t.Fatalf("unexpected assignment %+v", assignments[0])
	}
}

func TestListClientProgramsEmptyList(t *testing.T) {
	repo := &stubRepo{
		list: func(_, _ string) ([]models.ProgramAssignment, error) {
			return []models.ProgramAssignment{}, nil
		},
	}
	svc := newService(repo)

	assignments, err := svc.ListClientPrograms(context.Background(), trainerID, clientID)
	if err != nil {
		t.Fatalf("ListClientPrograms: %v", err)
	}
	if assignments == nil || len(assignments) != 0 {
		t.Fatalf("expected an empty non-nil list, got %#v", assignments)
	}
}

func TestListClientProgramsInvalidInput(t *testing.T) {
	svc := newService(&stubRepo{})

	cases := map[string][2]string{
		"empty trainer": {"", clientID},
		"bad trainer":   {"not-a-uuid", clientID},
		"empty client":  {trainerID, ""},
		"bad client":    {trainerID, "not-a-uuid"},
	}
	for name, ids := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.ListClientPrograms(context.Background(), ids[0], ids[1]); !errors.Is(err, program_assignments.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestListClientProgramsRepoFailureNotExposed(t *testing.T) {
	repo := &stubRepo{
		list: func(_, _ string) ([]models.ProgramAssignment, error) {
			return nil, errRepoFailure
		},
	}
	svc := newService(repo)

	_, err := svc.ListClientPrograms(context.Background(), trainerID, clientID)
	if err == nil || errors.Is(err, program_assignments.ErrInvalidInput) {
		t.Fatalf("expected an internal failure to be hidden, got %v", err)
	}
	if !errors.Is(err, errRepoFailure) {
		t.Fatalf("expected the repository failure to stay wrapped for logs, got %v", err)
	}
}

func TestRemoveAssignmentSuccess(t *testing.T) {
	repo := &stubRepo{}
	svc := newService(repo)

	if err := svc.RemoveAssignment(context.Background(), trainerID, clientID, assignmentID); err != nil {
		t.Fatalf("RemoveAssignment: %v", err)
	}
	if repo.gotTrainerID != trainerID || repo.gotUserID != clientID || repo.gotEntryID != assignmentID {
		t.Fatalf("expected scope %q/%q/%q, got %q/%q/%q", trainerID, clientID, assignmentID, repo.gotTrainerID, repo.gotUserID, repo.gotEntryID)
	}
}

func TestRemoveAssignmentInvalidInput(t *testing.T) {
	svc := newService(&stubRepo{})

	cases := map[string][3]string{
		"empty trainer":    {"", clientID, assignmentID},
		"bad trainer":      {"not-a-uuid", clientID, assignmentID},
		"empty client":     {trainerID, "", assignmentID},
		"bad client":       {trainerID, "not-a-uuid", assignmentID},
		"empty assignment": {trainerID, clientID, ""},
		"bad assignment":   {trainerID, clientID, "not-a-uuid"},
	}
	for name, ids := range cases {
		t.Run(name, func(t *testing.T) {
			if err := svc.RemoveAssignment(context.Background(), ids[0], ids[1], ids[2]); !errors.Is(err, program_assignments.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestRemoveAssignmentNotFound(t *testing.T) {
	repo := &stubRepo{
		softDelete: func(_, _, _ string) error {
			return repositories.ErrAssignmentNotFound
		},
	}
	svc := newService(repo)

	if err := svc.RemoveAssignment(context.Background(), trainerID, clientID, assignmentID); !errors.Is(err, program_assignments.ErrAssignmentNotFound) {
		t.Fatalf("expected ErrAssignmentNotFound, got %v", err)
	}
}

func TestRemoveAssignmentRepoFailureNotExposed(t *testing.T) {
	repo := &stubRepo{
		softDelete: func(_, _, _ string) error {
			return errRepoFailure
		},
	}
	svc := newService(repo)

	err := svc.RemoveAssignment(context.Background(), trainerID, clientID, assignmentID)
	if err == nil || errors.Is(err, program_assignments.ErrInvalidInput) {
		t.Fatalf("expected an internal failure to be hidden, got %v", err)
	}
	if !errors.Is(err, errRepoFailure) {
		t.Fatalf("expected the repository failure to stay wrapped for logs, got %v", err)
	}
}
