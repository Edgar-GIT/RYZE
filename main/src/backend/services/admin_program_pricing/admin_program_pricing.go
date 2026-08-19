package admin_program_pricing

import (
	"context"
	"errors"
	"fmt"

	"ryze/backend/services/programs"
)

var (
	// ErrProgramNotFound indicates the program does not exist or is
	// soft-deleted.
	ErrProgramNotFound = errors.New("program not found")
)

// Program is the safe program representation returned by the admin pricing
// service. It mirrors the programs service DTO to avoid leaking internal data.
type Program = programs.Program

// UpdatePricingInput carries the pricing fields accepted by the admin pricing
// update operation.
type UpdatePricingInput = programs.UpdatePricingInput

// Service exposes the administrator program pricing operations. Authorization
// (which administrator may operate) is enforced by the route middleware; this
// service never knows about HTTP, Gin or the authentication context.
type Service interface {
	GetProgram(ctx context.Context, programID string) (*Program, error)
	UpdatePricing(ctx context.Context, programID string, input UpdatePricingInput) (*Program, error)
}

type service struct {
	programs programs.Service
}

func NewService(programs programs.Service) Service {
	return &service{programs: programs}
}

// GetProgram returns one active program by its id without ownership scoping.
// A missing or soft-deleted program maps to ErrProgramNotFound.
func (s *service) GetProgram(ctx context.Context, programID string) (*Program, error) {
	program, err := s.programs.GetProgramByID(ctx, programID)
	if err != nil {
		if errors.Is(err, programs.ErrProgramNotFound) {
			return nil, ErrProgramNotFound
		}
		return nil, fmt.Errorf("failed to load program: %w", err)
	}
	return program, nil
}

// UpdatePricing updates the price of any active program. Free programs must
// have a price of 0; paid programs must meet the configured minimum. A missing
// or soft-deleted program maps to ErrProgramNotFound.
func (s *service) UpdatePricing(ctx context.Context, programID string, input UpdatePricingInput) (*Program, error) {
	program, err := s.programs.UpdateProgramPricing(ctx, programID, input)
	if err != nil {
		if errors.Is(err, programs.ErrProgramNotFound) {
			return nil, ErrProgramNotFound
		}
		return nil, fmt.Errorf("failed to update program pricing: %w", err)
	}
	return program, nil
}
