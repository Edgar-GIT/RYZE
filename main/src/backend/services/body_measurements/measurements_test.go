package body_measurements

import (
	"context"
	"errors"
	"testing"
	"time"

	"ryze/backend/repositories"
)

type fakeRepo struct {
	saved   []repositories.BodyMeasurementRow
	deleted bool
}

func (f *fakeRepo) List(context.Context, string, int) ([]repositories.BodyMeasurementRow, error) {
	return f.saved, nil
}

func (f *fakeRepo) Save(_ context.Context, _ string, kind string, value float64, on time.Time) (*repositories.BodyMeasurementRow, error) {
	row := repositories.BodyMeasurementRow{ID: "id", Kind: kind, Value: value, MeasuredOn: on}
	f.saved = append(f.saved, row)
	return &row, nil
}

func (f *fakeRepo) Delete(context.Context, string, string) (bool, error) { return f.deleted, nil }

func TestSaveValidatesKindValueAndDate(t *testing.T) {
	repo := &fakeRepo{}
	service := NewService(repo)
	service.now = func() time.Time { return time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC) }
	ctx := context.Background()

	cases := []struct {
		kind  string
		value float64
		date  string
		want  error
	}{
		{"height", 180, "2026-11-20", ErrInvalidKind},
		{"weight", 5, "2026-11-20", ErrInvalidValue},
		{"body_fat", 80, "2026-11-20", ErrInvalidValue},
		{"weight", 80, "20-11-2026", ErrInvalidDate},
		{"weight", 80, "2026-11-23", ErrInvalidDate},
	}
	for _, tc := range cases {
		if _, err := service.Save(ctx, "user", tc.kind, tc.value, tc.date); !errors.Is(err, tc.want) {
			t.Errorf("%s %v %s: got %v, want %v", tc.kind, tc.value, tc.date, err, tc.want)
		}
	}
	entry, err := service.Save(ctx, "user", "waist", 84.456, "2026-11-21")
	if err != nil || entry.Value != 84.46 || entry.MeasuredOn != "2026-11-21" {
		t.Fatalf("save = %+v, %v", entry, err)
	}
}

func TestDeleteReportsMissingMeasurements(t *testing.T) {
	service := NewService(&fakeRepo{deleted: false})
	if err := service.Delete(context.Background(), "user", "not-a-uuid"); !errors.Is(err, ErrMeasurementMissing) {
		t.Fatalf("invalid id: %v", err)
	}
	if err := service.Delete(context.Background(), "user", "8f14e45f-ceea-4e7a-9d2b-1b7f6c3a2e10"); !errors.Is(err, ErrMeasurementMissing) {
		t.Fatalf("someone else's id: %v", err)
	}
}
