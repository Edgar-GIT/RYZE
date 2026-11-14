package plan_preferences_test

import (
	"context"
	"errors"
	"testing"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/plan_preferences"
)

type fakeRepo struct{ stored repositories.PlanPreferences }

func (f *fakeRepo) Get(context.Context, string) (repositories.PlanPreferences, error) {
	return f.stored, nil
}
func (f *fakeRepo) Save(_ context.Context, _ string, preferences repositories.PlanPreferences) error {
	f.stored = preferences
	return nil
}

func ptr[T any](value T) *T { return &value }

func TestPlanPreferences(t *testing.T) {
	repo := &fakeRepo{}
	svc := plan_preferences.NewService(repo)
	ctx := context.Background()

	if _, err := svc.Save(ctx, "u", plan_preferences.Input{}); !errors.Is(err, plan_preferences.ErrInvalidInput) {
		t.Fatalf("an empty update must be refused: %v", err)
	}
	if _, err := svc.Save(ctx, "u", plan_preferences.Input{Sex: ptr("other")}); !errors.Is(err, plan_preferences.ErrInvalidInput) {
		t.Fatalf("an unknown sex must be refused: %v", err)
	}
	if _, err := svc.Save(ctx, "u", plan_preferences.Input{FilterBySex: ptr(true)}); !errors.Is(err, plan_preferences.ErrInvalidInput) {
		t.Fatalf("the filter needs a sex: %v", err)
	}
	saved, err := svc.Save(ctx, "u", plan_preferences.Input{Sex: ptr(models.SexMale), FilterBySex: ptr(true)})
	if err != nil || *saved.Sex != models.SexMale || !*saved.FilterBySex {
		t.Fatalf("Save = %+v, %v", saved, err)
	}
	// A partial update keeps the stored sex.
	saved, err = svc.Save(ctx, "u", plan_preferences.Input{FilterBySex: ptr(false)})
	if err != nil || *saved.Sex != models.SexMale || *saved.FilterBySex {
		t.Fatalf("partial Save = %+v, %v", saved, err)
	}
}
