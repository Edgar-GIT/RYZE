package trainer_client_insights

import (
	"context"
	"errors"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/statistics"
	"ryze/backend/services/trainer_clients"
	"ryze/backend/services/workout_sessions"
)

const (
	testTrainerID = "11111111-1111-1111-1111-111111111111"
	testClientID  = "22222222-2222-2222-2222-222222222222"
	testSessionID = "33333333-3333-3333-3333-333333333333"
	testExercise  = "44444444-4444-4444-4444-444444444444"
)

type fakeClientAccess struct {
	err   error
	calls int
}

func (f *fakeClientAccess) GetClient(_ context.Context, _, _ string) (*trainer_clients.Client, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return &trainer_clients.Client{}, nil
}

type fakeStatistics struct {
	result *statistics.ClientStatisticsResponse
	err    error
	calls  int
}

func (f *fakeStatistics) GetClientStatistics(_ context.Context, _ string) (*statistics.ClientStatisticsResponse, error) {
	f.calls++
	return f.result, f.err
}

type fakeSessionService struct {
	result   *workout_sessions.Session
	err      error
	lastUser string
	lastID   string
	calls    int
}

func (f *fakeSessionService) Get(_ context.Context, userID, sessionID string) (*workout_sessions.Session, error) {
	f.calls++
	f.lastUser = userID
	f.lastID = sessionID
	return f.result, f.err
}

type fakeInsightRepository struct {
	assignments    []models.ProgramAssignment
	entitlements   []models.Entitlement
	activations    []models.FreeProgramActivation
	sessions       []repositories.TrainerClientSessionSummary
	active         *repositories.TrainerClientSessionSummary
	legacy         repositories.TrainerClientLegacySummary
	exercises      []repositories.TrainerClientExerciseSummary
	exerciseSess   []models.WorkoutSession
	count          int64
	err            error
	listCalls      int
	listPageLimit  int
	listPageOffset int
	historyLimit   int
}

func (f *fakeInsightRepository) ListActiveAssignmentsByUser(context.Context, string) ([]models.ProgramAssignment, error) {
	return f.assignments, f.err
}

func (f *fakeInsightRepository) ListActiveEntitlementsByUser(context.Context, string) ([]models.Entitlement, error) {
	return f.entitlements, f.err
}

func (f *fakeInsightRepository) ListActiveFreeActivationsByUser(context.Context, string) ([]models.FreeProgramActivation, error) {
	return f.activations, f.err
}

func (f *fakeInsightRepository) ListSessions(_ context.Context, _ string, limit, offset int) ([]repositories.TrainerClientSessionSummary, error) {
	f.listCalls++
	f.listPageLimit = limit
	f.listPageOffset = offset
	return f.sessions, f.err
}

func (f *fakeInsightRepository) CountSessions(context.Context, string) (int64, error) {
	return f.count, f.err
}

func (f *fakeInsightRepository) FindActiveSession(context.Context, string) (*repositories.TrainerClientSessionSummary, error) {
	return f.active, f.err
}

func (f *fakeInsightRepository) LegacySummary(context.Context, string) (repositories.TrainerClientLegacySummary, error) {
	return f.legacy, f.err
}

func (f *fakeInsightRepository) ListExerciseSummaries(context.Context, string) ([]repositories.TrainerClientExerciseSummary, error) {
	return f.exercises, f.err
}

func (f *fakeInsightRepository) ListExerciseHistory(_ context.Context, _, _ string, limit int) ([]models.WorkoutSession, error) {
	f.historyLimit = limit
	return f.exerciseSess, f.err
}

func newTestService(clients ClientAccess, stats StatisticsService, sessions SessionService, repo repositories.TrainerClientInsightRepository) *Service {
	service := NewService(clients, stats, sessions, repo)
	service.now = func() time.Time { return time.Date(2026, 11, 6, 12, 0, 30, 0, time.UTC) }
	return service
}

