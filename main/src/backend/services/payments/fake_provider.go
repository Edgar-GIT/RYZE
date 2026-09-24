package payments

import (
	"context"
	"fmt"
	"sync/atomic"
)

// FakeProvider is a deterministic test-only payment provider. It never performs
// network requests, never processes real money and always returns predictable
// payment identifiers. It is designed exclusively for automated tests and must
// never be exposed in production configuration.
type FakeProvider struct {
	// FailWhen true causes InitiatePayment to return an error, enabling
	// provider-failure test scenarios.
	FailWhen bool
	// paymentCounter produces unique sequential payment identifiers.
	paymentCounter int64
}

// NewFakeProvider returns a FakeProvider ready for use in tests.
func NewFakeProvider() *FakeProvider {
	return &FakeProvider{}
}

// InitiatePayment returns a deterministic fake payment result. When FailWhen is
// true it returns an error instead.
func (f *FakeProvider) InitiatePayment(_ context.Context, request PaymentRequest) (PaymentResult, error) {
	if f.FailWhen {
		return PaymentResult{}, fmt.Errorf("fake provider: simulated failure: %w", ErrProviderFailure)
	}

	id := atomic.AddInt64(&f.paymentCounter, 1)

	return PaymentResult{
		PaymentID:   fmt.Sprintf("fake_pay_%d", id),
		Status:      PaymentStatusPending,
		CheckoutURL: fmt.Sprintf("https://fake-checkout.example.com/pay/%d", id),
		Provider:    "fake",
		PurchaseID:  request.PurchaseID,
	}, nil
}

// CapturePayment returns a deterministic successful capture for any non-empty
// payment id. It requires a non-empty purchase id so capture requests always
// carry the immutable purchase reference.
func (f *FakeProvider) CapturePayment(_ context.Context, request CaptureRequest) (CaptureResult, error) {
	if request.PurchaseID == "" {
		return CaptureResult{}, fmt.Errorf("fake provider: missing purchase id: %w", ErrProviderFailure)
	}
	if request.PaymentID == "" {
		return CaptureResult{}, fmt.Errorf("fake provider: missing payment id: %w", ErrProviderFailure)
	}

	return CaptureResult{
		PaymentID: request.PaymentID,
		Provider:  "fake",
	}, nil
}

// Ensure FakeProvider satisfies the Provider and CaptureProvider interfaces at
// compile time.
var (
	_ Provider        = (*FakeProvider)(nil)
	_ CaptureProvider = (*FakeProvider)(nil)
)
