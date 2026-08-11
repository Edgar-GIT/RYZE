package admin_trainers_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/admin_trainers"
	"ryze/backend/services/admin_users"
	"ryze/backend/services/password"
	"ryze/backend/services/registration"
)

var errRepoFailure = errors.New("repository failure")

// stubRegistrar simulates the registration dependency used by CreateTrainer.
type stubRegistrar struct {
	user *models.User
	err  error
}

func (r stubRegistrar) Register(_ context.Context, _ registration.RegisterInput) (*models.User, error) {
	return r.user, r.err
}

// stubUserRepo simulates the user data-access surface used by the trainer
// lifecycle (the reactivation user guard and the create compensation).
type stubUserRepo struct {
	users []models.User
}

func (s *stubUserRepo) FindByID(_ context.Context, id string) (*models.User, error) {
	for i := range s.users {
		if s.users[i].ID == id && !s.users[i].DeletedAt.Valid {
			return &s.users[i], nil
		}
	}
	return nil, repositories.ErrUserNotFound
}

func (s *stubUserRepo) SoftDelete(_ context.Context, id string) error {
	for i := range s.users {
		if s.users[i].ID == id && !s.users[i].DeletedAt.Valid {
			s.users[i].DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
			return nil
		}
	}
	return repositories.ErrUserNotFound
}

// stubUserUpdater simulates the user-profile update dependency used by
// UpdateTrainer.
type stubUserUpdater struct {
	user *models.User
	err  error
}

func (u stubUserUpdater) UpdateUser(_ context.Context, _ string, _ admin_users.UpdateUserInput) (*models.User, error) {
	return u.user, u.err
}

// stubTrainerRepo is a stateful in-memory fake of the trainer data-access
// surface. It mirrors the repository semantics: soft-deleted rows are kept and
// only surfaced through the IncludingDeleted/ListDeleted operations, and
// reactivation is guarded by the deleted state and the one-to-one link rule.
type stubTrainerRepo struct {
	all          []models.Trainer
	createErr    error
	listErr      error
	deletedListErr error
	getErr       error
	findUserErr  error
	deleteErr    error
	reactivateErr error
	deletedID    string
	reactivatedID string
}

func (s *stubTrainerRepo) Create(_ context.Context, trainer *models.Trainer) error {
	if s.createErr != nil {
		return s.createErr
	}
	for _, existing := range s.all {
		if existing.UserID == trainer.UserID && !existing.DeletedAt.Valid {
			return repositories.ErrTrainerAlreadyLinked
		}
	}
	if trainer.ID == "" {
		trainer.ID = uuid.NewString()
	}
	s.all = append(s.all, *trainer)
	return nil
}

func (s *stubTrainerRepo) FindByID(_ context.Context, id string) (*models.Trainer, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	for i := range s.all {
		if s.all[i].ID == id && !s.all[i].DeletedAt.Valid {
			return &s.all[i], nil
		}
	}
	return nil, repositories.ErrTrainerNotFound
}

func (s *stubTrainerRepo) FindByIDIncludingDeleted(_ context.Context, id string) (*models.Trainer, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	for i := range s.all {
		if s.all[i].ID == id {
			return &s.all[i], nil
		}
	}
	return nil, repositories.ErrTrainerNotFound
}

func (s *stubTrainerRepo) FindByUserID(_ context.Context, userID string) (*models.Trainer, error) {
	if s.findUserErr != nil {
		return nil, s.findUserErr
	}
	for i := range s.all {
		if s.all[i].UserID == userID && !s.all[i].DeletedAt.Valid {
			return &s.all[i], nil
		}
	}
	return nil, repositories.ErrTrainerNotFound
}

func (s *stubTrainerRepo) ListActive(_ context.Context, _ int, _ int) ([]models.Trainer, int64, error) {
	if s.listErr != nil {
		return nil, 0, s.listErr
	}
	active := make([]models.Trainer, 0)
	for _, t := range s.all {
		if !t.DeletedAt.Valid {
			active = append(active, t)
		}
	}
	return active, int64(len(active)), nil
}

func (s *stubTrainerRepo) ListDeleted(_ context.Context, _ int, _ int) ([]models.Trainer, int64, error) {
	if s.deletedListErr != nil {
		return nil, 0, s.deletedListErr
	}
	deleted := make([]models.Trainer, 0)
	for _, t := range s.all {
		if t.DeletedAt.Valid {
			deleted = append(deleted, t)
		}
	}
	return deleted, int64(len(deleted)), nil
}

