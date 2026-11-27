// Package account_anonymization erases the personal data of accounts deleted
// more than 30 days ago (GDPR art. 17; owner request, 2026-11-24). During the
// 30 days a re-registration with the same e-mail restores the account; after
// them the account keeps only what the law requires (sales documents) and
// anonymous training records.
package account_anonymization

import (
	"context"
	"log/slog"
	"time"

	"ryze/backend/repositories"
)

// GracePeriod is how long a deleted account can still be restored.
const GracePeriod = 30 * 24 * time.Hour

const batch = 100

// FileRemover deletes the stored pictures of an anonymised account.
type FileRemover interface {
	DeleteAvatar(key string) error
	DeleteSessionPhoto(key string) error
	DeleteFeedbackImage(key string) error
}

type Service struct {
	repo  repositories.AccountAnonymizationRepository
	files FileRemover
	now   func() time.Time
}

func New(repo repositories.AccountAnonymizationRepository, files FileRemover) *Service {
	return &Service{repo: repo, files: files, now: func() time.Time { return time.Now().UTC() }}
}

// Sweep anonymises the accounts whose grace period ended and returns how many.
func (s *Service) Sweep(ctx context.Context) (int, error) {
	now := s.now()
	due, err := s.repo.Due(ctx, now.Add(-GracePeriod), batch)
	if err != nil {
		return 0, err
	}
	done := 0
	for _, userID := range due {
		files, err := s.repo.Anonymize(ctx, userID, now)
		if err != nil {
			return done, err
		}
		done++
		s.removeFiles(files)
	}
	return done, nil
}

// removeFiles deletes the pictures after the transaction; a failure leaves
// an orphan file (no longer referenced) and is logged without the key.
func (s *Service) removeFiles(files *repositories.AnonymizedFiles) {
	if s.files == nil || files == nil {
		return
	}
	if files.Avatar != nil {
		if err := s.files.DeleteAvatar(*files.Avatar); err != nil {
			slog.Warn("anonymisation: avatar not deleted", slog.String("error", err.Error()))
		}
	}
	for _, key := range files.SessionPhotos {
		if err := s.files.DeleteSessionPhoto(key); err != nil {
			slog.Warn("anonymisation: session photo not deleted", slog.String("error", err.Error()))
		}
	}
	for _, key := range files.FeedbackImages {
		if err := s.files.DeleteFeedbackImage(key); err != nil {
			slog.Warn("anonymisation: feedback picture not deleted", slog.String("error", err.Error()))
		}
	}
}
