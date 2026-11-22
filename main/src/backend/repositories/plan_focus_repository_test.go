package repositories_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

// The chosen plan is absent until set; setting again replaces it.
func TestPlanFocusSetAndGet(t *testing.T) {
	db := openInvoiceDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	ctx := context.Background()
	user := &models.User{Email: fmt.Sprintf("focus%d@example.com", time.Now().UnixNano()), PasswordHash: "hash", FirstName: "A", LastName: "B"}
	if err := repositories.NewUserRepository(tx).Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	plans := []*models.Program{
		{Name: "Focus A", Type: models.ProgramTypeFree, ProductType: models.ProgramProductTypeGeneric, Status: models.ProgramStatusPublished, Currency: "EUR"},
		{Name: "Focus B", Type: models.ProgramTypeFree, ProductType: models.ProgramProductTypeGeneric, Status: models.ProgramStatusPublished, Currency: "EUR"},
	}
	for _, plan := range plans {
		if err := tx.Omit("TrainerID").Create(plan).Error; err != nil {
			t.Fatalf("create program: %v", err)
		}
	}
	repo := repositories.NewPlanFocusRepository(tx)
	if row, err := repo.Get(ctx, user.ID); err != nil || row != nil {
		t.Fatalf("expected no focus, got %+v %v", row, err)
	}
	for _, plan := range plans {
		if _, err := repo.Set(ctx, user.ID, plan.ID); err != nil {
			t.Fatalf("set: %v", err)
		}
	}
	if row, err := repo.Get(ctx, user.ID); err != nil || row == nil || row.ProgramID != plans[1].ID {
		t.Fatalf("expected the latest choice, got %+v %v", row, err)
	}
}
