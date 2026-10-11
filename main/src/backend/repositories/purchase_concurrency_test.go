package repositories_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

// openConcurrencyRepository connects the pooled database (not a transaction) so
// each repository call runs in its own database transaction and the concurrent
// invocations genuinely race against each other. The pool is capped at 25 open
// connections, comfortably above the goroutine counts used here.
func openConcurrencyRepository(t *testing.T) (*gorm.DB, repositories.PurchaseRepository, func()) {
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

	repo := repositories.NewPurchaseRepository(db)
	cleanup := func() {
		sqlDB, sqlErr := db.DB()
		if sqlErr == nil {
			_ = sqlDB.Close()
		}
	}
	return db, repo, cleanup
}

// seedConcurrencyPair commits a user and a published program for the purchase
// concurrency tests, returning their ids.
func seedConcurrencyPair(t *testing.T, db *gorm.DB) (string, string) {
	t.Helper()
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	user := &models.User{
		Email:        "concurrency-" + suffix + "@ryze.local",
		PasswordHash: "prepared-hash-outside-repository-scope",
		FirstName:    "Concurrent",
		LastName:     "Client",
	}
	userRepo := repositories.NewUserRepository(db)
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	program := &models.Program{
		Name:            "Concurrency Program " + suffix,
		Description:     "Description",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 4999,
		Currency:        string(models.ProgramCurrencyEUR),
	}
	programRepo := repositories.NewGenericProgramRepository(db)
	if err := programRepo.CreateFull(ctx, program); err != nil {
		t.Fatalf("create program: %v", err)
	}
	return user.ID, program.ID
}

// cleanupConcurrencyPair removes every record the concurrent tests created, in
// dependency order, so the shared development database stays clean.
func cleanupConcurrencyPair(t *testing.T, db *gorm.DB, userID, programID string) {
	t.Helper()
	db.Exec("DELETE FROM purchase_reconciliation_events WHERE purchase_id IN (SELECT id FROM purchases WHERE user_id = ? AND program_id = ?)", userID, programID)
	db.Exec("DELETE FROM entitlements WHERE user_id = ? AND program_id = ?", userID, programID)
	db.Exec("DELETE FROM purchases WHERE user_id = ? AND program_id = ?", userID, programID)
	db.Exec("DELETE FROM programs WHERE id = ?", programID)
	db.Exec("DELETE FROM users WHERE id = ?", userID)
}

// TestReconcileFailedPurchaseConcurrentSingleWinner proves the reconciliation
// guard is linearizable under real database contention: when several verified
// settlements race for the same failed (user, program) pair, exactly one wins
// and every loser is rejected. This is the durability guarantee behind the
// "no double count, no double entitlement" financial invariant.
func TestReconcileFailedPurchaseConcurrentSingleWinner(t *testing.T) {
	db, repo, cleanup := openConcurrencyRepository(t)
	t.Cleanup(cleanup)

	userID, programID := seedConcurrencyPair(t, db)
	t.Cleanup(func() { cleanupConcurrencyPair(t, db, userID, programID) })

	ctx := context.Background()
	failed := &models.Purchase{
		UserID:          userID,
		ProgramID:       programID,
		PriceMinorUnits: 4999,
		Currency:        "EUR",
		Status:          models.PurchaseStatusFailed,
	}
	if err := repo.Create(ctx, failed); err != nil {
		t.Fatalf("create failed purchase: %v", err)
	}

	const workers = 4
	results := make([]error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = repo.ReconcileFailedPurchase(ctx, failed.ID, userID, programID, &models.Entitlement{UserID: userID, ProgramID: programID})
		}(i)
	}
	wg.Wait()

	successes := 0
	for i, err := range results {
		if err == nil {
			successes++
			continue
		}
		if !errors.Is(err, repositories.ErrCompletedPurchaseExists) && !errors.Is(err, repositories.ErrEntitlementAlreadyExists) && !errors.Is(err, repositories.ErrPurchaseNotFound) {
			t.Fatalf("worker %d: expected a guarded rejection, got %v", i, err)
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly one winning reconciliation, got %d (results: %+v)", successes, results)
	}

	var purchase models.Purchase
	if err := db.Where("id = ?", failed.ID).First(&purchase).Error; err != nil {
		t.Fatalf("load purchase: %v", err)
	}
	if purchase.Status != models.PurchaseStatusCompleted {
		t.Fatalf("the winning reconciliation must complete the purchase, got %s", purchase.Status)
	}

	var completedCount, entitlementCount int64
	if err := db.Model(&models.Purchase{}).Where("user_id = ? AND program_id = ? AND status = ?", userID, programID, models.PurchaseStatusCompleted).Count(&completedCount).Error; err != nil {
		t.Fatalf("count completed purchases: %v", err)
	}
	if completedCount != 1 {
		t.Fatalf("a (user, program) pair must never have more than one completed purchase, got %d", completedCount)
	}
	if err := db.Model(&models.Entitlement{}).Where("user_id = ? AND program_id = ? AND deleted_at IS NULL", userID, programID).Count(&entitlementCount).Error; err != nil {
		t.Fatalf("count entitlements: %v", err)
	}
	if entitlementCount != 1 {
		t.Fatalf("a (user, program) pair must never have more than one active entitlement, got %d", entitlementCount)
	}
}

// TestCompletePendingPurchaseConcurrentSingleWinner proves the completion path
// is single-writer under real database contention: concurrent completions of
// the same pending purchase leave exactly one completed purchase and one
// entitlement, and the losers are rejected without side effects.
func TestCompletePendingPurchaseConcurrentSingleWinner(t *testing.T) {
	db, repo, cleanup := openConcurrencyRepository(t)
	t.Cleanup(cleanup)

	userID, programID := seedConcurrencyPair(t, db)
	t.Cleanup(func() { cleanupConcurrencyPair(t, db, userID, programID) })

	ctx := context.Background()
	pending := &models.Purchase{
		UserID:          userID,
		ProgramID:       programID,
		PriceMinorUnits: 4999,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}
	if err := repo.Create(ctx, pending); err != nil {
		t.Fatalf("create pending purchase: %v", err)
	}

	const workers = 4
	results := make([]error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = repo.CompleteWithEntitlement(ctx, pending.ID, &models.Entitlement{UserID: userID, ProgramID: programID})
		}(i)
	}
	wg.Wait()

	successes := 0
	for i, err := range results {
		if err == nil {
			successes++
			continue
		}
		if !errors.Is(err, repositories.ErrPurchaseNotFound) && !errors.Is(err, repositories.ErrEntitlementAlreadyExists) {
			t.Fatalf("worker %d: expected a guarded rejection, got %v", i, err)
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly one winning completion, got %d (results: %+v)", successes, results)
	}

	var completedCount, entitlementCount int64
	if err := db.Model(&models.Purchase{}).Where("user_id = ? AND program_id = ? AND status = ?", userID, programID, models.PurchaseStatusCompleted).Count(&completedCount).Error; err != nil {
		t.Fatalf("count completed purchases: %v", err)
	}
	if completedCount != 1 {
		t.Fatalf("exactly one completed purchase must survive the race, got %d", completedCount)
	}
	if err := db.Model(&models.Entitlement{}).Where("user_id = ? AND program_id = ? AND deleted_at IS NULL", userID, programID).Count(&entitlementCount).Error; err != nil {
		t.Fatalf("count entitlements: %v", err)
	}
	if entitlementCount != 1 {
		t.Fatalf("exactly one active entitlement must survive the race, got %d", entitlementCount)
	}
}