package repositories_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

func TestStatisticsRepository(t *testing.T) {
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
	statsRepo := repositories.NewStatisticsRepository(tx)
	ctx := context.Background()

	seedUser := func() *models.User {
		user := &models.User{
			Email:        fmt.Sprintf("stats-repo-%d@ryze.local", time.Now().UnixNano()),
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

	assign := func(trainerID, userID, programID string) *models.ProgramAssignment {
		assignment := &models.ProgramAssignment{}
		if err := assignmentRepo.Create(ctx, trainerID, userID, programID, assignment); err != nil {
			t.Fatalf("create assignment: %v", err)
		}
		return assignment
	}

	// 1. Client with no assignment returns empty stats.
	noAssignmentClient := seedUser()
	stats, err := statsRepo.GetClientStats(ctx, noAssignmentClient.ID)
	if err != nil {
		t.Fatalf("get stats no assignment: %v", err)
	}
	if stats.HasActiveAssignment {
		t.Fatal("no assignment: expected HasActiveAssignment=false")
	}
	if stats.TotalExecutions != 0 {
		t.Fatalf("no assignment: expected TotalExecutions=0, got %d", stats.TotalExecutions)
	}
	if stats.UniqueWorkoutsCompleted != 0 {
		t.Fatalf("no assignment: expected UniqueWorkoutsCompleted=0, got %d", stats.UniqueWorkoutsCompleted)
	}
	if stats.TotalWorkoutsInProgram != 0 {
		t.Fatalf("no assignment: expected TotalWorkoutsInProgram=0, got %d", stats.TotalWorkoutsInProgram)
	}
	if stats.LastWorkoutDate != nil {
		t.Fatalf("no assignment: expected nil LastWorkoutDate, got %v", stats.LastWorkoutDate)
	}

	// 2. Client with assignment but no history returns assignment info and zeros.
	trainer := seedTrainer()
	client := seedUser()
	program := seedProgram(trainer.ID, "Strength Builder")
	week := seedWeek(trainer.ID, program.ID)
	workout := seedWorkout(trainer.ID, program.ID, week.ID)
	_ = workout
	seedRelation(trainer.ID, client.ID)
	assign(trainer.ID, client.ID, program.ID)

	stats, err = statsRepo.GetClientStats(ctx, client.ID)
	if err != nil {
		t.Fatalf("get stats with assignment: %v", err)
	}
	if !stats.HasActiveAssignment {
		t.Fatal("with assignment: expected HasActiveAssignment=true")
	}
	if stats.CurrentProgramName != "Strength Builder" {
		t.Fatalf("with assignment: expected program name 'Strength Builder', got %q", stats.CurrentProgramName)
	}
	if stats.TotalWorkoutsInProgram != 1 {
		t.Fatalf("with assignment: expected TotalWorkoutsInProgram=1, got %d", stats.TotalWorkoutsInProgram)
	}
	if stats.TotalExecutions != 0 {
		t.Fatalf("with assignment: expected TotalExecutions=0, got %d", stats.TotalExecutions)
	}

	// 3. Complete one workout: executions=1, unique=1.
	if err := historyRepo.Create(ctx, client.ID, workout.ID, &models.WorkoutHistory{}); err != nil {
		t.Fatalf("complete workout: %v", err)
	}
	stats, err = statsRepo.GetClientStats(ctx, client.ID)
	if err != nil {
		t.Fatalf("get stats after completion: %v", err)
	}
	if stats.TotalExecutions != 1 {
		t.Fatalf("after completion: expected TotalExecutions=1, got %d", stats.TotalExecutions)
	}
	if stats.UniqueWorkoutsCompleted != 1 {
		t.Fatalf("after completion: expected UniqueWorkoutsCompleted=1, got %d", stats.UniqueWorkoutsCompleted)
	}
	if stats.LastWorkoutDate == nil {
		t.Fatal("after completion: expected non-nil LastWorkoutDate")
	}

	// 4. Complete the same workout again: executions=2, unique=1.
	if err := historyRepo.Create(ctx, client.ID, workout.ID, &models.WorkoutHistory{}); err != nil {
		t.Fatalf("repeat completion: %v", err)
	}
	stats, err = statsRepo.GetClientStats(ctx, client.ID)
	if err != nil {
		t.Fatalf("get stats after repeat: %v", err)
	}
	if stats.TotalExecutions != 2 {
		t.Fatalf("after repeat: expected TotalExecutions=2, got %d", stats.TotalExecutions)
	}
	if stats.UniqueWorkoutsCompleted != 1 {
		t.Fatalf("after repeat: expected UniqueWorkoutsCompleted=1, got %d", stats.UniqueWorkoutsCompleted)
	}

	// 5. Add a second workout to the program, complete it: unique=2.
	week2 := seedWeek(trainer.ID, program.ID)
	workout2 := seedWorkout(trainer.ID, program.ID, week2.ID)
	if err := historyRepo.Create(ctx, client.ID, workout2.ID, &models.WorkoutHistory{}); err != nil {
		t.Fatalf("complete second workout: %v", err)
	}
	stats, err = statsRepo.GetClientStats(ctx, client.ID)
	if err != nil {
		t.Fatalf("get stats after second workout: %v", err)
	}
	if stats.TotalWorkoutsInProgram != 2 {
		t.Fatalf("after second workout: expected TotalWorkoutsInProgram=2, got %d", stats.TotalWorkoutsInProgram)
	}
	if stats.TotalExecutions != 3 {
		t.Fatalf("after second workout: expected TotalExecutions=3, got %d", stats.TotalExecutions)
	}
	if stats.UniqueWorkoutsCompleted != 2 {
		t.Fatalf("after second workout: expected UniqueWorkoutsCompleted=2, got %d", stats.UniqueWorkoutsCompleted)
	}

	// 6. Soft-deleting a workout reduces the program count but preserves
	// completed workout history (the unique count stays the same).
	if err := workoutRepo.SoftDelete(ctx, trainer.ID, program.ID, week2.ID, workout2.ID); err != nil {
		t.Fatalf("soft delete workout: %v", err)
	}
	stats, err = statsRepo.GetClientStats(ctx, client.ID)
	if err != nil {
		t.Fatalf("get stats after workout deletion: %v", err)
	}
	if stats.TotalWorkoutsInProgram != 1 {
		t.Fatalf("after workout deletion: expected TotalWorkoutsInProgram=1, got %d", stats.TotalWorkoutsInProgram)
	}
	// Unique completed stays 2 — history references are preserved.
	if stats.UniqueWorkoutsCompleted != 2 {
		t.Fatalf("after workout deletion: expected UniqueWorkoutsCompleted=2, got %d", stats.UniqueWorkoutsCompleted)
	}

	// 7. Other client's stats are isolated.
	otherTrainer := seedTrainer()
	otherClient := seedUser()
	otherProgram := seedProgram(otherTrainer.ID, "Foreign Program")
	otherWeek := seedWeek(otherTrainer.ID, otherProgram.ID)
	otherWorkout := seedWorkout(otherTrainer.ID, otherProgram.ID, otherWeek.ID)
	seedRelation(otherTrainer.ID, otherClient.ID)
	assign(otherTrainer.ID, otherClient.ID, otherProgram.ID)
	if err := historyRepo.Create(ctx, otherClient.ID, otherWorkout.ID, &models.WorkoutHistory{}); err != nil {
		t.Fatalf("other client completion: %v", err)
	}

	stats, err = statsRepo.GetClientStats(ctx, client.ID)
	if err != nil {
		t.Fatalf("get stats isolation check: %v", err)
	}
	if stats.TotalExecutions != 3 {
		t.Fatalf("isolation: expected TotalExecutions=3, got %d", stats.TotalExecutions)
	}
	if stats.UniqueWorkoutsCompleted != 2 {
		t.Fatalf("isolation: expected UniqueWorkoutsCompleted=2, got %d", stats.UniqueWorkoutsCompleted)
	}
}
