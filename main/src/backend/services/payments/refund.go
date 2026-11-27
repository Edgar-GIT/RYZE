package payments

import (
	"context"
	"errors"
	"fmt"
	"strings"

	stripe "github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/paymentintent"
	"github.com/stripe/stripe-go/v86/refund"
)

// ErrRefundPaymentNotFound: the provider has no payment for the purchase
// (paid before payments carried the purchase, or paid elsewhere); refund it
// in the provider's dashboard and record it as already refunded.
var ErrRefundPaymentNotFound = errors.New("payments: no provider payment found for the purchase")

// RefundRequest refunds the whole payment of a purchase.
type RefundRequest struct {
	PurchaseID       string
	AmountMinorUnits int64
	Currency         string
}

// RefundResult is the provider's reference of the refund.
type RefundResult struct {
	ProviderRef string
}

// Refunder is the optional capability of returning a purchase's payment
// (owner request, 2026-11-24: refunds for non-conformity and withdrawal).
type Refunder interface {
	RefundPayment(ctx context.Context, request RefundRequest) (RefundResult, error)
}

// RefundPayment finds the purchase's succeeded payment (its metadata carries
// the purchase) and refunds it in full. The idempotency key makes a retried
// refund return the same refund instead of a second one.
func (p *StripeProvider) RefundPayment(ctx context.Context, request RefundRequest) (RefundResult, error) {
	if strings.TrimSpace(request.PurchaseID) == "" || request.AmountMinorUnits <= 0 {
		return RefundResult{}, fmt.Errorf("stripe: invalid refund request: %w", ErrProviderFailure)
	}
	if stripe.Key == "" {
		return RefundResult{}, fmt.Errorf("stripe: provider is not configured: %w", ErrProviderFailure)
	}
	search := &stripe.PaymentIntentSearchParams{}
	search.Context = ctx
	search.Query = fmt.Sprintf("metadata['purchase_id']:'%s' AND status:'succeeded'", strings.ReplaceAll(request.PurchaseID, "'", ""))
	iter := paymentintent.Search(search)
	var intent *stripe.PaymentIntent
	for iter.Next() {
		candidate := iter.PaymentIntent()
		if candidate.Amount == request.AmountMinorUnits && strings.EqualFold(string(candidate.Currency), request.Currency) {
			intent = candidate
			break
		}
	}
	if err := iter.Err(); err != nil {
		return RefundResult{}, fmt.Errorf("stripe: could not search the payment: %w: %w", ErrProviderFailure, err)
	}
	if intent == nil {
		return RefundResult{}, ErrRefundPaymentNotFound
	}
	params := &stripe.RefundParams{PaymentIntent: stripe.String(intent.ID)}
	params.Context = ctx
	params.SetIdempotencyKey("ryze-refund-" + request.PurchaseID)
	created, err := refund.New(params)
	if err != nil {
		return RefundResult{}, fmt.Errorf("stripe: refund failed: %w: %w", ErrProviderFailure, err)
	}
	return RefundResult{ProviderRef: created.ID}, nil
}

var _ Refunder = (*StripeProvider)(nil)
