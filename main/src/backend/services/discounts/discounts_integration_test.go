package discounts_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/discounts"
)

func TestDiscountCodesAtCheckout(t *testing.T) {
	config.LoadEnvFile()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	tx := db.Begin()
	defer tx.Rollback()
	ctx := context.Background()
	suffix := time.Now().UnixNano()

	user := &models.User{Email: fmt.Sprintf("discount-%d@ryze.local", suffix), PasswordHash: "hash", FirstName: "D", LastName: "C"}
	if err := repositories.NewUserRepository(tx).Create(ctx, user); err != nil {
		t.Fatalf("user: %v", err)
	}
	plan := &models.Program{Name: "Pacote", Type: models.ProgramTypePremium, ProductType: models.ProgramProductTypePremiumLevel1,
		Status: models.ProgramStatusPublished, PriceMinorUnits: 1499, Currency: "EUR"}
	if err := repositories.NewGenericProgramRepository(tx).CreateFull(ctx, plan); err != nil {
		t.Fatalf("plan: %v", err)
	}
	// A pending sale of a trainer-like split (30% commission).
	purchase := &models.Purchase{UserID: user.ID, ProgramID: plan.ID, PriceMinorUnits: 1499, Currency: "EUR", CommissionBPS: 3000,
		PlatformAmount: 449, TrainerAmount: 1050, Status: models.PurchaseStatusPending}
	if err := tx.Create(purchase).Error; err != nil {
		t.Fatalf("purchase: %v", err)
	}

	service := discounts.New(repositories.NewDiscountCodeRepository(tx))
	code := fmt.Sprintf("LAUNCH%d", suffix%100000)
	if _, err := service.Create(ctx, "admin-1", discounts.CreateInput{Code: code, PercentOff: 20, AppliesTo: "premium_level_1", DurationHours: 24}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := service.Create(ctx, "admin-1", discounts.CreateInput{Code: "TOOLONG", PercentOff: 20, DurationHours: 721}); !errors.Is(err, discounts.ErrInvalidInput) {
		t.Fatalf("more than 30 days must be refused: %v", err)
	}
	if _, err := service.Create(ctx, "admin-2", discounts.CreateInput{Code: code, PercentOff: 10, DurationHours: 5}); !errors.Is(err, discounts.ErrCodeTaken) {
		t.Fatalf("codes are unique: %v", err)
	}

	if err := service.ApplyToPurchase(ctx, user.ID, purchase.ID, " "+code[:3]+code[3:]+" "); err != nil {
		t.Fatalf("apply: %v", err)
	}
	var applied models.Purchase
	tx.First(&applied, "id = ?", purchase.ID)
	if applied.PriceMinorUnits != 1200 || applied.DiscountMinorUnits != 299 || applied.OriginalPriceMinorUnits == nil ||
		*applied.OriginalPriceMinorUnits != 1499 || applied.PlatformAmount != 360 || applied.TrainerAmount != 840 {
		t.Fatalf("20%% off 14,99 with the split recomputed, got %+v", applied)
	}
	if err := service.ApplyToPurchase(ctx, user.ID, purchase.ID, ""); err != nil {
		t.Fatalf("remove: %v", err)
	}
	tx.First(&applied, "id = ?", purchase.ID)
	if applied.PriceMinorUnits != 1499 || applied.DiscountCodeID != nil || applied.TrainerAmount != 1050 {
		t.Fatalf("removing the code restores the price: %+v", applied)
	}

	// Once the payment opened, the price no longer changes.
	tx.Exec(`UPDATE purchases SET payment_method = 'card' WHERE id = ?`, purchase.ID)
	if err := service.ApplyToPurchase(ctx, user.ID, purchase.ID, code); !errors.Is(err, discounts.ErrLocked) {
		t.Fatalf("locked after the payment opened: %v", err)
	}

	// The code only covers its family, and once per person.
	level2 := &repositories.DiscountTarget{ProductType: models.ProgramProductTypePremiumLevel2}
	if _, err := service.Check(ctx, user.ID, code, level2); !errors.Is(err, discounts.ErrNotApplicable) {
		t.Fatalf("a Level 1 code does not cover Level 2: %v", err)
	}
	tx.Exec(`UPDATE purchases SET payment_method = NULL WHERE id = ?`, purchase.ID)
	if err := service.ApplyToPurchase(ctx, user.ID, purchase.ID, code); err != nil {
		t.Fatalf("apply again: %v", err)
	}
	tx.Exec(`UPDATE purchases SET status = 'completed' WHERE id = ?`, purchase.ID)
	level1 := &repositories.DiscountTarget{ProductType: models.ProgramProductTypePremiumLevel1}
	if _, err := service.Check(ctx, user.ID, code, level1); !errors.Is(err, discounts.ErrAlreadyUsed) {
		t.Fatalf("once per person: %v", err)
	}
	if _, err := service.Check(ctx, user.ID, "NOPE123", level1); !errors.Is(err, discounts.ErrNotFound) {
		t.Fatalf("unknown code: %v", err)
	}
}
