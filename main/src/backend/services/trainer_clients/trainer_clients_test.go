package trainer_clients_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/trainer_clients"
)

const (
	trainerID = "11111111-1111-1111-1111-111111111111"
	userID    = "22222222-2222-2222-2222-222222222222"
)

var errRepoFailure = errors.New("repository failure")

// stubClientRepo is an in-memory fake of the relationship data-access surface.
// It records the identifiers passed to every operation so tests can prove the
// service forwards the trainer context identity and never invents or accepts a
// client-supplied one.
type stubClientRepo struct {
	create              func(relation *models.TrainerClient) error
	findActive          func(trainerID, userID string) (*models.TrainerClient, error)
	listActive          func(trainerID string, page, limit int) ([]models.TrainerClient, int64, error)
	softDelete          func(trainerID, userID string) error
	reactivate          func(trainerID, userID string) error
	createGotTrainerID  string
	createGotUserID     string
	softDeleteTrainerID string
	softDeleteUserID    string
	reactivateTrainerID string
	reactivateUserID    string
}

func (s *stubClientRepo) Create(_ context.Context, relation *models.TrainerClient) error {
	s.createGotTrainerID = relation.TrainerID
	s.createGotUserID = relation.UserID
	if s.create == nil {
		return nil
	}
	return s.create(relation)
}

func (s *stubClientRepo) FindActiveByTrainerAndUser(_ context.Context, trainerID, userID string) (*models.TrainerClient, error) {
	if s.findActive == nil {
		return validRelation(), nil
	}
	return s.findActive(trainerID, userID)
}

func (s *stubClientRepo) FindIncludingDeletedByTrainerAndUser(_ context.Context, _, _ string) (*models.TrainerClient, error) {
	return validRelation(), nil
}

func (s *stubClientRepo) ListActiveClients(_ context.Context, trainerID string, page, limit int) ([]models.TrainerClient, int64, error) {
	if s.listActive == nil {
		return []models.TrainerClient{*validRelation()}, 1, nil
	}
	return s.listActive(trainerID, page, limit)
}

func (s *stubClientRepo) SoftDelete(_ context.Context, trainerID, userID string) error {
	s.softDeleteTrainerID = trainerID
	s.softDeleteUserID = userID
	if s.softDelete == nil {
		return nil
	}
	return s.softDelete(trainerID, userID)
}

func (s *stubClientRepo) Reactivate(_ context.Context, trainerID, userID string) error {
	s.reactivateTrainerID = trainerID
	s.reactivateUserID = userID
	if s.reactivate == nil {
		return nil
	}
	return s.reactivate(trainerID, userID)
}

type stubUserRepo struct {
	findByID func(userID string) (*models.User, error)
}

func (s stubUserRepo) FindByID(_ context.Context, userID string) (*models.User, error) {
	if s.findByID == nil {
		return &models.User{ID: userID}, nil
	}
	return s.findByID(userID)
}

