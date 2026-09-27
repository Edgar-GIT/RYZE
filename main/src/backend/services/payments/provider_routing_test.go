package payments_test

import (
	"context"
	"errors"
	"testing"

	"ryze/backend/services/payments"
)

func TestValidatePaymentMethodValid(t *testing.T) {
	methods := []string{"card", "mbway", "paypal"}
	for _, m := range methods {
		if err := payments.ValidatePaymentMethod(m); err != nil {
			t.Errorf("expected nil error for method %q, got %v", m, err)
		}
	}
}

func TestValidatePaymentMethodEmpty(t *testing.T) {
	if err := payments.ValidatePaymentMethod(""); err == nil {
		t.Fatal("expected error for empty method")
	} else if !errors.Is(err, payments.ErrInvalidPaymentMethod) {
		t.Fatalf("expected ErrInvalidPaymentMethod, got %v", err)
	}
}

func TestValidatePaymentMethodInvalid(t *testing.T) {
	methods := []string{"bitcoin", "wire", "STRIPE", "Card", "PAYPAL"}
	for _, m := range methods {
		if err := payments.ValidatePaymentMethod(m); err == nil {
			t.Errorf("expected error for method %q", m)
		}
	}
}

func TestMethodProviderMapResolveCard(t *testing.T) {
	stripe := payments.NewFakeProvider()
	paypalProvider := payments.NewFakeProvider()
	methodMap := payments.NewMethodProviderMap(stripe, paypalProvider)

	provider, err := methodMap.Resolve(context.Background(), payments.PaymentMethodCard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != stripe {
		t.Fatal("expected stripe provider for card method")
	}
}

func TestMethodProviderMapResolveMBWay(t *testing.T) {
	stripe := payments.NewFakeProvider()
	paypalProvider := payments.NewFakeProvider()
	methodMap := payments.NewMethodProviderMap(stripe, paypalProvider)

	provider, err := methodMap.Resolve(context.Background(), payments.PaymentMethodMBWay)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != stripe {
		t.Fatal("expected stripe provider for mbway method")
	}
}

func TestMethodProviderMapResolvePayPal(t *testing.T) {
	stripe := payments.NewFakeProvider()
	paypalProvider := payments.NewFakeProvider()
	methodMap := payments.NewMethodProviderMap(stripe, paypalProvider)

	provider, err := methodMap.Resolve(context.Background(), payments.PaymentMethodPayPal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != paypalProvider {
		t.Fatal("expected paypal provider for paypal method")
	}
}

func TestMethodProviderMapResolveUnknown(t *testing.T) {
	methodMap := payments.NewMethodProviderMap(payments.NewFakeProvider(), payments.NewFakeProvider())

	_, err := methodMap.Resolve(context.Background(), "bitcoin")
	if err == nil {
		t.Fatal("expected error for unknown method")
	}
	if !errors.Is(err, payments.ErrInvalidPaymentMethod) {
		t.Fatalf("expected ErrInvalidPaymentMethod, got %v", err)
	}
}

func TestMethodProviderMapStripeNotConfigured(t *testing.T) {
	methodMap := payments.NewMethodProviderMap(nil, payments.NewFakeProvider())

	_, err := methodMap.Resolve(context.Background(), payments.PaymentMethodCard)
	if err == nil {
		t.Fatal("expected error when stripe not configured")
	}
	if !errors.Is(err, payments.ErrNoProviderAvailable) {
		t.Fatalf("expected ErrNoProviderAvailable, got %v", err)
	}

	_, err = methodMap.Resolve(context.Background(), payments.PaymentMethodMBWay)
	if err == nil {
		t.Fatal("expected error when stripe not configured for mbway")
	}
	if !errors.Is(err, payments.ErrNoProviderAvailable) {
		t.Fatalf("expected ErrNoProviderAvailable, got %v", err)
	}
}

func TestMethodProviderMapPayPalNotConfigured(t *testing.T) {
	methodMap := payments.NewMethodProviderMap(payments.NewFakeProvider(), nil)

	_, err := methodMap.Resolve(context.Background(), payments.PaymentMethodPayPal)
	if err == nil {
		t.Fatal("expected error when paypal not configured")
	}
	if !errors.Is(err, payments.ErrNoProviderAvailable) {
		t.Fatalf("expected ErrNoProviderAvailable, got %v", err)
	}
}

func TestMethodProviderMapNoProvidersConfigured(t *testing.T) {
	methodMap := payments.NewMethodProviderMap(nil, nil)

	_, err := methodMap.Resolve(context.Background(), payments.PaymentMethodCard)
	if err == nil {
		t.Fatal("expected error when no providers configured")
	}
	if !errors.Is(err, payments.ErrNoProviderAvailable) {
		t.Fatalf("expected ErrNoProviderAvailable, got %v", err)
	}
}

func TestMethodProviderMapAvailableMethods(t *testing.T) {
	stripe := payments.NewFakeProvider()
	paypalProvider := payments.NewFakeProvider()
	methodMap := payments.NewMethodProviderMap(stripe, paypalProvider)

	got := methodMap.AvailableMethods()
	want := []payments.PaymentMethod{payments.PaymentMethodCard, payments.PaymentMethodMBWay, payments.PaymentMethodPayPal}
	if len(got) != len(want) {
		t.Fatalf("expected %d available methods, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected stable order %v, got %v", want, got)
		}
	}
}

func TestMethodProviderMapAvailableMethodsOmitsUnconfigured(t *testing.T) {
	methodMap := payments.NewMethodProviderMap(nil, payments.NewFakeProvider())

	got := methodMap.AvailableMethods()
	want := []payments.PaymentMethod{payments.PaymentMethodPayPal}
	if len(got) != len(want) {
		t.Fatalf("expected only %d available method, got %v", len(want), got)
	}
	if got[0] != want[0] {
		t.Fatalf("expected %q, got %q", want[0], got[0])
	}
}

func TestMethodProviderMapAvailableMethodsEmpty(t *testing.T) {
	methodMap := payments.NewMethodProviderMap(nil, nil)

	if got := methodMap.AvailableMethods(); len(got) != 0 {
		t.Fatalf("expected no available methods, got %v", got)
	}
}
