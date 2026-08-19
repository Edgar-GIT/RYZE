package entitlements_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/entitlements"
)

const (
	userID     = "22222222-2222-2222-2222-222222222222"
	programID  = "44444444-4444-4444-4444-444444444444"
	entID      = "99999999-9999-9999-9999-999999999999"
	programID2 = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	entID2     = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
)

var errRepoFailure = errors.New("repository failure")

// stubRepo is an in-memory fake of the entitlement data-access surface. It
// records the user id and entitlement id forwarded to the repository so tests
// can prove the service forwards the authentication-context identity and never
// invents one.
type stubRepo struct {
	entitlements []models.Entitlement
	createErr    error
	listErr      error
	findErr      error
	deleteErr    error
	gotUser      string
	gotEntID     string
	gotProgramID string
}

func (s *stubRepo) Create(_ context.Context, userID, programID string, entitlement *models.Entitlement) error {
	s.gotUser = userID
	s.gotProgramID = programID
	if s.createErr != nil {
		return s.createErr
	}
	entitlement.ID = entID
	entitlement.UserID = userID
	entitlement.ProgramID = programID
	entitlement.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	entitlement.UpdatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return nil
}

func (s *stubRepo) ListByUser(_ context.Context, userID string) ([]models.Entitlement, error) {
	s.gotUser = userID
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.entitlements, nil
}

func (s *stubRepo) FindByIDAndUser(_ context.Context, userID, entitlementID string) (*models.Entitlement, error) {
	s.gotUser = userID
	s.gotEntID = entitlementID
	if s.findErr != nil {
		return nil, s.findErr
	}
	return nil, nil
}

func (s *stubRepo) SoftDelete(_ context.Context, userID, entitlementID string) error {
	s.gotUser = userID
	s.gotEntID = entitlementID
	return s.deleteErr
}

func newService(repo *stubRepo) entitlements.Service {
	return entitlements.NewService(repo)
}

func validEntitlements() []models.Entitlement {
	return []models.Entitlement{
		{
			ID:        entID,
			UserID:    userID,
			ProgramID: programID,
			Program: models.Program{
				ID:          programID,
				Name:        "Strength Builder",
				Description: "Progressive strength program",
				Type:        models.ProgramTypePremium,
				Status:      models.ProgramStatusPublished,
				CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        entID2,
			UserID:    userID,
			ProgramID: programID2,
			Program: models.Program{
				ID:          programID2,
				Name:        "HIIT Blaster",
				Description: "High intensity interval training",
				Type:        models.ProgramTypeFree,
				Status:      models.ProgramStatusPublished,
				CreatedAt:   time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC),
			},
			CreatedAt: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC),
		},
	}
}

func TestListEntitlementsSuccess(t *testing.T) {
	repo := &stubRepo{entitlements: validEntitlements()}
	svc := newService(repo)

	entitlements, err := svc.ListEntitlements(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListEntitlements: %v", err)
	}
	if repo.gotUser != userID {
		t.Fatalf("expected scope %q, got %q", userID, repo.gotUser)
	}
	if len(entitlements) != 2 {
		t.Fatalf("expected 2 entitlements, got %d", len(entitlements))
	}
	if entitlements[0].ID != entID || entitlements[0].ProgramID != programID {
		t.Fatalf("unexpected entitlement %+v", entitlements[0])
	}
	if entitlements[0].Program.ID != programID || entitlements[0].Program.Name != "Strength Builder" {
		t.Fatalf("unexpected program %+v", entitlements[0].Program)
	}
	if entitlements[1].ID != entID2 || entitlements[1].ProgramID != programID2 {
		t.Fatalf("unexpected entitlement %+v", entitlements[1])
	}
	if entitlements[1].Program.ID != programID2 || entitlements[1].Program.Name != "HIIT Blaster" {
		t.Fatalf("unexpected program %+v", entitlements[1].Program)
	}
}

func TestListEntitlementsEmpty(t *testing.T) {
	repo := &stubRepo{entitlements: []models.Entitlement{}}
	svc := newService(repo)

	entitlements, err := svc.ListEntitlements(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListEntitlements: %v", err)
	}
	if len(entitlements) != 0 {
		t.Fatalf("expected empty list, got %d", len(entitlements))
	}
}

