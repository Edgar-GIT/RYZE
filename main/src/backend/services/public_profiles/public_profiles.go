// Package public_profiles serves the public profile of an account (owner
// request, 2026-11-21): photo, nickname (else @username), member since and a
// few training numbers anyone may see, plus following. Profiles are public to
// everyone (owner decision); they never show the email, the registered name,
// body data or anything health-related.
package public_profiles

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ryze/backend/repositories"
	"ryze/backend/services/achievements"
	"ryze/backend/services/coach_showcase"
	"ryze/backend/services/profile"
)

var (
	ErrNotFound   = errors.New("public profile not found")
	ErrFollowSelf = errors.New("cannot follow yourself")
)

// listLimit bounds the followers and following lists.
const listLimit = 100

// Person is an account as profiles and lists show it.
type Person struct {
	ID          string
	Username    string
	DisplayName *string
	AvatarKey   *string
}

// Profile is one public profile as a viewer sees it.
type Profile struct {
	Person
	MemberSince time.Time
	Workouts    int64
	Followers   int64
	Following   int64
	LevelRank   int
	LevelKey    string
	Medals      int
	// CoachID is the public coach profile of a trainer ("" when none).
	CoachID string
	// CoachViews is a trainer's reach: the views of their published plans
	// (only trainers have it; nil for everyone else).
	CoachViews *int64
	// Viewer: whether the viewer is this account and follows it.
	ViewerIsSelf    bool
	ViewerFollowing bool
	// About is the optional biography, age, years of training, weight and
	// social networks.
	About *profile.PublicAbout
}

// coachReach reads a coach's total plan views (optional so test fakes of
// Coaches keep compiling).
type coachReach interface {
	GetProfile(ctx context.Context, trainerID string) (*coach_showcase.Profile, error)
}

// AboutReader reads the public "about" of an account.
type AboutReader interface {
	Public(ctx context.Context, userID string) (*profile.PublicAbout, error)
}

type Achievements interface {
	Summary(ctx context.Context, userID string) (*achievements.Summary, error)
}

type Coaches interface {
	CoachOf(ctx context.Context, trainerID string) (*coach_showcase.Coach, error)
}

type Service struct {
	repo         repositories.PublicProfileRepository
	achievements Achievements
	coaches      Coaches
	about        AboutReader
}

func NewService(repo repositories.PublicProfileRepository, achievements Achievements, coaches Coaches, about AboutReader) *Service {
	return &Service{repo: repo, achievements: achievements, coaches: coaches, about: about}
}

func normalize(username string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(username), "@"))
}

func (s *Service) find(ctx context.Context, username string) (*repositories.PublicUserRow, error) {
	row, err := s.repo.FindByUsername(ctx, normalize(username))
	if err != nil {
		if errors.Is(err, repositories.ErrPublicProfileNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return row, nil
}

func toPerson(row repositories.PublicUserRow) Person {
	return Person{ID: row.ID, Username: row.Username, DisplayName: row.DisplayName, AvatarKey: row.AvatarKey}
}

// Get returns the profile; viewerID is "" for a visitor.
func (s *Service) Get(ctx context.Context, username, viewerID string) (*Profile, error) {
	row, err := s.find(ctx, username)
	if err != nil {
		return nil, err
	}
	profile := &Profile{Person: toPerson(*row), MemberSince: row.CreatedAt}
	if profile.Followers, profile.Following, err = s.repo.FollowCounts(ctx, row.ID); err != nil {
		return nil, err
	}
	if profile.Workouts, err = s.repo.CompletedWorkouts(ctx, row.ID); err != nil {
		return nil, err
	}
	if s.achievements != nil {
		summary, err := s.achievements.Summary(ctx, row.ID)
		if err != nil {
			return nil, fmt.Errorf("public profile achievements: %w", err)
		}
		profile.LevelRank, profile.LevelKey, profile.Medals = summary.Level.Rank, summary.Level.Key, summary.MedalCount
	}
	if s.coaches != nil {
		trainerID, err := s.repo.TrainerIDOf(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		if trainerID != "" {
			if coach, err := s.coaches.CoachOf(ctx, trainerID); err == nil {
				profile.CoachID = coach.ID
				if reach, ok := s.coaches.(coachReach); ok {
					if full, err := reach.GetProfile(ctx, trainerID); err == nil {
						views := full.TotalViews
						profile.CoachViews = &views
					}
				}
			}
		}
	}
	if s.about != nil {
		if profile.About, err = s.about.Public(ctx, row.ID); err != nil {
			return nil, err
		}
	}
	if viewerID != "" {
		profile.ViewerIsSelf = viewerID == row.ID
		if !profile.ViewerIsSelf {
			if profile.ViewerFollowing, err = s.repo.IsFollowing(ctx, viewerID, row.ID); err != nil {
				return nil, err
			}
		}
	}
	return profile, nil
}

// Follow makes the viewer follow the account (idempotent).
func (s *Service) Follow(ctx context.Context, viewerID, username string) error {
	row, err := s.find(ctx, username)
	if err != nil {
		return err
	}
	if row.ID == viewerID {
		return ErrFollowSelf
	}
	return s.repo.Follow(ctx, viewerID, row.ID)
}

// Unfollow stops following (idempotent).
func (s *Service) Unfollow(ctx context.Context, viewerID, username string) error {
	row, err := s.find(ctx, username)
	if err != nil {
		return err
	}
	return s.repo.Unfollow(ctx, viewerID, row.ID)
}

// Followers lists who follows the account, newest first.
func (s *Service) Followers(ctx context.Context, username string) ([]Person, error) {
	row, err := s.find(ctx, username)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.Followers(ctx, row.ID, listLimit)
	return toPeople(rows), err
}

// Following lists whom the account follows, newest first.
func (s *Service) Following(ctx context.Context, username string) ([]Person, error) {
	row, err := s.find(ctx, username)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.Following(ctx, row.ID, listLimit)
	return toPeople(rows), err
}

func toPeople(rows []repositories.PublicUserRow) []Person {
	people := make([]Person, 0, len(rows))
	for _, row := range rows {
		people = append(people, toPerson(row))
	}
	return people
}
