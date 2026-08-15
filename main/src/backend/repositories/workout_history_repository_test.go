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

func TestWorkoutHistoryRepository(t *testing.T) {
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
	weekRepo := repositories.NewProgramWeekRepository(tx)
	workoutRepo := repositories.NewProgramWorkoutRepository(tx)
	clientRepo := repositories.NewTrainerClientRepository(tx)
	assignmentRepo := repositories.NewProgramAssignmentRepository(tx)
	historyRepo := repositories.NewWorkoutHistoryRepository(tx)
	ctx := context.Background()

	seedUser := func() *models.User {
		user := &models.User{
			Email:        fmt.Sprintf("workout-history-repo-%d@ryze.local", time.Now().UnixNano()),
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

	seedWeek := func(trainerID, programID string) *models.ProgramWeek {
		week := &models.ProgramWeek{}
		if err := weekRepo.Create(ctx, trainerID, programID, week); err != nil {
			t.Fatalf("create week: %v", err)
		}
		return week
	}

	seedWorkout := func(trainerID, programID, weekID string) *models.ProgramWorkout {
		workout := &models.ProgramWorkout{}
		if err := workoutRepo.Create(ctx, trainerID, programID, weekID, workout); err != nil {
			t.Fatalf("create workout: %v", err)
		}
		return workout
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
	week := seedWeek(trainer.ID, program.ID)
	foreignWeek := seedWeek(otherTrainer.ID, foreignProgram.ID)
	workout := seedWorkout(trainer.ID, program.ID, week.ID)
	otherWorkout := seedWorkout(trainer.ID, program.ID, week.ID)
	foreignWorkout := seedWorkout(otherTrainer.ID, foreignProgram.ID, foreignWeek.ID)

	seedRelation(trainer.ID, client.ID)
	seedRelation(otherTrainer.ID, otherClient.ID)

	assign := func(trainerID, userID, programID string) *models.ProgramAssignment {
		assignment := &models.ProgramAssignment{}
		if err := assignmentRepo.Create(ctx, trainerID, userID, programID, assignment); err != nil {
			t.Fatalf("create assignment: %v", err)
		}
		return assignment
	}
	assignment := assign(trainer.ID, client.ID, program.ID)

	// 1. Create records a completion of one workout of the assigned program.
	entry := &models.WorkoutHistory{}
	if err := historyRepo.Create(ctx, client.ID, workout.ID, entry); err != nil {
		t.Fatalf("create history entry: %v", err)
	}
	if entry.ID == "" {
		t.Fatal("create history entry: expected generated UUID id")
	}
	if entry.UserID != client.ID {
		t.Fatalf("create history entry: expected user %q, got %q", client.ID, entry.UserID)
	}
	if entry.ProgramWorkoutID != workout.ID {
		t.Fatalf("create history entry: expected workout %q, got %q", workout.ID, entry.ProgramWorkoutID)
	}
	if entry.CompletedAt.IsZero() {
		t.Fatal("create history entry: expected a completion timestamp")
	}
	if entry.CreatedAt.IsZero() || entry.UpdatedAt.IsZero() {
		t.Fatal("create history entry: expected non-zero timestamps")
	}

	// 2. A workout may be completed more than once: the history is append-only.
	if err := historyRepo.Create(ctx, client.ID, workout.ID, &models.WorkoutHistory{}); err != nil {
		t.Fatalf("repeated completion: %v", err)
	}

	// 3. A user without an active assignment can never complete a workout, even
	// one that exists.
	if err := historyRepo.Create(ctx, otherClient.ID, workout.ID, &models.WorkoutHistory{}); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("no-assignment completion: expected ErrWorkoutNotFound, got %v", err)
	}

	// 4. A workout that does not belong to the assigned program is rejected even
	// when it exists: it is indistinguishable from an unknown one.
	if err := historyRepo.Create(ctx, client.ID, foreignWorkout.ID, &models.WorkoutHistory{}); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("foreign-workout completion: expected ErrWorkoutNotFound, got %v", err)
	}
	if err := historyRepo.Create(ctx, client.ID, "00000000-0000-0000-0000-000000000000", &models.WorkoutHistory{}); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("unknown-workout completion: expected ErrWorkoutNotFound, got %v", err)
	}

	// 5. A soft-deleted workout, week or program breaks the chain and blocks
	// completion without revealing the reason.
	deletedWorkout := seedWorkout(trainer.ID, program.ID, week.ID)
	if err := workoutRepo.SoftDelete(ctx, trainer.ID, program.ID, week.ID, deletedWorkout.ID); err != nil {
		t.Fatalf("soft delete workout: %v", err)
	}
	if err := historyRepo.Create(ctx, client.ID, deletedWorkout.ID, &models.WorkoutHistory{}); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("soft-deleted-workout completion: expected ErrWorkoutNotFound, got %v", err)
	}
	deletedWeek := seedWeek(trainer.ID, program.ID)
	deletedWeekWorkout := seedWorkout(trainer.ID, program.ID, deletedWeek.ID)
	if err := weekRepo.SoftDelete(ctx, trainer.ID, program.ID, deletedWeek.ID); err != nil {
		t.Fatalf("soft delete week: %v", err)
	}
	if err := historyRepo.Create(ctx, client.ID, deletedWeekWorkout.ID, &models.WorkoutHistory{}); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("soft-deleted-week completion: expected ErrWorkoutNotFound, got %v", err)
	}
	if err := historyRepo.Create(ctx, client.ID, workout.ID, &models.WorkoutHistory{}); err != nil {
		t.Fatalf("completion after unrelated week deletion: %v", err)
	}

	// 6. A soft-deleted program is no longer executable even though its
	// historical assignment row is preserved.
	if err := assignmentRepo.SoftDelete(ctx, trainer.ID, client.ID, assignment.ID); err != nil {
		t.Fatalf("soft delete assignment: %v", err)
	}
	deletedProgram := seedProgram(trainer.ID, "Retired Program")
	deletedProgramWeek := seedWeek(trainer.ID, deletedProgram.ID)
	deletedProgramWorkout := seedWorkout(trainer.ID, deletedProgram.ID, deletedProgramWeek.ID)
	deletedProgramAssignment := assign(trainer.ID, client.ID, deletedProgram.ID)
	if err := programRepo.SoftDelete(ctx, trainer.ID, deletedProgram.ID); err != nil {
		t.Fatalf("soft delete program: %v", err)
	}
	if err := historyRepo.Create(ctx, client.ID, deletedProgramWorkout.ID, &models.WorkoutHistory{}); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("soft-deleted-program completion: expected ErrWorkoutNotFound, got %v", err)
	}
	// The newest assignment points to the soft-deleted program, so the previous
	// program is no longer executable either (the most recent assignment wins).
	if err := historyRepo.Create(ctx, client.ID, workout.ID, &models.WorkoutHistory{}); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("completion after newer deleted-program assignment: expected ErrWorkoutNotFound, got %v", err)
	}

	// 7. A soft-deleted assignment revokes the grant for its program, and a new
	// assignment restores it for its program.
	if err := assignmentRepo.SoftDelete(ctx, trainer.ID, client.ID, deletedProgramAssignment.ID); err != nil {
		t.Fatalf("soft delete assignment: %v", err)
	}
	recent := assign(trainer.ID, client.ID, program.ID)
	if err := historyRepo.Create(ctx, client.ID, otherWorkout.ID, &models.WorkoutHistory{}); err != nil {
		t.Fatalf("completion after reassignment: %v", err)
	}
	if err := assignmentRepo.SoftDelete(ctx, trainer.ID, client.ID, recent.ID); err != nil {
		t.Fatalf("soft delete recent assignment: %v", err)
	}
	if err := historyRepo.Create(ctx, client.ID, otherWorkout.ID, &models.WorkoutHistory{}); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("soft-deleted-assignment completion: expected ErrWorkoutNotFound, got %v", err)
	}

	// 8. ListByUser returns only the user's own entries, newest first, with the
	// other user's history never leaking.
	if err := assignmentRepo.Create(ctx, otherTrainer.ID, otherClient.ID, foreignProgram.ID, &models.ProgramAssignment{}); err != nil {
		t.Fatalf("assign foreign program to other client: %v", err)
	}
	if err := historyRepo.Create(ctx, otherClient.ID, foreignWorkout.ID, &models.WorkoutHistory{}); err != nil {
		t.Fatalf("other client completion: %v", err)
	}

	entries, total, err := historyRepo.ListByUser(ctx, client.ID, 1, 10)
	if err != nil {
		t.Fatalf("list history: %v", err)
	}
	if total != 4 {
		t.Fatalf("list history: expected 4 own entries, got %d", total)
	}
	if len(entries) != 4 {
		t.Fatalf("list history: expected 4 entries, got %d", len(entries))
	}
	for i := 1; i < len(entries); i++ {
		if entries[i].CompletedAt.After(entries[i-1].CompletedAt) {
			t.Fatalf("list history: expected newest-first order, got entry %d newer than entry %d", i, i-1)
		}
	}

	// 9. Pagination slices the user's own history.
	page2, _, err := historyRepo.ListByUser(ctx, client.ID, 2, 2)
	if err != nil {
		t.Fatalf("list history page 2: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("list history page 2: expected 2 entries, got %d", len(page2))
	}
}
