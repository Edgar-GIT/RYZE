package purchases

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

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
	// terminal state (failed) that prevents completion.
	ErrPurchaseNotCompleted = errors.New("purchase is not in a completable state")
	// ErrEntitlementIntegrity indicates the purchase is already completed but
	// the corresponding entitlement is missing — a data consistency violation
	// that requires investigation.
	ErrEntitlementIntegrity = errors.New("entitlement integrity error")
	// ErrPurchaseNotPending indicates the purchase cannot accept a payment
	// initiation because it is not in the pending state.
	ErrPurchaseNotPending = errors.New("purchase is not pending")
	// ErrPaymentMethodMismatch indicates the requested payment method differs
	// from the method already recorded on the purchase at initiation. The
	// recorded method is immutable, so the purchase keeps its original method
	// and no provider is contacted.
	ErrPaymentMethodMismatch = errors.New("payment method does not match the recorded method")
	// ErrPaymentProvider indicates the payment provider could not initiate the
	// payment. Internal provider details are never exposed to the client.
	ErrPaymentProvider = errors.New("payment provider error")
	// ErrPrerequisiteNotMet indicates the product requires something the buyer
	// has not provided yet, such as a validated intake, so the purchase cannot
	// be started. It is reported before any purchase row is created, so a
	// rejected checkout never leaves a pending purchase behind.
	ErrPrerequisiteNotMet = errors.New("purchase prerequisite not met")
	// ErrPurchaseExpired indicates a pending purchase outlived its payment
	// window and can no longer be initiated — a provider session for it is no
	// longer payable. The buyer must start a fresh purchase intent. Completion
	// paths are deliberately unaffected: a provider-verified payment that
	// arrives late still completes instead of being discarded by a local timer.
	ErrPurchaseExpired = errors.New("purchase has expired")
	// ErrReconciliationConflict indicates a late, provider-verified payment
	// could not be reconciled because the (user, program) pair already resolved
	// another way: a completed purchase or an active entitlement exists, so the
	// late payment cannot define a second sale. The failed purchase is left
	// untouched as the persisted, diagnosable recovery state. This refusal is a
	// webhook-internal signal, never surfaced to a client route.
	ErrReconciliationConflict = errors.New("reconciliation conflict")
)

// defaultPurchaseExpiryWindow is the pending-purchase payment window applied
// when no option overrides it. It matches the payment providers' own defaults —
// Stripe Checkout Sessions expire after 24 hours and PayPal orders shortly
// after — so a pending purchase older than this can no longer be meaningfully
// paid at the provider.
const defaultPurchaseExpiryWindow = 24 * time.Hour

// ProgramRepository is the data-access surface for reading program data.
type ProgramRepository interface {
	FindPublishedByID(ctx context.Context, programID string) (*models.Program, error)
}

// PurchaseRepository is the data-access surface for purchase persistence.
type PurchaseRepository interface {
	Create(ctx context.Context, purchase *models.Purchase) error
	FindByID(ctx context.Context, purchaseID string) (*models.Purchase, error)
	ListActiveByUser(ctx context.Context, userID string) ([]models.Purchase, error)
	FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.Purchase, error)
	CompleteWithEntitlement(ctx context.Context, purchaseID string, entitlement *models.Entitlement) error
	// CompleteWithRestoredEntitlement atomically restores a soft-deleted
	// entitlement for the (user, program) pair and transitions the pending
	// purchase to completed inside one transaction, so a repurchase can never
	// leave a restored entitlement without a completed purchase (or vice
	// versa). It maps to ErrEntitlementNotFound when no soft-deleted
	// entitlement exists and to ErrPurchaseNotFound when the purchase is not
	// pending.
	CompleteWithRestoredEntitlement(ctx context.Context, purchaseID string, entitlement *models.Entitlement) error
	CompleteTestPurchase(ctx context.Context, purchase *models.Purchase, entitlement *models.Entitlement) error
	SetPaymentMethod(ctx context.Context, purchaseID, method string) error
	// Fail transitions a pending purchase to the failed status. The
	// `status = 'pending'` guard makes the transition single-writer: a
	// concurrently completed purchase is left untouched and maps to
	// ErrPurchaseNotFound so the caller re-evaluates the outcome. Failed is the
	// terminal state of an expired or superseded purchase; it never grants
	// access.
	Fail(ctx context.Context, purchaseID string) error
	// ReconcileFailedPurchase returns a failed purchase to completed and
	// provisions its entitlement atomically (see the repository
	// implementation for the full contract). It is the verdict-enforcement
	// layer for a late, provider-verified settlement.
	ReconcileFailedPurchase(ctx context.Context, purchaseID, userID, programID string, entitlement *models.Entitlement) error
}

