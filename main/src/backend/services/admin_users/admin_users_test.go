package admin_users_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/admin_users"
	"ryze/backend/services/password"
	"ryze/backend/services/registration"
)

var errRepoFailure = errors.New("repository failure")

// stubRegistrar simulates the registration dependency used by CreateUser.
type stubRegistrar struct {
	user *models.User
	err  error
}

func (r stubRegistrar) Register(_ context.Context, _ registration.RegisterInput) (*models.User, error) {
	return r.user, r.err
}

// stubHasher simulates the hashing dependency used by ResetUserPassword.
type stubHasher struct {
	hash string
	err  error
}

func (h stubHasher) HashPassword(_ string) (string, error) {
	return h.hash, h.err
}

// stubRepo is a stateful in-memory fake of the data-access surface required by
// the service. It mirrors the repository semantics: soft-deleted rows are kept
// and only surfaced through the IncludingDeleted/ListDeleted operations, and
// DeleteAccount increments the session version.
type stubRepo struct {
	all            []models.User
	listErr        error
	deletedListErr error
	getErr         error
	deleteErr      error
	updateErr      error
	clearErr       error
	passwordErr    error
	deletedID      string
	clearedID      string
	changedID      string
	changedHash    string
}

func (s *stubRepo) ListActive(_ context.Context, _ int, _ int) ([]models.User, int64, error) {
	if s.listErr != nil {
		return nil, 0, s.listErr
	}
	active := make([]models.User, 0)
	for _, u := range s.all {
		if !u.DeletedAt.Valid {
			active = append(active, u)
		}
	}
	return active, int64(len(active)), nil
}

func (s *stubRepo) ListDeleted(_ context.Context, _ int, _ int) ([]models.User, int64, error) {
	if s.deletedListErr != nil {
		return nil, 0, s.deletedListErr
	}
	deleted := make([]models.User, 0)
	for _, u := range s.all {
		if u.DeletedAt.Valid {
			deleted = append(deleted, u)
		}
	}
	return deleted, int64(len(deleted)), nil
}

func (s *stubRepo) FindByID(_ context.Context, id string) (*models.User, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	for i := range s.all {
		if s.all[i].ID == id && !s.all[i].DeletedAt.Valid {
			return &s.all[i], nil
		}
	}
	return nil, repositories.ErrUserNotFound
}

func (s *stubRepo) FindByIDIncludingDeleted(_ context.Context, id string) (*models.User, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	for i := range s.all {
		if s.all[i].ID == id {
			return &s.all[i], nil
		}
	}
	return nil, repositories.ErrUserNotFound
}

func (s *stubRepo) FindByEmailIncludingDeleted(_ context.Context, email string) (*models.User, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	for i := range s.all {
		if s.all[i].Email == email {
			return &s.all[i], nil
		}
	}
	return nil, repositories.ErrUserNotFound
}

func (s *stubRepo) DeleteAccount(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	for i := range s.all {
		if s.all[i].ID == id && !s.all[i].DeletedAt.Valid {
			now := time.Now()
			s.all[i].DeletedAt = gorm.DeletedAt{Time: now, Valid: true}
			s.all[i].SessionVersion++
			s.deletedID = id
			return nil
		}
	}
	return repositories.ErrUserNotFound
}

func (s *stubRepo) ClearDeletedAt(_ context.Context, id string) error {
	if s.clearErr != nil {
		return s.clearErr
	}
	for i := range s.all {
		if s.all[i].ID == id && s.all[i].DeletedAt.Valid {
			s.all[i].DeletedAt = gorm.DeletedAt{}
			s.clearedID = id
			return nil
		}
	}
	return repositories.ErrUserNotFound
}

func (s *stubRepo) Update(_ context.Context, user *models.User) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	for i := range s.all {
		if s.all[i].ID == user.ID {
			s.all[i] = *user
			return nil
		}
	}
	return repositories.ErrUserNotFound
}

func (s *stubRepo) ChangePassword(_ context.Context, id string, newHash string) error {
	if s.passwordErr != nil {
		return s.passwordErr
	}
	for i := range s.all {
		if s.all[i].ID == id && !s.all[i].DeletedAt.Valid {
			s.all[i].PasswordHash = newHash
			s.all[i].SessionVersion++
			s.changedID = id
			s.changedHash = newHash
			return nil
		}
	}
	return repositories.ErrUserNotFound
}

