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
	methodMap := payments.NewMethodProviderMap(stripe, nil, nil)

	provider, err := methodMap.Resolve(context.Background(), payments.PaymentMethodCard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != stripe {
		t.Fatal("expected stripe provider for card method")
	}
}

func TestMethodProviderMapResolveMBWay(t *testing.T) {
	mbway := payments.NewFakeProvider()
	methodMap := payments.NewMethodProviderMap(nil, mbway, nil)

	provider, err := methodMap.Resolve(context.Background(), payments.PaymentMethodMBWay)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != mbway {
		t.Fatal("expected the MB WAY provider for the mbway method")
	}
}

// TestMethodProviderMapResolveMBWayWithSharedStripeProvider covers the real
// wiring: Stripe settles both card and MB WAY, so the same provider instance is
// registered in both slots and each method resolves to it.
func TestMethodProviderMapResolveMBWayWithSharedStripeProvider(t *testing.T) {
	stripe := payments.NewFakeProvider()
	methodMap := payments.NewMethodProviderMap(stripe, stripe, nil)

	card, err := methodMap.Resolve(context.Background(), payments.PaymentMethodCard)
	if err != nil {
		t.Fatalf("card: unexpected error: %v", err)
	}
	mbway, err := methodMap.Resolve(context.Background(), payments.PaymentMethodMBWay)
	if err != nil {
		t.Fatalf("mbway: unexpected error: %v", err)
	}
	if card != stripe || mbway != stripe {
		t.Fatal("expected both methods to resolve to the shared Stripe provider")
	}
}

func TestMethodProviderMapResolveMBWayNotConfigured(t *testing.T) {
	// MB WAY must fail closed when it has no provider, independently of the
	// other methods.
	methodMap := payments.NewMethodProviderMap(payments.NewFakeProvider(), nil, payments.NewFakeProvider())

	provider, err := methodMap.Resolve(context.Background(), payments.PaymentMethodMBWay)
	if err == nil {
		t.Fatal("expected error for mbway when unconfigured")
	}
	if !errors.Is(err, payments.ErrNoProviderAvailable) {
		t.Fatalf("expected ErrNoProviderAvailable, got %v", err)
	}
	if provider != nil {
		t.Fatal("expected no provider for mbway")
	}

	// The other methods must be unaffected.
	if _, err := methodMap.Resolve(context.Background(), payments.PaymentMethodCard); err != nil {
		t.Fatalf("card must remain available: %v", err)
	}
	if _, err := methodMap.Resolve(context.Background(), payments.PaymentMethodPayPal); err != nil {
		t.Fatalf("paypal must remain available: %v", err)
	}
}

func TestMethodProviderMapResolveMBWayWithoutAnyProvider(t *testing.T) {
	methodMap := payments.NewMethodProviderMap(nil, nil, nil)

	_, err := methodMap.Resolve(context.Background(), payments.PaymentMethodMBWay)
	if !errors.Is(err, payments.ErrNoProviderAvailable) {
		t.Fatalf("expected ErrNoProviderAvailable, got %v", err)
	}
}