// EntitlementRepository is the data-access surface for checking existing
// entitlements.
type EntitlementRepository interface {
	Create(ctx context.Context, userID, programID string, entitlement *models.Entitlement) error
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

// Program is the safe program summary exposed inside purchase history. It
// carries only public product metadata and never exposes the owning trainer,
// parent identifiers, deletion markers or any internal data.
type Program struct {
	ID               string
	Name             string
	Description      string
	Type             string
	Status           string
	Level            *string
	DurationWeeks    *int
	FrequencyPerWeek *int
	TrainingType     *string
	PriceMinorUnits  int64
	Currency         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Purchase is the safe representation of a purchase transaction. It carries
// only public commercial metadata and never exposes internal identifiers
// beyond the purchase and program id. Commission and payout data remain
// internal to the backend and are never serialized into the purchase history.
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
	PaymentMethod   string
	Test            bool
	Program         Program
	Access          bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
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
	InitiatePayment(ctx context.Context, userID, purchaseID, paymentMethod string) (*PaymentResult, error)
	CompletePurchase(ctx context.Context, purchaseID string) (*Purchase, error)
	GetPurchaseByID(ctx context.Context, purchaseID string) (*Purchase, error)
	ListPurchases(ctx context.Context, userID string) ([]Purchase, error)
	CapturePayment(ctx context.Context, userID, purchaseID, providerPaymentID string) (*Purchase, error)
	CompletePurchaseWithCapture(ctx context.Context, purchaseID, providerPaymentID string) (*Purchase, error)
	CompleteTestPurchase(ctx context.Context, userID, programID string) (*Purchase, error)
	// ReconcilePurchase completes a purchase whose provider-verified payment
	// arrived after the purchase left the pending state. It is the recovery
	// path for a late settlement and is intended exclusively for the verified
	// webhook layer: no client route accepts a purchase id without an
	// authenticated owner, so a browser can never trigger it. A failed purchase
	// is atomically returned to completed with its entitlement as long as no
	// completed purchase or active entitlement already resolved the pair
	// (ErrReconciliationConflict otherwise); an already-completed purchase is a
	// safe idempotent success.
	ReconcilePurchase(ctx context.Context, purchaseID string) (*Purchase, error)
}

type service struct {
	programs      ProgramRepository
	purchases     PurchaseRepository
	entitlements  EntitlementRepository
	commission    CommissionResolver
	resolver      payments.ProviderResolver
	expiryWindow  time.Duration
	prerequisites CheckoutPrerequisite
	fulfiller     PurchaseFulfiller
}

// CheckoutPrerequisite enforces a per-product precondition before a purchase can
// be started. It exists so a product that requires a validated intake — such as
// Premium Level 1 — cannot be bought before that intake exists, which keeps the
// derived plan grounded in a stored, validated document.
//
// The buyer identity is always the authenticated user resolved by the caller, so
// a client can never satisfy a precondition on behalf of another account.
type CheckoutPrerequisite interface {
	// Satisfied returns nil when the precondition for the product is met, or a
	// sentinel error when it is not. The product type is passed in because the
	// caller has already resolved the program.
	Satisfied(ctx context.Context, userID, programID, productType string) error
}

// PurchaseFulfiller provisions the derived artifacts a completed purchase owes,
// such as the Premium Level 1 nutrition plan. It is the seam that keeps product
// delivery out of the payment path: this service knows how to take money and grant
// access, and nothing about how a purchased product is built afterwards.
//
// The hook runs after the entitlement is durably committed and its failure is
// never allowed to fail the purchase.
type PurchaseFulfiller interface {
	// FulfillPurchase must be idempotent, because a purchase can be delivered
	// more than once — a provider webhook, a client capture callback or Test Mode
	// — and every delivery calls it.
	FulfillPurchase(ctx context.Context, userID, programID string) error
}

// Option configures an optional service collaborator.
type Option func(*service)

// WithCheckoutPrerequisites installs the per-product checkout precondition gate.
// It is optional: a service built without it performs no precondition checks,
// which is the behaviour every product that has no intake requirement relies on.
func WithCheckoutPrerequisites(prerequisites CheckoutPrerequisite) Option {
	return func(s *service) {
		s.prerequisites = prerequisites
	}
}

// WithFulfillment installs the post-commit product delivery hook. It is optional:
// a service built without it grants the entitlement and nothing more, which is
// the behaviour every product without derived artifacts relies on.
func WithFulfillment(fulfiller PurchaseFulfiller) Option {
	return func(s *service) {
		s.fulfiller = fulfiller
	}
}

// WithPurchaseExpiryWindow sets how long a pending purchase can be paid before
// it is considered expired. The window is derived from the purchase creation
// timestamp and is never persisted, so tuning it costs no migration. After the
// window lapses the purchase can no longer be initiated (InitiatePayment fails
// with ErrPurchaseExpired) but a provider-verified payment that arrives late is
// still completed — a captured payment is never discarded because of a local
// timer. Defaults to defaultPurchaseExpiryWindow.
func WithPurchaseExpiryWindow(window time.Duration) Option {
	return func(s *service) {
		s.expiryWindow = window
	}
}

func NewService(
	programs ProgramRepository,
	purchases PurchaseRepository,
	entitlements EntitlementRepository,
	commission CommissionResolver,
	resolver payments.ProviderResolver,
	options ...Option,
) Service {
	svc := &service{
		programs:     programs,
		purchases:    purchases,
		entitlements: entitlements,
		commission:   commission,
		resolver:     resolver,
		expiryWindow: defaultPurchaseExpiryWindow,
	}
	for _, option := range options {
		option(svc)
	}
	return svc
}

// fulfill runs the post-commit product delivery hook.
//
// A failure here is reported and then deliberately not propagated. By the time this
// runs the payment is captured and the entitlement is committed, so failing the
// request would tell a paying buyer their purchase failed while telling them the
// truth about their access. The derived artifact is recoverable instead — the hook
// is idempotent, so a repeated delivery or a retry recovers it, and the client's
// own access surface provisions and generates on demand — whereas a wrongly failed
// purchase is not recoverable at all.
func (s *service) fulfill(ctx context.Context, userID, programID string) {
	if s.fulfiller == nil {
		return
	}
	if err := s.fulfiller.FulfillPurchase(ctx, userID, programID); err != nil {
		slog.Error("purchase fulfillment failed",
			slog.String("user_id", userID),
			slog.String("program_id", programID),
			slog.String("error", err.Error()),
		)
	}
}

// checkPrerequisites enforces the per-product checkout precondition. It runs
// before any purchase row is created, so a product that is not ready to sell
// never produces a pending purchase.
func (s *service) checkPrerequisites(ctx context.Context, userID, programID, productType string) error {
	if s.prerequisites == nil {
		return nil
	}
	if err := s.prerequisites.Satisfied(ctx, userID, programID, productType); err != nil {
		return fmt.Errorf("%w: %w", ErrPrerequisiteNotMet, err)
	}
	return nil
}

// isExpired reports whether a pending purchase can no longer be paid. Expiry is
// derived from the creation timestamp plus the configured window and is never
// persisted. Only pending purchases expire, and the completion paths
// deliberately ignore expiry so a provider-verified payment that arrives after
// the window still completes instead of being discarded.
func (s *service) isExpired(purchase *models.Purchase) bool {
	if purchase.Status != models.PurchaseStatusPending {
		return false
	}
	return time.Since(purchase.CreatedAt) > s.expiryWindow
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

	// A product with a checkout precondition is only sellable once that
	// precondition holds. The check runs before the duplicate checks and before
	// any row is written, so a rejected checkout leaves no pending purchase
	// behind and never snapshots a price for a product the buyer cannot yet buy.
	if err := s.checkPrerequisites(ctx, userID, programID, program.ProductType); err != nil {
		return nil, err
	}

	// An active payable purchase already exists for this (user, program) pair:
	// the buyer must resume it instead of starting another. The only exception
	// is an expired pending purchase, which is superseded (marked failed) so
	// its window can be reopened with a fresh intent. A superseded purchase is
	// only ever the pending row; a concurrently completed purchase or an active
	// entitlement fall through to the duplicate checks that follow.
	existingPurchase, err := s.purchases.FindActiveByUserAndProgram(ctx, userID, programID)
	switch {
	case err == nil && existingPurchase == nil:
		// The repository reported no error and no row; treat as not found.
	case err != nil && !errors.Is(err, repositories.ErrPurchaseNotFound):
		return nil, fmt.Errorf("failed to check purchase: %w", err)
	case err == nil && s.isExpired(existingPurchase):
		if err := s.purchases.Fail(ctx, existingPurchase.ID); err != nil {
			switch {
			case errors.Is(err, repositories.ErrPurchaseNotFound):
				// The row moved on concurrently (it completed or was already
				// superseded). Re-evaluate ownership below before deciding
				// whether a fresh intent is still allowed.
			default:
				return nil, fmt.Errorf("failed to supersede expired purchase: %w", err)
			}
		}
		// Fall through: the expired pending slot is closed, so a fresh intent
		// below is allowed unless an active entitlement now exists.
	case err == nil && !s.isExpired(existingPurchase):
		return nil, ErrDuplicatePurchase
	}

	// After a supersede the concurrent operation may have completed the
	// purchase and created an entitlement, so the ownership check re-runs
	// below. It is the authoritative guard against selling a program the buyer
	// already owns.
	if _, err := s.entitlements.FindActiveByUserAndProgram(ctx, userID, programID); err == nil {
		return nil, ErrDuplicateEntitlement
	} else if !errors.Is(err, repositories.ErrEntitlementNotFound) {
		return nil, fmt.Errorf("failed to check entitlement: %w", err)
	}

	resolution := CommissionResolution{CommissionBPS: 0, IsOverride: false}
	calculation := CommissionCalculation{}
	if program.TrainerID != "" {
		// Only trainer-owned programs go through commission resolution.
		var err error
		resolution, err = s.commission.ResolveCommission(ctx, program.TrainerID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve commission: %w", err)
		}
		calculation = s.commission.CalculateCommissionSplit(program.PriceMinorUnits, resolution)
	} else {
		// Platform-owned generic programs (trainer_id NULL) have no owning
		// trainer, so no commission rule applies and the platform keeps the
		// full sale amount.
		calculation = CommissionCalculation{
			PlatformAmount: program.PriceMinorUnits,
			TrainerAmount:  0,
		}
	}

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
		if errors.Is(err, repositories.ErrPurchaseAlreadyExists) {
			// A concurrent writer created a purchase (or completed one) for the
			// same pair between the ownership checks above and the insert, which
			// the database rejected through uq_purchases_pending_purchase or
			// uq_purchases_completed_purchase. Re-evaluate the resolved state
			// instead of surfacing a raw conflict: an active entitlement means
			// the program is owned, otherwise the buyer should resume the other
			// live intent.
			if _, entErr := s.entitlements.FindActiveByUserAndProgram(ctx, userID, programID); entErr == nil {
				return nil, ErrDuplicateEntitlement
			} else if !errors.Is(entErr, repositories.ErrEntitlementNotFound) {
				return nil, fmt.Errorf("failed to check entitlement: %w", entErr)
			}
			return nil, ErrDuplicatePurchase
		}
		return nil, fmt.Errorf("failed to create purchase: %w", err)
	}

	return newPurchase(purchase), nil
}

