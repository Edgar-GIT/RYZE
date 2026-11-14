package repositories_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

// TestCommissionRuleRepositoryUpsert covers the trainer commission override on
// PostgreSQL: active_commission_rule is a generated column, so GORM must never
// write it (it did, and every override failed with SQLSTATE 428C9).
func TestCommissionRuleRepositoryUpsert(t *testing.T) {
	config.LoadEnvFile()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	tx := db.Begin()
	defer tx.Rollback()

	ctx := context.Background()
	user := &models.User{
		Email:        fmt.Sprintf("commission-%d@ryze.local", time.Now().UnixNano()),
		PasswordHash: "prepared-hash-outside-repository-scope",
		FirstName:    "Commission",
		LastName:     "Coach",
	}
	if err := repositories.NewUserRepository(tx).Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	trainer := &models.Trainer{UserID: user.ID}
	if err := repositories.NewTrainerRepository(tx).Create(ctx, trainer); err != nil {
		t.Fatalf("create trainer: %v", err)
	}

	rules := repositories.NewCommissionRuleRepository(tx)
	if err := rules.UpsertOverride(ctx, &models.CommissionRule{TrainerID: trainer.ID, CommissionBPS: 2500}); err != nil {
		t.Fatalf("first override: %v", err)
	}
	if err := rules.UpsertOverride(ctx, &models.CommissionRule{TrainerID: trainer.ID, CommissionBPS: 3500}); err != nil {
		t.Fatalf("replacing override: %v", err)
	}
	active, err := rules.FindActiveOverride(ctx, trainer.ID)
	if err != nil {
		t.Fatalf("find override: %v", err)
	}
	if active.CommissionBPS != 3500 {
		t.Fatalf("active commission = %d, want 3500", active.CommissionBPS)
	}
	if err := rules.SoftDelete(ctx, trainer.ID); err != nil {
		t.Fatalf("remove override: %v", err)
	}
}
