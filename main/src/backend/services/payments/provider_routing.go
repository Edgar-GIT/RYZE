package payments

import (
	"context"
	"fmt"
)

// ProviderResolver maps a payment method to the appropriate payment provider.
// The resolver is configured once at startup and called for every payment
// initiation request. It must return a valid provider or an error for every
// supported method.
type ProviderResolver func(ctx context.Context, method PaymentMethod) (Provider, error)

// ValidatePaymentMethod reports whether the given payment method string is
// supported. An empty method is never valid — the client must explicitly
// select a payment method.
func ValidatePaymentMethod(method string) error {
	switch PaymentMethod(method) {
	case PaymentMethodCard, PaymentMethodMBWay, PaymentMethodPayPal:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidPaymentMethod, method)
	}
}

// MethodProviderMap maps payment methods to provider instances. It is
// configured once at startup and used by NewProviderResolver.
type MethodProviderMap struct {
	stripe Provider
	paypal Provider
}

// NewMethodProviderMap creates a method-to-provider mapping. Nil providers
// indicate the method is not configured.
func NewMethodProviderMap(stripe, paypal Provider) *MethodProviderMap {
	return &MethodProviderMap{
		stripe: stripe,
		paypal: paypal,
	}
}

// Resolve returns the provider for the given method, or an error if no provider
// is available for that method.
func (m *MethodProviderMap) Resolve(_ context.Context, method PaymentMethod) (Provider, error) {
	switch method {
	case PaymentMethodCard, PaymentMethodMBWay:
		if m.stripe == nil {
			return nil, fmt.Errorf("%w: stripe not configured", ErrNoProviderAvailable)
		}
		return m.stripe, nil
	case PaymentMethodPayPal:
		if m.paypal == nil {
			return nil, fmt.Errorf("%w: paypal not configured", ErrNoProviderAvailable)
		}
		return m.paypal, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrInvalidPaymentMethod, method)
	}
}