func validRelation() *models.TrainerClient {
	return &models.TrainerClient{
		ID:        "33333333-3333-3333-3333-333333333333",
		TrainerID: trainerID,
		UserID:    userID,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		User: models.User{
			ID:        userID,
			Email:     "client@ryze.local",
			FirstName: "Jane",
			LastName:  "Roe",
			CreatedAt: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}

func newService(repo *stubClientRepo) trainer_clients.Service {
	return trainer_clients.NewService(repo, stubUserRepo{})
}

func TestAddClientSuccess(t *testing.T) {
	repo := &stubClientRepo{}
	svc := newService(repo)

	client, err := svc.AddClient(context.Background(), trainerID, userID)
	if err != nil {
		t.Fatalf("AddClient: %v", err)
	}
	if repo.createGotTrainerID != trainerID {
		t.Fatalf("expected trainer id %q, got %q", trainerID, repo.createGotTrainerID)
	}
	if repo.createGotUserID != userID {
		t.Fatalf("expected user id %q, got %q", userID, repo.createGotUserID)
	}
	if client.UserID != userID || client.Email != "client@ryze.local" {
		t.Fatalf("unexpected client %+v", client)
	}
	if client.RelationCreatedAt.IsZero() || client.UserCreatedAt.IsZero() {
		t.Fatal("expected non-zero timestamps")
	}
}

func TestAddClientRejectsInvalidIDs(t *testing.T) {
	svc := newService(&stubClientRepo{})

	for name, ids := range map[string][2]string{
		"empty trainer":  {"", userID},
		"bad trainer":    {"not-a-uuid", userID},
		"empty user":     {trainerID, ""},
		"bad user":       {trainerID, "not-a-uuid"},
		"self relation":  {trainerID, trainerID},
		"same uuid pair": {userID, userID},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.AddClient(context.Background(), ids[0], ids[1]); !errors.Is(err, trainer_clients.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestAddClientClientNotFound(t *testing.T) {
	repo := &stubClientRepo{}
	svc := trainer_clients.NewService(repo, stubUserRepo{
		findByID: func(_ string) (*models.User, error) {
			return nil, repositories.ErrUserNotFound
		},
	})

	_, err := svc.AddClient(context.Background(), trainerID, userID)
	if !errors.Is(err, trainer_clients.ErrClientNotFound) {
		t.Fatalf("expected ErrClientNotFound, got %v", err)
	}
	if repo.createGotTrainerID != "" {
		t.Fatalf("no relationship may be created for a missing client, got trainer id %q", repo.createGotTrainerID)
	}
}

func TestAddClientClientAlreadyActive(t *testing.T) {
	repo := &stubClientRepo{
		create: func(_ *models.TrainerClient) error {
			return repositories.ErrClientRelationAlreadyActive
		},
	}
	svc := newService(repo)

	_, err := svc.AddClient(context.Background(), trainerID, userID)
	if !errors.Is(err, trainer_clients.ErrClientAlreadyActive) {
		t.Fatalf("expected ErrClientAlreadyActive, got %v", err)
	}
}

func TestAddClientRepositoryFailure(t *testing.T) {
	repo := &stubClientRepo{
		create: func(_ *models.TrainerClient) error {
			return errRepoFailure
		},
	}
	svc := newService(repo)

	_, err := svc.AddClient(context.Background(), trainerID, userID)
	if errors.Is(err, trainer_clients.ErrInvalidInput) ||
		errors.Is(err, trainer_clients.ErrClientNotFound) ||
		errors.Is(err, trainer_clients.ErrClientAlreadyActive) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestAddClientUserCheckRepositoryFailure(t *testing.T) {
	svc := trainer_clients.NewService(&stubClientRepo{}, stubUserRepo{
		findByID: func(_ string) (*models.User, error) {
			return nil, errRepoFailure
		},
	})

	_, err := svc.AddClient(context.Background(), trainerID, userID)
	if errors.Is(err, trainer_clients.ErrClientNotFound) || errors.Is(err, trainer_clients.ErrInvalidInput) {
		t.Fatalf("user repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestListClientsSuccess(t *testing.T) {
	var gotTrainerID string
	repo := &stubClientRepo{
		listActive: func(trainerID string, page, limit int) ([]models.TrainerClient, int64, error) {
			gotTrainerID = trainerID
			if page != 2 || limit != 10 {
				t.Fatalf("expected page 2 limit 10, got %d/%d", page, limit)
			}
			return []models.TrainerClient{*validRelation()}, 1, nil
		},
	}
	svc := newService(repo)

	result, err := svc.ListClients(context.Background(), trainerID, 2, 10)
	if err != nil {
		t.Fatalf("ListClients: %v", err)
	}
	if gotTrainerID != trainerID {
		t.Fatalf("expected trainer id %q, got %q", trainerID, gotTrainerID)
	}
	if result.Total != 1 || len(result.Clients) != 1 {
		t.Fatalf("expected one client, got %+v", result)
	}
	if result.Page != 2 || result.Limit != 10 {
		t.Fatalf("expected page 2 limit 10, got %d/%d", result.Page, result.Limit)
	}
}

func TestListClientsClampsLimit(t *testing.T) {
	var gotLimit int
	repo := &stubClientRepo{
		listActive: func(_ string, _, limit int) ([]models.TrainerClient, int64, error) {
			gotLimit = limit
			return nil, 0, nil
		},
	}
	svc := newService(repo)

	if _, err := svc.ListClients(context.Background(), trainerID, 1, 99999); err != nil {
		t.Fatalf("ListClients: %v", err)
	}
	if gotLimit != trainer_clients.MaxPageSize {
		t.Fatalf("expected limit clamped to %d, got %d", trainer_clients.MaxPageSize, gotLimit)
	}
}

func TestListClientsRejectsInvalidInput(t *testing.T) {
	svc := newService(&stubClientRepo{})

	for name, args := range map[string][3]int{
		"empty trainer id": {0, 1, 10},
	} {
		t.Run(name, func(t *testing.T) {
			// The trainer id is validated before pagination, so any malformed
			// identity is rejected first.
			if _, err := svc.ListClients(context.Background(), "", args[1], args[2]); !errors.Is(err, trainer_clients.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}

	for name, args := range map[string][2]int{
		"page zero":      {0, 10},
		"page negative":  {-1, 10},
		"limit zero":     {1, 0},
		"limit negative": {1, -5},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.ListClients(context.Background(), trainerID, args[0], args[1]); !errors.Is(err, trainer_clients.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestListClientsRepositoryFailure(t *testing.T) {
	repo := &stubClientRepo{
		listActive: func(_ string, _, _ int) ([]models.TrainerClient, int64, error) {
			return nil, 0, errRepoFailure
		},
	}
	svc := newService(repo)

	_, err := svc.ListClients(context.Background(), trainerID, 1, 10)
	if errors.Is(err, trainer_clients.ErrInvalidInput) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestGetClientSuccess(t *testing.T) {
	var gotTrainerID, gotUserID string
	repo := &stubClientRepo{
		findActive: func(trainerID, userID string) (*models.TrainerClient, error) {
			gotTrainerID = trainerID
			gotUserID = userID
			return validRelation(), nil
		},
	}
	svc := newService(repo)

	client, err := svc.GetClient(context.Background(), trainerID, userID)
	if err != nil {
		t.Fatalf("GetClient: %v", err)
	}
	if gotTrainerID != trainerID || gotUserID != userID {
		t.Fatalf("expected identifiers %q/%q, got %q/%q", trainerID, userID, gotTrainerID, gotUserID)
	}
	if client.UserID != userID || client.Email != "client@ryze.local" {
		t.Fatalf("unexpected client %+v", client)
	}
}

func TestGetClientRejectsInvalidIDs(t *testing.T) {
	svc := newService(&stubClientRepo{})

	for name, ids := range map[string][2]string{
		"empty trainer": {"", userID},
		"bad trainer":   {"not-a-uuid", userID},
		"empty user":    {trainerID, ""},
		"bad user":      {trainerID, "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.GetClient(context.Background(), ids[0], ids[1]); !errors.Is(err, trainer_clients.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestGetClientRelationNotFound(t *testing.T) {
	repo := &stubClientRepo{
		findActive: func(_, _ string) (*models.TrainerClient, error) {
			return nil, repositories.ErrClientRelationNotFound
		},
	}
	svc := newService(repo)

	if _, err := svc.GetClient(context.Background(), trainerID, userID); !errors.Is(err, trainer_clients.ErrClientRelationNotFound) {
		t.Fatalf("expected ErrClientRelationNotFound, got %v", err)
	}
}

func TestGetClientSoftDeletedUserHidden(t *testing.T) {
	// A soft-deleted linked user leaves the relationship present but the
	// preloaded user empty: the profile read must never reveal whether the
	// user exists, so it maps to the same not-found error.
	relation := validRelation()
	relation.User = models.User{}
	repo := &stubClientRepo{
		findActive: func(_, _ string) (*models.TrainerClient, error) {
			return relation, nil
		},
	}
	svc := newService(repo)

	if _, err := svc.GetClient(context.Background(), trainerID, userID); !errors.Is(err, trainer_clients.ErrClientRelationNotFound) {
		t.Fatalf("expected ErrClientRelationNotFound, got %v", err)
	}
}

func TestGetClientRepositoryFailure(t *testing.T) {
	repo := &stubClientRepo{
		findActive: func(_, _ string) (*models.TrainerClient, error) {
			return nil, errRepoFailure
		},
	}
	svc := newService(repo)

	_, err := svc.GetClient(context.Background(), trainerID, userID)
	if errors.Is(err, trainer_clients.ErrClientRelationNotFound) || errors.Is(err, trainer_clients.ErrInvalidInput) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestRemoveClientSuccess(t *testing.T) {
	repo := &stubClientRepo{}
	svc := newService(repo)

	if err := svc.RemoveClient(context.Background(), trainerID, userID); err != nil {
		t.Fatalf("RemoveClient: %v", err)
	}
	if repo.softDeleteTrainerID != trainerID || repo.softDeleteUserID != userID {
		t.Fatalf("expected soft delete on %q/%q, got %q/%q", trainerID, userID, repo.softDeleteTrainerID, repo.softDeleteUserID)
	}
}

func TestRemoveClientRejectsInvalidIDs(t *testing.T) {
	svc := newService(&stubClientRepo{})

	for name, ids := range map[string][2]string{
		"empty trainer": {"", userID},
		"bad user":      {trainerID, "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := svc.RemoveClient(context.Background(), ids[0], ids[1]); !errors.Is(err, trainer_clients.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestRemoveClientRelationNotFound(t *testing.T) {
	repo := &stubClientRepo{
		softDelete: func(_, _ string) error {
			return repositories.ErrClientRelationNotFound
		},
	}
	svc := newService(repo)

	if err := svc.RemoveClient(context.Background(), trainerID, userID); !errors.Is(err, trainer_clients.ErrClientRelationNotFound) {
		t.Fatalf("expected ErrClientRelationNotFound, got %v", err)
	}
}

func TestReactivateClientSuccess(t *testing.T) {
	repo := &stubClientRepo{}
	svc := newService(repo)

	client, err := svc.ReactivateClient(context.Background(), trainerID, userID)
	if err != nil {
		t.Fatalf("ReactivateClient: %v", err)
	}
	if repo.reactivateTrainerID != trainerID || repo.reactivateUserID != userID {
		t.Fatalf("expected reactivation on %q/%q, got %q/%q", trainerID, userID, repo.reactivateTrainerID, repo.reactivateUserID)
	}
	if client.UserID != userID {
		t.Fatalf("unexpected client %+v", client)
	}
}

func TestReactivateClientClientNotFound(t *testing.T) {
	svc := trainer_clients.NewService(&stubClientRepo{}, stubUserRepo{
		findByID: func(_ string) (*models.User, error) {
			return nil, repositories.ErrUserNotFound
		},
	})

	_, err := svc.ReactivateClient(context.Background(), trainerID, userID)
	if !errors.Is(err, trainer_clients.ErrClientNotFound) {
		t.Fatalf("expected ErrClientNotFound, got %v", err)
	}
}

func TestReactivateClientRelationNotFound(t *testing.T) {
	repo := &stubClientRepo{
		reactivate: func(_, _ string) error {
			return repositories.ErrClientRelationNotFound
		},
	}
	svc := newService(repo)

	_, err := svc.ReactivateClient(context.Background(), trainerID, userID)
	if !errors.Is(err, trainer_clients.ErrClientRelationNotFound) {
		t.Fatalf("expected ErrClientRelationNotFound, got %v", err)
	}
}

func TestReactivateClientRejectsInvalidIDs(t *testing.T) {
	svc := newService(&stubClientRepo{})

	for name, ids := range map[string][2]string{
		"empty trainer": {"", userID},
		"bad user":      {trainerID, "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.ReactivateClient(context.Background(), ids[0], ids[1]); !errors.Is(err, trainer_clients.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestClientNeverExposesSecrets(t *testing.T) {
	relation := validRelation()
	relation.User.PasswordHash = "hash"
	relation.User.SessionVersion = 7
	relation.User.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}

	repo := &stubClientRepo{
		findActive: func(_, _ string) (*models.TrainerClient, error) {
			return relation, nil
		},
	}
	svc := newService(repo)

	client, err := svc.AddClient(context.Background(), trainerID, userID)
	if err != nil {
		t.Fatalf("AddClient: %v", err)
	}
	// Client is the safe representation: it has no password hash, session
	// version or deletion marker fields, so they can never be exposed.
	if client.Email == "" || client.FirstName == "" {
		t.Fatal("safe user fields must be present")
	}
}