func TestServiceRejectsClientWithoutRelationship(t *testing.T) {
	clients := &fakeClientAccess{err: trainer_clients.ErrClientNotFound}
	stats := &fakeStatistics{result: &statistics.ClientStatisticsResponse{}}
	sessions := &fakeSessionService{result: &workout_sessions.Session{}}
	repo := &fakeInsightRepository{}
	service := newTestService(clients, stats, sessions, repo)

	if _, err := service.GetOverview(context.Background(), testTrainerID, testClientID); !errors.Is(err, ErrClientNotFound) {
		t.Fatalf("GetOverview: expected ErrClientNotFound, got %v", err)
	}
	if _, err := service.GetStatistics(context.Background(), testTrainerID, testClientID); !errors.Is(err, ErrClientNotFound) {
		t.Fatalf("GetStatistics: expected ErrClientNotFound, got %v", err)
	}
	if _, err := service.ListWorkoutSessions(context.Background(), testTrainerID, testClientID, 1, 20); !errors.Is(err, ErrClientNotFound) {
		t.Fatalf("ListWorkoutSessions: expected ErrClientNotFound, got %v", err)
	}
	if _, err := service.GetWorkoutSession(context.Background(), testTrainerID, testClientID, testSessionID); !errors.Is(err, ErrClientNotFound) {
		t.Fatalf("GetWorkoutSession: expected ErrClientNotFound, got %v", err)
	}
	if _, err := service.GetExerciseHistory(context.Background(), testTrainerID, testClientID, testExercise, 20); !errors.Is(err, ErrClientNotFound) {
		t.Fatalf("GetExerciseHistory: expected ErrClientNotFound, got %v", err)
	}
	if stats.calls != 0 || sessions.calls != 0 || repo.listCalls != 0 {
		t.Fatalf("unrelated client reached downstream services: stats=%d sessions=%d repo=%d", stats.calls, sessions.calls, repo.listCalls)
	}
}

