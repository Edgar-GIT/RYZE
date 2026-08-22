package purchases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/payments"
	"ryze/backend/services/purchases"
)

// --- stubs ---

type stubProgramRepository struct {
	program *models.Program
	err     error
}

func (s *stubProgramRepository) FindPublishedByID(_ context.Context, _ string) (*models.Program, error) {
	return s.program, s.err
}

type stubPurchaseRepository struct {
	purchase         *models.Purchase
	existing         *models.Purchase
	createErr        error
	findErr          error
	findByIDPurchase *models.Purchase
	findByIDErr      error
}

func (s *stubPurchaseRepository) Create(_ context.Context, purchase *models.Purchase) error {
	if s.createErr != nil {
		return s.createErr
	}
	purchase.ID = "00000000-0000-0000-0000-000000000001"
	purchase.CreatedAt = time.Now()
	purchase.UpdatedAt = time.Now()
	s.purchase = purchase
	return nil
}

func (s *stubPurchaseRepository) FindByID(_ context.Context, _ string) (*models.Purchase, error) {
	if s.findByIDErr != nil {
		return nil, s.findByIDErr
	}
	if s.findByIDPurchase != nil {
		return s.findByIDPurchase, nil
	}
	return nil, repositories.ErrPurchaseNotFound
}

func (s *stubPurchaseRepository) FindActiveByUserAndProgram(_ context.Context, _, _ string) (*models.Purchase, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	if s.existing != nil {
		return s.existing, nil
	}
	return nil, repositories.ErrPurchaseNotFound
}

func (s *stubPurchaseRepository) Complete(_ context.Context, _ string) error {
	return nil
}

func (s *stubPurchaseRepository) CompleteWithEntitlement(_ context.Context, _ string, _ *models.Entitlement) error {
	return nil
}

type stubEntitlementRepository struct {
	existing *models.Entitlement
	err      error
}

func (s *stubEntitlementRepository) Create(_ context.Context, _, _ string, _ *models.Entitlement) error {
	return nil
}

func (s *stubEntitlementRepository) FindActiveByUserAndProgram(_ context.Context, _, _ string) (*models.Entitlement, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.existing != nil {
		return s.existing, nil
	}
	return nil, repositories.ErrEntitlementNotFound
}

func (s *stubEntitlementRepository) RestoreByUserAndProgram(_ context.Context, _, _ string) error {
	return repositories.ErrEntitlementNotFound
}

type stubCommissionResolver struct {
	resolution purchases.CommissionResolution
	calc       purchases.CommissionCalculation
	err        error
}

func (s *stubCommissionResolver) ResolveCommission(_ context.Context, _ string) (purchases.CommissionResolution, error) {
	return s.resolution, s.err
}

func (s *stubCommissionResolver) CalculateCommissionSplit(_ int64, _ purchases.CommissionResolution) purchases.CommissionCalculation {
	return s.calc
}

type stubPaymentProvider struct {
	result         payments.PaymentResult
	err            error
	captureRequest *payments.PaymentRequest
}

func (s *stubPaymentProvider) InitiatePayment(_ context.Context, req payments.PaymentRequest) (payments.PaymentResult, error) {
	if s.captureRequest != nil {
		*s.captureRequest = req
	}
	return s.result, s.err
}

// stubResolver returns a ProviderResolver that always resolves to the given
// provider, enabling InitiatePayment tests without a real routing layer.
func stubResolver(provider payments.Provider) payments.ProviderResolver {
	return func(_ context.Context, _ payments.PaymentMethod) (payments.Provider, error) {
		return provider, nil
	}
}

// --- tests ---

func TestCreatePurchaseIntentSuccess(t *testing.T) {
	program := &models.Program{
		ID:              "11111111-1111-1111-1111-111111111111",
		TrainerID:       "22222222-2222-2222-2222-222222222222",
		Name:            "Premium Program",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 10000,
		Currency:        "EUR",
	}

	programs := &stubProgramRepository{program: program}
	purchasesRepo := &stubPurchaseRepository{}
	entitlements := &stubEntitlementRepository{}
	commission := &stubCommissionResolver{
		resolution: purchases.CommissionResolution{CommissionBPS: 2000, IsOverride: false},
		calc:       purchases.CommissionCalculation{PlatformAmount: 2000, TrainerAmount: 8000},
	}

	svc := purchases.NewService(programs, purchasesRepo, entitlements, commission, &stubPaymentProvider{}, nil)

	purchase, err := svc.CreatePurchaseIntent(context.Background(), "33333333-3333-3333-3333-333333333333", "11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if purchase.Status != models.PurchaseStatusPending {
		t.Fatalf("expected status %q, got %q", models.PurchaseStatusPending, purchase.Status)
	}
	if purchase.PriceMinorUnits != 10000 {
		t.Fatalf("expected price 10000, got %d", purchase.PriceMinorUnits)
	}
	if purchase.Currency != "EUR" {
		t.Fatalf("expected currency EUR, got %s", purchase.Currency)
	}
	if purchase.CommissionBPS != 2000 {
		t.Fatalf("expected commission 2000, got %d", purchase.CommissionBPS)
	}
	if purchase.PlatformAmount != 2000 {
		t.Fatalf("expected platform amount 2000, got %d", purchase.PlatformAmount)
	}
	if purchase.TrainerAmount != 8000 {
		t.Fatalf("expected trainer amount 8000, got %d", purchase.TrainerAmount)
	}
}

