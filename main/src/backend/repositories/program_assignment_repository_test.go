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
		"foreign client": {otherTrainer.ID, otherClient.ID},
		"unknown client": {trainer.ID, "00000000-0000-0000-0000-000000000000"},
		"removed client": {trainer.ID, removedClient.ID},
		"unrelated pair": {otherTrainer.ID, client.ID},
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

// TestProgramAssignmentRepositoryFindAssignedProgram exercises the client-side
// read surface: the user's most recent active assignment resolves the assigned
// program with its full active structure preloaded in display order, and every
// missing or soft-deleted component maps to an indistinguishable not-found
// outcome.
func TestProgramAssignmentRepositoryFindAssignedProgram(t *testing.T) {
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
	weekRepo := repositories.NewProgramWeekRepository(tx)
	workoutRepo := repositories.NewProgramWorkoutRepository(tx)
	workoutExerciseRepo := repositories.NewWorkoutExerciseRepository(tx)
	ctx := context.Background()

	seedUser := func() *models.User {
		user := &models.User{
			Email:        fmt.Sprintf("assigned-program-repo-%d@ryze.local", time.Now().UnixNano()),
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

	seedExercise := func(name string) *models.Exercise {
		exercise := &models.Exercise{
			Name:          name,
			Description:   "A catalog exercise",
			TargetMuscles: "Chest",
			Equipment:     "Barbell",
			Difficulty:    "Intermediate",
		}
		if err := tx.Create(exercise).Error; err != nil {
			t.Fatalf("create exercise: %v", err)
		}
		return exercise
	}

	seedWeek := func(trainerID, programID string) *models.ProgramWeek {
		week := &models.ProgramWeek{}
		if err := weekRepo.Create(ctx, trainerID, programID, week); err != nil {
			t.Fatalf("create program week: %v", err)
		}
		return week
	}

	seedWorkout := func(trainerID, programID, weekID string) *models.ProgramWorkout {
		workout := &models.ProgramWorkout{}
		if err := workoutRepo.Create(ctx, trainerID, programID, weekID, workout); err != nil {
			t.Fatalf("create program workout: %v", err)
		}
		return workout
	}

	seedWorkoutExercise := func(trainerID, programID, weekID, workoutID string, exercise *models.Exercise) *models.WorkoutExercise {
		workoutExercise := &models.WorkoutExercise{ExerciseID: exercise.ID}
		if err := workoutExerciseRepo.AddExercise(ctx, trainerID, programID, weekID, workoutID, workoutExercise); err != nil {
			t.Fatalf("add workout exercise: %v", err)
		}
		return workoutExercise
	}

	trainer := seedTrainer()
	client := seedUser()
	seedRelation(trainer.ID, client.ID)
	program := seedProgram(trainer.ID, "Strength Builder")

	week1 := seedWeek(trainer.ID, program.ID)
	week2 := seedWeek(trainer.ID, program.ID)
	workout1 := seedWorkout(trainer.ID, program.ID, week1.ID)
	workout2 := seedWorkout(trainer.ID, program.ID, week2.ID)
	exercise := seedExercise("Bench Press")
	workoutExercise := seedWorkoutExercise(trainer.ID, program.ID, week1.ID, workout1.ID, exercise)

	assignment := &models.ProgramAssignment{}
	if err := assignmentRepo.Create(ctx, trainer.ID, client.ID, program.ID, assignment); err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	// 1. The assigned program is returned with its full active structure.
	assigned, err := assignmentRepo.FindAssignedProgram(ctx, client.ID)
	if err != nil {
		t.Fatalf("find assigned program: %v", err)
	}
	if assigned.ID != program.ID || assigned.Name != "Strength Builder" {
		t.Fatalf("expected the assigned program, got %+v", assigned)
	}
	if len(assigned.Weeks) != 2 {
		t.Fatalf("expected 2 weeks, got %d", len(assigned.Weeks))
	}
	if assigned.Weeks[0].ID != week1.ID || assigned.Weeks[0].WeekNumber != 1 {
		t.Fatalf("expected week 1 first, got %+v", assigned.Weeks[0])
	}
	if assigned.Weeks[1].ID != week2.ID || assigned.Weeks[1].WeekNumber != 2 {
		t.Fatalf("expected week 2 second, got %+v", assigned.Weeks[1])
	}
	if len(assigned.Weeks[0].Workouts) != 1 || assigned.Weeks[0].Workouts[0].ID != workout1.ID {
		t.Fatalf("expected workout 1 inside week 1, got %+v", assigned.Weeks[0].Workouts)
	}
	if len(assigned.Weeks[1].Workouts) != 1 || assigned.Weeks[1].Workouts[0].ID != workout2.ID {
		t.Fatalf("expected workout 2 inside week 2, got %+v", assigned.Weeks[1].Workouts)
	}
	if len(assigned.Weeks[0].Workouts[0].Exercises) != 1 || assigned.Weeks[0].Workouts[0].Exercises[0].ID != workoutExercise.ID {
		t.Fatalf("expected the workout exercise inside workout 1, got %+v", assigned.Weeks[0].Workouts[0].Exercises)
	}
	loaded := assigned.Weeks[0].Workouts[0].Exercises[0].Exercise
	if loaded == nil || loaded.ID != exercise.ID || loaded.Name != "Bench Press" {
		t.Fatalf("expected the catalog exercise embedded, got %+v", loaded)
	}

	// 2. A user without an active assignment is indistinguishable from a user
	// without any assignment.
	stranger := seedUser()
	if _, err := assignmentRepo.FindAssignedProgram(ctx, stranger.ID); !errors.Is(err, repositories.ErrAssignmentNotFound) {
		t.Fatalf("no assignment: expected ErrAssignmentNotFound, got %v", err)
	}
	if _, err := assignmentRepo.FindAssignedProgram(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrAssignmentNotFound) {
		t.Fatalf("unknown user: expected ErrAssignmentNotFound, got %v", err)
	}

	// 3. A soft-deleted assignment removes the program from the read surface.
	deletedClient := seedUser()
	seedRelation(trainer.ID, deletedClient.ID)
	deletedAssignment := &models.ProgramAssignment{}
	if err := assignmentRepo.Create(ctx, trainer.ID, deletedClient.ID, program.ID, deletedAssignment); err != nil {
		t.Fatalf("create deletable assignment: %v", err)
	}
	if err := assignmentRepo.SoftDelete(ctx, trainer.ID, deletedClient.ID, deletedAssignment.ID); err != nil {
		t.Fatalf("soft delete assignment: %v", err)
	}
	if _, err := assignmentRepo.FindAssignedProgram(ctx, deletedClient.ID); !errors.Is(err, repositories.ErrAssignmentNotFound) {
		t.Fatalf("soft-deleted assignment: expected ErrAssignmentNotFound, got %v", err)
	}

	// 4. A soft-deleted program makes the assignment unreadable.
	deletedProgramClient := seedUser()
	seedRelation(trainer.ID, deletedProgramClient.ID)
	deletedProgram := seedProgram(trainer.ID, "Soon Deleted")
	if err := assignmentRepo.Create(ctx, trainer.ID, deletedProgramClient.ID, deletedProgram.ID, &models.ProgramAssignment{}); err != nil {
		t.Fatalf("create deleted-program assignment: %v", err)
	}
	if err := programRepo.SoftDelete(ctx, trainer.ID, deletedProgram.ID); err != nil {
		t.Fatalf("soft delete program: %v", err)
	}
	if _, err := assignmentRepo.FindAssignedProgram(ctx, deletedProgramClient.ID); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("soft-deleted program: expected ErrProgramNotFound, got %v", err)
	}

	// 5. Soft-deleted structure entries are excluded from the read surface.
	prunedClient := seedUser()
	seedRelation(trainer.ID, prunedClient.ID)
	prunedProgram := seedProgram(trainer.ID, "Pruned")
	prunedWeek := seedWeek(trainer.ID, prunedProgram.ID)
	prunedWorkout := seedWorkout(trainer.ID, prunedProgram.ID, prunedWeek.ID)
	if err := workoutExerciseRepo.AddExercise(ctx, trainer.ID, prunedProgram.ID, prunedWeek.ID, prunedWorkout.ID, &models.WorkoutExercise{ExerciseID: exercise.ID}); err != nil {
		t.Fatalf("add pruned workout exercise: %v", err)
	}
	if err := weekRepo.SoftDelete(ctx, trainer.ID, prunedProgram.ID, prunedWeek.ID); err != nil {
		t.Fatalf("soft delete week: %v", err)
	}
	if err := assignmentRepo.Create(ctx, trainer.ID, prunedClient.ID, prunedProgram.ID, &models.ProgramAssignment{}); err != nil {
		t.Fatalf("create pruned assignment: %v", err)
	}
	pruned, err := assignmentRepo.FindAssignedProgram(ctx, prunedClient.ID)
	if err != nil {
		t.Fatalf("find pruned program: %v", err)
	}
	if len(pruned.Weeks) != 0 {
		t.Fatalf("a soft-deleted week must be excluded, got %d weeks", len(pruned.Weeks))
	}
	if err := workoutExerciseRepo.SoftDelete(ctx, trainer.ID, program.ID, week1.ID, workout1.ID, workoutExercise.ID); err != nil {
		t.Fatalf("soft delete workout exercise: %v", err)
	}
	if err := workoutRepo.SoftDelete(ctx, trainer.ID, program.ID, week1.ID, workout1.ID); err != nil {
		t.Fatalf("soft delete workout: %v", err)
	}
	afterPrune, err := assignmentRepo.FindAssignedProgram(ctx, client.ID)
	if err != nil {
		t.Fatalf("find after prune: %v", err)
	}
	if len(afterPrune.Weeks[0].Workouts) != 0 {
		t.Fatalf("a soft-deleted workout must be excluded, got %d workouts", len(afterPrune.Weeks[0].Workouts))
	}

	// 6. A user with several active assignments (one per trainer) always reads
	// the most recently created one.
	trainer2 := seedTrainer()
	otherProgram := seedProgram(trainer2.ID, "HIIT Blaster")
	seedRelation(trainer2.ID, client.ID)
	time.Sleep(2 * time.Millisecond)
	if err := assignmentRepo.Create(ctx, trainer2.ID, client.ID, otherProgram.ID, &models.ProgramAssignment{}); err != nil {
		t.Fatalf("create second assignment: %v", err)
	}
	latest, err := assignmentRepo.FindAssignedProgram(ctx, client.ID)
	if err != nil {
		t.Fatalf("find latest assignment: %v", err)
	}
	if latest.ID != otherProgram.ID || latest.Name != "HIIT Blaster" {
		t.Fatalf("expected the most recent assignment to win, got %+v", latest)
	}
}
