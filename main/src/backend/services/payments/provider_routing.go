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
// select a payment method. A valid method is not necessarily available: the
// method must also be advertised by the backend and resolvable to a provider.
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

// AvailableMethods returns the payment methods that have a configured provider,
// in a stable order. Methods whose provider is nil (not configured at startup)
// are omitted, so callers never advertise a payment method that would fail at
// initiation. MB WAY is never advertised: it is a prepared but unimplemented
// method, so it is omitted even when Stripe is configured.
func (m *MethodProviderMap) AvailableMethods() []PaymentMethod {
	var methods []PaymentMethod
	for _, method := range []PaymentMethod{PaymentMethodCard, PaymentMethodPayPal} {
		switch method {
		case PaymentMethodCard:
			if m.stripe != nil {
				methods = append(methods, method)
			}
		case PaymentMethodPayPal:
			if m.paypal != nil {
				methods = append(methods, method)
			}
		}
	}
	return methods
}

// Resolve returns the provider for the given method, or an error if no provider
// is available for that method. MB WAY is never resolved: it is a prepared but
// unimplemented method with no provider, so requesting it fails closed with
// ErrNoProviderAvailable instead of being routed to Stripe.
func (m *MethodProviderMap) Resolve(_ context.Context, method PaymentMethod) (Provider, error) {
	switch method {
	case PaymentMethodCard:
		if m.stripe == nil {
			return nil, fmt.Errorf("%w: stripe not configured", ErrNoProviderAvailable)
		}
		return m.stripe, nil
	case PaymentMethodMBWay:
		return nil, fmt.Errorf("%w: %q is not implemented by any provider", ErrNoProviderAvailable, method)
	case PaymentMethodPayPal:
		if m.paypal == nil {
			return nil, fmt.Errorf("%w: paypal not configured", ErrNoProviderAvailable)
		}
		return m.paypal, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrInvalidPaymentMethod, method)
	}
}
