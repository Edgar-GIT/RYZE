package purchases

import (
	"context"
	"errors"
	"fmt"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

var (
	// ErrInvalidInput indicates the purchase input was malformed or incomplete.
	ErrInvalidInput = errors.New("invalid purchase input")
	// ErrProgramNotFound indicates the program does not exist, is soft-deleted
	// or is not published.
	ErrProgramNotFound = errors.New("program not found")
	// ErrProgramNotPurchasable indicates the program is free and cannot be
	// purchased through the commercial transaction path.
	ErrProgramNotPurchasable = errors.New("program is not purchasable")
	// ErrDuplicateEntitlement indicates an active entitlement already exists
	// for this (user, program) pair. The user already owns this program.
	ErrDuplicateEntitlement = errors.New("duplicate entitlement")
	// ErrDuplicatePurchase indicates an active purchase already exists for this
	// (user, program) pair. The purchase must be resolved before creating a new
	// one.
	ErrDuplicatePurchase = errors.New("duplicate purchase")
)

// ProgramRepository is the data-access surface for reading program data.
type ProgramRepository interface {
	FindPublishedByID(ctx context.Context, programID string) (*models.Program, error)
}

// PurchaseRepository is the data-access surface for purchase persistence.
type PurchaseRepository interface {
	Create(ctx context.Context, purchase *models.Purchase) error
	FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.Purchase, error)
}

// EntitlementRepository is the data-access surface for checking existing
// entitlements.
type EntitlementRepository interface {
	FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.Entitlement, error)
}

// CommissionResolver resolves the applicable commission for a trainer.
type CommissionResolver interface {
	ResolveCommission(ctx context.Context, trainerID string) (CommissionResolution, error)
	CalculateCommissionSplit(priceMinorUnits int64, resolution CommissionResolution) CommissionCalculation
}

// CommissionResolution represents the outcome of resolving the applicable
// commission for a given trainer.
type CommissionResolution struct {
	CommissionBPS uint32
	IsOverride    bool
}

// CommissionCalculation is the result of applying a commission resolution to a
// price.
type CommissionCalculation struct {
	PlatformAmount int64
	TrainerAmount  int64
}

// Purchase is the safe representation of a purchase transaction. It carries
// only public commercial metadata and never exposes internal identifiers
// beyond the purchase and program id.
type Purchase struct {
	ID              string
	UserID          string
	ProgramID       string
	PriceMinorUnits int64
	Currency        string
	CommissionBPS   uint32
	PlatformAmount  int64
	TrainerAmount   int64
	Status          string
	CreatedAt       string
	UpdatedAt       string
}

// Service implements the client-facing purchase transaction flow. The
// requesting user identity always comes from the authentication context and is
// never accepted from the client. This service never knows about HTTP, Gin or
// the authentication context.
type Service interface {
	CreatePurchaseIntent(ctx context.Context, userID, programID string) (*Purchase, error)
}

type service struct {
	programs     ProgramRepository
	purchases    PurchaseRepository
	entitlements EntitlementRepository
	commission   CommissionResolver
}

func NewService(
	programs ProgramRepository,
	purchases PurchaseRepository,
	entitlements EntitlementRepository,
	commission CommissionResolver,
) Service {
	return &service{
		programs:     programs,
		purchases:    purchases,
		entitlements: entitlements,
		commission:   commission,
	}
}

// CreatePurchaseIntent validates the program, snapshots the current price,
// resolves the applicable commission, calculates the split, and persists a
// pending purchase record. The purchase status is always "pending"; no payment
// is processed. A future payment provider will transition the purchase to
// "completed" and create the entitlement atomically.
func (s *service) CreatePurchaseIntent(ctx context.Context, userID, programID string) (*Purchase, error) {
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
		return nil, fmt.Errorf("failed to load program: %w", err)
	}

	if program.Type == models.ProgramTypeFree {
		return nil, ErrProgramNotPurchasable
	}

	if _, err := s.entitlements.FindActiveByUserAndProgram(ctx, userID, programID); err == nil {
		return nil, ErrDuplicateEntitlement
	} else if !errors.Is(err, repositories.ErrEntitlementNotFound) {
		return nil, fmt.Errorf("failed to check entitlement: %w", err)
	}

	if _, err := s.purchases.FindActiveByUserAndProgram(ctx, userID, programID); err == nil {
		return nil, ErrDuplicatePurchase
	} else if !errors.Is(err, repositories.ErrPurchaseNotFound) {
		return nil, fmt.Errorf("failed to check purchase: %w", err)
	}

	resolution, err := s.commission.ResolveCommission(ctx, program.TrainerID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve commission: %w", err)
	}

	calculation := s.commission.CalculateCommissionSplit(program.PriceMinorUnits, resolution)

	purchase := &models.Purchase{
		UserID:          userID,
		ProgramID:       programID,
		PriceMinorUnits: program.PriceMinorUnits,
		Currency:        program.Currency,
		CommissionBPS:   resolution.CommissionBPS,
		PlatformAmount:  calculation.PlatformAmount,
		TrainerAmount:   calculation.TrainerAmount,
		Status:          models.PurchaseStatusPending,
	}

	if err := s.purchases.Create(ctx, purchase); err != nil {
		return nil, fmt.Errorf("failed to create purchase: %w", err)
	}

	return newPurchase(purchase), nil
}

func newPurchase(model *models.Purchase) *Purchase {
	return &Purchase{
		ID:              model.ID,
		UserID:          model.UserID,
		ProgramID:       model.ProgramID,
		PriceMinorUnits: model.PriceMinorUnits,
		Currency:        model.Currency,
		CommissionBPS:   model.CommissionBPS,
		PlatformAmount:  model.PlatformAmount,
		TrainerAmount:   model.TrainerAmount,
		Status:          model.Status,
	}
}

func validateUserID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	return nil
}

func validateProgramID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: program id is required", ErrInvalidInput)
	}
	return nil
}
