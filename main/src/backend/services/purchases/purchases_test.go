package purchases_test

import (
	"context"
	"errors"
	"fmt"
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
	list             []models.Purchase
	listErr          error
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

func (s *stubPurchaseRepository) ListActiveByUser(_ context.Context, _ string) ([]models.Purchase, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.list, nil
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

func (s *stubPurchaseRepository) CompleteTestPurchase(_ context.Context, purchase *models.Purchase, _ *models.Entitlement) error {
	if s.createErr != nil {
		return s.createErr
	}
	purchase.ID = "00000000-0000-0000-0000-000000000001"
	purchase.CreatedAt = time.Now()
	purchase.UpdatedAt = time.Now()
	s.purchase = purchase
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

// stubCaptureProvider implements both Provider and CaptureProvider so it can be
// resolved by the service and used for capture flows. It records the last
// capture request for assertions.
type stubCaptureProvider struct {
	result         payments.CaptureResult
	captureErr     error
	captureRequest *payments.CaptureRequest
}

func (s *stubCaptureProvider) InitiatePayment(_ context.Context, _ payments.PaymentRequest) (payments.PaymentResult, error) {
	return payments.PaymentResult{}, nil
}

func (s *stubCaptureProvider) CapturePayment(_ context.Context, req payments.CaptureRequest) (payments.CaptureResult, error) {
	if s.captureRequest != nil {
		*s.captureRequest = req
	}
	return s.result, s.captureErr
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
		TrainerID:       "22222222-2222-2222-2222-222222222222",
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
		TrainerID:       "22222222-2222-2222-2222-222222222222",
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

func TestCreatePurchaseIntentPlatformOwnedProgramNoCommission(t *testing.T) {
	program := &models.Program{
		ID:              "11111111-1111-1111-1111-111111111111",
		TrainerID:       "",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 4999,
		Currency:        "EUR",
	}

	// A failing resolver proves commission resolution is never invoked for
	// platform-owned generic programs (trainer_id NULL): the platform keeps
	// the full sale amount.
	commission := &stubCommissionResolver{err: errors.New("must not be called")}

	purchasesRepo := &stubPurchaseRepository{}
	svc := purchases.NewService(
		&stubProgramRepository{program: program},
		purchasesRepo,
		&stubEntitlementRepository{},
		commission,
		&stubPaymentProvider{},
		nil,
	)

	purchase, err := svc.CreatePurchaseIntent(context.Background(), "33333333-3333-3333-3333-333333333333", "11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if purchase.CommissionBPS != 0 {
		t.Fatalf("expected 0 commission bps, got %d", purchase.CommissionBPS)
	}
	if purchase.PlatformAmount != 4999 {
		t.Fatalf("expected platform amount 4999, got %d", purchase.PlatformAmount)
	}
	if purchase.TrainerAmount != 0 {
		t.Fatalf("expected trainer amount 0, got %d", purchase.TrainerAmount)
	}
	if purchasesRepo.purchase == nil {
		t.Fatal("expected a persisted purchase")
	}
}

func TestListPurchasesScopedToUser(t *testing.T) {
	now := time.Now()
	records := []models.Purchase{
		{
			ID:              "aaaa1111-1111-1111-1111-111111111111",
			UserID:          "33333333-3333-3333-3333-333333333333",
			ProgramID:       "11111111-1111-1111-1111-111111111111",
			PriceMinorUnits: 4999,
			Currency:        "EUR",
			Status:          models.PurchaseStatusPending,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			ID:              "bbbb2222-2222-2222-2222-222222222222",
			UserID:          "33333333-3333-3333-3333-333333333333",
			ProgramID:       "22222222-2222-2222-2222-222222222222",
			PriceMinorUnits: 8999,
			Currency:        "EUR",
			Status:          models.PurchaseStatusCompleted,
			CreatedAt:       now.Add(-time.Hour),
			UpdatedAt:       now.Add(-time.Hour),
		},
	}

	purchasesRepo := &stubPurchaseRepository{list: records}
	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	list, err := svc.ListPurchases(context.Background(), "33333333-3333-3333-3333-333333333333")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 purchases, got %d", len(list))
	}
	if list[0].ID != records[0].ID || list[0].ProgramID != records[0].ProgramID {
		t.Fatalf("expected record order preserved, got %+v", list[0])
	}
	if list[0].UserID != records[0].UserID {
		t.Fatalf("expected user id preserved internally, got %q", list[0].UserID)
	}
	if list[0].Status != models.PurchaseStatusPending || list[1].Status != models.PurchaseStatusCompleted {
		t.Fatalf("unexpected statuses: %q, %q", list[0].Status, list[1].Status)
	}
}

func TestListPurchasesEmptyUserID(t *testing.T) {
	svc := purchases.NewService(
		&stubProgramRepository{},
		&stubPurchaseRepository{},
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	if _, err := svc.ListPurchases(context.Background(), ""); !errors.Is(err, purchases.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestListPurchasesRepositoryFailure(t *testing.T) {
	purchasesRepo := &stubPurchaseRepository{listErr: errors.New("db failure")}
	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		&stubPaymentProvider{},
		nil,
	)

	_, err := svc.ListPurchases(context.Background(), "33333333-3333-3333-3333-333333333333")
	if err == nil {
		t.Fatal("expected error for repository failure")
	}
	if errors.Is(err, purchases.ErrInvalidInput) {
		t.Fatalf("got user-facing error instead of internal error: %v", err)
	}
}

func TestCreatePurchaseIntentOverrideCommission(t *testing.T) {
	program := &models.Program{
		ID:              "11111111-1111-1111-1111-111111111111",
		TrainerID:       "22222222-2222-2222-2222-222222222222",
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
func (r *completionPurchaseRepo) ListActiveByUser(_ context.Context, _ string) ([]models.Purchase, error) {
	return nil, nil
}
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

func (r *completionPurchaseRepo) CompleteTestPurchase(_ context.Context, purchase *models.Purchase, entitlement *models.Entitlement) error {
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

// --- capture tests ---

func TestCapturePaymentSuccess(t *testing.T) {
	pending := &models.Purchase{
		ID:              "purchase-capture-success",
		UserID:          "22222222-2222-2222-2222-222222222222",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 2500,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}
	purchasesRepo := &stubPurchaseRepository{findByIDPurchase: pending}
	entitlements := &stubEntitlementRepository{}
	var captured payments.CaptureRequest
	provider := &stubCaptureProvider{
		result:         payments.CaptureResult{PaymentID: "paypal-order-123", Provider: "paypal"},
		captureRequest: &captured,
	}

	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		entitlements,
		&stubCommissionResolver{},
		provider,
		stubResolver(provider),
	)

	result, err := svc.CapturePayment(context.Background(), "22222222-2222-2222-2222-222222222222", pending.ID, "paypal-order-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != models.PurchaseStatusCompleted {
		t.Fatalf("purchase must be completed after capture, got %q", result.Status)
	}
	if captured.PurchaseID != pending.ID {
		t.Fatalf("capture request must reference the purchase, got %q", captured.PurchaseID)
	}
	if captured.PaymentID != "paypal-order-123" {
		t.Fatalf("capture request must carry the provider payment id, got %q", captured.PaymentID)
	}
	if captured.AmountMinorUnits != 2500 {
		t.Fatalf("capture request must carry the immutable amount, got %d", captured.AmountMinorUnits)
	}
	if captured.Currency != "EUR" {
		t.Fatalf("capture request must carry the immutable currency, got %q", captured.Currency)
	}
}

func TestCapturePaymentOwnershipEnforced(t *testing.T) {
	pending := &models.Purchase{
		ID:              "purchase-owned-by-other",
		UserID:          "99999999-9999-9999-9999-999999999999",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 1000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}
	purchasesRepo := &stubPurchaseRepository{findByIDPurchase: pending}
	provider := &stubCaptureProvider{}
	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		provider,
		stubResolver(provider),
	)

	_, err := svc.CapturePayment(context.Background(), "22222222-2222-2222-2222-222222222222", pending.ID, "paypal-order-123")
	if !errors.Is(err, purchases.ErrPurchaseNotFound) {
		t.Fatalf("ownership mismatch must surface as ErrPurchaseNotFound, got %v", err)
	}
}

func TestCapturePaymentUnknownPurchase(t *testing.T) {
	purchasesRepo := &stubPurchaseRepository{findByIDErr: repositories.ErrPurchaseNotFound}
	provider := &stubCaptureProvider{}
	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		provider,
		stubResolver(provider),
	)

	_, err := svc.CapturePayment(context.Background(), "22222222-2222-2222-2222-222222222222", "purchase-missing", "paypal-order-123")
	if !errors.Is(err, purchases.ErrPurchaseNotFound) {
		t.Fatalf("missing purchase must surface as ErrPurchaseNotFound, got %v", err)
	}
}

func TestCapturePaymentIdempotentForCompletedPurchase(t *testing.T) {
	completed := &models.Purchase{
		ID:              "purchase-already-completed",
		UserID:          "22222222-2222-2222-2222-222222222222",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 1000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusCompleted,
	}
	purchasesRepo := &stubPurchaseRepository{findByIDPurchase: completed}
	entitlements := &stubEntitlementRepository{
		existing: &models.Entitlement{UserID: completed.UserID, ProgramID: completed.ProgramID},
	}
	// The capture provider records the request so an idempotent re-capture can
	// be detected.
	var captured payments.CaptureRequest
	provider := &stubCaptureProvider{captureRequest: &captured}
	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		entitlements,
		&stubCommissionResolver{},
		provider,
		stubResolver(provider),
	)

	result, err := svc.CapturePayment(context.Background(), completed.UserID, completed.ID, "paypal-order-123")
	if err != nil {
		t.Fatalf("re-capture of a completed purchase must succeed idempotently, got %v", err)
	}
	if result.Status != models.PurchaseStatusCompleted {
		t.Fatalf("expected completed purchase, got %q", result.Status)
	}
	if captured.PurchaseID != "" {
		t.Fatalf("idempotent re-capture must not call the provider, got capture request %+v", captured)
	}
}

func TestCapturePaymentCompletedWithoutEntitlement(t *testing.T) {
	completed := &models.Purchase{
		ID:              "purchase-completed-no-entitlement",
		UserID:          "22222222-2222-2222-2222-222222222222",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 1000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusCompleted,
	}
	purchasesRepo := &stubPurchaseRepository{findByIDPurchase: completed}
	provider := &stubCaptureProvider{}
	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		provider,
		stubResolver(provider),
	)

	_, err := svc.CapturePayment(context.Background(), completed.UserID, completed.ID, "paypal-order-123")
	if !errors.Is(err, purchases.ErrEntitlementIntegrity) {
		t.Fatalf("completed purchase without entitlement must surface as ErrEntitlementIntegrity, got %v", err)
	}
}

func TestCapturePaymentRejectsNonPendingPurchase(t *testing.T) {
	failed := &models.Purchase{
		ID:              "purchase-failed",
		UserID:          "22222222-2222-2222-2222-222222222222",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 1000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusFailed,
	}
	purchasesRepo := &stubPurchaseRepository{findByIDPurchase: failed}
	provider := &stubCaptureProvider{}
	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		provider,
		stubResolver(provider),
	)

	_, err := svc.CapturePayment(context.Background(), failed.UserID, failed.ID, "paypal-order-123")
	if !errors.Is(err, purchases.ErrPurchaseNotPending) {
		t.Fatalf("failed purchase must surface as ErrPurchaseNotPending, got %v", err)
	}
}

func TestCapturePaymentRejectsEmptyProviderPaymentID(t *testing.T) {
	pending := &models.Purchase{
		ID:              "purchase-empty-payment-id",
		UserID:          "22222222-2222-2222-2222-222222222222",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 1000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}
	purchasesRepo := &stubPurchaseRepository{findByIDPurchase: pending}
	provider := &stubCaptureProvider{}
	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		provider,
		stubResolver(provider),
	)

	_, err := svc.CapturePayment(context.Background(), pending.UserID, pending.ID, "")
	if !errors.Is(err, purchases.ErrInvalidInput) {
		t.Fatalf("empty provider payment id must surface as ErrInvalidInput, got %v", err)
	}
}

func TestCapturePaymentProviderFailure(t *testing.T) {
	pending := &models.Purchase{
		ID:              "purchase-provider-failure",
		UserID:          "22222222-2222-2222-2222-222222222222",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 1000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}
	purchasesRepo := &stubPurchaseRepository{findByIDPurchase: pending}
	provider := &stubCaptureProvider{
		captureErr: fmt.Errorf("provider refused capture: %w", payments.ErrProviderFailure),
	}
	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		provider,
		stubResolver(provider),
	)

	_, err := svc.CapturePayment(context.Background(), pending.UserID, pending.ID, "paypal-order-123")
	if !errors.Is(err, purchases.ErrPaymentProvider) {
		t.Fatalf("provider failure must surface as ErrPaymentProvider, got %v", err)
	}
	if pending.Status != models.PurchaseStatusPending {
		t.Fatalf("failed capture must never complete the purchase, got status %q", pending.Status)
	}
}

