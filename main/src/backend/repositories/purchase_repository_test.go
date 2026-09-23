package repositories_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

func TestPurchaseRepositoryListActiveByUserScoping(t *testing.T) {
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

	userRepo := repositories.NewUserRepository(tx)
	genericProgramRepo := repositories.NewGenericProgramRepository(tx)
	purchaseRepo := repositories.NewPurchaseRepository(tx)
	ctx := context.Background()

	seedUser := func() *models.User {
		user := &models.User{
			Email:        fmt.Sprintf("purchase-repo-%d@ryze.local", time.Now().UnixNano()),
			PasswordHash: "prepared-hash-outside-repository-scope",
			FirstName:    "Client",
			LastName:     "One",
		}
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("create user: %v", err)
		}
		return user
	}

	seedProgram := func(name string, price int64) *models.Program {
		program := &models.Program{
			Name:            name,
			Description:     "Description",
			Type:            models.ProgramTypePremium,
			Status:          models.ProgramStatusPublished,
			PriceMinorUnits: price,
			Currency:        string(models.ProgramCurrencyEUR),
		}
		if err := genericProgramRepo.CreateFull(ctx, program); err != nil {
			t.Fatalf("create program: %v", err)
		}
		return program
	}

	owner := seedUser()
	other := seedUser()
	program := seedProgram("Purchase Repo Program A", 4999)

	if err := purchaseRepo.Create(ctx, &models.Purchase{
		UserID:          owner.ID,
		ProgramID:       program.ID,
		PriceMinorUnits: 4999,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}); err != nil {
		t.Fatalf("create purchase: %v", err)
	}
	if err := purchaseRepo.Create(ctx, &models.Purchase{
		UserID:          other.ID,
		ProgramID:       program.ID,
		PriceMinorUnits: 4999,
		Currency:        "EUR",
		Status:          models.PurchaseStatusCompleted,
	}); err != nil {
		t.Fatalf("create purchase: %v", err)
	}

	list, err := purchaseRepo.ListActiveByUser(ctx, owner.ID)
	if err != nil {
		t.Fatalf("list purchases: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected exactly 1 purchase for the owner, got %d", len(list))
	}
	if list[0].UserID != owner.ID || list[0].ProgramID != program.ID {
		t.Fatalf("unexpected purchase row: %+v", list[0])
	}
	if list[0].Status != models.PurchaseStatusPending {
		t.Fatalf("expected pending purchase, got %q", list[0].Status)
	}
}

func TestPurchaseRepositoryCompletedPurchaseUniqueConstraint(t *testing.T) {
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

	userRepo := repositories.NewUserRepository(tx)
	genericProgramRepo := repositories.NewGenericProgramRepository(tx)
	purchaseRepo := repositories.NewPurchaseRepository(tx)
	ctx := context.Background()

	user := &models.User{
		Email:        fmt.Sprintf("purchase-unique-%d@ryze.local", time.Now().UnixNano()),
		PasswordHash: "prepared-hash-outside-repository-scope",
		FirstName:    "Client",
		LastName:     "Two",
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	program := &models.Program{
		Name:            "Purchase Repo Program B",
		Description:     "Description",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 8999,
		Currency:        string(models.ProgramCurrencyEUR),
	}
	if err := genericProgramRepo.CreateFull(ctx, program); err != nil {
		t.Fatalf("create program: %v", err)
	}

	newPurchase := func(status string) *models.Purchase {
		return &models.Purchase{
			UserID:          user.ID,
			ProgramID:       program.ID,
			PriceMinorUnits: 8999,
			Currency:        "EUR",
			Status:          status,
		}
	}

	if err := purchaseRepo.Create(ctx, newPurchase(models.PurchaseStatusCompleted)); err != nil {
		t.Fatalf("create first completed purchase: %v", err)
	}

	err = purchaseRepo.Create(ctx, newPurchase(models.PurchaseStatusCompleted))
	if err == nil {
		t.Fatal("expected a duplicate entry error for a second completed purchase")
	}
	if !strings.Contains(err.Error(), "uq_purchases_completed_purchase") {
		t.Fatalf("expected the completed-purchase unique constraint to reject the row, got %v", err)
	}

	if err := purchaseRepo.Create(ctx, newPurchase(models.PurchaseStatusPending)); err != nil {
		t.Fatalf("a pending retry after a completed purchase must remain possible, got %v", err)
	}

	// A different user on the same program is a distinct pair and must be
	// allowed, proving the constraint is scoped to (user, program).
	otherUser := &models.User{
		Email:        fmt.Sprintf("purchase-unique-other-%d@ryze.local", time.Now().UnixNano()),
		PasswordHash: "prepared-hash-outside-repository-scope",
		FirstName:    "Client",
		LastName:     "Three",
	}
	if err := userRepo.Create(ctx, otherUser); err != nil {
		t.Fatalf("create other user: %v", err)
	}
	if err := purchaseRepo.Create(ctx, &models.Purchase{
		UserID:          otherUser.ID,
		ProgramID:       program.ID,
		PriceMinorUnits: 8999,
		Currency:        "EUR",
		Status:          models.PurchaseStatusCompleted,
	}); err != nil {
		t.Fatalf("a completed purchase by another user must remain possible, got %v", err)
	}
}