func TestCreatePurchaseIntentEmptyUserID(t *testing.T) {
	svc := purchases.NewService(
		&stubProgramRepository{},
		&stubPurchaseRepository{},
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.CreatePurchaseIntent(context.Background(), "", "11111111-1111-1111-1111-111111111111")
	if err == nil {
		t.Fatal("expected error for empty user id")
	}
	if !errors.Is(err, purchases.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreatePurchaseIntentEmptyProgramID(t *testing.T) {
	svc := purchases.NewService(
		&stubProgramRepository{},
		&stubPurchaseRepository{},
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.CreatePurchaseIntent(context.Background(), "33333333-3333-3333-3333-333333333333", "")
	if err == nil {
		t.Fatal("expected error for empty program id")
	}
	if !errors.Is(err, purchases.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreatePurchaseIntentProgramNotFound(t *testing.T) {
	programs := &stubProgramRepository{err: repositories.ErrProgramNotFound}
	svc := purchases.NewService(
		programs,
		&stubPurchaseRepository{},
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.CreatePurchaseIntent(context.Background(), "33333333-3333-3333-3333-333333333333", "11111111-1111-1111-1111-111111111111")
	if err == nil {
		t.Fatal("expected error for missing program")
	}
	if !errors.Is(err, purchases.ErrProgramNotFound) {
		t.Fatalf("expected ErrProgramNotFound, got %v", err)
	}
}

func TestCreatePurchaseIntentFreeProgram(t *testing.T) {
	program := &models.Program{
		ID:              "11111111-1111-1111-1111-111111111111",
		Type:            models.ProgramTypeFree,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 0,
		Currency:        "EUR",
	}

	svc := purchases.NewService(
		&stubProgramRepository{program: program},
		&stubPurchaseRepository{},
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.CreatePurchaseIntent(context.Background(), "33333333-3333-3333-3333-333333333333", "11111111-1111-1111-1111-111111111111")
	if err == nil {
		t.Fatal("expected error for free program")
	}
	if !errors.Is(err, purchases.ErrProgramNotPurchasable) {
		t.Fatalf("expected ErrProgramNotPurchasable, got %v", err)
	}
}

func TestCreatePurchaseIntentDuplicateEntitlement(t *testing.T) {
	program := &models.Program{
		ID:              "11111111-1111-1111-1111-111111111111",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 5000,
		Currency:        "EUR",
	}

	entitlements := &stubEntitlementRepository{
		existing: &models.Entitlement{
			ID:        "44444444-4444-4444-4444-444444444444",
			UserID:    "33333333-3333-3333-3333-333333333333",
			ProgramID: "11111111-1111-1111-1111-111111111111",
		},
		err: nil,
	}

	svc := purchases.NewService(
		&stubProgramRepository{program: program},
		&stubPurchaseRepository{},
		entitlements,
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.CreatePurchaseIntent(context.Background(), "33333333-3333-3333-3333-333333333333", "11111111-1111-1111-1111-111111111111")
	if err == nil {
		t.Fatal("expected error for duplicate entitlement")
	}
	if !errors.Is(err, purchases.ErrDuplicateEntitlement) {
		t.Fatalf("expected ErrDuplicateEntitlement, got %v", err)
	}
}

func TestCreatePurchaseIntentDuplicatePurchase(t *testing.T) {
	program := &models.Program{
		ID:              "11111111-1111-1111-1111-111111111111",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 5000,
		Currency:        "EUR",
	}

	existingPurchase := &models.Purchase{
		ID:     "55555555-5555-5555-5555-555555555555",
		Status: models.PurchaseStatusPending,
	}

	purchasesRepo := &stubPurchaseRepository{existing: existingPurchase}

	svc := purchases.NewService(
		&stubProgramRepository{program: program},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.CreatePurchaseIntent(context.Background(), "33333333-3333-3333-3333-333333333333", "11111111-1111-1111-1111-111111111111")
	if err == nil {
		t.Fatal("expected error for duplicate purchase")
	}
	if !errors.Is(err, purchases.ErrDuplicatePurchase) {
		t.Fatalf("expected ErrDuplicatePurchase, got %v", err)
	}
}

func TestCreatePurchaseIntentCommissionResolutionFailure(t *testing.T) {
	program := &models.Program{
		ID:              "11111111-1111-1111-1111-111111111111",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 5000,
		Currency:        "EUR",
	}

	commission := &stubCommissionResolver{err: errors.New("resolution failed")}

	svc := purchases.NewService(
		&stubProgramRepository{program: program},
		&stubPurchaseRepository{},
		&stubEntitlementRepository{},
		commission,
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.CreatePurchaseIntent(context.Background(), "33333333-3333-3333-3333-333333333333", "11111111-1111-1111-1111-111111111111")
	if err == nil {
		t.Fatal("expected error for commission resolution failure")
	}
	if errors.Is(err, purchases.ErrInvalidInput) || errors.Is(err, purchases.ErrProgramNotFound) {
		t.Fatalf("got user-facing error instead of internal error: %v", err)
	}
}

func TestCreatePurchaseIntentCreateFailure(t *testing.T) {
	program := &models.Program{
		ID:              "11111111-1111-1111-1111-111111111111",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 5000,
		Currency:        "EUR",
	}

	purchasesRepo := &stubPurchaseRepository{createErr: errors.New("db failure")}
	commission := &stubCommissionResolver{
		resolution: purchases.CommissionResolution{CommissionBPS: 2000},
		calc:       purchases.CommissionCalculation{PlatformAmount: 1000, TrainerAmount: 4000},
	}

	svc := purchases.NewService(
		&stubProgramRepository{program: program},
		purchasesRepo,
		&stubEntitlementRepository{},
		commission,
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.CreatePurchaseIntent(context.Background(), "33333333-3333-3333-3333-333333333333", "11111111-1111-1111-1111-111111111111")
	if err == nil {
		t.Fatal("expected error for create failure")
	}
	if errors.Is(err, purchases.ErrInvalidInput) || errors.Is(err, purchases.ErrProgramNotFound) {
		t.Fatalf("got user-facing error instead of internal error: %v", err)
	}
}

func TestCreatePurchaseIntentOverrideCommission(t *testing.T) {
	program := &models.Program{
		ID:              "11111111-1111-1111-1111-111111111111",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 20000,
		Currency:        "EUR",
	}

	commission := &stubCommissionResolver{
		resolution: purchases.CommissionResolution{CommissionBPS: 1500, IsOverride: true},
		calc:       purchases.CommissionCalculation{PlatformAmount: 3000, TrainerAmount: 17000},
	}

	svc := purchases.NewService(
		&stubProgramRepository{program: program},
		&stubPurchaseRepository{},
		&stubEntitlementRepository{},
		commission,
		&stubPaymentProvider{},
		nil,
	)

	purchase, err := svc.CreatePurchaseIntent(context.Background(), "33333333-3333-3333-3333-333333333333", "11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if purchase.CommissionBPS != 1500 {
		t.Fatalf("expected commission 1500, got %d", purchase.CommissionBPS)
	}
	if purchase.PlatformAmount != 3000 {
		t.Fatalf("expected platform amount 3000, got %d", purchase.PlatformAmount)
	}
	if purchase.TrainerAmount != 17000 {
		t.Fatalf("expected trainer amount 17000, got %d", purchase.TrainerAmount)
	}
}

// --- CompletePurchase tests ---

type completionPurchaseRepo struct {
	findByIDPurchase      *models.Purchase
	findByIDErr           error
	completeErr           error
	completeWithEntErr    error
	completedPurchaseID   string
	completedWithEntID    string
	completedWithEntModel *models.Entitlement
}

func (r *completionPurchaseRepo) Create(_ context.Context, _ *models.Purchase) error { return nil }
func (r *completionPurchaseRepo) FindActiveByUserAndProgram(_ context.Context, _, _ string) (*models.Purchase, error) {
	return nil, repositories.ErrPurchaseNotFound
}
func (r *completionPurchaseRepo) FindByID(_ context.Context, _ string) (*models.Purchase, error) {
	return r.findByIDPurchase, r.findByIDErr
}
func (r *completionPurchaseRepo) Complete(_ context.Context, purchaseID string) error {
	r.completedPurchaseID = purchaseID
	return r.completeErr
}
func (r *completionPurchaseRepo) CompleteWithEntitlement(_ context.Context, purchaseID string, entitlement *models.Entitlement) error {
	r.completedWithEntID = purchaseID
	r.completedWithEntModel = entitlement
	return r.completeWithEntErr
}

type completionEntitlementRepo struct {
	activeEntitlement *models.Entitlement
	activeErr         error
	createErr         error
	restoreErr        error
	restoredUserID    string
	restoredProgramID string
}

func (r *completionEntitlementRepo) Create(_ context.Context, userID, programID string, _ *models.Entitlement) error {
	return r.createErr
}
func (r *completionEntitlementRepo) FindActiveByUserAndProgram(_ context.Context, _, _ string) (*models.Entitlement, error) {
	if r.activeEntitlement != nil {
		return r.activeEntitlement, r.activeErr
	}
	return nil, repositories.ErrEntitlementNotFound
}
func (r *completionEntitlementRepo) RestoreByUserAndProgram(_ context.Context, userID, programID string) error {
	r.restoredUserID = userID
	r.restoredProgramID = programID
	return r.restoreErr
}

func TestCompletePurchaseSuccess(t *testing.T) {
	purchase := &models.Purchase{
		ID:              "aaaa1111-1111-1111-1111-111111111111",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 10000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}

	purchasesRepo := &completionPurchaseRepo{findByIDPurchase: purchase}
	entitlementsRepo := &completionEntitlementRepo{restoreErr: repositories.ErrEntitlementNotFound}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		entitlementsRepo,
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	result, err := svc.CompletePurchase(context.Background(), purchase.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != models.PurchaseStatusCompleted {
		t.Fatalf("expected status %q, got %q", models.PurchaseStatusCompleted, result.Status)
	}
	if purchasesRepo.completedWithEntID != purchase.ID {
		t.Fatalf("expected CompleteWithEntitlement called with %q, got %q", purchase.ID, purchasesRepo.completedWithEntID)
	}
}

func TestCompletePurchaseIdempotentSuccess(t *testing.T) {
	purchase := &models.Purchase{
		ID:              "aaaa1111-1111-1111-1111-111111111111",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 10000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusCompleted,
	}

	activeEntitlement := &models.Entitlement{
		ID:        "bbbb1111-1111-1111-1111-111111111111",
		UserID:    purchase.UserID,
		ProgramID: purchase.ProgramID,
	}

	purchasesRepo := &completionPurchaseRepo{findByIDPurchase: purchase}
	entitlementsRepo := &completionEntitlementRepo{activeEntitlement: activeEntitlement}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		entitlementsRepo,
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	result, err := svc.CompletePurchase(context.Background(), purchase.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != models.PurchaseStatusCompleted {
		t.Fatalf("expected status %q, got %q", models.PurchaseStatusCompleted, result.Status)
	}
	if purchasesRepo.completedPurchaseID != "" {
		t.Fatal("Complete must not be called for idempotent success")
	}
	if purchasesRepo.completedWithEntID != "" {
		t.Fatal("CompleteWithEntitlement must not be called for idempotent success")
	}
}

func TestCompletePurchaseEntitlementIntegrityError(t *testing.T) {
	purchase := &models.Purchase{
		ID:              "aaaa1111-1111-1111-1111-111111111111",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 10000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusCompleted,
	}

	purchasesRepo := &completionPurchaseRepo{findByIDPurchase: purchase}
	entitlementsRepo := &completionEntitlementRepo{}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		entitlementsRepo,
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.CompletePurchase(context.Background(), purchase.ID)
	if !errors.Is(err, purchases.ErrEntitlementIntegrity) {
		t.Fatalf("expected ErrEntitlementIntegrity, got %v", err)
	}
}

func TestCompletePurchaseNotFound(t *testing.T) {
	purchasesRepo := &completionPurchaseRepo{
		findByIDErr: repositories.ErrPurchaseNotFound,
	}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&completionEntitlementRepo{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.CompletePurchase(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, purchases.ErrPurchaseNotFound) {
		t.Fatalf("expected ErrPurchaseNotFound, got %v", err)
	}
}

func TestCompletePurchaseEmptyID(t *testing.T) {
	svc := purchases.NewService(
		&stubProgramRepository{},
		&completionPurchaseRepo{},
		&completionEntitlementRepo{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.CompletePurchase(context.Background(), "")
	if !errors.Is(err, purchases.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCompletePurchaseFailedStatus(t *testing.T) {
	purchase := &models.Purchase{
		ID:              "aaaa1111-1111-1111-1111-111111111111",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 10000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusFailed,
	}

	purchasesRepo := &completionPurchaseRepo{findByIDPurchase: purchase}
	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&completionEntitlementRepo{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.CompletePurchase(context.Background(), purchase.ID)
	if !errors.Is(err, purchases.ErrPurchaseNotCompleted) {
		t.Fatalf("expected ErrPurchaseNotCompleted, got %v", err)
	}
}

func TestCompletePurchaseRefundedStatus(t *testing.T) {
	purchase := &models.Purchase{
		ID:              "aaaa1111-1111-1111-1111-111111111111",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 10000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusRefunded,
	}

	purchasesRepo := &completionPurchaseRepo{findByIDPurchase: purchase}
	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&completionEntitlementRepo{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.CompletePurchase(context.Background(), purchase.ID)
	if !errors.Is(err, purchases.ErrPurchaseNotCompleted) {
		t.Fatalf("expected ErrPurchaseNotCompleted, got %v", err)
	}
}

func TestCreatePurchaseIntentEntitlementCheckFailure(t *testing.T) {
	program := &models.Program{
		ID:              "11111111-1111-1111-1111-111111111111",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 5000,
		Currency:        "EUR",
	}

	entitlements := &stubEntitlementRepository{
		existing: nil,
		err:      errors.New("database connection lost"),
	}

	svc := purchases.NewService(
		&stubProgramRepository{program: program},
		&stubPurchaseRepository{},
		entitlements,
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.CreatePurchaseIntent(context.Background(), "33333333-3333-3333-3333-333333333333", "11111111-1111-1111-1111-111111111111")
	if err == nil {
		t.Fatal("expected error for entitlement check failure")
	}
	if errors.Is(err, purchases.ErrInvalidInput) || errors.Is(err, purchases.ErrProgramNotFound) {
		t.Fatalf("internal error must not map to user-facing error: %v", err)
	}
}

func TestCreatePurchaseIntentPurchaseCheckFailure(t *testing.T) {
	program := &models.Program{
		ID:              "11111111-1111-1111-1111-111111111111",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 5000,
		Currency:        "EUR",
	}

	purchasesRepo := &stubPurchaseRepository{
		findErr: errors.New("database connection lost"),
	}

	svc := purchases.NewService(
		&stubProgramRepository{program: program},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.CreatePurchaseIntent(context.Background(), "33333333-3333-3333-3333-333333333333", "11111111-1111-1111-1111-111111111111")
	if err == nil {
		t.Fatal("expected error for purchase check failure")
	}
	if errors.Is(err, purchases.ErrInvalidInput) || errors.Is(err, purchases.ErrProgramNotFound) {
		t.Fatalf("internal error must not map to user-facing error: %v", err)
	}
}

func TestInitiatePaymentInvalidMethod(t *testing.T) {
	svc := purchases.NewService(
		&stubProgramRepository{},
		&stubPurchaseRepository{},
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.InitiatePayment(context.Background(), "33333333-3333-3333-3333-333333333333", "purchase-001", "bitcoin")
	if err == nil {
		t.Fatal("expected error for invalid payment method")
	}
	if !errors.Is(err, purchases.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCompletePurchaseEntitlementReactivation(t *testing.T) {
	purchase := &models.Purchase{
		ID:              "aaaa1111-1111-1111-1111-111111111111",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 10000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}

	purchasesRepo := &completionPurchaseRepo{findByIDPurchase: purchase}
	entitlementsRepo := &completionEntitlementRepo{restoreErr: nil}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		entitlementsRepo,
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	result, err := svc.CompletePurchase(context.Background(), purchase.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != models.PurchaseStatusCompleted {
		t.Fatalf("expected status %q, got %q", models.PurchaseStatusCompleted, result.Status)
	}
	if entitlementsRepo.restoredUserID != purchase.UserID {
		t.Fatalf("expected restore called with user %q, got %q", purchase.UserID, entitlementsRepo.restoredUserID)
	}
	if entitlementsRepo.restoredProgramID != purchase.ProgramID {
		t.Fatalf("expected restore called with program %q, got %q", purchase.ProgramID, entitlementsRepo.restoredProgramID)
	}
	if purchasesRepo.completedPurchaseID != purchase.ID {
		t.Fatalf("expected Complete called with %q, got %q", purchase.ID, purchasesRepo.completedPurchaseID)
	}
	if purchasesRepo.completedWithEntID != "" {
		t.Fatal("CompleteWithEntitlement must not be called when restore succeeds")
	}
}

func TestCompletePurchaseSnapshotPreserved(t *testing.T) {
	purchase := &models.Purchase{
		ID:              "aaaa1111-1111-1111-1111-111111111111",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 15000,
		Currency:        "EUR",
		CommissionBPS:   1500,
		PlatformAmount:  2250,
		TrainerAmount:   12750,
		Status:          models.PurchaseStatusPending,
	}

	purchasesRepo := &completionPurchaseRepo{findByIDPurchase: purchase}
	entitlementsRepo := &completionEntitlementRepo{restoreErr: repositories.ErrEntitlementNotFound}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		entitlementsRepo,
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	result, err := svc.CompletePurchase(context.Background(), purchase.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PriceMinorUnits != 15000 {
		t.Fatalf("expected price 15000, got %d", result.PriceMinorUnits)
	}
	if result.CommissionBPS != 1500 {
		t.Fatalf("expected commission 1500, got %d", result.CommissionBPS)
	}
	if result.PlatformAmount != 2250 {
		t.Fatalf("expected platform amount 2250, got %d", result.PlatformAmount)
	}
	if result.TrainerAmount != 12750 {
		t.Fatalf("expected trainer amount 12750, got %d", result.TrainerAmount)
	}
}

func TestCompletePurchaseInternalErrorNotExposed(t *testing.T) {
	purchase := &models.Purchase{
		ID:        "aaaa1111-1111-1111-1111-111111111111",
		UserID:    "33333333-3333-3333-3333-333333333333",
		ProgramID: "11111111-1111-1111-1111-111111111111",
		Status:    models.PurchaseStatusPending,
	}

	purchasesRepo := &completionPurchaseRepo{
		findByIDPurchase:   purchase,
		completeWithEntErr: errors.New("database connection lost"),
	}
	entitlementsRepo := &completionEntitlementRepo{restoreErr: repositories.ErrEntitlementNotFound}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		entitlementsRepo,
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.CompletePurchase(context.Background(), purchase.ID)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, purchases.ErrPurchaseNotFound) || errors.Is(err, purchases.ErrInvalidInput) {
		t.Fatalf("internal error must not map to user-facing error: %v", err)
	}
}

func TestInitiatePaymentSuccess(t *testing.T) {
	purchase := &models.Purchase{
		ID:              "purchase-001",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 10000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}

	purchasesRepo := &stubPurchaseRepository{
		findByIDPurchase: purchase,
	}
	payment := &stubPaymentProvider{
		result: payments.PaymentResult{
			PaymentID:   "pay_abc123",
			Status:      payments.PaymentStatusPending,
			CheckoutURL: "https://checkout.example.com/pay/abc123",
			Provider:    "fake",
			PurchaseID:  purchase.ID,
		},
	}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		payment,
		stubResolver(payment),
	)

	result, err := svc.InitiatePayment(context.Background(), "33333333-3333-3333-3333-333333333333", "purchase-001", "card")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PaymentID != "pay_abc123" {
		t.Fatalf("expected payment id %q, got %q", "pay_abc123", result.PaymentID)
	}
	if result.CheckoutURL != "https://checkout.example.com/pay/abc123" {
		t.Fatalf("expected checkout url, got %q", result.CheckoutURL)
	}
	if result.Status != payments.PaymentStatusPending {
		t.Fatalf("expected status %q, got %q", payments.PaymentStatusPending, result.Status)
	}
	if result.Provider != "fake" {
		t.Fatalf("expected provider %q, got %q", "fake", result.Provider)
	}
	if result.PurchaseID != purchase.ID {
		t.Fatalf("expected purchase id %q, got %q", purchase.ID, result.PurchaseID)
	}
}

func TestInitiatePaymentEmptyUserID(t *testing.T) {
	svc := purchases.NewService(
		&stubProgramRepository{},
		&stubPurchaseRepository{},
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.InitiatePayment(context.Background(), "", "purchase-001", "card")
	if err == nil {
		t.Fatal("expected error for empty user id")
	}
	if !errors.Is(err, purchases.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestInitiatePaymentEmptyPurchaseID(t *testing.T) {
	svc := purchases.NewService(
		&stubProgramRepository{},
		&stubPurchaseRepository{},
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.InitiatePayment(context.Background(), "33333333-3333-3333-3333-333333333333", "", "card")
	if err == nil {
		t.Fatal("expected error for empty purchase id")
	}
	if !errors.Is(err, purchases.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestInitiatePaymentPurchaseNotFound(t *testing.T) {
	purchasesRepo := &stubPurchaseRepository{
		findByIDErr: repositories.ErrPurchaseNotFound,
	}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.InitiatePayment(context.Background(), "33333333-3333-3333-3333-333333333333", "nonexistent", "card")
	if err == nil {
		t.Fatal("expected error for nonexistent purchase")
	}
	if !errors.Is(err, purchases.ErrPurchaseNotFound) {
		t.Fatalf("expected ErrPurchaseNotFound, got %v", err)
	}
}

func TestInitiatePaymentRepositoryFailureNotExposed(t *testing.T) {
	purchasesRepo := &stubPurchaseRepository{
		findByIDErr: errors.New("database connection lost"),
	}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.InitiatePayment(context.Background(), "33333333-3333-3333-3333-333333333333", "purchase-001", "card")
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, purchases.ErrPurchaseNotFound) || errors.Is(err, purchases.ErrInvalidInput) {
		t.Fatalf("internal error must not map to user-facing error: %v", err)
	}
}

func TestInitiatePaymentIDOR(t *testing.T) {
	purchase := &models.Purchase{
		ID:              "purchase-001",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 10000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}

	purchasesRepo := &stubPurchaseRepository{
		findByIDPurchase: purchase,
	}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	differentUserID := "44444444-4444-4444-4444-444444444444"
	_, err := svc.InitiatePayment(context.Background(), differentUserID, "purchase-001", "card")
	if err == nil {
		t.Fatal("expected error for IDOR")
	}
	if !errors.Is(err, purchases.ErrPurchaseNotFound) {
		t.Fatalf("expected ErrPurchaseNotFound for IDOR, got %v", err)
	}
}

func TestInitiatePaymentNotPending(t *testing.T) {
	testCases := []struct {
		name   string
		status string
	}{
		{"completed", models.PurchaseStatusCompleted},
		{"failed", models.PurchaseStatusFailed},
		{"refunded", models.PurchaseStatusRefunded},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			purchase := &models.Purchase{
				ID:              "purchase-001",
				UserID:          "33333333-3333-3333-3333-333333333333",
				ProgramID:       "11111111-1111-1111-1111-111111111111",
				PriceMinorUnits: 10000,
				Currency:        "EUR",
				Status:          tc.status,
			}

			purchasesRepo := &stubPurchaseRepository{
				findByIDPurchase: purchase,
			}

			svc := purchases.NewService(
				&stubProgramRepository{},
				purchasesRepo,
				&stubEntitlementRepository{},
				&stubCommissionResolver{},
				&stubPaymentProvider{},
				nil,
			)

			_, err := svc.InitiatePayment(context.Background(), "33333333-3333-3333-3333-333333333333", "purchase-001", "card")
			if err == nil {
				t.Fatal("expected error for non-pending purchase")
			}
			if !errors.Is(err, purchases.ErrPurchaseNotPending) {
				t.Fatalf("expected ErrPurchaseNotPending, got %v", err)
			}
		})
	}
}

func TestInitiatePaymentProviderFailure(t *testing.T) {
	purchase := &models.Purchase{
		ID:              "purchase-001",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 10000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}

	purchasesRepo := &stubPurchaseRepository{
		findByIDPurchase: purchase,
	}
	payment := &stubPaymentProvider{
		err: payments.ErrProviderFailure,
	}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		payment,
		stubResolver(payment),
	)

	_, err := svc.InitiatePayment(context.Background(), "33333333-3333-3333-3333-333333333333", "purchase-001", "card")
	if err == nil {
		t.Fatal("expected error when provider fails")
	}
	if !errors.Is(err, purchases.ErrPaymentProvider) {
		t.Fatalf("expected ErrPaymentProvider, got %v", err)
	}
}

func TestGetPurchaseByIDSuccess(t *testing.T) {
	existing := &models.Purchase{
		ID:              "purchase-001",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 10000,
		Currency:        "EUR",
		CommissionBPS:   2000,
		PlatformAmount:  2000,
		TrainerAmount:   8000,
		Status:          models.PurchaseStatusPending,
	}

	svc := purchases.NewService(
		&stubProgramRepository{},
		&stubPurchaseRepository{findByIDPurchase: existing},
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	purchase, err := svc.GetPurchaseByID(context.Background(), "purchase-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if purchase.ID != "purchase-001" {
		t.Fatalf("expected purchase id purchase-001, got %s", purchase.ID)
	}
	if purchase.PriceMinorUnits != 10000 {
		t.Fatalf("expected price 10000, got %d", purchase.PriceMinorUnits)
	}
	if purchase.Currency != "EUR" {
		t.Fatalf("expected currency EUR, got %s", purchase.Currency)
	}
	if purchase.CommissionBPS != 2000 {
		t.Fatalf("expected commission 2000, got %d", purchase.CommissionBPS)
	}
	if purchase.PlatformAmount != 2000 {
		t.Fatalf("expected platform amount 2000, got %d", purchase.PlatformAmount)
	}
	if purchase.TrainerAmount != 8000 {
		t.Fatalf("expected trainer amount 8000, got %d", purchase.TrainerAmount)
	}
	if purchase.Status != models.PurchaseStatusPending {
		t.Fatalf("expected status pending, got %s", purchase.Status)
	}
}

func TestGetPurchaseByIDEmptyID(t *testing.T) {
	svc := purchases.NewService(
		&stubProgramRepository{},
		&stubPurchaseRepository{},
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.GetPurchaseByID(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty purchase id")
	}
	if !errors.Is(err, purchases.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetPurchaseByIDNotFound(t *testing.T) {
	svc := purchases.NewService(
		&stubProgramRepository{},
		&stubPurchaseRepository{},
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.GetPurchaseByID(context.Background(), "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent purchase")
	}
	if !errors.Is(err, purchases.ErrPurchaseNotFound) {
		t.Fatalf("expected ErrPurchaseNotFound, got %v", err)
	}
}

func TestGetPurchaseByIDRepositoryFailureNotExposed(t *testing.T) {
	svc := purchases.NewService(
		&stubProgramRepository{},
		&stubPurchaseRepository{findByIDErr: errors.New("database connection lost")},
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.GetPurchaseByID(context.Background(), "purchase-001")
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, purchases.ErrPurchaseNotFound) || errors.Is(err, purchases.ErrInvalidInput) {
		t.Fatalf("internal error must not map to user-facing error: %v", err)
	}
}

func TestInitiatePaymentStatusNotMutated(t *testing.T) {
	purchase := &models.Purchase{
		ID:              "purchase-001",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 10000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}

	purchasesRepo := &stubPurchaseRepository{
		findByIDPurchase: purchase,
	}
	payment := &stubPaymentProvider{
		result: payments.PaymentResult{
			PaymentID:   "pay_test",
			Status:      payments.PaymentStatusPending,
			CheckoutURL: "https://checkout.example.com/pay/test",
			Provider:    "fake",
			PurchaseID:  "purchase-001",
		},
	}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		payment,
		stubResolver(payment),
	)

	_, err := svc.InitiatePayment(context.Background(), "33333333-3333-3333-3333-333333333333", "purchase-001", "card")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if purchase.Status != models.PurchaseStatusPending {
		t.Fatalf("InitiatePayment must not mutate purchase status, got %q", purchase.Status)
	}
}

func TestInitiatePaymentProviderFailureDoesNotComplete(t *testing.T) {
	purchase := &models.Purchase{
		ID:              "purchase-001",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 10000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}

	purchasesRepo := &stubPurchaseRepository{
		findByIDPurchase: purchase,
	}
	payment := &stubPaymentProvider{
		err: payments.ErrProviderFailure,
	}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		payment,
		stubResolver(payment),
	)

	_, err := svc.InitiatePayment(context.Background(), "33333333-3333-3333-3333-333333333333", "purchase-001", "card")
	if err == nil {
		t.Fatal("expected error when provider fails")
	}
	if !errors.Is(err, purchases.ErrPaymentProvider) {
		t.Fatalf("expected ErrPaymentProvider, got %v", err)
	}

	if purchase.Status != models.PurchaseStatusPending {
		t.Fatalf("provider failure must not change purchase status, got %q", purchase.Status)
	}
}

func TestCompletePurchaseEntitlementAlreadyActiveDuringPending(t *testing.T) {
	purchase := &models.Purchase{
		ID:              "aaaa1111-1111-1111-1111-111111111111",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 10000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}

	purchasesRepo := &completionPurchaseRepo{findByIDPurchase: purchase}
	activeEntitlement := &models.Entitlement{
		ID:        "bbbb1111-1111-1111-1111-111111111111",
		UserID:    purchase.UserID,
		ProgramID: purchase.ProgramID,
	}
	entitlementsRepo := &completionEntitlementRepo{activeEntitlement: activeEntitlement, restoreErr: repositories.ErrEntitlementNotFound}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		entitlementsRepo,
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.CompletePurchase(context.Background(), purchase.ID)
	if !errors.Is(err, purchases.ErrDuplicateEntitlement) {
		t.Fatalf("expected ErrDuplicateEntitlement when active entitlement exists, got %v", err)
	}
	if purchasesRepo.completedWithEntID != "" {
		t.Fatal("CompleteWithEntitlement must not be called when active entitlement already exists")
	}
}

func TestCompletePurchaseRepositoryLoadFailure(t *testing.T) {
	purchasesRepo := &completionPurchaseRepo{
		findByIDErr: errors.New("database connection lost"),
	}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&completionEntitlementRepo{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.CompletePurchase(context.Background(), "aaaa1111-1111-1111-1111-111111111111")
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, purchases.ErrPurchaseNotFound) || errors.Is(err, purchases.ErrInvalidInput) {
		t.Fatalf("internal error must not map to user-facing error: %v", err)
	}
}

func TestCompletePurchaseEntitlementReactivationFailure(t *testing.T) {
	purchase := &models.Purchase{
		ID:              "aaaa1111-1111-1111-1111-111111111111",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 10000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}

	purchasesRepo := &completionPurchaseRepo{findByIDPurchase: purchase}
	entitlementsRepo := &completionEntitlementRepo{restoreErr: errors.New("database failure")}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		entitlementsRepo,
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.CompletePurchase(context.Background(), purchase.ID)
	if err == nil {
		t.Fatal("expected error when entitlement restoration fails")
	}
	if errors.Is(err, purchases.ErrPurchaseNotFound) || errors.Is(err, purchases.ErrInvalidInput) {
		t.Fatalf("internal error must not map to user-facing error: %v", err)
	}
}

func TestInitiatePaymentUsesSnapshotValues(t *testing.T) {
	purchase := &models.Purchase{
		ID:              "purchase-snapshot",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "program-snapshot",
		PriceMinorUnits: 25000,
		Currency:        "USD",
		Status:          models.PurchaseStatusPending,
	}

	var capturedRequest payments.PaymentRequest
	purchasesRepo := &stubPurchaseRepository{
		findByIDPurchase: purchase,
	}
	payment := &stubPaymentProvider{
		result: payments.PaymentResult{
			PaymentID:   "pay_test",
			Status:      payments.PaymentStatusPending,
			CheckoutURL: "https://checkout.example.com/pay/test",
			Provider:    "fake",
			PurchaseID:  purchase.ID,
		},
		captureRequest: &capturedRequest,
	}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		payment,
		stubResolver(payment),
	)

	_, err := svc.InitiatePayment(context.Background(), "33333333-3333-3333-3333-333333333333", "purchase-snapshot", "card")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedRequest.AmountMinorUnits != 25000 {
		t.Fatalf("expected amount 25000, got %d", capturedRequest.AmountMinorUnits)
	}
	if capturedRequest.Currency != "USD" {
		t.Fatalf("expected currency USD, got %q", capturedRequest.Currency)
	}
	if capturedRequest.ProgramID != "program-snapshot" {
		t.Fatalf("expected program id program-snapshot, got %q", capturedRequest.ProgramID)
	}
	if capturedRequest.PurchaseID != "purchase-snapshot" {
		t.Fatalf("expected purchase id purchase-snapshot, got %q", capturedRequest.PurchaseID)
	}
	if capturedRequest.Method != payments.PaymentMethodCard {
		t.Fatalf("expected method %q, got %q", payments.PaymentMethodCard, capturedRequest.Method)
	}
}

func TestInitiatePaymentMBWayAccepted(t *testing.T) {
	purchase := &models.Purchase{
		ID:              "purchase-mbway",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "program-mbway",
		PriceMinorUnits: 1500,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}

	var capturedRequest payments.PaymentRequest
	purchasesRepo := &stubPurchaseRepository{
		findByIDPurchase: purchase,
	}
	payment := &stubPaymentProvider{
		result: payments.PaymentResult{
			PaymentID:   "pay_mbway_001",
			Status:      payments.PaymentStatusPending,
			CheckoutURL: "https://checkout.example.com/mbway/001",
			Provider:    "fake",
			PurchaseID:  purchase.ID,
		},
		captureRequest: &capturedRequest,
	}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		payment,
		stubResolver(payment),
	)

	result, err := svc.InitiatePayment(context.Background(), "33333333-3333-3333-3333-333333333333", "purchase-mbway", "mbway")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PaymentID != "pay_mbway_001" {
		t.Fatalf("expected payment id %q, got %q", "pay_mbway_001", result.PaymentID)
	}
	if result.CheckoutURL != "https://checkout.example.com/mbway/001" {
		t.Fatalf("expected checkout url, got %q", result.CheckoutURL)
	}
	if capturedRequest.Method != payments.PaymentMethodMBWay {
		t.Fatalf("expected method %q, got %q", payments.PaymentMethodMBWay, capturedRequest.Method)
	}
	if capturedRequest.AmountMinorUnits != 1500 {
		t.Fatalf("expected amount 1500, got %d", capturedRequest.AmountMinorUnits)
	}
	if capturedRequest.Currency != "EUR" {
		t.Fatalf("expected currency EUR, got %q", capturedRequest.Currency)
	}
}

func TestInitiatePaymentPayPalAccepted(t *testing.T) {
	purchase := &models.Purchase{
		ID:              "purchase-paypal",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "program-paypal",
		PriceMinorUnits: 5000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}

	var capturedRequest payments.PaymentRequest
	purchasesRepo := &stubPurchaseRepository{
		findByIDPurchase: purchase,
	}
	payment := &stubPaymentProvider{
		result: payments.PaymentResult{
			PaymentID:   "pay_pal_order_001",
			Status:      payments.PaymentStatusPending,
			CheckoutURL: "https://paypal.com/approve/001",
			Provider:    "fake",
			PurchaseID:  purchase.ID,
		},
		captureRequest: &capturedRequest,
	}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		payment,
		stubResolver(payment),
	)

	result, err := svc.InitiatePayment(context.Background(), "33333333-3333-3333-3333-333333333333", "purchase-paypal", "paypal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PaymentID != "pay_pal_order_001" {
		t.Fatalf("expected payment id %q, got %q", "pay_pal_order_001", result.PaymentID)
	}
	if result.CheckoutURL != "https://paypal.com/approve/001" {
		t.Fatalf("expected checkout url, got %q", result.CheckoutURL)
	}
	if capturedRequest.Method != payments.PaymentMethodPayPal {
		t.Fatalf("expected method %q, got %q", payments.PaymentMethodPayPal, capturedRequest.Method)
	}
	if capturedRequest.AmountMinorUnits != 5000 {
		t.Fatalf("expected amount 5000, got %d", capturedRequest.AmountMinorUnits)
	}
	if capturedRequest.Currency != "EUR" {
		t.Fatalf("expected currency EUR, got %q", capturedRequest.Currency)
	}
	if capturedRequest.ProgramID != "program-paypal" {
		t.Fatalf("expected program id program-paypal, got %q", capturedRequest.ProgramID)
	}
}

func TestInitiatePaymentProviderResolutionError(t *testing.T) {
	purchase := &models.Purchase{
		ID:              "purchase-001",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 10000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}

	purchasesRepo := &stubPurchaseRepository{
		findByIDPurchase: purchase,
	}

	failingResolver := func(_ context.Context, _ payments.PaymentMethod) (payments.Provider, error) {
		return nil, payments.ErrNoProviderAvailable
	}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		failingResolver,
	)

	_, err := svc.InitiatePayment(context.Background(), "33333333-3333-3333-3333-333333333333", "purchase-001", "card")
	if err == nil {
		t.Fatal("expected error when resolver fails")
	}
	if !errors.Is(err, purchases.ErrPaymentProvider) {
		t.Fatalf("expected ErrPaymentProvider, got %v", err)
	}
}

func TestInitiatePaymentMethodCannotAlterSnapshot(t *testing.T) {
	purchase := &models.Purchase{
		ID:              "purchase-snapshot2",
		UserID:          "33333333-3333-3333-3333-333333333333",
		ProgramID:       "program-snapshot2",
		PriceMinorUnits: 9900,
		Currency:        "GBP",
		Status:          models.PurchaseStatusPending,
	}

	purchasesRepo := &stubPurchaseRepository{
		findByIDPurchase: purchase,
	}
	payment := &stubPaymentProvider{
		result: payments.PaymentResult{
			PaymentID:   "pay_test",
			Status:      payments.PaymentStatusPending,
			CheckoutURL: "https://checkout.example.com/pay/test",
			Provider:    "fake",
			PurchaseID:  purchase.ID,
		},
	}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		payment,
		stubResolver(payment),
	)

	methods := []string{"card", "mbway", "paypal"}
	for _, method := range methods {
		result, err := svc.InitiatePayment(context.Background(), "33333333-3333-3333-3333-333333333333", "purchase-snapshot2", method)
		if err != nil {
			t.Fatalf("unexpected error for method %q: %v", method, err)
		}
		if result.PurchaseID != "purchase-snapshot2" {
			t.Fatalf("expected purchase id purchase-snapshot2 for method %q, got %q", method, result.PurchaseID)
		}
	}

	if purchase.Status != models.PurchaseStatusPending {
		t.Fatalf("InitiatePayment must not mutate purchase status, got %q", purchase.Status)
	}
}