func TestListUsersReturnsPageAndTotal(t *testing.T) {
	repo := &stubRepo{all: []models.User{
		{ID: uuid.NewString(), Email: "a@ryze.local"},
		{ID: uuid.NewString(), Email: "b@ryze.local"},
	}}
	svc := admin_users.NewAdminUserService(repo, stubRegistrar{}, stubHasher{})

	result, err := svc.ListUsers(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
	if len(result.Users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(result.Users))
	}
	if result.Page != 1 || result.Limit != 20 {
		t.Fatalf("expected page 1 limit 20, got page %d limit %d", result.Page, result.Limit)
	}
}

func TestListUsersClampsOversizedLimit(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{}, stubRegistrar{}, stubHasher{})

	result, err := svc.ListUsers(context.Background(), 1, 10000)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if result.Limit != admin_users.MaxPageSize {
		t.Fatalf("expected limit clamped to %d, got %d", admin_users.MaxPageSize, result.Limit)
	}
}

func TestListUsersRejectsInvalidPagination(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{}, stubRegistrar{}, stubHasher{})

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
			if _, err := svc.ListUsers(context.Background(), tc.page, tc.limit); !errors.Is(err, admin_users.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestListUsersWrapsRepositoryFailure(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{listErr: errRepoFailure}, stubRegistrar{}, stubHasher{})

	if _, err := svc.ListUsers(context.Background(), 1, 20); err == nil {
		t.Fatal("expected an error")
	}
}

func TestListDeletedUsersReturnsDeletedOnly(t *testing.T) {
	repo := &stubRepo{all: []models.User{
		{ID: uuid.NewString(), Email: "active@ryze.local"},
		{ID: uuid.NewString(), Email: "gone@ryze.local", DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true}},
	}}
	svc := admin_users.NewAdminUserService(repo, stubRegistrar{}, stubHasher{})

	result, err := svc.ListDeletedUsers(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("ListDeletedUsers: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
	if len(result.Users) != 1 || result.Users[0].Email != "gone@ryze.local" {
		t.Fatalf("expected only the soft-deleted user, got %+v", result.Users)
	}
}

func TestListDeletedUsersClampsOversizedLimit(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{}, stubRegistrar{}, stubHasher{})

	result, err := svc.ListDeletedUsers(context.Background(), 1, 99999)
	if err != nil {
		t.Fatalf("ListDeletedUsers: %v", err)
	}
	if result.Limit != admin_users.MaxPageSize {
		t.Fatalf("expected limit clamped to %d, got %d", admin_users.MaxPageSize, result.Limit)
	}
}

func TestListDeletedUsersRejectsInvalidPagination(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{}, stubRegistrar{}, stubHasher{})

	for _, tc := range []struct {
		page  int
		limit int
	}{
		{page: 0, limit: 20},
		{page: 1, limit: 0},
	} {
		if _, err := svc.ListDeletedUsers(context.Background(), tc.page, tc.limit); !errors.Is(err, admin_users.ErrInvalidInput) {
			t.Fatalf("page %d limit %d: expected ErrInvalidInput, got %v", tc.page, tc.limit, err)
		}
	}
}

func TestListDeletedUsersWrapsRepositoryFailure(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{deletedListErr: errRepoFailure}, stubRegistrar{}, stubHasher{})

	if _, err := svc.ListDeletedUsers(context.Background(), 1, 20); err == nil {
		t.Fatal("expected an error")
	}
}

func TestGetUserReturnsUser(t *testing.T) {
	user := models.User{ID: uuid.NewString(), Email: "a@ryze.local"}
	svc := admin_users.NewAdminUserService(&stubRepo{all: []models.User{user}}, stubRegistrar{}, stubHasher{})

	got, err := svc.GetUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("expected user %q, got %q", user.ID, got.ID)
	}
}

func TestGetUserRejectsInvalidID(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{}, stubRegistrar{}, stubHasher{})

	for _, id := range []string{"", "not-a-uuid", "  "} {
		if _, err := svc.GetUser(context.Background(), id); !errors.Is(err, admin_users.ErrInvalidInput) {
			t.Fatalf("id %q: expected ErrInvalidInput, got %v", id, err)
		}
	}
}

