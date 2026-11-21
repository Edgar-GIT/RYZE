package repositories_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

// A trainer with no sales has no totals; the EXP adjustment of a new account
// is zero, then the sum of what was added (v2.130.0).
func TestTrainerEarningsAndXPAdjustments(t *testing.T) {
	db := openInvoiceDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	ctx := context.Background()

	totals, err := repositories.NewTrainerEarningsRepository(tx).Totals(ctx, "00000000-0000-0000-0000-000000000000")
	if err != nil || len(totals) != 0 {
		t.Fatalf("unknown trainer: %+v %v", totals, err)
	}

	user := &models.User{Email: fmt.Sprintf("xp%d@example.com", time.Now().UnixNano()), PasswordHash: "hash", FirstName: "A", LastName: "B"}
	if err := repositories.NewUserRepository(tx).Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	repo := repositories.NewAchievementsRepository(tx).(interface {
		XPAdjustment(ctx context.Context, userID string) (int, error)
		AddXPAdjustment(ctx context.Context, userID string, xp int, reason string) error
	})
	if got, err := repo.XPAdjustment(ctx, user.ID); err != nil || got != 0 {
		t.Fatalf("new account adjustment: %d %v", got, err)
	}
	for _, xp := range []int{40000, -1} {
		if err := repo.AddXPAdjustment(ctx, user.ID, xp, "test"); err != nil {
			t.Fatalf("add adjustment: %v", err)
		}
	}
	if got, err := repo.XPAdjustment(ctx, user.ID); err != nil || got != 39999 {
		t.Fatalf("adjustment sum: %d %v", got, err)
	}
}
