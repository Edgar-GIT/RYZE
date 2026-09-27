package payments

import "context"

// Provider defines the contract every payment provider must implement. The
// abstraction is provider-independent: it knows nothing about Gin, HTTP
// handlers, MariaDB, GORM, or frontend concerns.
//
// Implementations receive only trusted server-side commercial information
// extracted from the immutable purchase snapshot. No client-supplied values
// are ever forwarded.
//
// A Provider is expected to be safe for concurrent use by multiple goroutines.
type Provider interface {
	// InitiatePayment requests the provider to create a payment for the given
	// purchase. The request contains only server-authoritative values taken
	// from the purchase snapshot. The result carries provider-independent
	// status information; the provider must not mark the RYZE purchase as
	// completed — that responsibility belongs to the verified provider event
	// → CompletePurchase() flow.
	InitiatePayment(ctx context.Context, request PaymentRequest) (PaymentResult, error)
}

// CaptureProvider is an optional capability implemented by providers whose
// payment can be verified server-side after the buyer returns from the
// provider page. It covers both providers that capture an approved payment
// (e.g. PayPal Orders) and providers that complete the payment on a hosted
// page and can only be re-verified (e.g. Stripe Checkout Sessions).
//
// In both cases the provider is the only source of truth: the browser response
// is treated as untrusted input and the payment is always re-loaded from the
// provider API before anything is completed.
//
// Implementations must verify that the referenced provider payment belongs to
// the RYZE purchase and that its amount and currency match the immutable
// purchase snapshot. A capture is idempotent: re-capturing an
// already-captured payment is a success, never a duplicate charge.
type CaptureProvider interface {
	// CapturePayment captures an approved payment. The request contains only
	// server-authoritative values taken from the purchase snapshot; the
	// payment identifier is always considered untrusted input and is bound to
	// the purchase by the provider before any capture happens.
	CapturePayment(ctx context.Context, request CaptureRequest) (CaptureResult, error)
}