func TestServiceBuildsOverviewAccessAndActiveSession(t *testing.T) {
	lastResumed := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	legacyAt := time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)
	recentAt := time.Date(2026, 11, 4, 8, 0, 0, 0, time.UTC)
	expires := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)

	repo := &fakeInsightRepository{
		assignments: []models.ProgramAssignment{
			{
				ID:        "assignment-other",
				TrainerID: "99999999-9999-9999-9999-999999999999",
				ProgramID: "program-other",
				Program:   models.Program{ID: "program-other", Name: "Other plan", Status: models.ProgramStatusPublished},
				CreatedAt: legacyAt,
			},
			{
				ID:        "assignment-current",
				TrainerID: testTrainerID,
				ProgramID: "program-current",
				Program:   models.Program{ID: "program-current", Name: "Current plan", Status: models.ProgramStatusPublished},
				CreatedAt: legacyAt,
			},
		},
		entitlements: []models.Entitlement{{
			ProgramID: "program-purchased",
			Program:   models.Program{ID: "program-purchased", Name: "Purchased plan", Status: models.ProgramStatusPublished},
			ExpiresAt: &expires,
			CreatedAt: legacyAt,
		}},
		activations: []models.FreeProgramActivation{{
			ProgramID:   "program-free",
			Program:     models.Program{ID: "program-free", Name: "Free plan", Status: models.ProgramStatusPublished},
			ActivatedAt: legacyAt,
		}},
		active: &repositories.TrainerClientSessionSummary{
			ID:                 "active-session",
			ProgramName:        "Current plan",
			Status:             models.WorkoutSessionStatusInProgress,
			StartedAt:          legacyAt,
			LastResumedAt:      &lastResumed,
			AccumulatedSeconds: 10,
			TotalSets:          4,
			CompletedSets:      2,
		},
		sessions: []repositories.TrainerClientSessionSummary{{
			ID:            "recent-session",
			ProgramName:   "Current plan",
			Status:        models.WorkoutSessionStatusCompleted,
			StartedAt:     recentAt,
			CompletedAt:   &recentAt,
			CompletedSets: 3,
			TotalSets:     3,
		}},
		legacy: repositories.TrainerClientLegacySummary{Count: 2, LastCompletedAt: &legacyAt},
		exercises: []repositories.TrainerClientExerciseSummary{{
			ExerciseID:      testExercise,
			Name:            "Squat",
			SessionCount:    3,
			LastPerformedAt: legacyAt,
		}},
	}
	stats := &fakeStatistics{result: &statistics.ClientStatisticsResponse{TotalExecutions: 4}}
	service := newTestService(&fakeClientAccess{}, stats, &fakeSessionService{}, repo)

	overview, err := service.GetOverview(context.Background(), testTrainerID, testClientID)
	if err != nil {
		t.Fatalf("GetOverview: %v", err)
	}
	if overview.ActiveSession == nil || overview.ActiveSession.ElapsedSeconds != 40 {
		t.Fatalf("active session elapsed = %+v, want 40 seconds", overview.ActiveSession)
	}
	if overview.Statistics == nil || overview.Statistics.TotalExecutions != 4 {
		t.Fatalf("statistics not embedded: %+v", overview.Statistics)
	}
	if len(overview.Access.Assignments) != 2 {
		t.Fatalf("assignments = %d, want 2", len(overview.Access.Assignments))
	}
	if overview.Access.Assignments[0].AssignmentID != "" {
		t.Fatalf("other trainer assignment id leaked: %q", overview.Access.Assignments[0].AssignmentID)
	}
	if overview.Access.Assignments[1].AssignmentID != "assignment-current" || !overview.Access.Assignments[1].OwnedByCurrentTrainer {
		t.Fatalf("current assignment not identified: %+v", overview.Access.Assignments[1])
	}
	if len(overview.Access.Purchases) != 1 || overview.Access.Purchases[0].ExpiresAt == nil {
		t.Fatalf("purchase access not mapped: %+v", overview.Access.Purchases)
	}
	if len(overview.Access.FreeActivations) != 1 {
		t.Fatalf("free activation not mapped: %+v", overview.Access.FreeActivations)
	}
	if len(overview.RecentSessions) != 1 || overview.RecentSessions[0].ID != "recent-session" {
		t.Fatalf("recent sessions not mapped: %+v", overview.RecentSessions)
	}
	if overview.Legacy.Count != 2 || overview.Legacy.LastCompletedAt == nil {
		t.Fatalf("legacy summary not mapped: %+v", overview.Legacy)
	}
	if len(overview.Exercises) != 1 || overview.Exercises[0].Name != "Squat" {
		t.Fatalf("exercise summaries not mapped: %+v", overview.Exercises)
	}
}

func TestServiceListSessionsValidatesPaginationAndUsesOffset(t *testing.T) {
	repo := &fakeInsightRepository{
		count:    45,
		sessions: []repositories.TrainerClientSessionSummary{{ID: "one", Status: models.WorkoutSessionStatusCompleted}},
	}
	service := newTestService(&fakeClientAccess{}, &fakeStatistics{}, &fakeSessionService{}, repo)

	if _, err := service.ListWorkoutSessions(context.Background(), testTrainerID, testClientID, 0, 20); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("page 0: expected ErrInvalidInput, got %v", err)
	}
	if _, err := service.ListWorkoutSessions(context.Background(), testTrainerID, testClientID, 1, 101); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("limit 101: expected ErrInvalidInput, got %v", err)
	}

	result, err := service.ListWorkoutSessions(context.Background(), testTrainerID, testClientID, 3, 10)
	if err != nil {
		t.Fatalf("ListWorkoutSessions: %v", err)
	}
	if repo.listPageLimit != 10 || repo.listPageOffset != 20 {
		t.Fatalf("repo pagination = limit %d offset %d, want 10/20", repo.listPageLimit, repo.listPageOffset)
	}
	if result.Total != 45 || result.TotalPages != 5 || result.Page != 3 || result.Limit != 10 {
		t.Fatalf("result pagination = %+v", result)
	}
}