// InitiatePayment requests a provider payment for an existing pending purchase.
// The purchase must belong to the authenticated user and be in "pending" status.
// The immutable purchase snapshot is used to construct the provider request; no
// client-supplied commercial values are accepted. The payment method is validated
// and resolved to the appropriate provider before initiation. The validated
// method is recorded on the purchase only once the provider accepted the
// initiation: from that point the method is immutable, so a later capture can
// always resolve the correct provider server-side without trusting any client
// input. A provider rejection leaves the method unbound, so the buyer can retry
// with another method instead of being locked to one the provider refused.
// Repeating the initiation with the same method is safe and replays the same
// provider payment; repeating it with a different method fails with
// ErrPaymentMethodMismatch and never rebinds the purchase.
// The purchase status is NOT modified during initiation — it remains "pending"
// until a verified provider event flows through CompletePurchase().
func (s *service) InitiatePayment(ctx context.Context, userID, purchaseID, paymentMethod string) (*PaymentResult, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if err := validatePurchaseID(purchaseID); err != nil {
		return nil, err
	}

	method := payments.PaymentMethod(paymentMethod)
	if err := payments.ValidatePaymentMethod(paymentMethod); err != nil {
		return nil, ErrInvalidInput
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

	// A pending purchase that outlived its payment window can no longer be
	// initiated — the provider session for it is no longer payable, so asking
	// the provider again would either fail or produce a payment that cannot
	// complete. The buyer recovers by starting a fresh intent, which supersedes
	// the expired purchase.
	if s.isExpired(purchase) {
		return nil, ErrPurchaseExpired
	}

	// The program must still be published and not soft-deleted before a new
	// provider payment is opened for it. The program was checked when the
	// intent was created, but it can be retired in between; a provider session
	// must never be opened for a program that no longer sells. Completion paths
	// are deliberately unaffected: a payment already captured at the provider
	// still completes, and access rendering already derives from publication.
	if _, err := s.programs.FindPublishedByID(ctx, purchase.ProgramID); err != nil {
		if errors.Is(err, repositories.ErrProgramNotFound) {
			return nil, ErrProgramNotFound
		}
		return nil, fmt.Errorf("failed to load program: %w", err)
	}

	// The recorded method is immutable for the purchase. A re-initiation with a
	// different method is rejected instead of rebinding the purchase: the
	// outstanding provider payment belongs to the method recorded at the first
	// successful initiation, so overwriting it would make that payment
	// uncapturable. The check runs before provider resolution so it never
	// depends on or exposes provider configuration.
	if purchase.PaymentMethod != nil && *purchase.PaymentMethod != string(method) {
		return nil, ErrPaymentMethodMismatch
	}

	provider, err := s.resolver(ctx, method)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPaymentProvider, err)
	}

	request := payments.PaymentRequest{
		PurchaseID:       purchase.ID,
		AmountMinorUnits: purchase.PriceMinorUnits,
		Currency:         purchase.Currency,
		ProgramID:        purchase.ProgramID,
		Method:           method,
	}

	result, err := provider.InitiatePayment(ctx, request)
	if err != nil {
		// Nothing has been recorded yet: a provider rejection leaves the
		// purchase free to be retried with any method, so a buyer is never
		// locked to a method the provider refuses.
		return nil, fmt.Errorf("%w: %v", ErrPaymentProvider, err)
	}

	// Record the validated method now that the provider accepted the initiation.
	// From this point the method is immutable for the purchase, which is what
	// lets the capture flow resolve the correct provider without any client
	// input. A same-method replay skips the write because the method is already
	// recorded; the mismatch check above rejects any different method.
	if purchase.PaymentMethod == nil {
		if err := s.purchases.SetPaymentMethod(ctx, purchase.ID, string(method)); err != nil {
			if errors.Is(err, repositories.ErrPurchaseNotFound) {
				return nil, ErrPurchaseNotFound
			}
			return nil, fmt.Errorf("failed to record payment method: %w", err)
		}
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
			// A replayed delivery still fulfils. The hook is idempotent, so a
			// purchase whose delivery failed the first time recovers on the next
			// delivery instead of staying permanently unfulfilled.
			s.fulfill(ctx, purchase.UserID, purchase.ProgramID)
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

	s.fulfill(ctx, purchase.UserID, purchase.ProgramID)

	purchase.Status = models.PurchaseStatusCompleted
	return newPurchase(purchase), nil
}

