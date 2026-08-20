package payments_test

import (
	"context"
	"errors"
	"testing"

	"ryze/backend/services/payments"
)

func TestFakeProviderSuccess(t *testing.T) {
	provider := payments.NewFakeProvider()

	result, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-001",
		AmountMinorUnits: 10000,
		Currency:         "EUR",
		ProgramID:        "program-001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PaymentID == "" {
		t.Fatal("expected non-empty payment id")
	}
	if result.Status != payments.PaymentStatusPending {
		t.Fatalf("expected status %q, got %q", payments.PaymentStatusPending, result.Status)
	}
	if result.CheckoutURL == "" {
		t.Fatal("expected non-empty checkout url")
	}
	if result.Provider != "fake" {
		t.Fatalf("expected provider %q, got %q", "fake", result.Provider)
	}
	if result.PurchaseID != "purchase-001" {
		t.Fatalf("expected purchase id %q, got %q", "purchase-001", result.PurchaseID)
	}
}

func TestFakeProviderDeterministicIDs(t *testing.T) {
	provider := payments.NewFakeProvider()

	r1, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{PurchaseID: "p1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r2, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{PurchaseID: "p2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r1.PaymentID == r2.PaymentID {
		t.Fatal("expected unique payment ids")
	}
}

func TestFakeProviderFailure(t *testing.T) {
	provider := payments.NewFakeProvider()
	provider.FailWhen = true

	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{PurchaseID: "p1"})
	if err == nil {
		t.Fatal("expected error when FailWhen is true")
	}
	if !errors.Is(err, payments.ErrProviderFailure) {
		t.Fatalf("expected ErrProviderFailure wrapped, got %v", err)
	}
}