func TestServiceWorkoutSessionUsesClientScope(t *testing.T) {
	sessions := &fakeSessionService{result: &workout_sessions.Session{ID: testSessionID}}
	service := newTestService(&fakeClientAccess{}, &fakeStatistics{}, sessions, &fakeInsightRepository{})

	session, err := service.GetWorkoutSession(context.Background(), testTrainerID, testClientID, testSessionID)
	if err != nil {
		t.Fatalf("GetWorkoutSession: %v", err)
	}
	if session.ID != testSessionID || sessions.lastUser != testClientID || sessions.lastID != testSessionID {
		t.Fatalf("session lookup scope = user %q session %q", sessions.lastUser, sessions.lastID)
	}

	if _, err := service.GetWorkoutSession(context.Background(), testTrainerID, testClientID, "not-a-uuid"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid session id: expected ErrInvalidInput, got %v", err)
	}
}

func TestServiceExerciseHistoryMapsPrescribedAndActualSets(t *testing.T) {
	reps := 10
	actualReps := 8
	weight := 80.5
	actualWeight := 75.0
	completedAt := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
	repo := &fakeInsightRepository{exerciseSess: []models.WorkoutSession{{
		ID:              testSessionID,
		ProgramID:       "program-1",
		ProgramName:     "Strength",
		WeekNumber:      2,
		WorkoutPosition: 1,
		CompletedAt:     &completedAt,
		Exercises: []models.WorkoutSessionExercise{{
			ExerciseID:   testExercise,
			Name:         "Back Squat",
			Instructions: "Brace and drive.",
			Sets: []models.WorkoutSessionSet{{
				ID:                 "set-1",
				SetNumber:          1,
				SetType:            "normal",
				PrescribedReps:     &reps,
				PrescribedWeightKg: &weight,
				ActualReps:         &actualReps,
				ActualWeightKg:     &actualWeight,
				Status:             models.WorkoutSetStatusCompleted,
			}},
		}},
	}}}
	service := newTestService(&fakeClientAccess{}, &fakeStatistics{}, &fakeSessionService{}, repo)

	history, err := service.GetExerciseHistory(context.Background(), testTrainerID, testClientID, testExercise, 0)
	if err != nil {
		t.Fatalf("GetExerciseHistory: %v", err)
	}
	if repo.historyLimit != DefaultHistoryLimit {
		t.Fatalf("history limit = %d, want %d", repo.historyLimit, DefaultHistoryLimit)
	}
	if history.Name != "Back Squat" || len(history.Sessions) != 1 {
		t.Fatalf("history = %+v", history)
	}
	sets := history.Sessions[0].Sets
	if len(sets) != 1 || sets[0].PrescribedReps == nil || *sets[0].PrescribedReps != 10 || sets[0].ActualReps == nil || *sets[0].ActualReps != 8 {
		t.Fatalf("prescribed/actual sets not mapped: %+v", sets)
	}
}

func TestServiceMapsStatisticsErrors(t *testing.T) {
	service := newTestService(
		&fakeClientAccess{},
		&fakeStatistics{err: statistics.ErrInvalidInput},
		&fakeSessionService{},
		&fakeInsightRepository{},
	)
	if _, err := service.GetStatistics(context.Background(), testTrainerID, testClientID); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid stats: expected ErrInvalidInput, got %v", err)
	}

	service = newTestService(
		&fakeClientAccess{},
		&fakeStatistics{err: errors.New("database down")},
		&fakeSessionService{},
		&fakeInsightRepository{},
	)
	if _, err := service.GetStatistics(context.Background(), testTrainerID, testClientID); !errors.Is(err, ErrStatisticsInternal) {
		t.Fatalf("stats failure: expected ErrStatisticsInternal, got %v", err)
	}
}
