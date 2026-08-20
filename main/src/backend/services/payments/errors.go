package payments

import "errors"

var (
	// ErrProviderFailure indicates the payment provider returned an error or
	// could not initiate the payment. The caller must not expose internal
	// provider details to the client.
	ErrProviderFailure = errors.New("payment provider failure")
)
