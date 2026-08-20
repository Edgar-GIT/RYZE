package payments

import "errors"

var (
	// ErrProviderFailure indicates the payment provider returned an error or
	// could not initiate the payment. The caller must not expose internal
	// provider details to the client.
	ErrProviderFailure = errors.New("payment provider failure")

	// ErrInvalidPaymentMethod indicates the client provided an unsupported or
	// missing payment method.
	ErrInvalidPaymentMethod = errors.New("invalid payment method")

	// ErrNoProviderAvailable indicates no payment provider is configured for
	// the requested payment method.
	ErrNoProviderAvailable = errors.New("no payment provider available for this method")
)
