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

	// The catalog ships with platform seed data, so every assertion in this test
	// is relative to a baseline measured at the start of the transaction. The
	// test's own rows use a distinctive name prefix to stay isolated from seeds.
	_, baselineTotal, err := exerciseRepo.List(ctx, 1, 1)
	if err != nil {
		t.Fatalf("baseline list: %v", err)
	}

	seedExercise := func(name string) *models.Exercise {
		exercise := &models.Exercise{Name: name}
		if err := tx.Create(exercise).Error; err != nil {
			t.Fatalf("seed exercise: %v", err)
		}
		return exercise
	}

	seedFullExercise := func(name, primary, category, difficulty string) *models.Exercise {
		exercise := &models.Exercise{
			Name:                 name,
			Description:          "Repository test exercise.",
			Instructions:         "Perform the movement under control.",
			TargetMuscles:        primary,
			PrimaryMuscleGroup:   primary,
			SecondaryMuscleGroups: "Core",
			Equipment:            "Barbell, Bench",
			Difficulty:           difficulty,
			MovementCategory:     category,
		}
		if err := tx.Create(exercise).Error; err != nil {
			t.Fatalf("seed full exercise: %v", err)
		}
		return exercise
	}

	// 1. Seeding generates the id and the lifecycle timestamps.
	squat := seedExercise("RYZE-Test-A")
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
	if found.ID != squat.ID || found.Name != "RYZE-Test-A" {
		t.Fatalf("unexpected exercise %+v", found)
	}

	// 3. An unknown id is indistinguishable from a missing one.
	if _, err := exerciseRepo.FindByID(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrExerciseNotFound) {
		t.Fatalf("unknown exercise: expected ErrExerciseNotFound, got %v", err)
	}

	// 4. List returns active exercises with a total relative to the baseline.
	chestMove := seedFullExercise("RYZE-Test-B2", "Chest", "Compound", "Intermediate")
	quadsMove := seedFullExercise("RYZE-Test-C3", "Quads", "Isolation", "Beginner")

	exercises, total, err := exerciseRepo.List(ctx, 1, 10)
	if err != nil {
		t.Fatalf("list exercises: %v", err)
	}
	if total != baselineTotal+3 {
		t.Fatalf("list exercises: expected total %d, got %d", baselineTotal+3, total)
	}
	if len(exercises) != 10 {
		t.Fatalf("list exercises: expected 10 per page, got %d", len(exercises))
	}

	// 5. Pagination returns the requested page size.
	pageExercises, pageTotal, err := exerciseRepo.List(ctx, 1, 2)
	if err != nil {
		t.Fatalf("list exercises page: %v", err)
	}
	if pageTotal != baselineTotal+3 || len(pageExercises) != 2 {
		t.Fatalf("expected page total %d size 2, got %d/%d", baselineTotal+3, pageTotal, len(pageExercises))
	}

	// 6. Search matches a name substring, case-insensitively, in alphabetical
	// order. The name prefix isolates the test's own rows from the catalog.
	matches, matchTotal, err := exerciseRepo.Search(ctx, "RYZE-Test", 1, 10)
	if err != nil {
		t.Fatalf("search exercises: %v", err)
	}
	if matchTotal != 3 || len(matches) != 3 {
		t.Fatalf("search exercises: expected the three test rows, got %d/%d", matchTotal, len(matches))
	}
	if matches[0].ID != squat.ID || matches[1].ID != chestMove.ID || matches[2].ID != quadsMove.ID {
		t.Fatalf("search exercises: expected alphabetical order, got %q, %q, %q", matches[0].Name, matches[1].Name, matches[2].Name)
	}
	matches, matchTotal, err = exerciseRepo.Search(ctx, "ryze-test-a", 1, 10)
	if err != nil {
		t.Fatalf("search exercises case: %v", err)
	}
	if matchTotal != 1 || matches[0].ID != squat.ID {
		t.Fatalf("search exercises: expected case-insensitive match, got %d", matchTotal)
	}
	if _, _, err := exerciseRepo.Search(ctx, "no-such-exercise", 1, 10); err != nil {
		t.Fatalf("search exercises no match: %v", err)
	}

	// 7. LIKE wildcards in the query are treated literally.
	wildExercise := seedExercise("RYZE-Wild-100% Effort")
	matches, matchTotal, err = exerciseRepo.Search(ctx, "RYZE-Wild-100%", 1, 10)
	if err != nil {
		t.Fatalf("search wildcard percent: %v", err)
	}
	if matchTotal != 1 || len(matches) != 1 || matches[0].ID != wildExercise.ID {
		t.Fatalf("search wildcard percent: expected only the literal match, got %d/%d", matchTotal, len(matches))
	}
	matches, matchTotal, err = exerciseRepo.Search(ctx, "%%%", 1, 10)
	if err != nil {
		t.Fatalf("search wildcard only: %v", err)
	}
	if matchTotal != 0 || len(matches) != 0 {
		t.Fatalf("search wildcard only: wildcards must never match everything, got %d", matchTotal)
	}

	// 8. Library combines name, muscle and difficulty criteria. The name prefix
	// keeps these counts stable regardless of the catalog contents.
	filter := repositories.ExerciseSearchFilter{Query: "RYZE-Test", Difficulty: "Intermediate"}
	matches, matchTotal, err = exerciseRepo.Library(ctx, filter, 1, 10)
	if err != nil {
		t.Fatalf("library compound filter: %v", err)
	}
	if matchTotal != 1 || matches[0].ID != chestMove.ID {
		t.Fatalf("library compound filter: expected one match for B2, got %d", matchTotal)
	}
	filter = repositories.ExerciseSearchFilter{Query: "RYZE-Test", Category: "Isolation"}
	matches, matchTotal, err = exerciseRepo.Library(ctx, filter, 1, 10)
	if err != nil {
		t.Fatalf("library category filter: %v", err)
	}
	if matchTotal != 1 || matches[0].ID != quadsMove.ID {
		t.Fatalf("library category filter: expected one match for C3, got %d", matchTotal)
	}

	// 9. Soft delete removes the exercise from find, list and search.
	if err := tx.Delete(&models.Exercise{}, "id = ?", chestMove.ID).Error; err != nil {
		t.Fatalf("soft delete exercise: %v", err)
	}
	if _, err := exerciseRepo.FindByID(ctx, chestMove.ID); !errors.Is(err, repositories.ErrExerciseNotFound) {
		t.Fatalf("soft-deleted exercise must not be found, got %v", err)
	}
	var deletedRecord models.Exercise
	if err := tx.Unscoped().First(&deletedRecord, "id = ?", chestMove.ID).Error; err != nil {
		t.Fatalf("soft-deleted exercise row must be preserved: %v", err)
	}
	if !deletedRecord.DeletedAt.Valid {
		t.Fatal("soft-deleted exercise must carry a deleted_at marker")
	}
	exercises, total, err = exerciseRepo.List(ctx, 1, 10)
	if err != nil {
		t.Fatalf("list exercises after delete: %v", err)
	}
	if total != baselineTotal+3 {
		t.Fatalf("expected %d active exercises after delete, got %d", baselineTotal+3, total)
	}
	for _, e := range exercises {
		if e.ID == chestMove.ID {
			t.Fatal("soft-deleted exercise must never appear in the list")
		}
	}

	// 10. Alternative links resolve the alternative name from the catalog, hide
	// soft-deleted links and always point to a different exercise.
	link := &models.ExerciseAlternative{ExerciseID: squat.ID, AlternativeExerciseID: quadsMove.ID}
	if err := tx.Create(link).Error; err != nil {
		t.Fatalf("create alternative link: %v", err)
	}
	if link.ID == "" {
		t.Fatal("alternative link: expected generated UUID id")
	}
	links, err := exerciseRepo.ListAlternatives(ctx, squat.ID)
	if err != nil {
		t.Fatalf("list alternative links: %v", err)
	}
	if len(links) != 1 || links[0].AlternativeExerciseID != quadsMove.ID || links[0].AlternativeName != "RYZE-Test-C3" {
		t.Fatalf("unexpected alternative links %+v", links)
	}
	if err := tx.Delete(&models.ExerciseAlternative{}, "id = ?", link.ID).Error; err != nil {
		t.Fatalf("soft delete alternative link: %v", err)
	}
	links, err = exerciseRepo.ListAlternatives(ctx, squat.ID)
	if err != nil {
		t.Fatalf("list alternative links after delete: %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("soft-deleted alternative link must not be returned, got %d", len(links))
	}
	if err := tx.Create(&models.ExerciseAlternative{ExerciseID: squat.ID, AlternativeExerciseID: squat.ID}).Error; err == nil {
		t.Fatal("self-referencing alternative link: expected a database error")
	}

	// 11. The database rejects a NULL name (NOT NULL constraint). A non-empty
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

	// 12. Every exercise exists only once: an active name is unique.
	duplicate := &models.Exercise{Name: "RYZE-Test-A"}
	if err := tx.Create(duplicate).Error; err == nil {
		t.Fatal("duplicate active exercise name: expected a database error")
	}

	// 13. Soft deleting an exercise frees its name for a new catalog entry.
	if err := tx.Delete(&models.Exercise{}, "id = ?", squat.ID).Error; err != nil {
		t.Fatalf("soft delete squat: %v", err)
	}
	if err := tx.Create(&models.Exercise{Name: "RYZE-Test-A"}).Error; err != nil {
		t.Fatalf("reused name after soft delete must be allowed: %v", err)
	}
}