// Package health_consent manages the explicit consent for health data
// (GDPR art. 9(2)(a); owner request, 2026-11-24): body measurements and the
// Premium Level 2 questionnaire are sent only with it, the server records
// when, and withdrawing it erases that data.
package health_consent

import (
	"context"
	"errors"
	"time"

	"ryze/backend/repositories"
)

// ErrRequired: the request carries health data without the consent.
var ErrRequired = errors.New("health data consent required")

type Service struct {
	repo repositories.HealthConsentRepository
	now  func() time.Time
}

func New(repo repositories.HealthConsentRepository) *Service {
	return &Service{repo: repo, now: func() time.Time { return time.Now().UTC() }}
}

// ConsentedAt returns when the person consented (nil: not now).
func (s *Service) ConsentedAt(ctx context.Context, userID string) (*time.Time, error) {
	return s.repo.ConsentedAt(ctx, userID)
}

// Give records the consent and returns its time.
func (s *Service) Give(ctx context.Context, userID string) (*time.Time, error) {
	if err := s.repo.Give(ctx, userID, s.now()); err != nil {
		return nil, err
	}
	return s.repo.ConsentedAt(ctx, userID)
}

// Require lets health data through when the person consented before or
// consents with this request (given), recording it.
func (s *Service) Require(ctx context.Context, userID string, given bool) error {
	if given {
		return s.repo.Give(ctx, userID, s.now())
	}
	at, err := s.repo.ConsentedAt(ctx, userID)
	if err != nil {
		return err
	}
	if at == nil {
		return ErrRequired
	}
	return nil
}

// StampIntake records the consent on a questionnaire just sent.
func (s *Service) StampIntake(ctx context.Context, intakeID string) error {
	return s.repo.StampIntake(ctx, intakeID, s.now())
}

// Withdraw removes the consent and erases the health data.
func (s *Service) Withdraw(ctx context.Context, userID string) error {
	return s.repo.Withdraw(ctx, userID)
}
