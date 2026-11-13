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

func TestCoachShowcaseRepository(t *testing.T) {
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
	users := repositories.NewUserRepository(tx)
	trainers := repositories.NewTrainerRepository(tx)
	programs := repositories.NewProgramRepository(tx)
	showcase := repositories.NewCoachShowcaseRepository(tx)

	seedCoach := func(first string) (*models.User, *models.Trainer) {
		user := &models.User{
			Email:        fmt.Sprintf("showcase-%s-%d@ryze.local", first, time.Now().UnixNano()),
			PasswordHash: "prepared-hash-outside-repository-scope",
			FirstName:    first,
			LastName:     "Coach",
		}
		if err := users.Create(ctx, user); err != nil {
			t.Fatalf("create user: %v", err)
		}
		trainer := &models.Trainer{UserID: user.ID}
		if err := trainers.Create(ctx, trainer); err != nil {
			t.Fatalf("create trainer: %v", err)
		}
		return user, trainer
	}
	seedProgram := func(trainerID, name, status string, createdAt time.Time) *models.Program {
		program := &models.Program{
			TrainerID: trainerID,
			Name:      name,
			Type:      models.ProgramTypeFree,
			Status:    status,
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		}
		if err := programs.Create(ctx, program); err != nil {
			t.Fatalf("create program: %v", err)
		}
		return program
	}

	ownerUser, coach := seedCoach("Ana")
	_, quietCoach := seedCoach("Rui")
	personaUser, personaCoach := seedCoach("Persona")
	if err := tx.Model(&models.User{}).Where("id = ?", personaUser.ID).Update("test_persona", "trainer").Error; err != nil {
		t.Fatalf("mark persona: %v", err)
	}

	base := time.Now().Add(-time.Hour)
	popular := seedProgram(coach.ID, "Popular", models.ProgramStatusPublished, base)
	fresh := seedProgram(coach.ID, "Fresh", models.ProgramStatusPublished, base.Add(time.Minute))
	draft := seedProgram(coach.ID, "Draft", models.ProgramStatusDraft, base)
	seedProgram(quietCoach.ID, "Unpublished", models.ProgramStatusDraft, base)
	persona := seedProgram(personaCoach.ID, "Test content", models.ProgramStatusPublished, base)

	// The view target names the owner of a published program and hides drafts.
	target, err := showcase.FindProgramViewTarget(ctx, popular.ID)
	if err != nil || target.OwnerUserID != ownerUser.ID {
		t.Fatalf("FindProgramViewTarget = %+v, %v", target, err)
	}
	if _, err := showcase.FindProgramViewTarget(ctx, draft.ID); !errors.Is(err, repositories.ErrProgramNotFound) {
		t.Fatalf("a draft must not be viewable: %v", err)
	}

	for i := 0; i < 3; i++ {
		if err := showcase.RecordView(ctx, popular.ID, time.Now()); err != nil {
			t.Fatalf("record view: %v", err)
		}
	}
	if err := showcase.RecordView(ctx, persona.ID, time.Now()); err != nil {
		t.Fatalf("record persona view: %v", err)
	}

	featured, err := showcase.ListFeaturedCoachPrograms(ctx, 50)
	if err != nil {
		t.Fatalf("featured: %v", err)
	}
	position := map[string]int{}
	for i, row := range featured {
		position[row.ID] = i
		if row.ID == persona.ID || row.ID == draft.ID {
			t.Fatalf("featured must exclude drafts and Test Mode personas: %+v", row)
		}
	}
	popularAt, okPopular := position[popular.ID]
	freshAt, okFresh := position[fresh.ID]
	if !okPopular || !okFresh || popularAt > freshAt {
		t.Fatalf("most viewed must rank first: %+v", featured)
	}
	if row := featured[popularAt]; row.ViewCount != 3 || row.CoachID != coach.ID || row.CoachFirstName != "Ana" {
		t.Fatalf("unexpected featured row: %+v", row)
	}

	if found, err := showcase.FindPublicCoach(ctx, coach.ID); err != nil || found.FirstName != "Ana" {
		t.Fatalf("FindPublicCoach = %+v, %v", found, err)
	}
	for _, hidden := range []string{quietCoach.ID, personaCoach.ID} {
		if _, err := showcase.FindPublicCoach(ctx, hidden); !errors.Is(err, repositories.ErrCoachNotFound) {
			t.Fatalf("coach %s must not be public: %v", hidden, err)
		}
	}
	coachPrograms, err := showcase.ListCoachPrograms(ctx, coach.ID, 50)
	if err != nil || len(coachPrograms) != 2 || coachPrograms[0].ID != popular.ID {
		t.Fatalf("ListCoachPrograms = %+v, %v", coachPrograms, err)
	}
}

func TestAdminAuditRepositoryIsAppendOnly(t *testing.T) {
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
	repo := repositories.NewAdminAuditRepository(tx)
	target := "target-1"
	first := &models.AdminAuditLog{AdminID: config.Admin1ID, Action: "user.disable", Method: "PATCH", Route: "/r/:id", TargetID: &target, StatusCode: 200, Outcome: models.AdminAuditOutcomeSuccess, CreatedAt: time.Now().Add(-time.Minute)}
	second := &models.AdminAuditLog{AdminID: config.Admin2ID, Action: "feedback.approve", Method: "POST", Route: "/f/:id", StatusCode: 403, Outcome: models.AdminAuditOutcomeDenied, CreatedAt: time.Now()}
	for _, entry := range []*models.AdminAuditLog{first, second} {
		if err := repo.Create(ctx, entry); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	denied, total, err := repo.List(ctx, repositories.AdminAuditFilter{Outcome: models.AdminAuditOutcomeDenied}, 1, 10)
	if err != nil || total < 1 {
		t.Fatalf("List = %d, %v", total, err)
	}
	for _, entry := range denied {
		if entry.Outcome != models.AdminAuditOutcomeDenied {
			t.Fatalf("filter leaked %+v", entry)
		}
	}
	page, _, err := repo.List(ctx, repositories.AdminAuditFilter{}, 1, 1)
	if err != nil || len(page) != 1 || page[0].ID != second.ID {
		t.Fatalf("newest first: %+v, %v", page, err)
	}

	// The table rejects edits and deletions.
	if err := tx.SavePoint("audit_update").Error; err != nil {
		t.Fatalf("savepoint: %v", err)
	}
	if err := tx.Model(&models.AdminAuditLog{}).Where("id = ?", first.ID).Update("action", "rewritten").Error; err == nil {
		t.Fatal("an audit record must not be editable")
	}
	tx.RollbackTo("audit_update")
	if err := tx.SavePoint("audit_delete").Error; err != nil {
		t.Fatalf("savepoint: %v", err)
	}
	if err := tx.Exec("DELETE FROM admin_audit_logs WHERE id = ?", first.ID).Error; err == nil {
		t.Fatal("an audit record must not be deletable")
	}
	tx.RollbackTo("audit_delete")
}
