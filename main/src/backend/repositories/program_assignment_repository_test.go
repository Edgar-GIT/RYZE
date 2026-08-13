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

func TestProgramAssignmentRepository(t *testing.T) {
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
	clientRepo := repositories.NewTrainerClientRepository(tx)
	assignmentRepo := repositories.NewProgramAssignmentRepository(tx)
	ctx := context.Background()

	seedUser := func() *models.User {
		user := &models.User{
			Email:        fmt.Sprintf("program-assignment-repo-%d@ryze.local", time.Now().UnixNano()),
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
			Status:    models.ProgramStatusDraft,
		}
		if err := programRepo.Create(ctx, program); err != nil {
			t.Fatalf("create program: %v", err)
		}
		return program
	}

	seedRelation := func(trainerID, userID string) {
		if err := clientRepo.Create(ctx, &models.TrainerClient{TrainerID: trainerID, UserID: userID}); err != nil {
			t.Fatalf("create trainer-client relationship: %v", err)
		}
	}

	trainer := seedTrainer()
	otherTrainer := seedTrainer()
	client := seedUser()
	otherClient := seedUser()
	program := seedProgram(trainer.ID, "Strength Builder")
	foreignProgram := seedProgram(otherTrainer.ID, "Foreign Program")

	seedRelation(trainer.ID, client.ID)
	seedRelation(otherTrainer.ID, otherClient.ID)

	// 1. Create assigns one of the trainer's own programs to one of the
	// trainer's active clients.
	assignment := &models.ProgramAssignment{}
	if err := assignmentRepo.Create(ctx, trainer.ID, client.ID, program.ID, assignment); err != nil {
		t.Fatalf("create assignment: %v", err)
	}
	if assignment.ID == "" {
		t.Fatal("create assignment: expected generated UUID id")
	}
	if assignment.TrainerID != trainer.ID {
		t.Fatalf("create assignment: expected trainer %q, got %q", trainer.ID, assignment.TrainerID)
	}
	if assignment.ProgramID != program.ID {
		t.Fatalf("create assignment: expected program %q, got %q", program.ID, assignment.ProgramID)
	}
	if assignment.UserID != client.ID {
		t.Fatalf("create assignment: expected client %q, got %q", client.ID, assignment.UserID)
	}
	if assignment.CreatedAt.IsZero() || assignment.UpdatedAt.IsZero() {
		t.Fatal("create assignment: expected non-zero timestamps")
	}
	if assignment.Program.ID == "" || assignment.Program.Name != program.Name {
		t.Fatalf("create assignment: the assigned program must be embedded, got %+v", assignment.Program)
	}

	// 2. A client can never receive a second active program from the same
	// trainer: the one-active-assignment rule is enforced.
	secondProgram := seedProgram(trainer.ID, "HIIT Blaster")
	if err := assignmentRepo.Create(ctx, trainer.ID, client.ID, secondProgram.ID, &models.ProgramAssignment{}); !errors.Is(err, repositories.ErrAssignmentAlreadyActive) {
		t.Fatalf("duplicate assignment: expected ErrAssignmentAlreadyActive, got %v", err)
	}

	// 3. The same program is reusable: it can be assigned to another client of
	// the same trainer.
	client2 := seedUser()
	seedRelation(trainer.ID, client2.ID)
	if err := assignmentRepo.Create(ctx, trainer.ID, client2.ID, program.ID, &models.ProgramAssignment{}); err != nil {
		t.Fatalf("assign reusable program: %v", err)
	}

	// 4. An unknown user, a non-client user and a soft-deleted user are
	// indistinguishable and never assignable.
	if err := assignmentRepo.Create(ctx, trainer.ID, "00000000-0000-0000-0000-000000000000", program.ID, &models.ProgramAssignment{}); !errors.Is(err, repositories.ErrClientRelationNotFound) {
		t.Fatalf("unknown user: expected ErrClientRelationNotFound, got %v", err)
	}
	stranger := seedUser()
	if err := assignmentRepo.Create(ctx, trainer.ID, stranger.ID, program.ID, &models.ProgramAssignment{}); !errors.Is(err, repositories.ErrClientRelationNotFound) {
		t.Fatalf("non-client user: expected ErrClientRelationNotFound, got %v", err)
	}
	deletedUser := seedUser()
	seedRelation(trainer.ID, deletedUser.ID)
	if err := userRepo.SoftDelete(ctx, deletedUser.ID); err != nil {
		t.Fatalf("soft delete user: %v", err)
	}
	if err := assignmentRepo.Create(ctx, trainer.ID, deletedUser.ID, program.ID, &models.ProgramAssignment{}); !errors.Is(err, repositories.ErrClientRelationNotFound) {
		t.Fatalf("soft-deleted user: expected ErrClientRelationNotFound, got %v", err)
	}

	// 5. A soft-deleted trainer-client relationship blocks new assignments.
	removedClient := seedUser()
	seedRelation(trainer.ID, removedClient.ID)
	if err := clientRepo.SoftDelete(ctx, trainer.ID, removedClient.ID); err != nil {
		t.Fatalf("soft delete relationship: %v", err)
	}
	if err := assignmentRepo.Create(ctx, trainer.ID, removedClient.ID, program.ID, &models.ProgramAssignment{}); !errors.Is(err, repositories.ErrClientRelationNotFound) {
		t.Fatalf("removed client: expected ErrClientRelationNotFound, got %v", err)
	}

	// 6. A program that is foreign, unknown or soft-deleted can never be
	// assigned.
	if err := assignmentRepo.Create(ctx, trainer.ID, client2.ID, foreignProgram.ID, &models.ProgramAssignment{}); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("cross-trainer program: expected ErrProgramNotFound, got %v", err)
	}
	if err := assignmentRepo.Create(ctx, trainer.ID, client2.ID, "00000000-0000-0000-0000-000000000000", &models.ProgramAssignment{}); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("unknown program: expected ErrProgramNotFound, got %v", err)
	}
	deletedProgram := seedProgram(trainer.ID, "Soon Deleted")
	if err := programRepo.SoftDelete(ctx, trainer.ID, deletedProgram.ID); err != nil {
		t.Fatalf("soft delete program: %v", err)
	}
	if err := assignmentRepo.Create(ctx, trainer.ID, client2.ID, deletedProgram.ID, &models.ProgramAssignment{}); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("soft-deleted program: expected ErrProgramNotFound, got %v", err)
	}

	// 7. ListByClient returns the client's active assignments in assignment
	// order with the safe program data embedded.
	assignments, err := assignmentRepo.ListByClient(ctx, trainer.ID, client.ID)
	if err != nil {
		t.Fatalf("list client programs: %v", err)
	}
	if len(assignments) != 1 {
		t.Fatalf("list client programs: expected 1, got %d", len(assignments))
	}
	if assignments[0].ID != assignment.ID {
		t.Fatalf("list client programs: unexpected assignment %q", assignments[0].ID)
	}
	if assignments[0].Program.ID != program.ID || assignments[0].Program.Name != "Strength Builder" {
		t.Fatalf("list client programs: unexpected embedded program %+v", assignments[0].Program)
	}

	// 8. A foreign client, an unknown client and a removed client are all
	// indistinguishable from a client without assignments.
	for name, pair := range map[string][2]string{
		"foreign client":  {otherTrainer.ID, otherClient.ID},
		"unknown client":  {trainer.ID, "00000000-0000-0000-0000-000000000000"},
		"removed client":  {trainer.ID, removedClient.ID},
		"unrelated pair":  {otherTrainer.ID, client.ID},
	} {
		entries, err := assignmentRepo.ListByClient(ctx, pair[0], pair[1])
		if err != nil {
			t.Fatalf("list %s: %v", name, err)
		}
		if len(entries) != 0 {
			t.Fatalf("list %s must be empty, got %d", name, len(entries))
		}
	}

	// 9. FindByIDAndClient returns one of the trainer's own client assignments
	// scoped by the trainer, the client and the assignment id.
	found, err := assignmentRepo.FindByIDAndClient(ctx, trainer.ID, client.ID, assignment.ID)
	if err != nil {
		t.Fatalf("find assignment: %v", err)
	}
	if found.ID != assignment.ID || found.ProgramID != program.ID || found.UserID != client.ID {
		t.Fatalf("find assignment: unexpected assignment %+v", found)
	}
	if found.Program.ID != program.ID {
		t.Fatalf("find assignment: expected program embedded, got %+v", found.Program)
	}

	// 10. An assignment of another client or of a foreign trainer is
	// indistinguishable and never revealed.
	otherClientAssignment := &models.ProgramAssignment{}
	if err := assignmentRepo.Create(ctx, otherTrainer.ID, otherClient.ID, foreignProgram.ID, otherClientAssignment); err != nil {
		t.Fatalf("create other trainer assignment: %v", err)
	}
	if _, err := assignmentRepo.FindByIDAndClient(ctx, trainer.ID, otherClient.ID, otherClientAssignment.ID); !errors.Is(err, repositories.ErrAssignmentNotFound) {
		t.Fatalf("cross-client find: expected ErrAssignmentNotFound, got %v", err)
	}
	if _, err := assignmentRepo.FindByIDAndClient(ctx, otherTrainer.ID, otherClient.ID, assignment.ID); !errors.Is(err, repositories.ErrAssignmentNotFound) {
		t.Fatalf("cross-trainer find: expected ErrAssignmentNotFound, got %v", err)
	}
	if _, err := assignmentRepo.FindByIDAndClient(ctx, trainer.ID, client.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrAssignmentNotFound) {
		t.Fatalf("unknown find: expected ErrAssignmentNotFound, got %v", err)
	}

	// 11. SoftDelete removes the assignment from find and list but keeps the
	// row and never touches the program or the relationship.
	if err := assignmentRepo.SoftDelete(ctx, trainer.ID, client.ID, assignment.ID); err != nil {
		t.Fatalf("soft delete assignment: %v", err)
	}
	if _, err := assignmentRepo.FindByIDAndClient(ctx, trainer.ID, client.ID, assignment.ID); !errors.Is(err, repositories.ErrAssignmentNotFound) {
		t.Fatalf("soft-deleted assignment must not be found, got %v", err)
	}
	var deletedRecord models.ProgramAssignment
	if err := tx.Unscoped().First(&deletedRecord, "id = ?", assignment.ID).Error; err != nil {
		t.Fatalf("soft-deleted row must be preserved: %v", err)
	}
	if !deletedRecord.DeletedAt.Valid {
		t.Fatal("soft-deleted assignment must carry a deleted_at marker")
	}
	var programAfterDelete models.Program
	if err := tx.First(&programAfterDelete, "id = ?", program.ID).Error; err != nil {
		t.Fatalf("program must survive the assignment delete: %v", err)
	}
	if _, err := clientRepo.FindActiveByTrainerAndUser(ctx, trainer.ID, client.ID); err != nil {
		t.Fatalf("relationship must survive the assignment delete: %v", err)
	}

	// 12. A soft-deleted assignment can never be replaced while the parent
	// relation is active: the pair is simply without an active assignment.
	if err := assignmentRepo.Create(ctx, trainer.ID, client.ID, program.ID, &models.ProgramAssignment{}); err != nil {
		t.Fatalf("reassign after delete: %v", err)
	}

	// 13. Soft deleting a foreign or unknown assignment maps to not found and
	// never touches the target.
	if err := assignmentRepo.SoftDelete(ctx, otherTrainer.ID, otherClient.ID, assignment.ID); !errors.Is(err, repositories.ErrAssignmentNotFound) {
		t.Fatalf("cross-trainer soft delete: expected ErrAssignmentNotFound, got %v", err)
	}
	if err := assignmentRepo.SoftDelete(ctx, trainer.ID, otherClient.ID, assignment.ID); !errors.Is(err, repositories.ErrAssignmentNotFound) {
		t.Fatalf("cross-client soft delete: expected ErrAssignmentNotFound, got %v", err)
	}
	if err := assignmentRepo.SoftDelete(ctx, trainer.ID, client.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrAssignmentNotFound) {
		t.Fatalf("unknown soft delete: expected ErrAssignmentNotFound, got %v", err)
	}
	var untouched models.ProgramAssignment
	if err := tx.Unscoped().First(&untouched, "id = ?", assignment.ID).Error; err != nil {
		t.Fatalf("assignment row must survive foreign deletes: %v", err)
	}
	if !untouched.DeletedAt.Valid {
		t.Fatal("assignment row must stay soft-deleted after foreign delete attempts")
	}

	// 14. A program soft delete leaves its historical assignments listed with
	// their program data intact: existing assignments are never destroyed by a
	// program soft-delete.
	keptProgram := seedProgram(trainer.ID, "Kept History")
	keptClient := seedUser()
	seedRelation(trainer.ID, keptClient.ID)
	if err := assignmentRepo.Create(ctx, trainer.ID, keptClient.ID, keptProgram.ID, &models.ProgramAssignment{}); err != nil {
		t.Fatalf("create historical assignment: %v", err)
	}
	if err := programRepo.SoftDelete(ctx, trainer.ID, keptProgram.ID); err != nil {
		t.Fatalf("soft delete kept program: %v", err)
	}
	history, err := assignmentRepo.ListByClient(ctx, trainer.ID, keptClient.ID)
	if err != nil {
		t.Fatalf("list after program delete: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected the historical assignment to survive the program delete, got %d", len(history))
	}
	if history[0].Program.ID != keptProgram.ID || history[0].Program.Name != "Kept History" {
		t.Fatalf("historical assignment must keep its program data, got %+v", history[0].Program)
	}

	// 15. Removing the trainer-client relationship makes the client's
	// assignments inaccessible but never destroys them; reactivating the
	// relationship makes them visible again.
	lifecycleClient := seedUser()
	seedRelation(trainer.ID, lifecycleClient.ID)
	lifecycleAssignment := &models.ProgramAssignment{}
	if err := assignmentRepo.Create(ctx, trainer.ID, lifecycleClient.ID, program.ID, lifecycleAssignment); err != nil {
		t.Fatalf("create lifecycle assignment: %v", err)
	}
	if err := clientRepo.SoftDelete(ctx, trainer.ID, lifecycleClient.ID); err != nil {
		t.Fatalf("soft delete lifecycle relationship: %v", err)
	}
	entries, err := assignmentRepo.ListByClient(ctx, trainer.ID, lifecycleClient.ID)
	if err != nil {
		t.Fatalf("list after relationship delete: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("assignments must be inaccessible while the relationship is removed, got %d", len(entries))
	}
	var preservedRow models.ProgramAssignment
	if err := tx.Unscoped().First(&preservedRow, "id = ?", lifecycleAssignment.ID).Error; err != nil {
		t.Fatalf("assignment rows must survive the relationship soft delete: %v", err)
	}
	if err := clientRepo.Reactivate(ctx, trainer.ID, lifecycleClient.ID); err != nil {
		t.Fatalf("reactivate lifecycle relationship: %v", err)
	}
	entries, err = assignmentRepo.ListByClient(ctx, trainer.ID, lifecycleClient.ID)
	if err != nil {
		t.Fatalf("list after relationship reactivation: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != lifecycleAssignment.ID {
		t.Fatalf("expected the assignment visible again after reactivation, got %d entries", len(entries))
	}

	// 16. Cross-trainer isolation: other trainer's assignments were never
	// touched by any of the operations above.
	otherEntries, err := assignmentRepo.ListByClient(ctx, otherTrainer.ID, otherClient.ID)
	if err != nil {
		t.Fatalf("list other trainer client: %v", err)
	}
	if len(otherEntries) != 1 || otherEntries[0].ID != otherClientAssignment.ID {
		t.Fatalf("other trainer assignments must be untouched, got %d", len(otherEntries))
	}
}
