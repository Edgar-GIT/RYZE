package plan_focus_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ryze/backend/repositories"
	"ryze/backend/services/plan_focus"
	"ryze/backend/services/program_access"
)

type memoryRepo struct{ row *repositories.PlanFocusRow }

func (m *memoryRepo) Get(context.Context, string) (*repositories.PlanFocusRow, error) {
	return m.row, nil
}
func (m *memoryRepo) Set(_ context.Context, _ string, programID string) (*repositories.PlanFocusRow, error) {
	m.row = &repositories.PlanFocusRow{ProgramID: programID, ChosenAt: time.Now()}
	return m.row, nil
}

type grants struct{ allowed map[string]bool }

func (g grants) ResolveGrant(_ context.Context, _ string, programID string) (program_access.GrantSource, error) {
	if g.allowed[programID] {
		return program_access.GrantSourceFreeActivation, nil
	}
	return "", program_access.ErrProgramNotAccessible
}

// Only a plan the member can open is chosen, and a choice that is no longer
// accessible is ignored.
func TestPlanFocusChecksAccess(t *testing.T) {
	repo := &memoryRepo{}
	access := grants{allowed: map[string]bool{"plan-a": true}}
	service := plan_focus.NewService(repo, access)
	ctx := context.Background()

	if _, err := service.Set(ctx, "user", "plan-b"); !errors.Is(err, plan_focus.ErrProgramNotAccessible) {
		t.Fatalf("expected ErrProgramNotAccessible, got %v", err)
	}
	if focus, err := service.Set(ctx, "user", "plan-a"); err != nil || focus.ProgramID != "plan-a" {
		t.Fatalf("set: %+v %v", focus, err)
	}
	access.allowed["plan-a"] = false
	if focus, err := service.Get(ctx, "user"); err != nil || focus != nil {
		t.Fatalf("a plan no longer accessible must be ignored: %+v %v", focus, err)
	}
}
