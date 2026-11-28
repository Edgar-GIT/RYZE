// Package training_reminders nudges members who stopped training (owner
// request, 2026-11-24): a member with a plan who has not trained for
// IdleAfter gets a reminder in the app and by e-mail, at most once every
// MinGap, during the day in Portugal. They can turn it off in the account.
package training_reminders

import (
	"context"
	"fmt"
	"time"

	"ryze/backend/email"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/notifications"
)

const (
	// IdleAfter is how long without training before a reminder.
	IdleAfter = 4 * 24 * time.Hour
	// MinGap is the least time between two reminders to the same member.
	MinGap = 7 * 24 * time.Hour
	batch  = 200
)

type Service struct {
	repo     repositories.TrainingReminderRepository
	notifier notifications.Notifier
	now      func() time.Time
	zone     *time.Location
}

func New(repo repositories.TrainingReminderRepository, notifier notifications.Notifier) *Service {
	zone, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		zone = time.UTC
	}
	return &Service{repo: repo, notifier: notifier, now: func() time.Time { return time.Now().UTC() }, zone: zone}
}

// Sweep reminds the members due, between 10:00 and 19:59 in Portugal, and
// returns how many.
func (s *Service) Sweep(ctx context.Context) (int, error) {
	now := s.now()
	if hour := now.In(s.zone).Hour(); hour < 10 || hour >= 20 {
		return 0, nil
	}
	due, err := s.repo.Due(ctx, now.Add(-IdleAfter), now.Add(-MinGap), batch)
	if err != nil {
		return 0, err
	}
	sent := 0
	for _, member := range due {
		if s.notifier != nil {
			// The link opens the member's home, not straight into a workout
			// (owner request, 2026-11-28).
			s.notifier.Notify(ctx, notifications.Event{
				UserID:        member.UserID,
				Kind:          models.NotificationTrainingReminder,
				Link:          "/profile",
				DedupeKey:     fmt.Sprintf("training_reminder:%s:%s", member.UserID, now.In(s.zone).Format("2006-01-02")),
				EmailTemplate: email.TemplateTrainingReminder,
			})
		}
		if err := s.repo.MarkReminded(ctx, member.UserID, now); err != nil {
			return sent, err
		}
		sent++
	}
	return sent, nil
}

func (s *Service) Enabled(ctx context.Context, userID string) (bool, error) {
	return s.repo.Enabled(ctx, userID)
}

func (s *Service) SetEnabled(ctx context.Context, userID string, enabled bool) error {
	return s.repo.SetEnabled(ctx, userID, enabled)
}
