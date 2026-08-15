package workout_history_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/workout_history"
)

const (
	userID    = "22222222-2222-2222-2222-222222222222"
	workoutID = "66666666-6666-6666-6666-666666666666"
)

var errRepoFailure = errors.New("repository failure")

// stubRepo is an in-memory fake of the workout history data-access surface. It
// records the user id and workout id forwarded to the repository so tests can
// prove the service forwards the authentication-context identity and the path
// workout id, and never invents either one.
type stubRepo struct {
	entry      *models.WorkoutHistory
	entries    []models.WorkoutHistory
	total      int64
	err        error
	gotUser    string
	gotWorkout string
	gotPage    int
	gotLimit   int
}

func (s *stubRepo) Create(_ context.Context, userID, programWorkoutID string, entry *models.WorkoutHistory) error {
	s.gotUser = userID
	s.gotWorkout = programWorkoutID
	if s.err != nil {
		return s.err
	}
	*entry = *s.entry
	return nil
}

func (s *stubRepo) ListByUser(_ context.Context, userID string, page, limit int) ([]models.WorkoutHistory, int64, error) {
	s.gotUser = userID
	s.gotPage = page
	s.gotLimit = limit
	if s.err != nil {
		return nil, 0, s.err
	}
	return s.entries, s.total, nil
}

func newService(repo *stubRepo) workout_history.Service {
	return workout_history.NewService(repo)
}

func validEntry() *models.WorkoutHistory {
	return &models.WorkoutHistory{
		ID:               "99999999-9999-9999-9999-999999999999",
		UserID:           userID,
		ProgramWorkoutID: workoutID,
		CompletedAt:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestCompleteWorkoutSuccess(t *testing.T) {
	repo := &stubRepo{entry: validEntry()}
	svc := newService(repo)

	entry, err := svc.CompleteWorkout(context.Background(), userID, workoutID)
	if err != nil {
		t.Fatalf("CompleteWorkout: %v", err)
	}
	if repo.gotUser != userID {
		t.Fatalf("expected scope %q, got %q", userID, repo.gotUser)
	}
	if repo.gotWorkout != workoutID {
		t.Fatalf("expected workout %q, got %q", workoutID, repo.gotWorkout)
	}
	if entry.ID == "" || entry.ProgramWorkoutID != workoutID {
		t.Fatalf("unexpected entry %+v", entry)
	}
	if entry.CompletedAt.IsZero() {
		t.Fatal("expected a completion timestamp")
	}
}

func TestCompleteWorkoutInvalidInput(t *testing.T) {
	svc := newService(&stubRepo{})

	cases := map[string]struct{ user, workout string }{
		"empty user":    {"", workoutID},
		"bad user":      {"not-a-uuid", workoutID},
		"empty workout": {userID, ""},
		"bad workout":   {userID, "not-a-uuid"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.CompleteWorkout(context.Background(), tc.user, tc.workout); !errors.Is(err, workout_history.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestCompleteWorkoutCollapsesNotFound(t *testing.T) {
	svc := newService(&stubRepo{err: repositories.ErrWorkoutNotFound})

	if _, err := svc.CompleteWorkout(context.Background(), userID, workoutID); !errors.Is(err, workout_history.ErrWorkoutNotFound) {
		t.Fatalf("expected ErrWorkoutNotFound, got %v", err)
	}
}

func TestCompleteWorkoutRepoFailureNotExposed(t *testing.T) {
	svc := newService(&stubRepo{err: errRepoFailure})

	_, err := svc.CompleteWorkout(context.Background(), userID, workoutID)
	if err == nil || errors.Is(err, workout_history.ErrInvalidInput) || errors.Is(err, workout_history.ErrWorkoutNotFound) {
		t.Fatalf("expected an internal failure to be hidden, got %v", err)
	}
	if !errors.Is(err, errRepoFailure) {
		t.Fatalf("expected the repository failure to stay wrapped for logs, got %v", err)
	}
}

func TestListHistorySuccess(t *testing.T) {
	repo := &stubRepo{entries: []models.WorkoutHistory{*validEntry()}, total: 1}
	svc := newService(repo)

	result, err := svc.ListHistory(context.Background(), userID, 1, 20)
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if repo.gotUser != userID {
		t.Fatalf("expected scope %q, got %q", userID, repo.gotUser)
	}
	if repo.gotPage != 1 || repo.gotLimit != 20 {
		t.Fatalf("expected pagination (1, 20), got (%d, %d)", repo.gotPage, repo.gotLimit)
	}
	if result.Total != 1 || len(result.Entries) != 1 || result.Entries[0].ProgramWorkoutID != workoutID {
		t.Fatalf("unexpected result %+v", result)
	}
}

func TestListHistoryClampsOversizedLimit(t *testing.T) {
	repo := &stubRepo{entries: []models.WorkoutHistory{*validEntry()}, total: 1}
	svc := newService(repo)

	result, err := svc.ListHistory(context.Background(), userID, 1, 10000)
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if repo.gotLimit != workout_history.MaxPageSize {
		t.Fatalf("expected limit clamped to %d, got %d", workout_history.MaxPageSize, repo.gotLimit)
	}
	if result.Limit != workout_history.MaxPageSize {
		t.Fatalf("expected result limit %d, got %d", workout_history.MaxPageSize, result.Limit)
	}
}

func TestListHistoryInvalidInput(t *testing.T) {
	svc := newService(&stubRepo{})

	cases := map[string]struct {
		user        string
		page, limit int
	}{
		"empty user":    {"", 1, 20},
		"bad user":      {"not-a-uuid", 1, 20},
		"page below 1":  {userID, 0, 20},
		"limit below 1": {userID, 1, 0},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.ListHistory(context.Background(), tc.user, tc.page, tc.limit); !errors.Is(err, workout_history.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestListHistoryRepoFailureNotExposed(t *testing.T) {
	svc := newService(&stubRepo{err: errRepoFailure})

	_, err := svc.ListHistory(context.Background(), userID, 1, 20)
	if err == nil || errors.Is(err, workout_history.ErrInvalidInput) {
		t.Fatalf("expected an internal failure to be hidden, got %v", err)
	}
	if !errors.Is(err, errRepoFailure) {
		t.Fatalf("expected the repository failure to stay wrapped for logs, got %v", err)
	}
}
