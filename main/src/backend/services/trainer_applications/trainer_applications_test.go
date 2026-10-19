package trainer_applications_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/trainer_applications"
	"ryze/backend/services/trainer_evidence"
)

const (
	activeUserID       = "11111111-1111-1111-1111-111111111111"
	unknownUserID      = "22222222-2222-2222-2222-222222222222"
	activeApplication  = "33333333-3333-3333-3333-333333333333"
	missingApplication = "44444444-4444-4444-4444-444444444444"
	activeDocument     = "55555555-5555-5555-5555-555555555555"
	missingDocument    = "66666666-6666-6666-6666-666666666666"
	reviewerAdminID    = "ADMIN_2"
)

var errRepoFailure = errors.New("repository failure")

// stubApplicationRepo is a stateful in-memory fake of the application
// data-access surface. It mirrors the repository semantics: only PENDING or
// APPROVED applications are considered active, a user can hold at most one
// active application, and Approve/Reject only transition PENDING applications
// while recording the reviewer decision.
type stubApplicationRepo struct {
	apps       []models.TrainerApplication
	decisions  []models.TrainerApplicationDecision
	createErr  error
	findErr    error
	activeErr  error
	listErr    error
	approveErr error
	rejectErr  error
}

func (s *stubApplicationRepo) Create(_ context.Context, application *models.TrainerApplication) error {
	if s.createErr != nil {
		return s.createErr
	}
	for i := range s.apps {
		if s.apps[i].UserID == application.UserID && isActiveStatus(s.apps[i].Status) {
			return repositories.ErrApplicationAlreadyActive
		}
	}
	application.ID = uuid.NewString()
	s.apps = append(s.apps, *application)
	return nil
}

func (s *stubApplicationRepo) FindActiveByUserID(_ context.Context, userID string) (*models.TrainerApplication, error) {
	if s.activeErr != nil {
		return nil, s.activeErr
	}
	for i := range s.apps {
		if s.apps[i].UserID == userID && isActiveStatus(s.apps[i].Status) {
			return &s.apps[i], nil
		}
	}
	return nil, repositories.ErrApplicationNotFound
}

func (s *stubApplicationRepo) FindLatestByUserID(_ context.Context, userID string) (*models.TrainerApplication, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	latest := -1
	for i := range s.apps {
		if s.apps[i].UserID != userID {
			continue
		}
		if latest == -1 || s.apps[i].CreatedAt.After(s.apps[latest].CreatedAt) {
			latest = i
		}
	}
	if latest == -1 {
		return nil, repositories.ErrApplicationNotFound
	}
	return &s.apps[latest], nil
}

func (s *stubApplicationRepo) FindByID(_ context.Context, id string) (*models.TrainerApplication, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	for i := range s.apps {
		if s.apps[i].ID == id {
			return &s.apps[i], nil
		}
	}
	return nil, repositories.ErrApplicationNotFound
}

func (s *stubApplicationRepo) List(_ context.Context, _, _ int, status string) ([]models.TrainerApplication, int64, error) {
	if s.listErr != nil {
		return nil, 0, s.listErr
	}
	matched := make([]models.TrainerApplication, 0)
	for _, app := range s.apps {
		if status == "" || app.Status == status {
			matched = append(matched, app)
		}
	}
	return matched, int64(len(matched)), nil
}

func (s *stubApplicationRepo) Approve(_ context.Context, id string, decision *models.TrainerApplicationDecision) (*models.TrainerApplication, error) {
	if s.approveErr != nil {
		return nil, s.approveErr
	}
	app, err := s.FindByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	if app.Status != models.ApplicationStatusPending {
		return nil, repositories.ErrApplicationStateConflict
	}
	app.Status = models.ApplicationStatusApproved
	s.recordDecision(app.ID, decision)
	return app, nil
}

func (s *stubApplicationRepo) Reject(_ context.Context, id string, decision *models.TrainerApplicationDecision) error {
	if s.rejectErr != nil {
		return s.rejectErr
	}
	app, err := s.FindByID(context.Background(), id)
	if err != nil {
		return err
	}
	if app.Status != models.ApplicationStatusPending {
		return repositories.ErrApplicationStateConflict
	}
	app.Status = models.ApplicationStatusRejected
	s.recordDecision(app.ID, decision)
	return nil
}

func (s *stubApplicationRepo) recordDecision(applicationID string, decision *models.TrainerApplicationDecision) {
	if decision == nil {
		return
	}
	recorded := *decision
	recorded.ApplicationID = applicationID
	if recorded.ID == "" {
		recorded.ID = uuid.NewString()
	}
	s.decisions = append(s.decisions, recorded)
}

func isActiveStatus(status string) bool {
	return status == models.ApplicationStatusPending || status == models.ApplicationStatusApproved
}

// stubUserRepo simulates the user data-access surface: only active users are
// found.
type stubUserRepo struct {
	users []models.User
	err   error
}

