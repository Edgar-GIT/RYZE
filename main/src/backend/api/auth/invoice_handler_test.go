package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/middleware/authcontext"
	"ryze/backend/models"
	"ryze/backend/services/invoices"
)

type stubInvoiceService struct {
	owner   string
	invoice *models.Invoice
}

func (s *stubInvoiceService) FulfillPurchase(context.Context, string, string) error { return nil }
func (s *stubInvoiceService) Sweep(context.Context) error                           { return nil }
func (s *stubInvoiceService) ForPurchase(_ context.Context, userID, purchaseID string) (*models.Invoice, error) {
	if userID != s.owner || purchaseID != s.invoice.PurchaseID {
		return nil, invoices.ErrNotFound
	}
	return s.invoice, nil
}
func (s *stubInvoiceService) List(_ context.Context, userID string) ([]models.Invoice, error) {
	if userID != s.owner {
		return nil, nil
	}
	return []models.Invoice{*s.invoice}, nil
}
func (s *stubInvoiceService) ForPurchaseAdmin(_ context.Context, purchaseID string) (*models.Invoice, error) {
	if purchaseID != s.invoice.PurchaseID {
		return nil, invoices.ErrNotFound
	}
	return s.invoice, nil
}

func TestInvoiceHandlerServesOnlyTheOwnersDocument(t *testing.T) {
	owner := "33333333-3333-3333-3333-333333333333"
	stranger := "44444444-4444-4444-4444-444444444444"
	svc := &stubInvoiceService{owner: owner, invoice: &models.Invoice{ID: "inv-1", Number: "RYZE 2026/000001", PurchaseID: "purchase-1", UserID: owner, Currency: "EUR"}}
	handler := auth.NewInvoiceHandler(svc)
	request := func(identity, path string) *httptest.ResponseRecorder {
		router := gin.New()
		me := router.Group("/api/v1/me")
		me.Use(func(c *gin.Context) { c.Set(authcontext.UserIDContextKey, identity); c.Next() })
		me.GET("/invoices", handler.List)
		me.GET("/purchases/:purchaseID/invoice", handler.ForPurchase)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}

	rec := request(owner, "/api/v1/me/purchases/purchase-1/invoice")
	if rec.Code != http.StatusOK {
		t.Fatalf("owner read = %d", rec.Code)
	}
	var body struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data["number"] != "RYZE 2026/000001" || body.Data["document_type"] != auth.InvoiceDocumentType || body.Data["fiscal_value"] != false {
		t.Fatalf("unexpected document: %v", body.Data)
	}
	if strings.Contains(rec.Body.String(), owner) {
		t.Fatal("the owner's account id must not be exposed")
	}

	if rec := request(stranger, "/api/v1/me/purchases/purchase-1/invoice"); rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "INVOICE_NOT_FOUND") {
		t.Fatalf("another account must get 404, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := request(stranger, "/api/v1/me/invoices"); rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "RYZE 2026/000001") {
		t.Fatalf("another account's list must be empty, got %s", rec.Body.String())
	}
}
