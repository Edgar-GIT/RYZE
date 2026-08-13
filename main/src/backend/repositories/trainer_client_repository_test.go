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

func TestTrainerClientRepository(t *testing.T) {
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
	clientRepo := repositories.NewTrainerClientRepository(tx)
	ctx := context.Background()

	seedUser := func() *models.User {
		user := &models.User{
			Email:        fmt.Sprintf("trainer-client-repo-%d@ryze.local", time.Now().UnixNano()),
			PasswordHash: "prepared-hash-outside-repository-scope",
			FirstName:    "John",
			LastName:     "Doe",
		}
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("create user: %v", err)
		}
		return user
	}

	seedTrainer := func(user *models.User) *models.Trainer {
		trainer := &models.Trainer{UserID: user.ID}
		if err := trainerRepo.Create(ctx, trainer); err != nil {
			t.Fatalf("create trainer: %v", err)
		}
		return trainer
	}

	trainer := seedTrainer(seedUser())
	client := seedUser()
	otherTrainer := seedTrainer(seedUser())
	otherClient := seedUser()

	// 1. Create an active relationship.
	relation := &models.TrainerClient{TrainerID: trainer.ID, UserID: client.ID}
	if err := clientRepo.Create(ctx, relation); err != nil {
		t.Fatalf("create relationship: %v", err)
	}
	if relation.ID == "" {
		t.Fatal("create relationship: expected generated UUID id")
	}
	if relation.CreatedAt.IsZero() || relation.UpdatedAt.IsZero() {
		t.Fatal("create relationship: expected non-zero timestamps")
	}

	// 2. A second active relationship for the same pair is rejected.
	duplicate := &models.TrainerClient{TrainerID: trainer.ID, UserID: client.ID}
	if err := clientRepo.Create(ctx, duplicate); !errors.Is(err, repositories.ErrClientRelationAlreadyActive) {
		t.Fatalf("duplicate active relationship: expected ErrClientRelationAlreadyActive, got %v", err)
	}

	// 3. The database rejects a relationship without a valid trainer.
	if err := clientRepo.Create(ctx, &models.TrainerClient{
		TrainerID: "00000000-0000-0000-0000-000000000000",
		UserID:    client.ID,
	}); err == nil {
		t.Fatal("relationship with unknown trainer: expected an error")
	} else if errors.Is(err, repositories.ErrClientRelationAlreadyActive) {
		t.Fatal("relationship with unknown trainer must not map to a duplicate error")
	}

	// 4. The database rejects a relationship without a valid user.
	if err := clientRepo.Create(ctx, &models.TrainerClient{
		TrainerID: trainer.ID,
		UserID:    "00000000-0000-0000-0000-000000000000",
	}); err == nil {
		t.Fatal("relationship with unknown user: expected an error")
	} else if errors.Is(err, repositories.ErrClientRelationAlreadyActive) {
		t.Fatal("relationship with unknown user must not map to a duplicate error")
	}

	// 5. The database rejects a self relationship (the same UUID used as both
	// trainer_id and user_id). The CHECK constraint enforces it structurally
	// even when the same UUID exists in both referenced tables.
	selfUser := seedUser()
	selfTrainer := &models.Trainer{ID: selfUser.ID, UserID: selfUser.ID}
	if err := trainerRepo.Create(ctx, selfTrainer); err != nil {
		t.Fatalf("create self trainer: %v", err)
	}
	if err := clientRepo.Create(ctx, &models.TrainerClient{
		TrainerID: selfUser.ID,
		UserID:    selfUser.ID,
	}); err == nil {
		t.Fatal("self relationship: expected an error")
	} else if errors.Is(err, repositories.ErrClientRelationAlreadyActive) {
		t.Fatal("self relationship must not map to a duplicate error")
	}

	// 6. Find the active relationship with the linked user preloaded.
	found, err := clientRepo.FindActiveByTrainerAndUser(ctx, trainer.ID, client.ID)
	if err != nil {
		t.Fatalf("find active relationship: %v", err)
	}
	if found.ID != relation.ID {
		t.Fatalf("find active: expected relation %q, got %q", relation.ID, found.ID)
	}
	if found.User.ID != client.ID {
		t.Fatalf("find active: expected linked user %q, got %q", client.ID, found.User.ID)
	}

	// 7. The same user under another trainer is its own relationship.
	otherRelation := &models.TrainerClient{TrainerID: otherTrainer.ID, UserID: otherClient.ID}
	if err := clientRepo.Create(ctx, otherRelation); err != nil {
		t.Fatalf("create second relationship: %v", err)
	}

	// 8. List only returns the clients of the requested trainer (isolation).
	relations, total, err := clientRepo.ListActiveClients(ctx, trainer.ID, 1, 20)
	if err != nil {
		t.Fatalf("list active clients: %v", err)
	}
	if total != 1 {
		t.Fatalf("list active: expected total 1 for trainer A, got %d", total)
	}
	if len(relations) != 1 || relations[0].UserID != client.ID {
		t.Fatalf("list active: expected only client %q, got %+v", client.ID, relations)
	}

	otherRelations, otherTotal, err := clientRepo.ListActiveClients(ctx, otherTrainer.ID, 1, 20)
	if err != nil {
		t.Fatalf("list other trainer clients: %v", err)
	}
	if otherTotal != 1 || len(otherRelations) != 1 || otherRelations[0].UserID != otherClient.ID {
		t.Fatalf("list other: expected only client %q, got %+v", otherClient.ID, otherRelations)
	}

	// 9. Soft delete the relationship: only the row is touched.
	if err := clientRepo.SoftDelete(ctx, trainer.ID, client.ID); err != nil {
		t.Fatalf("soft delete relationship: %v", err)
	}
	if _, err := clientRepo.FindActiveByTrainerAndUser(ctx, trainer.ID, client.ID); !errors.Is(err, repositories.ErrClientRelationNotFound) {
		t.Fatalf("find after delete: expected ErrClientRelationNotFound, got %v", err)
	}

	// 10. The user and the trainer continue to exist after the removal.
	if _, err := userRepo.FindByID(ctx, client.ID); err != nil {
		t.Fatalf("user must still exist after removal: %v", err)
	}
	if _, err := trainerRepo.FindByID(ctx, trainer.ID); err != nil {
		t.Fatalf("trainer must still exist after removal: %v", err)
	}

	// 11. The removed relationship is preserved with its original identity.
	preserved, err := clientRepo.FindIncludingDeletedByTrainerAndUser(ctx, trainer.ID, client.ID)
	if err != nil {
		t.Fatalf("find including deleted: %v", err)
	}
	if preserved.ID != relation.ID {
		t.Fatalf("row preservation: expected relation %q, got %q", relation.ID, preserved.ID)
	}
	if preserved.DeletedAt.Valid == false {
		t.Fatal("row preservation: expected a populated deleted_at")
	}

	// 12. Reactivation restores the exact same row.
	if err := clientRepo.Reactivate(ctx, trainer.ID, client.ID); err != nil {
		t.Fatalf("reactivate relationship: %v", err)
	}
	restored, err := clientRepo.FindActiveByTrainerAndUser(ctx, trainer.ID, client.ID)
	if err != nil {
		t.Fatalf("find after reactivation: %v", err)
	}
	if restored.ID != relation.ID {
		t.Fatalf("reactivation must restore the same row: expected %q, got %q", relation.ID, restored.ID)
	}

	// 13. Reactivation never creates a second active relationship.
	active, totalAfter, err := clientRepo.ListActiveClients(ctx, trainer.ID, 1, 20)
	if err != nil {
		t.Fatalf("list after reactivation: %v", err)
	}
	if totalAfter != 1 || len(active) != 1 {
		t.Fatalf("after reactivation: expected exactly one active relationship, got %d", totalAfter)
	}

	// 14. Reactivating an already-active relationship is rejected.
	if err := clientRepo.Reactivate(ctx, trainer.ID, client.ID); !errors.Is(err, repositories.ErrClientRelationNotFound) {
		t.Fatalf("reactivate active: expected ErrClientRelationNotFound, got %v", err)
	}

	// 15. Operations scoped to the wrong trainer can never touch the relation.
	if err := clientRepo.SoftDelete(ctx, otherTrainer.ID, client.ID); !errors.Is(err, repositories.ErrClientRelationNotFound) {
		t.Fatalf("soft delete with other trainer: expected ErrClientRelationNotFound, got %v", err)
	}
	if _, err := clientRepo.FindActiveByTrainerAndUser(ctx, otherTrainer.ID, client.ID); !errors.Is(err, repositories.ErrClientRelationNotFound) {
		t.Fatalf("find with other trainer: expected ErrClientRelationNotFound, got %v", err)
	}
}
