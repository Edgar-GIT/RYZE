package repositories_test

import (
	"context"
	"errors"
	"testing"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

func TestExerciseRepository(t *testing.T) {
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

	exerciseRepo := repositories.NewExerciseRepository(tx)
	ctx := context.Background()

	seedExercise := func(name string) *models.Exercise {
		exercise := &models.Exercise{Name: name}
		if err := tx.Create(exercise).Error; err != nil {
			t.Fatalf("seed exercise: %v", err)
		}
		return exercise
	}

	// 1. Seeding generates the id and the lifecycle timestamps.
	squat := seedExercise("Barbell Squat")
	if squat.ID == "" {
		t.Fatal("seed exercise: expected generated UUID id")
	}
	if squat.CreatedAt.IsZero() || squat.UpdatedAt.IsZero() {
		t.Fatal("seed exercise: expected non-zero timestamps")
	}

	// 2. Find one active exercise.
	found, err := exerciseRepo.FindByID(ctx, squat.ID)
	if err != nil {
		t.Fatalf("find exercise: %v", err)
	}
	if found.ID != squat.ID || found.Name != "Barbell Squat" {
		t.Fatalf("unexpected exercise %+v", found)
	}

	// 3. An unknown id is indistinguishable from a missing one.
	if _, err := exerciseRepo.FindByID(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrExerciseNotFound) {
		t.Fatalf("unknown exercise: expected ErrExerciseNotFound, got %v", err)
	}

	// 4. List returns active exercises in alphabetical order.
	deadlift := seedExercise("Deadlift")
	pushUp := seedExercise("Push-Up")
	exercises, total, err := exerciseRepo.List(ctx, 1, 10)
	if err != nil {
		t.Fatalf("list exercises: %v", err)
	}
	if total != 3 {
		t.Fatalf("list exercises: expected total 3, got %d", total)
	}
	if len(exercises) != 3 {
		t.Fatalf("list exercises: expected 3 exercises, got %d", len(exercises))
	}
	if exercises[0].ID != squat.ID || exercises[1].ID != deadlift.ID || exercises[2].ID != pushUp.ID {
		t.Fatalf("list exercises: expected alphabetical order, got %q, %q, %q", exercises[0].ID, exercises[1].ID, exercises[2].ID)
	}

	// 5. Pagination returns the requested page.
	exercises, total, err = exerciseRepo.List(ctx, 1, 2)
	if err != nil {
		t.Fatalf("list exercises page: %v", err)
	}
	if total != 3 || len(exercises) != 2 {
		t.Fatalf("expected page total 3 size 2, got %d/%d", total, len(exercises))
	}

	// 6. Search matches a name substring, case-insensitively.
	matches, matchTotal, err := exerciseRepo.Search(ctx, "squat", 1, 10)
	if err != nil {
		t.Fatalf("search exercises: %v", err)
	}
	if matchTotal != 1 || len(matches) != 1 || matches[0].ID != squat.ID {
		t.Fatalf("search exercises: expected exactly the squat, got %d/%d", matchTotal, len(matches))
	}
	matches, _, err = exerciseRepo.Search(ctx, "BARBELL", 1, 10)
	if err != nil {
		t.Fatalf("search exercises case: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("search exercises: expected case-insensitive match, got %d", len(matches))
	}
	if _, _, err := exerciseRepo.Search(ctx, "no-such-exercise", 1, 10); err != nil {
		t.Fatalf("search exercises no match: %v", err)
	}

	// 7. LIKE wildcards in the query are treated literally.
	percentExercise := seedExercise("100% Effort")
	matches, matchTotal, err = exerciseRepo.Search(ctx, "100%", 1, 10)
	if err != nil {
		t.Fatalf("search wildcard percent: %v", err)
	}
	if matchTotal != 1 || len(matches) != 1 || matches[0].ID != percentExercise.ID {
		t.Fatalf("search wildcard percent: expected only the literal match, got %d/%d", matchTotal, len(matches))
	}
	matches, matchTotal, err = exerciseRepo.Search(ctx, "%%%", 1, 10)
	if err != nil {
		t.Fatalf("search wildcard only: %v", err)
	}
	if matchTotal != 0 || len(matches) != 0 {
		t.Fatalf("search wildcard only: wildcards must never match everything, got %d", matchTotal)
	}

	// 8. Soft delete removes the exercise from find, list and search.
	if err := tx.Delete(&models.Exercise{}, "id = ?", deadlift.ID).Error; err != nil {
		t.Fatalf("soft delete exercise: %v", err)
	}
	if _, err := exerciseRepo.FindByID(ctx, deadlift.ID); !errors.Is(err, repositories.ErrExerciseNotFound) {
		t.Fatalf("soft-deleted exercise must not be found, got %v", err)
	}
	var deletedRecord models.Exercise
	if err := tx.Unscoped().First(&deletedRecord, "id = ?", deadlift.ID).Error; err != nil {
		t.Fatalf("soft-deleted exercise row must be preserved: %v", err)
	}
	if !deletedRecord.DeletedAt.Valid {
		t.Fatal("soft-deleted exercise must carry a deleted_at marker")
	}
	exercises, total, err = exerciseRepo.List(ctx, 1, 10)
	if err != nil {
		t.Fatalf("list exercises after delete: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected 3 active exercises after delete, got %d", total)
	}
	for _, e := range exercises {
		if e.ID == deadlift.ID {
			t.Fatal("soft-deleted exercise must never appear in the list")
		}
	}

	// 9. The database rejects a NULL name (NOT NULL constraint). A non-empty
	// name is enforced by the service, following the project pattern where
	// business validation lives outside the repository.
	var blank models.Exercise
	blank.Name = "to-be-cleared"
	if err := tx.Create(&blank).Error; err != nil {
		t.Fatalf("seed blank-name exercise: %v", err)
	}
	if err := tx.Exec("UPDATE exercises SET name = NULL WHERE id = ?", blank.ID).Error; err == nil {
		t.Fatal("NULL exercise name: expected a database error")
	}
	if err := tx.Delete(&models.Exercise{}, "id = ?", blank.ID).Error; err != nil {
		t.Fatalf("cleanup blank-name exercise: %v", err)
	}

	// 10. Every exercise exists only once: an active name is unique.
	duplicate := &models.Exercise{Name: "Barbell Squat"}
	if err := tx.Create(duplicate).Error; err == nil {
		t.Fatal("duplicate active exercise name: expected a database error")
	}

	// 11. Soft deleting an exercise frees its name for a new catalog entry.
	if err := tx.Delete(&models.Exercise{}, "id = ?", squat.ID).Error; err != nil {
		t.Fatalf("soft delete squat: %v", err)
	}
	if err := tx.Create(&models.Exercise{Name: "Barbell Squat"}).Error; err != nil {
		t.Fatalf("reused name after soft delete must be allowed: %v", err)
	}
}
