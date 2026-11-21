package premium_level_2

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ryze/backend/email"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/notifications"
)

// Allocation by offer (owner decision, 2026-11-21, until AI exists): after
// the intake the request is offered to the available trainer with the fewest
// open requests; accepting allocates it, declining or 24 hours without an
// answer passes it to the next trainer; with nobody left ADMIN_2 allocates it
// by hand, as before.

var (
	ErrOfferNotFound = errors.New("premium level 2 offer not found")
	ErrOfferExpired  = errors.New("premium level 2 offer expired")
)

// pendingBatch bounds how many waiting requests one sweep offers.
const pendingBatch = 50

// OfferService is the trainer side of allocation by offer.
type OfferService interface {
	// ListOffers returns the trainer's open offers and whether they accept
	// new requests.
	ListOffers(ctx context.Context, trainerID string) ([]repositories.PremiumLevel2OfferView, bool, error)
	AcceptOffer(ctx context.Context, trainerID, trainerUserID, offerID string) (*models.PremiumLevel2Request, error)
	DeclineOffer(ctx context.Context, trainerID, trainerUserID, offerID string) error
	SetAvailability(ctx context.Context, trainerID string, accepts bool) error
	// SweepOffers expires overdue offers and offers every waiting request.
	SweepOffers(ctx context.Context) error
}

// TrainerUsers resolves the account of a trainer (for the notification).
type TrainerUsers interface {
	FindTrainerUserID(ctx context.Context, trainerID string) (string, error)
}

// WithOffers enables allocation by offer.
func WithOffers(offers repositories.PremiumLevel2OfferRepository) ServiceOption {
	return func(s *service) { s.offers = offers }
}

func (s *service) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now().UTC()
}

// offerNext offers the request to the next available trainer, if any.
func (s *service) offerNext(ctx context.Context, requestID string) error {
	if s.offers == nil {
		return nil
	}
	trainerID, err := s.offers.PickTrainer(ctx, requestID)
	if err != nil || trainerID == "" {
		return err
	}
	offer, created, err := s.offers.CreateOffer(ctx, requestID, trainerID, s.clock().Add(models.PremiumLevel2OfferWindow))
	if err != nil || !created {
		return err
	}
	if s.notifier == nil {
		return nil
	}
	request, err := s.requests.FindRequestByID(ctx, requestID)
	if err != nil {
		return nil
	}
	users, ok := s.offers.(TrainerUsers)
	if !ok {
		return nil
	}
	if trainerUserID, err := users.FindTrainerUserID(ctx, trainerID); err == nil && trainerUserID != "" {
		s.notifier.Notify(ctx, notifications.Event{
			UserID:        trainerUserID,
			Kind:          models.NotificationPL2RequestOffered,
			Program:       request.OfferName,
			Link:          "/trainer/premium-level-2",
			DedupeKey:     "pl2_offered:" + offer.ID,
			EmailTemplate: email.TemplateRequestOffered,
		})
	}
	return nil
}

func (s *service) ListOffers(ctx context.Context, trainerID string) ([]repositories.PremiumLevel2OfferView, bool, error) {
	if s.offers == nil {
		return nil, false, nil
	}
	offers, err := s.offers.ListOpenOffers(ctx, trainerID)
	if err != nil {
		return nil, false, err
	}
	accepts, err := s.offers.TrainerAvailability(ctx, trainerID)
	if err != nil {
		return nil, false, err
	}
	return offers, accepts, nil
}

func offerError(err error) error {
	switch {
	case errors.Is(err, repositories.ErrPremiumLevel2OfferNotFound):
		return ErrOfferNotFound
	case errors.Is(err, repositories.ErrPremiumLevel2OfferExpired):
		return ErrOfferExpired
	case errors.Is(err, repositories.ErrPremiumLevel2RequestNotFound):
		return ErrRequestNotFound
	case errors.Is(err, repositories.ErrPremiumLevel2InvalidState):
		return ErrInvalidState
	}
	return err
}

func (s *service) AcceptOffer(ctx context.Context, trainerID, trainerUserID, offerID string) (*models.PremiumLevel2Request, error) {
	if s.offers == nil {
		return nil, ErrOfferNotFound
	}
	request, err := s.offers.AcceptOffer(ctx, offerID, trainerID, trainerUserID, s.clock())
	if err != nil {
		if errors.Is(err, repositories.ErrPremiumLevel2OfferExpired) {
			// The window closed: the request moves on to the next trainer.
			_ = s.SweepOffers(ctx)
		}
		return nil, offerError(err)
	}
	s.notifyAssigned(ctx, request.ID, trainerID)
	return request, nil
}

func (s *service) DeclineOffer(ctx context.Context, trainerID, trainerUserID, offerID string) error {
	if s.offers == nil {
		return ErrOfferNotFound
	}
	requestID, err := s.offers.DeclineOffer(ctx, offerID, trainerID, trainerUserID, s.clock())
	if err != nil {
		return offerError(err)
	}
	if err := s.offerNext(ctx, requestID); err != nil {
		return fmt.Errorf("failed to offer the request to the next trainer: %w", err)
	}
	return nil
}

func (s *service) SetAvailability(ctx context.Context, trainerID string, accepts bool) error {
	if s.offers == nil {
		return ErrInvalidInput
	}
	if err := s.offers.SetTrainerAvailability(ctx, trainerID, accepts); err != nil {
		return err
	}
	if accepts {
		// Requests waiting with nobody to offer them to go out now.
		return s.SweepOffers(ctx)
	}
	return nil
}

func (s *service) SweepOffers(ctx context.Context) error {
	if s.offers == nil {
		return nil
	}
	if _, err := s.offers.ExpireOffers(ctx, s.clock()); err != nil {
		return err
	}
	waiting, err := s.offers.RequestsAwaitingOffer(ctx, pendingBatch)
	if err != nil {
		return err
	}
	var firstErr error
	for _, requestID := range waiting {
		if err := s.offerNext(ctx, requestID); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
