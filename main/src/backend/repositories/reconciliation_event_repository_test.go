package repositories_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

// openReconciliationTestRepository connects a fresh transaction-scoped
// repository for the reconciliation event tests.
func openReconciliationTestRepository(t *testing.T) (*gorm.DB, repositories.ReconciliationEventRepository, func()) {
	t.Helper()
	config.LoadEnvFile()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}

	tx := db.Begin()
	repo := repositories.NewReconciliationEventRepository(tx)
	return tx, repo, func() { tx.Rollback() }
}

// seedReconciliationPurchase creates a minimal published program and returns
// the purchase used as the reconciliation target.
func seedReconciliationPurchase(t *testing.T, db *gorm.DB) *models.Purchase {
	t.Helper()
	ctx := context.Background()

	userRepo := repositories.NewUserRepository(db)
	user := &models.User{
		Email:        fmt.Sprintf("recon-repo-%d@ryze.local", time.Now().UnixNano()),
		PasswordHash: "prepared-hash-outside-repository-scope",
		FirstName:    "Recon",
		LastName:     "Client",
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	programRepo := repositories.NewGenericProgramRepository(db)
	program := &models.Program{
		Name:            "Reconciliation Repo Program",
		Description:     "Description",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 4999,
		Currency:        string(models.ProgramCurrencyEUR),
	}
	if err := programRepo.CreateFull(ctx, program); err != nil {
		t.Fatalf("create program: %v", err)
	}

	purchaseRepo := repositories.NewPurchaseRepository(db)
	purchase := &models.Purchase{
		UserID:          user.ID,
		ProgramID:       program.ID,
		PriceMinorUnits: 4999,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}
	if err := purchaseRepo.Create(ctx, purchase); err != nil {
		t.Fatalf("create purchase: %v", err)
	}
	return purchase
}

// TestReconciliationEventRepositoryUpsertProvisionalOverwrite proves a
// provider retry can upgrade a provisional outcome (transient_failure) to its
// final verdict, turning a delivery that needed investigation into a clean
// settlement.
func TestReconciliationEventRepositoryUpsertProvisionalOverwrite(t *testing.T) {
	db, repo, done := openReconciliationTestRepository(t)
	defer done()
	ctx := context.Background()

	purchase := seedReconciliationPurchase(t, db)

	if err := repo.Record(ctx, purchase.ID, "stripe", "evt-a", models.ReconciliationOutcomeTransientFailure); err != nil {
		t.Fatalf("record provisional: %v", err)
	}
	if err := repo.Record(ctx, purchase.ID, "stripe", "evt-a", models.ReconciliationOutcomeReconciled); err != nil {
		t.Fatalf("record overwrite: %v", err)
	}

	final, total, err := repo.ListAdmin(ctx, repositories.AdminReconciliationFilter{Outcome: models.ReconciliationOutcomeReconciled}, 1, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected exactly one reconciled record, got total=%d", total)
	}
	if len(final) != 1 || final[0].Outcome != models.ReconciliationOutcomeReconciled || final[0].EventID != "evt-a" {
		t.Fatalf("expected the provisional outcome to be replaced, got %+v", final)
	}
}

// TestReconciliationEventRepositoryUpsertFinalOutcomeIsImmutable proves a
// resolved verdict survives any redelivery: after a clean settlement a retry
// carrying a transient verdict must never downgrade the audit history.
func TestReconciliationEventRepositoryUpsertFinalOutcomeIsImmutable(t *testing.T) {
	db, repo, done := openReconciliationTestRepository(t)
	defer done()
	ctx := context.Background()

	purchase := seedReconciliationPurchase(t, db)

	if err := repo.Record(ctx, purchase.ID, "paypal", "evt-b", models.ReconciliationOutcomeCompleted); err != nil {
		t.Fatalf("record completed: %v", err)
	}
	if err := repo.Record(ctx, purchase.ID, "paypal", "evt-b", models.ReconciliationOutcomeTransientFailure); err != nil {
		t.Fatalf("record downgrade attempt: %v", err)
	}

	final, _, err := repo.ListAdmin(ctx, repositories.AdminReconciliationFilter{Outcome: models.ReconciliationOutcomeCompleted}, 1, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(final) != 1 || final[0].Outcome != models.ReconciliationOutcomeCompleted {
		t.Fatalf("expected the final outcome to remain completed, got %+v", final)
	}
}

// TestReconciliationEventRepositoryUpsertInvalidOutcomeRejected proves the
// fixed outcome vocabulary is enforced at the persistence layer: no code path
// can write an undocumented outcome.
func TestReconciliationEventRepositoryUpsertInvalidOutcomeRejected(t *testing.T) {
	_, repo, done := openReconciliationTestRepository(t)
	defer done()
	ctx := context.Background()

	err := repo.Record(ctx, "purchase-c", "stripe", "evt-c", "admin_forced_paid")
	if err == nil {
		t.Fatal("expected an invalid outcome to be rejected")
	}
	if !errors.Is(err, repositories.ErrInvalidReconciliationOutcome) {
		t.Fatalf("expected ErrInvalidReconciliationOutcome, got %v", err)
	}
}

// TestReconciliationEventRepositoryUpsertMissingFieldsRejected proves records
// missing required fields never reach the database.
func TestReconciliationEventRepositoryUpsertMissingFieldsRejected(t *testing.T) {
	_, repo, done := openReconciliationTestRepository(t)
	defer done()
	ctx := context.Background()

	if err := repo.Record(ctx, "", "stripe", "evt-d", models.ReconciliationOutcomeCompleted); err == nil {
		t.Fatal("expected a missing purchase_id to be rejected")
	}
	if err := repo.Record(ctx, "purchase-d", "stripe", "", models.ReconciliationOutcomeCompleted); err == nil {
		t.Fatal("expected a missing event_id to be rejected")
	}
}

// TestReconciliationEventRepositoryListAdminFiltersAndPagination proves the
// administrative read surface: open scope by default narrows to investigations,
// an exact outcome filters precisely, the provider narrows either case and
// pagination pages over the newest records first while preloading the purchase,
// customer and program.
func TestReconciliationEventRepositoryListAdminFiltersAndPagination(t *testing.T) {
	db, repo, done := openReconciliationTestRepository(t)
	defer done()
	ctx := context.Background()

	purchase := seedReconciliationPurchase(t, db)

	// Two provisional/needing-investigation outcomes and one clean settlement.
	clean := time.Now().UTC().Add(-3 * time.Hour)
	if err := repo.Record(ctx, purchase.ID, "stripe", "evt-open-1", models.ReconciliationOutcomeAmountMismatch); err != nil {
		t.Fatalf("record amount_mismatch: %v", err)
	}
	if err := repo.Record(ctx, purchase.ID, "paypal", "evt-open-2", models.ReconciliationOutcomeTransientFailure); err != nil {
		t.Fatalf("record transient: %v", err)
	}
	if db.Exec("UPDATE purchase_reconciliation_events SET created_at = ? WHERE event_id = ?", clean, "evt-open-2").Error != nil {
		t.Fatalf("backdate transient event: %v", db.Error)
	}
	if err := repo.Record(ctx, purchase.ID, "stripe", "evt-clean-1", models.ReconciliationOutcomeCompleted); err != nil {
		t.Fatalf("record completed: %v", err)
	}

	open, total, err := repo.ListAdmin(ctx, repositories.AdminReconciliationFilter{Open: true}, 1, 10)
	if err != nil {
		t.Fatalf("list open: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 open outcomes, got %d", total)
	}
	if len(open) != 2 {
		t.Fatalf("expected 2 open events, got %d", len(open))
	}
	// Newest first: evt-open-1 recorded after evt-open-2.
	if open[0].EventID != "evt-open-1" || open[1].EventID != "evt-open-2" {
		t.Fatalf("expected open events in newest-first order, got %s, %s", open[0].EventID, open[1].EventID)
	}

	byOutcome, total, err := repo.ListAdmin(ctx, repositories.AdminReconciliationFilter{Outcome: models.ReconciliationOutcomeCompleted}, 1, 10)
	if err != nil {
		t.Fatalf("list by outcome: %v", err)
	}
	if total != 1 || len(byOutcome) != 1 || byOutcome[0].EventID != "evt-clean-1" {
		t.Fatalf("unexpected outcome-filtered result: total=%d len=%d", total, len(byOutcome))
	}

	byProvider, total, err := repo.ListAdmin(ctx, repositories.AdminReconciliationFilter{Open: true, Provider: "stripe"}, 1, 10)
	if err != nil {
		t.Fatalf("list open stripe: %v", err)
	}
	if total != 1 || len(byProvider) != 1 || byProvider[0].EventID != "evt-open-1" {
		t.Fatalf("unexpected provider-filtered result: total=%d len=%d", total, len(byProvider))
	}

	page1, total, err := repo.ListAdmin(ctx, repositories.AdminReconciliationFilter{}, 1, 1)
	if err != nil {
		t.Fatalf("list page: %v", err)
	}
	if total != 3 || len(page1) != 1 || page1[0].EventID != "evt-clean-1" {
		t.Fatalf("unexpected pagination: total=%d len=%d first=%s", total, len(page1), page1[0].EventID)
	}
}

// TestReconciliationEventRepositoryListAdminPreloadsPurchase proves the
// administrative record resolves the purchase, customer and program so the
// admin view can present the impact of an outcome without exposing provider
// internals.
func TestReconciliationEventRepositoryListAdminPreloadsPurchase(t *testing.T) {
	db, repo, done := openReconciliationTestRepository(t)
	defer done()
	ctx := context.Background()

	purchase := seedReconciliationPurchase(t, db)
	if err := repo.Record(ctx, purchase.ID, "stripe", "evt-preload", models.ReconciliationOutcomeEntitlementIntegrity); err != nil {
		t.Fatalf("record: %v", err)
	}

	found, _, err := repo.ListAdmin(ctx, repositories.AdminReconciliationFilter{Outcome: models.ReconciliationOutcomeEntitlementIntegrity}, 1, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("expected 1 event, got %d", len(found))
	}
	event := &found[0]
	if event.Purchase == nil {
		t.Fatal("expected the purchase to be preloaded")
	}
	if event.Purchase.ID != purchase.ID {
		t.Fatalf("expected purchase %s, got %s", purchase.ID, event.Purchase.ID)
	}
	if event.Purchase.User.ID == "" {
		t.Fatal("expected the customer to be preloaded")
	}
	if event.Purchase.Program.ID == "" {
		t.Fatal("expected the program to be preloaded")
	}
}