package program_access

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"ryze/backend/config"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/program_content"
)

var (
	// ErrInvalidInput indicates the user or program identifier was malformed.
	ErrInvalidInput = errors.New("invalid program access input")
	// ErrProgramNotAccessible indicates the authenticated user does not hold a
	// valid grant for the requested program, or the program itself does not
	// exist, is soft-deleted or is not published. Both situations are
	// indistinguishable so access is never revealed.
	ErrProgramNotAccessible = errors.New("program not accessible")
)

// GrantSource identifies why a client may access a program. It is internal to
// the backend and never required from the client.
type GrantSource string

const (
	GrantSourceEntitlement    GrantSource = "entitlement"
	GrantSourceAssignment     GrantSource = "assignment"
	GrantSourceFreeActivation GrantSource = "free_activation"
)

// EntitlementReader verifies a purchased grant.
type EntitlementReader interface {
	FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.Entitlement, error)
}

// AssignmentReader verifies a trainer-assigned grant.
type AssignmentReader interface {
	ExistsByUserAndProgram(ctx context.Context, userID, programID string) (bool, error)
}

// FreeActivationReader verifies a client-activated Free grant.
type FreeActivationReader interface {
	FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.FreeProgramActivation, error)
}

// ProgramReader loads the published program with its complete structure.
type ProgramReader interface {
	FindPublishedByIDWithStructure(ctx context.Context, programID string) (*models.Program, error)
}

// Service exposes the authenticated client's programme access read and the
// single grant rule used by every protected programme surface.
type Service interface {
	GetProgramAccess(ctx context.Context, userID, programID string) (*program_content.Program, error)
	ResolveGrant(ctx context.Context, userID, programID string) (GrantSource, error)
}

type service struct {
	entitlements EntitlementReader
	assignments  AssignmentReader
	activations  FreeActivationReader
	programs     ProgramReader
	content      *program_content.Builder
}

// NewService wires the three grant sources, the published-program reader and
// the shared safe content mapper.
func NewService(
	entitlements EntitlementReader,
	assignments AssignmentReader,
	activations FreeActivationReader,
	programs ProgramReader,
	media config.ExerciseMediaConfig,
) Service {
	return &service{
		entitlements: entitlements,
		assignments:  assignments,
		activations:  activations,
		programs:     programs,
		content:      program_content.NewBuilder(media),
	}
}

// ResolveGrant returns the grant that allows the user to access the program.
// A missing grant and an internal source error that represents absence both
// map to ErrProgramNotAccessible.
func (s *service) ResolveGrant(ctx context.Context, userID, programID string) (GrantSource, error) {
	if err := validateUserID(userID); err != nil {
		return "", err
	}
	if err := validateProgramID(programID); err != nil {
		return "", err
	}

	if _, err := s.entitlements.FindActiveByUserAndProgram(ctx, userID, programID); err == nil {
		return GrantSourceEntitlement, nil
	} else if !errors.Is(err, repositories.ErrEntitlementNotFound) {
		return "", fmt.Errorf("failed to verify purchased access: %w", err)
	}

	assigned, err := s.assignments.ExistsByUserAndProgram(ctx, userID, programID)
	if err != nil {
		return "", fmt.Errorf("failed to verify assigned access: %w", err)
	}
	if assigned {
		return GrantSourceAssignment, nil
	}

	if _, err := s.activations.FindActiveByUserAndProgram(ctx, userID, programID); err == nil {
		return GrantSourceFreeActivation, nil
	} else if !errors.Is(err, repositories.ErrFreeActivationNotFound) {
		return "", fmt.Errorf("failed to verify free access: %w", err)
	}

	return "", ErrProgramNotAccessible
}

// GetProgramAccess returns the safe structure of one accessible, published
// program. The grant source decides which content is available: a Free
// activation can only ever open a Free generic program.
func (s *service) GetProgramAccess(ctx context.Context, userID, programID string) (*program_content.Program, error) {
	source, err := s.ResolveGrant(ctx, userID, programID)
	if err != nil {
		return nil, err
	}

	program, err := s.programs.FindPublishedByIDWithStructure(ctx, programID)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrProgramNotFound):
			return nil, ErrProgramNotAccessible
		default:
			return nil, fmt.Errorf("failed to load accessible program: %w", err)
		}
	}
	if source == GrantSourceFreeActivation &&
		(program.Type != models.ProgramTypeFree || program.ProductType != models.ProgramProductTypeGeneric) {
		return nil, ErrProgramNotAccessible
	}

	return s.content.Build(program), nil
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