func (s stubUserRepo) FindByID(_ context.Context, id string) (*models.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	for i := range s.users {
		if s.users[i].ID == id && !s.users[i].DeletedAt.Valid {
			return &s.users[i], nil
		}
	}
	return nil, repositories.ErrUserNotFound
}

// stubTrainerRepo simulates the trainer data-access surface: only active
// trainer profiles are found.
type stubTrainerRepo struct {
	trainers []models.Trainer
	err      error
}

func (s stubTrainerRepo) FindByUserID(_ context.Context, userID string) (*models.Trainer, error) {
	if s.err != nil {
		return nil, s.err
	}
	for i := range s.trainers {
		if s.trainers[i].UserID == userID && !s.trainers[i].DeletedAt.Valid {
			return &s.trainers[i], nil
		}
	}
	return nil, repositories.ErrTrainerNotFound
}

// stubDocumentRepo simulates the document metadata data-access surface,
// including soft deletion semantics.
type stubDocumentRepo struct {
	documents []models.TrainerApplicationDocument
	createErr error
	findErr   error
	listErr   error
	deleteErr error
}

func (s *stubDocumentRepo) Create(_ context.Context, document *models.TrainerApplicationDocument) error {
	if s.createErr != nil {
		return s.createErr
	}
	if document.ID == "" {
		document.ID = uuid.NewString()
	}
	s.documents = append(s.documents, *document)
	return nil
}

func (s *stubDocumentRepo) ListByApplication(_ context.Context, applicationID string) ([]models.TrainerApplicationDocument, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	matched := make([]models.TrainerApplicationDocument, 0)
	for _, document := range s.documents {
		if document.ApplicationID == applicationID && !document.DeletedAt.Valid {
			matched = append(matched, document)
		}
	}
	return matched, nil
}

func (s *stubDocumentRepo) FindByID(_ context.Context, documentID string) (*models.TrainerApplicationDocument, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	for i := range s.documents {
		if s.documents[i].ID == documentID && !s.documents[i].DeletedAt.Valid {
			return &s.documents[i], nil
		}
	}
	return nil, repositories.ErrApplicationDocumentNotFound
}

func (s *stubDocumentRepo) SoftDelete(_ context.Context, documentID string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	for i := range s.documents {
		if s.documents[i].ID != documentID {
			continue
		}
		if s.documents[i].DeletedAt.Valid {
			return repositories.ErrApplicationDocumentAlreadyRemoved
		}
		s.documents[i].DeletedAt.Valid = true
		return nil
	}
	return repositories.ErrApplicationDocumentNotFound
}

// stubDecisionRepo simulates the append-only decision read surface.
type stubDecisionRepo struct {
	decisions []models.TrainerApplicationDecision
	listErr   error
}

func (s *stubDecisionRepo) ListByApplication(_ context.Context, applicationID string) ([]models.TrainerApplicationDecision, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	matched := make([]models.TrainerApplicationDecision, 0)
	for _, decision := range s.decisions {
		if decision.ApplicationID == applicationID {
			matched = append(matched, decision)
		}
	}
	return matched, nil
}

// stubEvidenceStore simulates the private evidence store. It records how many
// times Save was called so tests can assert that a rejected upload never wrote
// anything, and can be configured to fail with any store-level error.
type stubEvidenceStore struct {
	saves    int
	savedKey string
	saved    trainer_evidence.SavedFile
	saveErr  error
	content  string
	openErr  error
}

func (s *stubEvidenceStore) Save(_ context.Context, prefix string, content io.Reader) (string, trainer_evidence.SavedFile, error) {
	s.saves++
	if s.saveErr != nil {
		return "", trainer_evidence.SavedFile{}, s.saveErr
	}
	if _, err := io.ReadAll(content); err != nil {
		return "", trainer_evidence.SavedFile{}, err
	}
	key := s.savedKey
	if key == "" {
		key = prefix + "/" + activeDocument + ".pdf"
	}
	saved := s.saved
	if saved.ContentType == "" {
		saved = trainer_evidence.SavedFile{ByteSize: 6, ContentType: "application/pdf", SHA256: strings.Repeat("a", 64)}
	}
	return key, saved, nil
}

func (s *stubEvidenceStore) Open(_ context.Context, _ string) (io.ReadCloser, error) {
	if s.openErr != nil {
		return nil, s.openErr
	}
	return io.NopCloser(strings.NewReader(s.content)), nil
}

func newTestService(apps *stubApplicationRepo, users stubUserRepo, trainers stubTrainerRepo) trainer_applications.Service {
	return trainer_applications.NewService(apps, users, trainers, &stubDocumentRepo{}, &stubDecisionRepo{}, &stubEvidenceStore{})
}

func newTestServiceWithEvidence(
	apps *stubApplicationRepo,
	users stubUserRepo,
	trainers stubTrainerRepo,
	documents *stubDocumentRepo,
	decisions *stubDecisionRepo,
	evidence *stubEvidenceStore,
) trainer_applications.Service {
	return trainer_applications.NewService(apps, users, trainers, documents, decisions, evidence)
}

