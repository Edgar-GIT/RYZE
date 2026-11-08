package purchase_fulfillment

import (
	"context"
	"errors"
)

// Fulfiller is the post-purchase delivery contract. Each product family's
// fulfiller must ignore programs it does not own, so a single completed
// purchase can safely pass through every fulfiller.
type Fulfiller interface {
	FulfillPurchase(ctx context.Context, userID, programID string) error
}

// Multiplexer runs every registered fulfiller and joins their failures. A
// failure never undoes the purchase; it is returned so the caller can record
// it while leaving each product's delivery retryable in place.
type Multiplexer struct {
	fulfillers []Fulfiller
}

func New(fulfillers ...Fulfiller) *Multiplexer {
	return &Multiplexer{fulfillers: fulfillers}
}

func (m *Multiplexer) FulfillPurchase(ctx context.Context, userID, programID string) error {
	var failures []error
	for _, fulfiller := range m.fulfillers {
		if fulfiller == nil {
			continue
		}
		if err := fulfiller.FulfillPurchase(ctx, userID, programID); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}
