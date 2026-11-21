package repositories_test

import (
	"context"
	"testing"

	"ryze/backend/repositories"
)

// The outline query runs against the real schema (table and column names) and
// returns nothing for a programme without structure.
func TestProgramOutlineQueryRuns(t *testing.T) {
	db := openInvoiceDB(t)
	days, err := repositories.NewProgramOutlineRepository(db).Outline(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("outline query: %v", err)
	}
	if len(days) != 0 {
		t.Fatalf("expected no days, got %d", len(days))
	}
}
