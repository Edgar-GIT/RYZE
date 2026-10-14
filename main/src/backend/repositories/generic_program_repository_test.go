package repositories_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

func TestGenericProgramRepository(t *testing.T) {
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

	ctx := context.Background()
	programRepo := repositories.NewGenericProgramRepository(tx)

	seedExercise := func(name string) *models.Exercise {
		exercise := &models.Exercise{
			Name:         name,
			Description:  "A catalogue exercise.",
			Instructions: "Perform the movement.",
			Difficulty:   "Intermediate",
		}
		if err := tx.Create(exercise).Error; err != nil {
			t.Fatalf("seed exercise: %v", err)
		}
		return exercise
	}

	setModel := func(exerciseID string, setNumber int) models.WorkoutExerciseSet {
		return models.WorkoutExerciseSet{
			SetNumber: setNumber,
			SetType:   "working",
			Tempo:     "2010",
		}
	}

	exerciseModel := func(exerciseID string, position int, setCount int) models.WorkoutExercise {
		sets := make([]models.WorkoutExerciseSet, 0, setCount)
		for i := 1; i <= setCount; i++ {
			sets = append(sets, setModel(exerciseID, i))
		}
		return models.WorkoutExercise{ExerciseID: exerciseID, Position: position, Sets: sets}
	}

	workoutModel := func(position int, exercises ...models.WorkoutExercise) models.ProgramWorkout {
		return models.ProgramWorkout{Position: position, Exercises: exercises}
	}

	weekModel := func(weekNumber int, workouts ...models.ProgramWorkout) models.ProgramWeek {
		return models.ProgramWeek{WeekNumber: weekNumber, Workouts: workouts}
	}

	level := "Intermediate"
	duration := 12
	frequency := 4

	buildProgram := func(name string, weeks []models.ProgramWeek) *models.Program {
		return &models.Program{
			Name:             name,
			Description:      "A generic program.",
			Type:             models.ProgramTypePremium,
			Status:           models.ProgramStatusDraft,
			Level:            &level,
			DurationWeeks:    &duration,
			FrequencyPerWeek: &frequency,
			PriceMinorUnits:  1449,
			Currency:         "EUR",
			Weeks:            weeks,
		}
	}

	exerciseA := seedExercise("Generic Bench Press")
	exerciseB := seedExercise("Generic Barbell Row")

	// 1. CreateFull persists the whole tree with trainer_id NULL.
	program := buildProgram("Hypertrophy 101", []models.ProgramWeek{
		weekModel(1,
			workoutModel(1, exerciseModel(exerciseA.ID, 1, 2)),
			workoutModel(2, exerciseModel(exerciseB.ID, 1, 1)),
		),
		weekModel(2, workoutModel(1, exerciseModel(exerciseB.ID, 1, 3))),
	})
	if err := programRepo.CreateFull(ctx, program); err != nil {
		t.Fatalf("CreateFull: %v", err)
	}
	if program.ID == "" {
		t.Fatal("CreateFull: expected generated program id")
	}
	if program.CreatedAt.IsZero() || program.UpdatedAt.IsZero() {
		t.Fatal("CreateFull: expected timestamps")
	}

	// The program row must be created without a trainer (platform-owned).
	var programRow struct {
		TrainerID *string
	}
	if err := tx.Model(&models.Program{}).
		Where("id = ?", program.ID).
		Select("trainer_id").
		Scan(&programRow).Error; err != nil {
		t.Fatalf("verify trainer_id: %v", err)
	}
	if programRow.TrainerID != nil {
		t.Fatalf("generic programs must be created with trainer_id NULL, got %q", *programRow.TrainerID)
	}
	if program.TrainerID != "" {
		t.Fatalf("model trainer id must stay empty, got %q", program.TrainerID)
	}
	for i := range program.Weeks {
		if program.Weeks[i].ID == "" {
			t.Fatalf("week %d: expected generated id", i+1)
		}
		for k := range program.Weeks[i].Workouts {
			if program.Weeks[i].Workouts[k].ID == "" {
				t.Fatalf("workout %d-%d: expected generated id", i+1, k+1)
			}
			for j := range program.Weeks[i].Workouts[k].Exercises {
				if program.Weeks[i].Workouts[k].Exercises[j].ID == "" {
					t.Fatalf("exercise %d-%d-%d: expected generated id", i+1, k+1, j+1)
				}
			}
		}
	}

	// 2. FindByID returns the full ordered structure.
	found, err := programRepo.FindByID(ctx, models.ProgramProductTypeGeneric, program.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Name != "Hypertrophy 101" || len(found.Weeks) != 2 {
		t.Fatalf("unexpected program %+v", found)
	}
	if found.Weeks[0].WeekNumber != 1 || len(found.Weeks[0].Workouts) != 2 {
		t.Fatalf("unexpected weeks %+v", found.Weeks)
	}
	firstWorkout := found.Weeks[0].Workouts[0]
	if firstWorkout.Position != 1 || len(firstWorkout.Exercises) != 1 || len(firstWorkout.Exercises[0].Sets) != 2 {
		t.Fatalf("unexpected structure %+v", firstWorkout)
	}
	if firstWorkout.Exercises[0].Sets[0].SetNumber != 1 || firstWorkout.Exercises[0].Sets[1].SetNumber != 2 {
		t.Fatalf("sets must be ordered, got %+v", firstWorkout.Exercises[0].Sets)
	}
	if found.Weeks[0].Workouts[1].Exercises[0].Exercise.Name != "Generic Barbell Row" {
		t.Fatalf("expected preloaded catalog exercise, got %+v", found.Weeks[0].Workouts[1].Exercises[0].Exercise)
	}

	// 3. FindByID rejects a missing program.
	if _, err := programRepo.FindByID(ctx, models.ProgramProductTypeGeneric, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrGenericProgramNotFound) {
		t.Fatalf("missing program: expected ErrGenericProgramNotFound, got %v", err)
	}

	// 4. CreateFull rejects unknown catalogue exercises atomically.
	badProgram := buildProgram("Broken Program", []models.ProgramWeek{
		weekModel(1, workoutModel(1, exerciseModel("00000000-0000-0000-0000-000000000000", 1, 1))),
	})
	if err := programRepo.CreateFull(ctx, badProgram); !errors.Is(err, repositories.ErrExerciseNotFound) {
		t.Fatalf("unknown exercise: expected ErrExerciseNotFound, got %v", err)
	}
	var brokenCount int64
	tx.Model(&models.Program{}).Where("name = ?", "Broken Program").Count(&brokenCount)
	if brokenCount != 0 {
		t.Fatalf("the transaction must roll back on failure, found %d programs", brokenCount)
	}

	// 5. Search: query, filters and ordering.
	result, total, err := programRepo.Search(ctx, models.ProgramProductTypeGeneric, repositories.GenericProgramFilter{Query: "hypertrophy"}, 1, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total != 1 || len(result) != 1 || result[0].ID != program.ID {
		t.Fatalf("expected one result for hypertrophy, got %d/%d", total, len(result))
	}
	if _, total, err := programRepo.Search(ctx, models.ProgramProductTypeGeneric, repositories.GenericProgramFilter{Query: "NONE" + fmt.Sprint(time.Now().UnixNano())}, 1, 10); err != nil || total != 0 {
		t.Fatalf("expected zero results, got total=%d err=%v", total, err)
	}
	if _, total, err := programRepo.Search(ctx, models.ProgramProductTypeGeneric, repositories.GenericProgramFilter{Type: "free"}, 1, 10); err != nil || total != 0 {
		t.Fatalf("type filter: expected zero results, got total=%d err=%v", total, err)
	}
	if _, total, err := programRepo.Search(ctx, models.ProgramProductTypeGeneric, repositories.GenericProgramFilter{Level: "Beginner"}, 1, 10); err != nil || total != 0 {
		t.Fatalf("level filter: expected zero results, got total=%d err=%v", total, err)
	}
	if _, total, err := programRepo.Search(ctx, models.ProgramProductTypeGeneric, repositories.GenericProgramFilter{DurationWeeks: 12}, 1, 10); err != nil || total != 1 {
		t.Fatalf("duration filter: expected 1 result, got total=%d err=%v", total, err)
	}
	if _, total, err := programRepo.Search(ctx, models.ProgramProductTypeGeneric, repositories.GenericProgramFilter{FrequencyPerWeek: 4}, 1, 10); err != nil || total != 1 {
		t.Fatalf("frequency filter: expected 1 result, got total=%d err=%v", total, err)
	}

	// 6. UpdateFull grows the structure (new week) and reuses existing ids.
	week0ID := program.Weeks[0].ID
	workout0ID := program.Weeks[0].Workouts[0].ID
	updated := buildProgram("Hypertrophy 101 v2", []models.ProgramWeek{
		weekModel(1,
			workoutModel(1, exerciseModel(exerciseA.ID, 1, 3)),
			workoutModel(2, exerciseModel(exerciseB.ID, 1, 1)),
		),
		weekModel(2, workoutModel(1, exerciseModel(exerciseB.ID, 1, 2))),
		weekModel(3, workoutModel(1, exerciseModel(exerciseA.ID, 1, 1))),
	})
	if err := programRepo.UpdateFull(ctx, models.ProgramProductTypeGeneric, program.ID, updated); err != nil {
		t.Fatalf("UpdateFull: %v", err)
	}

	after, err := programRepo.FindByID(ctx, models.ProgramProductTypeGeneric, program.ID)
	if err != nil {
		t.Fatalf("FindByID after update: %v", err)
	}
	if len(after.Weeks) != 3 {
		t.Fatalf("expected 3 weeks after update, got %d", len(after.Weeks))
	}
	// Reused week keeps its row id.
	if after.Weeks[0].ID != week0ID {
		t.Fatalf("reused week id must be preserved, got %q want %q", after.Weeks[0].ID, week0ID)
	}
	// The new week is created and ordered last.
	if after.Weeks[2].WeekNumber != 3 {
		t.Fatalf("expected third week number 3, got %d", after.Weeks[2].WeekNumber)
	}
	// The reused first workout keeps its row id and its exercise set grew.
	if after.Weeks[0].Workouts[0].ID != workout0ID {
		t.Fatalf("reused workout id must be preserved, got %q want %q", after.Weeks[0].Workouts[0].ID, workout0ID)
	}
	if len(after.Weeks[0].Workouts[0].Exercises[0].Sets) != 3 {
		t.Fatalf("expected 3 sets after update, got %d", len(after.Weeks[0].Workouts[0].Exercises[0].Sets))
	}

	// 7. UpdateFull shrinks the structure: removing weeks soft-deletes the tail.
	shrunk := buildProgram("Hypertrophy 101 v1", []models.ProgramWeek{
		weekModel(1, workoutModel(1, exerciseModel(exerciseA.ID, 1, 1))),
	})
	if err := programRepo.UpdateFull(ctx, models.ProgramProductTypeGeneric, program.ID, shrunk); err != nil {
		t.Fatalf("UpdateFull shrink: %v", err)
	}
	afterShrink, err := programRepo.FindByID(ctx, models.ProgramProductTypeGeneric, program.ID)
	if err != nil {
		t.Fatalf("FindByID after shrink: %v", err)
	}
	if len(afterShrink.Weeks) != 1 {
		t.Fatalf("expected 1 week after shrink, got %d", len(afterShrink.Weeks))
	}
	var softDeletedWeeks int64
	tx.Unscoped().Model(&models.ProgramWeek{}).Where("program_id = ?", program.ID).Where("deleted_at IS NOT NULL").Count(&softDeletedWeeks)
	if softDeletedWeeks != 2 {
		t.Fatalf("expected 2 soft-deleted tail weeks, got %d", softDeletedWeeks)
	}

	// 8. Publish transitions only draft programs; already published maps to not
	// found.
	if err := programRepo.Publish(ctx, models.ProgramProductTypeGeneric, program.ID); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	published, err := programRepo.FindByID(ctx, models.ProgramProductTypeGeneric, program.ID)
	if err != nil {
		t.Fatalf("FindByID after publish: %v", err)
	}
	if published.Status != models.ProgramStatusPublished {
		t.Fatalf("expected published, got %q", published.Status)
	}
	if err := programRepo.Publish(ctx, models.ProgramProductTypeGeneric, program.ID); !errors.Is(err, repositories.ErrGenericProgramNotFound) {
		t.Fatalf("republish: expected ErrGenericProgramNotFound, got %v", err)
	}

	// 9. UpdateFull rejects a missing program and unknown exercises.
	if err := programRepo.UpdateFull(ctx, models.ProgramProductTypeGeneric, "00000000-0000-0000-0000-000000000000", buildProgram("Ghost", nil)); !errors.Is(err, repositories.ErrGenericProgramNotFound) {
		t.Fatalf("missing program update: expected ErrGenericProgramNotFound, got %v", err)
	}
	broken := buildProgram("Broken Update", []models.ProgramWeek{
		weekModel(1, workoutModel(1, exerciseModel("00000000-0000-0000-0000-000000000000", 1, 1))),
	})
	if err := programRepo.UpdateFull(ctx, models.ProgramProductTypeGeneric, program.ID, broken); !errors.Is(err, repositories.ErrExerciseNotFound) {
		t.Fatalf("unknown exercise update: expected ErrExerciseNotFound, got %v", err)
	}

	// 10. SoftDelete hides the program everywhere.
	if err := programRepo.SoftDelete(ctx, models.ProgramProductTypeGeneric, program.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	if _, err := programRepo.FindByID(ctx, models.ProgramProductTypeGeneric, program.ID); !errors.Is(err, repositories.ErrGenericProgramNotFound) {
		t.Fatalf("deleted program: expected ErrGenericProgramNotFound, got %v", err)
	}
	if _, total, err := programRepo.Search(ctx, models.ProgramProductTypeGeneric, repositories.GenericProgramFilter{}, 1, 10); err != nil || total != 0 {
		t.Fatalf("deleted program search: expected 0 total, got %d err=%v", total, err)
	}
	if err := programRepo.SoftDelete(ctx, models.ProgramProductTypeGeneric, program.ID); !errors.Is(err, repositories.ErrGenericProgramNotFound) {
		t.Fatalf("double delete: expected ErrGenericProgramNotFound, got %v", err)
	}

	// 11. Search excludes soft-deleted programs while ordering the rest.
	other := buildProgram("Alpha Strength", []models.ProgramWeek{
		weekModel(1, workoutModel(1, exerciseModel(exerciseA.ID, 1, 1))),
	})
	if err := programRepo.CreateFull(ctx, other); err != nil {
		t.Fatalf("CreateFull other: %v", err)
	}
	list, total, err := programRepo.Search(ctx, models.ProgramProductTypeGeneric, repositories.GenericProgramFilter{}, 1, 10)
	if err != nil {
		t.Fatalf("Search after delete: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != other.ID {
		t.Fatalf("expected only the active program, got total=%d", total)
	}
	if list[0].Level == nil || *list[0].Level != "Intermediate" {
		t.Fatalf("expected level metadata in search results, got %v", list[0].Level)
	}
}

// TestGenericProgramRepositoryExcludesPremiumProductFamily verifies that
// generic-program management is scoped by product family and not by ownership
// alone. A platform-owned Premium Level 1 program has trainer_id NULL, so an
// ownership-only predicate would make it reachable here and let it be edited
// outside the family that enforces the questionnaire prerequisite. Every
// operation here is executed through the generic family scope, which must leave
// a premium_level_1 program untouched.
func TestGenericProgramRepositoryExcludesPremiumProductFamily(t *testing.T) {
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

	repo := repositories.NewGenericProgramRepository(tx)
	ctx := context.Background()

	// A platform-owned Premium Level 1 program: owned by nobody, but not a
	// generic program.
	premium := &models.Program{
		ID:          uuid.NewString(),
		Name:        "Premium Family Probe",
		Type:        models.ProgramTypePremium,
		Status:      models.ProgramStatusPublished,
		ProductType: models.ProgramProductTypePremiumLevel1,
	}
	if err := tx.Exec(
		"INSERT INTO programs (id, trainer_id, name, type, status, product_type) VALUES (?, NULL, ?, ?, ?, ?)",
		premium.ID, premium.Name, premium.Type, premium.Status, premium.ProductType,
	).Error; err != nil {
		t.Fatalf("seed premium program: %v", err)
	}

	if _, err := repo.FindByID(ctx, models.ProgramProductTypeGeneric, premium.ID); !errors.Is(err, repositories.ErrGenericProgramNotFound) {
		t.Fatalf("FindByID on a Premium Level 1 program = %v, want ErrGenericProgramNotFound", err)
	}

	if err := repo.UpdateFull(ctx, models.ProgramProductTypeGeneric, premium.ID, &models.Program{Name: "Renamed"}); !errors.Is(err, repositories.ErrGenericProgramNotFound) {
		t.Fatalf("UpdateFull on a Premium Level 1 program = %v, want ErrGenericProgramNotFound", err)
	}

	if err := repo.Publish(ctx, models.ProgramProductTypeGeneric, premium.ID); !errors.Is(err, repositories.ErrGenericProgramNotFound) {
		t.Fatalf("Publish on a Premium Level 1 program = %v, want ErrGenericProgramNotFound", err)
	}

	if err := repo.SoftDelete(ctx, models.ProgramProductTypeGeneric, premium.ID); !errors.Is(err, repositories.ErrGenericProgramNotFound) {
		t.Fatalf("SoftDelete on a Premium Level 1 program = %v, want ErrGenericProgramNotFound", err)
	}

	// The seeded row is untouched and still published.
	var name, status, productType string
	if err := tx.Raw(
		"SELECT name, status, product_type FROM programs WHERE id = ?", premium.ID,
	).Row().Scan(&name, &status, &productType); err != nil {
		t.Fatalf("re-read premium program: %v", err)
	}
	if name != premium.Name || status != string(models.ProgramStatusPublished) || productType != models.ProgramProductTypePremiumLevel1 {
		t.Fatalf("premium program was modified through generic management: name %q, status %q, product_type %q", name, status, productType)
	}
}

// TestGenericProgramRepositoryPremiumLevel1Family verifies that the same
// repository manages a platform-owned Premium Level 1 program when the family
// scope is premium_level_1: create persists the family, find/search only answer
// under the premium scope, and generic-family operations leave the premium
// program untouched in both directions. This is the data-access layer that makes
// the premium administration route safe.
func TestGenericProgramRepositoryPremiumLevel1Family(t *testing.T) {
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

	repo := repositories.NewGenericProgramRepository(tx)
	ctx := context.Background()

	exercise := &models.Exercise{
		Name:         "Premium Family Bench Press",
		Description:  "A catalogue exercise.",
		Instructions: "Perform the movement.",
		Difficulty:   "Intermediate",
	}
	if err := tx.Create(exercise).Error; err != nil {
		t.Fatalf("seed exercise: %v", err)
	}

	level := "Beginner"
	duration := 12
	frequency := 3
	premium := &models.Program{
		Name:             "Premium Level 1 Starter",
		Description:      "A premium level 1 program.",
		Type:             models.ProgramTypePremium,
		Status:           models.ProgramStatusDraft,
		ProductType:      models.ProgramProductTypePremiumLevel1,
		Level:            &level,
		DurationWeeks:    &duration,
		FrequencyPerWeek: &frequency,
		PriceMinorUnits:  4999,
		Currency:         "EUR",
		Weeks: []models.ProgramWeek{
			{
				WeekNumber: 1,
				Workouts: []models.ProgramWorkout{
					{
						Position: 1,
						Exercises: []models.WorkoutExercise{
							{
								ExerciseID: exercise.ID,
								Position:   1,
								Sets: []models.WorkoutExerciseSet{
									{SetNumber: 1, SetType: "working", Tempo: "2010"},
								},
							},
						},
					},
				},
			},
		},
	}
	if err := repo.CreateFull(ctx, premium); err != nil {
		t.Fatalf("CreateFull premium: %v", err)
	}
	if premium.ProductType != models.ProgramProductTypePremiumLevel1 {
		t.Fatalf("premium family must be persisted, got %q", premium.ProductType)
	}
	if premium.TrainerID != "" {
		t.Fatalf("premium program must stay platform-owned, got trainer %q", premium.TrainerID)
	}

	// The premium program is reachable under its own family...
	found, err := repo.FindByID(ctx, models.ProgramProductTypePremiumLevel1, premium.ID)
	if err != nil {
		t.Fatalf("FindByID premium family: %v", err)
	}
	if found.Name != premium.Name || found.ProductType != models.ProgramProductTypePremiumLevel1 {
		t.Fatalf("unexpected premium program %+v", found)
	}
	if _, total, err := repo.Search(ctx, models.ProgramProductTypePremiumLevel1, repositories.GenericProgramFilter{}, 1, 10); err != nil || total != 1 {
		t.Fatalf("premium search: expected 1 result, got total=%d err=%v", total, err)
	}

	// ...and invisible under the generic family.
	if _, err := repo.FindByID(ctx, models.ProgramProductTypeGeneric, premium.ID); !errors.Is(err, repositories.ErrGenericProgramNotFound) {
		t.Fatalf("generic FindByID on premium program = %v, want ErrGenericProgramNotFound", err)
	}
	if _, total, err := repo.Search(ctx, models.ProgramProductTypeGeneric, repositories.GenericProgramFilter{}, 1, 10); err != nil || total != 0 {
		t.Fatalf("generic search must not surface premium programs, got total=%d err=%v", total, err)
	}

	// Family is immutable: the generic scope cannot modify, publish or delete
	// the premium program.
	if err := repo.UpdateFull(ctx, models.ProgramProductTypeGeneric, premium.ID, &models.Program{Name: "Hijacked"}); !errors.Is(err, repositories.ErrGenericProgramNotFound) {
		t.Fatalf("generic UpdateFull on premium program = %v, want ErrGenericProgramNotFound", err)
	}
	if err := repo.Publish(ctx, models.ProgramProductTypeGeneric, premium.ID); !errors.Is(err, repositories.ErrGenericProgramNotFound) {
		t.Fatalf("generic Publish on premium program = %v, want ErrGenericProgramNotFound", err)
	}
	if err := repo.SoftDelete(ctx, models.ProgramProductTypeGeneric, premium.ID); !errors.Is(err, repositories.ErrGenericProgramNotFound) {
		t.Fatalf("generic SoftDelete on premium program = %v, want ErrGenericProgramNotFound", err)
	}

	// The premium family manages its own lifecycle and never touches generic
	// programs.
	updated := *premium
	updated.Name = "Premium Level 1 Starter v2"
	if err := repo.UpdateFull(ctx, models.ProgramProductTypePremiumLevel1, premium.ID, &updated); err != nil {
		t.Fatalf("UpdateFull premium family: %v", err)
	}
	if err := repo.Publish(ctx, models.ProgramProductTypePremiumLevel1, premium.ID); err != nil {
		t.Fatalf("Publish premium family: %v", err)
	}
	published, err := repo.FindByID(ctx, models.ProgramProductTypePremiumLevel1, premium.ID)
	if err != nil {
		t.Fatalf("FindByID after premium publish: %v", err)
	}
	if published.Status != models.ProgramStatusPublished || published.Name != "Premium Level 1 Starter v2" {
		t.Fatalf("expected published and renamed, got %+v", published)
	}

	generic := &models.Program{
		Name:            "Generic Compare",
		Description:     "A generic program.",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusDraft,
		PriceMinorUnits: 1449,
		Currency:        "EUR",
	}
	if err := repo.CreateFull(ctx, generic); err != nil {
		t.Fatalf("CreateFull generic: %v", err)
	}
	if err := repo.UpdateFull(ctx, models.ProgramProductTypePremiumLevel1, generic.ID, &models.Program{Name: "Cross Family"}); !errors.Is(err, repositories.ErrGenericProgramNotFound) {
		t.Fatalf("premium UpdateFull on generic program = %v, want ErrGenericProgramNotFound", err)
	}
	if err := repo.SoftDelete(ctx, models.ProgramProductTypePremiumLevel1, premium.ID); err != nil {
		t.Fatalf("SoftDelete premium family: %v", err)
	}
	if _, err := repo.FindByID(ctx, models.ProgramProductTypePremiumLevel1, premium.ID); !errors.Is(err, repositories.ErrGenericProgramNotFound) {
		t.Fatalf("deleted premium program: expected ErrGenericProgramNotFound, got %v", err)
	}
	if err := repo.SoftDelete(ctx, models.ProgramProductTypeGeneric, generic.ID); err != nil {
		t.Fatalf("SoftDelete generic: %v", err)
	}
}