func TestListEntitlementsInvalidInput(t *testing.T) {
	svc := newService(&stubRepo{})

	cases := map[string]string{
		"empty user": "",
		"bad user":   "not-a-uuid",
	}
	for name, id := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.ListEntitlements(context.Background(), id); !errors.Is(err, entitlements.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestListEntitlementsRepoFailure(t *testing.T) {
	svc := newService(&stubRepo{listErr: errRepoFailure})

	_, err := svc.ListEntitlements(context.Background(), userID)
	if err == nil || errors.Is(err, entitlements.ErrInvalidInput) {
		t.Fatalf("expected internal failure to be hidden, got %v", err)
	}
	if !errors.Is(err, errRepoFailure) {
		t.Fatalf("expected the repository failure to stay wrapped, got %v", err)
	}
}

func TestCreateEntitlementSuccess(t *testing.T) {
	repo := &stubRepo{}
	svc := newService(repo)

	ent, err := svc.CreateEntitlement(context.Background(), userID, programID)
	if err != nil {
		t.Fatalf("CreateEntitlement: %v", err)
	}
	if repo.gotUser != userID {
		t.Fatalf("expected scope %q, got %q", userID, repo.gotUser)
	}
	if repo.gotProgramID != programID {
		t.Fatalf("expected program %q, got %q", programID, repo.gotProgramID)
	}
	if ent.ID != entID {
		t.Fatalf("expected entitlement id %q, got %q", entID, ent.ID)
	}
	if ent.ProgramID != programID {
		t.Fatalf("expected program %q, got %q", programID, ent.ProgramID)
	}
}

func TestCreateEntitlementInvalidInput(t *testing.T) {
	svc := newService(&stubRepo{})

	cases := map[string][2]string{
		"empty user":    {"", programID},
		"bad user":      {"not-a-uuid", programID},
		"empty program": {userID, ""},
		"bad program":   {userID, "not-a-uuid"},
		"both empty":    {"", ""},
	}
	for name, pair := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.CreateEntitlement(context.Background(), pair[0], pair[1]); !errors.Is(err, entitlements.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestCreateEntitlementDuplicateCollapses(t *testing.T) {
	svc := newService(&stubRepo{createErr: repositories.ErrEntitlementAlreadyExists})

	_, err := svc.CreateEntitlement(context.Background(), userID, programID)
	if !errors.Is(err, entitlements.ErrEntitlementNotFound) {
		t.Fatalf("expected ErrEntitlementNotFound, got %v", err)
	}
}

func TestCreateEntitlementRepoFailure(t *testing.T) {
	svc := newService(&stubRepo{createErr: errRepoFailure})

	_, err := svc.CreateEntitlement(context.Background(), userID, programID)
	if err == nil || errors.Is(err, entitlements.ErrInvalidInput) || errors.Is(err, entitlements.ErrEntitlementNotFound) {
		t.Fatalf("expected internal failure to be hidden, got %v", err)
	}
	if !errors.Is(err, errRepoFailure) {
		t.Fatalf("expected the repository failure to stay wrapped, got %v", err)
	}
}

func TestRevokeEntitlementSuccess(t *testing.T) {
	repo := &stubRepo{}
	svc := newService(repo)

	if err := svc.RevokeEntitlement(context.Background(), userID, entID); err != nil {
		t.Fatalf("RevokeEntitlement: %v", err)
	}
	if repo.gotUser != userID {
		t.Fatalf("expected scope %q, got %q", userID, repo.gotUser)
	}
	if repo.gotEntID != entID {
		t.Fatalf("expected entitlement %q, got %q", entID, repo.gotEntID)
	}
}

func TestRevokeEntitlementInvalidInput(t *testing.T) {
	svc := newService(&stubRepo{})

	cases := map[string][2]string{
		"empty user":        {"", entID},
		"bad user":          {"not-a-uuid", entID},
		"empty entitlement": {userID, ""},
		"bad entitlement":   {userID, "not-a-uuid"},
		"both empty":        {"", ""},
	}
	for name, pair := range cases {
		t.Run(name, func(t *testing.T) {
			if err := svc.RevokeEntitlement(context.Background(), pair[0], pair[1]); !errors.Is(err, entitlements.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestRevokeEntitlementNotFound(t *testing.T) {
	svc := newService(&stubRepo{deleteErr: repositories.ErrEntitlementNotFound})

	if err := svc.RevokeEntitlement(context.Background(), userID, entID); !errors.Is(err, entitlements.ErrEntitlementNotFound) {
		t.Fatalf("expected ErrEntitlementNotFound, got %v", err)
	}
}

func TestRevokeEntitlementRepoFailure(t *testing.T) {
	svc := newService(&stubRepo{deleteErr: errRepoFailure})

	err := svc.RevokeEntitlement(context.Background(), userID, entID)
	if err == nil || errors.Is(err, entitlements.ErrInvalidInput) || errors.Is(err, entitlements.ErrEntitlementNotFound) {
		t.Fatalf("expected internal failure to be hidden, got %v", err)
	}
	if !errors.Is(err, errRepoFailure) {
		t.Fatalf("expected the repository failure to stay wrapped, got %v", err)
	}
}
