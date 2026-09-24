package payments

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	paypal "github.com/plutov/paypal/v4"
)

// PayPalProvider implements the Provider interface using the PayPal Orders API.
// It creates a new PayPal Order for each payment initiation request, returning
// the PayPal approval URL for client-side redirect. The provider never marks a
// purchase as completed: that responsibility belongs to the verified provider
// event → CompletePurchase() flow.
//
// PayPalProvider additionally implements CaptureProvider: it can capture an
// approved PayPal Order server-side. Captures always verify that the order
// belongs to the RYZE purchase (reference ID) and that its amount and currency
// match the immutable purchase snapshot before any capture happens.
//
// PayPalProvider is safe for concurrent use by multiple goroutines.
type PayPalProvider struct {
	client    *paypal.Client
	returnURL string
	cancelURL string
}

// NewPayPalProvider returns a PayPalProvider configured with the given client
// ID, secret and mode. The mode must be "sandbox" or "live" to select the
// appropriate PayPal API base URL.
func NewPayPalProvider(clientID, secret, mode string) (*PayPalProvider, error) {
	clientID = strings.TrimSpace(clientID)
	secret = strings.TrimSpace(secret)
	mode = strings.TrimSpace(strings.ToLower(mode))

	if clientID == "" || secret == "" {
		return nil, fmt.Errorf("paypal: client ID and secret are required")
	}

	var apiBase string
	switch mode {
	case "sandbox":
		apiBase = paypal.APIBaseSandBox
	case "live":
		apiBase = paypal.APIBaseLive
	default:
		return nil, fmt.Errorf("paypal: mode must be 'sandbox' or 'live', got %q", mode)
	}

	client, err := paypal.NewClient(clientID, secret, apiBase)
	if err != nil {
		return nil, fmt.Errorf("paypal: failed to create client: %w", ErrProviderFailure)
	}

	return &PayPalProvider{client: client}, nil
}

// SetHTTPClient overrides the HTTP client used by the PayPal SDK. This is
// intended for testing with mock servers and must not be used in production.
func (p *PayPalProvider) SetHTTPClient(c *http.Client) {
	p.client.SetHTTPClient(c)
}

// SetCheckoutRedirectURLs configures the URLs PayPal redirects the buyer to
// after approving or cancelling a payment. Both templates may contain the
// "{program_id}" and "{purchase_id}" placeholders, which are replaced with the
// values of the current purchase when each order is created. PayPal appends the
// order token to both URLs automatically. When a URL is empty, the
// corresponding redirect is not configured on the order.
func (p *PayPalProvider) SetCheckoutRedirectURLs(returnURL, cancelURL string) {
	p.returnURL = returnURL
	p.cancelURL = cancelURL
}

// InitiatePayment creates a PayPal Order for the given purchase. The order is
// configured in CAPTURE intent mode with a single purchase unit whose amount
// and currency come exclusively from the immutable purchase snapshot. The
// purchase ID is used as the reference ID and as the PayPal-Request-Id header
// for idempotency.
//
// On success, the returned PaymentResult contains the PayPal Order ID as
// PaymentID and the approval URL for client redirect. The status is always
// PaymentStatusRequiresAction because the user must visit the approval URL to
// complete payment.
func (p *PayPalProvider) InitiatePayment(_ context.Context, request PaymentRequest) (PaymentResult, error) {
	if request.PurchaseID == "" {
		return PaymentResult{}, fmt.Errorf("paypal: purchase ID is required: %w", ErrProviderFailure)
	}
	if request.AmountMinorUnits <= 0 {
		return PaymentResult{}, fmt.Errorf("paypal: amount must be positive: %w", ErrProviderFailure)
	}
	if request.Currency == "" {
		return PaymentResult{}, fmt.Errorf("paypal: currency is required: %w", ErrProviderFailure)
	}

	amountValue := minorUnitsToDecimalString(request.AmountMinorUnits)

	purchaseUnits := []paypal.PurchaseUnitRequest{
		{
			ReferenceID: request.PurchaseID,
			Amount: &paypal.PurchaseUnitAmount{
				Currency: strings.ToUpper(request.Currency),
				Value:    amountValue,
			},
		},
	}

	requestID := fmt.Sprintf("ryze-purchase-%s", request.PurchaseID)

	var applicationContext *paypal.ApplicationContext
	if p.returnURL != "" || p.cancelURL != "" {
		applicationContext = &paypal.ApplicationContext{
			ReturnURL: substituteRedirectPlaceholders(p.returnURL, request),
			CancelURL: substituteRedirectPlaceholders(p.cancelURL, request),
		}
	}

	order, err := p.client.CreateOrderWithPaypalRequestID(
		context.Background(),
		paypal.OrderIntentCapture,
		purchaseUnits,
		nil,
		applicationContext,
		requestID,
	)
	if err != nil {
		return PaymentResult{}, fmt.Errorf("paypal: order creation failed: %w", ErrProviderFailure)
	}

	approvalURL := findPayPalApprovalURL(order)

	return PaymentResult{
		PaymentID:   order.ID,
		Status:      PaymentStatusRequiresAction,
		CheckoutURL: approvalURL,
		Provider:    "paypal",
		PurchaseID:  request.PurchaseID,
	}, nil
}

