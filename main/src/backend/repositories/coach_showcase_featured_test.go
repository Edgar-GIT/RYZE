package repositories_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

// The featured programmes of the free marketplace show free plans only; a
// Premium Level 1 package never appears there, while the coach's own page
// lists both (owner, 2026-11-27).
func TestFeaturedCoachProgramsNeverShowLevel1(t *testing.T) {
	db := openInvoiceDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	ctx := context.Background()

	user := &models.User{Email: fmt.Sprintf("featured-%d@ryze.local", time.Now().UnixNano()), PasswordHash: "hash", FirstName: "F", LastName: "C"}
	if err := repositories.NewUserRepository(tx).Create(ctx, user); err != nil {
		t.Fatalf("user: %v", err)
	}
	trainer := &models.Trainer{UserID: user.ID}
	if err := repositories.NewTrainerRepository(tx).Create(ctx, trainer); err != nil {
		t.Fatalf("trainer: %v", err)
	}
	programs := repositories.NewProgramRepository(tx)
	free := &models.Program{TrainerID: trainer.ID, Name: "Free featured probe", Type: models.ProgramTypeFree, Status: models.ProgramStatusPublished, Currency: "EUR"}
	if err := programs.Create(ctx, free); err != nil {
		t.Fatalf("free: %v", err)
	}
	objective := models.ProgramObjectiveGainMuscle
	mode := models.ProgramNutritionModeCurated
	level1 := &models.Program{
		TrainerID: trainer.ID, Name: "Level 1 featured probe", Type: models.ProgramTypePremium, Status: models.ProgramStatusPublished,
		Currency: "EUR", PriceMinorUnits: 1499, ProductType: models.ProgramProductTypePremiumLevel1, Objective: &objective, NutritionMode: &mode,
	}
	if err := programs.Create(ctx, level1); err != nil {
		t.Fatalf("level 1: %v", err)
	}
	// Many views: it would be first if it were allowed.
	if err := tx.Exec("INSERT INTO program_view_stats (program_id, view_count, last_viewed_at) VALUES (?, 1000000, now())", level1.ID).Error; err != nil {
		t.Fatalf("views: %v", err)
	}

	showcase := repositories.NewCoachShowcaseRepository(tx)
	featured, err := showcase.ListFeaturedCoachPrograms(ctx, 500)
	if err != nil {
		t.Fatalf("featured: %v", err)
	}
	foundFree := false
	for _, row := range featured {
		if row.ID == level1.ID || row.ProductType != models.ProgramProductTypeGeneric || row.Type != models.ProgramTypeFree {
			t.Fatalf("the free marketplace featured a non-free programme: %s (%s, %s)", row.Name, row.ProductType, row.Type)
		}
		if row.ID == free.ID {
			foundFree = true
		}
	}
	if !foundFree {
		t.Fatalf("the free programme should be featured")
	}

	own, err := showcase.ListCoachPrograms(ctx, trainer.ID, 50)
	if err != nil {
		t.Fatalf("coach programmes: %v", err)
	}
	if len(own) != 2 {
		t.Fatalf("the coach page lists the free plan and the Level 1 package, got %d", len(own))
	}

	// Public counters (v2.133.0): plans on the market and real purchases.
	counter := showcase.(interface {
		CoachCounts(ctx context.Context, trainerID string) (int64, int64, error)
	})
	for i, test := range []bool{false, true} {
		buyer := &models.User{Email: fmt.Sprintf("featured-buyer-%d-%d@ryze.local", i, time.Now().UnixNano()), PasswordHash: "hash", FirstName: "B", LastName: "Y"}
		if err := repositories.NewUserRepository(tx).Create(ctx, buyer); err != nil {
			t.Fatalf("buyer: %v", err)
		}
		if err := tx.Create(&models.Purchase{UserID: buyer.ID, ProgramID: level1.ID, PriceMinorUnits: 1499, Currency: "EUR", Status: models.PurchaseStatusCompleted, Test: test}).Error; err != nil {
			t.Fatalf("purchase: %v", err)
		}
	}
	plans, subscriptions, err := counter.CoachCounts(ctx, trainer.ID)
	if err != nil || plans != 2 || subscriptions != 1 {
		t.Fatalf("counters: plans=%d subscriptions=%d (want 2 and 1, Test Mode excluded) %v", plans, subscriptions, err)
	}
}