func TestGetUserMapsNotFound(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{}, stubRegistrar{}, stubHasher{})

	_, err := svc.GetUser(context.Background(), uuid.NewString())
	if !errors.Is(err, admin_users.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestGetUserWrapsRepositoryFailure(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{getErr: errRepoFailure}, stubRegistrar{}, stubHasher{})

	if _, err := svc.GetUser(context.Background(), uuid.NewString()); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCreateUserReturnsCreatedUser(t *testing.T) {
	created := &models.User{ID: uuid.NewString(), Email: "new@ryze.local"}
	svc := admin_users.NewAdminUserService(&stubRepo{}, stubRegistrar{user: created}, stubHasher{})

	got, err := svc.CreateUser(context.Background(), admin_users.CreateUserInput{
		Email:     "new@ryze.local",
		Password:  "Password123!",
		FirstName: "New",
		LastName:  "User",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected user %q, got %q", created.ID, got.ID)
	}
}

func TestCreateUserMapsDuplicateEmail(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{}, stubRegistrar{err: repositories.ErrDuplicateEmail}, stubHasher{})

	_, err := svc.CreateUser(context.Background(), admin_users.CreateUserInput{
		Email:     "dup@ryze.local",
		Password:  "Password123!",
		FirstName: "Dup",
		LastName:  "User",
	})
	if !errors.Is(err, admin_users.ErrDuplicateEmail) {
		t.Fatalf("expected ErrDuplicateEmail, got %v", err)
	}
}

func TestCreateUserMapsInvalidInput(t *testing.T) {
	for _, regErr := range []error{registration.ErrInvalidInput, password.ErrEmptyPassword} {
		svc := admin_users.NewAdminUserService(&stubRepo{}, stubRegistrar{err: regErr}, stubHasher{})

		_, err := svc.CreateUser(context.Background(), admin_users.CreateUserInput{
			Email:     "bad@ryze.local",
			Password:  "Password123!",
			FirstName: "Bad",
			LastName:  "Input",
		})
		if !errors.Is(err, admin_users.ErrInvalidInput) {
			t.Fatalf("registrar error %v: expected ErrInvalidInput, got %v", regErr, err)
		}
	}
}

func TestCreateUserWrapsRepositoryFailure(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{}, stubRegistrar{err: errRepoFailure}, stubHasher{})

	_, err := svc.CreateUser(context.Background(), admin_users.CreateUserInput{
		Email:     "fail@ryze.local",
		Password:  "Password123!",
		FirstName: "Fail",
		LastName:  "User",
	})
	if err == nil || errors.Is(err, admin_users.ErrInvalidInput) || errors.Is(err, admin_users.ErrDuplicateEmail) {
		t.Fatalf("expected an unexpected failure, got %v", err)
	}
}

func TestUpdateUserUpdatesWhitelistedFields(t *testing.T) {
	id := uuid.NewString()
	repo := &stubRepo{all: []models.User{{ID: id, Email: "a@ryze.local", FirstName: "A", LastName: "B"}}}
	svc := admin_users.NewAdminUserService(repo, stubRegistrar{}, stubHasher{})

	firstName := "Renamed"
	email := "b@ryze.local"
	updated, err := svc.UpdateUser(context.Background(), id, admin_users.UpdateUserInput{
		Email:     &email,
		FirstName: &firstName,
	})
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	if updated.FirstName != firstName || updated.Email != email {
		t.Fatalf("expected updated fields, got %+v", updated)
	}
	if updated.LastName != "B" {
		t.Fatalf("untouched fields must be preserved, got %+v", updated)
	}
}

