package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ryze/backend/models"
)

var (
	// ErrPremiumLevel2OfferNotFound: no open offer with this id for the trainer.
	ErrPremiumLevel2OfferNotFound = errors.New("premium level 2 offer not found")
	// ErrPremiumLevel2OfferExpired: the 24-hour window closed before the answer.
	ErrPremiumLevel2OfferExpired = errors.New("premium level 2 offer expired")
)

// PremiumLevel2OfferView is an open offer as the trainer sees it before
// accepting: the service and the dates, never the intake answers.
type PremiumLevel2OfferView struct {
	ID               string
	RequestID        string
	OfferName        string
	RequestCreatedAt time.Time
	OfferedAt        time.Time
	ExpiresAt        time.Time
}

// PremiumLevel2OfferRepository allocates Premium Level 2 requests by offer
// (owner decision, 2026-11-21).
type PremiumLevel2OfferRepository interface {
	// OfferToAvailableTrainers offers a waiting request to every available
	// trainer who was not offered it yet (owner decision, 2026-11-24: all at
	// once, the first to accept takes it) and returns the new offers.
	OfferToAvailableTrainers(ctx context.Context, requestID string, expiresAt time.Time) ([]models.PremiumLevel2Offer, error)
	ListOpenOffers(ctx context.Context, trainerID string) ([]PremiumLevel2OfferView, error)
	// AcceptOffer allocates the request to the trainer in one transaction.
	AcceptOffer(ctx context.Context, offerID, trainerID, actorUserID string, now time.Time) (*models.PremiumLevel2Request, error)
	// DeclineOffer closes the offer and returns its request.
	DeclineOffer(ctx context.Context, offerID, trainerID, actorUserID string, now time.Time) (string, error)
	// ExpireOffers closes the offers past their deadline and returns their
	// requests.
	ExpireOffers(ctx context.Context, now time.Time) ([]string, error)
	// RequestsAwaitingOffer lists the requests that still wait for someone
	// to take them.
	RequestsAwaitingOffer(ctx context.Context, limit int) ([]string, error)
	TrainerAvailability(ctx context.Context, trainerID string) (bool, error)
	SetTrainerAvailability(ctx context.Context, trainerID string, accepts bool) error
}

func NewPremiumLevel2OfferRepository(db *gorm.DB) PremiumLevel2OfferRepository {
	return &premiumLevel2Repository{db: db}
}

func (r *premiumLevel2Repository) OfferToAvailableTrainers(ctx context.Context, requestID string, expiresAt time.Time) ([]models.PremiumLevel2Offer, error) {
	var created []models.PremiumLevel2Offer
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var request models.PremiumLevel2Request
		if err := lockPremiumLevel2Request(tx, requestID, &request); err != nil {
			return err
		}
		if request.Status != models.PremiumLevel2StatusAwaitingTrainerAssignment || request.IntakeVersion == 0 ||
			request.TrainerID != nil || request.HandledByRyze {
			return nil
		}
		// Real requests go to real trainers and Test Mode requests to Test
		// Mode trainers; a trainer is never offered their own request, nor the
		// same request twice.
		var trainerIDs []string
		if err := tx.Raw(`
			SELECT t.id
			FROM trainers t
			JOIN users u ON u.id = t.user_id AND u.deleted_at IS NULL
			JOIN users client ON client.id = ?
			WHERE t.deleted_at IS NULL AND t.accepts_level2_requests AND t.user_id <> client.id
			  AND ((u.test_persona IS NULL) = (client.test_persona IS NULL))
			  AND NOT EXISTS (SELECT 1 FROM premium_level_2_offers o WHERE o.request_id = ? AND o.trainer_id = t.id)
			ORDER BY t.created_at ASC`, request.UserID, requestID).Scan(&trainerIDs).Error; err != nil {
			return fmt.Errorf("failed to find available trainers: %w", err)
		}
		for _, trainerID := range trainerIDs {
			offer := models.PremiumLevel2Offer{
				ID:        uuid.NewString(),
				RequestID: requestID,
				TrainerID: trainerID,
				Status:    models.PremiumLevel2OfferOffered,
				ExpiresAt: expiresAt,
			}
			if err := tx.Create(&offer).Error; err != nil {
				return fmt.Errorf("failed to create premium level 2 offer: %w", err)
			}
			created = append(created, offer)
		}
		if len(created) == 0 {
			return nil
		}
		return createPremiumLevel2Event(tx, requestID, "system", "system", "trainer_offered", request.Status, request.Status)
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (r *premiumLevel2Repository) ListOpenOffers(ctx context.Context, trainerID string) ([]PremiumLevel2OfferView, error) {
	var rows []PremiumLevel2OfferView
	err := r.db.WithContext(ctx).Raw(`
		SELECT o.id, o.request_id, COALESCE(p.name, '') AS offer_name, r.created_at AS request_created_at,
		       o.offered_at, o.expires_at
		FROM premium_level_2_offers o
		JOIN premium_level_2_requests r ON r.id = o.request_id AND r.deleted_at IS NULL
		LEFT JOIN programs p ON p.id = r.program_id
		WHERE o.trainer_id = ? AND o.status = ? AND o.expires_at > CURRENT_TIMESTAMP
		ORDER BY o.expires_at ASC`, trainerID, models.PremiumLevel2OfferOffered).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list premium level 2 offers: %w", err)
	}
	return rows, nil
}

