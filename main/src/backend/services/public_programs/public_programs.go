package public_programs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/admin_users"
)

var (
	// ErrInvalidInput indicates the program query was malformed.
	ErrInvalidInput = errors.New("invalid program input")
	// ErrProgramNotFound indicates the program does not exist, is
	// soft-deleted, or is not published.
	ErrProgramNotFound = errors.New("program not found")
)

const (
	// MaxPageSize caps the number of programs returned in a single page.
	MaxPageSize = admin_users.MaxPageSize
)

// ProgramRepository is the read-only data-access surface required by the
// public programs service. The catalog is global: there is no ownership
// scoping on this entity and no write operation is exposed.
type ProgramRepository interface {
	ListPublished(ctx context.Context, page, limit int) ([]models.Program, int64, error)
	FindPublishedByID(ctx context.Context, programID string) (*models.Program, error)
}

// Program is the safe representation of one published program. It carries
// only the public product metadata and never exposes deletion markers,
// draft programs, or any internal data.
type Program struct {
	ID              string
	TrainerID       string
	Name            string
	Description     string
	Type            string
	Status          string
	PriceMinorUnits int64
	Currency        string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ListProgramsResult carries one page of published programs plus the
// pagination metadata needed to render the list.
type ListProgramsResult struct {
	Programs []Program
	Total    int64
	Page     int
	Limit    int
}

// Service exposes the public, read-only program catalog. The catalog is
// global: every published program is visible to every caller, there is no
// ownership scoping, and no write operation is exposed.
type Service interface {
	ListPublishedPrograms(ctx context.Context, page, limit int) (ListProgramsResult, error)
	GetPublishedProgram(ctx context.Context, programID string) (*Program, error)
}

type service struct {
	programs ProgramRepository
}

func NewService(programs ProgramRepository) Service {
	return &service{programs: programs}
}

// ListPublishedPrograms returns one page of published programs ordered by
// creation time (newest first), plus the total count.
func (s *service) ListPublishedPrograms(ctx context.Context, page, limit int) (ListProgramsResult, error) {
	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return ListProgramsResult{}, err
	}

	models, total, err := s.programs.ListPublished(ctx, page, limit)
	if err != nil {
		return ListProgramsResult{}, fmt.Errorf("failed to list published programs: %w", err)
	}

	programs := make([]Program, 0, len(models))
	for i := range models {
		programs = append(programs, *toSafe(&models[i]))
	}
	return ListProgramsResult{Programs: programs, Total: total, Page: page, Limit: limit}, nil
}

// GetPublishedProgram returns one published, non-deleted program. A missing,
// draft or soft-deleted program maps to ErrProgramNotFound; there is no way
// to distinguish between the two.
func (s *service) GetPublishedProgram(ctx context.Context, programID string) (*Program, error) {
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}

	model, err := s.programs.FindPublishedByID(ctx, programID)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrProgramNotFound):
			return nil, ErrProgramNotFound
		default:
			return nil, fmt.Errorf("failed to find published program: %w", err)
		}
	}

	return toSafe(model), nil
}

func toSafe(model *models.Program) *Program {
	return &Program{
		ID:              model.ID,
		TrainerID:       model.TrainerID,
		Name:            model.Name,
		Description:     model.Description,
		Type:            model.Type,
		Status:          model.Status,
		PriceMinorUnits: model.PriceMinorUnits,
		Currency:        model.Currency,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
}

// normalizePagination validates the pagination parameters and clamps oversized
// limits to MaxPageSize.
func normalizePagination(page, limit int) (int, int, error) {
	if page < 1 {
		return 0, 0, fmt.Errorf("%w: page must be at least 1", ErrInvalidInput)
	}
	if limit < 1 {
		return 0, 0, fmt.Errorf("%w: limit must be at least 1", ErrInvalidInput)
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	return page, limit, nil
}

// validateProgramID rejects empty and malformed identifiers before any database
// access.
func validateProgramID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: program id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid program id", ErrInvalidInput)
	}
	return nil
}