func TestUpdateUserRequiresAtLeastOneField(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{}, stubRegistrar{}, stubHasher{})

	_, err := svc.UpdateUser(context.Background(), uuid.NewString(), admin_users.UpdateUserInput{})
	if !errors.Is(err, admin_users.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateUserRejectsInvalidFields(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{}, stubRegistrar{}, stubHasher{})
	id := uuid.NewString()

	empty := ""
	invalidEmail := "not-an-email"
	firstName := "Valid"

	for _, tc := range []struct {
		name  string
		input admin_users.UpdateUserInput
	}{
		{name: "empty email", input: admin_users.UpdateUserInput{Email: &empty}},
		{name: "malformed email", input: admin_users.UpdateUserInput{Email: &invalidEmail}},
		{name: "empty first name", input: admin_users.UpdateUserInput{FirstName: &empty}},
		{name: "empty last name", input: admin_users.UpdateUserInput{LastName: &empty}},
		{name: "invalid id", input: admin_users.UpdateUserInput{FirstName: &firstName}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			targetID := id
			if tc.name == "invalid id" {
				targetID = "not-a-uuid"
			}
			if _, err := svc.UpdateUser(context.Background(), targetID, tc.input); !errors.Is(err, admin_users.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestUpdateUserMapsNotFound(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{}, stubRegistrar{}, stubHasher{})
	firstName := "Nobody"

	_, err := svc.UpdateUser(context.Background(), uuid.NewString(), admin_users.UpdateUserInput{FirstName: &firstName})
	if !errors.Is(err, admin_users.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUpdateUserRejectsEmailOwnedByOtherIdentity(t *testing.T) {
	id := uuid.NewString()
	other := models.User{ID: uuid.NewString(), Email: "taken@ryze.local"}
	repo := &stubRepo{all: []models.User{{ID: id, Email: "a@ryze.local"}, other}}
	svc := admin_users.NewAdminUserService(repo, stubRegistrar{}, stubHasher{})

	taken := "taken@ryze.local"
	_, err := svc.UpdateUser(context.Background(), id, admin_users.UpdateUserInput{Email: &taken})
	if !errors.Is(err, admin_users.ErrDuplicateEmail) {
		t.Fatalf("expected ErrDuplicateEmail, got %v", err)
	}
}

func TestUpdateUserAllowsUnchangedEmail(t *testing.T) {
	id := uuid.NewString()
	repo := &stubRepo{all: []models.User{{ID: id, Email: "a@ryze.local"}}}
	svc := admin_users.NewAdminUserService(repo, stubRegistrar{}, stubHasher{})

	same := "a@ryze.local"
	if _, err := svc.UpdateUser(context.Background(), id, admin_users.UpdateUserInput{Email: &same}); err != nil {
		t.Fatalf("unchanged email must be allowed, got %v", err)
	}
}

func TestUpdateUserMapsRepositoryDuplicateEmail(t *testing.T) {
	id := uuid.NewString()
	repo := &stubRepo{all: []models.User{{ID: id, Email: "a@ryze.local"}}, updateErr: repositories.ErrDuplicateEmail}
	svc := admin_users.NewAdminUserService(repo, stubRegistrar{}, stubHasher{})

	firstName := "Renamed"
	_, err := svc.UpdateUser(context.Background(), id, admin_users.UpdateUserInput{FirstName: &firstName})
	if !errors.Is(err, admin_users.ErrDuplicateEmail) {
		t.Fatalf("expected ErrDuplicateEmail, got %v", err)
	}
}

func TestUpdateUserWrapsRepositoryFailure(t *testing.T) {
	id := uuid.NewString()
	repo := &stubRepo{all: []models.User{{ID: id, Email: "a@ryze.local"}}, updateErr: errRepoFailure}
	svc := admin_users.NewAdminUserService(repo, stubRegistrar{}, stubHasher{})

	firstName := "Renamed"
	if _, err := svc.UpdateUser(context.Background(), id, admin_users.UpdateUserInput{FirstName: &firstName}); err == nil {
		t.Fatal("expected an error")
	}
}

func TestSoftDeleteUserSucceeds(t *testing.T) {
	id := uuid.NewString()
	repo := &stubRepo{all: []models.User{{ID: id, Email: "a@ryze.local"}}}
	svc := admin_users.NewAdminUserService(repo, stubRegistrar{}, stubHasher{})

	if err := svc.SoftDeleteUser(context.Background(), id); err != nil {
		t.Fatalf("SoftDeleteUser: %v", err)
	}
	if repo.deletedID != id {
		t.Fatalf("expected soft delete for %q, got %q", id, repo.deletedID)
	}
}

func TestSoftDeleteUserRejectsInvalidID(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{}, stubRegistrar{}, stubHasher{})

	for _, id := range []string{"", "not-a-uuid", "x"} {
		if err := svc.SoftDeleteUser(context.Background(), id); !errors.Is(err, admin_users.ErrInvalidInput) {
			t.Fatalf("id %q: expected ErrInvalidInput, got %v", id, err)
		}
	}
}

func TestSoftDeleteUserMapsNotFound(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{deleteErr: repositories.ErrUserNotFound}, stubRegistrar{}, stubHasher{})

	err := svc.SoftDeleteUser(context.Background(), uuid.NewString())
	if !errors.Is(err, admin_users.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestSoftDeleteUserWrapsRepositoryFailure(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{deleteErr: errRepoFailure}, stubRegistrar{}, stubHasher{})

	if err := svc.SoftDeleteUser(context.Background(), uuid.NewString()); err == nil {
		t.Fatal("expected an error")
	}
}

func TestReactivateUserSucceeds(t *testing.T) {
	id := uuid.NewString()
	repo := &stubRepo{all: []models.User{{ID: id, Email: "a@ryze.local", DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true}}}}
	svc := admin_users.NewAdminUserService(repo, stubRegistrar{}, stubHasher{})

	user, err := svc.ReactivateUser(context.Background(), id)
	if err != nil {
		t.Fatalf("ReactivateUser: %v", err)
	}
	if repo.clearedID != id {
		t.Fatalf("expected reactivation for %q, got %q", id, repo.clearedID)
	}
	if user.ID != id || user.DeletedAt.Valid {
		t.Fatalf("expected active user, got %+v", user)
	}
}

func TestReactivateUserRejectsAlreadyActive(t *testing.T) {
	id := uuid.NewString()
	repo := &stubRepo{all: []models.User{{ID: id, Email: "a@ryze.local"}}}
	svc := admin_users.NewAdminUserService(repo, stubRegistrar{}, stubHasher{})

	if _, err := svc.ReactivateUser(context.Background(), id); !errors.Is(err, admin_users.ErrAlreadyActive) {
		t.Fatalf("expected ErrAlreadyActive, got %v", err)
	}
}

func TestReactivateUserMapsNotFound(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{}, stubRegistrar{}, stubHasher{})

	if _, err := svc.ReactivateUser(context.Background(), uuid.NewString()); !errors.Is(err, admin_users.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestReactivateUserRejectsInvalidID(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{}, stubRegistrar{}, stubHasher{})

	for _, id := range []string{"", "not-a-uuid"} {
		if _, err := svc.ReactivateUser(context.Background(), id); !errors.Is(err, admin_users.ErrInvalidInput) {
			t.Fatalf("id %q: expected ErrInvalidInput, got %v", id, err)
		}
	}
}

func TestReactivateUserWrapsRepositoryFailure(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{getErr: errRepoFailure}, stubRegistrar{}, stubHasher{})

	if _, err := svc.ReactivateUser(context.Background(), uuid.NewString()); err == nil {
		t.Fatal("expected an error")
	}
}

func TestReactivateUserWrapsClearFailure(t *testing.T) {
	id := uuid.NewString()
	repo := &stubRepo{
		all:      []models.User{{ID: id, Email: "a@ryze.local", DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true}}},
		clearErr: errRepoFailure,
	}
	svc := admin_users.NewAdminUserService(repo, stubRegistrar{}, stubHasher{})

	if _, err := svc.ReactivateUser(context.Background(), id); err == nil {
		t.Fatal("expected an error")
	}
}

func TestResetUserPasswordSucceeds(t *testing.T) {
	id := uuid.NewString()
	repo := &stubRepo{all: []models.User{{ID: id, Email: "a@ryze.local"}}}
	svc := admin_users.NewAdminUserService(repo, stubRegistrar{}, stubHasher{hash: "argon2hash"})

	if err := svc.ResetUserPassword(context.Background(), id, "NewPassword123!"); err != nil {
		t.Fatalf("ResetUserPassword: %v", err)
	}
	if repo.changedID != id {
		t.Fatalf("expected password reset for %q, got %q", id, repo.changedID)
	}
	if repo.changedHash != "argon2hash" {
		t.Fatalf("expected hash to be stored, got %q", repo.changedHash)
	}
}

func TestResetUserPasswordRejectsEmptyPassword(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{}, stubRegistrar{}, stubHasher{})

	err := svc.ResetUserPassword(context.Background(), uuid.NewString(), "")
	if !errors.Is(err, password.ErrEmptyPassword) {
		t.Fatalf("expected ErrEmptyPassword, got %v", err)
	}
}

