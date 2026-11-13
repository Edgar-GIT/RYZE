// Package coach_showcase serves the public coach showcase: the "Featured coach
// programs" ranking of the generic marketplace, public coach profiles and the
// program view counters that drive the ranking.
package coach_showcase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"ryze/backend/repositories"
)

var (
	// ErrInvalidInput indicates a malformed identifier or limit.
	ErrInvalidInput = errors.New("invalid coach showcase input")
	// ErrNotFound covers unknown, unpublished and non-public programs and
	// coaches alike.
	ErrNotFound = errors.New("coach showcase resource not found")
)

const (
	// DefaultFeaturedLimit is the number of featured programs shown.
	DefaultFeaturedLimit = 6
	// MaxFeaturedLimit caps one featured request.
	MaxFeaturedLimit = 12
	// coachProgramsLimit bounds one coach profile.
	coachProgramsLimit = 50
)

// Coach is a coach's public identity: the trainer id used in links and the
// name the trainer registered with. Contact data is never public.
type Coach struct {
	ID   string
	Name string
}

// Program is one published, trainer-owned generic program as shown in public.
type Program struct {
	ID               string
	Name             string
	Description      string
	Type             string
	Level            *string
	DurationWeeks    *int
	FrequencyPerWeek *int
	TrainingType     *string
	PriceMinorUnits  int64
	Currency         string
	ProductType      string
	CreatedAt        time.Time
	Views            int64
	Coach            Coach
}

// Profile is a public coach profile with the coach's published programs.
type Profile struct {
	Coach      Coach
	CoachSince time.Time
	TotalViews int64
	Programs   []Program
}

// Viewer identifies who opened a program page. Key is an opaque, stable
// identifier (a user id or an address digest) used only for deduplication;
// UserID is set when the viewer is signed in.
type Viewer struct {
	Key    string
	UserID string
}

// Deduper answers whether a key may count again. middleware.Limiter
// satisfies it with a one-per-window policy.
type Deduper interface {
	Allow(key string) (bool, time.Duration)
}

// Service is the coach showcase.
type Service interface {
	// RecordView counts one view of a published program. It reports whether
	// the view was counted: repeated views within the dedupe window and the
	// owner's own visits are not.
	RecordView(ctx context.Context, programID string, viewer Viewer) (bool, error)
	FeaturedPrograms(ctx context.Context, limit int) ([]Program, error)
	GetProfile(ctx context.Context, trainerID string) (*Profile, error)
	// CoachOf returns the public coach of a trainer, for program attribution.
	CoachOf(ctx context.Context, trainerID string) (*Coach, error)
}

type service struct {
	repo  repositories.CoachShowcaseRepository
	dedup Deduper
	now   func() time.Time
}

// New builds the showcase. dedup may be nil, in which case every view counts.
func New(repo repositories.CoachShowcaseRepository, dedup Deduper) Service {
	return &service{repo: repo, dedup: dedup, now: time.Now}
}

func (s *service) RecordView(ctx context.Context, programID string, viewer Viewer) (bool, error) {
	if err := validateID(programID); err != nil {
		return false, err
	}
	target, err := s.repo.FindProgramViewTarget(ctx, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrProgramNotFound) {
			return false, ErrNotFound
		}
		return false, err
	}
	if viewer.UserID != "" && viewer.UserID == target.OwnerUserID {
		return false, nil
	}
	if s.dedup != nil && viewer.Key != "" {
		if allowed, _ := s.dedup.Allow("program-view:" + viewer.Key + ":" + programID); !allowed {
			return false, nil
		}
	}
	if err := s.repo.RecordView(ctx, programID, s.now().UTC()); err != nil {
		return false, err
	}
	return true, nil
}

func (s *service) FeaturedPrograms(ctx context.Context, limit int) ([]Program, error) {
	if limit == 0 {
		limit = DefaultFeaturedLimit
	}
	if limit < 1 || limit > MaxFeaturedLimit {
		return nil, ErrInvalidInput
	}
	rows, err := s.repo.ListFeaturedCoachPrograms(ctx, limit)
	if err != nil {
		return nil, err
	}
	return toPrograms(rows), nil
}

func (s *service) GetProfile(ctx context.Context, trainerID string) (*Profile, error) {
	if err := validateID(trainerID); err != nil {
		return nil, err
	}
	coach, err := s.repo.FindPublicCoach(ctx, trainerID)
	if err != nil {
		if errors.Is(err, repositories.ErrCoachNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	rows, err := s.repo.ListCoachPrograms(ctx, trainerID, coachProgramsLimit)
	if err != nil {
		return nil, err
	}
	profile := &Profile{
		Coach:      Coach{ID: coach.ID, Name: displayName(coach.FirstName, coach.LastName)},
		CoachSince: coach.CreatedAt,
		Programs:   toPrograms(rows),
	}
	for _, program := range profile.Programs {
		profile.TotalViews += program.Views
	}
	return profile, nil
}

func (s *service) CoachOf(ctx context.Context, trainerID string) (*Coach, error) {
	if err := validateID(trainerID); err != nil {
		return nil, err
	}
	coach, err := s.repo.FindPublicCoach(ctx, trainerID)
	if err != nil {
		if errors.Is(err, repositories.ErrCoachNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &Coach{ID: coach.ID, Name: displayName(coach.FirstName, coach.LastName)}, nil
}

func toPrograms(rows []repositories.CoachProgramRow) []Program {
	programs := make([]Program, 0, len(rows))
	for _, row := range rows {
		programs = append(programs, Program{
			ID:               row.ID,
			Name:             row.Name,
			Description:      row.Description,
			Type:             row.Type,
			Level:            row.Level,
			DurationWeeks:    row.DurationWeeks,
			FrequencyPerWeek: row.FrequencyPerWeek,
			TrainingType:     row.TrainingType,
			PriceMinorUnits:  row.PriceMinorUnits,
			Currency:         row.Currency,
			ProductType:      row.ProductType,
			CreatedAt:        row.CreatedAt,
			Views:            row.ViewCount,
			Coach:            Coach{ID: row.CoachID, Name: displayName(row.CoachFirstName, row.CoachLastName)},
		})
	}
	return programs
}

func displayName(first, last string) string {
	return strings.TrimSpace(strings.TrimSpace(first) + " " + strings.TrimSpace(last))
}

func validateID(id string) error {
	if _, err := uuid.Parse(strings.TrimSpace(id)); err != nil {
		return fmt.Errorf("%w: malformed identifier", ErrInvalidInput)
	}
	return nil
}
