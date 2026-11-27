// Package refunds lets ADMIN_2 refund a completed purchase (owner request,
// 2026-11-24): for non-conformity of the digital content, under the right of
// withdrawal (Premium Level 2), a duplicate or goodwill. The money goes back
// through the payment provider when RYZE can find the payment there (Stripe
// payments made since v2.131.0); otherwise the administrator refunds it in the
// provider's dashboard and records it here. Either way the access ends at
// once and the sale leaves the trainer's earnings.
package refunds

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/payments"
)

var (
	ErrInvalidInput   = errors.New("invalid refund")
	ErrNotFound       = errors.New("purchase not found")
	ErrNotRefundable  = errors.New("purchase cannot be refunded")
	ErrManualRequired = errors.New("refund the payment in the provider's dashboard first")
	ErrProvider       = errors.New("the provider could not refund the payment")
)

// Reasons accepted for a refund.
var Reasons = map[string]bool{"non_conformity": true, "withdrawal": true, "duplicate": true, "goodwill": true}

// Input is the administrator's refund.
type Input struct {
	Reason string
	Note   string
	// AlreadyRefunded: the money was already returned in the provider's
	// dashboard; RYZE only records it and removes the access.
	AlreadyRefunded bool
}

type Service struct {
	repo     repositories.RefundRepository
	refunder payments.Refunder
	now      func() time.Time
}

// New builds the service; refunder is the Stripe provider (nil when card
// payments are not configured).
func New(repo repositories.RefundRepository, refunder payments.Refunder) *Service {
	return &Service{repo: repo, refunder: refunder, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) Refund(ctx context.Context, purchaseID, adminID string, input Input) (*models.Purchase, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	input.Note = strings.TrimSpace(input.Note)
	if !Reasons[input.Reason] || utf8.RuneCountInString(input.Note) > 500 || strings.TrimSpace(adminID) == "" {
		return nil, ErrInvalidInput
	}
	purchase, err := s.repo.FindRefundable(ctx, purchaseID)
	if err != nil {
		return nil, mapError(err)
	}
	providerRef := ""
	paid := !purchase.Test && purchase.PriceMinorUnits > 0
	if paid && !input.AlreadyRefunded {
		method := ""
		if purchase.PaymentMethod != nil {
			method = *purchase.PaymentMethod
		}
		if s.refunder == nil || (method != "card" && method != "mbway") {
			// PayPal (or an unknown method): refunded in its dashboard.
			return nil, ErrManualRequired
		}
		result, err := s.refunder.RefundPayment(ctx, payments.RefundRequest{
			PurchaseID: purchase.ID, AmountMinorUnits: purchase.PriceMinorUnits, Currency: purchase.Currency,
		})
		switch {
		case errors.Is(err, payments.ErrRefundPaymentNotFound):
			return nil, ErrManualRequired
		case err != nil:
			return nil, fmt.Errorf("%w: %v", ErrProvider, err)
		}
		providerRef = result.ProviderRef
	}
	refunded, err := s.repo.Refund(ctx, purchase.ID, repositories.RefundRecord{
		Reason: input.Reason, Note: input.Note, AdminID: adminID, ProviderRef: providerRef, At: s.now(),
	})
	if err != nil {
		if providerRef != "" {
			// The money went back but the record failed: retrying is safe (the
			// provider refund is idempotent per purchase).
			slog.Error("refund recorded at the provider but not in RYZE", slog.String("purchase_id", purchase.ID), slog.String("error", err.Error()))
		}
		return nil, mapError(err)
	}
	return refunded, nil
}

func mapError(err error) error {
	switch {
	case errors.Is(err, repositories.ErrPurchaseNotFound):
		return ErrNotFound
	case errors.Is(err, repositories.ErrRefundNotAllowed):
		return ErrNotRefundable
	}
	return err
}
