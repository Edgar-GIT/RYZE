package repositories_test

import (
	"context"
	"testing"

	"ryze/backend/repositories"
)

// The ranking query runs on PostgreSQL; a member with no workouts has none.
func TestRankingListForUserWithoutWorkouts(t *testing.T) {
	db := openInvoiceDB(t)
	rows, err := repositories.NewRankingRepository(db).ListForUser(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err != nil || len(rows) != 0 {
		t.Fatalf("expected no rankings, got %+v %v", rows, err)
	}
}
