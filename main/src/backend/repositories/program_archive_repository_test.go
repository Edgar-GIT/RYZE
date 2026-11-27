package repositories_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

// Archiving takes a published programme off the catalogue, at most 15 per
// owner, and restoring puts it back (v2.132.0).
func TestProgramArchive(t *testing.T) {
	db := openInvoiceDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	ctx := context.Background()

	user := &models.User{Email: fmt.Sprintf("archive-%d@ryze.local", time.Now().UnixNano()), PasswordHash: "hash", FirstName: "A", LastName: "R"}
	if err := repositories.NewUserRepository(tx).Create(ctx, user); err != nil {
		t.Fatalf("user: %v", err)
	}
	trainer := &models.Trainer{UserID: user.ID}
	if err := repositories.NewTrainerRepository(tx).Create(ctx, trainer); err != nil {
		t.Fatalf("trainer: %v", err)
	}
	programs := repositories.NewProgramRepository(tx)
	newProgram := func(name, status string) *models.Program {
		program := &models.Program{TrainerID: trainer.ID, Name: name, Type: models.ProgramTypeFree, Status: status, Currency: "EUR"}
		if err := programs.Create(ctx, program); err != nil {
			t.Fatalf("program: %v", err)
		}
		return program
	}
	archive := repositories.NewProgramArchiveRepository(tx)
	owner := repositories.ProgramOwner{TrainerID: trainer.ID}

	draft := newProgram("Draft", models.ProgramStatusDraft)
	if _, err := archive.Archive(ctx, owner, draft.ID, time.Now()); !errors.Is(err, repositories.ErrArchiveState) {
		t.Fatalf("a draft cannot be archived: %v", err)
	}

	live := newProgram("Archive probe", models.ProgramStatusPublished)
	if _, err := archive.Archive(ctx, owner, live.ID, time.Now()); err != nil {
		t.Fatalf("archive: %v", err)
	}
	if _, err := archive.Archive(ctx, repositories.ProgramOwner{TrainerID: "someone-else"}, live.ID, time.Now()); err == nil {
		t.Fatalf("only the owner archives")
	}
	listed, _, err := programs.ListPublished(ctx, 1, 1000)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, program := range listed {
		if program.ID == live.ID {
			t.Fatalf("an archived programme must leave the catalogue")
		}
	}
	// It stays readable for the people who have it.
	if _, err := programs.FindByIDWithStructure(ctx, live.ID); err != nil {
		t.Fatalf("an archived programme stays readable: %v", err)
	}

	// The limit: 15 archived per owner.
	for i := 1; i < repositories.MaxArchivedPrograms; i++ {
		extra := newProgram(fmt.Sprintf("Extra %d", i), models.ProgramStatusPublished)
		if _, err := archive.Archive(ctx, owner, extra.ID, time.Now()); err != nil {
			t.Fatalf("archive %d: %v", i, err)
		}
	}
	one := newProgram("One too many", models.ProgramStatusPublished)
	if _, err := archive.Archive(ctx, owner, one.ID, time.Now()); !errors.Is(err, repositories.ErrArchiveLimit) {
		t.Fatalf("the 16th archive must be refused: %v", err)
	}
	if count, err := archive.CountArchived(ctx, trainer.ID); err != nil || count != repositories.MaxArchivedPrograms {
		t.Fatalf("count: %d %v", count, err)
	}

	if _, err := archive.Restore(ctx, owner, live.ID); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if _, err := archive.Restore(ctx, owner, live.ID); !errors.Is(err, repositories.ErrArchiveState) {
		t.Fatalf("restoring twice: %v", err)
	}
}
