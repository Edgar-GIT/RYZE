package trainer_applications_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/trainer_applications"
)

const (
	activeUserID       = "11111111-1111-1111-1111-111111111111"
	unknownUserID      = "22222222-2222-2222-2222-222222222222"
	activeApplication  = "33333333-3333-3333-3333-333333333333"
	missingApplication = "44444444-4444-4444-4444-444444444444"
)

var errRepoFailure = errors.New("repository failure")

// stubApplicationRepo is a stateful in-memory fake of the application
// data-access surface. It mirrors the repository semantics: only PENDING or
// APPROVED applications are considered active, a user can hold at most one
// active application, and Approve/Reject only transition PENDING applications.
type stubApplicationRepo struct {
	apps       []models.TrainerApplication
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

func (s *stubApplicationRepo) Approve(_ context.Context, id string) (*models.TrainerApplication, error) {
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
	return app, nil
}

func (s *stubApplicationRepo) Reject(_ context.Context, id string) error {
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
	return nil
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

func newTestService(apps *stubApplicationRepo, users stubUserRepo, trainers stubTrainerRepo) trainer_applications.Service {
	return trainer_applications.NewService(apps, users, trainers)
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
	service := newTestService(
		&stubApplicationRepo{apps: []models.TrainerApplication{testApp(models.ApplicationStatusPending)}},
		stubUserRepo{users: []models.User{activeUser()}},
		stubTrainerRepo{},
	)

	application, err := service.ApproveApplication(context.Background(), activeApplication)
	if err != nil {
		t.Fatalf("ApproveApplication: %v", err)
	}
	if application.Status != models.ApplicationStatusApproved {
		t.Fatalf("expected APPROVED, got %q", application.Status)
	}
}

func TestApproveApplicationNotFound(t *testing.T) {
	service := newTestService(&stubApplicationRepo{}, stubUserRepo{}, stubTrainerRepo{})

	if _, err := service.ApproveApplication(context.Background(), missingApplication); !errors.Is(err, trainer_applications.ErrApplicationNotFound) {
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

		if _, err := service.ApproveApplication(context.Background(), activeApplication); !errors.Is(err, trainer_applications.ErrApplicationStateConflict) {
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

	if _, err := service.ApproveApplication(context.Background(), activeApplication); !errors.Is(err, trainer_applications.ErrUserNotFound) {
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

	if _, err := service.ApproveApplication(context.Background(), activeApplication); !errors.Is(err, trainer_applications.ErrAlreadyTrainer) {
		t.Fatalf("expected ErrAlreadyTrainer, got %v", err)
	}
}

func TestRejectApplication(t *testing.T) {
	service := newTestService(
		&stubApplicationRepo{apps: []models.TrainerApplication{testApp(models.ApplicationStatusPending)}},
		stubUserRepo{},
		stubTrainerRepo{},
	)

	if err := service.RejectApplication(context.Background(), activeApplication); err != nil {
		t.Fatalf("RejectApplication: %v", err)
	}
	application, err := service.GetApplication(context.Background(), activeApplication)
	if err != nil {
		t.Fatalf("GetApplication: %v", err)
	}
	if application.Status != models.ApplicationStatusRejected {
		t.Fatalf("expected REJECTED, got %q", application.Status)
	}
}

func TestRejectApplicationNotPending(t *testing.T) {
	service := newTestService(
		&stubApplicationRepo{apps: []models.TrainerApplication{testApp(models.ApplicationStatusApproved)}},
		stubUserRepo{},
		stubTrainerRepo{},
	)

	if err := service.RejectApplication(context.Background(), activeApplication); !errors.Is(err, trainer_applications.ErrApplicationStateConflict) {
		t.Fatalf("expected ErrApplicationStateConflict, got %v", err)
	}
}

func TestRejectApplicationNotFound(t *testing.T) {
	service := newTestService(&stubApplicationRepo{}, stubUserRepo{}, stubTrainerRepo{})

	if err := service.RejectApplication(context.Background(), missingApplication); !errors.Is(err, trainer_applications.ErrApplicationNotFound) {
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

	if err := service.RejectApplication(context.Background(), activeApplication); err != nil {
		t.Fatalf("RejectApplication: %v", err)
	}

	reapplied, err := service.Apply(context.Background(), activeUserID)
	if err != nil {
		t.Fatalf("Apply after rejection: %v", err)
	}
	if reapplied.Status != models.ApplicationStatusPending {
		t.Fatalf("expected PENDING after re-apply, got %q", reapplied.Status)
	}

	if _, err := service.ApproveApplication(context.Background(), reapplied.ID); err != nil {
		t.Fatalf("ApproveApplication after re-apply: %v", err)
	}
}