// CapturePayment captures an approved PayPal Order and verifies it belongs to
// the RYZE purchase. The order is loaded from the PayPal API (never trusted
// from the client), its purchase unit must reference the RYZE purchase id and
// its amount and currency must match the immutable purchase snapshot. Orders
// already in COMPLETED status are treated as a successful idempotent capture.
func (p *PayPalProvider) CapturePayment(_ context.Context, request CaptureRequest) (CaptureResult, error) {
	if request.PurchaseID == "" {
		return CaptureResult{}, fmt.Errorf("paypal: purchase ID is required: %w", ErrProviderFailure)
	}
	if request.PaymentID == "" {
		return CaptureResult{}, fmt.Errorf("paypal: payment ID is required: %w", ErrProviderFailure)
	}
	if request.AmountMinorUnits <= 0 {
		return CaptureResult{}, fmt.Errorf("paypal: amount must be positive: %w", ErrProviderFailure)
	}
	if request.Currency == "" {
		return CaptureResult{}, fmt.Errorf("paypal: currency is required: %w", ErrProviderFailure)
	}

	order, err := p.client.GetOrder(context.Background(), request.PaymentID)
	if err != nil {
		return CaptureResult{}, fmt.Errorf("paypal: get order failed: %w", ErrProviderFailure)
	}
	if order == nil {
		return CaptureResult{}, fmt.Errorf("paypal: order not found: %w", ErrProviderFailure)
	}

	amount := minorUnitsToDecimalString(request.AmountMinorUnits)
	expectedCurrency := strings.ToUpper(request.Currency)

	orderReference := ""
	orderAmount := ""
	orderCurrency := ""
	for _, unit := range order.PurchaseUnits {
		if unit.ReferenceID == request.PurchaseID {
			orderReference = unit.ReferenceID
			if unit.Amount != nil {
				orderAmount = unit.Amount.Value
				orderCurrency = unit.Amount.Currency
			}
			break
		}
	}

	if orderReference != request.PurchaseID {
		return CaptureResult{}, fmt.Errorf("paypal: order %s does not reference purchase %s: %w", order.ID, request.PurchaseID, ErrProviderFailure)
	}
	if strings.ToUpper(orderCurrency) != expectedCurrency {
		return CaptureResult{}, fmt.Errorf("paypal: order %s currency %q does not match purchase %s currency %q: %w", order.ID, orderCurrency, request.PurchaseID, request.Currency, ErrProviderFailure)
	}
	if orderAmount != amount {
		return CaptureResult{}, fmt.Errorf("paypal: order %s amount %q does not match purchase %s amount %q: %w", order.ID, orderAmount, request.PurchaseID, amount, ErrProviderFailure)
	}

	// An already-captured order (COMPLETED) is a success: the capture is
	// idempotent and must never double-charge or error.
	if order.Status == paypal.OrderStatusCompleted {
		return CaptureResult{PaymentID: order.ID, Provider: "paypal"}, nil
	}

	if order.Status != paypal.OrderStatusApproved {
		return CaptureResult{}, fmt.Errorf("paypal: order %s is not approved for capture (status=%s): %w", order.ID, order.Status, ErrProviderFailure)
	}

	captureID := fmt.Sprintf("ryze-capture-%s", request.PurchaseID)
	resp, err := p.client.CaptureOrderWithPaypalRequestId(context.Background(), order.ID, paypal.CaptureOrderRequest{}, captureID, nil)
	if err != nil {
		// A concurrent capture (e.g. the webhook racing the browser return)
		// may have completed the order between GetOrder and CaptureOrder.
		// Re-read the order and treat COMPLETED as the idempotent success
		// case instead of surfacing a duplicate-capture error.
		reloaded, reloadErr := p.client.GetOrder(context.Background(), order.ID)
		if reloadErr == nil && reloaded != nil && reloaded.Status == paypal.OrderStatusCompleted {
			return CaptureResult{PaymentID: order.ID, Provider: "paypal"}, nil
		}
		return CaptureResult{}, fmt.Errorf("paypal: order capture failed: %w", ErrProviderFailure)
	}

	if !verifyPayPalCapturedAmount(resp, request.AmountMinorUnits, request.Currency) {
		if !orderIsCompleted(p.client, order.ID) {
			return CaptureResult{}, fmt.Errorf("paypal: captured amount for order %s could not be verified: %w", order.ID, ErrProviderFailure)
		}
	}

	return CaptureResult{PaymentID: order.ID, Provider: "paypal"}, nil
}

