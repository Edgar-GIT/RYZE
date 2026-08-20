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
// PayPalProvider is safe for concurrent use by multiple goroutines.
type PayPalProvider struct {
	client *paypal.Client
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

	order, err := p.client.CreateOrderWithPaypalRequestID(
		context.Background(),
		paypal.OrderIntentCapture,
		purchaseUnits,
		nil,
		nil,
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

// Ensure PayPalProvider satisfies the Provider interface at compile time.
var _ Provider = (*PayPalProvider)(nil)
