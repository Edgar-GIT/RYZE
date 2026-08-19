package entitlements

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
	// ErrInvalidInput indicates the input was malformed or incomplete.
	ErrInvalidInput = errors.New("invalid entitlements input")
	// ErrEntitlementNotFound indicates the entitlement does not exist, is
	// soft-deleted or does not belong to the authenticated user.
	ErrEntitlementNotFound = errors.New("entitlement not found")
)

// EntitlementRepository is the data-access surface required by the
// entitlements service. Every operation is scoped to the authenticated user id,
// which always comes from the authentication context and is never accepted from
// the client.
type EntitlementRepository interface {
	Create(ctx context.Context, userID, programID string, entitlement *models.Entitlement) error
	ListByUser(ctx context.Context, userID string) ([]models.Entitlement, error)
	FindByIDAndUser(ctx context.Context, userID, entitlementID string) (*models.Entitlement, error)
	SoftDelete(ctx context.Context, userID, entitlementID string) error
}

// Program is the safe program summary exposed inside entitlement metadata. It
// carries only public product metadata and never exposes the owning trainer,
// parent identifiers, deletion markers or any internal data.
type Program struct {
	ID          string
	Name        string
	Description string
	Type        string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Entitlement is the safe representation of a purchase-backed right to access
// a program. It carries only public metadata and never exposes internal
// identifiers beyond the entitlement and program id.
type Entitlement struct {
	ID        string
	ProgramID string
	Program   Program
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Service implements the client-facing entitlement read flow. The requesting
// user identity always comes from the authentication context and is never
// accepted from the client. This service never knows about HTTP, Gin or the
// authentication context.
type Service interface {
	ListEntitlements(ctx context.Context, userID string) ([]Entitlement, error)
	CreateEntitlement(ctx context.Context, userID, programID string) (*Entitlement, error)
	RevokeEntitlement(ctx context.Context, userID, entitlementID string) error
}

type service struct {
	entitlements EntitlementRepository
}

func NewService(entitlements EntitlementRepository) Service {
	return &service{entitlements: entitlements}
}

// ListEntitlements returns the safe entitlement metadata for every active
// entitlement held by the authenticated user. The identity comes exclusively
// from the authentication context.
func (s *service) ListEntitlements(ctx context.Context, userID string) ([]Entitlement, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	rows, err := s.entitlements.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list entitlements: %w", err)
	}
	result := make([]Entitlement, 0, len(rows))
	for i := range rows {
		result = append(result, newEntitlement(&rows[i]))
	}
	return result, nil
}

// CreateEntitlement persists a new entitlement for the given user and program.
// This is currently only used for repository and service tests; no purchase
// creation path exists yet.
func (s *service) CreateEntitlement(ctx context.Context, userID, programID string) (*Entitlement, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}

	entitlement := &models.Entitlement{}
	if err := s.entitlements.Create(ctx, userID, programID, entitlement); err != nil {
		switch {
		case errors.Is(err, repositories.ErrEntitlementAlreadyExists):
			return nil, ErrEntitlementNotFound
		default:
			return nil, fmt.Errorf("failed to create entitlement: %w", err)
		}
	}
	result := newEntitlement(entitlement)
	return &result, nil
}

// RevokeEntitlement soft-deletes one of the user's entitlements. This is
// currently only used for repository and service tests; no purchase creation or
// revocation path exists yet.
func (s *service) RevokeEntitlement(ctx context.Context, userID, entitlementID string) error {
	if err := validateUserID(userID); err != nil {
		return err
	}
	if err := validateEntitlementID(entitlementID); err != nil {
		return err
	}

	if err := s.entitlements.SoftDelete(ctx, userID, entitlementID); err != nil {
		switch {
		case errors.Is(err, repositories.ErrEntitlementNotFound):
			return ErrEntitlementNotFound
		default:
			return fmt.Errorf("failed to revoke entitlement: %w", err)
		}
	}
	return nil
}

func newEntitlement(model *models.Entitlement) Entitlement {
	return Entitlement{
		ID:        model.ID,
		ProgramID: model.ProgramID,
		Program:   newProgram(&model.Program),
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

func newProgram(model *models.Program) Program {
	return Program{
		ID:          model.ID,
		Name:        model.Name,
		Description: model.Description,
		Type:        model.Type,
		Status:      model.Status,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}

func validateUserID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid user id", ErrInvalidInput)
	}
	return nil
}

func validateProgramID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: program id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid program id", ErrInvalidInput)
	}
	return nil
}

func validateEntitlementID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: entitlement id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid entitlement id", ErrInvalidInput)
	}
	return nil
}
