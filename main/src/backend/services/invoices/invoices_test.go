package invoices

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"ryze/backend/config"
	"ryze/backend/email"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

type fakeRepo struct {
	purchases map[string]*models.InvoicePurchase
	invoices  map[string]*models.Invoice // by purchase id
	emailed   map[string]bool            // invoice ids whose email was recorded
	next      int
	issueErr  error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{purchases: map[string]*models.InvoicePurchase{}, invoices: map[string]*models.Invoice{}, emailed: map[string]bool{}}
}

func (f *fakeRepo) add(id, userID string, test bool, status string) *models.InvoicePurchase {
	persona := "client"
	buyer := models.User{ID: userID, Email: userID + "@example.com", FirstName: "Rita", LastName: "Costa"}
	if test {
		buyer.Email = userID + "@ryze.local"
		buyer.TestPersona = &persona
	}
	tax := "123456789"
	price := int64(2500)
	if test {
		price = 0 // Test Mode purchases are zero-price
	}
	loaded := &models.InvoicePurchase{
		Purchase: models.Purchase{ID: id, UserID: userID, ProgramID: "program-" + id, PriceMinorUnits: price, Currency: "EUR", Status: status, Test: test, BuyerTaxID: &tax, UpdatedAt: time.Now()},
		Buyer:    buyer,
		Program:  models.Program{ID: "program-" + id, Name: "Hypertrophy Foundations", ProductType: "generic"},
	}
	f.purchases[id] = loaded
	return loaded
}

func (f *fakeRepo) Issue(_ context.Context, invoice *models.Invoice) (*models.Invoice, bool, error) {
	if f.issueErr != nil {
		return nil, false, f.issueErr
	}
	if existing, ok := f.invoices[invoice.PurchaseID]; ok {
		return existing, false, nil
	}
	f.next++
	invoice.ID = fmt.Sprintf("invoice-%d", f.next)
	invoice.Sequence = f.next
	invoice.Number = fmt.Sprintf("%s %d/%06d", invoice.Series, invoice.Year, f.next)
	f.invoices[invoice.PurchaseID] = invoice
	return invoice, true, nil
}

func (f *fakeRepo) FindByPurchaseID(_ context.Context, purchaseID string) (*models.Invoice, error) {
	if invoice, ok := f.invoices[purchaseID]; ok {
		return invoice, nil
	}
	return nil, repositories.ErrInvoiceNotFound
}

func (f *fakeRepo) FindByPurchaseIDForUser(ctx context.Context, purchaseID, userID string) (*models.Invoice, error) {
	invoice, err := f.FindByPurchaseID(ctx, purchaseID)
	if err != nil || invoice.UserID != userID {
		return nil, repositories.ErrInvoiceNotFound
	}
	return invoice, nil
}

func (f *fakeRepo) ListByUser(_ context.Context, userID string) ([]models.Invoice, error) {
	var out []models.Invoice
	for _, invoice := range f.invoices {
		if invoice.UserID == userID {
			out = append(out, *invoice)
		}
	}
	return out, nil
}

func (f *fakeRepo) CompletedWithoutInvoice(_ context.Context, userID, programID string, _ int) ([]models.InvoicePurchase, error) {
	var out []models.InvoicePurchase
	for id, loaded := range f.purchases {
		if _, ok := f.invoices[id]; ok || loaded.Purchase.Status != models.PurchaseStatusCompleted {
			continue
		}
		if (userID == "" || loaded.Purchase.UserID == userID) && (programID == "" || loaded.Purchase.ProgramID == programID) {
			out = append(out, *loaded)
		}
	}
	return out, nil
}

func (f *fakeRepo) LoadForIssue(_ context.Context, purchaseID string) (*models.InvoicePurchase, error) {
	loaded, ok := f.purchases[purchaseID]
	if !ok {
		return nil, repositories.ErrPurchaseNotFound
	}
	copied := *loaded
	return &copied, nil
}

func (f *fakeRepo) WithoutEmail(_ context.Context, _ time.Time, _ int) ([]models.Invoice, error) {
	var out []models.Invoice
	for _, invoice := range f.invoices {
		if !f.emailed[invoice.ID] {
			out = append(out, *invoice)
		}
	}
	return out, nil
}

type fakeMailer struct {
	repo     *fakeRepo
	requests []email.Request
	fail     bool
}

func (m *fakeMailer) Enabled() bool { return true }

func (m *fakeMailer) Enqueue(_ context.Context, request email.Request) error {
	if m.fail {
		return errors.New("outbox unavailable")
	}
	for _, previous := range m.requests {
		if previous.DedupeKey == request.DedupeKey {
			return nil
		}
	}
	m.requests = append(m.requests, request)
	for _, invoice := range m.repo.invoices {
		if repositories.InvoiceEmailDedupeKey(invoice.ID) == request.DedupeKey {
			m.repo.emailed[invoice.ID] = true
		}
	}
	return nil
}

type fakeRecipients map[string]string

func (f fakeRecipients) FindActiveRecipientByPersona(_ context.Context, personaUserID string) (string, error) {
	return f[personaUserID], nil
}

func newTestService(recipients fakeRecipients) (*service, *fakeRepo, *fakeMailer) {
	repo := newFakeRepo()
	mailer := &fakeMailer{repo: repo}
	svc := New(repo, mailer, recipients, config.InvoiceConfig{SellerName: "RYZE", SellerTaxID: "999999990"}).(*service)
	svc.now = func() time.Time { return time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC) }
	return svc, repo, mailer
}