func activeUser() models.User {
	return models.User{ID: activeUserID}
}

func testApp(status string) models.TrainerApplication {
	return models.TrainerApplication{ID: activeApplication, UserID: activeUserID, Status: status}
}

func TestApplySuccess(t *testing.T) {
	service := newTestService(&stubApplicationRepo{}, stubUserRepo{users: []models.User{activeUser()}}, stubTrainerRepo{})

	application, err := service.Apply(context.Background(), activeUserID)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if application.Status != models.ApplicationStatusPending {
		t.Fatalf("expected PENDING, got %q", application.Status)
	}
	if application.UserID != activeUserID {
		t.Fatalf("expected user %q, got %q", activeUserID, application.UserID)
	}
}

func TestApplyInvalidUserID(t *testing.T) {
	service := newTestService(&stubApplicationRepo{}, stubUserRepo{users: []models.User{activeUser()}}, stubTrainerRepo{})

	for _, id := range []string{"", "not-a-uuid"} {
		if _, err := service.Apply(context.Background(), id); !errors.Is(err, trainer_applications.ErrInvalidInput) {
			t.Fatalf("Apply(%q): expected ErrInvalidInput, got %v", id, err)
		}
	}
}

func TestApplyUnknownUser(t *testing.T) {
	service := newTestService(&stubApplicationRepo{}, stubUserRepo{}, stubTrainerRepo{})

	if _, err := service.Apply(context.Background(), unknownUserID); !errors.Is(err, trainer_applications.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestApplyAlreadyTrainer(t *testing.T) {
	service := newTestService(
		&stubApplicationRepo{},
		stubUserRepo{users: []models.User{activeUser()}},
		stubTrainerRepo{trainers: []models.Trainer{{UserID: activeUserID}}},
	)

	if _, err := service.Apply(context.Background(), activeUserID); !errors.Is(err, trainer_applications.ErrAlreadyTrainer) {
		t.Fatalf("expected ErrAlreadyTrainer, got %v", err)
	}
}

func TestApplyAlreadyActiveApplication(t *testing.T) {
	service := newTestService(
		&stubApplicationRepo{apps: []models.TrainerApplication{testApp(models.ApplicationStatusPending)}},
		stubUserRepo{users: []models.User{activeUser()}},
		stubTrainerRepo{},
	)

	if _, err := service.Apply(context.Background(), activeUserID); !errors.Is(err, trainer_applications.ErrApplicationAlreadyActive) {
		t.Fatalf("expected ErrApplicationAlreadyActive, got %v", err)
	}
}

func TestApplyRepositoryError(t *testing.T) {
	service := newTestService(
		&stubApplicationRepo{createErr: errRepoFailure},
		stubUserRepo{users: []models.User{activeUser()}},
		stubTrainerRepo{},
	)

	if _, err := service.Apply(context.Background(), activeUserID); err == nil || errors.Is(err, trainer_applications.ErrApplicationAlreadyActive) {
		t.Fatalf("expected a wrapped repository error, got %v", err)
	}
}

func TestListApplicationsValidatesInput(t *testing.T) {
	service := newTestService(&stubApplicationRepo{}, stubUserRepo{}, stubTrainerRepo{})

	for _, tc := range []struct {
		page   int
		limit  int
		status string
	}{
		{page: 0, limit: 20},
		{page: -1, limit: 20},
		{page: 1, limit: 0},
		{page: 1, limit: -3},
		{page: 1, limit: 20, status: "DELETED"},
	} {
		if _, err := service.ListApplications(context.Background(), tc.page, tc.limit, tc.status); !errors.Is(err, trainer_applications.ErrInvalidInput) {
			t.Fatalf("ListApplications(page=%d, limit=%d, status=%q): expected ErrInvalidInput, got %v", tc.page, tc.limit, tc.status, err)
		}
	}
}

func TestListApplicationsClampsLimit(t *testing.T) {
	service := newTestService(&stubApplicationRepo{}, stubUserRepo{}, stubTrainerRepo{})

	result, err := service.ListApplications(context.Background(), 1, 9999, "")
	if err != nil {
		t.Fatalf("ListApplications: %v", err)
	}
	if result.Limit != trainer_applications.MaxPageSize {
		t.Fatalf("expected clamped limit %d, got %d", trainer_applications.MaxPageSize, result.Limit)
	}
}

func TestGetApplication(t *testing.T) {
	service := newTestService(
		&stubApplicationRepo{apps: []models.TrainerApplication{testApp(models.ApplicationStatusPending)}},
		stubUserRepo{},
		stubTrainerRepo{},
	)

	application, err := service.GetApplication(context.Background(), activeApplication)
	if err != nil {
		t.Fatalf("GetApplication: %v", err)
	}
	if application.Status != models.ApplicationStatusPending {
		t.Fatalf("expected PENDING, got %q", application.Status)
	}

	if _, err := service.GetApplication(context.Background(), missingApplication); !errors.Is(err, trainer_applications.ErrApplicationNotFound) {
		t.Fatalf("expected ErrApplicationNotFound, got %v", err)
	}
	if _, err := service.GetApplication(context.Background(), "not-a-uuid"); !errors.Is(err, trainer_applications.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestApproveApplicationCreatesApproval(t *testing.T) {
	apps := &stubApplicationRepo{apps: []models.TrainerApplication{testApp(models.ApplicationStatusPending)}}
	service := newTestService(apps, stubUserRepo{users: []models.User{activeUser()}}, stubTrainerRepo{})

	application, err := service.ApproveApplication(context.Background(), activeApplication, reviewerAdminID, "Documents verified.")
	if err != nil {
		t.Fatalf("ApproveApplication: %v", err)
	}
	if application.Status != models.ApplicationStatusApproved {
		t.Fatalf("expected APPROVED, got %q", application.Status)
	}
	if len(apps.decisions) != 1 {
		t.Fatalf("expected one recorded decision, got %d", len(apps.decisions))
	}
	decision := apps.decisions[0]
	if decision.ApplicationID != activeApplication || decision.AdminID != reviewerAdminID ||
		decision.FromStatus != models.ApplicationStatusPending || decision.ToStatus != models.ApplicationStatusApproved ||
		decision.Reason != "Documents verified." {
		t.Fatalf("unexpected recorded decision %+v", decision)
	}
}

func TestApproveApplicationRejectsInvalidReviewer(t *testing.T) {
	apps := &stubApplicationRepo{apps: []models.TrainerApplication{testApp(models.ApplicationStatusPending)}}
	service := newTestService(apps, stubUserRepo{users: []models.User{activeUser()}}, stubTrainerRepo{})

	if _, err := service.ApproveApplication(context.Background(), activeApplication, "  ", ""); !errors.Is(err, trainer_applications.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for a blank reviewer, got %v", err)
	}
	longReason := strings.Repeat("x", 501)
	if _, err := service.ApproveApplication(context.Background(), activeApplication, reviewerAdminID, longReason); !errors.Is(err, trainer_applications.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for an oversized reason, got %v", err)
	}
	if len(apps.decisions) != 0 {
		t.Fatalf("invalid reviews must not record a decision, got %d", len(apps.decisions))
	}
	application, err := service.GetApplication(context.Background(), activeApplication)
	if err != nil {
		t.Fatalf("GetApplication: %v", err)
	}
	if application.Status != models.ApplicationStatusPending {
		t.Fatalf("invalid reviews must not change the status, got %q", application.Status)
	}
}

func TestApproveApplicationNotFound(t *testing.T) {
	service := newTestService(&stubApplicationRepo{}, stubUserRepo{}, stubTrainerRepo{})

	if _, err := service.ApproveApplication(context.Background(), missingApplication, reviewerAdminID, ""); !errors.Is(err, trainer_applications.ErrApplicationNotFound) {
		t.Fatalf("expected ErrApplicationNotFound, got %v", err)
	}
}

func TestApproveApplicationNotPending(t *testing.T) {
	for _, status := range []string{models.ApplicationStatusApproved, models.ApplicationStatusRejected} {
		service := newTestService(
			&stubApplicationRepo{apps: []models.TrainerApplication{testApp(status)}},
			stubUserRepo{users: []models.User{activeUser()}},
			stubTrainerRepo{},
		)

		if _, err := service.ApproveApplication(context.Background(), activeApplication, reviewerAdminID, ""); !errors.Is(err, trainer_applications.ErrApplicationStateConflict) {
			t.Fatalf("status %q: expected ErrApplicationStateConflict, got %v", status, err)
		}
	}
}

func TestApproveApplicationUserInactive(t *testing.T) {
	service := newTestService(
		&stubApplicationRepo{apps: []models.TrainerApplication{testApp(models.ApplicationStatusPending)}},
		stubUserRepo{},
		stubTrainerRepo{},
	)

	if _, err := service.ApproveApplication(context.Background(), activeApplication, reviewerAdminID, ""); !errors.Is(err, trainer_applications.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestApproveApplicationAlreadyTrainer(t *testing.T) {
	service := newTestService(
		&stubApplicationRepo{
			apps:       []models.TrainerApplication{testApp(models.ApplicationStatusPending)},
			approveErr: repositories.ErrTrainerAlreadyLinked,
		},
		stubUserRepo{users: []models.User{activeUser()}},
		stubTrainerRepo{},
	)

	if _, err := service.ApproveApplication(context.Background(), activeApplication, reviewerAdminID, ""); !errors.Is(err, trainer_applications.ErrAlreadyTrainer) {
		t.Fatalf("expected ErrAlreadyTrainer, got %v", err)
	}
}

func TestRejectApplication(t *testing.T) {
	apps := &stubApplicationRepo{apps: []models.TrainerApplication{testApp(models.ApplicationStatusPending)}}
	service := newTestService(apps, stubUserRepo{}, stubTrainerRepo{})

	if err := service.RejectApplication(context.Background(), activeApplication, reviewerAdminID, "Blurred identification."); err != nil {
		t.Fatalf("RejectApplication: %v", err)
	}
	application, err := service.GetApplication(context.Background(), activeApplication)
	if err != nil {
		t.Fatalf("GetApplication: %v", err)
	}
	if application.Status != models.ApplicationStatusRejected {
		t.Fatalf("expected REJECTED, got %q", application.Status)
	}
	if len(apps.decisions) != 1 {
		t.Fatalf("expected one recorded decision, got %d", len(apps.decisions))
	}
	decision := apps.decisions[0]
	if decision.ApplicationID != activeApplication || decision.AdminID != reviewerAdminID ||
		decision.FromStatus != models.ApplicationStatusPending || decision.ToStatus != models.ApplicationStatusRejected ||
		decision.Reason != "Blurred identification." {
		t.Fatalf("unexpected recorded decision %+v", decision)
	}
}

func TestRejectApplicationNotPending(t *testing.T) {
	service := newTestService(
		&stubApplicationRepo{apps: []models.TrainerApplication{testApp(models.ApplicationStatusApproved)}},
		stubUserRepo{},
		stubTrainerRepo{},
	)

	if err := service.RejectApplication(context.Background(), activeApplication, reviewerAdminID, ""); !errors.Is(err, trainer_applications.ErrApplicationStateConflict) {
		t.Fatalf("expected ErrApplicationStateConflict, got %v", err)
	}
}

func TestRejectApplicationNotFound(t *testing.T) {
	service := newTestService(&stubApplicationRepo{}, stubUserRepo{}, stubTrainerRepo{})

	if err := service.RejectApplication(context.Background(), missingApplication, reviewerAdminID, ""); !errors.Is(err, trainer_applications.ErrApplicationNotFound) {
		t.Fatalf("expected ErrApplicationNotFound, got %v", err)
	}
}

// TestRejectThenReapply covers the full lifecycle: after a rejection the user
// can apply again and the new application can be approved.
func TestRejectThenReapply(t *testing.T) {
	apps := &stubApplicationRepo{
		apps: []models.TrainerApplication{testApp(models.ApplicationStatusPending)},
	}
	service := newTestService(apps, stubUserRepo{users: []models.User{activeUser()}}, stubTrainerRepo{})

	if err := service.RejectApplication(context.Background(), activeApplication, reviewerAdminID, ""); err != nil {
		t.Fatalf("RejectApplication: %v", err)
	}

	reapplied, err := service.Apply(context.Background(), activeUserID)
	if err != nil {
		t.Fatalf("Apply after rejection: %v", err)
	}
	if reapplied.Status != models.ApplicationStatusPending {
		t.Fatalf("expected PENDING after re-apply, got %q", reapplied.Status)
	}

	if _, err := service.ApproveApplication(context.Background(), reapplied.ID, reviewerAdminID, ""); err != nil {
		t.Fatalf("ApproveApplication after re-apply: %v", err)
	}
	if len(apps.decisions) != 2 {
		t.Fatalf("expected both decisions to be recorded, got %d", len(apps.decisions))
	}
}

func TestGetAccessStatusTrainer(t *testing.T) {
	apps := &stubApplicationRepo{
		apps: []models.TrainerApplication{testApp(models.ApplicationStatusApproved)},
	}
	service := newTestService(apps, stubUserRepo{users: []models.User{activeUser()}}, stubTrainerRepo{
		trainers: []models.Trainer{{ID: "trainer-1", UserID: activeUserID}},
	})

	status, err := service.GetAccessStatus(context.Background(), activeUserID)
	if err != nil {
		t.Fatalf("GetAccessStatus: %v", err)
	}
	if !status.IsTrainer {
		t.Fatal("expected IsTrainer true for an active trainer profile")
	}
	if status.Application == nil || status.Application.Status != models.ApplicationStatusApproved {
		t.Fatalf("expected the approved application, got %+v", status.Application)
	}
}

func TestGetAccessStatusRejectedApplication(t *testing.T) {
	apps := &stubApplicationRepo{
		apps: []models.TrainerApplication{testApp(models.ApplicationStatusRejected)},
	}
	service := newTestService(apps, stubUserRepo{users: []models.User{activeUser()}}, stubTrainerRepo{})

	status, err := service.GetAccessStatus(context.Background(), activeUserID)
	if err != nil {
		t.Fatalf("GetAccessStatus: %v", err)
	}
	if status.IsTrainer {
		t.Fatal("expected IsTrainer false without a trainer profile")
	}
	if status.Application == nil || status.Application.Status != models.ApplicationStatusRejected {
		t.Fatalf("expected the rejected application, got %+v", status.Application)
	}
}

func TestGetAccessStatusNoApplication(t *testing.T) {
	service := newTestService(&stubApplicationRepo{}, stubUserRepo{users: []models.User{activeUser()}}, stubTrainerRepo{})

	status, err := service.GetAccessStatus(context.Background(), activeUserID)
	if err != nil {
		t.Fatalf("GetAccessStatus: %v", err)
	}
	if status.IsTrainer || status.Application != nil {
		t.Fatalf("expected no trainer access and no application, got %+v", status)
	}
}

func TestGetAccessStatusInvalidUserID(t *testing.T) {
	service := newTestService(&stubApplicationRepo{}, stubUserRepo{}, stubTrainerRepo{})

	if _, err := service.GetAccessStatus(context.Background(), "not-a-uuid"); !errors.Is(err, trainer_applications.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetAccessStatusRepositoryError(t *testing.T) {
	apps := &stubApplicationRepo{findErr: errRepoFailure}
	service := newTestService(apps, stubUserRepo{users: []models.User{activeUser()}}, stubTrainerRepo{})

	if _, err := service.GetAccessStatus(context.Background(), activeUserID); !errors.Is(err, errRepoFailure) {
		t.Fatalf("expected wrapped repository failure, got %v", err)
	}
}

func pendingEvidenceSet() (*stubApplicationRepo, *stubDocumentRepo, *stubDecisionRepo, *stubEvidenceStore) {
	apps := &stubApplicationRepo{apps: []models.TrainerApplication{testApp(models.ApplicationStatusPending)}}
	documents := &stubDocumentRepo{}
	decisions := &stubDecisionRepo{}
	evidence := &stubEvidenceStore{}
	return apps, documents, decisions, evidence
}

func TestUploadDocumentSuccess(t *testing.T) {
	apps, documents, decisions, evidence := pendingEvidenceSet()
	service := newTestServiceWithEvidence(apps, stubUserRepo{users: []models.User{activeUser()}}, stubTrainerRepo{}, documents, decisions, evidence)

	document, err := service.UploadDocument(context.Background(), activeUserID, "certification", "../../etc/passwd.jpg", strings.NewReader("ignored-by-stub"))
	if err != nil {
		t.Fatalf("UploadDocument: %v", err)
	}
	if document.ApplicationID != activeApplication {
		t.Fatalf("expected the caller's active application %q, got %q", activeApplication, document.ApplicationID)
	}
	if document.Kind != "certification" {
		t.Fatalf("expected kind certification, got %q", document.Kind)
	}
	if document.OriginalFilename != "passwd.jpg" {
		t.Fatalf("expected a sanitized filename, got %q", document.OriginalFilename)
	}
	if document.ContentType != "application/pdf" || document.ByteSize != 6 || document.SHA256 == "" {
		t.Fatalf("expected the store-validated metadata, got %+v", document)
	}
	if document.StorageKey == "" || document.ID == "" {
		t.Fatalf("expected a generated id and storage key, got %+v", document)
	}
	if evidence.saves != 1 {
		t.Fatalf("expected one store write, got %d", evidence.saves)
	}
	listed, err := service.ListOwnDocuments(context.Background(), activeUserID)
	if err != nil {
		t.Fatalf("ListOwnDocuments: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != document.ID {
		t.Fatalf("expected the uploaded document to be listed, got %+v", listed)
	}
}

func TestUploadDocumentValidatesInputBeforeStoring(t *testing.T) {
	apps, documents, decisions, evidence := pendingEvidenceSet()
	service := newTestServiceWithEvidence(apps, stubUserRepo{users: []models.User{activeUser()}}, stubTrainerRepo{}, documents, decisions, evidence)

	for _, tc := range []struct {
		name   string
		userID string
		kind   string
		store  error
		want   error
	}{
		{name: "invalid user", userID: "not-a-uuid", kind: "certification", want: trainer_applications.ErrInvalidInput},
		{name: "invalid kind", userID: activeUserID, kind: "passport", want: trainer_applications.ErrInvalidInput},
		{name: "unsupported", userID: activeUserID, kind: "other", store: trainer_evidence.ErrUnsupportedContentType, want: trainer_applications.ErrDocumentUnsupportedType},
		{name: "too large", userID: activeUserID, kind: "other", store: trainer_evidence.ErrTooLarge, want: trainer_applications.ErrDocumentTooLarge},
		{name: "empty", userID: activeUserID, kind: "other", store: trainer_evidence.ErrEmptyContent, want: trainer_applications.ErrInvalidInput},
	} {
		evidence.saveErr = tc.store
		_, err := service.UploadDocument(context.Background(), tc.userID, tc.kind, "scan.pdf", strings.NewReader("data"))
		if !errors.Is(err, tc.want) {
			t.Fatalf("%s: expected %v, got %v", tc.name, tc.want, err)
		}
	}
	if len(documents.documents) != 0 {
		t.Fatalf("rejected uploads must not record metadata, got %d", len(documents.documents))
	}
}

func TestUploadDocumentRequiresPendingApplication(t *testing.T) {
	// An APPROVED application is still active, so the upload is refused as a
	// state conflict (evidence cannot change after review).
	apps := &stubApplicationRepo{apps: []models.TrainerApplication{testApp(models.ApplicationStatusApproved)}}
	documents, decisions, evidence := &stubDocumentRepo{}, &stubDecisionRepo{}, &stubEvidenceStore{}
	service := newTestServiceWithEvidence(apps, stubUserRepo{users: []models.User{activeUser()}}, stubTrainerRepo{}, documents, decisions, evidence)

	_, err := service.UploadDocument(context.Background(), activeUserID, "certification", "scan.pdf", strings.NewReader("data"))
	if !errors.Is(err, trainer_applications.ErrApplicationStateConflict) {
		t.Fatalf("approved application: expected ErrApplicationStateConflict, got %v", err)
	}
	if evidence.saves != 0 {
		t.Fatal("a locked application must never reach the store")
	}

	// A REJECTED application is no longer active, so there is nothing to
	// attach evidence to and no active application is reported.
	rejected := &stubApplicationRepo{apps: []models.TrainerApplication{testApp(models.ApplicationStatusRejected)}}
	service = newTestServiceWithEvidence(rejected, stubUserRepo{users: []models.User{activeUser()}}, stubTrainerRepo{}, &stubDocumentRepo{}, &stubDecisionRepo{}, &stubEvidenceStore{})
	if _, err := service.UploadDocument(context.Background(), activeUserID, "certification", "scan.pdf", strings.NewReader("data")); !errors.Is(err, trainer_applications.ErrApplicationNotFound) {
		t.Fatalf("rejected application: expected ErrApplicationNotFound, got %v", err)
	}

	// No application at all.
	service = newTestServiceWithEvidence(&stubApplicationRepo{}, stubUserRepo{users: []models.User{activeUser()}}, stubTrainerRepo{}, &stubDocumentRepo{}, &stubDecisionRepo{}, &stubEvidenceStore{})
	if _, err := service.UploadDocument(context.Background(), activeUserID, "certification", "scan.pdf", strings.NewReader("data")); !errors.Is(err, trainer_applications.ErrApplicationNotFound) {
		t.Fatalf("no active application: expected ErrApplicationNotFound, got %v", err)
	}
}

func TestOwnDocumentsAreScopedToTheCaller(t *testing.T) {
	otherUserID := "77777777-7777-7777-7777-777777777777"
	otherApplicationID := "88888888-8888-8888-8888-888888888888"
	apps := &stubApplicationRepo{apps: []models.TrainerApplication{
		testApp(models.ApplicationStatusPending),
		{ID: otherApplicationID, UserID: otherUserID, Status: models.ApplicationStatusPending},
	}}
	documents := &stubDocumentRepo{documents: []models.TrainerApplicationDocument{
		{ID: activeDocument, ApplicationID: activeApplication, Kind: "other"},
		{ID: missingDocument, ApplicationID: otherApplicationID, Kind: "other"},
	}}
	service := newTestServiceWithEvidence(apps, stubUserRepo{users: []models.User{activeUser()}}, stubTrainerRepo{}, documents, &stubDecisionRepo{}, &stubEvidenceStore{})
	ctx := context.Background()

	if _, err := service.GetOwnDocument(ctx, activeUserID, activeDocument); err != nil {
		t.Fatalf("GetOwnDocument(own): %v", err)
	}
	if _, err := service.GetOwnDocument(ctx, activeUserID, missingDocument); !errors.Is(err, trainer_applications.ErrDocumentNotFound) {
		t.Fatalf("GetOwnDocument(other user): expected ErrDocumentNotFound, got %v", err)
	}
	if _, _, err := service.OpenOwnDocument(ctx, activeUserID, missingDocument); !errors.Is(err, trainer_applications.ErrDocumentNotFound) {
		t.Fatalf("OpenOwnDocument(other user): expected ErrDocumentNotFound, got %v", err)
	}
	if err := service.DeleteOwnDocument(ctx, activeUserID, missingDocument); !errors.Is(err, trainer_applications.ErrDocumentNotFound) {
		t.Fatalf("DeleteOwnDocument(other user): expected ErrDocumentNotFound, got %v", err)
	}

	owned, err := service.ListOwnDocuments(ctx, activeUserID)
	if err != nil {
		t.Fatalf("ListOwnDocuments: %v", err)
	}
	if len(owned) != 1 || owned[0].ID != activeDocument {
		t.Fatalf("expected only the caller's document, got %+v", owned)
	}
}

func TestOpenOwnDocumentStreamsStoredContent(t *testing.T) {
	apps, documents, decisions, evidence := pendingEvidenceSet()
	evidence.content = "stored-bytes"
	documents.documents = []models.TrainerApplicationDocument{{ID: activeDocument, ApplicationID: activeApplication, Kind: "other", StorageKey: activeApplication + "/" + activeDocument + ".pdf"}}
	service := newTestServiceWithEvidence(apps, stubUserRepo{users: []models.User{activeUser()}}, stubTrainerRepo{}, documents, decisions, evidence)

	document, content, err := service.OpenOwnDocument(context.Background(), activeUserID, activeDocument)
	if err != nil {
		t.Fatalf("OpenOwnDocument: %v", err)
	}
	defer content.Close()
	if document.ID != activeDocument {
		t.Fatalf("expected document %q, got %q", activeDocument, document.ID)
	}
	streamed, err := io.ReadAll(content)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	if string(streamed) != "stored-bytes" {
		t.Fatalf("expected the stored bytes, got %q", string(streamed))
	}

	evidence.openErr = trainer_evidence.ErrDocumentNotFound
	if _, _, err := service.OpenOwnDocument(context.Background(), activeUserID, activeDocument); !errors.Is(err, trainer_applications.ErrDocumentNotFound) {
		t.Fatalf("missing stored file: expected ErrDocumentNotFound, got %v", err)
	}
}

func TestDeleteOwnDocumentLocksAfterReview(t *testing.T) {
	apps, documents, decisions, evidence := pendingEvidenceSet()
	documents.documents = []models.TrainerApplicationDocument{{ID: activeDocument, ApplicationID: activeApplication, Kind: "other"}}
	service := newTestServiceWithEvidence(apps, stubUserRepo{users: []models.User{activeUser()}}, stubTrainerRepo{}, documents, decisions, evidence)
	ctx := context.Background()

	if err := service.DeleteOwnDocument(ctx, activeUserID, activeDocument); err != nil {
		t.Fatalf("DeleteOwnDocument: %v", err)
	}
	if !documents.documents[0].DeletedAt.Valid {
		t.Fatal("expected a soft-deleted metadata row")
	}
	if _, err := service.GetOwnDocument(ctx, activeUserID, activeDocument); !errors.Is(err, trainer_applications.ErrDocumentNotFound) {
		t.Fatalf("after withdrawal: expected ErrDocumentNotFound, got %v", err)
	}

	apps.apps[0].Status = models.ApplicationStatusApproved
	documents.documents[0].DeletedAt.Valid = false
	if err := service.DeleteOwnDocument(ctx, activeUserID, activeDocument); !errors.Is(err, trainer_applications.ErrApplicationStateConflict) {
		t.Fatalf("reviewed application: expected ErrApplicationStateConflict, got %v", err)
	}
}

func TestReviewDocumentScoping(t *testing.T) {
	apps, documents, decisions, evidence := pendingEvidenceSet()
	otherApplicationID := "88888888-8888-8888-8888-888888888888"
	documents.documents = []models.TrainerApplicationDocument{
		{ID: activeDocument, ApplicationID: activeApplication, Kind: "other"},
		{ID: missingDocument, ApplicationID: otherApplicationID, Kind: "other"},
	}
	decisions.decisions = []models.TrainerApplicationDecision{{ID: "decision-1", ApplicationID: activeApplication, AdminID: reviewerAdminID}}
	service := newTestServiceWithEvidence(apps, stubUserRepo{}, stubTrainerRepo{}, documents, decisions, evidence)
	ctx := context.Background()

	if _, err := service.ListApplicationDocuments(ctx, missingApplication); !errors.Is(err, trainer_applications.ErrApplicationNotFound) {
		t.Fatalf("ListApplicationDocuments(unknown): expected ErrApplicationNotFound, got %v", err)
	}
	listed, err := service.ListApplicationDocuments(ctx, activeApplication)
	if err != nil {
		t.Fatalf("ListApplicationDocuments: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != activeDocument {
		t.Fatalf("expected only the application's documents, got %+v", listed)
	}
	if _, err := service.GetApplicationDocument(ctx, activeApplication, missingDocument); !errors.Is(err, trainer_applications.ErrDocumentNotFound) {
		t.Fatalf("cross-application document: expected ErrDocumentNotFound, got %v", err)
	}
	if _, _, err := service.OpenApplicationDocument(ctx, activeApplication, activeDocument); err != nil {
		t.Fatalf("OpenApplicationDocument: %v", err)
	}

	listedDecisions, err := service.ListDecisions(ctx, activeApplication)
	if err != nil {
		t.Fatalf("ListDecisions: %v", err)
	}
	if len(listedDecisions) != 1 || listedDecisions[0].ID != "decision-1" {
		t.Fatalf("expected the recorded decision, got %+v", listedDecisions)
	}
	if _, err := service.ListDecisions(ctx, missingApplication); !errors.Is(err, trainer_applications.ErrApplicationNotFound) {
		t.Fatalf("ListDecisions(unknown): expected ErrApplicationNotFound, got %v", err)
	}
}

func TestListOwnDocumentsWithoutApplication(t *testing.T) {
	service := newTestService(&stubApplicationRepo{}, stubUserRepo{users: []models.User{activeUser()}}, stubTrainerRepo{})
	if _, err := service.ListOwnDocuments(context.Background(), activeUserID); !errors.Is(err, trainer_applications.ErrApplicationNotFound) {
		t.Fatalf("expected ErrApplicationNotFound, got %v", err)
	}
}
