package statistics_test

import (
	"context"
	"errors"
	"testing"

	"ryze/backend/repositories"
	"ryze/backend/services/statistics"
)

const userID = "22222222-2222-2222-2222-222222222222"

var errRepoFailure = errors.New("repository failure")

// stubStatsRepo is an in-memory fake of the statistics data-access surface.
// It records the user id forwarded to the repository so tests can prove the
// service forwards the authentication-context identity and never invents it.
type stubStatsRepo struct {
	stats   repositories.ClientStatistics
	err     error
	gotUser string
}

func (s *stubStatsRepo) GetClientStats(_ context.Context, userID string) (repositories.ClientStatistics, error) {
	s.gotUser = userID
	if s.err != nil {
		return repositories.ClientStatistics{}, s.err
	}
	return s.stats, nil
}

func newService(repo *stubStatsRepo) statistics.Service {
	return statistics.NewService(repo)
}

func TestGetClientStatisticsSuccess(t *testing.T) {
	lastDate := "2026-01-15T10:00:00Z"
	repo := &stubStatsRepo{
		stats: repositories.ClientStatistics{
			HasActiveAssignment:     true,
			CurrentProgramName:      "Strength Builder",
			TotalExecutions:         5,
			UniqueWorkoutsCompleted: 3,
			TotalWorkoutsInProgram:  4,
			LastWorkoutDate:         &lastDate,
		},
	}
	svc := newService(repo)

	resp, err := svc.GetClientStatistics(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetClientStatistics: %v", err)
	}
	if repo.gotUser != userID {
		t.Fatalf("expected scope %q, got %q", userID, repo.gotUser)
	}
	if !resp.HasActiveAssignment {
		t.Fatal("expected HasActiveAssignment=true")
	}
	if resp.CurrentProgramName != "Strength Builder" {
		t.Fatalf("expected program name 'Strength Builder', got %q", resp.CurrentProgramName)
	}
	if resp.TotalExecutions != 5 {
		t.Fatalf("expected TotalExecutions=5, got %d", resp.TotalExecutions)
	}
	if resp.UniqueWorkoutsCompleted != 3 {
		t.Fatalf("expected UniqueWorkoutsCompleted=3, got %d", resp.UniqueWorkoutsCompleted)
	}
	if resp.TotalWorkoutsInProgram != 4 {
		t.Fatalf("expected TotalWorkoutsInProgram=4, got %d", resp.TotalWorkoutsInProgram)
	}
	if resp.CompletionPercentage != 75.0 {
		t.Fatalf("expected CompletionPercentage=75.0, got %f", resp.CompletionPercentage)
	}
	if resp.LastWorkoutDate == nil || *resp.LastWorkoutDate != lastDate {
		t.Fatalf("expected LastWorkoutDate=%q, got %v", lastDate, resp.LastWorkoutDate)
	}
}

func TestGetClientStatisticsNoAssignment(t *testing.T) {
	repo := &stubStatsRepo{
		stats: repositories.ClientStatistics{
			HasActiveAssignment: false,
		},
	}
	svc := newService(repo)

	resp, err := svc.GetClientStatistics(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetClientStatistics: %v", err)
	}
	if resp.HasActiveAssignment {
		t.Fatal("expected HasActiveAssignment=false")
	}
	if resp.CurrentProgramName != "" {
		t.Fatalf("expected empty program name, got %q", resp.CurrentProgramName)
	}
	if resp.CompletionPercentage != 0 {
		t.Fatalf("expected CompletionPercentage=0, got %f", resp.CompletionPercentage)
	}
	if resp.LastWorkoutDate != nil {
		t.Fatalf("expected nil LastWorkoutDate, got %v", resp.LastWorkoutDate)
	}
}

func TestGetClientStatisticsZeroWorkouts(t *testing.T) {
	repo := &stubStatsRepo{
		stats: repositories.ClientStatistics{
			HasActiveAssignment:     true,
			CurrentProgramName:      "Empty Program",
			TotalWorkoutsInProgram:  0,
			UniqueWorkoutsCompleted: 0,
		},
	}
	svc := newService(repo)

	resp, err := svc.GetClientStatistics(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetClientStatistics: %v", err)
	}
	if resp.CompletionPercentage != 0 {
		t.Fatalf("expected CompletionPercentage=0 for zero workouts, got %f", resp.CompletionPercentage)
	}
}

func TestGetClientStatisticsInvalidInput(t *testing.T) {
	svc := newService(&stubStatsRepo{})

	_, err := svc.GetClientStatistics(context.Background(), "")
	if !errors.Is(err, statistics.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetClientStatisticsRepoFailureNotExposed(t *testing.T) {
	svc := newService(&stubStatsRepo{err: errRepoFailure})

	_, err := svc.GetClientStatistics(context.Background(), userID)
	if err == nil || errors.Is(err, statistics.ErrInvalidInput) {
		t.Fatalf("expected an internal failure to be hidden, got %v", err)
	}
	if !errors.Is(err, errRepoFailure) {
		t.Fatalf("expected the repository failure to stay wrapped for logs, got %v", err)
	}
}
