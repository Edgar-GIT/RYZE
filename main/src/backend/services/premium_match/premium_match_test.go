package premium_match_test

import (
	"context"
	"errors"
	"testing"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/premium_match"
)

type fakeRepo struct {
	total       int64
	candidates  []models.Program
	created     []models.PremiumLevel1Match
	userSeen    string
	sexSeen     string
	profileSeen repositories.MatchProfile
}

func (f *fakeRepo) CountPublishedPackages(context.Context, repositories.MatchProfile) (int64, error) {
	return f.total, nil
}
func (f *fakeRepo) ListMatchCandidates(_ context.Context, userID string, profile repositories.MatchProfile) ([]models.Program, error) {
	f.userSeen = userID
	f.sexSeen = profile.Sex
	f.profileSeen = profile
	return f.candidates, nil
}
func (f *fakeRepo) CreateMatch(_ context.Context, match *models.PremiumLevel1Match) error {
	f.created = append(f.created, *match)
	return nil
}

var valid = premium_match.Input{Objective: models.ProgramObjectiveGainMuscle, Sex: models.SexFemale, WeightKg: 78.24, HeightCm: 181, Age: 29}

func TestMatchPicksAmongCandidatesAndRecords(t *testing.T) {
	repo := &fakeRepo{total: 3, candidates: []models.Program{{ID: "a"}, {ID: "b"}, {ID: "c"}}}
	svc := premium_match.NewServiceWithPicker(repo, func(n int) int { return n - 1 })
	result, err := svc.Match(context.Background(), "user-1", valid)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if result.Program.ID != "c" || repo.userSeen != "user-1" || repo.sexSeen != models.SexFemale {
		t.Fatalf("unexpected pick %q for %q (%q)", result.Program.ID, repo.userSeen, repo.sexSeen)
	}
	if len(repo.created) != 1 || repo.created[0].WeightKg != 78.2 || repo.created[0].ProgramID != "c" ||
		repo.created[0].Sex == nil || *repo.created[0].Sex != models.SexFemale {
		t.Fatalf("match not recorded as expected: %+v", repo.created)
	}
}

func TestMatchExplainsWhyNothingIsGiven(t *testing.T) {
	none := &fakeRepo{total: 0}
	if _, err := premium_match.NewService(none).Match(context.Background(), "u", valid); !errors.Is(err, premium_match.ErrNoPackages) {
		t.Fatalf("got %v, want ErrNoPackages", err)
	}
	owned := &fakeRepo{total: 2}
	if _, err := premium_match.NewService(owned).Match(context.Background(), "u", valid); !errors.Is(err, premium_match.ErrAllOwned) {
		t.Fatalf("got %v, want ErrAllOwned", err)
	}
}

func TestMatchValidatesInput(t *testing.T) {
	cases := []premium_match.Input{
		{Objective: "dance", Sex: models.SexMale, WeightKg: 70, HeightCm: 170, Age: 30},
		{Objective: models.ProgramObjectiveLoseFat, Sex: models.SexMale, WeightKg: 10, HeightCm: 170, Age: 30},
		{Objective: models.ProgramObjectiveLoseFat, Sex: models.SexMale, WeightKg: 70, HeightCm: 300, Age: 30},
		{Objective: models.ProgramObjectiveLoseFat, Sex: models.SexMale, WeightKg: 70, HeightCm: 170, Age: 9},
		{Objective: models.ProgramObjectiveLoseFat, WeightKg: 70, HeightCm: 170, Age: 30},
		{Objective: models.ProgramObjectiveLoseFat, Sex: "other", WeightKg: 70, HeightCm: 170, Age: 30},
	}
	for _, input := range cases {
		if _, err := premium_match.NewService(&fakeRepo{}).Match(context.Background(), "u", input); !errors.Is(err, premium_match.ErrInvalidInput) {
			t.Fatalf("%+v: got %v, want ErrInvalidInput", input, err)
		}
	}
}
