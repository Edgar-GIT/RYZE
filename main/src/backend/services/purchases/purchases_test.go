package purchases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
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
	purchase  *models.Purchase
	existing  *models.Purchase
	createErr error
	findErr   error
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
	return nil, repositories.ErrPurchaseNotFound
}

func (s *stubPurchaseRepository) FindActiveByUserAndProgram(_ context.Context, _, _ string) (*models.Purchase, error) {
	if s.existing != nil {
		return s.existing, s.findErr
	}
	return nil, repositories.ErrPurchaseNotFound
}

type stubEntitlementRepository struct {
	existing *models.Entitlement
	err      error
}

func (s *stubEntitlementRepository) FindActiveByUserAndProgram(_ context.Context, _, _ string) (*models.Entitlement, error) {
	if s.existing != nil {
		return s.existing, s.err
	}
	return nil, repositories.ErrEntitlementNotFound
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

	svc := purchases.NewService(programs, purchasesRepo, entitlements, commission)

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
