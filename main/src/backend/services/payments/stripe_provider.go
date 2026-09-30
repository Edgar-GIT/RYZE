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

// stripePaymentMethodTypeCard and stripePaymentMethodTypeMBWay are Stripe's
// own API identifiers for the payment method types RYZE offers through Stripe.
// They are provider-specific values and deliberately live only in this file:
// no generic RYZE code ever depends on them.
const (
	stripePaymentMethodTypeCard  = "card"
	stripePaymentMethodTypeMBWay = "mb_way"
)

// Stripe documents MB WAY as a Portuguese digital wallet that can only be
// presented and settled in Euro, with a supported transaction window of
// 0.50 EUR to 5,000 EUR. These are Stripe's documented limits, expressed in
// minor units, and are enforced before any API call so a purchase that MB WAY
// can never settle fails closed instead of creating an unpayable session.
const (
	stripeMBWayCurrency            = "eur"
	stripeMBWayMinAmountMinorUnits = 50
	stripeMBWayMaxAmountMinorUnits = 500000
)

// StripeProvider is the Stripe payment provider. It offers the card and MB WAY
// payment methods using Stripe-hosted Checkout Sessions, and verifies payments
// server-side on the Stripe API before RYZE treats them as successful.
//
// RYZE never receives, stores or transmits raw card data or the buyer's MB WAY
// phone number: both are entered exclusively in the Stripe-hosted Checkout
// page, which is reached through CheckoutURL. Because the flow is redirected,
// the shared capture path is used to re-verify the payment on the return leg:
// the browser only ever sends the Checkout Session ID, never a success flag or
// a purchase token.
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
	paymentMethodType, err := stripePaymentMethodType(request.Method)
	if err != nil {
		return PaymentResult{}, err
	}
	currency, err := stripeCurrencyCode(request.Currency)
	if err != nil {
		return PaymentResult{}, err
	}
	if err := validateStripeMethodCompatibility(paymentMethodType, currency, request.AmountMinorUnits); err != nil {
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
		// Only the method selected for this purchase is ever requested from
		// Stripe: card or MB WAY. No wallet beyond MB WAY, no bank transfer and
		// no other method is offered.
		PaymentMethodTypes: stripe.StringSlice([]string{paymentMethodType}),
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

// stripePaymentMethodType maps an RYZE payment method to the Stripe payment
// method type used to create the Checkout Session. Only the methods RYZE
// actually offers through Stripe are accepted; anything else — including the
// PayPal method and every unknown value — fails closed, so a routing mistake
// can never present an unsupported method to a buyer.
func stripePaymentMethodType(method PaymentMethod) (string, error) {
	switch method {
	case PaymentMethodCard:
		return stripePaymentMethodTypeCard, nil
	case PaymentMethodMBWay:
		return stripePaymentMethodTypeMBWay, nil
	default:
		return "", fmt.Errorf("stripe: payment method %q is not supported by the Stripe provider: %w", method, ErrProviderFailure)
	}
}

// IsStripePaymentMethodType reports whether a Stripe payment method type
// corresponds to the given RYZE payment method. It lets the webhook layer
// verify that a settled session was paid with the method bound to the purchase
// without exposing Stripe's own identifiers outside this file.
func IsStripePaymentMethodType(method PaymentMethod, stripeType string) bool {
	expected, err := stripePaymentMethodType(method)
	if err != nil {
		return false
	}
	return expected == stripeType
}

// validateStripeMethodCompatibility enforces the constraints Stripe documents
// for the requested payment method, using the purchase snapshot values. It runs
// before any API call so an unsupported combination fails closed instead of
// creating a Checkout Session that could never be paid.
//
// MB WAY can only be presented and settled in Euro and only for amounts
// between 0.50 EUR and 5,000 EUR. These limits are a property of the payment
// method, not of RYZE: a purchase outside the window is still perfectly valid
// and remains payable through the other advertised methods.
func validateStripeMethodCompatibility(paymentMethodType, currency string, amountMinorUnits int64) error {
	if paymentMethodType != stripePaymentMethodTypeMBWay {
		return nil
	}
	if currency != stripeMBWayCurrency {
		return fmt.Errorf("stripe: MB WAY can only be paid in %s, purchase is in %s: %w", stripeMBWayCurrency, currency, ErrProviderFailure)
	}
	if amountMinorUnits < stripeMBWayMinAmountMinorUnits || amountMinorUnits > stripeMBWayMaxAmountMinorUnits {
		return fmt.Errorf("stripe: MB WAY supports amounts between %d and %d minor units, purchase is %d: %w",
			stripeMBWayMinAmountMinorUnits, stripeMBWayMaxAmountMinorUnits, amountMinorUnits, ErrProviderFailure)
	}
	return nil
}

// stripeSessionIsSuccessfulPayment reports whether a Checkout Session is a
// fully paid one-time payment that belongs to the RYZE purchase, was paid with
// the recorded payment method, and matches the immutable amount and currency of
// the purchase snapshot.
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
	if !currencyMatches(string(reloaded.Currency), request.Currency) {
		return false
	}
	// The session must have been paid with the method bound to the purchase at
	// initiation. This closes the last gap between the recorded method and the
	// verified payment: a session created for a different method can never
	// complete a purchase, even if it carries the right reference and amount.
	return stripeSessionUsesRecordedMethod(reloaded, request.Method)
}

// stripeSessionUsesRecordedMethod verifies that a Checkout Session was created
// for the payment method recorded on the purchase. An empty recorded method is
// treated as a failure: a purchase always carries a method once a payment has
// been initiated, so an empty value means the caller could not prove which
// method this session belongs to.
func stripeSessionUsesRecordedMethod(reloaded *stripe.CheckoutSession, recordedMethod PaymentMethod) bool {
	if recordedMethod == "" {
		return false
	}
	expected, err := stripePaymentMethodType(recordedMethod)
	if err != nil {
		return false
	}
	for _, methodType := range reloaded.PaymentMethodTypes {
		if methodType == expected {
			return true
		}
	}
	return false
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
	// The shape check is the shared one used by every provider so a currency
	// that is unusable for one is unusable for all; only the case convention
	// differs, and Stripe expects lower case.
	if err := ValidateCurrencyCode(purchaseCurrency); err != nil {
		return "", fmt.Errorf("stripe: %w: %w", err, ErrProviderFailure)
	}
	return strings.ToLower(strings.TrimSpace(purchaseCurrency)), nil
}

// currencyMatches compares a provider currency code with the purchase
// currency code. Provider codes are upper case while RYZE stores the ISO 4217
// code in upper case, so the comparison is case-insensitive.
func currencyMatches(providerCurrency, purchaseCurrency string) bool {
	return strings.EqualFold(strings.TrimSpace(providerCurrency), strings.TrimSpace(purchaseCurrency))
}