func TestCapturePaymentResolverRejectsNonCaptureProvider(t *testing.T) {
	pending := &models.Purchase{
		ID:              "purchase-no-capture-provider",
		UserID:          "22222222-2222-2222-2222-222222222222",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 1000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}
	purchasesRepo := &stubPurchaseRepository{findByIDPurchase: pending}
	// A provider that implements only InitiatePayment cannot capture.
	payment := &stubPaymentProvider{}
	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		payment,
		stubResolver(payment),
	)

	_, err := svc.CapturePayment(context.Background(), pending.UserID, pending.ID, "paypal-order-123")
	if !errors.Is(err, purchases.ErrPaymentProvider) {
		t.Fatalf("non-capture provider must surface as ErrPaymentProvider, got %v", err)
	}
}

func TestCompletePurchaseWithCaptureSuccess(t *testing.T) {
	pending := &models.Purchase{
		ID:              "purchase-webhook-capture",
		UserID:          "22222222-2222-2222-2222-222222222222",
		ProgramID:       "11111111-1111-1111-1111-111111111111",
		PriceMinorUnits: 2500,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}
	purchasesRepo := &stubPurchaseRepository{findByIDPurchase: pending}
	entitlements := &stubEntitlementRepository{}
	var captured payments.CaptureRequest
	provider := &stubCaptureProvider{
		result:         payments.CaptureResult{PaymentID: "paypal-order-456", Provider: "paypal"},
		captureRequest: &captured,
	}
	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		entitlements,
		&stubCommissionResolver{},
		provider,
		stubResolver(provider),
	)

	result, err := svc.CompletePurchaseWithCapture(context.Background(), pending.ID, "paypal-order-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != models.PurchaseStatusCompleted {
		t.Fatalf("purchase must be completed after webhook capture, got %q", result.Status)
	}
	if captured.PaymentID != "paypal-order-456" {
		t.Fatalf("capture request must carry the verified order id, got %q", captured.PaymentID)
	}
}