// substituteRedirectPlaceholders replaces the "{program_id}" and
// "{purchase_id}" placeholders inside a redirect URL template with the current
// purchase values. Unknown or absent placeholders are left untouched.
func substituteRedirectPlaceholders(template string, request PaymentRequest) string {
	return strings.NewReplacer(
		"{program_id}", request.ProgramID,
		"{purchase_id}", request.PurchaseID,
	).Replace(template)
}

// verifyPayPalCapturedAmount checks that the capture response contains a
// completed capture whose amount and currency match the purchase snapshot.
func verifyPayPalCapturedAmount(resp *paypal.CaptureOrderResponse, amountMinorUnits int64, currency string) bool {
	if resp == nil {
		return false
	}
	wantAmount := minorUnitsToDecimalString(amountMinorUnits)
	wantCurrency := strings.ToUpper(currency)
	for _, unit := range resp.PurchaseUnits {
		if unit.Payments == nil {
			continue
		}
		for _, capture := range unit.Payments.Captures {
			if capture.Amount == nil {
				continue
			}
			if strings.EqualFold(capture.Amount.Currency, wantCurrency) && capture.Amount.Value == wantAmount {
				return true
			}
		}
	}
	return false
}

// orderIsCompleted reloads the order and reports whether it reached the
// COMPLETED status, used as the final fallback when a capture response cannot
// be verified.
func orderIsCompleted(client *paypal.Client, orderID string) bool {
	order, err := client.GetOrder(context.Background(), orderID)
	return err == nil && order != nil && order.Status == paypal.OrderStatusCompleted
}

// minorUnitsToDecimalString converts a minor currency units amount (e.g. 4999
// cents) to a decimal string (e.g. "49.99") as required by the PayPal API.
func minorUnitsToDecimalString(minorUnits int64) string {
	negative := minorUnits < 0
	if negative {
		minorUnits = -minorUnits
	}

	whole := minorUnits / 100
	fraction := minorUnits % 100

	if fraction == 0 {
		if negative {
			return fmt.Sprintf("-%d.00", whole)
		}
		return fmt.Sprintf("%d.00", whole)
	}

	if negative {
		return fmt.Sprintf("-%d.%02d", whole, fraction)
	}
	return fmt.Sprintf("%d.%02d", whole, fraction)
}

// findPayPalApprovalURL extracts the approval URL from a PayPal Order response.
// The approval URL is the URL the user must visit to approve the payment. It
// is identified by the rel="approve" link. If no approval URL is found, an
// empty string is returned.
func findPayPalApprovalURL(order *paypal.Order) string {
	if order == nil {
		return ""
	}
	for _, link := range order.Links {
		if strings.EqualFold(link.Rel, "approve") {
			return link.Href
		}
	}
	return ""
}

// Ensure PayPalProvider satisfies the Provider and CaptureProvider interfaces
// at compile time.
var (
	_ Provider        = (*PayPalProvider)(nil)
	_ CaptureProvider = (*PayPalProvider)(nil)
)
