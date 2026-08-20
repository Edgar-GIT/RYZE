package purchases

import (
	"context"
	"errors"
	"fmt"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/payments"
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
	// ErrPurchaseNotFound indicates the purchase does not exist, is
	// soft-deleted or is not in a completable state.
	ErrPurchaseNotFound = errors.New("purchase not found")
	// ErrPurchaseNotCompleted indicates the purchase is in an unexpected
	// terminal state (failed, refunded) that prevents completion.
	ErrPurchaseNotCompleted = errors.New("purchase is not in a completable state")
	// ErrEntitlementIntegrity indicates the purchase is already completed but
	// the corresponding entitlement is missing — a data consistency violation
	// that requires investigation.
	ErrEntitlementIntegrity = errors.New("entitlement integrity error")
	// ErrPurchaseNotPending indicates the purchase cannot accept a payment
	// initiation because it is not in the pending state.
	ErrPurchaseNotPending = errors.New("purchase is not pending")
	// ErrPaymentProvider indicates the payment provider could not initiate the
	// payment. Internal provider details are never exposed to the client.
	ErrPaymentProvider = errors.New("payment provider error")
)

// ProgramRepository is the data-access surface for reading program data.
type ProgramRepository interface {
	FindPublishedByID(ctx context.Context, programID string) (*models.Program, error)
}

// PurchaseRepository is the data-access surface for purchase persistence.
type PurchaseRepository interface {
	Create(ctx context.Context, purchase *models.Purchase) error
	FindByID(ctx context.Context, purchaseID string) (*models.Purchase, error)
	FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.Purchase, error)
	Complete(ctx context.Context, purchaseID string) error
	CompleteWithEntitlement(ctx context.Context, purchaseID string, entitlement *models.Entitlement) error
}

// EntitlementRepository is the data-access surface for checking existing
// entitlements.
type EntitlementRepository interface {
	Create(ctx context.Context, userID, programID string, entitlement *models.Entitlement) error
	FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.Entitlement, error)
	RestoreByUserAndProgram(ctx context.Context, userID, programID string) error
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

// PaymentResult is the safe representation of a payment initiation result.
// It carries only public payment metadata and never exposes provider secrets.
type PaymentResult struct {
	PaymentID   string
	Status      payments.PaymentStatus
	CheckoutURL string
	Provider    string
	PurchaseID  string
}

// Service implements the client-facing purchase transaction flow. The
// requesting user identity always comes from the authentication context and is
// never accepted from the client. This service never knows about HTTP, Gin or
// the authentication context.
type Service interface {
	CreatePurchaseIntent(ctx context.Context, userID, programID string) (*Purchase, error)
	InitiatePayment(ctx context.Context, userID, purchaseID string) (*PaymentResult, error)
	CompletePurchase(ctx context.Context, purchaseID string) (*Purchase, error)
}

type service struct {
	programs     ProgramRepository
	purchases    PurchaseRepository
	entitlements EntitlementRepository
	commission   CommissionResolver
	payment      payments.Provider
}

