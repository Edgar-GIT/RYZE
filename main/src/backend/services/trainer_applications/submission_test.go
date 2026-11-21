package trainer_applications_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/trainer_applications"
)

type stubHashes map[string]bool

func (s stubHashes) HashUsedByAnotherUser(_ context.Context, sha256, _ string) (bool, error) {
	return s[sha256], nil
}

type stubUsernames map[string]*models.User

func (s stubUsernames) FindByUsername(_ context.Context, username string) (*models.User, error) {
	if user, ok := s[username]; ok {
		return user, nil
	}
	return nil, repositories.ErrUserNotFound
}

func validForm() trainer_applications.ApplicationForm {
	end := 2023
	return trainer_applications.ApplicationForm{
		FullName: "Rita Costa", BirthDate: "1992-04-15", City: "Lisboa", Profession: "Personal trainer", YearsExperience: 6,
		WorkHistory: []trainer_applications.WorkEntry{{Organization: "Ginásio Central", Role: "Personal trainer", StartYear: 2018, EndYear: &end}},
		Education:   []trainer_applications.EducationEntry{{Institution: "FMH", Qualification: "Licenciatura em Desporto", Year: 2017}},
		About:       "Treino força e condição física há seis anos, com foco em iniciantes e reabilitação de lesões ligeiras.",
	}
}

func pdf() []byte {
	return []byte("%PDF-1.4\n% " + uuid.NewString() + "\n1 0 obj << /Producer (Scanner) >>\nendobj\n%%EOF\n")
}

func validDocuments() []trainer_applications.SubmittedDocument {
	return []trainer_applications.SubmittedDocument{
		{Kind: models.DocumentKindCertification, Filename: "cert.pdf", Data: pdf()},
		{Kind: models.DocumentKindIdentification, Filename: "id.pdf", Data: pdf()},
	}
}

func newSubmissionService(apps *stubApplicationRepo, documents *stubDocumentRepo, evidence *stubEvidenceStore, userID string, opts ...trainer_applications.Option) trainer_applications.Service {
	users := stubUserRepo{users: []models.User{{ID: userID}}}
	return trainer_applications.NewService(apps, users, stubTrainerRepo{}, documents, &stubDecisionRepo{}, evidence, opts...)
}

func TestSubmitApplicationStoresDetailsAndScreenedDocuments(t *testing.T) {
	userID := uuid.NewString()
	apps, documents, evidence := &stubApplicationRepo{}, &stubDocumentRepo{}, &stubEvidenceStore{}
	svc := newSubmissionService(apps, documents, evidence, userID)

	application, err := svc.SubmitApplication(context.Background(), userID, validForm(), validDocuments())
	if err != nil {
		t.Fatalf("SubmitApplication: %v", err)
	}
	if application.Status != models.ApplicationStatusPending || application.FullName == nil || *application.FullName != "Rita Costa" || application.BirthDate == nil {
		t.Fatalf("unexpected application: %+v", application)
	}
	if evidence.saves != 2 || len(documents.documents) != 2 || documents.documents[0].ScreeningStatus == "" {
		t.Fatalf("expected two screened documents, got %d saves and %+v", evidence.saves, documents.documents)
	}
}

func TestSubmitApplicationRejectsBadInputBeforeStoringAnything(t *testing.T) {
	userID := uuid.NewString()
	apps, documents, evidence := &stubApplicationRepo{}, &stubDocumentRepo{}, &stubEvidenceStore{}
	svc := newSubmissionService(apps, documents, evidence, userID, trainer_applications.WithDocumentHashes(stubHashes{}))
	ctx := context.Background()

	form := validForm()
	form.BirthDate = "2015-01-01" // a minor
	form.About = "too short"
	var formErr *trainer_applications.FormError
	if _, err := svc.SubmitApplication(ctx, userID, form, validDocuments()); !errors.As(err, &formErr) || len(formErr.Fields) != 2 {
		t.Fatalf("expected the two invalid fields, got %v", err)
	}

	onlyCert := validDocuments()[:1]
	if _, err := svc.SubmitApplication(ctx, userID, validForm(), onlyCert); !errors.Is(err, trainer_applications.ErrMissingDocuments) {
		t.Fatalf("an identification document is required, got %v", err)
	}

	generated := validDocuments()
	generated[0].Data = []byte("%PDF-1.7\n1 0 obj << /Producer (ChatGPT export) >>\nendobj\n%%EOF\n")
	var rejected *trainer_applications.DocumentsRejectedError
	if _, err := svc.SubmitApplication(ctx, userID, validForm(), generated); !errors.As(err, &rejected) || rejected.Rejections[0].Filename != "cert.pdf" {
		t.Fatalf("an AI-made document must be rejected, got %v", err)
	}
	if len(apps.apps) != 0 || evidence.saves != 0 {
		t.Fatalf("nothing may be stored after a refusal: %d applications, %d files", len(apps.apps), evidence.saves)
	}
}

func TestSubmitApplicationRejectsADocumentFromAnotherAccount(t *testing.T) {
	userID := uuid.NewString()
	documents := validDocuments()
	hashes := stubHashes{}
	// Mark the certification's hash as already used by someone else.
	svc := newSubmissionService(&stubApplicationRepo{}, &stubDocumentRepo{}, &stubEvidenceStore{}, userID, trainer_applications.WithDocumentHashes(hashOf(documents[0].Data, hashes)))
	var rejected *trainer_applications.DocumentsRejectedError
	if _, err := svc.SubmitApplication(context.Background(), userID, validForm(), documents); !errors.As(err, &rejected) || rejected.Rejections[0].Reasons[0] != trainer_applications.ReasonDuplicateDocument {
		t.Fatalf("a reused document must be rejected, got %v", err)
	}
}

func TestPromoteUserByUsername(t *testing.T) {
	user := &models.User{ID: uuid.NewString(), Username: "teste2"}
	apps := &stubApplicationRepo{}
	users := stubUserRepo{users: []models.User{*user}}
	svc := trainer_applications.NewService(apps, users, stubTrainerRepo{}, &stubDocumentRepo{}, &stubDecisionRepo{}, &stubEvidenceStore{},
		trainer_applications.WithUsernames(stubUsernames{"teste2": user}))

	approved, err := svc.PromoteUser(context.Background(), "@Teste2", "ADMIN_1", "")
	if err != nil {
		t.Fatalf("PromoteUser: %v", err)
	}
	if approved.Status != models.ApplicationStatusApproved || approved.UserID != user.ID || len(apps.decisions) != 1 {
		t.Fatalf("expected an approved application with a recorded decision: %+v", approved)
	}
	if _, err := svc.PromoteUser(context.Background(), "nobody", "ADMIN_1", ""); !errors.Is(err, trainer_applications.ErrUserNotFound) {
		t.Fatalf("an unknown username must be not found, got %v", err)
	}
}

func hashOf(data []byte, hashes stubHashes) stubHashes {
	sum := sha256.Sum256(data)
	hashes[hex.EncodeToString(sum[:])] = true
	return hashes
}