func TestCompletePurchaseWithCaptureUnknownPurchase(t *testing.T) {
	purchasesRepo := &stubPurchaseRepository{findByIDErr: repositories.ErrPurchaseNotFound}
	provider := &stubCaptureProvider{}
	svc := purchases.NewService(
		&stubProgramRepository{},
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		provider,
		stubResolver(provider),
	)

	_, err := svc.CompletePurchaseWithCapture(context.Background(), "purchase-missing", "paypal-order-456")
	if !errors.Is(err, purchases.ErrPurchaseNotFound) {
		t.Fatalf("missing purchase must surface as ErrPurchaseNotFound, got %v", err)
	}
}

func TestCompleteTestPurchaseSuccess(t *testing.T) {
	programs := &stubProgramRepository{
		program: &models.Program{
			ID:              "program-test-1",
			Name:            "Strength Builder",
			Type:            models.ProgramTypePremium,
			Status:          models.ProgramStatusPublished,
			PriceMinorUnits: 4999,
			Currency:        "EUR",
			TrainerID:       "trainer-test-1",
		},
	}
	purchasesRepo := &stubPurchaseRepository{}
	svc := purchases.NewService(
		programs,
		purchasesRepo,
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		nil,
		nil,
	)

	result, err := svc.CompleteTestPurchase(context.Background(), "user-test-1", "program-test-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != models.PurchaseStatusCompleted {
		t.Fatalf("expected a completed test purchase, got %q", result.Status)
	}
	if result.PriceMinorUnits != 0 {
		t.Fatalf("expected a zero test price, got %d", result.PriceMinorUnits)
	}
	if result.PlatformAmount != 0 || result.TrainerAmount != 0 {
		t.Fatalf("expected no commission split for a test purchase, got platform=%d trainer=%d", result.PlatformAmount, result.TrainerAmount)
	}
	if !purchasesRepo.purchase.Test {
		t.Fatal("expected the persisted purchase to carry the test marker")
	}
	// The test path must never invoke a payment provider. The service was built
	// with a nil provider and resolver: reaching completion proves no payment
	// interaction happened.
	if purchasesRepo.purchase.Status != models.PurchaseStatusCompleted {
		t.Fatalf("expected the persisted purchase to be completed, got %q", purchasesRepo.purchase.Status)
	}
}

