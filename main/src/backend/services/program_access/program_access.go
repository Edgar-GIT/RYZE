package program_access

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/public_programs"
)

var (
	// ErrInvalidInput indicates the user or program identifier was malformed.
	ErrInvalidInput = errors.New("invalid program access input")
	// ErrProgramNotAccessible indicates the authenticated user does not hold
	// an active entitlement for the requested program, or the program itself
	// does not exist, is soft-deleted or is not published. Both situations
	// are indistinguishable so access is never revealed.
	ErrProgramNotAccessible = errors.New("program not accessible")
)

// EntitlementReader verifies that the authenticated user holds an active
// entitlement for the requested program. The user identity always comes from
// the authentication context and is never accepted from the client.
type EntitlementReader interface {
	FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.Entitlement, error)
}

// ProgramReader loads the published program with its complete client-safe
// structure, reusing the existing public catalog read surface.
type ProgramReader interface {
	GetPublishedProgram(ctx context.Context, programID string) (*public_programs.ProgramDetail, error)
}

// Service exposes the authenticated client's entitlement-backed program
// access read. Access is granted only when both conditions hold: the
// authenticated user holds an active entitlement for the program and the
// program itself is still published (matches the marketplace contract).
// A missing entitlement, a missing program, a soft-deleted program and an
// unpublished program are all indistinguishable. The requesting user identity
// always comes from the authentication context; the service never accepts a
// client-supplied identity.
type Service interface {
	GetProgramAccess(ctx context.Context, userID, programID string) (*public_programs.ProgramDetail, error)
}

type service struct {
	entitlements EntitlementReader
	programs     ProgramReader
}

// NewService wires the entitlement gate and the public program reader. Both
// dependencies already exist in the codebase and are reused as-is.
func NewService(entitlements EntitlementReader, programs ProgramReader) Service {
	return &service{entitlements: entitlements, programs: programs}
}

// GetProgramAccess returns the client-safe structure of one entitled,
// published program, or ErrProgramNotAccessible when the user has no active
// entitlement or the program is unavailable.
func (s *service) GetProgramAccess(ctx context.Context, userID, programID string) (*public_programs.ProgramDetail, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}

	if _, err := s.entitlements.FindActiveByUserAndProgram(ctx, userID, programID); err != nil {
		switch {
		case errors.Is(err, repositories.ErrEntitlementNotFound):
			return nil, ErrProgramNotAccessible
		default:
			return nil, fmt.Errorf("failed to verify program access: %w", err)
		}
	}

	detail, err := s.programs.GetPublishedProgram(ctx, programID)
	if err != nil {
		switch {
		case errors.Is(err, public_programs.ErrProgramNotFound):
			return nil, ErrProgramNotAccessible
		default:
			return nil, fmt.Errorf("failed to load entitled program: %w", err)
		}
	}

	return detail, nil
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
