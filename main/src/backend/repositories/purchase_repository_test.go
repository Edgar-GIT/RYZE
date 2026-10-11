package repositories_test

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
	if !errors.Is(err, repositories.ErrPurchaseAlreadyExists) {
		t.Fatalf("expected a second completed purchase for the pair to map to ErrPurchaseAlreadyExists, got %v", err)
	}

	// A raw duplicate of an already-created row is the same verdict: the
	// repository never leaks a driver-level constraint name.
	if err := purchaseRepo.Create(ctx, newPurchase(models.PurchaseStatusCompleted)); !errors.Is(err, repositories.ErrPurchaseAlreadyExists) {
		t.Fatalf("expected the completed-purchase duplicate to keep mapping to ErrPurchaseAlreadyExists, got %v", err)
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

func TestPurchaseRepositorySetPaymentMethod(t *testing.T) {
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
		Email:        fmt.Sprintf("purchase-method-%d@ryze.local", time.Now().UnixNano()),
		PasswordHash: "prepared-hash-outside-repository-scope",
		FirstName:    "Client",
		LastName:     "Four",
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	program := &models.Program{
		Name:            "Purchase Repo Program C",
		Description:     "Description",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 4999,
		Currency:        string(models.ProgramCurrencyEUR),
	}
	if err := genericProgramRepo.CreateFull(ctx, program); err != nil {
		t.Fatalf("create program: %v", err)
	}

	purchase := &models.Purchase{
		UserID:          user.ID,
		ProgramID:       program.ID,
		PriceMinorUnits: 4999,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}
	if err := purchaseRepo.Create(ctx, purchase); err != nil {
		t.Fatalf("create purchase: %v", err)
	}

	if err := purchaseRepo.SetPaymentMethod(ctx, purchase.ID, "paypal"); err != nil {
		t.Fatalf("set payment method: %v", err)
	}

	loaded, err := purchaseRepo.FindByID(ctx, purchase.ID)
	if err != nil {
		t.Fatalf("find purchase: %v", err)
	}
	if loaded.PaymentMethod == nil || *loaded.PaymentMethod != "paypal" {
		t.Fatalf("expected recorded payment method %q, got %v", "paypal", loaded.PaymentMethod)
	}
	if loaded.Status != models.PurchaseStatusPending {
		t.Fatalf("recording the method must not change the purchase status, got %q", loaded.Status)
	}

	if err := purchaseRepo.SetPaymentMethod(ctx, "missing-purchase-id", "card"); !errors.Is(err, repositories.ErrPurchaseNotFound) {
		t.Fatalf("unknown purchase must surface as ErrPurchaseNotFound, got %v", err)
	}
}

func TestPurchaseRepositoryFail(t *testing.T) {
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
		Email:        fmt.Sprintf("purchase-fail-%d@ryze.local", time.Now().UnixNano()),
		PasswordHash: "prepared-hash-outside-repository-scope",
		FirstName:    "Client",
		LastName:     "Five",
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	program := &models.Program{
		Name:            "Purchase Repo Program Fail",
		Description:     "Description",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 4999,
		Currency:        string(models.ProgramCurrencyEUR),
	}
	if err := genericProgramRepo.CreateFull(ctx, program); err != nil {
		t.Fatalf("create program: %v", err)
	}

	pending := &models.Purchase{
		UserID:          user.ID,
		ProgramID:       program.ID,
		PriceMinorUnits: 4999,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}
	if err := purchaseRepo.Create(ctx, pending); err != nil {
		t.Fatalf("create pending purchase: %v", err)
	}

	if err := purchaseRepo.Fail(ctx, pending.ID); err != nil {
		t.Fatalf("a pending purchase must transition to failed, got %v", err)
	}
	loaded, err := purchaseRepo.FindByID(ctx, pending.ID)
	if err != nil {
		t.Fatalf("find purchase: %v", err)
	}
	if loaded.Status != models.PurchaseStatusFailed {
		t.Fatalf("expected failed status, got %q", loaded.Status)
	}

	if err := purchaseRepo.Fail(ctx, pending.ID); !errors.Is(err, repositories.ErrPurchaseNotFound) {
		t.Fatalf("a second fail must be refused by the pending guard, got %v", err)
	}

	// A completed purchase is equally guarded: a late supersede attempt must
	// not downgrade a captured purchase.
	completed := &models.Purchase{
		UserID:          user.ID,
		ProgramID:       program.ID,
		PriceMinorUnits: 4999,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}
	if err := purchaseRepo.Create(ctx, completed); err != nil {
		t.Fatalf("create second pending purchase: %v", err)
	}
	if err := purchaseRepo.CompleteWithEntitlement(ctx, completed.ID, &models.Entitlement{UserID: user.ID, ProgramID: program.ID}); err != nil {
		t.Fatalf("complete purchase: %v", err)
	}
	if err := purchaseRepo.Fail(ctx, completed.ID); !errors.Is(err, repositories.ErrPurchaseNotFound) {
		t.Fatalf("a completed purchase must be left untouched by Fail, got %v", err)
	}
	loaded, err = purchaseRepo.FindByID(ctx, completed.ID)
	if err != nil {
		t.Fatalf("find purchase: %v", err)
	}
	if loaded.Status != models.PurchaseStatusCompleted {
		t.Fatalf("Fail must not downgrade a completed purchase, got %q", loaded.Status)
	}
}

func TestPurchaseRepositoryCompleteWithRestoredEntitlement(t *testing.T) {
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
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	ctx := context.Background()

	user := &models.User{
		Email:        fmt.Sprintf("purchase-restore-%d@ryze.local", time.Now().UnixNano()),
		PasswordHash: "prepared-hash-outside-repository-scope",
		FirstName:    "Client",
		LastName:     "Six",
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	program := &models.Program{
		Name:            "Purchase Repo Program Restore",
		Description:     "Description",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 5999,
		Currency:        string(models.ProgramCurrencyEUR),
	}
	if err := genericProgramRepo.CreateFull(ctx, program); err != nil {
		t.Fatalf("create program: %v", err)
	}

	entitlement := &models.Entitlement{}
	if err := entitlementRepo.Create(ctx, user.ID, program.ID, entitlement); err != nil {
		t.Fatalf("create entitlement: %v", err)
	}
	if err := tx.Where("id = ?", entitlement.ID).Delete(&models.Entitlement{}).Error; err != nil {
		t.Fatalf("soft-delete entitlement: %v", err)
	}

	purchase := &models.Purchase{
		UserID:          user.ID,
		ProgramID:       program.ID,
		PriceMinorUnits: 5999,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}
	if err := purchaseRepo.Create(ctx, purchase); err != nil {
		t.Fatalf("create purchase: %v", err)
	}

	if err := purchaseRepo.CompleteWithRestoredEntitlement(ctx, purchase.ID, entitlement); err != nil {
		t.Fatalf("restore and complete: %v", err)
	}

	var restored models.Entitlement
	if err := tx.Unscoped().Where("id = ?", entitlement.ID).First(&restored).Error; err != nil {
		t.Fatalf("load entitlement: %v", err)
	}
	if restored.DeletedAt.Valid {
		t.Fatal("expected the entitlement to be restored (deleted_at cleared)")
	}

	loaded, err := purchaseRepo.FindByID(ctx, purchase.ID)
	if err != nil {
		t.Fatalf("find purchase: %v", err)
	}
	if loaded.Status != models.PurchaseStatusCompleted {
		t.Fatalf("expected completed status, got %q", loaded.Status)
	}

	// Nothing is left to restore now: a replay must fail closed without
	// touching the already-active entitlement.
	err = purchaseRepo.CompleteWithRestoredEntitlement(ctx, purchase.ID, entitlement)
	if !errors.Is(err, repositories.ErrEntitlementNotFound) {
		t.Fatalf("expected ErrEntitlementNotFound after the entitlement was restored, got %v", err)
	}
}

func TestPurchaseRepositoryCompleteWithRestoredEntitlementRollsBackOnNonPending(t *testing.T) {
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
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	ctx := context.Background()

	user := &models.User{
		Email:        fmt.Sprintf("purchase-restore-race-%d@ryze.local", time.Now().UnixNano()),
		PasswordHash: "prepared-hash-outside-repository-scope",
		FirstName:    "Client",
		LastName:     "Seven",
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	program := &models.Program{
		Name:            "Purchase Repo Program Restore Race",
		Description:     "Description",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 6999,
		Currency:        string(models.ProgramCurrencyEUR),
	}
	if err := genericProgramRepo.CreateFull(ctx, program); err != nil {
		t.Fatalf("create program: %v", err)
	}

	// A repurchase came in, then the buyer gave up and the supersede marked it
	// failed. A completion event arriving afterwards must not resurrect the
	// soft-deleted entitlement: the guarded purchase update fails and the whole
	// transaction must roll back.
	entitlement := &models.Entitlement{}
	if err := entitlementRepo.Create(ctx, user.ID, program.ID, entitlement); err != nil {
		t.Fatalf("create entitlement: %v", err)
	}
	if err := tx.Where("id = ?", entitlement.ID).Delete(&models.Entitlement{}).Error; err != nil {
		t.Fatalf("soft-delete entitlement: %v", err)
	}

	purchase := &models.Purchase{
		UserID:          user.ID,
		ProgramID:       program.ID,
		PriceMinorUnits: 6999,
		Currency:        "EUR",
		Status:          models.PurchaseStatusFailed,
	}
	if err := purchaseRepo.Create(ctx, purchase); err != nil {
		t.Fatalf("create failed purchase: %v", err)
	}

	err = purchaseRepo.CompleteWithRestoredEntitlement(ctx, purchase.ID, entitlement)
	if !errors.Is(err, repositories.ErrPurchaseNotFound) {
		t.Fatalf("a non-pending purchase must refuse completion, got %v", err)
	}

	var stillDeleted models.Entitlement
	if err := tx.Unscoped().Where("id = ?", entitlement.ID).First(&stillDeleted).Error; err != nil {
		t.Fatalf("load entitlement: %v", err)
	}
	if !stillDeleted.DeletedAt.Valid {
		t.Fatal("the failed completion must roll back the entitlement restore")
	}
}

func TestPurchaseRepositoryCreateRejectsSecondPendingIntent(t *testing.T) {
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
		Email:        fmt.Sprintf("purchase-single-pending-%d@ryze.local", time.Now().UnixNano()),
		PasswordHash: "prepared-hash-outside-repository-scope",
		FirstName:    "Client",
		LastName:     "Eight",
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	program := &models.Program{
		Name:            "Purchase Repo Program Single Pending",
		Description:     "Description",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 5999,
		Currency:        string(models.ProgramCurrencyEUR),
	}
	if err := genericProgramRepo.CreateFull(ctx, program); err != nil {
		t.Fatalf("create program: %v", err)
	}

	newPurchase := func(status string) *models.Purchase {
		return &models.Purchase{
			UserID:          user.ID,
			ProgramID:       program.ID,
			PriceMinorUnits: 5999,
			Currency:        "EUR",
			Status:          status,
		}
	}

	if err := purchaseRepo.Create(ctx, newPurchase(models.PurchaseStatusPending)); err != nil {
		t.Fatalf("create first pending purchase: %v", err)
	}

	err = purchaseRepo.Create(ctx, newPurchase(models.PurchaseStatusPending))
	if !errors.Is(err, repositories.ErrPurchaseAlreadyExists) {
		t.Fatalf("a second pending intent for the same pair must be rejected with ErrPurchaseAlreadyExists, got %v", err)
	}
}

func TestPurchaseRepositoryReconcileFailedPurchase(t *testing.T) {
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
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	ctx := context.Background()

	user := &models.User{
		Email:        fmt.Sprintf("purchase-reconcile-%d@ryze.local", time.Now().UnixNano()),
		PasswordHash: "prepared-hash-outside-repository-scope",
		FirstName:    "Client",
		LastName:     "Nine",
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	program := &models.Program{
		Name:            "Purchase Repo Program Reconcile",
		Description:     "Description",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 5999,
		Currency:        string(models.ProgramCurrencyEUR),
	}
	if err := genericProgramRepo.CreateFull(ctx, program); err != nil {
		t.Fatalf("create program: %v", err)
	}

	newPurchase := func(status string) *models.Purchase {
		return &models.Purchase{
			UserID:          user.ID,
			ProgramID:       program.ID,
			PriceMinorUnits: 5999,
			Currency:        "EUR",
			Status:          status,
		}
	}

	// A stale pending sibling for the same pair must be superseded by the
	// reconciliation: a refailed intent can neither accept a second capture nor
	// block a fresh one once the late payment owned the program.
	failed := newPurchase(models.PurchaseStatusFailed)
	if err := purchaseRepo.Create(ctx, failed); err != nil {
		t.Fatalf("create failed purchase: %v", err)
	}
	sibling := newPurchase(models.PurchaseStatusPending)
	if err := purchaseRepo.Create(ctx, sibling); err != nil {
		t.Fatalf("create pending sibling: %v", err)
	}

	entitlement := &models.Entitlement{UserID: user.ID, ProgramID: program.ID}
	if err := purchaseRepo.ReconcileFailedPurchase(ctx, failed.ID, user.ID, program.ID, entitlement); err != nil {
		t.Fatalf("reconcile failed purchase: %v", err)
	}

	var reconciled models.Purchase
	if err := tx.Where("id = ?", failed.ID).First(&reconciled).Error; err != nil {
		t.Fatalf("load reconciled purchase: %v", err)
	}
	if reconciled.Status != models.PurchaseStatusCompleted {
		t.Fatalf("expected the reconciled purchase to be completed, got %s", reconciled.Status)
	}

	var superseded models.Purchase
	if err := tx.Where("id = ?", sibling.ID).First(&superseded).Error; err != nil {
		t.Fatalf("load superseded sibling: %v", err)
	}
	if superseded.Status != models.PurchaseStatusFailed {
		t.Fatalf("expected the stale pending sibling to be superseded, got %s", superseded.Status)
	}

	ent, err := entitlementRepo.FindActiveByUserAndProgram(ctx, user.ID, program.ID)
	if err != nil {
		t.Fatalf("the reconciliation must provision an active entitlement, got %v", err)
	}
	if ent.ProgramID != program.ID || ent.UserID != user.ID {
		t.Fatalf("reconciled entitlement does not match the pair: user=%s program=%s", ent.UserID, ent.ProgramID)
	}
}

func TestPurchaseRepositoryReconcileFailedPurchaseRestoresSoftDeletedEntitlement(t *testing.T) {
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
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	ctx := context.Background()

	user := &models.User{
		Email:        fmt.Sprintf("purchase-reconcile-restore-%d@ryze.local", time.Now().UnixNano()),
		PasswordHash: "prepared-hash-outside-repository-scope",
		FirstName:    "Client",
		LastName:     "Ten",
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	program := &models.Program{
		Name:            "Purchase Repo Program Reconcile Restore",
		Description:     "Description",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 5999,
		Currency:        string(models.ProgramCurrencyEUR),
	}
	if err := genericProgramRepo.CreateFull(ctx, program); err != nil {
		t.Fatalf("create program: %v", err)
	}

	failed := &models.Purchase{
		UserID:          user.ID,
		ProgramID:       program.ID,
		PriceMinorUnits: 5999,
		Currency:        "EUR",
		Status:          models.PurchaseStatusFailed,
	}
	if err := purchaseRepo.Create(ctx, failed); err != nil {
		t.Fatalf("create failed purchase: %v", err)
	}

	// The buyer previously owned the program and it was soft-deleted during a
	// supersede; the late settlement must restore, not duplicate, it.
	entitlement := &models.Entitlement{UserID: user.ID, ProgramID: program.ID}
	if err := entitlementRepo.Create(ctx, user.ID, program.ID, entitlement); err != nil {
		t.Fatalf("create entitlement: %v", err)
	}
	if err := tx.Where("id = ?", entitlement.ID).Delete(&models.Entitlement{}).Error; err != nil {
		t.Fatalf("soft-delete entitlement: %v", err)
	}

	if err := purchaseRepo.ReconcileFailedPurchase(ctx, failed.ID, user.ID, program.ID, &models.Entitlement{UserID: user.ID, ProgramID: program.ID}); err != nil {
		t.Fatalf("reconcile failed purchase: %v", err)
	}

	var restored models.Entitlement
	if err := tx.Unscoped().Where("id = ?", entitlement.ID).First(&restored).Error; err != nil {
		t.Fatalf("load restored entitlement: %v", err)
	}
	if restored.DeletedAt.Valid {
		t.Fatal("the reconciliation must restore the soft-deleted entitlement")
	}
}

func TestPurchaseRepositoryReconcileFailedPurchaseCompletedSiblingRejected(t *testing.T) {
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
		Email:        fmt.Sprintf("purchase-reconcile-conflict-%d@ryze.local", time.Now().UnixNano()),
		PasswordHash: "prepared-hash-outside-repository-scope",
		FirstName:    "Client",
		LastName:     "Eleven",
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	program := &models.Program{
		Name:            "Purchase Repo Program Reconcile Conflict",
		Description:     "Description",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 5999,
		Currency:        string(models.ProgramCurrencyEUR),
	}
	if err := genericProgramRepo.CreateFull(ctx, program); err != nil {
		t.Fatalf("create program: %v", err)
	}

	newPurchase := func(status string) *models.Purchase {
		return &models.Purchase{
			UserID:          user.ID,
			ProgramID:       program.ID,
			PriceMinorUnits: 5999,
			Currency:        "EUR",
			Status:          status,
		}
	}

	if err := purchaseRepo.Create(ctx, newPurchase(models.PurchaseStatusCompleted)); err != nil {
		t.Fatalf("create completed purchase: %v", err)
	}
	failed := newPurchase(models.PurchaseStatusFailed)
	if err := purchaseRepo.Create(ctx, failed); err != nil {
		t.Fatalf("create failed purchase: %v", err)
	}

	err = purchaseRepo.ReconcileFailedPurchase(ctx, failed.ID, user.ID, program.ID, &models.Entitlement{UserID: user.ID, ProgramID: program.ID})
	if !errors.Is(err, repositories.ErrCompletedPurchaseExists) {
		t.Fatalf("a completed sibling must reject the reconciliation with ErrCompletedPurchaseExists, got %v", err)
	}

	var stillFailed models.Purchase
	if err := tx.Where("id = ?", failed.ID).First(&stillFailed).Error; err != nil {
		t.Fatalf("load purchase: %v", err)
	}
	if stillFailed.Status != models.PurchaseStatusFailed {
		t.Fatalf("the rejected reconciliation must roll back, expected failed, got %s", stillFailed.Status)
	}
}

func TestPurchaseRepositoryReconcileFailedPurchaseActiveEntitlementRejected(t *testing.T) {
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
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	ctx := context.Background()

	user := &models.User{
		Email:        fmt.Sprintf("purchase-reconcile-active-ent-%d@ryze.local", time.Now().UnixNano()),
		PasswordHash: "prepared-hash-outside-repository-scope",
		FirstName:    "Client",
		LastName:     "Twelve",
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	program := &models.Program{
		Name:            "Purchase Repo Program Reconcile Active Ent",
		Description:     "Description",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 5999,
		Currency:        string(models.ProgramCurrencyEUR),
	}
	if err := genericProgramRepo.CreateFull(ctx, program); err != nil {
		t.Fatalf("create program: %v", err)
	}

	failed := &models.Purchase{
		UserID:          user.ID,
		ProgramID:       program.ID,
		PriceMinorUnits: 5999,
		Currency:        "EUR",
		Status:          models.PurchaseStatusFailed,
	}
	if err := purchaseRepo.Create(ctx, failed); err != nil {
		t.Fatalf("create failed purchase: %v", err)
	}

	if err := entitlementRepo.Create(ctx, user.ID, program.ID, &models.Entitlement{UserID: user.ID, ProgramID: program.ID}); err != nil {
		t.Fatalf("create active entitlement: %v", err)
	}

	err = purchaseRepo.ReconcileFailedPurchase(ctx, failed.ID, user.ID, program.ID, &models.Entitlement{UserID: user.ID, ProgramID: program.ID})
	if !errors.Is(err, repositories.ErrEntitlementAlreadyExists) {
		t.Fatalf("an active entitlement must reject the reconciliation with ErrEntitlementAlreadyExists, got %v", err)
	}

	var stillFailed models.Purchase
	if err := tx.Where("id = ?", failed.ID).First(&stillFailed).Error; err != nil {
		t.Fatalf("load purchase: %v", err)
	}
	if stillFailed.Status != models.PurchaseStatusFailed {
		t.Fatalf("the rejected reconciliation must roll back, expected failed, got %s", stillFailed.Status)
	}
}

func TestPurchaseRepositoryReconcileFailedPurchaseNonFailedRejected(t *testing.T) {
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
		Email:        fmt.Sprintf("purchase-reconcile-nonfailed-%d@ryze.local", time.Now().UnixNano()),
		PasswordHash: "prepared-hash-outside-repository-scope",
		FirstName:    "Client",
		LastName:     "Thirteen",
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	program := &models.Program{
		Name:            "Purchase Repo Program Reconcile Non Failed",
		Description:     "Description",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 5999,
		Currency:        string(models.ProgramCurrencyEUR),
	}
	if err := genericProgramRepo.CreateFull(ctx, program); err != nil {
		t.Fatalf("create program: %v", err)
	}

	pending := &models.Purchase{
		UserID:          user.ID,
		ProgramID:       program.ID,
		PriceMinorUnits: 5999,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}
	if err := purchaseRepo.Create(ctx, pending); err != nil {
		t.Fatalf("create pending purchase: %v", err)
	}

	err = purchaseRepo.ReconcileFailedPurchase(ctx, pending.ID, user.ID, program.ID, &models.Entitlement{UserID: user.ID, ProgramID: program.ID})
	if !errors.Is(err, repositories.ErrPurchaseNotFound) {
		t.Fatalf("a non-failed purchase must reject reconciliation with ErrPurchaseNotFound, got %v", err)
	}

	var unchanged models.Purchase
	if err := tx.Where("id = ?", pending.ID).First(&unchanged).Error; err != nil {
		t.Fatalf("load purchase: %v", err)
	}
	if unchanged.Status != models.PurchaseStatusPending {
		t.Fatalf("the rejected reconciliation must leave the purchase pending, got %s", unchanged.Status)
	}
}