func TestResetUserPasswordRejectsInvalidID(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{}, stubRegistrar{}, stubHasher{})

	err := svc.ResetUserPassword(context.Background(), "", "NewPassword123!")
	if !errors.Is(err, admin_users.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestResetUserPasswordMapsNotFound(t *testing.T) {
	repo := &stubRepo{}
	svc := admin_users.NewAdminUserService(repo, stubRegistrar{}, stubHasher{hash: "argon2hash"})

	err := svc.ResetUserPassword(context.Background(), uuid.NewString(), "NewPassword123!")
	if !errors.Is(err, admin_users.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestResetUserPasswordWrapsHashingFailure(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{}, stubRegistrar{}, stubHasher{err: errRepoFailure})

	if err := svc.ResetUserPassword(context.Background(), uuid.NewString(), "NewPassword123!"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestResetUserPasswordWrapsRepositoryFailure(t *testing.T) {
	id := uuid.NewString()
	repo := &stubRepo{all: []models.User{{ID: id, Email: "a@ryze.local"}}, passwordErr: errRepoFailure}
	svc := admin_users.NewAdminUserService(repo, stubRegistrar{}, stubHasher{hash: "argon2hash"})

	if err := svc.ResetUserPassword(context.Background(), id, "NewPassword123!"); err == nil {
		t.Fatal("expected an error")
	}
}