func (s *stubTrainerRepo) SoftDelete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	for i := range s.all {
		if s.all[i].ID == id && !s.all[i].DeletedAt.Valid {
			s.all[i].DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
			s.deletedID = id
			return nil
		}
	}
	return repositories.ErrTrainerNotFound
}

func (s *stubTrainerRepo) Reactivate(_ context.Context, id string) error {
	if s.reactivateErr != nil {
		return s.reactivateErr
	}
	for i := range s.all {
		if s.all[i].ID == id && s.all[i].DeletedAt.Valid {
			for j := range s.all {
				if j != i && s.all[j].UserID == s.all[i].UserID && !s.all[j].DeletedAt.Valid {
					return repositories.ErrTrainerAlreadyLinked
				}
			}
			s.all[i].DeletedAt = gorm.DeletedAt{}
			s.reactivatedID = id
			return nil
		}
	}
	return repositories.ErrTrainerNotFound
}

func newTestService(repo *stubTrainerRepo, userRepo *stubUserRepo, registrar stubRegistrar, updater stubUserUpdater) admin_trainers.AdminTrainerService {
	return admin_trainers.NewAdminTrainerService(repo, userRepo, registrar, updater)
}

func TestListTrainersReturnsPageAndTotal(t *testing.T) {
	repo := &stubTrainerRepo{all: []models.Trainer{
		{ID: uuid.NewString(), UserID: uuid.NewString()},
		{ID: uuid.NewString(), UserID: uuid.NewString()},
	}}
	svc := newTestService(repo, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	result, err := svc.ListTrainers(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("ListTrainers: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
	if len(result.Trainers) != 2 {
		t.Fatalf("expected 2 trainers, got %d", len(result.Trainers))
	}
	if result.Page != 1 || result.Limit != 20 {
		t.Fatalf("expected page 1 limit 20, got page %d limit %d", result.Page, result.Limit)
	}
}

func TestListTrainersClampsOversizedLimit(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{}, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	result, err := svc.ListTrainers(context.Background(), 1, 10000)
	if err != nil {
		t.Fatalf("ListTrainers: %v", err)
	}
	if result.Limit != admin_trainers.MaxPageSize {
		t.Fatalf("expected limit clamped to %d, got %d", admin_trainers.MaxPageSize, result.Limit)
	}
}

func TestListTrainersRejectsInvalidPagination(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{}, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	cases := []struct {
		name  string
		page  int
		limit int
	}{
		{name: "page zero", page: 0, limit: 20},
		{name: "page negative", page: -1, limit: 20},
		{name: "limit zero", page: 1, limit: 0},
		{name: "limit negative", page: 1, limit: -5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.ListTrainers(context.Background(), tc.page, tc.limit); !errors.Is(err, admin_trainers.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestListTrainersWrapsRepositoryFailure(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{listErr: errRepoFailure}, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	if _, err := svc.ListTrainers(context.Background(), 1, 20); err == nil {
		t.Fatal("expected an error")
	}
}

func TestListDeletedTrainersReturnsDeletedOnly(t *testing.T) {
	repo := &stubTrainerRepo{all: []models.Trainer{
		{ID: uuid.NewString(), UserID: uuid.NewString()},
		{ID: uuid.NewString(), UserID: uuid.NewString(), DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true}},
	}}
	svc := newTestService(repo, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	result, err := svc.ListDeletedTrainers(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("ListDeletedTrainers: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
	if len(result.Trainers) != 1 {
		t.Fatalf("expected only the soft-deleted trainer, got %+v", result.Trainers)
	}
}

func TestListDeletedTrainersClampsOversizedLimit(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{}, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	result, err := svc.ListDeletedTrainers(context.Background(), 1, 99999)
	if err != nil {
		t.Fatalf("ListDeletedTrainers: %v", err)
	}
	if result.Limit != admin_trainers.MaxPageSize {
		t.Fatalf("expected limit clamped to %d, got %d", admin_trainers.MaxPageSize, result.Limit)
	}
}

func TestListDeletedTrainersRejectsInvalidPagination(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{}, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	for _, tc := range []struct {
		page  int
		limit int
	}{
		{page: 0, limit: 20},
		{page: 1, limit: 0},
	} {
		if _, err := svc.ListDeletedTrainers(context.Background(), tc.page, tc.limit); !errors.Is(err, admin_trainers.ErrInvalidInput) {
			t.Fatalf("page %d limit %d: expected ErrInvalidInput, got %v", tc.page, tc.limit, err)
		}
	}
}

func TestListDeletedTrainersWrapsRepositoryFailure(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{deletedListErr: errRepoFailure}, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	if _, err := svc.ListDeletedTrainers(context.Background(), 1, 20); err == nil {
		t.Fatal("expected an error")
	}
}

func TestGetTrainerReturnsTrainer(t *testing.T) {
	id := uuid.NewString()
	trainer := models.Trainer{ID: id, UserID: uuid.NewString()}
	svc := newTestService(&stubTrainerRepo{all: []models.Trainer{trainer}}, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	got, err := svc.GetTrainer(context.Background(), id)
	if err != nil {
		t.Fatalf("GetTrainer: %v", err)
	}
	if got.Trainer.ID != id {
		t.Fatalf("expected trainer %q, got %q", id, got.Trainer.ID)
	}
}

func TestGetTrainerRejectsInvalidID(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{}, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	for _, id := range []string{"", "not-a-uuid", "  "} {
		if _, err := svc.GetTrainer(context.Background(), id); !errors.Is(err, admin_trainers.ErrInvalidInput) {
			t.Fatalf("id %q: expected ErrInvalidInput, got %v", id, err)
		}
	}
}

func TestGetTrainerMapsNotFound(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{}, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	_, err := svc.GetTrainer(context.Background(), uuid.NewString())
	if !errors.Is(err, admin_trainers.ErrTrainerNotFound) {
		t.Fatalf("expected ErrTrainerNotFound, got %v", err)
	}
}

func TestGetTrainerWrapsRepositoryFailure(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{getErr: errRepoFailure}, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	if _, err := svc.GetTrainer(context.Background(), uuid.NewString()); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCreateTrainerCreatesUserAndTrainer(t *testing.T) {
	user := &models.User{ID: uuid.NewString(), Email: "new@ryze.local", FirstName: "New", LastName: "Trainer"}
	svc := newTestService(&stubTrainerRepo{}, &stubUserRepo{}, stubRegistrar{user: user}, stubUserUpdater{})

	result, err := svc.CreateTrainer(context.Background(), admin_trainers.CreateTrainerInput{
		Email:     "new@ryze.local",
		Password:  "Password123!",
		FirstName: "New",
		LastName:  "Trainer",
	})
	if err != nil {
		t.Fatalf("CreateTrainer: %v", err)
	}
	if result.User.ID != user.ID {
		t.Fatalf("expected user %q, got %q", user.ID, result.User.ID)
	}
	if result.Trainer.UserID != user.ID {
		t.Fatalf("expected trainer user link %q, got %q", user.ID, result.Trainer.UserID)
	}
	if result.Trainer.ID == "" {
		t.Fatal("expected a trainer id to be generated")
	}
}

func TestCreateTrainerMapsDuplicateEmail(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{}, &stubUserRepo{}, stubRegistrar{err: repositories.ErrDuplicateEmail}, stubUserUpdater{})

	_, err := svc.CreateTrainer(context.Background(), admin_trainers.CreateTrainerInput{
		Email:     "dup@ryze.local",
		Password:  "Password123!",
		FirstName: "Dup",
		LastName:  "Trainer",
	})
	if !errors.Is(err, admin_trainers.ErrDuplicateEmail) {
		t.Fatalf("expected ErrDuplicateEmail, got %v", err)
	}
}

func TestCreateTrainerMapsInvalidInput(t *testing.T) {
	for _, regErr := range []error{registration.ErrInvalidInput, password.ErrEmptyPassword} {
		svc := newTestService(&stubTrainerRepo{}, &stubUserRepo{}, stubRegistrar{err: regErr}, stubUserUpdater{})

		_, err := svc.CreateTrainer(context.Background(), admin_trainers.CreateTrainerInput{
			Email:     "bad@ryze.local",
			Password:  "Password123!",
			FirstName: "Bad",
			LastName:  "Input",
		})
		if !errors.Is(err, admin_trainers.ErrInvalidInput) {
			t.Fatalf("registrar error %v: expected ErrInvalidInput, got %v", regErr, err)
		}
	}
}

func TestCreateTrainerWrapsUnexpectedRegistrarFailure(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{}, &stubUserRepo{}, stubRegistrar{err: errRepoFailure}, stubUserUpdater{})

	_, err := svc.CreateTrainer(context.Background(), admin_trainers.CreateTrainerInput{
		Email:     "fail@ryze.local",
		Password:  "Password123!",
		FirstName: "Fail",
		LastName:  "Trainer",
	})
	if err == nil || errors.Is(err, admin_trainers.ErrInvalidInput) || errors.Is(err, admin_trainers.ErrDuplicateEmail) {
		t.Fatalf("expected an unexpected failure, got %v", err)
	}
}

func TestCreateTrainerRejectsExistingActiveTrainerAndCompensates(t *testing.T) {
	user := &models.User{ID: uuid.NewString(), Email: "linked@ryze.local"}
	userRepo := &stubUserRepo{users: []models.User{*user}}
	repo := &stubTrainerRepo{all: []models.Trainer{{ID: uuid.NewString(), UserID: user.ID}}}
	svc := newTestService(repo, userRepo, stubRegistrar{user: user}, stubUserUpdater{})

	_, err := svc.CreateTrainer(context.Background(), admin_trainers.CreateTrainerInput{
		Email:     "linked@ryze.local",
		Password:  "Password123!",
		FirstName: "Linked",
		LastName:  "Trainer",
	})
	if !errors.Is(err, admin_trainers.ErrTrainerAlreadyLinked) {
		t.Fatalf("expected ErrTrainerAlreadyLinked, got %v", err)
	}
	if !userRepo.users[0].DeletedAt.Valid {
		t.Fatal("the created user must be soft-deleted as a compensation")
	}
}

func TestCreateTrainerCompensatesOnTrainerCreationFailure(t *testing.T) {
	user := &models.User{ID: uuid.NewString(), Email: "txfail@ryze.local"}
	userRepo := &stubUserRepo{users: []models.User{*user}}
	repo := &stubTrainerRepo{createErr: errRepoFailure}
	svc := newTestService(repo, userRepo, stubRegistrar{user: user}, stubUserUpdater{})

	_, err := svc.CreateTrainer(context.Background(), admin_trainers.CreateTrainerInput{
		Email:     "txfail@ryze.local",
		Password:  "Password123!",
		FirstName: "Tx",
		LastName:  "Fail",
	})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !userRepo.users[0].DeletedAt.Valid {
		t.Fatal("the created user must be soft-deleted as a compensation on trainer creation failure")
	}
}

func TestCreateTrainerCompensatesOnLookupFailure(t *testing.T) {
	user := &models.User{ID: uuid.NewString(), Email: "lookup@ryze.local"}
	userRepo := &stubUserRepo{users: []models.User{*user}}
	repo := &stubTrainerRepo{findUserErr: errRepoFailure}
	svc := newTestService(repo, userRepo, stubRegistrar{user: user}, stubUserUpdater{})

	if _, err := svc.CreateTrainer(context.Background(), admin_trainers.CreateTrainerInput{
		Email:     "lookup@ryze.local",
		Password:  "Password123!",
		FirstName: "Lookup",
		LastName:  "Fail",
	}); err == nil {
		t.Fatal("expected an error")
	}
	if !userRepo.users[0].DeletedAt.Valid {
		t.Fatal("the created user must be soft-deleted as a compensation on lookup failure")
	}
}

func TestUpdateTrainerUpdatesLinkedUserProfile(t *testing.T) {
	userID := uuid.NewString()
	trainerID := uuid.NewString()
	updated := &models.User{ID: userID, Email: "new@ryze.local", FirstName: "Renamed", LastName: "Trainer"}
	repo := &stubTrainerRepo{all: []models.Trainer{{ID: trainerID, UserID: userID}}}
	svc := newTestService(repo, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{user: updated})

	firstName := "Renamed"
	email := "new@ryze.local"
	result, err := svc.UpdateTrainer(context.Background(), trainerID, admin_trainers.UpdateTrainerInput{
		Email:     &email,
		FirstName: &firstName,
	})
	if err != nil {
		t.Fatalf("UpdateTrainer: %v", err)
	}
	if result.Trainer.ID != trainerID {
		t.Fatalf("expected trainer %q, got %q", trainerID, result.Trainer.ID)
	}
	if result.User.ID != userID || result.User.FirstName != firstName || result.User.Email != email {
		t.Fatalf("expected updated user profile, got %+v", result.User)
	}
}

func TestUpdateTrainerRequiresAtLeastOneField(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{}, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	_, err := svc.UpdateTrainer(context.Background(), uuid.NewString(), admin_trainers.UpdateTrainerInput{})
	if !errors.Is(err, admin_trainers.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateTrainerRejectsInvalidID(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{}, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})
	firstName := "Valid"

	for _, id := range []string{"", "not-a-uuid"} {
		if _, err := svc.UpdateTrainer(context.Background(), id, admin_trainers.UpdateTrainerInput{FirstName: &firstName}); !errors.Is(err, admin_trainers.ErrInvalidInput) {
			t.Fatalf("id %q: expected ErrInvalidInput, got %v", id, err)
		}
	}
}

func TestUpdateTrainerMapsTrainerNotFound(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{}, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})
	firstName := "Nobody"

	_, err := svc.UpdateTrainer(context.Background(), uuid.NewString(), admin_trainers.UpdateTrainerInput{FirstName: &firstName})
	if !errors.Is(err, admin_trainers.ErrTrainerNotFound) {
		t.Fatalf("expected ErrTrainerNotFound, got %v", err)
	}
}

func TestUpdateTrainerMapsUserInactive(t *testing.T) {
	userID := uuid.NewString()
	trainerID := uuid.NewString()
	repo := &stubTrainerRepo{all: []models.Trainer{{ID: trainerID, UserID: userID}}}
	svc := newTestService(repo, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{err: admin_users.ErrUserNotFound})

	firstName := "Renamed"
	_, err := svc.UpdateTrainer(context.Background(), trainerID, admin_trainers.UpdateTrainerInput{FirstName: &firstName})
	if !errors.Is(err, admin_trainers.ErrUserInactive) {
		t.Fatalf("expected ErrUserInactive, got %v", err)
	}
}

func TestUpdateTrainerMapsDuplicateEmail(t *testing.T) {
	userID := uuid.NewString()
	trainerID := uuid.NewString()
	repo := &stubTrainerRepo{all: []models.Trainer{{ID: trainerID, UserID: userID}}}
	svc := newTestService(repo, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{err: admin_users.ErrDuplicateEmail})

	email := "taken@ryze.local"
	_, err := svc.UpdateTrainer(context.Background(), trainerID, admin_trainers.UpdateTrainerInput{Email: &email})
	if !errors.Is(err, admin_trainers.ErrDuplicateEmail) {
		t.Fatalf("expected ErrDuplicateEmail, got %v", err)
	}
}

func TestUpdateTrainerMapsInvalidInputFromUpdater(t *testing.T) {
	userID := uuid.NewString()
	trainerID := uuid.NewString()
	repo := &stubTrainerRepo{all: []models.Trainer{{ID: trainerID, UserID: userID}}}
	svc := newTestService(repo, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{err: admin_users.ErrInvalidInput})

	email := "not-an-email"
	_, err := svc.UpdateTrainer(context.Background(), trainerID, admin_trainers.UpdateTrainerInput{Email: &email})
	if !errors.Is(err, admin_trainers.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateTrainerWrapsUpdaterFailure(t *testing.T) {
	userID := uuid.NewString()
	trainerID := uuid.NewString()
	repo := &stubTrainerRepo{all: []models.Trainer{{ID: trainerID, UserID: userID}}}
	svc := newTestService(repo, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{err: errRepoFailure})

	firstName := "Renamed"
	if _, err := svc.UpdateTrainer(context.Background(), trainerID, admin_trainers.UpdateTrainerInput{FirstName: &firstName}); err == nil {
		t.Fatal("expected an error")
	}
}

func TestSoftDeleteTrainerSucceeds(t *testing.T) {
	id := uuid.NewString()
	repo := &stubTrainerRepo{all: []models.Trainer{{ID: id, UserID: uuid.NewString()}}}
	svc := newTestService(repo, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	if err := svc.SoftDeleteTrainer(context.Background(), id); err != nil {
		t.Fatalf("SoftDeleteTrainer: %v", err)
	}
	if repo.deletedID != id {
		t.Fatalf("expected soft delete for %q, got %q", id, repo.deletedID)
	}
}

func TestSoftDeleteTrainerRejectsInvalidID(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{}, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	for _, id := range []string{"", "not-a-uuid", "x"} {
		if err := svc.SoftDeleteTrainer(context.Background(), id); !errors.Is(err, admin_trainers.ErrInvalidInput) {
			t.Fatalf("id %q: expected ErrInvalidInput, got %v", id, err)
		}
	}
}

func TestSoftDeleteTrainerMapsNotFound(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{}, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	err := svc.SoftDeleteTrainer(context.Background(), uuid.NewString())
	if !errors.Is(err, admin_trainers.ErrTrainerNotFound) {
		t.Fatalf("expected ErrTrainerNotFound, got %v", err)
	}
}

func TestSoftDeleteTrainerWrapsRepositoryFailure(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{deleteErr: errRepoFailure}, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	if err := svc.SoftDeleteTrainer(context.Background(), uuid.NewString()); err == nil {
		t.Fatal("expected an error")
	}
}

func TestReactivateTrainerSucceeds(t *testing.T) {
	userID := uuid.NewString()
	id := uuid.NewString()
	user := models.User{ID: userID}
	repo := &stubTrainerRepo{all: []models.Trainer{{ID: id, UserID: userID, DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true}}}}
	svc := newTestService(repo, &stubUserRepo{users: []models.User{user}}, stubRegistrar{}, stubUserUpdater{})

	result, err := svc.ReactivateTrainer(context.Background(), id)
	if err != nil {
		t.Fatalf("ReactivateTrainer: %v", err)
	}
	if repo.reactivatedID != id {
		t.Fatalf("expected reactivation for %q, got %q", id, repo.reactivatedID)
	}
	if result.Trainer.ID != id || result.Trainer.UserID != userID || result.Trainer.DeletedAt.Valid {
		t.Fatalf("expected active trainer with preserved link, got %+v", result.Trainer)
	}
}

func TestReactivateTrainerRejectsAlreadyActive(t *testing.T) {
	id := uuid.NewString()
	repo := &stubTrainerRepo{all: []models.Trainer{{ID: id, UserID: uuid.NewString()}}}
	svc := newTestService(repo, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	if _, err := svc.ReactivateTrainer(context.Background(), id); !errors.Is(err, admin_trainers.ErrAlreadyActive) {
		t.Fatalf("expected ErrAlreadyActive, got %v", err)
	}
}

func TestReactivateTrainerMapsNotFound(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{}, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	if _, err := svc.ReactivateTrainer(context.Background(), uuid.NewString()); !errors.Is(err, admin_trainers.ErrTrainerNotFound) {
		t.Fatalf("expected ErrTrainerNotFound, got %v", err)
	}
}

func TestReactivateTrainerRejectsInvalidID(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{}, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	for _, id := range []string{"", "not-a-uuid"} {
		if _, err := svc.ReactivateTrainer(context.Background(), id); !errors.Is(err, admin_trainers.ErrInvalidInput) {
			t.Fatalf("id %q: expected ErrInvalidInput, got %v", id, err)
		}
	}
}

func TestReactivateTrainerMapsUserInactive(t *testing.T) {
	id := uuid.NewString()
	repo := &stubTrainerRepo{all: []models.Trainer{{ID: id, UserID: uuid.NewString(), DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true}}}}
	svc := newTestService(repo, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	if _, err := svc.ReactivateTrainer(context.Background(), id); !errors.Is(err, admin_trainers.ErrUserInactive) {
		t.Fatalf("expected ErrUserInactive, got %v", err)
	}
}

func TestReactivateTrainerMapsAlreadyLinked(t *testing.T) {
	userID := uuid.NewString()
	id := uuid.NewString()
	user := models.User{ID: userID}
	repo := &stubTrainerRepo{all: []models.Trainer{
		{ID: id, UserID: userID, DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true}},
		{ID: uuid.NewString(), UserID: userID},
	}}
	svc := newTestService(repo, &stubUserRepo{users: []models.User{user}}, stubRegistrar{}, stubUserUpdater{})

	if _, err := svc.ReactivateTrainer(context.Background(), id); !errors.Is(err, admin_trainers.ErrTrainerAlreadyLinked) {
		t.Fatalf("expected ErrTrainerAlreadyLinked, got %v", err)
	}
}

func TestReactivateTrainerWrapsRepositoryFailure(t *testing.T) {
	svc := newTestService(&stubTrainerRepo{getErr: errRepoFailure}, &stubUserRepo{}, stubRegistrar{}, stubUserUpdater{})

	if _, err := svc.ReactivateTrainer(context.Background(), uuid.NewString()); err == nil {
		t.Fatal("expected an error")
	}
}

func TestReactivateTrainerWrapsReactivateFailure(t *testing.T) {
	id := uuid.NewString()
	userID := uuid.NewString()
	repo := &stubTrainerRepo{
		all:           []models.Trainer{{ID: id, UserID: userID, DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true}}},
		reactivateErr: errRepoFailure,
	}
	svc := newTestService(repo, &stubUserRepo{users: []models.User{{ID: userID}}}, stubRegistrar{}, stubUserUpdater{})

	if _, err := svc.ReactivateTrainer(context.Background(), id); err == nil {
		t.Fatal("expected an error")
	}
}