// completeAndCreateEntitlement provisions the entitlement for a purchase that
// is currently pending. Two distinct atomic paths exist, both guarded by the
// repository layer:
//
//  1. A previously soft-deleted entitlement exists for the same (user, program)
//     pair — the row is restored (deleted_at cleared) and the purchase is
//     completed inside a single database transaction via
//     CompleteWithRestoredEntitlement, so a repurchase can never leave a
//     restored entitlement without a completed purchase (or vice versa).
//
//  2. No soft-deleted entitlement exists — the purchase status change and the
//     new entitlement row are created inside a single database transaction
//     via CompleteWithEntitlement.
//
// Both paths report NOT FOUND when their precondition is absent and fall
// through to the other, mirroring the original restore-then-complete flow
// without the non-transactional window between the two steps.
func (s *service) completeAndCreateEntitlement(ctx context.Context, purchase *models.Purchase) error {
	activeEntitlement, entErr := s.entitlements.FindActiveByUserAndProgram(ctx, purchase.UserID, purchase.ProgramID)
	if entErr == nil && activeEntitlement != nil {
		return fmt.Errorf("active entitlement already exists: %w", ErrDuplicateEntitlement)
	}
	if entErr != nil && !errors.Is(entErr, repositories.ErrEntitlementNotFound) {
		return fmt.Errorf("failed to check active entitlement: %w", entErr)
	}

	entitlement := &models.Entitlement{
		UserID:    purchase.UserID,
		ProgramID: purchase.ProgramID,
	}

	err := s.purchases.CompleteWithRestoredEntitlement(ctx, purchase.ID, entitlement)
	switch {
	case err == nil:
		return nil
	case !errors.Is(err, repositories.ErrEntitlementNotFound):
		if errors.Is(err, repositories.ErrPurchaseNotFound) {
			return ErrPurchaseNotFound
		}
		return fmt.Errorf("failed to restore entitlement and complete purchase: %w", err)
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

// CompleteTestPurchase provisions a completed zero-price purchase and its
// entitlement for a paid program. It is the exclusive Test Mode purchase path:
// the entitlement is real (the persona owns the program afterwards) but the
// transaction is completed server-side without an order, a payment provider or
// any commission computation. The purchase is persisted with the test marker
// so it can never be confused with a paid sale. An already-owned program
// (active entitlement or an existing completed purchase) reports
// ErrDuplicateEntitlement.
func (s *service) CompleteTestPurchase(ctx context.Context, userID, programID string) (*Purchase, error) {
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

	// A product with a checkout precondition is only sellable once that
	// precondition holds. The check runs before the duplicate checks and before
	// any row is written, so a rejected checkout leaves no pending purchase
	// behind and never snapshots a price for a product the buyer cannot yet buy.
	if err := s.checkPrerequisites(ctx, userID, programID, program.ProductType); err != nil {
		return nil, err
	}

	if _, err := s.entitlements.FindActiveByUserAndProgram(ctx, userID, programID); err == nil {
		return nil, ErrDuplicateEntitlement
	} else if !errors.Is(err, repositories.ErrEntitlementNotFound) {
		return nil, fmt.Errorf("failed to check entitlement: %w", err)
	}

	purchase := &models.Purchase{
		UserID:          userID,
		ProgramID:       programID,
		PriceMinorUnits: 0,
		Currency:        program.Currency,
		Status:          models.PurchaseStatusCompleted,
		Test:            true,
	}
	entitlement := &models.Entitlement{
		UserID:    userID,
		ProgramID: programID,
	}

	if err := s.purchases.CompleteTestPurchase(ctx, purchase, entitlement); err != nil {
		if errors.Is(err, repositories.ErrCompletedPurchaseExists) || errors.Is(err, repositories.ErrEntitlementAlreadyExists) {
			return nil, ErrDuplicateEntitlement
		}
		return nil, fmt.Errorf("failed to complete test purchase: %w", err)
	}

	// Test Mode takes the same delivery path as a real payment, so the product a
	// persona bought is provisioned exactly as a paying buyer's would be.
	s.fulfill(ctx, userID, programID)

	return newPurchase(purchase), nil
}

// ReconcilePurchase completes a purchase whose provider-verified payment
// arrived after the purchase left the pending state. It is the recovery path
// for a late settlement: a verified provider event (signature, amount, currency
// and method all matched against this purchase's immutable snapshot) is honoured
// by atomically returning the failed purchase to completed and provisioning its
// entitlement — but only when no other completed purchase or active entitlement
// already exists for the (user, program) pair, which would mean access was
// already granted and the late payment cannot define a second sale. Those
// conflicts map to ErrReconciliationConflict and leave the failed purchase
// untouched as the persisted, diagnosable recovery state.
//
// The operation is scoped purely by the purchase id and is intended for the
// verified webhook layer only: it is never reachable from a client endpoint
// because no route accepts a purchase id without an authenticated owner, so a
// browser can never trigger it. An already-completed purchase is a safe
// idempotent success and still runs the delivery hook.
func (s *service) ReconcilePurchase(ctx context.Context, purchaseID string) (*Purchase, error) {
	if err := validatePurchaseID(purchaseID); err != nil {
		return nil, err
	}

	purchase, err := s.purchases.FindByID(ctx, purchaseID)
	if err != nil {
		if errors.Is(err, repositories.ErrPurchaseNotFound) {
			// An unknown purchase is a settled state, not a transient fault:
			// acknowledging it stops the provider from retrying a sale that can
			// never be granted.
			return nil, ErrPurchaseNotCompleted
		}
		return nil, fmt.Errorf("failed to load purchase: %w", err)
	}

	if purchase.Status == models.PurchaseStatusCompleted {
		existing, entErr := s.entitlements.FindActiveByUserAndProgram(ctx, purchase.UserID, purchase.ProgramID)
		if entErr != nil && !errors.Is(entErr, repositories.ErrEntitlementNotFound) {
			return nil, fmt.Errorf("failed to check entitlement: %w", entErr)
		}
		if existing != nil {
			s.fulfill(ctx, purchase.UserID, purchase.ProgramID)
			return newPurchase(purchase), nil
		}
		return nil, ErrEntitlementIntegrity
	}

	if purchase.Status != models.PurchaseStatusFailed {
		return nil, ErrPurchaseNotCompleted
	}

	entitlement := &models.Entitlement{
		UserID:    purchase.UserID,
		ProgramID: purchase.ProgramID,
	}

	err = s.purchases.ReconcileFailedPurchase(ctx, purchase.ID, purchase.UserID, purchase.ProgramID, entitlement)
	switch {
	case err == nil:
	case errors.Is(err, repositories.ErrPurchaseNotFound):
		// The purchase moved on concurrently (it was reconciled or superseded
		// again). The outcome is settled by the current state, and the call is
		// idempotent on the next delivery, so the caller re-evaluates instead
		// of surfacing a stale conflict.
		return nil, ErrPurchaseNotCompleted
	case errors.Is(err, repositories.ErrCompletedPurchaseExists), errors.Is(err, repositories.ErrEntitlementAlreadyExists):
		return nil, ErrReconciliationConflict
	default:
		return nil, fmt.Errorf("failed to reconcile purchase: %w", err)
	}

	s.fulfill(ctx, purchase.UserID, purchase.ProgramID)

	purchase.Status = models.PurchaseStatusCompleted
	return newPurchase(purchase), nil
}

// GetPurchaseByID returns the safe representation of a purchase by its
// identifier. It is a read-only query intended for internal verification
// flows such as webhook processing, where the caller needs to inspect the
// immutable commercial snapshot before triggering completion.
func (s *service) GetPurchaseByID(ctx context.Context, purchaseID string) (*Purchase, error) {
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

	return newPurchase(purchase), nil
}

// ListPurchases returns the safe representation of every active purchase
// belonging to the given user, most recently created first. The user identity
// always comes from the caller, never from the client.
func (s *service) ListPurchases(ctx context.Context, userID string) ([]Purchase, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	records, err := s.purchases.ListActiveByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list purchases: %w", err)
	}

	purchases := make([]Purchase, 0, len(records))
	for i := range records {
		purchases = append(purchases, *newPurchase(&records[i]))
	}
	return purchases, nil
}

// CapturePayment captures an approved provider payment for an existing pending
// purchase that belongs to the authenticated user. The provider payment
// identifier comes from the browser callback and is always treated as untrusted
// input: it is bound to the purchase by the provider (reference id, amount and
// currency verification) before any capture happens. On success the purchase is
// completed atomically with its entitlement through CompletePurchase.
//
// The operation is idempotent: an already-completed purchase returns the
// existing purchase without re-capturing or duplicating anything, so a user
// returning twice (double tab, duplicated callback) is always safe.
func (s *service) CapturePayment(ctx context.Context, userID, purchaseID, providerPaymentID string) (*Purchase, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if err := validatePurchaseID(purchaseID); err != nil {
		return nil, err
	}
	if providerPaymentID == "" {
		return nil, ErrInvalidInput
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

	return s.captureAndComplete(ctx, purchase, providerPaymentID)
}

// CompletePurchaseWithCapture is the internal counterpart of CapturePayment,
// scoped purely by the purchase id with no user identity. It is used by the
// verified PayPal webhook flow to capture the approved order server-side and
// then complete the purchase. The provider payment identifier comes from the
// verified event payload and is bound to the purchase by the provider before
// any capture happens. It is idempotent like CapturePayment.
func (s *service) CompletePurchaseWithCapture(ctx context.Context, purchaseID, providerPaymentID string) (*Purchase, error) {
	if err := validatePurchaseID(purchaseID); err != nil {
		return nil, err
	}
	if providerPaymentID == "" {
		return nil, ErrInvalidInput
	}

	purchase, err := s.purchases.FindByID(ctx, purchaseID)
	if err != nil {
		if errors.Is(err, repositories.ErrPurchaseNotFound) {
			return nil, ErrPurchaseNotFound
		}
		return nil, fmt.Errorf("failed to load purchase: %w", err)
	}

	return s.captureAndComplete(ctx, purchase, providerPaymentID)
}

// captureAndComplete captures the approved provider payment for the purchase
// and completes it atomically. The capture provider is resolved from the
// payment method recorded on the purchase at initiation — it is never derived
// from client input. A purchase without a recorded method cannot be captured
// and fails closed as a provider error, keeping the purchase pending so the
// buyer can re-initiate. An already-completed purchase is a safe idempotent
// success. Capture failures surface as ErrPaymentProvider and NEVER complete
// the purchase — the status remains pending so the buyer can retry or
// re-initiate.
func (s *service) captureAndComplete(ctx context.Context, purchase *models.Purchase, providerPaymentID string) (*Purchase, error) {
	if purchase.Status == models.PurchaseStatusCompleted {
		existing, entErr := s.entitlements.FindActiveByUserAndProgram(ctx, purchase.UserID, purchase.ProgramID)
		if entErr != nil && !errors.Is(entErr, repositories.ErrEntitlementNotFound) {
			return nil, fmt.Errorf("failed to check entitlement: %w", entErr)
		}
		if existing != nil {
			s.fulfill(ctx, purchase.UserID, purchase.ProgramID)
			return newPurchase(purchase), nil
		}
		return nil, ErrEntitlementIntegrity
	}

	if purchase.Status != models.PurchaseStatusPending {
		return nil, ErrPurchaseNotPending
	}

	methodValue := ""
	if purchase.PaymentMethod != nil {
		methodValue = *purchase.PaymentMethod
	}
	method := payments.PaymentMethod(methodValue)
	if err := payments.ValidatePaymentMethod(methodValue); err != nil {
		return nil, fmt.Errorf("%w: purchase has no recorded payment method", ErrPaymentProvider)
	}
	provider, err := s.resolver(ctx, method)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPaymentProvider, err)
	}
	captureProvider, ok := provider.(payments.CaptureProvider)
	if !ok {
		return nil, fmt.Errorf("%w: provider does not support capture", ErrPaymentProvider)
	}

	captureRequest := payments.CaptureRequest{
		PurchaseID:       purchase.ID,
		PaymentID:        providerPaymentID,
		AmountMinorUnits: purchase.PriceMinorUnits,
		Currency:         purchase.Currency,
		Method:           payments.PaymentMethod(methodValue),
	}
	if _, err := captureProvider.CapturePayment(ctx, captureRequest); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPaymentProvider, err)
	}

	return s.CompletePurchase(ctx, purchase.ID)
}

