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

func TestProgramWorkoutRepository(t *testing.T) {
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
	ctx := context.Background()

	seedUser := func() *models.User {
		user := &models.User{
			Email:        fmt.Sprintf("workout-repo-%d@ryze.local", time.Now().UnixNano()),
			PasswordHash: "prepared-hash-outside-repository-scope",
			FirstName:    "John",
			LastName:     "Doe",
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

	trainer := seedTrainer()
	otherTrainer := seedTrainer()
	program := seedProgram(trainer.ID, "Strength Builder")
	foreignProgram := seedProgram(otherTrainer.ID, "Foreign Program")
	week := seedWeek(trainer.ID, program.ID)
	foreignWeek := seedWeek(otherTrainer.ID, foreignProgram.ID)

	// 1. Create appends workouts in increasing position order.
	workout1 := seedWorkout(trainer.ID, program.ID, week.ID)
	if workout1.ID == "" {
		t.Fatal("create workout: expected generated UUID id")
	}
	if workout1.Position != 1 {
		t.Fatalf("create workout: expected position 1, got %d", workout1.Position)
	}
	if workout1.ProgramWeekID != week.ID {
		t.Fatalf("create workout: expected week %q, got %q", week.ID, workout1.ProgramWeekID)
	}
	if workout1.CreatedAt.IsZero() || workout1.UpdatedAt.IsZero() {
		t.Fatal("create workout: expected non-zero timestamps")
	}
	workout2 := seedWorkout(trainer.ID, program.ID, week.ID)
	if workout2.Position != 2 {
		t.Fatalf("create workout: expected position 2, got %d", workout2.Position)
	}
	workout3 := seedWorkout(trainer.ID, program.ID, week.ID)
	if workout3.Position != 3 {
		t.Fatalf("create workout: expected position 3, got %d", workout3.Position)
	}

	// 2. Creating a workout on a foreign week is denied.
	foreign := &models.ProgramWorkout{}
	if err := workoutRepo.Create(ctx, otherTrainer.ID, program.ID, week.ID, foreign); !errors.Is(err, repositories.ErrWeekNotFound) {
		t.Fatalf("cross-trainer create: expected ErrWeekNotFound, got %v", err)
	}
	if err := workoutRepo.Create(ctx, trainer.ID, foreignProgram.ID, week.ID, foreign); !errors.Is(err, repositories.ErrWeekNotFound) {
		t.Fatalf("create on foreign week: expected ErrWeekNotFound, got %v", err)
	}
	if err := workoutRepo.Create(ctx, trainer.ID, program.ID, foreignWeek.ID, foreign); !errors.Is(err, repositories.ErrWeekNotFound) {
		t.Fatalf("create on foreign week: expected ErrWeekNotFound, got %v", err)
	}
	if err := workoutRepo.Create(ctx, trainer.ID, program.ID, "00000000-0000-0000-0000-000000000000", foreign); !errors.Is(err, repositories.ErrWeekNotFound) {
		t.Fatalf("create on unknown week: expected ErrWeekNotFound, got %v", err)
	}

	// 3. List returns only the week's workouts in position order.
	workouts, err := workoutRepo.ListByWeek(ctx, trainer.ID, program.ID, week.ID)
	if err != nil {
		t.Fatalf("list workouts: %v", err)
	}
	if len(workouts) != 3 {
		t.Fatalf("list workouts: expected 3 workouts, got %d", len(workouts))
	}
	if workouts[0].ID != workout1.ID || workouts[0].Position != 1 {
		t.Fatalf("list workouts: expected workout1 first, got %+v", workouts[0])
	}
	if workouts[1].ID != workout2.ID || workouts[2].ID != workout3.ID {
		t.Fatalf("list workouts: expected creation order, got %q, %q, %q", workouts[0].ID, workouts[1].ID, workouts[2].ID)
	}

	// 4. A foreign or unknown week is indistinguishable from one without
	// workouts.
	workouts, err = workoutRepo.ListByWeek(ctx, otherTrainer.ID, program.ID, week.ID)
	if err != nil {
		t.Fatalf("list workouts cross-trainer: %v", err)
	}
	if len(workouts) != 0 {
		t.Fatalf("cross-trainer list must be empty, got %d", len(workouts))
	}
	workouts, err = workoutRepo.ListByWeek(ctx, trainer.ID, program.ID, "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("list workouts unknown week: %v", err)
	}
	if len(workouts) != 0 {
		t.Fatalf("unknown week list must be empty, got %d", len(workouts))
	}

	// 5. Find returns one of the trainer's own workouts.
	found, err := workoutRepo.FindByIDAndWeek(ctx, trainer.ID, program.ID, week.ID, workout2.ID)
	if err != nil {
		t.Fatalf("find workout: %v", err)
	}
	if found.ID != workout2.ID || found.Position != 2 || found.ProgramWeekID != week.ID {
		t.Fatalf("unexpected workout %+v", found)
	}

	// 6. Owner isolation: a foreign, wrong-week or unknown workout is
	// indistinguishable.
	if _, err := workoutRepo.FindByIDAndWeek(ctx, otherTrainer.ID, program.ID, week.ID, workout2.ID); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("cross-trainer find: expected ErrWorkoutNotFound, got %v", err)
	}
	if _, err := workoutRepo.FindByIDAndWeek(ctx, trainer.ID, program.ID, foreignWeek.ID, workout2.ID); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("wrong-week find: expected ErrWorkoutNotFound, got %v", err)
	}
	if _, err := workoutRepo.FindByIDAndWeek(ctx, trainer.ID, program.ID, week.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("unknown workout: expected ErrWorkoutNotFound, got %v", err)
	}

	// 7. Reorder reassigns positions to the given order.
	if err := workoutRepo.Reorder(ctx, trainer.ID, program.ID, week.ID, []string{workout3.ID, workout1.ID, workout2.ID}); err != nil {
		t.Fatalf("reorder workouts: %v", err)
	}
	workouts, err = workoutRepo.ListByWeek(ctx, trainer.ID, program.ID, week.ID)
	if err != nil {
		t.Fatalf("list workouts after reorder: %v", err)
	}
	if workouts[0].ID != workout3.ID || workouts[0].Position != 1 {
		t.Fatalf("reorder: expected workout3 first, got %+v", workouts[0])
	}
	if workouts[1].ID != workout1.ID || workouts[2].ID != workout2.ID {
		t.Fatalf("reorder: expected order workout3, workout1, workout2, got %q, %q, %q", workouts[0].ID, workouts[1].ID, workouts[2].ID)
	}

	// 8. Reorder with a wrong list is rejected without touching the order.
	for name, ids := range map[string][]string{
		"missing":      {workout1.ID, workout2.ID},
		"extra":        {workout1.ID, workout2.ID, workout3.ID, "00000000-0000-0000-0000-000000000000"},
		"duplicate":    {workout1.ID, workout1.ID, workout3.ID},
		"empty":        {},
		"unknown only": {"00000000-0000-0000-0000-000000000000", "00000000-0000-0000-0000-000000000001"},
	} {
		if err := workoutRepo.Reorder(ctx, trainer.ID, program.ID, week.ID, ids); !errors.Is(err, repositories.ErrWorkoutReorderConflict) {
			t.Fatalf("reorder %s: expected ErrWorkoutReorderConflict, got %v", name, err)
		}
	}
	workouts, err = workoutRepo.ListByWeek(ctx, trainer.ID, program.ID, week.ID)
	if err != nil {
		t.Fatalf("list workouts after rejected reorder: %v", err)
	}
	if workouts[0].ID != workout3.ID || workouts[1].ID != workout1.ID || workouts[2].ID != workout2.ID {
		t.Fatalf("rejected reorder must never change the order, got %+v", workouts)
	}

	// 9. Reordering a foreign week is denied before any write.
	if err := workoutRepo.Reorder(ctx, otherTrainer.ID, program.ID, week.ID, []string{workout3.ID, workout1.ID, workout2.ID}); !errors.Is(err, repositories.ErrWeekNotFound) {
		t.Fatalf("cross-trainer reorder: expected ErrWeekNotFound, got %v", err)
	}

	// 10. Soft delete removes the workout from find and list but keeps the row.
	if err := workoutRepo.SoftDelete(ctx, trainer.ID, program.ID, week.ID, workout2.ID); err != nil {
		t.Fatalf("soft delete workout: %v", err)
	}
	if _, err := workoutRepo.FindByIDAndWeek(ctx, trainer.ID, program.ID, week.ID, workout2.ID); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("soft-deleted workout must not be found, got %v", err)
	}
	var deletedRecord models.ProgramWorkout
	if err := tx.Unscoped().First(&deletedRecord, "id = ?", workout2.ID).Error; err != nil {
		t.Fatalf("soft-deleted workout row must be preserved: %v", err)
	}
	if !deletedRecord.DeletedAt.Valid {
		t.Fatal("soft-deleted workout must carry a deleted_at marker")
	}
	workouts, err = workoutRepo.ListByWeek(ctx, trainer.ID, program.ID, week.ID)
	if err != nil {
		t.Fatalf("list workouts after delete: %v", err)
	}
	if len(workouts) != 2 {
		t.Fatalf("expected 2 active workouts after delete, got %d", len(workouts))
	}
	for _, w := range workouts {
		if w.ID == workout2.ID {
			t.Fatal("soft-deleted workout must never appear in the list")
		}
	}

	// 11. Soft deleting a foreign or unknown workout maps to not found.
	if err := workoutRepo.SoftDelete(ctx, otherTrainer.ID, program.ID, week.ID, workout1.ID); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("cross-trainer soft delete: expected ErrWorkoutNotFound, got %v", err)
	}
	if err := workoutRepo.SoftDelete(ctx, trainer.ID, program.ID, week.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("unknown soft delete: expected ErrWorkoutNotFound, got %v", err)
	}

	// 12. After a soft delete, new workouts reuse the next available position
	// of the remaining active workouts.
	workout4 := seedWorkout(trainer.ID, program.ID, week.ID)
	if workout4.Position != 3 {
		t.Fatalf("expected position 3 after delete, got %d", workout4.Position)
	}

	// 13. Reorder after a soft delete still requires the exact active set.
	if err := workoutRepo.Reorder(ctx, trainer.ID, program.ID, week.ID, []string{workout1.ID, workout3.ID, workout4.ID}); err != nil {
		t.Fatalf("reorder after delete: %v", err)
	}
	workouts, err = workoutRepo.ListByWeek(ctx, trainer.ID, program.ID, week.ID)
	if err != nil {
		t.Fatalf("list workouts after post-delete reorder: %v", err)
	}
	if len(workouts) != 3 || workouts[0].ID != workout1.ID || workouts[1].ID != workout3.ID || workouts[2].ID != workout4.ID {
		t.Fatalf("expected order workout1, workout3, workout4, got %+v", workouts)
	}

	// 14. A week soft delete leaves its workouts preserved but unreachable.
	if err := weekRepo.SoftDelete(ctx, trainer.ID, program.ID, week.ID); err != nil {
		t.Fatalf("soft delete week: %v", err)
	}
	workouts, err = workoutRepo.ListByWeek(ctx, trainer.ID, program.ID, week.ID)
	if err != nil {
		t.Fatalf("list workouts after week delete: %v", err)
	}
	if len(workouts) != 0 {
		t.Fatalf("workouts of a deleted week must be unreachable, got %d", len(workouts))
	}
	var preserved models.ProgramWorkout
	if err := tx.Unscoped().First(&preserved, "id = ?", workout1.ID).Error; err != nil {
		t.Fatalf("workout rows must survive the week soft delete: %v", err)
	}
	if preserved.DeletedAt.Valid {
		t.Fatal("workouts must not be soft-deleted with their week")
	}
}
