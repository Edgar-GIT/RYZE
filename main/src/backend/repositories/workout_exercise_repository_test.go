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

func TestWorkoutExerciseRepository(t *testing.T) {
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
	workoutExerciseRepo := repositories.NewWorkoutExerciseRepository(tx)
	ctx := context.Background()

	seedUser := func() *models.User {
		user := &models.User{
			Email:        fmt.Sprintf("workout-exercise-repo-%d@ryze.local", time.Now().UnixNano()),
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

	seedExercise := func(name string) *models.Exercise {
		exercise := &models.Exercise{Name: name}
		if err := tx.Create(exercise).Error; err != nil {
			t.Fatalf("seed exercise: %v", err)
		}
		return exercise
	}

	trainer := seedTrainer()
	otherTrainer := seedTrainer()
	program := seedProgram(trainer.ID, "Strength Builder")
	foreignProgram := seedProgram(otherTrainer.ID, "Foreign Program")
	week := seedWeek(trainer.ID, program.ID)
	foreignWeek := seedWeek(otherTrainer.ID, foreignProgram.ID)
	workout := seedWorkout(trainer.ID, program.ID, week.ID)
	otherWorkout := seedWorkout(trainer.ID, program.ID, week.ID)

	squat := seedExercise("Barbell Squat")
	deadlift := seedExercise("Deadlift")
	pushUp := seedExercise("Push-Up")

	// 1. AddExercise appends one active exercise to the end of the workout.
	entry1 := &models.WorkoutExercise{ExerciseID: squat.ID}
	if err := workoutExerciseRepo.AddExercise(ctx, trainer.ID, program.ID, week.ID, workout.ID, entry1); err != nil {
		t.Fatalf("add exercise: %v", err)
	}
	if entry1.ID == "" {
		t.Fatal("add exercise: expected generated UUID id")
	}
	if entry1.ProgramWorkoutID != workout.ID {
		t.Fatalf("add exercise: expected workout %q, got %q", workout.ID, entry1.ProgramWorkoutID)
	}
	if entry1.Position != 1 {
		t.Fatalf("add exercise: expected position 1, got %d", entry1.Position)
	}
	if entry1.CreatedAt.IsZero() || entry1.UpdatedAt.IsZero() {
		t.Fatal("add exercise: expected non-zero timestamps")
	}
	entry2 := &models.WorkoutExercise{ExerciseID: deadlift.ID}
	if err := workoutExerciseRepo.AddExercise(ctx, trainer.ID, program.ID, week.ID, workout.ID, entry2); err != nil {
		t.Fatalf("add second exercise: %v", err)
	}
	if entry2.Position != 2 {
		t.Fatalf("add exercise: expected position 2, got %d", entry2.Position)
	}

	// 2. The same exercise may be added more than once: position distinguishes
	// the two assignments (documented project decision).
	duplicate := &models.WorkoutExercise{ExerciseID: squat.ID}
	if err := workoutExerciseRepo.AddExercise(ctx, trainer.ID, program.ID, week.ID, workout.ID, duplicate); err != nil {
		t.Fatalf("add duplicated exercise: %v", err)
	}
	if duplicate.Position != 3 {
		t.Fatalf("add duplicated exercise: expected position 3, got %d", duplicate.Position)
	}

	// 3. A soft-deleted exercise of the global catalog can never be assigned.
	deletedExercise := seedExercise("Soon Deleted")
	if err := tx.Delete(&models.Exercise{}, "id = ?", deletedExercise.ID).Error; err != nil {
		t.Fatalf("soft delete exercise: %v", err)
	}
	blocked := &models.WorkoutExercise{ExerciseID: deletedExercise.ID}
	if err := workoutExerciseRepo.AddExercise(ctx, trainer.ID, program.ID, week.ID, workout.ID, blocked); !errors.Is(err, repositories.ErrExerciseNotFound) {
		t.Fatalf("add soft-deleted exercise: expected ErrExerciseNotFound, got %v", err)
	}
	if err := workoutExerciseRepo.AddExercise(ctx, trainer.ID, program.ID, week.ID, workout.ID, &models.WorkoutExercise{ExerciseID: "00000000-0000-0000-0000-000000000000"}); !errors.Is(err, repositories.ErrExerciseNotFound) {
		t.Fatalf("add unknown exercise: expected ErrExerciseNotFound, got %v", err)
	}

	// 4. The owning chain is verified on add: a foreign workout, a wrong
	// program, a wrong week or an unknown workout is always ErrWorkoutNotFound.
	if err := workoutExerciseRepo.AddExercise(ctx, otherTrainer.ID, program.ID, week.ID, workout.ID, &models.WorkoutExercise{ExerciseID: squat.ID}); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("cross-trainer add: expected ErrWorkoutNotFound, got %v", err)
	}
	if err := workoutExerciseRepo.AddExercise(ctx, trainer.ID, foreignProgram.ID, week.ID, workout.ID, &models.WorkoutExercise{ExerciseID: squat.ID}); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("wrong-program add: expected ErrWorkoutNotFound, got %v", err)
	}
	if err := workoutExerciseRepo.AddExercise(ctx, trainer.ID, program.ID, foreignWeek.ID, workout.ID, &models.WorkoutExercise{ExerciseID: squat.ID}); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("wrong-week add: expected ErrWorkoutNotFound, got %v", err)
	}
	if err := workoutExerciseRepo.AddExercise(ctx, trainer.ID, program.ID, week.ID, "00000000-0000-0000-0000-000000000000", &models.WorkoutExercise{ExerciseID: squat.ID}); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("unknown-workout add: expected ErrWorkoutNotFound, got %v", err)
	}

	// 5. ListByWorkout returns only the workout's active exercises, in position
	// order, with the safe catalog data of the assigned exercise.
	entries, err := workoutExerciseRepo.ListByWorkout(ctx, trainer.ID, program.ID, week.ID, workout.ID)
	if err != nil {
		t.Fatalf("list workout exercises: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("list workout exercises: expected 3, got %d", len(entries))
	}
	if entries[0].ID != entry1.ID || entries[0].Position != 1 {
		t.Fatalf("list workout exercises: expected entry1 first, got %+v", entries[0])
	}
	if entries[1].ID != entry2.ID || entries[2].ID != duplicate.ID {
		t.Fatalf("list workout exercises: expected add order, got %q, %q, %q", entries[0].ID, entries[1].ID, entries[2].ID)
	}
	for i := range entries {
		if entries[i].Exercise.ID == "" {
			t.Fatalf("list workout exercises: entry %d must embed its exercise", i)
		}
	}
	if entries[0].Exercise.ID != squat.ID || entries[0].Exercise.Name != "Barbell Squat" {
		t.Fatalf("list workout exercises: unexpected embedded exercise %+v", entries[0].Exercise)
	}
	if entries[2].Exercise.ID != squat.ID {
		t.Fatalf("list workout exercises: the duplicated exercise must embed its exercise too, got %q", entries[2].Exercise.ID)
	}

	// 6. A foreign or unknown workout is indistinguishable from one without
	// entries.
	entries, err = workoutExerciseRepo.ListByWorkout(ctx, otherTrainer.ID, program.ID, week.ID, workout.ID)
	if err != nil {
		t.Fatalf("cross-trainer list: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("cross-trainer list must be empty, got %d", len(entries))
	}
	entries, err = workoutExerciseRepo.ListByWorkout(ctx, trainer.ID, program.ID, week.ID, "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("unknown-workout list: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("unknown-workout list must be empty, got %d", len(entries))
	}

	// 7. FindByIDAndWorkout returns one of the trainer's own workout exercises,
	// scoped by the full owning chain and the target workout.
	found, err := workoutExerciseRepo.FindByIDAndWorkout(ctx, trainer.ID, program.ID, week.ID, workout.ID, entry2.ID)
	if err != nil {
		t.Fatalf("find workout exercise: %v", err)
	}
	if found.ID != entry2.ID || found.Position != 2 || found.ProgramWorkoutID != workout.ID {
		t.Fatalf("unexpected workout exercise %+v", found)
	}
	if found.Exercise.ID != deadlift.ID {
		t.Fatalf("find workout exercise: expected deadlift embedded, got %q", found.Exercise.ID)
	}

	// 8. A workout exercise of another workout or of a foreign chain is
	// indistinguishable and never revealed.
	if _, err := workoutExerciseRepo.FindByIDAndWorkout(ctx, trainer.ID, program.ID, week.ID, otherWorkout.ID, entry2.ID); !errors.Is(err, repositories.ErrWorkoutExerciseNotFound) {
		t.Fatalf("wrong-workout find: expected ErrWorkoutExerciseNotFound, got %v", err)
	}
	if _, err := workoutExerciseRepo.FindByIDAndWorkout(ctx, otherTrainer.ID, program.ID, week.ID, workout.ID, entry2.ID); !errors.Is(err, repositories.ErrWorkoutExerciseNotFound) {
		t.Fatalf("cross-trainer find: expected ErrWorkoutExerciseNotFound, got %v", err)
	}
	if _, err := workoutExerciseRepo.FindByIDAndWorkout(ctx, trainer.ID, program.ID, week.ID, workout.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrWorkoutExerciseNotFound) {
		t.Fatalf("unknown find: expected ErrWorkoutExerciseNotFound, got %v", err)
	}

	// 9. Reorder reassigns positions to the given order.
	if err := workoutExerciseRepo.Reorder(ctx, trainer.ID, program.ID, week.ID, workout.ID, []string{duplicate.ID, entry1.ID, entry2.ID}); err != nil {
		t.Fatalf("reorder workout exercises: %v", err)
	}
	entries, err = workoutExerciseRepo.ListByWorkout(ctx, trainer.ID, program.ID, week.ID, workout.ID)
	if err != nil {
		t.Fatalf("list after reorder: %v", err)
	}
	if entries[0].ID != duplicate.ID || entries[0].Position != 1 {
		t.Fatalf("reorder: expected duplicate first, got %+v", entries[0])
	}
	if entries[1].ID != entry1.ID || entries[2].ID != entry2.ID {
		t.Fatalf("reorder: expected duplicate, entry1, entry2, got %q, %q, %q", entries[0].ID, entries[1].ID, entries[2].ID)
	}

	// 10. Reorder with a wrong list is rejected without touching the order.
	for name, ids := range map[string][]string{
		"missing":      {entry1.ID, entry2.ID},
		"extra":        {entry1.ID, entry2.ID, duplicate.ID, "00000000-0000-0000-0000-000000000000"},
		"duplicate":    {entry1.ID, entry1.ID, duplicate.ID},
		"empty":        {},
		"unknown only": {"00000000-0000-0000-0000-000000000000", "00000000-0000-0000-0000-000000000001"},
	} {
		if err := workoutExerciseRepo.Reorder(ctx, trainer.ID, program.ID, week.ID, workout.ID, ids); !errors.Is(err, repositories.ErrWorkoutExerciseReorderConflict) {
			t.Fatalf("reorder %s: expected ErrWorkoutExerciseReorderConflict, got %v", name, err)
		}
	}
	entries, err = workoutExerciseRepo.ListByWorkout(ctx, trainer.ID, program.ID, week.ID, workout.ID)
	if err != nil {
		t.Fatalf("list after rejected reorder: %v", err)
	}
	if entries[0].ID != duplicate.ID || entries[1].ID != entry1.ID || entries[2].ID != entry2.ID {
		t.Fatalf("rejected reorder must never change the order, got %+v", entries)
	}

	// 11. Reordering a foreign or unknown workout is denied before any write.
	if err := workoutExerciseRepo.Reorder(ctx, otherTrainer.ID, program.ID, week.ID, workout.ID, []string{duplicate.ID, entry1.ID, entry2.ID}); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("cross-trainer reorder: expected ErrWorkoutNotFound, got %v", err)
	}
	if err := workoutExerciseRepo.Reorder(ctx, trainer.ID, program.ID, week.ID, "00000000-0000-0000-0000-000000000000", []string{duplicate.ID, entry1.ID, entry2.ID}); !errors.Is(err, repositories.ErrWorkoutNotFound) {
		t.Fatalf("unknown-workout reorder: expected ErrWorkoutNotFound, got %v", err)
	}

	// 12. Soft delete removes the entry from find and list but keeps the row.
	if err := workoutExerciseRepo.SoftDelete(ctx, trainer.ID, program.ID, week.ID, workout.ID, entry2.ID); err != nil {
		t.Fatalf("soft delete workout exercise: %v", err)
	}
	if _, err := workoutExerciseRepo.FindByIDAndWorkout(ctx, trainer.ID, program.ID, week.ID, workout.ID, entry2.ID); !errors.Is(err, repositories.ErrWorkoutExerciseNotFound) {
		t.Fatalf("soft-deleted entry must not be found, got %v", err)
	}
	var deletedRecord models.WorkoutExercise
	if err := tx.Unscoped().First(&deletedRecord, "id = ?", entry2.ID).Error; err != nil {
		t.Fatalf("soft-deleted row must be preserved: %v", err)
	}
	if !deletedRecord.DeletedAt.Valid {
		t.Fatal("soft-deleted entry must carry a deleted_at marker")
	}
	var sibling models.WorkoutExercise
	if err := tx.Unscoped().First(&sibling, "id = ?", entry1.ID).Error; err != nil {
		t.Fatalf("sibling entries must survive the delete: %v", err)
	}
	if sibling.DeletedAt.Valid {
		t.Fatal("sibling entries must not be soft-deleted with the target")
	}
	entries, err = workoutExerciseRepo.ListByWorkout(ctx, trainer.ID, program.ID, week.ID, workout.ID)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 active entries after delete, got %d", len(entries))
	}
	for _, e := range entries {
		if e.ID == entry2.ID {
			t.Fatal("soft-deleted entry must never appear in the list")
		}
	}

	// 13. The deleted entry no longer participates in reorder: the active set
	// is what must match exactly.
	if err := workoutExerciseRepo.Reorder(ctx, trainer.ID, program.ID, week.ID, workout.ID, []string{duplicate.ID, entry1.ID}); err != nil {
		t.Fatalf("reorder after delete: %v", err)
	}
	if err := workoutExerciseRepo.Reorder(ctx, trainer.ID, program.ID, week.ID, workout.ID, []string{duplicate.ID, entry1.ID, entry2.ID}); !errors.Is(err, repositories.ErrWorkoutExerciseReorderConflict) {
		t.Fatalf("reorder with deleted id: expected ErrWorkoutExerciseReorderConflict, got %v", err)
	}

	// 14. Soft deleting a foreign or unknown workout exercise maps to not found
	// and never touches the target.
	if err := workoutExerciseRepo.SoftDelete(ctx, otherTrainer.ID, program.ID, week.ID, workout.ID, entry1.ID); !errors.Is(err, repositories.ErrWorkoutExerciseNotFound) {
		t.Fatalf("cross-trainer soft delete: expected ErrWorkoutExerciseNotFound, got %v", err)
	}
	if err := workoutExerciseRepo.SoftDelete(ctx, trainer.ID, program.ID, week.ID, otherWorkout.ID, entry1.ID); !errors.Is(err, repositories.ErrWorkoutExerciseNotFound) {
		t.Fatalf("wrong-workout soft delete: expected ErrWorkoutExerciseNotFound, got %v", err)
	}
	if err := workoutExerciseRepo.SoftDelete(ctx, trainer.ID, program.ID, week.ID, workout.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrWorkoutExerciseNotFound) {
		t.Fatalf("unknown soft delete: expected ErrWorkoutExerciseNotFound, got %v", err)
	}
	var untouched models.WorkoutExercise
	if err := tx.Unscoped().First(&untouched, "id = ?", entry1.ID).Error; err != nil {
		t.Fatalf("entry must survive foreign deletes: %v", err)
	}
	if untouched.DeletedAt.Valid {
		t.Fatal("entry must not be soft-deleted by a foreign delete attempt")
	}

	// 15. New additions after a delete reuse the next position of the remaining
	// active entries.
	reentry := &models.WorkoutExercise{ExerciseID: pushUp.ID}
	if err := workoutExerciseRepo.AddExercise(ctx, trainer.ID, program.ID, week.ID, workout.ID, reentry); err != nil {
		t.Fatalf("add after delete: %v", err)
	}
	if reentry.Position != 3 {
		t.Fatalf("expected position 3 after delete, got %d", reentry.Position)
	}

	// 16. A workout soft delete leaves its entries preserved but unreachable.
	if err := workoutRepo.SoftDelete(ctx, trainer.ID, program.ID, week.ID, workout.ID); err != nil {
		t.Fatalf("soft delete workout: %v", err)
	}
	entries, err = workoutExerciseRepo.ListByWorkout(ctx, trainer.ID, program.ID, week.ID, workout.ID)
	if err != nil {
		t.Fatalf("list after workout delete: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("entries of a deleted workout must be unreachable, got %d", len(entries))
	}
	var preserved models.WorkoutExercise
	if err := tx.Unscoped().First(&preserved, "id = ?", duplicate.ID).Error; err != nil {
		t.Fatalf("entry rows must survive the workout soft delete: %v", err)
	}
	if preserved.DeletedAt.Valid {
		t.Fatal("entries must not be soft-deleted with their workout")
	}

	// 17. The workout exercises of the other workout were never touched: the
	// ownership chain kept every scoped write isolated.
	otherEntries, err := workoutExerciseRepo.ListByWorkout(ctx, trainer.ID, program.ID, week.ID, otherWorkout.ID)
	if err != nil {
		t.Fatalf("list other workout: %v", err)
	}
	if len(otherEntries) != 0 {
		t.Fatalf("expected other workout untouched, got %d", len(otherEntries))
	}
}
