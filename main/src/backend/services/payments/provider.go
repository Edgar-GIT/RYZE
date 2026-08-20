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
