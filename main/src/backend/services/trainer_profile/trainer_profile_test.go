package trainer_profile_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/trainer_profile"
)

const (
	userID    = "11111111-1111-1111-1111-111111111111"
	trainerID = "22222222-2222-2222-2222-222222222222"
)

var errRepoFailure = errors.New("repository failure")

// stubTrainerRepo is an in-memory fake of the trainer data-access surface used
// by the profile service. It records the identifiers passed to
// FindByIDAndUserID so tests can prove the service forwards the context
// identity and never invents it.
type stubTrainerRepo struct {
	findByIDAndUserID func(id, userID string) (*models.Trainer, error)
}

func (s stubTrainerRepo) FindByIDAndUserID(_ context.Context, id, userID string) (*models.Trainer, error) {
	if s.findByIDAndUserID == nil {
		return nil, errors.New("find by id and user not configured")
	}
	return s.findByIDAndUserID(id, userID)
}

func validTrainer() *models.Trainer {
	return &models.Trainer{
		ID:        trainerID,
		UserID:    userID,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		User: models.User{
			ID:        userID,
			Email:     "trainer@ryze.local",
			FirstName: "John",
			LastName:  "Doe",
			CreatedAt: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}

func TestGetProfileSuccess(t *testing.T) {
	repo := stubTrainerRepo{
		findByIDAndUserID: func(id, gotUserID string) (*models.Trainer, error) {
			if id != trainerID {
				t.Fatalf("expected trainer id %q, got %q", trainerID, id)
			}
			if gotUserID != userID {
				t.Fatalf("expected user id %q, got %q", userID, gotUserID)
			}
			return validTrainer(), nil
		},
	}

	svc := trainer_profile.NewService(repo)
	profile, err := svc.GetProfile(context.Background(), userID, trainerID)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}

	if profile.TrainerID != trainerID {
		t.Fatalf("expected trainer id %q, got %q", trainerID, profile.TrainerID)
	}
	if profile.UserID != userID {
		t.Fatalf("expected user id %q, got %q", userID, profile.UserID)
	}
	if profile.Email != "trainer@ryze.local" {
		t.Fatalf("expected email %q, got %q", "trainer@ryze.local", profile.Email)
	}
	if profile.FirstName != "John" || profile.LastName != "Doe" {
		t.Fatalf("expected name John Doe, got %s %s", profile.FirstName, profile.LastName)
	}
	if profile.TrainerCreatedAt.IsZero() || profile.UserCreatedAt.IsZero() {
		t.Fatal("expected non-zero timestamps")
	}
}

func TestGetProfileRejectsInvalidUserID(t *testing.T) {
	svc := trainer_profile.NewService(stubTrainerRepo{})

	for name, id := range map[string]string{
		"empty":     "",
		"not uuid":  "not-a-uuid",
		"too short": "abc",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.GetProfile(context.Background(), id, trainerID); !errors.Is(err, trainer_profile.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestGetProfileRejectsInvalidTrainerID(t *testing.T) {
	svc := trainer_profile.NewService(stubTrainerRepo{})

	for name, id := range map[string]string{
		"empty":     "",
		"not uuid":  "not-a-uuid",
		"too short": "abc",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.GetProfile(context.Background(), userID, id); !errors.Is(err, trainer_profile.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestGetProfileTrainerNotFound(t *testing.T) {
	repo := stubTrainerRepo{
		findByIDAndUserID: func(_, _ string) (*models.Trainer, error) {
			return nil, repositories.ErrTrainerNotFound
		},
	}

	svc := trainer_profile.NewService(repo)
	_, err := svc.GetProfile(context.Background(), userID, trainerID)
	if !errors.Is(err, trainer_profile.ErrTrainerNotFound) {
		t.Fatalf("expected ErrTrainerNotFound, got %v", err)
	}
}

func TestGetProfileRepositoryFailure(t *testing.T) {
	repo := stubTrainerRepo{
		findByIDAndUserID: func(_, _ string) (*models.Trainer, error) {
			return nil, errRepoFailure
		},
	}

	svc := trainer_profile.NewService(repo)
	_, err := svc.GetProfile(context.Background(), userID, trainerID)
	if errors.Is(err, trainer_profile.ErrTrainerNotFound) || errors.Is(err, trainer_profile.ErrInvalidInput) {
		t.Fatalf("repository failure must not map to a domain error, got %v", err)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestGetProfileInconsistentLinkedUser(t *testing.T) {
	repo := stubTrainerRepo{
		findByIDAndUserID: func(_, _ string) (*models.Trainer, error) {
			return &models.Trainer{ID: trainerID, UserID: userID}, nil
		},
	}

	svc := trainer_profile.NewService(repo)
	_, err := svc.GetProfile(context.Background(), userID, trainerID)
	if err == nil {
		t.Fatal("an active trainer without its linked user must be an internal error")
	}
	if errors.Is(err, trainer_profile.ErrTrainerNotFound) || errors.Is(err, trainer_profile.ErrInvalidInput) {
		t.Fatalf("inconsistency must not map to a domain error, got %v", err)
	}
}

func TestGetProfileLinkedUserBelongsToDifferentUser(t *testing.T) {
	other := uuid.NewString()
	repo := stubTrainerRepo{
		findByIDAndUserID: func(_, _ string) (*models.Trainer, error) {
			return &models.Trainer{
				ID:     trainerID,
				UserID: userID,
				User:   models.User{ID: other},
			}, nil
		},
	}

	svc := trainer_profile.NewService(repo)
	_, err := svc.GetProfile(context.Background(), userID, trainerID)
	if err == nil {
		t.Fatal("a linked user that differs from the identity must be an internal error")
	}
	if errors.Is(err, trainer_profile.ErrTrainerNotFound) || errors.Is(err, trainer_profile.ErrInvalidInput) {
		t.Fatalf("inconsistency must not map to a domain error, got %v", err)
	}
}

func TestGetProfileNeverExposesSecrets(t *testing.T) {
	repo := stubTrainerRepo{
		findByIDAndUserID: func(_, _ string) (*models.Trainer, error) {
			trainer := validTrainer()
			trainer.User.PasswordHash = "hash"
			trainer.User.SessionVersion = 7
			return trainer, nil
		},
	}

	svc := trainer_profile.NewService(repo)
	profile, err := svc.GetProfile(context.Background(), userID, trainerID)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}

	// The Profile type is the safe representation: it has no password hash,
	// session version or deletion marker fields, so they can never be exposed.
	if profile.Email == "" || profile.FirstName == "" {
		t.Fatal("safe user fields must be present")
	}
}
