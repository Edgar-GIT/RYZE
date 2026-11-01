package free_programs

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

var (
	// ErrInvalidInput indicates the user or program identifier was malformed.
	ErrInvalidInput = errors.New("invalid free program input")
	// ErrProgramNotFound indicates the program is missing, unpublished or not a
	// Free program. All cases are indistinguishable to the client.
	ErrProgramNotFound = errors.New("free program not found")
)

// ProgramReader loads one published program.
type ProgramReader interface {
	FindPublishedByID(ctx context.Context, programID string) (*models.Program, error)
}

// ActivationRepository is the Free activation data-access surface.
type ActivationRepository interface {
	Activate(ctx context.Context, userID, programID string, activation *models.FreeProgramActivation) error
	FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.FreeProgramActivation, error)
	ListActiveByUser(ctx context.Context, userID string) ([]models.FreeProgramActivation, error)
}

// ProgramSummary is the safe public metadata returned with an activation.
type ProgramSummary struct {
	ID               string
	Name             string
	Description      string
	Type             string
	Level            *string
	DurationWeeks    *int
	FrequencyPerWeek *int
	TrainingType     *string
	PriceMinorUnits  int64
	Currency         string
}

// Activation is the client-safe view of a Free program activation.
type Activation struct {
	ID          string
	ActivatedAt string
	ExpiresAt   *string
	Program     ProgramSummary
}

// Service implements the authenticated Free program activation flow.
type Service interface {
	Activate(ctx context.Context, userID, programID string) (*Activation, error)
	ListActive(ctx context.Context, userID string) ([]Activation, error)
}

type service struct {
	programs    ProgramReader
	activations ActivationRepository
}

func NewService(programs ProgramReader, activations ActivationRepository) Service {
	return &service{programs: programs, activations: activations}
}

// Activate grants the authenticated user access to a published Free program.
// The user identity always comes from the authentication context. Activation is
// idempotent: activating the already-active Free program returns it unchanged.
// Activating a different Free program replaces the previous activation.
func (s *service) Activate(ctx context.Context, userID, programID string) (*Activation, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}

	program, err := s.programs.FindPublishedByID(ctx, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrProgramNotFound) {
			return nil, ErrProgramNotFound
		}
		return nil, fmt.Errorf("failed to load free program: %w", err)
	}
	if program.Type != models.ProgramTypeFree || program.ProductType != models.ProgramProductTypeGeneric {
		return nil, ErrProgramNotFound
	}

	activation := &models.FreeProgramActivation{}
	if err := s.activations.Activate(ctx, userID, programID, activation); err != nil {
		if errors.Is(err, repositories.ErrFreeActivationAlreadyActive) {
			existing, findErr := s.activations.FindActiveByUserAndProgram(ctx, userID, programID)
			if findErr != nil {
				return nil, fmt.Errorf("failed to load active free program: %w", findErr)
			}
			return newActivation(existing, program), nil
		}
		return nil, fmt.Errorf("failed to activate free program: %w", err)
	}
	return newActivation(activation, program), nil
}

// ListActive returns the user's active Free activation, if any.
func (s *service) ListActive(ctx context.Context, userID string) ([]Activation, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	rows, err := s.activations.ListActiveByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list free activations: %w", err)
	}

	activations := make([]Activation, 0, len(rows))
	for i := range rows {
		activations = append(activations, *newActivation(&rows[i], &rows[i].Program))
	}
	return activations, nil
}

func newActivation(model *models.FreeProgramActivation, program *models.Program) *Activation {
	activation := &Activation{
		ID:          model.ID,
		ActivatedAt: model.ActivatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"),
		Program:     newProgramSummary(program),
	}
	if model.ExpiresAt != nil {
		value := model.ExpiresAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00")
		activation.ExpiresAt = &value
	}
	return activation
}

func newProgramSummary(program *models.Program) ProgramSummary {
	if program == nil {
		return ProgramSummary{}
	}
	return ProgramSummary{
		ID:               program.ID,
		Name:             program.Name,
		Description:      program.Description,
		Type:             program.Type,
		Level:            program.Level,
		DurationWeeks:    program.DurationWeeks,
		FrequencyPerWeek: program.FrequencyPerWeek,
		TrainingType:     program.TrainingType,
		PriceMinorUnits:  program.PriceMinorUnits,
		Currency:         program.Currency,
	}
}

func validateUserID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid user id", ErrInvalidInput)
	}
	return nil
}

func validateProgramID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: program id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid program id", ErrInvalidInput)
	}
	return nil
}