func TestMethodProviderMapResolvePayPal(t *testing.T) {
	paypalProvider := payments.NewFakeProvider()
	methodMap := payments.NewMethodProviderMap(nil, nil, paypalProvider)

	provider, err := methodMap.Resolve(context.Background(), payments.PaymentMethodPayPal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != paypalProvider {
		t.Fatal("expected paypal provider for paypal method")
	}
}

func TestMethodProviderMapResolveUnknown(t *testing.T) {
	methodMap := payments.NewMethodProviderMap(payments.NewFakeProvider(), payments.NewFakeProvider(), payments.NewFakeProvider())

	_, err := methodMap.Resolve(context.Background(), "bitcoin")
	if err == nil {
		t.Fatal("expected error for unknown method")
	}
	if !errors.Is(err, payments.ErrInvalidPaymentMethod) {
		t.Fatalf("expected ErrInvalidPaymentMethod, got %v", err)
	}
}

func TestMethodProviderMapStripeNotConfigured(t *testing.T) {
	methodMap := payments.NewMethodProviderMap(nil, nil, payments.NewFakeProvider())

	_, err := methodMap.Resolve(context.Background(), payments.PaymentMethodCard)
	if err == nil {
		t.Fatal("expected error when stripe not configured")
	}
	if !errors.Is(err, payments.ErrNoProviderAvailable) {
		t.Fatalf("expected ErrNoProviderAvailable, got %v", err)
	}
}

func TestMethodProviderMapPayPalNotConfigured(t *testing.T) {
	methodMap := payments.NewMethodProviderMap(payments.NewFakeProvider(), payments.NewFakeProvider(), nil)

	_, err := methodMap.Resolve(context.Background(), payments.PaymentMethodPayPal)
	if err == nil {
		t.Fatal("expected error when paypal not configured")
	}
	if !errors.Is(err, payments.ErrNoProviderAvailable) {
		t.Fatalf("expected ErrNoProviderAvailable, got %v", err)
	}
}

func TestMethodProviderMapNoProvidersConfigured(t *testing.T) {
	methodMap := payments.NewMethodProviderMap(nil, nil, nil)

	_, err := methodMap.Resolve(context.Background(), payments.PaymentMethodCard)
	if err == nil {
		t.Fatal("expected error when no providers configured")
	}
	if !errors.Is(err, payments.ErrNoProviderAvailable) {
		t.Fatalf("expected ErrNoProviderAvailable, got %v", err)
	}
}

func TestMethodProviderMapAvailableMethods(t *testing.T) {
	methodMap := payments.NewMethodProviderMap(
		payments.NewFakeProvider(),
		payments.NewFakeProvider(),
		payments.NewFakeProvider(),
	)

	got := methodMap.AvailableMethods()
	want := []payments.PaymentMethod{
		payments.PaymentMethodCard,
		payments.PaymentMethodMBWay,
		payments.PaymentMethodPayPal,
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d available methods, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected stable order %v, got %v", want, got)
		}
	}
}

func TestMethodProviderMapAvailableMethodsWithSharedStripeProvider(t *testing.T) {
	// The production wiring registers one Stripe provider for both card and
	// MB WAY, so both must be advertised together.
	stripe := payments.NewFakeProvider()
	methodMap := payments.NewMethodProviderMap(stripe, stripe, nil)

	want := []payments.PaymentMethod{payments.PaymentMethodCard, payments.PaymentMethodMBWay}
	got := methodMap.AvailableMethods()
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

// TestMethodProviderMapAvailabilityMatchesResolution is the invariant that
// prevents the checkout UI from offering a method the backend cannot service:
// everything advertised must resolve, and everything resolvable must be
// advertised.
func TestMethodProviderMapAvailabilityMatchesResolution(t *testing.T) {
	stripe := payments.NewFakeProvider()
	paypal := payments.NewFakeProvider()

	cases := map[string]*payments.MethodProviderMap{
		"all configured":     payments.NewMethodProviderMap(stripe, stripe, paypal),
		"stripe only":        payments.NewMethodProviderMap(stripe, stripe, nil),
		"paypal only":        payments.NewMethodProviderMap(nil, nil, paypal),
		"mbway without card": payments.NewMethodProviderMap(nil, stripe, nil),
		"nothing configured": payments.NewMethodProviderMap(nil, nil, nil),
		"mbway and paypal":   payments.NewMethodProviderMap(nil, stripe, paypal),
	}

	all := []payments.PaymentMethod{
		payments.PaymentMethodCard,
		payments.PaymentMethodMBWay,
		payments.PaymentMethodPayPal,
	}

	for name, methodMap := range cases {
		t.Run(name, func(t *testing.T) {
			advertised := map[payments.PaymentMethod]bool{}
			for _, method := range methodMap.AvailableMethods() {
				advertised[method] = true
			}

			for _, method := range all {
				_, err := methodMap.Resolve(context.Background(), method)
				resolvable := err == nil
				if advertised[method] != resolvable {
					t.Errorf("%s: advertised=%v but resolvable=%v; availability and resolution must never diverge", method, advertised[method], resolvable)
				}
			}
		})
	}
}

func TestMethodProviderMapAvailableMethodsOmitsUnconfigured(t *testing.T) {
	methodMap := payments.NewMethodProviderMap(nil, nil, payments.NewFakeProvider())

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
	methodMap := payments.NewMethodProviderMap(nil, nil, nil)

	if got := methodMap.AvailableMethods(); len(got) != 0 {
		t.Fatalf("expected no available methods, got %v", got)
	}
}
