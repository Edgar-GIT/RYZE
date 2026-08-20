package payments

import (
	"context"
	"fmt"
	"strings"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
)

// StripeProvider implements the Provider interface using Stripe Checkout
// Sessions. It creates a new Checkout Session for each payment initiation
// request, using the immutable purchase snapshot to populate the line item and
// returning the hosted Checkout URL for client-side redirect. The provider
// never marks a purchase as completed: that responsibility belongs to the
// verified provider event → CompletePurchase() flow.
//
// StripeProvider is safe for concurrent use by multiple goroutines.
type StripeProvider struct {
	successURL string
	cancelURL  string
}

// NewStripeProvider returns a StripeProvider configured with the given success
// and cancel URLs. Both URLs are optional: when empty, Stripe uses its default
// redirect behaviour. The Stripe API key must be set on the global
// stripe.Key before calling this constructor.
func NewStripeProvider(successURL, cancelURL string) *StripeProvider {
	return &StripeProvider{
		successURL: strings.TrimSpace(successURL),
		cancelURL:  strings.TrimSpace(cancelURL),
	}
}

// InitiatePayment creates a Stripe Checkout Session for the given purchase.
// The session is configured in payment mode with a single line item whose
// amount and currency come exclusively from the immutable purchase snapshot.
// The purchase ID is used as the client reference for reconciliation and as
// the Stripe idempotency key to prevent duplicate session creation for the
// same purchase.
//
// On success, the returned PaymentResult contains the Stripe Checkout Session
// ID as PaymentID and the hosted Checkout URL for client redirect. The status
// is always PaymentStatusRequiresAction because the user must visit the
// Checkout URL to complete payment.
func (p *StripeProvider) InitiatePayment(_ context.Context, request PaymentRequest) (PaymentResult, error) {
	if request.PurchaseID == "" {
		return PaymentResult{}, fmt.Errorf("stripe: purchase ID is required: %w", ErrProviderFailure)
	}
	if request.AmountMinorUnits <= 0 {
		return PaymentResult{}, fmt.Errorf("stripe: amount must be positive: %w", ErrProviderFailure)
	}
	if request.Currency == "" {
		return PaymentResult{}, fmt.Errorf("stripe: currency is required: %w", ErrProviderFailure)
	}

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(strings.ToLower(request.Currency)),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(fmt.Sprintf("Program %s", request.ProgramID)),
					},
					UnitAmount: stripe.Int64(request.AmountMinorUnits),
				},
				Quantity: stripe.Int64(1),
			},
		},
		ClientReferenceID: stripe.String(request.PurchaseID),
		Metadata: map[string]string{
			"purchase_id": request.PurchaseID,
			"program_id":  request.ProgramID,
		},
	}

	if p.successURL != "" {
		params.SuccessURL = stripe.String(p.successURL)
	}
	if p.cancelURL != "" {
		params.CancelURL = stripe.String(p.cancelURL)
	}

	params.SetIdempotencyKey(fmt.Sprintf("ryze-purchase-%s", request.PurchaseID))

	s, err := session.New(params)
	if err != nil {
		return PaymentResult{}, fmt.Errorf("stripe: session creation failed: %w", ErrProviderFailure)
	}

	if s.URL == "" {
		return PaymentResult{}, fmt.Errorf("stripe: session created without checkout URL: %w", ErrProviderFailure)
	}

	return PaymentResult{
		PaymentID:   s.ID,
		Status:      PaymentStatusRequiresAction,
		CheckoutURL: s.URL,
		Provider:    "stripe",
		PurchaseID:  request.PurchaseID,
	}, nil
}
