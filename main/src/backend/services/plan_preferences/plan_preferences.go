// Package plan_preferences stores the member's sex and whether they prefer to
// see the training plans made for it first (owner request, 2026-11-14: asked
// the first time the member opens the training plans).
package plan_preferences

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

var (
	// ErrInvalidInput indicates an empty update, an unknown sex, or a filter
	// request without a known sex.
	ErrInvalidInput = errors.New("invalid plan preferences")
	// ErrUserNotFound indicates the account no longer exists.
	ErrUserNotFound = errors.New("user not found")
)

// Input is a partial update: nil fields keep their stored value.
type Input struct {
	Sex         *string
	FilterBySex *bool
}

type Service interface {
	Save(ctx context.Context, userID string, input Input) (repositories.PlanPreferences, error)
}

type service struct {
	repo repositories.PlanPreferencesRepository
}

func NewService(repo repositories.PlanPreferencesRepository) Service {
	return &service{repo: repo}
}

func (s *service) Save(ctx context.Context, userID string, input Input) (repositories.PlanPreferences, error) {
	if strings.TrimSpace(userID) == "" {
		return repositories.PlanPreferences{}, ErrUserNotFound
	}
	if input.Sex == nil && input.FilterBySex == nil {
		return repositories.PlanPreferences{}, fmt.Errorf("%w: nothing to save", ErrInvalidInput)
	}
	current, err := s.repo.Get(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return repositories.PlanPreferences{}, ErrUserNotFound
		}
		return repositories.PlanPreferences{}, err
	}
	if input.Sex != nil {
		sex := strings.TrimSpace(*input.Sex)
		if !models.ValidSex(sex) {
			return repositories.PlanPreferences{}, fmt.Errorf("%w: sex must be female or male", ErrInvalidInput)
		}
		current.Sex = &sex
	}
	if input.FilterBySex != nil {
		filter := *input.FilterBySex
		if filter && current.Sex == nil {
			return repositories.PlanPreferences{}, fmt.Errorf("%w: the filter needs a sex", ErrInvalidInput)
		}
		current.FilterBySex = &filter
	}
	if err := s.repo.Save(ctx, userID, current); err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return repositories.PlanPreferences{}, ErrUserNotFound
		}
		return repositories.PlanPreferences{}, err
	}
	return current, nil
}