func TestCompleteTestPurchaseInvalidInput(t *testing.T) {
	svc := purchases.NewService(&stubProgramRepository{}, &stubPurchaseRepository{}, &stubEntitlementRepository{}, &stubCommissionResolver{}, nil, nil)

	if _, err := svc.CompleteTestPurchase(context.Background(), "", "program-test-1"); !errors.Is(err, purchases.ErrInvalidInput) {
		t.Fatalf("missing user must surface as ErrInvalidInput, got %v", err)
	}
	if _, err := svc.CompleteTestPurchase(context.Background(), "user-test-1", ""); !errors.Is(err, purchases.ErrInvalidInput) {
		t.Fatalf("missing program must surface as ErrInvalidInput, got %v", err)
	}
}

func TestCompleteTestPurchaseProgramNotFound(t *testing.T) {
	svc := purchases.NewService(
		&stubProgramRepository{err: repositories.ErrProgramNotFound},
		&stubPurchaseRepository{},
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		nil,
		nil,
	)

	_, err := svc.CompleteTestPurchase(context.Background(), "user-test-1", "program-missing")
	if !errors.Is(err, purchases.ErrProgramNotFound) {
		t.Fatalf("missing program must surface as ErrProgramNotFound, got %v", err)
	}
}

func TestCompleteTestPurchaseRejectsFreeProgram(t *testing.T) {
	svc := purchases.NewService(
		&stubProgramRepository{
			program: &models.Program{ID: "program-free", Type: models.ProgramTypeFree, Status: models.ProgramStatusPublished},
		},
		&stubPurchaseRepository{},
		&stubEntitlementRepository{},
		&stubCommissionResolver{},
		nil,
		nil,
	)

	_, err := svc.CompleteTestPurchase(context.Background(), "user-test-1", "program-free")
	if !errors.Is(err, purchases.ErrProgramNotPurchasable) {
		t.Fatalf("a free program must never be purchasable, got %v", err)
	}
}

