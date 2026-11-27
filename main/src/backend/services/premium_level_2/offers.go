package premium_level_2

import (
	"context"
	"errors"
	"time"

	"ryze/backend/email"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/notifications"
)

// Allocation by offer (owner decision, 2026-11-24, until AI exists): after
// the intake the request is offered at once to every trainer who accepts
// Premium Level 2 requests; the first to accept takes it and the other offers
// are withdrawn. Offers close after 24 hours; RYZE itself (ADMIN_2) can take
// a waiting request at any time, or allocate it to a trainer by hand.

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

// offerRequest offers a waiting request to every available trainer who was
// not offered it yet and notifies each of them.
func (s *service) offerRequest(ctx context.Context, requestID string) error {
	if s.offers == nil {
		return nil
	}
	created, err := s.offers.OfferToAvailableTrainers(ctx, requestID, s.clock().Add(models.PremiumLevel2OfferWindow))
	if err != nil || len(created) == 0 || s.notifier == nil {
		return err
	}
	request, err := s.requests.FindRequestByID(ctx, requestID)
	if err != nil {
		return nil
	}
	users, ok := s.offers.(TrainerUsers)
	if !ok {
		return nil
	}
	for _, offer := range created {
		trainerUserID, err := users.FindTrainerUserID(ctx, offer.TrainerID)
		if err != nil || trainerUserID == "" {
			continue
		}
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
		return nil, offerError(err)
	}
	s.notifyAssigned(ctx, request.ID, trainerID)
	return request, nil
}

func (s *service) DeclineOffer(ctx context.Context, trainerID, trainerUserID, offerID string) error {
	if s.offers == nil {
		return ErrOfferNotFound
	}
	// The other trainers keep their offers; nobody new is asked.
	if _, err := s.offers.DeclineOffer(ctx, offerID, trainerID, trainerUserID, s.clock()); err != nil {
		return offerError(err)
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
		if err := s.offerRequest(ctx, requestID); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
