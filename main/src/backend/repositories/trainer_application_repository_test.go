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
	decisionRepo := repositories.NewTrainerApplicationDecisionRepository(tx)
	ctx := context.Background()

	// The development database may already contain committed applications, so
	// every count assertion below is expressed as a delta against a baseline
	// captured before this test seeds its own rows.
	_, baselineTotal, err := applicationRepo.List(ctx, 1, 20, "")
	if err != nil {
		t.Fatalf("baseline list applications: %v", err)
	}
	_, baselinePending, err := applicationRepo.List(ctx, 1, 20, models.ApplicationStatusPending)
	if err != nil {
		t.Fatalf("baseline list pending applications: %v", err)
	}
	_, baselineApproved, err := applicationRepo.List(ctx, 1, 20, models.ApplicationStatusApproved)
	if err != nil {
		t.Fatalf("baseline list approved applications: %v", err)
	}

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
	if total != baselineTotal+1 || !containsApplication(all, application.ID) {
		t.Fatalf("list applications: expected the created application to be listed, got total=%d baseline=%d", total, baselineTotal)
	}
	pending, pendingTotal, err := applicationRepo.List(ctx, 1, 20, models.ApplicationStatusPending)
	if err != nil {
		t.Fatalf("list pending applications: %v", err)
	}
	if pendingTotal != baselinePending+1 || !containsApplication(pending, application.ID) {
		t.Fatalf("list pending: expected the created application to be listed, got total=%d baseline=%d", pendingTotal, baselinePending)
	}
	_, approvedTotal, err := applicationRepo.List(ctx, 1, 20, models.ApplicationStatusApproved)
	if err != nil {
		t.Fatalf("list approved applications: %v", err)
	}
	if approvedTotal != baselineApproved {
		t.Fatalf("list approved: expected no additional approved applications, got total=%d baseline=%d", approvedTotal, baselineApproved)
	}

	// 6. Reject the PENDING application; it stays in history and the review
	// decision is recorded in the same transaction.
	decision := &models.TrainerApplicationDecision{
		AdminID:    "ADMIN_2",
		FromStatus: models.ApplicationStatusPending,
		ToStatus:   models.ApplicationStatusRejected,
		Reason:     "Documents are not readable.",
	}
	if err := applicationRepo.Reject(ctx, application.ID, decision); err != nil {
		t.Fatalf("reject application: %v", err)
	}
	rejected, err := applicationRepo.FindByID(ctx, application.ID)
	if err != nil {
		t.Fatalf("find rejected application: %v", err)
	}
	if rejected.Status != models.ApplicationStatusRejected {
		t.Fatalf("reject: expected REJECTED, got %q", rejected.Status)
	}
	decisions, err := decisionRepo.ListByApplication(ctx, application.ID)
	if err != nil {
		t.Fatalf("list decisions after reject: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("reject: expected exactly one recorded decision, got %d", len(decisions))
	}
	recorded := decisions[0]
	if recorded.ApplicationID != application.ID || recorded.AdminID != "ADMIN_2" ||
		recorded.FromStatus != models.ApplicationStatusPending || recorded.ToStatus != models.ApplicationStatusRejected ||
		recorded.Reason != "Documents are not readable." || recorded.ID == "" {
		t.Fatalf("reject: unexpected recorded decision %+v", recorded)
	}
	if _, err := applicationRepo.FindActiveByUserID(ctx, user.ID); !errors.Is(err, repositories.ErrApplicationNotFound) {
		t.Fatalf("find active after reject: expected ErrApplicationNotFound, got %v", err)
	}

	// 7. A rejected user can apply again with a new application. The latest
	// application (including a rejected one) is visible so the applicant can see
	// the outcome of their last submission.
	latest, err := applicationRepo.FindLatestByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("find latest application after reject: %v", err)
	}
	if latest.ID != application.ID || latest.Status != models.ApplicationStatusRejected {
		t.Fatalf("find latest after reject: expected rejected application %q, got %q (%s)", application.ID, latest.ID, latest.Status)
	}
	reapplication := seedApplication(user.ID)
	latest, err = applicationRepo.FindLatestByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("find latest application after reapply: %v", err)
	}
	if latest.ID != reapplication.ID || latest.Status != models.ApplicationStatusPending {
		t.Fatalf("find latest after reapply: expected pending application %q, got %q (%s)", reapplication.ID, latest.ID, latest.Status)
	}

	// 8. Approve atomically creates the trainer profile for the same user and
	// records the review decision in the same transaction.
	approvalDecision := &models.TrainerApplicationDecision{
		AdminID:    "ADMIN_2",
		FromStatus: models.ApplicationStatusPending,
		ToStatus:   models.ApplicationStatusApproved,
	}
	approvedApplication, err := applicationRepo.Approve(ctx, reapplication.ID, approvalDecision)
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
	approvalDecisions, err := decisionRepo.ListByApplication(ctx, reapplication.ID)
	if err != nil {
		t.Fatalf("list decisions after approve: %v", err)
	}
	if len(approvalDecisions) != 1 || approvalDecisions[0].ToStatus != models.ApplicationStatusApproved || approvalDecisions[0].AdminID != "ADMIN_2" {
		t.Fatalf("approve: unexpected recorded decisions %+v", approvalDecisions)
	}
	active, err = applicationRepo.FindActiveByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("find active after approve: %v", err)
	}
	if active.Status != models.ApplicationStatusApproved {
		t.Fatalf("find active after approve: expected APPROVED, got %q", active.Status)
	}

	// 9. Approving an already-approved application is a state conflict and
	// records no additional decision.
	if _, err := applicationRepo.Approve(ctx, reapplication.ID, nil); !errors.Is(err, repositories.ErrApplicationStateConflict) {
		t.Fatalf("approve approved application: expected ErrApplicationStateConflict, got %v", err)
	}
	approvalDecisions, err = decisionRepo.ListByApplication(ctx, reapplication.ID)
	if err != nil {
		t.Fatalf("list decisions after conflict: %v", err)
	}
	if len(approvalDecisions) != 1 {
		t.Fatalf("failed approve must not record a decision, got %d", len(approvalDecisions))
	}

	// 10. Approving a rejected application is a state conflict.
	other := seedUser()
	otherApplication := seedApplication(other.ID)
	if err := applicationRepo.Reject(ctx, otherApplication.ID, nil); err != nil {
		t.Fatalf("reject second application: %v", err)
	}
	if _, err := applicationRepo.Approve(ctx, otherApplication.ID, nil); !errors.Is(err, repositories.ErrApplicationStateConflict) {
		t.Fatalf("approve rejected application: expected ErrApplicationStateConflict, got %v", err)
	}
	otherDecisions, err := decisionRepo.ListByApplication(ctx, otherApplication.ID)
	if err != nil {
		t.Fatalf("list decisions for second application: %v", err)
	}
	if len(otherDecisions) != 0 {
		t.Fatalf("reject without a decision must not record one, got %d", len(otherDecisions))
	}

	// 11. Approving a nonexistent application is reported as not found.
	if _, err := applicationRepo.Approve(ctx, "00000000-0000-0000-0000-000000000000", nil); !errors.Is(err, repositories.ErrApplicationNotFound) {
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
	if _, err := applicationRepo.Approve(ctx, thirdApplication.ID, nil); !errors.Is(err, repositories.ErrTrainerAlreadyLinked) {
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
	if err := applicationRepo.Reject(ctx, reapplication.ID, nil); !errors.Is(err, repositories.ErrApplicationStateConflict) {
		t.Fatalf("reject approved application: expected ErrApplicationStateConflict, got %v", err)
	}
	if err := applicationRepo.Reject(ctx, otherApplication.ID, nil); !errors.Is(err, repositories.ErrApplicationStateConflict) {
		t.Fatalf("reject rejected application: expected ErrApplicationStateConflict, got %v", err)
	}

	// 14. Rejecting a nonexistent application is reported as not found.
	if err := applicationRepo.Reject(ctx, "00000000-0000-0000-0000-000000000000", nil); !errors.Is(err, repositories.ErrApplicationNotFound) {
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

	// 16. A user with no applications has no latest application.
	fifth := seedUser()
	if _, err := applicationRepo.FindLatestByUserID(ctx, fifth.ID); !errors.Is(err, repositories.ErrApplicationNotFound) {
		t.Fatalf("find latest without application: expected ErrApplicationNotFound, got %v", err)
	}
}

// containsApplication reports whether the listed applications include the
// given id.
func containsApplication(applications []models.TrainerApplication, id string) bool {
	for i := range applications {
		if applications[i].ID == id {
			return true
		}
	}
	return false
}