// lockOpenOffer locks the trainer's open offer; an offer past its deadline is
// closed as expired.
func lockOpenOffer(tx *gorm.DB, offerID, trainerID string, now time.Time) (*models.PremiumLevel2Offer, error) {
	var offer models.PremiumLevel2Offer
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND trainer_id = ? AND status = ?", offerID, trainerID, models.PremiumLevel2OfferOffered).
		First(&offer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPremiumLevel2OfferNotFound
		}
		return nil, fmt.Errorf("failed to load premium level 2 offer: %w", err)
	}
	if !offer.ExpiresAt.After(now) {
		return &offer, ErrPremiumLevel2OfferExpired
	}
	return &offer, nil
}

func closeOffer(tx *gorm.DB, offerID, status string, now time.Time) error {
	if err := tx.Model(&models.PremiumLevel2Offer{}).Where("id = ?", offerID).
		Updates(map[string]any{"status": status, "responded_at": now}).Error; err != nil {
		return fmt.Errorf("failed to close premium level 2 offer: %w", err)
	}
	return nil
}

func (r *premiumLevel2Repository) AcceptOffer(ctx context.Context, offerID, trainerID, actorUserID string, now time.Time) (*models.PremiumLevel2Request, error) {
	var request models.PremiumLevel2Request
	expired := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Every trainer accepting the same request queues on the request row
		// first, so the first one takes it and the others find it taken.
		var target struct{ RequestID string }
		if err := tx.Raw(`SELECT request_id FROM premium_level_2_offers WHERE id = ? AND trainer_id = ?`, offerID, trainerID).
			Scan(&target).Error; err != nil {
			return fmt.Errorf("failed to load premium level 2 offer: %w", err)
		}
		if target.RequestID == "" {
			return ErrPremiumLevel2OfferNotFound
		}
		var locked models.PremiumLevel2Request
		if err := lockPremiumLevel2Request(tx, target.RequestID, &locked); err != nil {
			return err
		}
		offer, err := lockOpenOffer(tx, offerID, trainerID, now)
		if errors.Is(err, ErrPremiumLevel2OfferExpired) {
			expired = true
			return closeOffer(tx, offer.ID, models.PremiumLevel2OfferExpired, now)
		}
		if err != nil {
			return err
		}
		if locked.TrainerID != nil || locked.HandledByRyze || locked.Status != models.PremiumLevel2StatusAwaitingTrainerAssignment {
			// Someone else took it first.
			if err := closeOffer(tx, offer.ID, models.PremiumLevel2OfferWithdrawn, now); err != nil {
				return err
			}
			return ErrPremiumLevel2OfferNotFound
		}
		if err := closeOffer(tx, offer.ID, models.PremiumLevel2OfferAccepted, now); err != nil {
			return err
		}
		if err := assignPremiumLevel2Trainer(tx, offer.RequestID, trainerID, "trainer", actorUserID, &request); err != nil {
			return err
		}
		return withdrawPremiumLevel2Offers(tx, offer.RequestID)
	})
	if err != nil {
		return nil, err
	}
	if expired {
		return nil, ErrPremiumLevel2OfferExpired
	}
	return &request, nil
}

