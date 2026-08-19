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

func TestEntitlementRepository(t *testing.T) {
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
	trainerRepo := repositories.NewTrainerRepository(tx)
	programRepo := repositories.NewProgramRepository(tx)
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	ctx := context.Background()

	seedUser := func() *models.User {
		user := &models.User{
			Email:        fmt.Sprintf("entitlement-repo-%d@ryze.local", time.Now().UnixNano()),
			PasswordHash: "prepared-hash-outside-repository-scope",
			FirstName:    "Jane",
			LastName:     "Roe",
		}
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("create user: %v", err)
		}
		return user
	}

	seedTrainer := func() *models.Trainer {
		trainer := &models.Trainer{UserID: seedUser().ID}
		if err := trainerRepo.Create(ctx, trainer); err != nil {
			t.Fatalf("create trainer: %v", err)
		}
		return trainer
	}

	seedProgram := func(trainerID, name string) *models.Program {
		program := &models.Program{
			TrainerID: trainerID,
			Name:      name,
			Type:      models.ProgramTypePremium,
			Status:    models.ProgramStatusPublished,
		}
		if err := programRepo.Create(ctx, program); err != nil {
			t.Fatalf("create program: %v", err)
		}
		return program
	}

	trainer := seedTrainer()
	user1 := seedUser()
	user2 := seedUser()
	program1 := seedProgram(trainer.ID, "Strength Builder")
	program2 := seedProgram(trainer.ID, "HIIT Blaster")

	// 1. Create persists a new entitlement for the given user and program.
	entitlement := &models.Entitlement{}
	if err := entitlementRepo.Create(ctx, user1.ID, program1.ID, entitlement); err != nil {
		t.Fatalf("create entitlement: %v", err)
	}
	if entitlement.ID == "" {
		t.Fatal("create entitlement: expected generated UUID id")
	}
	if entitlement.UserID != user1.ID {
		t.Fatalf("create entitlement: expected user %q, got %q", user1.ID, entitlement.UserID)
	}
	if entitlement.ProgramID != program1.ID {
		t.Fatalf("create entitlement: expected program %q, got %q", program1.ID, entitlement.ProgramID)
	}
	if entitlement.CreatedAt.IsZero() || entitlement.UpdatedAt.IsZero() {
		t.Fatal("create entitlement: expected non-zero timestamps")
	}

	// 2. A duplicate active entitlement for the same (user, program) pair is
	// rejected by the unique generated column constraint.
	if err := entitlementRepo.Create(ctx, user1.ID, program1.ID, &models.Entitlement{}); !errors.Is(err, repositories.ErrEntitlementAlreadyExists) {
		t.Fatalf("duplicate entitlement: expected ErrEntitlementAlreadyExists, got %v", err)
	}

	// 3. A different user can hold an entitlement for the same program.
	if err := entitlementRepo.Create(ctx, user2.ID, program1.ID, &models.Entitlement{}); err != nil {
		t.Fatalf("cross-user entitlement: %v", err)
	}

	// 4. A user can hold entitlements for different programs.
	if err := entitlementRepo.Create(ctx, user1.ID, program2.ID, &models.Entitlement{}); err != nil {
		t.Fatalf("cross-program entitlement: %v", err)
	}

	// 5. ListByUser returns the user's active entitlements in chronological
	// order with the associated program data preloaded.
	entitlements, err := entitlementRepo.ListByUser(ctx, user1.ID)
	if err != nil {
		t.Fatalf("list entitlements: %v", err)
	}
	if len(entitlements) != 2 {
		t.Fatalf("list entitlements: expected 2, got %d", len(entitlements))
	}
	if entitlements[0].ID != entitlement.ID || entitlements[0].ProgramID != program1.ID {
		t.Fatalf("list entitlements: unexpected entitlement %+v", entitlements[0])
	}
	if entitlements[0].Program.ID != program1.ID || entitlements[0].Program.Name != "Strength Builder" {
		t.Fatalf("list entitlements: expected program data, got %+v", entitlements[0].Program)
	}
	if entitlements[1].Program.ID != program2.ID || entitlements[1].Program.Name != "HIIT Blaster" {
		t.Fatalf("list entitlements: expected program data, got %+v", entitlements[1].Program)
	}

	// 6. A user without entitlements returns an empty list.
	stranger := seedUser()
	empty, err := entitlementRepo.ListByUser(ctx, stranger.ID)
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("list empty: expected 0, got %d", len(empty))
	}

	// 7. FindByIDAndUser returns one entitlement scoped by the user.
	found, err := entitlementRepo.FindByIDAndUser(ctx, user1.ID, entitlement.ID)
	if err != nil {
		t.Fatalf("find entitlement: %v", err)
	}
	if found.ID != entitlement.ID || found.ProgramID != program1.ID || found.UserID != user1.ID {
		t.Fatalf("find entitlement: unexpected entitlement %+v", found)
	}
	if found.Program.ID != program1.ID {
		t.Fatalf("find entitlement: expected program embedded, got %+v", found.Program)
	}

	// 8. A cross-user lookup or unknown id is indistinguishable and never
	// revealed.
	if _, err := entitlementRepo.FindByIDAndUser(ctx, user2.ID, entitlement.ID); !errors.Is(err, repositories.ErrEntitlementNotFound) {
		t.Fatalf("cross-user find: expected ErrEntitlementNotFound, got %v", err)
	}
	if _, err := entitlementRepo.FindByIDAndUser(ctx, user1.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrEntitlementNotFound) {
		t.Fatalf("unknown find: expected ErrEntitlementNotFound, got %v", err)
	}

	// 9. SoftDelete removes the entitlement from find and list but keeps the
	// row and never touches the program.
	if err := entitlementRepo.SoftDelete(ctx, user1.ID, entitlement.ID); err != nil {
		t.Fatalf("soft delete entitlement: %v", err)
	}
	if _, err := entitlementRepo.FindByIDAndUser(ctx, user1.ID, entitlement.ID); !errors.Is(err, repositories.ErrEntitlementNotFound) {
		t.Fatalf("soft-deleted entitlement must not be found, got %v", err)
	}
	var deletedRecord models.Entitlement
	if err := tx.Unscoped().First(&deletedRecord, "id = ?", entitlement.ID).Error; err != nil {
		t.Fatalf("soft-deleted row must be preserved: %v", err)
	}
	if !deletedRecord.DeletedAt.Valid {
		t.Fatal("soft-deleted entitlement must carry a deleted_at marker")
	}
	var programAfterDelete models.Program
	if err := tx.First(&programAfterDelete, "id = ?", program1.ID).Error; err != nil {
		t.Fatalf("program must survive the entitlement delete: %v", err)
	}
	remaining, err := entitlementRepo.ListByUser(ctx, user1.ID)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(remaining) != 1 || remaining[0].ID != entitlements[1].ID {
		t.Fatalf("expected the other entitlement to survive, got %d", len(remaining))
	}

	// 10. Soft deleting a foreign or unknown entitlement maps to not found.
	if err := entitlementRepo.SoftDelete(ctx, user2.ID, entitlement.ID); !errors.Is(err, repositories.ErrEntitlementNotFound) {
		t.Fatalf("cross-user soft delete: expected ErrEntitlementNotFound, got %v", err)
	}
	if err := entitlementRepo.SoftDelete(ctx, user1.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrEntitlementNotFound) {
		t.Fatalf("unknown soft delete: expected ErrEntitlementNotFound, got %v", err)
	}

	// 11. A soft-deleted entitlement no longer blocks a new entitlement for the
	// same (user, program) pair.
	if err := entitlementRepo.Create(ctx, user1.ID, program1.ID, &models.Entitlement{}); err != nil {
		t.Fatalf("recreate after delete: %v", err)
	}
}

// TestEntitlementRepositoryListByUserEmpty ensures a user with no
// entitlements receives an empty list without error.
func TestEntitlementRepositoryListByUserEmpty(t *testing.T) {
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

	entitlementRepo := repositories.NewEntitlementRepository(tx)
	ctx := context.Background()

	entitlements, err := entitlementRepo.ListByUser(ctx, "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entitlements) != 0 {
		t.Fatalf("expected empty list, got %d", len(entitlements))
	}
}