func TestFulfillPurchaseIssuesOneDocumentAndOneEmail(t *testing.T) {
	svc, repo, mailer := newTestService(nil)
	ctx := context.Background()
	repo.add("p1", "u1", false, models.PurchaseStatusCompleted)

	for i := 0; i < 3; i++ {
		if err := svc.FulfillPurchase(ctx, "u1", "program-p1"); err != nil {
			t.Fatalf("FulfillPurchase: %v", err)
		}
	}
	invoice := repo.invoices["p1"]
	if len(repo.invoices) != 1 || invoice.Number != "RYZE 2026/000001" || invoice.Test {
		t.Fatalf("expected one sales document, got %+v", repo.invoices)
	}
	if invoice.TotalMinorUnits != 2500 || invoice.TaxMinorUnits != 0 || invoice.TaxExemptionCode == nil || *invoice.TaxExemptionCode != "M10" {
		t.Fatalf("unexpected amounts or VAT treatment: %+v", invoice)
	}
	if invoice.BuyerName != "Rita Costa" || invoice.BuyerTaxID == nil || *invoice.BuyerTaxID != "123456789" || invoice.Description != "Hypertrophy Foundations" {
		t.Fatalf("unexpected snapshot: %+v", invoice)
	}
	if invoice.SellerTaxID == nil || *invoice.SellerTaxID != "999999990" {
		t.Fatalf("seller identity not recorded: %+v", invoice)
	}
	if len(mailer.requests) != 1 {
		t.Fatalf("expected one email, got %d", len(mailer.requests))
	}
	request := mailer.requests[0]
	if request.To != "u1@example.com" || request.Template != email.TemplateInvoiceIssued || request.Invoice == nil ||
		request.Invoice.Number != invoice.Number || request.Path != "/account/purchases/p1/receipt" {
		t.Fatalf("unexpected email: %+v", request)
	}
}

func TestTestModeDocumentsGoToTheSessionRecipient(t *testing.T) {
	svc, repo, mailer := newTestService(fakeRecipients{"persona": "qa@example.com"})
	ctx := context.Background()
	repo.add("p1", "persona", true, models.PurchaseStatusCompleted)

	if err := svc.FulfillPurchase(ctx, "persona", "program-p1"); err != nil {
		t.Fatal(err)
	}
	if invoice := repo.invoices["p1"]; invoice == nil || !invoice.Test || invoice.Series != models.InvoiceSeriesTest || invoice.TotalMinorUnits != 0 {
		t.Fatalf("expected a test-series document, got %+v", invoice)
	}
	if len(mailer.requests) != 1 || mailer.requests[0].To != "qa@example.com" || !mailer.requests[0].Invoice.Test {
		t.Fatalf("the document must go to the session recipient, got %+v", mailer.requests)
	}
}

func TestTestModeDocumentWithoutRecipientIsIssuedButNotEmailed(t *testing.T) {
	svc, repo, mailer := newTestService(fakeRecipients{})
	ctx := context.Background()
	repo.add("p1", "persona", true, models.PurchaseStatusCompleted)
	if err := svc.FulfillPurchase(ctx, "persona", "program-p1"); err != nil {
		t.Fatal(err)
	}
	if repo.invoices["p1"] == nil || len(mailer.requests) != 0 {
		t.Fatalf("expected the document without an email, got %d emails", len(mailer.requests))
	}
	// The sweep never re-sends a Test Mode document to a later session.
	if err := svc.Sweep(ctx); err != nil || len(mailer.requests) != 0 {
		t.Fatalf("sweep emailed a test document: %v %d", err, len(mailer.requests))
	}
}

func TestForPurchaseIsOwnerOnlyAndIssuesOnDemand(t *testing.T) {
	svc, repo, _ := newTestService(nil)
	ctx := context.Background()
	repo.add("p1", "u1", false, models.PurchaseStatusCompleted)
	repo.add("p2", "u1", false, models.PurchaseStatusPending)

	if _, err := svc.ForPurchase(ctx, "u2", "p1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another account must not read the document, got %v", err)
	}
	if len(repo.invoices) != 0 {
		t.Fatal("a foreign read must not issue anything")
	}
	if _, err := svc.ForPurchase(ctx, "u1", "p2"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a pending purchase has no document, got %v", err)
	}
	invoice, err := svc.ForPurchase(ctx, "u1", "p1")
	if err != nil || invoice.PurchaseID != "p1" {
		t.Fatalf("owner read must issue the missing document: %+v (%v)", invoice, err)
	}
	if _, err := svc.ForPurchase(ctx, "u1", "unknown"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown purchase: %v", err)
	}
}

func TestFailuresAreRetriedAndNeverPanic(t *testing.T) {
	svc, repo, mailer := newTestService(nil)
	ctx := context.Background()
	repo.add("p1", "u1", false, models.PurchaseStatusCompleted)

	repo.issueErr = errors.New("database unavailable")
	if err := svc.FulfillPurchase(ctx, "u1", "program-p1"); err == nil {
		t.Fatal("the hook reports the failure to the purchase service, which logs it")
	}
	repo.issueErr = nil
	mailer.fail = true
	if err := svc.Sweep(ctx); err != nil {
		t.Fatal(err)
	}
	if repo.invoices["p1"] == nil {
		t.Fatal("the sweep must issue the missing document")
	}
	mailer.fail = false
	if err := svc.Sweep(ctx); err != nil {
		t.Fatal(err)
	}
	if len(mailer.requests) != 1 || mailer.requests[0].To != "u1@example.com" {
		t.Fatalf("the sweep must record the missing email, got %+v", mailer.requests)
	}
}
