package generic_programs

import (
	"context"
	"errors"
	"fmt"

	"ryze/backend/config"
	"ryze/backend/repositories"
	"ryze/backend/services/exercise_media"
)

// ErrTrainerProgramLocked indicates the trainer plan was delivered through a
// Premium Level 2 request and its content can no longer change.
var ErrTrainerProgramLocked = errors.New("trainer program locked by premium level 2 delivery")

// PremiumLevel2LinkReader reports whether a plan is attached to a Premium
// Level 2 request and whether that request was delivered.
type PremiumLevel2LinkReader interface {
	PremiumLevel2Link(ctx context.Context, programID string) (linked bool, delivered bool, err error)
}

// TrainerStructureService reads and replaces the whole structure of a
// trainer-owned plan with the same rules as the platform builder (structural
// caps, training-day details, duplicate exercises, prescription). The trainer
// id always comes from the authenticated trainer context.
type TrainerStructureService interface {
	GetStructure(ctx context.Context, trainerID, programID string) ([]Week, error)
	ReplaceStructure(ctx context.Context, trainerID, programID string, weeks []WeekInput) ([]Week, error)
}

type trainerStructureService struct {
	repository repositories.TrainerStructureRepository
	links      PremiumLevel2LinkReader
	builder    *service
}

func NewTrainerStructureService(repository repositories.TrainerStructureRepository, links PremiumLevel2LinkReader, media config.ExerciseMediaConfig) TrainerStructureService {
	return &trainerStructureService{
		repository: repository,
		links:      links,
		builder:    &service{media: exercise_media.New(media)},
	}
}

func (s *trainerStructureService) GetStructure(ctx context.Context, trainerID, programID string) ([]Week, error) {
	if err := validateProgramID(trainerID); err != nil {
		return nil, err
	}
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}
	model, err := s.repository.FindTrainerProgramTree(ctx, trainerID, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrProgramNotFound) {
			return nil, ErrProgramNotFound
		}
		return nil, fmt.Errorf("failed to load trainer program: %w", err)
	}
	weeks := make([]Week, 0, len(model.Weeks))
	for i := range model.Weeks {
		weeks = append(weeks, s.builder.newWeek(&model.Weeks[i]))
	}
	return weeks, nil
}

func (s *trainerStructureService) ReplaceStructure(ctx context.Context, trainerID, programID string, weeks []WeekInput) ([]Week, error) {
	if err := validateProgramID(trainerID); err != nil {
		return nil, err
	}
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}
	if err := validateWeeks(weeks); err != nil {
		return nil, err
	}
	if s.links != nil {
		_, delivered, err := s.links.PremiumLevel2Link(ctx, programID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify premium level 2 link: %w", err)
		}
		if delivered {
			return nil, ErrTrainerProgramLocked
		}
	}
	if err := s.repository.ReplaceTrainerStructure(ctx, trainerID, programID, buildWeeks(weeks)); err != nil {
		switch {
		case errors.Is(err, repositories.ErrProgramNotFound):
			return nil, ErrProgramNotFound
		case errors.Is(err, repositories.ErrExerciseNotFound):
			return nil, ErrExerciseNotFound
		default:
			return nil, fmt.Errorf("failed to save trainer program structure: %w", err)
		}
	}
	return s.GetStructure(ctx, trainerID, programID)
}
