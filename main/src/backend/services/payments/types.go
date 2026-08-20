package payments

// PaymentMethod represents the payment method selected by the client. The
// server maps each method to the appropriate provider; the client never
// selects a provider directly.
type PaymentMethod string

const (
	// PaymentMethodCard selects card payment via Stripe Checkout.
	PaymentMethodCard PaymentMethod = "card"
	// PaymentMethodMBWay selects MB WAY payment via Stripe Checkout.
	PaymentMethodMBWay PaymentMethod = "mbway"
	// PaymentMethodPayPal selects PayPal payment via the PayPal provider.
	PaymentMethodPayPal PaymentMethod = "paypal"
)

// PaymentStatus represents the minimal provider-independent status of a
// payment. This is NOT the same as the RYZE Purchase status. A provider saying
// "paid" must eventually flow through the verified provider event to
// CompletePurchase().
type PaymentStatus string

const (
	// PaymentStatusPending indicates the payment has been initiated and is
	// awaiting completion or user action.
	PaymentStatusPending PaymentStatus = "pending"
	// PaymentStatusRequiresAction indicates the user must complete an
	// additional step (e.g. 3-D Secure, redirect) before the payment can
	// proceed.
	PaymentStatusRequiresAction PaymentStatus = "requires_action"
	// PaymentStatusFailed indicates the payment could not be completed.
	PaymentStatusFailed PaymentStatus = "failed"
)

// PaymentRequest contains the server-side trusted commercial information
// needed by a provider to initiate payment for an existing pending purchase.
// All values come exclusively from the immutable purchase snapshot; no
// client-supplied values are ever accepted.
type PaymentRequest struct {
	// PurchaseID is the RYZE purchase identifier.
	PurchaseID string
	// AmountMinorUnits is the payment amount in minor currency units, taken
	// directly from the purchase snapshot.
	AmountMinorUnits int64
	// Currency is the ISO 4217 currency code, taken directly from the
	// purchase snapshot.
	Currency string
	// ProgramID is the purchased program identifier, provided for provider
	// metadata purposes.
	ProgramID string
	// Method is the payment method selected by the client. The service
	// validates this value before it reaches the provider.
	Method PaymentMethod
}

// PaymentResult is the provider-independent representation of the outcome of
// initiating a payment. It contains only information the application actually
// needs to correlate the provider event with the RYZE purchase.
type PaymentResult struct {
	// PaymentID is the provider-specific payment identifier.
	PaymentID string
	// Status is the current provider-independent payment status.
	Status PaymentStatus
	// CheckoutURL is an optional URL the client may redirect the user to in
	// order to complete the payment (e.g. a Stripe Checkout session URL).
	// Empty when the provider does not use redirect-based flows.
	CheckoutURL string
	// Provider is the identifier of the provider that handled this payment
	// request (e.g. "stripe", "test").
	Provider string
	// PurchaseID is the RYZE purchase identifier this payment relates to.
	PurchaseID string
}
