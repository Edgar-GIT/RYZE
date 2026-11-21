package repositories_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

// The recent-sets query runs against the real schema and finds nothing for a
// member without workouts.
func TestMuscleLoadQueryRuns(t *testing.T) {
	db := openInvoiceDB(t)
	rows, err := repositories.NewMuscleLoadRepository(db).RecentSets(context.Background(), "00000000-0000-0000-0000-000000000000", time.Now().Add(-7*24*time.Hour))
	if err != nil {
		t.Fatalf("recent sets: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected no sets, got %d", len(rows))
	}
}

// Logging the same kind on the same day replaces the value, and a member
// cannot delete another member's measurement.
func TestBodyMeasurementsSaveReplaceAndDelete(t *testing.T) {
	db := openInvoiceDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	users := repositories.NewUserRepository(tx)
	owner := &models.User{Email: "measure" + suffix + "@example.com", PasswordHash: "hash", FirstName: "A", LastName: "B"}
	other := &models.User{Email: "measure" + suffix + "@other.example", PasswordHash: "hash", FirstName: "C", LastName: "D"}
	for _, user := range []*models.User{owner, other} {
		if err := users.Create(ctx, user); err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	repo := repositories.NewBodyMeasurementRepository(tx)
	day := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	first, err := repo.Save(ctx, owner.ID, "weight", 80.4, day)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	second, err := repo.Save(ctx, owner.ID, "weight", 79.9, day)
	if err != nil {
		t.Fatalf("save again: %v", err)
	}
	if second.ID != first.ID || second.Value != 79.9 || second.MeasuredOn.Format("2006-01-02") != "2026-11-20" {
		t.Fatalf("expected the day's value replaced, got %+v (first %+v)", second, first)
	}
	if _, err := repo.Save(ctx, owner.ID, "waist", 84, day.AddDate(0, 0, -7)); err != nil {
		t.Fatalf("save waist: %v", err)
	}
	rows, err := repo.List(ctx, owner.ID, 10)
	if err != nil || len(rows) != 2 || rows[0].Kind != "waist" {
		t.Fatalf("list oldest first: %+v, %v", rows, err)
	}
	if deleted, err := repo.Delete(ctx, other.ID, first.ID); err != nil || deleted {
		t.Fatalf("another member deleted the measurement: %v, %v", deleted, err)
	}
	if deleted, err := repo.Delete(ctx, owner.ID, first.ID); err != nil || !deleted {
		t.Fatalf("owner delete: %v, %v", deleted, err)
	}
}
