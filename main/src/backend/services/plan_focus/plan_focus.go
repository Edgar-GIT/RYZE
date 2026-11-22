// Package plan_focus keeps the plan a member chose to focus on (owner
// request, 2026-11-22). The choice only counts while the member can still open
// the plan: access is checked on every read and write, never trusted from the
// client.
package plan_focus

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ryze/backend/repositories"
	"ryze/backend/services/program_access"
)

var ErrProgramNotAccessible = errors.New("plan focus program not accessible")

// GrantResolver says whether the member can open a program (the program
// access service).
type GrantResolver interface {
	ResolveGrant(ctx context.Context, userID, programID string) (program_access.GrantSource, error)
}

// Focus is the member's chosen plan.
type Focus struct {
	ProgramID string    `json:"program_id"`
	ChosenAt  time.Time `json:"chosen_at"`
}

type Service struct {
	repo   repositories.PlanFocusRepository
	grants GrantResolver
}

func NewService(repo repositories.PlanFocusRepository, grants GrantResolver) *Service {
	return &Service{repo: repo, grants: grants}
}

// Get returns the chosen plan, or nil when none is chosen or it is no longer
// accessible.
func (s *Service) Get(ctx context.Context, userID string) (*Focus, error) {
	row, err := s.repo.Get(ctx, userID)
	if err != nil || row == nil {
		return nil, err
	}
	if _, err := s.grants.ResolveGrant(ctx, userID, row.ProgramID); err != nil {
		return nil, nil
	}
	return &Focus{ProgramID: row.ProgramID, ChosenAt: row.ChosenAt.UTC()}, nil
}

// Set chooses a plan the member can open.
func (s *Service) Set(ctx context.Context, userID, programID string) (*Focus, error) {
	if _, err := s.grants.ResolveGrant(ctx, userID, programID); err != nil {
		return nil, ErrProgramNotAccessible
	}
	row, err := s.repo.Set(ctx, userID, programID)
	if err != nil {
		return nil, fmt.Errorf("plan focus: %w", err)
	}
	return &Focus{ProgramID: row.ProgramID, ChosenAt: row.ChosenAt.UTC()}, nil
}
