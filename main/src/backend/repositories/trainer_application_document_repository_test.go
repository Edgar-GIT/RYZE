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

func TestTrainerApplicationDocumentRepository(t *testing.T) {
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
	applicationRepo := repositories.NewTrainerApplicationRepository(tx)
	documentRepo := repositories.NewTrainerApplicationDocumentRepository(tx)
	decisionRepo := repositories.NewTrainerApplicationDecisionRepository(tx)
	ctx := context.Background()

	user := &models.User{
		Email:        fmt.Sprintf("application-document-repo-%d@ryze.local", time.Now().UnixNano()),
		PasswordHash: "prepared-hash-outside-repository-scope",
		FirstName:    "John",
		LastName:     "Doe",
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	application := &models.TrainerApplication{UserID: user.ID, Status: models.ApplicationStatusPending}
	if err := applicationRepo.Create(ctx, application); err != nil {
		t.Fatalf("create application: %v", err)
	}

	// 1. Create two documents and list them in creation order. Explicit
	// distinct timestamps keep the assertion deterministic when both inserts
	// land inside the same database timestamp tick.
	createdAt := time.Now().UTC().Add(-time.Second)
	first := &models.TrainerApplicationDocument{
		ApplicationID:    application.ID,
		Kind:             models.DocumentKindCertification,
		OriginalFilename: "certification.pdf",
		StorageKey:       application.ID + "/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa.pdf",
		ContentType:      "application/pdf",
		ByteSize:         128,
		SHA256:           fmt.Sprintf("%064d", 0),
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
	}
	if err := documentRepo.Create(ctx, first); err != nil {
		t.Fatalf("create first document: %v", err)
	}
	if first.ID == "" {
		t.Fatal("create document: expected a generated UUID id")
	}
	second := &models.TrainerApplicationDocument{
		ApplicationID:    application.ID,
		Kind:             models.DocumentKindIdentification,
		OriginalFilename: "id.jpg",
		StorageKey:       application.ID + "/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb.jpg",
		ContentType:      "image/jpeg",
		ByteSize:         256,
		SHA256:           fmt.Sprintf("%064d", 1),
		CreatedAt:        createdAt.Add(time.Second),
		UpdatedAt:        createdAt.Add(time.Second),
	}
	if err := documentRepo.Create(ctx, second); err != nil {
		t.Fatalf("create second document: %v", err)
	}

	documents, err := documentRepo.ListByApplication(ctx, application.ID)
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	if len(documents) != 2 || documents[0].ID != first.ID || documents[1].ID != second.ID {
		t.Fatalf("unexpected document list %+v", documents)
	}

	// 2. Find one document by id.
	found, err := documentRepo.FindByID(ctx, first.ID)
	if err != nil {
		t.Fatalf("find document: %v", err)
	}
	if found.Kind != models.DocumentKindCertification || found.ContentType != "application/pdf" || found.ByteSize != 128 {
		t.Fatalf("unexpected document %+v", found)
	}

	// 3. A storage key can never be reused.
	duplicate := &models.TrainerApplicationDocument{
		ApplicationID:    application.ID,
		Kind:             models.DocumentKindOther,
		OriginalFilename: "duplicate.pdf",
		StorageKey:       first.StorageKey,
		ContentType:      "application/pdf",
		ByteSize:         1,
		SHA256:           fmt.Sprintf("%064d", 2),
	}
	if err := documentRepo.Create(ctx, duplicate); err == nil {
		t.Fatal("duplicate storage key: expected a uniqueness error")
	}

	// 4. Soft delete hides the document but preserves the row and the file
	// reference; a second delete is reported as already removed.
	if err := documentRepo.SoftDelete(ctx, first.ID); err != nil {
		t.Fatalf("soft delete document: %v", err)
	}
	if _, err := documentRepo.FindByID(ctx, first.ID); !errors.Is(err, repositories.ErrApplicationDocumentNotFound) {
		t.Fatalf("find after soft delete: expected ErrApplicationDocumentNotFound, got %v", err)
	}
	documents, err = documentRepo.ListByApplication(ctx, application.ID)
	if err != nil {
		t.Fatalf("list after soft delete: %v", err)
	}
	if len(documents) != 1 || documents[0].ID != second.ID {
		t.Fatalf("soft-deleted documents must not be listed, got %+v", documents)
	}
	if err := documentRepo.SoftDelete(ctx, first.ID); !errors.Is(err, repositories.ErrApplicationDocumentAlreadyRemoved) {
		t.Fatalf("second soft delete: expected ErrApplicationDocumentAlreadyRemoved, got %v", err)
	}
	if err := documentRepo.SoftDelete(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repositories.ErrApplicationDocumentNotFound) {
		t.Fatalf("soft delete unknown: expected ErrApplicationDocumentNotFound, got %v", err)
	}

	// 5. Decisions are append-only rows listed per application.
	decision := &models.TrainerApplicationDecision{
		ApplicationID: application.ID,
		AdminID:       "ADMIN_2",
		FromStatus:    models.ApplicationStatusPending,
		ToStatus:      models.ApplicationStatusRejected,
		Reason:        "Documents are not readable.",
	}
	if err := decisionRepo.Create(ctx, decision); err != nil {
		t.Fatalf("create decision: %v", err)
	}
	decisions, err := decisionRepo.ListByApplication(ctx, application.ID)
	if err != nil {
		t.Fatalf("list decisions: %v", err)
	}
	if len(decisions) != 1 || decisions[0].ID != decision.ID || decisions[0].Reason != "Documents are not readable." {
		t.Fatalf("unexpected decisions %+v", decisions)
	}
}
