package coach_showcase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"ryze/backend/repositories"
)

type fakeShowcaseRepo struct {
	owner     string
	missing   bool
	views     int
	featured  []repositories.CoachProgramRow
	coach     *repositories.PublicCoachRow
	limitSeen int
}

func (f *fakeShowcaseRepo) FindProgramViewTarget(context.Context, string) (*repositories.ProgramViewTarget, error) {
	if f.missing {
		return nil, repositories.ErrProgramNotFound
	}
	return &repositories.ProgramViewTarget{OwnerUserID: f.owner}, nil
}

func (f *fakeShowcaseRepo) RecordView(context.Context, string, time.Time) error {
	f.views++
	return nil
}

func (f *fakeShowcaseRepo) ListFeaturedCoachPrograms(_ context.Context, limit int) ([]repositories.CoachProgramRow, error) {
	f.limitSeen = limit
	return f.featured, nil
}

func (f *fakeShowcaseRepo) FindPublicCoach(context.Context, string) (*repositories.PublicCoachRow, error) {
	if f.coach == nil {
		return nil, repositories.ErrCoachNotFound
	}
	return f.coach, nil
}

func (f *fakeShowcaseRepo) ListCoachPrograms(context.Context, string, int) ([]repositories.CoachProgramRow, error) {
	return f.featured, nil
}

type onceDeduper struct{ seen map[string]bool }

func (d *onceDeduper) Allow(key string) (bool, time.Duration) {
	if d.seen[key] {
		return false, time.Minute
	}
	d.seen[key] = true
	return true, 0
}

func TestRecordViewCountsOncePerViewerAndNeverTheOwner(t *testing.T) {
	repo := &fakeShowcaseRepo{owner: "owner-user"}
	svc := New(repo, &onceDeduper{seen: map[string]bool{}})
	ctx := context.Background()
	programID := uuid.NewString()

	counted, err := svc.RecordView(ctx, programID, Viewer{Key: "user:client", UserID: "client"})
	if err != nil || !counted {
		t.Fatalf("first view = %v, %v; want counted", counted, err)
	}
	if counted, _ := svc.RecordView(ctx, programID, Viewer{Key: "user:client", UserID: "client"}); counted {
		t.Fatal("a repeated view inside the window must not count")
	}
	if counted, _ := svc.RecordView(ctx, programID, Viewer{Key: "user:owner-user", UserID: "owner-user"}); counted {
		t.Fatal("the owning coach's own visit must not count")
	}
	if counted, _ := svc.RecordView(ctx, programID, Viewer{Key: "ip:abc"}); !counted {
		t.Fatal("an anonymous visitor's first view must count")
	}
	if repo.views != 2 {
		t.Fatalf("stored %d views, want 2", repo.views)
	}
}

func TestRecordViewRejectsBadAndUnknownPrograms(t *testing.T) {
	svc := New(&fakeShowcaseRepo{missing: true}, nil)
	if _, err := svc.RecordView(context.Background(), "not-a-uuid", Viewer{}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("malformed id: %v", err)
	}
	if _, err := svc.RecordView(context.Background(), uuid.NewString(), Viewer{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown program: %v", err)
	}
}

func TestFeaturedProgramsLimitsAndNames(t *testing.T) {
	repo := &fakeShowcaseRepo{featured: []repositories.CoachProgramRow{{ID: "p", ViewCount: 3, CoachID: "t", CoachFirstName: " Ana ", CoachLastName: "Silva"}}}
	svc := New(repo, nil)
	programs, err := svc.FeaturedPrograms(context.Background(), 0)
	if err != nil || len(programs) != 1 || repo.limitSeen != DefaultFeaturedLimit {
		t.Fatalf("FeaturedPrograms = %+v, %v (limit %d)", programs, err, repo.limitSeen)
	}
	if programs[0].Coach.Name != "Ana Silva" || programs[0].Views != 3 {
		t.Fatalf("unexpected program: %+v", programs[0])
	}
	if _, err := svc.FeaturedPrograms(context.Background(), MaxFeaturedLimit+1); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("over-limit request: %v", err)
	}
}

func TestProfileSumsViewsAndHidesNonPublicCoaches(t *testing.T) {
	repo := &fakeShowcaseRepo{
		coach:    &repositories.PublicCoachRow{ID: "t", FirstName: "Rui", LastName: "Costa"},
		featured: []repositories.CoachProgramRow{{ID: "a", ViewCount: 2}, {ID: "b", ViewCount: 5}},
	}
	profile, err := New(repo, nil).GetProfile(context.Background(), uuid.NewString())
	if err != nil || profile.TotalViews != 7 || profile.Coach.Name != "Rui Costa" {
		t.Fatalf("GetProfile = %+v, %v", profile, err)
	}
	if _, err := New(&fakeShowcaseRepo{}, nil).GetProfile(context.Background(), uuid.NewString()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a coach without published programs must be hidden: %v", err)
	}
}
