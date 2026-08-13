package trainer_profile

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

var (
	// ErrInvalidInput indicates the trainer-profile input was malformed or
	// incomplete.
	ErrInvalidInput = errors.New("invalid trainer profile input")
	// ErrTrainerNotFound indicates the active trainer matching the
	// authenticated identity does not exist.
	ErrTrainerNotFound = errors.New("trainer not found")
)

// TrainerRepository is the data-access surface required by the trainer-profile
// service.
type TrainerRepository interface {
	FindByIDAndUserID(ctx context.Context, id, userID string) (*models.Trainer, error)
}

// Service exposes the authenticated trainer's profile. The trainer identity is
// always provided by the trainer context of the request; the service never
// reads or accepts client-supplied identity and never knows about HTTP.
type Service interface {
	GetProfile(ctx context.Context, userID, trainerID string) (*Profile, error)
}

// Profile is the safe trainer profile representation. It only carries public
// trainer and user information and never exposes authentication data (password
// hash, session version), tokens or deletion markers.
type Profile struct {
	TrainerID        string
	UserID           string
	Email            string
	FirstName        string
	LastName         string
	TrainerCreatedAt time.Time
	TrainerUpdatedAt time.Time
	UserCreatedAt    time.Time
	UserUpdatedAt    time.Time
}

type service struct {
	trainers TrainerRepository
}

func NewService(trainers TrainerRepository) Service {
	return &service{trainers: trainers}
}

// GetProfile returns the safe profile of the trainer identified by both the
// trainer id and the owning user id. The repository enforces the ownership in
// the query, so a trainer id belonging to a different user is never returned.
// An active trainer without its linked user is an impossible inconsistency and
// is treated as an internal error without exposing details.
func (s *service) GetProfile(ctx context.Context, userID, trainerID string) (*Profile, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if err := validateTrainerID(trainerID); err != nil {
		return nil, err
	}

	trainer, err := s.trainers.FindByIDAndUserID(ctx, trainerID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrTrainerNotFound) {
			return nil, ErrTrainerNotFound
		}
		return nil, fmt.Errorf("failed to get trainer profile: %w", err)
	}

	if trainer.User.ID == "" || trainer.User.ID != userID {
		return nil, fmt.Errorf("trainer profile inconsistent: linked user is missing")
	}

	return &Profile{
		TrainerID:        trainer.ID,
		UserID:           userID,
		Email:            trainer.User.Email,
		FirstName:        trainer.User.FirstName,
		LastName:         trainer.User.LastName,
		TrainerCreatedAt: trainer.CreatedAt,
		TrainerUpdatedAt: trainer.UpdatedAt,
		UserCreatedAt:    trainer.User.CreatedAt,
		UserUpdatedAt:    trainer.User.UpdatedAt,
	}, nil
}

// validateUserID rejects empty and malformed identifiers before any database
// access.
func validateUserID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid user id", ErrInvalidInput)
	}
	return nil
}

// validateTrainerID rejects empty and malformed identifiers before any database
// access.
func validateTrainerID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: trainer id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid trainer id", ErrInvalidInput)
	}
	return nil
}