func newPurchase(model *models.Purchase) *Purchase {
	paymentMethod := ""
	if model.PaymentMethod != nil {
		paymentMethod = *model.PaymentMethod
	}
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
		PaymentMethod:   paymentMethod,
		Test:            model.Test,
		Program:         newProgram(&model.Program),
		Access:          hasAccess(model),
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
}

func newProgram(model *models.Program) Program {
	return Program{
		ID:               model.ID,
		Name:             model.Name,
		Description:      model.Description,
		Type:             model.Type,
		Status:           model.Status,
		Level:            model.Level,
		DurationWeeks:    model.DurationWeeks,
		FrequencyPerWeek: model.FrequencyPerWeek,
		TrainingType:     model.TrainingType,
		PriceMinorUnits:  model.PriceMinorUnits,
		Currency:         model.Currency,
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
	}
}

// hasAccess reports whether a purchase currently grants access to its program.
// Access only exists for completed purchases of published programs that have
// not been soft-deleted. The program association must be preloaded by the
// caller; a missing association never grants access.
func hasAccess(model *models.Purchase) bool {
	if model.Status != models.PurchaseStatusCompleted {
		return false
	}
	if model.Program.DeletedAt.Valid {
		return false
	}
	return model.Program.Status == models.ProgramStatusPublished
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
