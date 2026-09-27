package payments

import (
	"context"
	"fmt"
	"strings"

	stripe "github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/checkout/session"
)

// stripeProviderName is the stable public identifier of the Stripe provider.
const stripeProviderName = "stripe"

// stripeCheckoutProductName is the line item name shown on the Stripe-hosted
// checkout page. It is a fixed label: it never contains customer or program
// data, so no RYZE content is disclosed to Stripe beyond what the API requires.
const stripeCheckoutProductName = "RYZE program access"

// StripeProvider is the Stripe payment provider. It offers the card payment
// method only, using Stripe-hosted Checkout Sessions, and verifies payments
// server-side on the Stripe API before RYZE treats them as successful.
//
// RYZE never receives, stores or transmits raw card data: the buyer enters it
// exclusively in the Stripe-hosted Checkout page, which is reached through
// CheckoutURL. Because the flow is redirected, the shared capture path is used
// to re-verify the payment on the return leg: the browser only ever sends the
// Checkout Session ID, never a success flag or a purchase token.
type StripeProvider struct {
	successURL string
	cancelURL  string
}

// Ensure the Stripe provider can be used for both phases of the shared
// payment lifecycle: initiation and server-side capture.
var _ Provider = (*StripeProvider)(nil)
var _ CaptureProvider = (*StripeProvider)(nil)

// NewStripeProvider creates a Stripe provider. Both redirect URL templates are
// optional: an empty template disables the corresponding redirect. Templates
// may contain the "{program_id}" and "{purchase_id}" placeholders, which are
// replaced with the values of the current purchase when each Checkout Session
// is created. The "{CHECKOUT_SESSION_ID}" placeholder is left untouched so
// Stripe can substitute the session identifier of the return leg.
func NewStripeProvider(successURL, cancelURL string) *StripeProvider {
	return &StripeProvider{
		successURL: successURL,
		cancelURL:  cancelURL,
	}
}

// InitiatePayment creates a Stripe Checkout Session for the given purchase.
// The session is a one-time payment whose total amount and currency come
// exclusively from the immutable purchase snapshot, and the purchase ID is
// carried both as the Stripe client reference ID and as session metadata.
//
// The request is issued with a per-purchase idempotency key, so a repeated
// initiation for the same purchase returns the same Checkout Session instead
// of creating a second payable session. The returned PaymentResult contains
// the Checkout Session ID as PaymentID and the Stripe-hosted page URL as
// CheckoutURL, and the status is always requires_action because the buyer must
// complete the payment on the Stripe-hosted page.
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
	currency, err := stripeCurrencyCode(request.Currency)
	if err != nil {
		return PaymentResult{}, err
	}
	if err := requireSupportedStripeMethod(request.Method); err != nil {
		return PaymentResult{}, err
	}
	if stripe.Key == "" {
		return PaymentResult{}, fmt.Errorf("stripe: provider is not configured: %w", ErrProviderFailure)
	}

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		// Both the amount and the currency are taken from the immutable purchase
		// snapshot, so the session can never be created in a currency that
		// differs from the one the purchase is verified against.
		Currency: stripe.String(currency),
		// Stripe only offers the card payment method in RYZE. No other method
		// is ever requested from Stripe.
		PaymentMethodTypes: stripe.StringSlice([]string{string(PaymentMethodCard)}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Quantity: stripe.Int64(1),
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String(currency),
					UnitAmount: stripe.Int64(request.AmountMinorUnits),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(stripeCheckoutProductName),
					},
				},
			},
		},
		// The session is bound to the RYZE purchase through the client
		// reference ID and the metadata, both of which the webhook and the
		// capture path verify before completing anything.
		ClientReferenceID: stripe.String(request.PurchaseID),
		Metadata: map[string]string{
			"purchase_id": request.PurchaseID,
			"program_id":  request.ProgramID,
		},
	}
	params.SetIdempotencyKey(stripeIdempotencyKey(request.PurchaseID))

	if p.successURL != "" {
		params.SuccessURL = stripe.String(substituteRedirectPlaceholders(p.successURL, request))
	}
	if p.cancelURL != "" {
		params.CancelURL = stripe.String(substituteRedirectPlaceholders(p.cancelURL, request))
	}

	created, err := session.New(params)
	if err != nil {
		return PaymentResult{}, fmt.Errorf("stripe: could not create checkout session: %w: %w", ErrProviderFailure, err)
	}
	if created == nil || created.ID == "" {
		return PaymentResult{}, fmt.Errorf("stripe: checkout session creation returned no session: %w", ErrProviderFailure)
	}
	if created.URL == "" {
		return PaymentResult{}, fmt.Errorf("stripe: checkout session has no hosted page: %w", ErrProviderFailure)
	}

	return PaymentResult{
		PaymentID:   created.ID,
		CheckoutURL: created.URL,
		Status:      PaymentStatusRequiresAction,
		Provider:    stripeProviderName,
		PurchaseID:  request.PurchaseID,
	}, nil
}