func (r *premiumLevel2Repository) DeclineOffer(ctx context.Context, offerID, trainerID, actorUserID string, now time.Time) (string, error) {
	var requestID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		offer, err := lockOpenOffer(tx, offerID, trainerID, now)
		if err != nil && !errors.Is(err, ErrPremiumLevel2OfferExpired) {
			return err
		}
		requestID = offer.RequestID
		status := models.PremiumLevel2OfferDeclined
		if errors.Is(err, ErrPremiumLevel2OfferExpired) {
			status = models.PremiumLevel2OfferExpired
		}
		if err := closeOffer(tx, offer.ID, status, now); err != nil {
			return err
		}
		state := models.PremiumLevel2StatusAwaitingTrainerAssignment
		return createPremiumLevel2Event(tx, offer.RequestID, "trainer", actorUserID, "offer_"+status, state, state)
	})
	return requestID, err
}

func (r *premiumLevel2Repository) ExpireOffers(ctx context.Context, now time.Time) ([]string, error) {
	var requestIDs []string
	err := r.db.WithContext(ctx).Raw(`
		UPDATE premium_level_2_offers SET status = ?, responded_at = ?
		WHERE status = ? AND expires_at <= ?
		RETURNING request_id`, models.PremiumLevel2OfferExpired, now, models.PremiumLevel2OfferOffered, now).Scan(&requestIDs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to expire premium level 2 offers: %w", err)
	}
	return requestIDs, nil
}

func (r *premiumLevel2Repository) RequestsAwaitingOffer(ctx context.Context, limit int) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).Raw(`
		SELECT r.id FROM premium_level_2_requests r
		WHERE r.deleted_at IS NULL AND r.status = ? AND r.intake_version > 0 AND r.trainer_id IS NULL AND NOT r.handled_by_ryze
		ORDER BY r.created_at ASC
		LIMIT ?`, models.PremiumLevel2StatusAwaitingTrainerAssignment, limit).Scan(&ids).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list requests awaiting an offer: %w", err)
	}
	return ids, nil
}

func (r *premiumLevel2Repository) TrainerAvailability(ctx context.Context, trainerID string) (bool, error) {
	var values []bool
	err := r.db.WithContext(ctx).Raw(`SELECT accepts_level2_requests FROM trainers WHERE id = ? AND deleted_at IS NULL`, trainerID).
		Scan(&values).Error
	if err != nil {
		return false, fmt.Errorf("failed to read trainer availability: %w", err)
	}
	return len(values) > 0 && values[0], nil
}

func (r *premiumLevel2Repository) SetTrainerAvailability(ctx context.Context, trainerID string, accepts bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`UPDATE trainers SET accepts_level2_requests = ? WHERE id = ? AND deleted_at IS NULL`,
			accepts, trainerID).Error; err != nil {
			return fmt.Errorf("failed to set trainer availability: %w", err)
		}
		if accepts {
			return nil
		}
		// A trainer who stops taking requests loses their open offers.
		if err := tx.Exec(`UPDATE premium_level_2_offers SET status = ?, responded_at = CURRENT_TIMESTAMP
			WHERE trainer_id = ? AND status = ?`, models.PremiumLevel2OfferWithdrawn, trainerID, models.PremiumLevel2OfferOffered).Error; err != nil {
			return fmt.Errorf("failed to withdraw trainer offers: %w", err)
		}
		return nil
	})
}

// FindTrainerUserID returns the account of an active trainer.
func (r *premiumLevel2Repository) FindTrainerUserID(ctx context.Context, trainerID string) (string, error) {
	var ids []string
	if err := r.db.WithContext(ctx).Raw(`SELECT user_id FROM trainers WHERE id = ? AND deleted_at IS NULL`, trainerID).
		Scan(&ids).Error; err != nil {
		return "", fmt.Errorf("failed to find trainer account: %w", err)
	}
	if len(ids) == 0 {
		return "", nil
	}
	return ids[0], nil
}
