package refunds_test

import (
	"context"
	"errors"
	"testing"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/payments"
	"ryze/backend/services/refunds"
)

type fakeRepo struct {
	purchase *models.Purchase
	recorded *repositories.RefundRecord
}

func (f *fakeRepo) FindRefundable(context.Context, string) (*models.Purchase, error) {
	return f.purchase, nil
}
func (f *fakeRepo) Refund(_ context.Context, _ string, record repositories.RefundRecord) (*models.Purchase, error) {
	f.recorded = &record
	refunded := *f.purchase
	refunded.Status = models.PurchaseStatusRefunded
	return &refunded, nil
}

type fakeRefunder struct {
	calls int
	err   error
}

func (f *fakeRefunder) RefundPayment(context.Context, payments.RefundRequest) (payments.RefundResult, error) {
	f.calls++
	return payments.RefundResult{ProviderRef: "re_1"}, f.err
}

func purchase(method string, test bool) *models.Purchase {
	return &models.Purchase{ID: "p1", PriceMinorUnits: 1499, Currency: "EUR", Status: models.PurchaseStatusCompleted, PaymentMethod: &method, Test: test}
}

func TestRefundRules(t *testing.T) {
	ctx := context.Background()
	input := refunds.Input{Reason: "non_conformity"}

	// Card: refunded through Stripe, with the provider's reference.
	repo, refunder := &fakeRepo{purchase: purchase("card", false)}, &fakeRefunder{}
	if _, err := refunds.New(repo, refunder).Refund(ctx, "p1", "admin-2", input); err != nil || refunder.calls != 1 || repo.recorded.ProviderRef != "re_1" {
		t.Fatalf("card refund: %v calls=%d %+v", err, refunder.calls, repo.recorded)
	}
	// PayPal: refunded in its dashboard first.
	repo, refunder = &fakeRepo{purchase: purchase("paypal", false)}, &fakeRefunder{}
	if _, err := refunds.New(repo, refunder).Refund(ctx, "p1", "admin-2", input); !errors.Is(err, refunds.ErrManualRequired) || repo.recorded != nil {
		t.Fatalf("paypal needs a manual refund: %v", err)
	}
	if _, err := refunds.New(repo, refunder).Refund(ctx, "p1", "admin-2", refunds.Input{Reason: "withdrawal", AlreadyRefunded: true}); err != nil || refunder.calls != 0 {
		t.Fatalf("a refund made in the dashboard is recorded: %v", err)
	}
	// A card payment Stripe cannot find also needs the dashboard.
	repo, refunder = &fakeRepo{purchase: purchase("card", false)}, &fakeRefunder{err: payments.ErrRefundPaymentNotFound}
	if _, err := refunds.New(repo, refunder).Refund(ctx, "p1", "admin-2", input); !errors.Is(err, refunds.ErrManualRequired) {
		t.Fatalf("unknown payment: %v", err)
	}
	// Test Mode: no money moved, only the access ends.
	repo, refunder = &fakeRepo{purchase: purchase("card", true)}, &fakeRefunder{}
	if _, err := refunds.New(repo, refunder).Refund(ctx, "p1", "admin-2", input); err != nil || refunder.calls != 0 {
		t.Fatalf("test purchase: %v calls=%d", err, refunder.calls)
	}
	// A reason is required.
	if _, err := refunds.New(repo, refunder).Refund(ctx, "p1", "admin-2", refunds.Input{}); !errors.Is(err, refunds.ErrInvalidInput) {
		t.Fatalf("reason required: %v", err)
	}
}
