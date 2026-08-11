package admin_users_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/admin_users"
)

var errRepoFailure = errors.New("repository failure")

// stubRepo simulates the data-access surface required by the service.
type stubRepo struct {
	users     []models.User
	total     int64
	listErr   error
	getErr    error
	deleteErr error
	deletedID string
}

func (s *stubRepo) ListActive(_ context.Context, _ int, _ int) ([]models.User, int64, error) {
	return s.users, s.total, s.listErr
}

func (s *stubRepo) FindByID(_ context.Context, _ string) (*models.User, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if len(s.users) == 0 {
		return nil, repositories.ErrUserNotFound
	}
	return &s.users[0], nil
}

func (s *stubRepo) DeleteAccount(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deletedID = id
	return nil
}

func TestListUsersReturnsPageAndTotal(t *testing.T) {
	users := []models.User{{ID: uuid.NewString(), Email: "a@ryze.local"}, {ID: uuid.NewString(), Email: "b@ryze.local"}}
	svc := admin_users.NewAdminUserService(&stubRepo{users: users, total: 2})

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
	svc := admin_users.NewAdminUserService(&stubRepo{total: 0})

	result, err := svc.ListUsers(context.Background(), 1, 10000)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if result.Limit != admin_users.MaxPageSize {
		t.Fatalf("expected limit clamped to %d, got %d", admin_users.MaxPageSize, result.Limit)
	}
}

func TestListUsersRejectsInvalidPagination(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{})

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
	svc := admin_users.NewAdminUserService(&stubRepo{listErr: errRepoFailure})

	if _, err := svc.ListUsers(context.Background(), 1, 20); err == nil {
		t.Fatal("expected an error")
	}
}

func TestGetUserReturnsUser(t *testing.T) {
	user := models.User{ID: uuid.NewString(), Email: "a@ryze.local"}
	svc := admin_users.NewAdminUserService(&stubRepo{users: []models.User{user}})

	got, err := svc.GetUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("expected user %q, got %q", user.ID, got.ID)
	}
}

func TestGetUserRejectsInvalidID(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{})

	for _, id := range []string{"", "not-a-uuid", "  "} {
		if _, err := svc.GetUser(context.Background(), id); !errors.Is(err, admin_users.ErrInvalidInput) {
			t.Fatalf("id %q: expected ErrInvalidInput, got %v", id, err)
		}
	}
}

func TestGetUserMapsNotFound(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{})

	_, err := svc.GetUser(context.Background(), uuid.NewString())
	if !errors.Is(err, admin_users.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestGetUserWrapsRepositoryFailure(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{getErr: errRepoFailure})

	if _, err := svc.GetUser(context.Background(), uuid.NewString()); err == nil {
		t.Fatal("expected an error")
	}
}

func TestSoftDeleteUserSucceeds(t *testing.T) {
	repo := &stubRepo{}
	svc := admin_users.NewAdminUserService(repo)
	id := uuid.NewString()

	if err := svc.SoftDeleteUser(context.Background(), id); err != nil {
		t.Fatalf("SoftDeleteUser: %v", err)
	}
	if repo.deletedID != id {
		t.Fatalf("expected soft delete for %q, got %q", id, repo.deletedID)
	}
}

func TestSoftDeleteUserRejectsInvalidID(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{})

	for _, id := range []string{"", "not-a-uuid", "x"} {
		if err := svc.SoftDeleteUser(context.Background(), id); !errors.Is(err, admin_users.ErrInvalidInput) {
			t.Fatalf("id %q: expected ErrInvalidInput, got %v", id, err)
		}
	}
}

func TestSoftDeleteUserMapsNotFound(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{deleteErr: repositories.ErrUserNotFound})

	err := svc.SoftDeleteUser(context.Background(), uuid.NewString())
	if !errors.Is(err, admin_users.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestSoftDeleteUserWrapsRepositoryFailure(t *testing.T) {
	svc := admin_users.NewAdminUserService(&stubRepo{deleteErr: errRepoFailure})

	if err := svc.SoftDeleteUser(context.Background(), uuid.NewString()); err == nil {
		t.Fatal("expected an error")
	}
}
