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

func TestTrainerRepository(t *testing.T) {
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
	ctx := context.Background()

	// The development database already holds committed trainers, and the test
	// transaction sees them, so list assertions compare deltas instead of
	// absolute totals.
	_, beforeTotal, err := trainerRepo.ListActive(ctx, 1, 20)
	if err != nil {
		t.Fatalf("list active trainers before: %v", err)
	}
	_, beforeDeletedTotal, err := trainerRepo.ListDeleted(ctx, 1, 20)
	if err != nil {
		t.Fatalf("list deleted trainers before: %v", err)
	}

	seedUser := func() *models.User {
		user := &models.User{
			Email:        fmt.Sprintf("trainer-repo-%d@ryze.local", time.Now().UnixNano()),
			PasswordHash: "prepared-hash-outside-repository-scope",
			FirstName:    "John",
			LastName:     "Doe",
		}
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("create user: %v", err)
		}
		return user
	}

	// 1. Create a trainer for a user.
	user := seedUser()
	trainer := &models.Trainer{UserID: user.ID}
	if err := trainerRepo.Create(ctx, trainer); err != nil {
		t.Fatalf("create trainer: %v", err)
	}
	if trainer.ID == "" {
		t.Fatal("create trainer: expected generated UUID id")
	}

	// 2. A second active trainer for the same user is rejected.
	duplicate := &models.Trainer{UserID: user.ID}
	if err := trainerRepo.Create(ctx, duplicate); !errors.Is(err, repositories.ErrTrainerAlreadyLinked) {
		t.Fatalf("duplicate active trainer: expected ErrTrainerAlreadyLinked, got %v", err)
	}

	// 3. Find the trainer by ID.
	byID, err := trainerRepo.FindByID(ctx, trainer.ID)
	if err != nil {
		t.Fatalf("find trainer by id: %v", err)
	}
	if byID.UserID != user.ID {
		t.Fatalf("find by id: expected user %q, got %q", user.ID, byID.UserID)
	}

	// 4. Find the trainer by user id.
	byUser, err := trainerRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("find trainer by user id: %v", err)
	}
	if byUser.ID != trainer.ID {
		t.Fatalf("find by user id: expected trainer %q, got %q", trainer.ID, byUser.ID)
	}

	// 5. A second user can hold its own trainer profile.
	other := seedUser()
	otherTrainer := &models.Trainer{UserID: other.ID}
	if err := trainerRepo.Create(ctx, otherTrainer); err != nil {
		t.Fatalf("create second trainer: %v", err)
	}

	// 5b. FindByIDAndUserID matches only the trainer owned by the given user.
	matched, err := trainerRepo.FindByIDAndUserID(ctx, trainer.ID, user.ID)
	if err != nil {
		t.Fatalf("find trainer by id and user: %v", err)
	}
	if matched.ID != trainer.ID || matched.UserID != user.ID {
		t.Fatalf("find by id and user: expected trainer %q for user %q, got %+v", trainer.ID, user.ID, matched)
	}
	if matched.User.ID != user.ID {
		t.Fatalf("find by id and user: expected linked user %q, got %q", user.ID, matched.User.ID)
	}

	// 5c. The same trainer id queried with a different owning user is never
	// returned.
	if _, err := trainerRepo.FindByIDAndUserID(ctx, trainer.ID, other.ID); !errors.Is(err, repositories.ErrTrainerNotFound) {
		t.Fatalf("find by id and user with other user: expected ErrTrainerNotFound, got %v", err)
	}
	if _, err := trainerRepo.FindByIDAndUserID(ctx, "00000000-0000-0000-0000-000000000000", user.ID); !errors.Is(err, repositories.ErrTrainerNotFound) {
		t.Fatalf("find by id and user with unknown id: expected ErrTrainerNotFound, got %v", err)
	}

	// 6. List active trainers reports the two new trainers in the total.
	_, total, err := trainerRepo.ListActive(ctx, 1, 20)
	if err != nil {
		t.Fatalf("list active trainers: %v", err)
	}
	if total != beforeTotal+2 {
		t.Fatalf("list active: expected total %d, got %d", beforeTotal+2, total)
	}

	// 7. Soft delete the trainer.
	if err := trainerRepo.SoftDelete(ctx, trainer.ID); err != nil {
		t.Fatalf("soft delete trainer: %v", err)
	}

	// 8. Deleted trainer is no longer returned by normal lookups.
	if _, err := trainerRepo.FindByID(ctx, trainer.ID); !errors.Is(err, repositories.ErrTrainerNotFound) {
		t.Fatalf("find by id after delete: expected ErrTrainerNotFound, got %v", err)
	}
	if _, err := trainerRepo.FindByUserID(ctx, user.ID); !errors.Is(err, repositories.ErrTrainerNotFound) {
		t.Fatalf("find by user id after delete: expected ErrTrainerNotFound, got %v", err)
	}

	// 9. The row still exists with deleted_at populated.
	var raw models.Trainer
	if err := tx.Unscoped().Where("id = ?", trainer.ID).First(&raw).Error; err != nil {
		t.Fatalf("row should still exist after soft delete: %v", err)
	}
	if !raw.DeletedAt.Valid {
		t.Fatal("expected deleted_at to be populated on the soft-deleted row")
	}

	// 10. The soft-deleted trainer is found when deleted rows are included.
	deleted, err := trainerRepo.FindByIDIncludingDeleted(ctx, trainer.ID)
	if err != nil {
		t.Fatalf("find trainer including deleted: %v", err)
	}
	if deleted.ID != trainer.ID {
		t.Fatalf("find including deleted: expected id %q, got %q", trainer.ID, deleted.ID)
	}
	if !deleted.DeletedAt.Valid {
		t.Fatal("find including deleted: expected deleted_at to be populated")
	}

	// 11. A new trainer profile can be created for the same user after the
	// soft delete frees the user link.
	replacement := &models.Trainer{UserID: user.ID}
	if err := trainerRepo.Create(ctx, replacement); err != nil {
		t.Fatalf("create replacement trainer: %v", err)
	}

	// 12. Reactivating the old trainer while the replacement is active is
	// rejected as a duplicate link.
	if err := trainerRepo.Reactivate(ctx, trainer.ID); !errors.Is(err, repositories.ErrTrainerAlreadyLinked) {
		t.Fatalf("reactivate while replacement active: expected ErrTrainerAlreadyLinked, got %v", err)
	}

	// 13. Soft-delete the replacement and reactivate the original trainer.
	if err := trainerRepo.SoftDelete(ctx, replacement.ID); err != nil {
		t.Fatalf("soft delete replacement trainer: %v", err)
	}
	if err := trainerRepo.Reactivate(ctx, trainer.ID); err != nil {
		t.Fatalf("reactivate trainer: %v", err)
	}

	// 14. The same row is restored and returned by normal lookups again.
	restored, err := trainerRepo.FindByID(ctx, trainer.ID)
	if err != nil {
		t.Fatalf("find reactivated trainer by id: %v", err)
	}
	if restored.ID != trainer.ID {
		t.Fatalf("reactivate: expected id %q, got %q", trainer.ID, restored.ID)
	}
	if restored.UserID != user.ID {
		t.Fatalf("reactivate: expected user %q, got %q", user.ID, restored.UserID)
	}
	if restored.DeletedAt.Valid {
		t.Fatal("reactivate: deleted_at must be cleared")
	}
	if !restored.CreatedAt.Equal(trainer.CreatedAt) {
		t.Fatal("reactivate: created_at must be preserved")
	}

	// 15. Reactivating an already-active trainer is rejected.
	if err := trainerRepo.Reactivate(ctx, trainer.ID); !errors.Is(err, repositories.ErrTrainerNotFound) {
		t.Fatalf("reactivate active trainer: expected ErrTrainerNotFound, got %v", err)
	}

	// 16. After soft-deleting the replacement and reactivating the original,
	// the list deltas reflect exactly one active trainer and one deleted row
	// created by this test.
	_, total, err = trainerRepo.ListActive(ctx, 1, 20)
	if err != nil {
		t.Fatalf("list active after reactivation: %v", err)
	}
	if total != beforeTotal+2 {
		t.Fatalf("list active after reactivation: expected total %d, got %d", beforeTotal+2, total)
	}
	_, deletedTotal, err := trainerRepo.ListDeleted(ctx, 1, 20)
	if err != nil {
		t.Fatalf("list deleted trainers: %v", err)
	}
	if deletedTotal != beforeDeletedTotal+1 {
		t.Fatalf("list deleted: expected total %d, got %d", beforeDeletedTotal+1, deletedTotal)
	}
	deletedReplacement, err := trainerRepo.FindByIDIncludingDeleted(ctx, replacement.ID)
	if err != nil {
		t.Fatalf("find deleted replacement: %v", err)
	}
	if !deletedReplacement.DeletedAt.Valid {
		t.Fatal("list deleted: expected the replacement trainer to be soft-deleted")
	}

	// 17. Deleting a non-existent trainer is reported as not found.
	if err := trainerRepo.SoftDelete(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrTrainerNotFound) {
		t.Fatalf("soft delete unknown trainer: expected ErrTrainerNotFound, got %v", err)
	}
}
