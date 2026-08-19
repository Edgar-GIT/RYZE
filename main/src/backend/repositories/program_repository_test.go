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

func TestProgramRepository(t *testing.T) {
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
	ctx := context.Background()

	seedUser := func() *models.User {
		user := &models.User{
			Email:        fmt.Sprintf("program-repo-%d@ryze.local", time.Now().UnixNano()),
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

	seedProgram := func(trainerID string, name string, createdAt time.Time) *models.Program {
		program := &models.Program{
			TrainerID: trainerID,
			Name:      name,
			Type:      models.ProgramTypePremium,
			Status:    models.ProgramStatusDraft,
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		}
		if err := programRepo.Create(ctx, program); err != nil {
			t.Fatalf("create program: %v", err)
		}
		return program
	}

	trainer := seedTrainer()
	otherTrainer := seedTrainer()

	// 1. Create a program owned by a trainer.
	program := seedProgram(trainer.ID, "Strength Builder", time.Now())
	if program.ID == "" {
		t.Fatal("create program: expected generated UUID id")
	}
	if program.CreatedAt.IsZero() || program.UpdatedAt.IsZero() {
		t.Fatal("create program: expected non-zero timestamps")
	}
	if program.TrainerID != trainer.ID {
		t.Fatalf("create program: expected trainer owner %q, got %q", trainer.ID, program.TrainerID)
	}

	// 2. Find one of the trainer's own active programs.
	found, err := programRepo.FindByIDAndTrainer(ctx, trainer.ID, program.ID)
	if err != nil {
		t.Fatalf("find program: %v", err)
	}
	if found.ID != program.ID || found.Name != "Strength Builder" {
		t.Fatalf("unexpected program %+v", found)
	}

	// 3. Owner isolation: another trainer can never find the program.
	if _, err := programRepo.FindByIDAndTrainer(ctx, otherTrainer.ID, program.ID); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("cross-trainer find: expected ErrProgramNotFound, got %v", err)
	}

	// 4. A random program id is indistinguishable from a missing one.
	if _, err := programRepo.FindByIDAndTrainer(ctx, trainer.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("unknown program: expected ErrProgramNotFound, got %v", err)
	}

	// 5. List returns only the trainer's programs, newest first, with totals.
	first := seedProgram(trainer.ID, "Hypertrophy 101", time.Now().Add(-2*time.Second))
	second := seedProgram(trainer.ID, "Conditioning", time.Now().Add(-1*time.Second))
	otherProgram := seedProgram(otherTrainer.ID, "Foreign Program", time.Now())

	programs, total, err := programRepo.ListByTrainer(ctx, trainer.ID, 1, 10)
	if err != nil {
		t.Fatalf("list programs: %v", err)
	}
	if total != 3 {
		t.Fatalf("list programs: expected total 3, got %d", total)
	}
	if len(programs) != 3 {
		t.Fatalf("list programs: expected 3 programs, got %d", len(programs))
	}
	if programs[0].ID != program.ID {
		t.Fatalf("list programs: expected newest first, got %q", programs[0].ID)
	}
	if programs[1].ID != second.ID || programs[2].ID != first.ID {
		t.Fatalf("list programs: expected creation order, got %q, %q, %q", programs[0].ID, programs[1].ID, programs[2].ID)
	}
	for _, p := range programs {
		if p.ID == otherProgram.ID {
			t.Fatalf("list programs: other trainer's program must never be listed")
		}
		if p.TrainerID != trainer.ID {
			t.Fatalf("list programs: expected only trainer's programs, got %q", p.TrainerID)
		}
	}

	// 6. Pagination returns the requested page.
	programs, total, err = programRepo.ListByTrainer(ctx, trainer.ID, 1, 2)
	if err != nil {
		t.Fatalf("list programs page: %v", err)
	}
	if total != 3 || len(programs) != 2 {
		t.Fatalf("expected page total 3 size 2, got %d/%d", total, len(programs))
	}

	// 7. Update applies whitelisted fields and keeps the owner.
	if err := programRepo.Update(ctx, trainer.ID, program.ID, map[string]any{"name": "Strength Builder v2", "status": models.ProgramStatusPublished}); err != nil {
		t.Fatalf("update program: %v", err)
	}
	updated, err := programRepo.FindByIDAndTrainer(ctx, trainer.ID, program.ID)
	if err != nil {
		t.Fatalf("find updated program: %v", err)
	}
	if updated.Name != "Strength Builder v2" || updated.Status != models.ProgramStatusPublished {
		t.Fatalf("update program: expected new values, got %+v", updated)
	}
	if updated.TrainerID != trainer.ID {
		t.Fatalf("update program: owner must never change, got %q", updated.TrainerID)
	}

	// 8. Updating a foreign program is a no-op that is then reported as not
	// found on read (ownership is scoped in the update itself).
	if err := programRepo.Update(ctx, otherTrainer.ID, program.ID, map[string]any{"name": "Hijacked"}); err != nil {
		t.Fatalf("cross-trainer update must not error: %v", err)
	}
	reloaded, err := programRepo.FindByIDAndTrainer(ctx, trainer.ID, program.ID)
	if err != nil {
		t.Fatalf("find program after cross-trainer update: %v", err)
	}
	if reloaded.Name != "Strength Builder v2" {
		t.Fatalf("cross-trainer update must never change the program, got %q", reloaded.Name)
	}

	// 9. Soft delete removes the program from find and list but keeps the row.
	if err := programRepo.SoftDelete(ctx, trainer.ID, first.ID); err != nil {
		t.Fatalf("soft delete program: %v", err)
	}
	if _, err := programRepo.FindByIDAndTrainer(ctx, trainer.ID, first.ID); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("soft-deleted program must not be found, got %v", err)
	}
	var deletedRecord models.Program
	if err := tx.Unscoped().First(&deletedRecord, "id = ?", first.ID).Error; err != nil {
		t.Fatalf("soft-deleted program row must be preserved: %v", err)
	}
	if !deletedRecord.DeletedAt.Valid {
		t.Fatal("soft-deleted program must carry a deleted_at marker")
	}
	programs, total, err = programRepo.ListByTrainer(ctx, trainer.ID, 1, 10)
	if err != nil {
		t.Fatalf("list programs after delete: %v", err)
	}
	if total != 2 || len(programs) != 2 {
		t.Fatalf("expected 2 active programs after delete, got %d/%d", total, len(programs))
	}
	for _, p := range programs {
		if p.ID == first.ID {
			t.Fatal("soft-deleted program must never appear in the list")
		}
	}

	// 10. Soft deleting a foreign program or an unknown one maps to not found.
	if err := programRepo.SoftDelete(ctx, otherTrainer.ID, program.ID); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("cross-trainer soft delete: expected ErrProgramNotFound, got %v", err)
	}
	if err := programRepo.SoftDelete(ctx, trainer.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("unknown soft delete: expected ErrProgramNotFound, got %v", err)
	}

	// 11. The database rejects an invalid program type (CHECK constraint).
	if err := programRepo.Create(ctx, &models.Program{
		TrainerID: trainer.ID,
		Name:      "Bad Type",
		Type:      "random",
		Status:    models.ProgramStatusDraft,
	}); err == nil {
		t.Fatal("invalid program type: expected a database error")
	}

	// 12. The database rejects an invalid program status (CHECK constraint).
	if err := programRepo.Create(ctx, &models.Program{
		TrainerID: trainer.ID,
		Name:      "Bad Status",
		Type:      models.ProgramTypeFree,
		Status:    "archived",
	}); err == nil {
		t.Fatal("invalid program status: expected a database error")
	}

	// 13. The database rejects a NULL name (NOT NULL constraint). A non-empty
	// name is enforced by the service, following the project pattern where
	// business validation lives outside the repository.
	var blank models.Program
	blank.TrainerID = trainer.ID
	blank.Name = "to-be-cleared"
	blank.Type = models.ProgramTypeFree
	blank.Status = models.ProgramStatusDraft
	if err := programRepo.Create(ctx, &blank); err != nil {
		t.Fatalf("seed blank-name program: %v", err)
	}
	if err := tx.Exec("UPDATE programs SET name = NULL WHERE id = ?", blank.ID).Error; err == nil {
		t.Fatal("NULL program name: expected a database error")
	}
	if err := programRepo.SoftDelete(ctx, trainer.ID, blank.ID); err != nil {
		t.Fatalf("cleanup blank-name program: %v", err)
	}

	// 14. The database rejects a program referencing an unknown trainer (FK).
	if err := programRepo.Create(ctx, &models.Program{
		TrainerID: "00000000-0000-0000-0000-000000000000",
		Name:      "Orphan",
		Type:      models.ProgramTypeFree,
		Status:    models.ProgramStatusDraft,
	}); err == nil {
		t.Fatal("program with unknown trainer: expected a database error")
	}

	// 15. A NULL trainer_id is structurally allowed: it is reserved for future
	// platform-owned programs and can never be produced by the trainer API.
	// The raw insert writes a real NULL, which GORM's zero-value string cannot
	// represent.
	platformProgram := &models.Program{
		Name:   "Platform Free Program",
		Type:   models.ProgramTypeFree,
		Status: models.ProgramStatusDraft,
	}
	if err := tx.Exec(
		"INSERT INTO programs (id, trainer_id, name, type, status) VALUES (?, NULL, ?, ?, ?)",
		platformProgram.ID, platformProgram.Name, platformProgram.Type, platformProgram.Status,
	).Error; err != nil {
		t.Fatalf("platform-owned program (NULL trainer_id) must be allowed: %v", err)
	}

	// 16. Trainer-owned programs and platform-owned programs never collide in
	// trainer lists.
	programs, total, err = programRepo.ListByTrainer(ctx, trainer.ID, 1, 10)
	if err != nil {
		t.Fatalf("list programs isolation: %v", err)
	}
	if total != 2 {
		t.Fatalf("platform-owned program must never leak into a trainer list, got %d", total)
	}
	for _, p := range programs {
		if p.ID == platformProgram.ID {
			t.Fatal("platform-owned program must never leak into a trainer list")
		}
	}

	// 17. Publish transitions a draft program to published.
	publishTarget := seedProgram(trainer.ID, "Publish Target", time.Now())
	if err := programRepo.Publish(ctx, trainer.ID, publishTarget.ID); err != nil {
		t.Fatalf("publish draft program: %v", err)
	}
	published, err := programRepo.FindByIDAndTrainer(ctx, trainer.ID, publishTarget.ID)
	if err != nil {
		t.Fatalf("find published program: %v", err)
	}
	if published.Status != models.ProgramStatusPublished {
		t.Fatalf("expected published status, got %q", published.Status)
	}

	// 18. Publishing an already published program returns ErrProgramNotFound
	// (the conditional WHERE status='draft' does not match).
	if err := programRepo.Publish(ctx, trainer.ID, publishTarget.ID); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("publish already published: expected ErrProgramNotFound, got %v", err)
	}
	// The program must remain published after the failed publish attempt.
	unchanged, err := programRepo.FindByIDAndTrainer(ctx, trainer.ID, publishTarget.ID)
	if err != nil {
		t.Fatalf("find unchanged program: %v", err)
	}
	if unchanged.Status != models.ProgramStatusPublished {
		t.Fatalf("program must remain published after idempotent publish, got %q", unchanged.Status)
	}

	// 19. Publishing a foreign program returns ErrProgramNotFound.
	if err := programRepo.Publish(ctx, otherTrainer.ID, publishTarget.ID); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("publish foreign program: expected ErrProgramNotFound, got %v", err)
	}

	// 20. Publishing an unknown program returns ErrProgramNotFound.
	if err := programRepo.Publish(ctx, trainer.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("publish unknown program: expected ErrProgramNotFound, got %v", err)
	}

	// 21. Publishing a soft-deleted program returns ErrProgramNotFound.
	softDeletedPublish := seedProgram(trainer.ID, "Soft Deleted Publish", time.Now())
	if err := programRepo.SoftDelete(ctx, trainer.ID, softDeletedPublish.ID); err != nil {
		t.Fatalf("soft delete for publish test: %v", err)
	}
	if err := programRepo.Publish(ctx, trainer.ID, softDeletedPublish.ID); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("publish soft-deleted program: expected ErrProgramNotFound, got %v", err)
	}

	// 22. ListPublished returns only published programs from all trainers,
	// ordered newest first. Draft programs and soft-deleted programs are
	// never included.
	_, baselineTotal, err := programRepo.ListPublished(ctx, 1, 1)
	if err != nil {
		t.Fatalf("list published baseline: %v", err)
	}

	pubA := seedProgram(trainer.ID, "Published A", time.Now().Add(-2*time.Second))
	pubA.Status = models.ProgramStatusPublished
	if err := programRepo.Update(ctx, trainer.ID, pubA.ID, map[string]any{"status": models.ProgramStatusPublished}); err != nil {
		t.Fatalf("publish pubA: %v", err)
	}
	pubB := seedProgram(otherTrainer.ID, "Published B", time.Now().Add(-1*time.Second))
	pubB.Status = models.ProgramStatusPublished
	if err := programRepo.Update(ctx, otherTrainer.ID, pubB.ID, map[string]any{"status": models.ProgramStatusPublished}); err != nil {
		t.Fatalf("publish pubB: %v", err)
	}

	publishedPrograms, total, err := programRepo.ListPublished(ctx, 1, 10)
	if err != nil {
		t.Fatalf("list published: %v", err)
	}
	if total != baselineTotal+2 {
		t.Fatalf("list published: expected total %d, got %d", baselineTotal+2, total)
	}
	if len(publishedPrograms) == 0 {
		t.Fatal("list published: expected at least 1 program on first page")
	}
	for _, p := range publishedPrograms {
		if p.Status != models.ProgramStatusPublished {
			t.Fatalf("list published: expected only published programs, got status %q", p.Status)
		}
	}
	pubAIdx, pubBIdx := -1, -1
	for i, p := range publishedPrograms {
		if p.ID == pubA.ID {
			pubAIdx = i
		}
		if p.ID == pubB.ID {
			pubBIdx = i
		}
	}
	if pubAIdx == -1 {
		t.Fatal("list published: pubA not found in results")
	}
	if pubBIdx == -1 {
		t.Fatal("list published: pubB not found in results")
	}
	if pubBIdx >= pubAIdx {
		t.Fatalf("list published: pubB (newer) should appear before pubA, got pubB at %d, pubA at %d", pubBIdx, pubAIdx)
	}

	// 23. ListPublished excludes draft programs.
	draftOnly := seedProgram(trainer.ID, "Draft Hidden", time.Now())
	if draftOnly.Status != models.ProgramStatusDraft {
		t.Fatalf("expected draft status, got %q", draftOnly.Status)
	}
	publishedPrograms, total, err = programRepo.ListPublished(ctx, 1, 10)
	if err != nil {
		t.Fatalf("list published after draft: %v", err)
	}
	if total != baselineTotal+2 {
		t.Fatalf("list published after draft: expected total %d, got %d", baselineTotal+2, total)
	}
	for _, p := range publishedPrograms {
		if p.ID == draftOnly.ID {
			t.Fatal("list published must never include draft programs")
		}
	}

	// 24. ListPublished excludes soft-deleted published programs.
	softDeletedPub := seedProgram(trainer.ID, "Deleted Published", time.Now())
	if err := programRepo.Update(ctx, trainer.ID, softDeletedPub.ID, map[string]any{"status": models.ProgramStatusPublished}); err != nil {
		t.Fatalf("publish softDeletedPub: %v", err)
	}
	if err := programRepo.SoftDelete(ctx, trainer.ID, softDeletedPub.ID); err != nil {
		t.Fatalf("soft delete published program: %v", err)
	}
	publishedPrograms, total, err = programRepo.ListPublished(ctx, 1, 10)
	if err != nil {
		t.Fatalf("list published after soft-delete: %v", err)
	}
	if total != baselineTotal+2 {
		t.Fatalf("list published after soft-delete: expected total %d, got %d", baselineTotal+2, total)
	}
	for _, p := range publishedPrograms {
		if p.ID == softDeletedPub.ID {
			t.Fatal("list published must never include soft-deleted programs")
		}
	}

	// 25. ListPublished pagination works correctly.
	publishedPrograms, total, err = programRepo.ListPublished(ctx, 1, 1)
	if err != nil {
		t.Fatalf("list published page: %v", err)
	}
	if total != baselineTotal+2 || len(publishedPrograms) != 1 {
		t.Fatalf("expected page total %d size 1, got %d/%d", baselineTotal+2, total, len(publishedPrograms))
	}

	// 26. FindPublishedByID returns a published program regardless of
	// trainer ownership.
	found, err = programRepo.FindPublishedByID(ctx, pubA.ID)
	if err != nil {
		t.Fatalf("find published: %v", err)
	}
	if found.ID != pubA.ID || found.Name != "Published A" {
		t.Fatalf("unexpected published program %+v", found)
	}

	// Cross-trainer lookup succeeds: the public catalog is global.
	found, err = programRepo.FindPublishedByID(ctx, pubB.ID)
	if err != nil {
		t.Fatalf("find published cross-trainer: %v", err)
	}
	if found.ID != pubB.ID || found.Name != "Published B" {
		t.Fatalf("unexpected published program %+v", found)
	}

	// 27. FindPublishedByID returns ErrProgramNotFound for a draft program.
	if _, err := programRepo.FindPublishedByID(ctx, draftOnly.ID); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("find published draft: expected ErrProgramNotFound, got %v", err)
	}

	// 28. FindPublishedByID returns ErrProgramNotFound for a soft-deleted
	// program.
	if _, err := programRepo.FindPublishedByID(ctx, softDeletedPub.ID); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("find published soft-deleted: expected ErrProgramNotFound, got %v", err)
	}

	// 29. FindPublishedByID returns ErrProgramNotFound for an unknown id.
	if _, err := programRepo.FindPublishedByID(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("find published unknown: expected ErrProgramNotFound, got %v", err)
	}

	// 30. FindPublishedByID returns ErrProgramNotFound for a platform-owned
	// draft program.
	if _, err := programRepo.FindPublishedByID(ctx, platformProgram.ID); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("find published platform draft: expected ErrProgramNotFound, got %v", err)
	}
}
