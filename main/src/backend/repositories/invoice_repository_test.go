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

func openInvoiceDB(t *testing.T) *gorm.DB {
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
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// seedCompletedPurchase creates a buyer, a programme and a completed purchase.
func seedCompletedPurchase(t *testing.T, db *gorm.DB, test bool) *models.Purchase {
	t.Helper()
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := &models.User{Email: "invoice-" + suffix + "@ryze.local", PasswordHash: "hash", FirstName: "Rita", LastName: "Costa"}
	if err := repositories.NewUserRepository(db).Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	program := &models.Program{
		Name: "Invoice Programme " + suffix, Description: "Description", Type: models.ProgramTypePremium,
		Status: models.ProgramStatusPublished, PriceMinorUnits: 2500, Currency: string(models.ProgramCurrencyEUR),
	}
	if err := repositories.NewGenericProgramRepository(db).CreateFull(ctx, program); err != nil {
		t.Fatalf("create program: %v", err)
	}
	price := int64(2500)
	if test {
		price = 0
	}
	purchase := &models.Purchase{UserID: user.ID, ProgramID: program.ID, PriceMinorUnits: price, Currency: "EUR", Status: models.PurchaseStatusCompleted, Test: test}
	if err := db.Create(purchase).Error; err != nil {
		t.Fatalf("create purchase: %v", err)
	}
	return purchase
}

func draftInvoice(purchase *models.Purchase, year int) *models.Invoice {
	series := models.InvoiceSeriesSales
	if purchase.Test {
		series = models.InvoiceSeriesTest
	}
	now := time.Now().UTC()
	return &models.Invoice{
		Series: series, Year: year, PurchaseID: purchase.ID, UserID: purchase.UserID, ProgramID: purchase.ProgramID,
		Test: purchase.Test, Status: models.InvoiceStatusIssued, IssuedAt: now, PurchasedAt: now, SellerName: "RYZE",
		BuyerName: "Rita Costa", Description: "Programme", ProductType: "generic", Quantity: 1,
		UnitPriceMinorUnits: purchase.PriceMinorUnits, TotalMinorUnits: purchase.PriceMinorUnits, Currency: "EUR",
	}
}

// Numbers are sequential per series and year, one document per purchase, and
// an issued document can be neither changed nor deleted.
func TestInvoiceRepositoryIssuesOncePerPurchaseInSequence(t *testing.T) {
	db := openInvoiceDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	repo := repositories.NewInvoiceRepository(tx)
	ctx := context.Background()
	const year = 2091

	first := seedCompletedPurchase(t, tx, false)
	second := seedCompletedPurchase(t, tx, false)
	testPurchase := seedCompletedPurchase(t, tx, true)

	pending, err := repo.CompletedWithoutInvoice(ctx, first.UserID, first.ProgramID, 10)
	if err != nil || len(pending) != 1 || pending[0].Purchase.ID != first.ID || pending[0].Buyer.ID != first.UserID {
		t.Fatalf("completed purchase without invoice not listed: %+v (%v)", pending, err)
	}

	a, created, err := repo.Issue(ctx, draftInvoice(first, year))
	if err != nil || !created || a.Number != "RYZE 2091/000001" || a.Sequence != 1 {
		t.Fatalf("first invoice = %+v, created %v (%v)", a, created, err)
	}
	b, created, err := repo.Issue(ctx, draftInvoice(second, year))
	if err != nil || !created || b.Number != "RYZE 2091/000002" {
		t.Fatalf("second invoice = %+v, created %v (%v)", b, created, err)
	}
	again, created, err := repo.Issue(ctx, draftInvoice(first, year))
	if err != nil || created || again.ID != a.ID || again.Number != a.Number {
		t.Fatalf("a repeated issue must return the first document, got %+v created %v (%v)", again, created, err)
	}
	test, _, err := repo.Issue(ctx, draftInvoice(testPurchase, year))
	if err != nil || test.Number != "TEST 2091/000001" {
		t.Fatalf("test series = %+v (%v)", test, err)
	}
	if pending, _ := repo.CompletedWithoutInvoice(ctx, first.UserID, first.ProgramID, 10); len(pending) != 0 {
		t.Fatalf("an invoiced purchase is still listed: %+v", pending)
	}

	if _, err := repo.FindByPurchaseIDForUser(ctx, first.ID, second.UserID); !errors.Is(err, repositories.ErrInvoiceNotFound) {
		t.Fatalf("another account must not read the document, got %v", err)
	}
	if mine, err := repo.FindByPurchaseIDForUser(ctx, first.ID, first.UserID); err != nil || mine.ID != a.ID {
		t.Fatalf("owner read = %+v (%v)", mine, err)
	}

	if err := inSavepoint(tx, func(sp *gorm.DB) error {
		return sp.Exec("UPDATE invoices SET total_minor_units = 1 WHERE id = ?", a.ID).Error
	}); err == nil {
		t.Fatal("an issued invoice must not be updatable")
	}
	if err := inSavepoint(tx, func(sp *gorm.DB) error {
		return sp.Exec("DELETE FROM invoices WHERE id = ?", a.ID).Error
	}); err == nil {
		t.Fatal("an issued invoice must not be deletable")
	}
}

// Concurrent deliveries of the same purchase issue one document, and
// concurrent issues for different purchases take distinct, gapless numbers.
func TestInvoiceRepositoryConcurrentIssues(t *testing.T) {
	db := openInvoiceDB(t)
	// Its purchases are committed: hold the shared test lock until they are
	// deleted (registered first, so it is released last).
	t.Cleanup(holdCommittedPurchasesLock(t, db))
	year := 2100 + int(time.Now().UnixNano()%800)
	purchases := []*models.Purchase{seedCompletedPurchase(t, db, false), seedCompletedPurchase(t, db, false), seedCompletedPurchase(t, db, false)}
	t.Cleanup(func() {
		// Issued documents are immutable; this isolated test database lifts the
		// guard only to remove what the test committed.
		_ = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("ALTER TABLE invoices DISABLE TRIGGER trg_invoices_immutable").Error; err != nil {
				return err
			}
			for _, purchase := range purchases {
				tx.Exec("DELETE FROM invoices WHERE purchase_id = ?", purchase.ID)
			}
			return tx.Exec("ALTER TABLE invoices ENABLE TRIGGER trg_invoices_immutable").Error
		})
		db.Exec("DELETE FROM invoice_counters WHERE year = ?", year)
		for _, purchase := range purchases {
			db.Exec("DELETE FROM purchases WHERE id = ?", purchase.ID)
			db.Exec("DELETE FROM programs WHERE id = ?", purchase.ProgramID)
			db.Exec("DELETE FROM users WHERE id = ?", purchase.UserID)
		}
	})

	repo := repositories.NewInvoiceRepository(db)
	ctx := context.Background()
	const perPurchase = 4
	var wg sync.WaitGroup
	var mu sync.Mutex
	numbers := map[string]map[string]bool{}
	for _, purchase := range purchases {
		for i := 0; i < perPurchase; i++ {
			wg.Add(1)
			go func(purchase *models.Purchase) {
				defer wg.Done()
				invoice, _, err := repo.Issue(ctx, draftInvoice(purchase, year))
				if err != nil {
					t.Errorf("issue: %v", err)
					return
				}
				mu.Lock()
				if numbers[purchase.ID] == nil {
					numbers[purchase.ID] = map[string]bool{}
				}
				numbers[purchase.ID][invoice.Number] = true
				mu.Unlock()
			}(purchase)
		}
	}
	wg.Wait()

	seen := map[string]bool{}
	for id, set := range numbers {
		if len(set) != 1 {
			t.Fatalf("purchase %s got %d different documents: %v", id, len(set), set)
		}
		for number := range set {
			seen[number] = true
		}
	}
	for sequence := 1; sequence <= len(purchases); sequence++ {
		if want := fmt.Sprintf("RYZE %d/%06d", year, sequence); !seen[want] {
			t.Fatalf("numbers are not gapless: missing %s in %v", want, seen)
		}
	}
	var count int64
	db.Model(&models.Invoice{}).Where("year = ?", year).Count(&count)
	if count != int64(len(purchases)) {
		t.Fatalf("expected %d documents, found %d", len(purchases), count)
	}
}
