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

func TestTrainerApplicationRepository(t *testing.T) {
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
	applicationRepo := repositories.NewTrainerApplicationRepository(tx)
	ctx := context.Background()

	seedUser := func() *models.User {
		user := &models.User{
			Email:        fmt.Sprintf("application-repo-%d@ryze.local", time.Now().UnixNano()),
			PasswordHash: "prepared-hash-outside-repository-scope",
			FirstName:    "John",
			LastName:     "Doe",
		}
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("create user: %v", err)
		}
		return user
	}

	seedApplication := func(userID string) *models.TrainerApplication {
		application := &models.TrainerApplication{
			UserID: userID,
			Status: models.ApplicationStatusPending,
		}
		if err := applicationRepo.Create(ctx, application); err != nil {
			t.Fatalf("create application: %v", err)
		}
		return application
	}

	// 1. Create a PENDING application for a user.
	user := seedUser()
	application := seedApplication(user.ID)
	if application.ID == "" {
		t.Fatal("create application: expected generated UUID id")
	}

	// 2. A second active application for the same user is rejected.
	duplicate := &models.TrainerApplication{UserID: user.ID, Status: models.ApplicationStatusPending}
	if err := applicationRepo.Create(ctx, duplicate); !errors.Is(err, repositories.ErrApplicationAlreadyActive) {
		t.Fatalf("duplicate active application: expected ErrApplicationAlreadyActive, got %v", err)
	}

	// 3. Find the active application by user id.
	active, err := applicationRepo.FindActiveByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("find active application by user id: %v", err)
	}
	if active.ID != application.ID {
		t.Fatalf("find active by user id: expected application %q, got %q", application.ID, active.ID)
	}

	// 4. Find by id preloads the linked user.
	byID, err := applicationRepo.FindByID(ctx, application.ID)
	if err != nil {
		t.Fatalf("find application by id: %v", err)
	}
	if byID.UserID != user.ID {
		t.Fatalf("find by id: expected user %q, got %q", user.ID, byID.UserID)
	}
	if byID.User.ID != user.ID {
		t.Fatalf("find by id: expected the linked user to be preloaded, got %+v", byID.User)
	}

	// 5. List returns the application and the total, and the status filter works.
	all, total, err := applicationRepo.List(ctx, 1, 20, "")
	if err != nil {
		t.Fatalf("list applications: %v", err)
	}
	if total != 1 || len(all) != 1 || all[0].ID != application.ID {
		t.Fatalf("list applications: expected 1 application, got total=%d len=%d", total, len(all))
	}
	pending, pendingTotal, err := applicationRepo.List(ctx, 1, 20, models.ApplicationStatusPending)
	if err != nil {
		t.Fatalf("list pending applications: %v", err)
	}
	if pendingTotal != 1 || len(pending) != 1 {
		t.Fatalf("list pending: expected 1, got total=%d len=%d", pendingTotal, len(pending))
	}
	approved, approvedTotal, err := applicationRepo.List(ctx, 1, 20, models.ApplicationStatusApproved)
	if err != nil {
		t.Fatalf("list approved applications: %v", err)
	}
	if approvedTotal != 0 || len(approved) != 0 {
		t.Fatalf("list approved: expected 0, got total=%d len=%d", approvedTotal, len(approved))
	}

	// 6. Reject the PENDING application; it stays in history.
	if err := applicationRepo.Reject(ctx, application.ID); err != nil {
		t.Fatalf("reject application: %v", err)
	}
	rejected, err := applicationRepo.FindByID(ctx, application.ID)
	if err != nil {
		t.Fatalf("find rejected application: %v", err)
	}
	if rejected.Status != models.ApplicationStatusRejected {
		t.Fatalf("reject: expected REJECTED, got %q", rejected.Status)
	}
	if _, err := applicationRepo.FindActiveByUserID(ctx, user.ID); !errors.Is(err, repositories.ErrApplicationNotFound) {
		t.Fatalf("find active after reject: expected ErrApplicationNotFound, got %v", err)
	}

	// 7. A rejected user can apply again with a new application.
	reapplication := seedApplication(user.ID)

	// 8. Approve atomically creates the trainer profile for the same user.
	approvedApplication, err := applicationRepo.Approve(ctx, reapplication.ID)
	if err != nil {
		t.Fatalf("approve application: %v", err)
	}
	if approvedApplication.Status != models.ApplicationStatusApproved {
		t.Fatalf("approve: expected APPROVED, got %q", approvedApplication.Status)
	}
	trainer, err := trainerRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("expected trainer after approval: %v", err)
	}
	if trainer.UserID != user.ID {
		t.Fatalf("approve: expected trainer linked to user %q, got %q", user.ID, trainer.UserID)
	}
	active, err = applicationRepo.FindActiveByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("find active after approve: %v", err)
	}
	if active.Status != models.ApplicationStatusApproved {
		t.Fatalf("find active after approve: expected APPROVED, got %q", active.Status)
	}

	// 9. Approving an already-approved application is a state conflict.
	if _, err := applicationRepo.Approve(ctx, reapplication.ID); !errors.Is(err, repositories.ErrApplicationStateConflict) {
		t.Fatalf("approve approved application: expected ErrApplicationStateConflict, got %v", err)
	}

	// 10. Approving a rejected application is a state conflict.
	other := seedUser()
	otherApplication := seedApplication(other.ID)
	if err := applicationRepo.Reject(ctx, otherApplication.ID); err != nil {
		t.Fatalf("reject second application: %v", err)
	}
	if _, err := applicationRepo.Approve(ctx, otherApplication.ID); !errors.Is(err, repositories.ErrApplicationStateConflict) {
		t.Fatalf("approve rejected application: expected ErrApplicationStateConflict, got %v", err)
	}

	// 11. Approving a nonexistent application is reported as not found.
	if _, err := applicationRepo.Approve(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrApplicationNotFound) {
		t.Fatalf("approve unknown application: expected ErrApplicationNotFound, got %v", err)
	}

	// 12. Approving a user who already owns a trainer profile fails atomically:
	// no duplicate trainer is created and the application stays PENDING.
	third := seedUser()
	if err := trainerRepo.Create(ctx, &models.Trainer{UserID: third.ID}); err != nil {
		t.Fatalf("create trainer for third user: %v", err)
	}
	beforeApprove, beforeTotal, err := trainerRepo.ListActive(ctx, 1, 20)
	if err != nil {
		t.Fatalf("list trainers before failed approve: %v", err)
	}
	thirdApplication := seedApplication(third.ID)
	if _, err := applicationRepo.Approve(ctx, thirdApplication.ID); !errors.Is(err, repositories.ErrTrainerAlreadyLinked) {
		t.Fatalf("approve user with trainer: expected ErrTrainerAlreadyLinked, got %v", err)
	}
	persisted, err := applicationRepo.FindByID(ctx, thirdApplication.ID)
	if err != nil {
		t.Fatalf("find after failed approve: %v", err)
	}
	if persisted.Status != models.ApplicationStatusPending {
		t.Fatalf("failed approve must leave the application PENDING, got %q", persisted.Status)
	}
	trainers, trainerTotal, err := trainerRepo.ListActive(ctx, 1, 20)
	if err != nil {
		t.Fatalf("list trainers after failed approve: %v", err)
	}
	if trainerTotal != beforeTotal || len(trainers) != len(beforeApprove) {
		t.Fatalf("failed approve must not create a trainer, got total=%d before=%d", trainerTotal, beforeTotal)
	}

	// 13. Rejecting a non-pending application is a state conflict.
	if err := applicationRepo.Reject(ctx, reapplication.ID); !errors.Is(err, repositories.ErrApplicationStateConflict) {
		t.Fatalf("reject approved application: expected ErrApplicationStateConflict, got %v", err)
	}
	if err := applicationRepo.Reject(ctx, otherApplication.ID); !errors.Is(err, repositories.ErrApplicationStateConflict) {
		t.Fatalf("reject rejected application: expected ErrApplicationStateConflict, got %v", err)
	}

	// 14. Rejecting a nonexistent application is reported as not found.
	if err := applicationRepo.Reject(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrApplicationNotFound) {
		t.Fatalf("reject unknown application: expected ErrApplicationNotFound, got %v", err)
	}

	// 15. A soft-deleted application is invisible to regular operations and
	// frees the user link.
	fourth := seedUser()
	fourthApplication := seedApplication(fourth.ID)
	if err := tx.Delete(&models.TrainerApplication{}, "id = ?", fourthApplication.ID).Error; err != nil {
		t.Fatalf("soft delete application: %v", err)
	}
	if _, err := applicationRepo.FindByID(ctx, fourthApplication.ID); !errors.Is(err, repositories.ErrApplicationNotFound) {
		t.Fatalf("find after soft delete: expected ErrApplicationNotFound, got %v", err)
	}
	if _, err := applicationRepo.FindActiveByUserID(ctx, fourth.ID); !errors.Is(err, repositories.ErrApplicationNotFound) {
		t.Fatalf("find active after soft delete: expected ErrApplicationNotFound, got %v", err)
	}
	if err := applicationRepo.Create(ctx, &models.TrainerApplication{UserID: fourth.ID, Status: models.ApplicationStatusPending}); err != nil {
		t.Fatalf("create application after soft delete: %v", err)
	}
}
