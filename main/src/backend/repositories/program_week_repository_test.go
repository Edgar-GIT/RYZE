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

func TestProgramWeekRepository(t *testing.T) {
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
	ctx := context.Background()

	seedUser := func() *models.User {
		user := &models.User{
			Email:        fmt.Sprintf("week-repo-%d@ryze.local", time.Now().UnixNano()),
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

	trainer := seedTrainer()
	otherTrainer := seedTrainer()
	program := seedProgram(trainer.ID, "Strength Builder")
	foreignProgram := seedProgram(otherTrainer.ID, "Foreign Program")

	// 1. Create appends weeks in increasing week number order.
	week1 := seedWeek(trainer.ID, program.ID)
	if week1.ID == "" {
		t.Fatal("create week: expected generated UUID id")
	}
	if week1.WeekNumber != 1 {
		t.Fatalf("create week: expected week number 1, got %d", week1.WeekNumber)
	}
	if week1.CreatedAt.IsZero() || week1.UpdatedAt.IsZero() {
		t.Fatal("create week: expected non-zero timestamps")
	}
	if week1.ProgramID != program.ID {
		t.Fatalf("create week: expected program %q, got %q", program.ID, week1.ProgramID)
	}
	week2 := seedWeek(trainer.ID, program.ID)
	if week2.WeekNumber != 2 {
		t.Fatalf("create week: expected week number 2, got %d", week2.WeekNumber)
	}
	week3 := seedWeek(trainer.ID, program.ID)
	if week3.WeekNumber != 3 {
		t.Fatalf("create week: expected week number 3, got %d", week3.WeekNumber)
	}

	// 2. Creating a week on a foreign program is denied.
	foreignWeek := &models.ProgramWeek{}
	if err := weekRepo.Create(ctx, otherTrainer.ID, program.ID, foreignWeek); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("cross-trainer create: expected ErrProgramNotFound, got %v", err)
	}
	if err := weekRepo.Create(ctx, trainer.ID, foreignProgram.ID, foreignWeek); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("create on foreign program: expected ErrProgramNotFound, got %v", err)
	}
	if err := weekRepo.Create(ctx, trainer.ID, "00000000-0000-0000-0000-000000000000", foreignWeek); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("create on unknown program: expected ErrProgramNotFound, got %v", err)
	}

	// 3. List returns only the trainer's program's weeks in order.
	weeks, err := weekRepo.ListByProgram(ctx, trainer.ID, program.ID)
	if err != nil {
		t.Fatalf("list weeks: %v", err)
	}
	if len(weeks) != 3 {
		t.Fatalf("list weeks: expected 3 weeks, got %d", len(weeks))
	}
	if weeks[0].ID != week1.ID || weeks[0].WeekNumber != 1 {
		t.Fatalf("list weeks: expected week 1 first, got %+v", weeks[0])
	}
	if weeks[1].ID != week2.ID || weeks[2].ID != week3.ID {
		t.Fatalf("list weeks: expected creation order, got %q, %q, %q", weeks[0].ID, weeks[1].ID, weeks[2].ID)
	}

	// 4. A foreign or unknown program is indistinguishable from one without
	// weeks.
	weeks, err = weekRepo.ListByProgram(ctx, otherTrainer.ID, program.ID)
	if err != nil {
		t.Fatalf("list weeks cross-trainer: %v", err)
	}
	if len(weeks) != 0 {
		t.Fatalf("cross-trainer list must be empty, got %d", len(weeks))
	}
	weeks, err = weekRepo.ListByProgram(ctx, trainer.ID, "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("list weeks unknown program: %v", err)
	}
	if len(weeks) != 0 {
		t.Fatalf("unknown program list must be empty, got %d", len(weeks))
	}

	// 5. Find returns one of the trainer's own weeks.
	found, err := weekRepo.FindByIDAndProgram(ctx, trainer.ID, program.ID, week2.ID)
	if err != nil {
		t.Fatalf("find week: %v", err)
	}
	if found.ID != week2.ID || found.WeekNumber != 2 || found.ProgramID != program.ID {
		t.Fatalf("unexpected week %+v", found)
	}

	// 6. Owner isolation: a foreign or unknown week is indistinguishable.
	if _, err := weekRepo.FindByIDAndProgram(ctx, otherTrainer.ID, program.ID, week2.ID); !errors.Is(err, repositories.ErrWeekNotFound) {
		t.Fatalf("cross-trainer find: expected ErrWeekNotFound, got %v", err)
	}
	if _, err := weekRepo.FindByIDAndProgram(ctx, trainer.ID, program.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrWeekNotFound) {
		t.Fatalf("unknown week: expected ErrWeekNotFound, got %v", err)
	}

	// 7. Reorder reassigns week numbers to the given order.
	if err := weekRepo.Reorder(ctx, trainer.ID, program.ID, []string{week3.ID, week1.ID, week2.ID}); err != nil {
		t.Fatalf("reorder weeks: %v", err)
	}
	weeks, err = weekRepo.ListByProgram(ctx, trainer.ID, program.ID)
	if err != nil {
		t.Fatalf("list weeks after reorder: %v", err)
	}
	if weeks[0].ID != week3.ID || weeks[0].WeekNumber != 1 {
		t.Fatalf("reorder: expected week3 first, got %+v", weeks[0])
	}
	if weeks[1].ID != week1.ID || weeks[2].ID != week2.ID {
		t.Fatalf("reorder: expected order week3, week1, week2, got %q, %q, %q", weeks[0].ID, weeks[1].ID, weeks[2].ID)
	}

	// 8. Reorder with a wrong list is rejected without touching the order.
	for name, ids := range map[string][]string{
		"missing":      {week1.ID, week2.ID},
		"extra":        {week1.ID, week2.ID, week3.ID, "00000000-0000-0000-0000-000000000000"},
		"duplicate":    {week1.ID, week1.ID, week3.ID},
		"empty":        {},
		"unknown only": {"00000000-0000-0000-0000-000000000000", "00000000-0000-0000-0000-000000000001"},
	} {
		if err := weekRepo.Reorder(ctx, trainer.ID, program.ID, ids); !errors.Is(err, repositories.ErrWeekReorderConflict) {
			t.Fatalf("reorder %s: expected ErrWeekReorderConflict, got %v", name, err)
		}
	}
	weeks, err = weekRepo.ListByProgram(ctx, trainer.ID, program.ID)
	if err != nil {
		t.Fatalf("list weeks after rejected reorder: %v", err)
	}
	if weeks[0].ID != week3.ID || weeks[1].ID != week1.ID || weeks[2].ID != week2.ID {
		t.Fatalf("rejected reorder must never change the order, got %+v", weeks)
	}

	// 9. Reordering a foreign program is denied before any write.
	if err := weekRepo.Reorder(ctx, otherTrainer.ID, program.ID, []string{week3.ID, week1.ID, week2.ID}); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("cross-trainer reorder: expected ErrProgramNotFound, got %v", err)
	}

	// 10. Soft delete removes the week from find and list but keeps the row.
	if err := weekRepo.SoftDelete(ctx, trainer.ID, program.ID, week2.ID); err != nil {
		t.Fatalf("soft delete week: %v", err)
	}
	if _, err := weekRepo.FindByIDAndProgram(ctx, trainer.ID, program.ID, week2.ID); !errors.Is(err, repositories.ErrWeekNotFound) {
		t.Fatalf("soft-deleted week must not be found, got %v", err)
	}
	var deletedRecord models.ProgramWeek
	if err := tx.Unscoped().First(&deletedRecord, "id = ?", week2.ID).Error; err != nil {
		t.Fatalf("soft-deleted week row must be preserved: %v", err)
	}
	if !deletedRecord.DeletedAt.Valid {
		t.Fatal("soft-deleted week must carry a deleted_at marker")
	}
	weeks, err = weekRepo.ListByProgram(ctx, trainer.ID, program.ID)
	if err != nil {
		t.Fatalf("list weeks after delete: %v", err)
	}
	if len(weeks) != 2 {
		t.Fatalf("expected 2 active weeks after delete, got %d", len(weeks))
	}
	for _, w := range weeks {
		if w.ID == week2.ID {
			t.Fatal("soft-deleted week must never appear in the list")
		}
	}

	// 11. Soft deleting a foreign or unknown week maps to not found.
	if err := weekRepo.SoftDelete(ctx, otherTrainer.ID, program.ID, week1.ID); !errors.Is(err, repositories.ErrWeekNotFound) {
		t.Fatalf("cross-trainer soft delete: expected ErrWeekNotFound, got %v", err)
	}
	if err := weekRepo.SoftDelete(ctx, trainer.ID, program.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrWeekNotFound) {
		t.Fatalf("unknown soft delete: expected ErrWeekNotFound, got %v", err)
	}

	// 12. After a soft delete, new weeks reuse the next available number of the
	// remaining active weeks.
	week4 := seedWeek(trainer.ID, program.ID)
	if week4.WeekNumber != 3 {
		t.Fatalf("expected week number 3 after delete, got %d", week4.WeekNumber)
	}

	// 13. Reorder after a soft delete still requires the exact active set.
	if err := weekRepo.Reorder(ctx, trainer.ID, program.ID, []string{week1.ID, week3.ID, week4.ID}); err != nil {
		t.Fatalf("reorder after delete: %v", err)
	}
	weeks, err = weekRepo.ListByProgram(ctx, trainer.ID, program.ID)
	if err != nil {
		t.Fatalf("list weeks after post-delete reorder: %v", err)
	}
	if len(weeks) != 3 || weeks[0].ID != week1.ID || weeks[1].ID != week3.ID || weeks[2].ID != week4.ID {
		t.Fatalf("expected order week1, week3, week4, got %+v", weeks)
	}

	// 14. The database rejects a week referencing an unknown program (FK).
	if err := weekRepo.Create(ctx, trainer.ID, "00000000-0000-0000-0000-000000000000", &models.ProgramWeek{}); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("unknown program create must not reach the database as an owner, got %v", err)
	}
}
