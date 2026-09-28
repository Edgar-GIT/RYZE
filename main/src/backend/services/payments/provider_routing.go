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
	mbway  Provider
	paypal Provider
}

// NewMethodProviderMap creates a method-to-provider mapping. Nil providers
// indicate the method is not configured. A single provider instance may serve
// more than one method when the provider genuinely supports them: Stripe
// serves both card and MB WAY, so both slots can hold the same Stripe
// provider.
func NewMethodProviderMap(stripe, mbway, paypal Provider) *MethodProviderMap {
	return &MethodProviderMap{
		stripe: stripe,
		mbway:  mbway,
		paypal: paypal,
	}
}

// AvailableMethods returns the payment methods that have a configured provider,
// in a stable order. Methods whose provider is nil (not configured at startup)
// are omitted, so callers never advertise a payment method that would fail at
// initiation. Availability and resolution are driven by the same provider
// fields, so the two can never diverge.
func (m *MethodProviderMap) AvailableMethods() []PaymentMethod {
	var methods []PaymentMethod
	for _, method := range []PaymentMethod{PaymentMethodCard, PaymentMethodMBWay, PaymentMethodPayPal} {
		switch method {
		case PaymentMethodCard:
			if m.stripe != nil {
				methods = append(methods, method)
			}
		case PaymentMethodMBWay:
			if m.mbway != nil {
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
// is available for that method. An unavailable method fails closed with
// ErrNoProviderAvailable; it is never silently substituted with another method
// or provider.
func (m *MethodProviderMap) Resolve(_ context.Context, method PaymentMethod) (Provider, error) {
	switch method {
	case PaymentMethodCard:
		if m.stripe == nil {
			return nil, fmt.Errorf("%w: stripe not configured", ErrNoProviderAvailable)
		}
		return m.stripe, nil
	case PaymentMethodMBWay:
		if m.mbway == nil {
			return nil, fmt.Errorf("%w: mbway not configured", ErrNoProviderAvailable)
		}
		return m.mbway, nil
	case PaymentMethodPayPal:
		if m.paypal == nil {
			return nil, fmt.Errorf("%w: paypal not configured", ErrNoProviderAvailable)
		}
		return m.paypal, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrInvalidPaymentMethod, method)
	}
}