func TestCompleteTestPurchaseDuplicateEntitlement(t *testing.T) {
	programs := &stubProgramRepository{
		program: &models.Program{ID: "program-test-1", Type: models.ProgramTypePremium, Status: models.ProgramStatusPublished},
	}
	entitlements := &stubEntitlementRepository{existing: &models.Entitlement{ID: "entitlement-existing"}}
	svc := purchases.NewService(programs, &stubPurchaseRepository{}, entitlements, &stubCommissionResolver{}, nil, nil)

	_, err := svc.CompleteTestPurchase(context.Background(), "user-test-1", "program-test-1")
	if !errors.Is(err, purchases.ErrDuplicateEntitlement) {
		t.Fatalf("an existing entitlement must surface as ErrDuplicateEntitlement, got %v", err)
	}
}

func TestCompleteTestPurchaseDuplicatePersistence(t *testing.T) {
	for name, createErr := range map[string]error{
		"existing completed purchase": repositories.ErrCompletedPurchaseExists,
		"existing entitlement":        repositories.ErrEntitlementAlreadyExists,
	} {
		t.Run(name, func(t *testing.T) {
			programs := &stubProgramRepository{
				program: &models.Program{ID: "program-test-1", Type: models.ProgramTypePremium, Status: models.ProgramStatusPublished},
			}
			svc := purchases.NewService(programs, &stubPurchaseRepository{createErr: createErr}, &stubEntitlementRepository{}, &stubCommissionResolver{}, nil, nil)

			_, err := svc.CompleteTestPurchase(context.Background(), "user-test-1", "program-test-1")
			if !errors.Is(err, purchases.ErrDuplicateEntitlement) {
				t.Fatalf("persistence duplicate must surface as ErrDuplicateEntitlement, got %v", err)
			}
		})
	}
}