// CapturePayment verifies a Stripe Checkout Session as the successful payment
// of the RYZE purchase. The session is always loaded from the Stripe API
// using the session ID received on the return leg: the browser response is
// never trusted as proof of payment.
//
// The session must belong to the purchase, be a one-time payment, be fully
// paid, and match the immutable amount and currency of the purchase snapshot.
// Sessions that are not paid, expired or otherwise incomplete are rejected, so
// an unpaid session can never complete a purchase. Verification is read-only
// and therefore idempotent: an already completed session still verifies
// successfully, which lets the browser return leg and the webhook race safely.
func (p *StripeProvider) CapturePayment(_ context.Context, request CaptureRequest) (CaptureResult, error) {
	if request.PurchaseID == "" {
		return CaptureResult{}, fmt.Errorf("stripe: purchase ID is required: %w", ErrProviderFailure)
	}
	if request.PaymentID == "" {
		return CaptureResult{}, fmt.Errorf("stripe: payment ID is required: %w", ErrProviderFailure)
	}
	if request.AmountMinorUnits <= 0 {
		return CaptureResult{}, fmt.Errorf("stripe: amount must be positive: %w", ErrProviderFailure)
	}
	if request.Currency == "" {
		return CaptureResult{}, fmt.Errorf("stripe: currency is required: %w", ErrProviderFailure)
	}
	if stripe.Key == "" {
		return CaptureResult{}, fmt.Errorf("stripe: provider is not configured: %w", ErrProviderFailure)
	}

	reloaded, err := session.Get(request.PaymentID, nil)
	if err != nil {
		return CaptureResult{}, fmt.Errorf("stripe: could not load checkout session: %w: %w", ErrProviderFailure, err)
	}
	if !stripeSessionIsSuccessfulPayment(reloaded, request) {
		return CaptureResult{}, fmt.Errorf("stripe: checkout session is not a successful payment of the purchase: %w", ErrProviderFailure)
	}

	return CaptureResult{
		PaymentID: reloaded.ID,
		Provider:  stripeProviderName,
	}, nil
}

// requireSupportedStripeMethod rejects any payment method that the Stripe
// provider does not serve. Only the card method is offered: no other Stripe
// payment method, wallet or bank transfer is presented by RYZE. This check
// fails closed so a routing mistake cannot send an unsupported method to
// Stripe.
func requireSupportedStripeMethod(method PaymentMethod) error {
	if method == PaymentMethodCard {
		return nil
	}
	return fmt.Errorf("stripe: payment method %q is not supported by the Stripe provider: %w", method, ErrProviderFailure)
}

// stripeSessionIsSuccessfulPayment reports whether a Checkout Session is a
// fully paid one-time payment that belongs to the RYZE purchase and matches
// the immutable amount and currency of the purchase snapshot.
func stripeSessionIsSuccessfulPayment(reloaded *stripe.CheckoutSession, request CaptureRequest) bool {
	if reloaded == nil || reloaded.ID == "" {
		return false
	}
	if reloaded.Mode != stripe.CheckoutSessionModePayment {
		return false
	}
	if reloaded.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
		return false
	}
	if reloaded.ClientReferenceID != request.PurchaseID {
		return false
	}
	if reloaded.AmountTotal != request.AmountMinorUnits {
		return false
	}
	return currencyMatches(string(reloaded.Currency), request.Currency)
}

// stripeIdempotencyKey builds the per-purchase Stripe idempotency key. Stripe
// replays the original Checkout Session for a repeated key, so one purchase
// can never hold two payable sessions.
func stripeIdempotencyKey(purchaseID string) string {
	return "ryze-purchase-" + purchaseID
}

// stripeCurrencyCode normalises the purchase currency into the lower-case ISO
// 4217 form Stripe expects. The value always originates from the immutable
// purchase snapshot: an unusable currency fails closed before any session is
// created, so a purchase can never be paid in a different currency than the one
// it is verified against.
func stripeCurrencyCode(purchaseCurrency string) (string, error) {
	currency := strings.ToLower(strings.TrimSpace(purchaseCurrency))
	if len(currency) != 3 {
		return "", fmt.Errorf("stripe: currency %q is not a supported currency code: %w", purchaseCurrency, ErrProviderFailure)
	}
	for _, char := range currency {
		if char < 'a' || char > 'z' {
			return "", fmt.Errorf("stripe: currency %q is not a supported currency code: %w", purchaseCurrency, ErrProviderFailure)
		}
	}
	return currency, nil
}

// currencyMatches compares a provider currency code with the purchase
// currency code. Provider codes are upper case while RYZE stores the ISO 4217
// code in upper case, so the comparison is case-insensitive.
func currencyMatches(providerCurrency, purchaseCurrency string) bool {
	return strings.EqualFold(strings.TrimSpace(providerCurrency), strings.TrimSpace(purchaseCurrency))
}