func NewService(
	programs ProgramRepository,
	purchases PurchaseRepository,
	entitlements EntitlementRepository,
	commission CommissionResolver,
	payment payments.Provider,
) Service {
	return &service{
		programs:     programs,
		purchases:    purchases,
		entitlements: entitlements,
		commission:   commission,
		payment:      payment,
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

// InitiatePayment requests a provider payment for an existing pending purchase.
// The purchase must belong to the authenticated user and be in "pending" status.
// The immutable purchase snapshot is used to construct the provider request; no
// client-supplied commercial values are accepted. The purchase status is NOT
// modified during initiation — it remains "pending" until a verified provider
// event flows through CompletePurchase().
func (s *service) InitiatePayment(ctx context.Context, userID, purchaseID string) (*PaymentResult, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if err := validatePurchaseID(purchaseID); err != nil {
		return nil, err
	}

	purchase, err := s.purchases.FindByID(ctx, purchaseID)
	if err != nil {
		if errors.Is(err, repositories.ErrPurchaseNotFound) {
			return nil, ErrPurchaseNotFound
		}
		return nil, fmt.Errorf("failed to load purchase: %w", err)
	}

	if purchase.UserID != userID {
		return nil, ErrPurchaseNotFound
	}

	if purchase.Status != models.PurchaseStatusPending {
		return nil, ErrPurchaseNotPending
	}

	request := payments.PaymentRequest{
		PurchaseID:       purchase.ID,
		AmountMinorUnits: purchase.PriceMinorUnits,
		Currency:         purchase.Currency,
		ProgramID:        purchase.ProgramID,
	}

	result, err := s.payment.InitiatePayment(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPaymentProvider, err)
	}

	return &PaymentResult{
		PaymentID:   result.PaymentID,
		Status:      result.Status,
		CheckoutURL: result.CheckoutURL,
		Provider:    result.Provider,
		PurchaseID:  result.PurchaseID,
	}, nil
}

// CompletePurchase finalises a purchase and provisions the entitlement atomically.
// It is intended to be called by an internal payment-provider adapter once
// external payment confirmation is received. No user identity is accepted: the
// operation is scoped purely by the purchase id. The method is idempotent when
// called with the same purchase id — a second call on an already-completed
// purchase returns a success when the entitlement is present, and an integrity
// error when it is not.
func (s *service) CompletePurchase(ctx context.Context, purchaseID string) (*Purchase, error) {
	if err := validatePurchaseID(purchaseID); err != nil {
		return nil, err
	}

	purchase, err := s.purchases.FindByID(ctx, purchaseID)
	if err != nil {
		if errors.Is(err, repositories.ErrPurchaseNotFound) {
			return nil, ErrPurchaseNotFound
		}
		return nil, fmt.Errorf("failed to load purchase: %w", err)
	}

	if purchase.Status == models.PurchaseStatusCompleted {
		existing, entErr := s.entitlements.FindActiveByUserAndProgram(ctx, purchase.UserID, purchase.ProgramID)
		if entErr != nil && !errors.Is(entErr, repositories.ErrEntitlementNotFound) {
			return nil, fmt.Errorf("failed to check entitlement: %w", entErr)
		}
		if existing != nil {
			return newPurchase(purchase), nil
		}
		return nil, ErrEntitlementIntegrity
	}

	if purchase.Status != models.PurchaseStatusPending {
		return nil, ErrPurchaseNotCompleted
	}

	if err := s.completeAndCreateEntitlement(ctx, purchase); err != nil {
		return nil, err
	}

	purchase.Status = models.PurchaseStatusCompleted
	return newPurchase(purchase), nil
}

// completeAndCreateEntitlement provisions the entitlement for a purchase that
// is currently pending. Two distinct paths exist, both guarded by the
// repository layer:
//
//  1. A previously soft-deleted entitlement exists for the same (user, program)
//     pair — the row is restored (deleted_at cleared) and the purchase is
//     completed separately. Restoration is idempotent; the second call hits a
//     NOT FOUND and falls through to path 2.
//
//  2. No soft-deleted entitlement exists — the purchase status change and the
//     new entitlement row are created inside a single database transaction
//     via CompleteWithEntitlement.
func (s *service) completeAndCreateEntitlement(ctx context.Context, purchase *models.Purchase) error {
	activeEntitlement, entErr := s.entitlements.FindActiveByUserAndProgram(ctx, purchase.UserID, purchase.ProgramID)
	if entErr == nil && activeEntitlement != nil {
		return fmt.Errorf("active entitlement already exists: %w", ErrDuplicateEntitlement)
	}
	if entErr != nil && !errors.Is(entErr, repositories.ErrEntitlementNotFound) {
		return fmt.Errorf("failed to check active entitlement: %w", entErr)
	}

	restoreErr := s.entitlements.RestoreByUserAndProgram(ctx, purchase.UserID, purchase.ProgramID)
	if restoreErr == nil {
		if err := s.purchases.Complete(ctx, purchase.ID); err != nil {
			if errors.Is(err, repositories.ErrPurchaseNotFound) {
				return ErrPurchaseNotFound
			}
			return fmt.Errorf("failed to complete purchase: %w", err)
		}
		return nil
	}
	if !errors.Is(restoreErr, repositories.ErrEntitlementNotFound) {
		return fmt.Errorf("failed to restore entitlement: %w", restoreErr)
	}

	entitlement := &models.Entitlement{
		UserID:    purchase.UserID,
		ProgramID: purchase.ProgramID,
	}
	if err := s.purchases.CompleteWithEntitlement(ctx, purchase.ID, entitlement); err != nil {
		if errors.Is(err, repositories.ErrPurchaseNotFound) {
			return ErrPurchaseNotFound
		}
		if errors.Is(err, repositories.ErrEntitlementAlreadyExists) {
			return ErrEntitlementIntegrity
		}
		return fmt.Errorf("failed to complete purchase with entitlement: %w", err)
	}
	return nil
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

func validatePurchaseID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: purchase id is required", ErrInvalidInput)
	}
	return nil
}
