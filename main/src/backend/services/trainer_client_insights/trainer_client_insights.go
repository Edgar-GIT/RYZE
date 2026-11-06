package trainer_client_insights

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/statistics"
	"ryze/backend/services/trainer_clients"
	"ryze/backend/services/workout_sessions"
)

const (
	DefaultHistoryLimit = 20
	MaxHistoryLimit     = 100
	RecentSessionLimit  = 10
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrClientNotFound     = errors.New("trainer client not found")
	ErrSessionNotFound    = errors.New("workout session not found")
	ErrStatisticsInternal = errors.New("statistics unavailable")
	ErrSessionInternal    = errors.New("workout session unavailable")
	ErrInsightUnavailable = errors.New("trainer client insight unavailable")
)

type ClientAccess interface {
	GetClient(ctx context.Context, trainerID, userID string) (*trainer_clients.Client, error)
}

type StatisticsService interface {
	GetClientStatistics(ctx context.Context, userID string) (*statistics.ClientStatisticsResponse, error)
}

type SessionService interface {
	Get(ctx context.Context, userID, sessionID string) (*workout_sessions.Session, error)
}

type AccessOverview struct {
	Assignments     []AssignmentAccess `json:"assignments"`
	Purchases       []ProgramAccess    `json:"purchases"`
	FreeActivations []ProgramAccess    `json:"free_activations"`
}

type AssignmentAccess struct {
	AssignmentID          string `json:"assignment_id,omitempty"`
	ProgramID             string `json:"program_id"`
	ProgramName           string `json:"program_name"`
	ProgramStatus         string `json:"program_status"`
	AssignedAt            string `json:"assigned_at"`
	OwnedByCurrentTrainer bool   `json:"owned_by_current_trainer"`
	AccessStatus          string `json:"access_status"`
}

type ProgramAccess struct {
	ProgramID     string  `json:"program_id"`
	ProgramName   string  `json:"program_name"`
	ProgramStatus string  `json:"program_status"`
	GrantedAt     string  `json:"granted_at"`
	ExpiresAt     *string `json:"expires_at"`
	AccessStatus  string  `json:"access_status"`
}

type SessionSummary struct {
	ID              string  `json:"id"`
	ProgramID       string  `json:"program_id"`
	ProgramName     string  `json:"program_name"`
	WeekNumber      int     `json:"week_number"`
	WorkoutPosition int     `json:"workout_position"`
	Status          string  `json:"status"`
	StartedAt       string  `json:"started_at"`
	CompletedAt     *string `json:"completed_at"`
	AbandonedAt     *string `json:"abandoned_at"`
	ElapsedSeconds  int     `json:"elapsed_seconds"`
	TotalSets       int64   `json:"total_sets"`
	CompletedSets   int64   `json:"completed_sets"`
	SkippedSets     int64   `json:"skipped_sets"`
}

type LegacySummary struct {
	Count           int64   `json:"count"`
	LastCompletedAt *string `json:"last_completed_at"`
}

type ExerciseSummary struct {
	ExerciseID      string `json:"exercise_id"`
	Name            string `json:"name"`
	SessionCount    int64  `json:"session_count"`
	LastPerformedAt string `json:"last_performed_at"`
}

type Overview struct {
	Access         AccessOverview                       `json:"access"`
	Statistics     *statistics.ClientStatisticsResponse `json:"statistics"`
	ActiveSession  *SessionSummary                      `json:"active_session"`
	RecentSessions []SessionSummary                     `json:"recent_sessions"`
	Legacy         LegacySummary                        `json:"legacy"`
	Exercises      []ExerciseSummary                    `json:"exercises"`
}

type SessionList struct {
	Sessions   []SessionSummary `json:"-"`
	Page       int              `json:"-"`
	Limit      int              `json:"-"`
	Total      int64            `json:"-"`
	TotalPages int              `json:"-"`
}

type ExerciseHistorySet struct {
	ID                    string   `json:"id"`
	SetNumber             int      `json:"set_number"`
	SetType               string   `json:"set_type"`
	PrescribedReps        *int     `json:"prescribed_reps"`
	PrescribedWeightKg    *float64 `json:"prescribed_weight_kg"`
	PrescribedRIR         *int     `json:"prescribed_rir"`
	PrescribedRPE         *float64 `json:"prescribed_rpe"`
	PrescribedRestSeconds *int     `json:"prescribed_rest_seconds"`
	PrescribedTempo       string   `json:"prescribed_tempo"`
	ActualReps            *int     `json:"actual_reps"`
	ActualWeightKg        *float64 `json:"actual_weight_kg"`
	ActualRIR             *int     `json:"actual_rir"`
	ActualRPE             *float64 `json:"actual_rpe"`
	Status                string   `json:"status"`
	CompletedAt           *string  `json:"completed_at"`
}

type ExerciseHistorySession struct {
	SessionID       string               `json:"session_id"`
	ProgramID       string               `json:"program_id"`
	ProgramName     string               `json:"program_name"`
	WeekNumber      int                  `json:"week_number"`
	WorkoutPosition int                  `json:"workout_position"`
	CompletedAt     *string              `json:"completed_at"`
	ExerciseID      string               `json:"exercise_id"`
	ExerciseName    string               `json:"exercise_name"`
	Instructions    string               `json:"instructions"`
	Notes           string               `json:"notes"`
	Sets            []ExerciseHistorySet `json:"sets"`
}

type ExerciseHistory struct {
	ExerciseID string                   `json:"exercise_id"`
	Name       string                   `json:"name"`
	Sessions   []ExerciseHistorySession `json:"sessions"`
}

type Service struct {
	clients    ClientAccess
	statistics StatisticsService
	sessions   SessionService
	repo       repositories.TrainerClientInsightRepository
	now        func() time.Time
}

func NewService(
	clients ClientAccess,
	stats StatisticsService,
	sessions SessionService,
	repo repositories.TrainerClientInsightRepository,
) *Service {
	return &Service{
		clients:    clients,
		statistics: stats,
		sessions:   sessions,
		repo:       repo,
		now:        time.Now,
	}
}

func (s *Service) GetOverview(ctx context.Context, trainerID, userID string) (Overview, error) {
	if err := s.verifyClient(ctx, trainerID, userID); err != nil {
		return Overview{}, err
	}

	access, err := s.getAccess(ctx, trainerID, userID)
	if err != nil {
		return Overview{}, err
	}
	stats, err := s.getStatistics(ctx, userID)
	if err != nil {
		return Overview{}, err
	}
	active, err := s.repo.FindActiveSession(ctx, userID)
	if err != nil {
		return Overview{}, fmt.Errorf("%w: %v", ErrInsightUnavailable, err)
	}
	recent, err := s.repo.ListSessions(ctx, userID, RecentSessionLimit, 0)
	if err != nil {
		return Overview{}, fmt.Errorf("%w: %v", ErrInsightUnavailable, err)
	}
	legacy, err := s.repo.LegacySummary(ctx, userID)
	if err != nil {
		return Overview{}, fmt.Errorf("%w: %v", ErrInsightUnavailable, err)
	}
	exercises, err := s.repo.ListExerciseSummaries(ctx, userID)
	if err != nil {
		return Overview{}, fmt.Errorf("%w: %v", ErrInsightUnavailable, err)
	}

	overview := Overview{
		Access:         access,
		Statistics:     stats,
		RecentSessions: make([]SessionSummary, 0, len(recent)),
		Legacy:         LegacySummary{Count: legacy.Count},
		Exercises:      make([]ExerciseSummary, 0, len(exercises)),
	}
	if active != nil {
		summary := s.toSessionSummary(*active)
		overview.ActiveSession = &summary
	}
	for _, session := range recent {
		overview.RecentSessions = append(overview.RecentSessions, s.toSessionSummary(session))
	}
	if legacy.LastCompletedAt != nil {
		value := formatTime(*legacy.LastCompletedAt)
		overview.Legacy.LastCompletedAt = &value
	}
	for _, exercise := range exercises {
		overview.Exercises = append(overview.Exercises, ExerciseSummary{
			ExerciseID:      exercise.ExerciseID,
			Name:            exercise.Name,
			SessionCount:    exercise.SessionCount,
			LastPerformedAt: formatTime(exercise.LastPerformedAt),
		})
	}
	return overview, nil
}

func (s *Service) getAccess(ctx context.Context, trainerID, userID string) (AccessOverview, error) {
	assignments, err := s.repo.ListActiveAssignmentsByUser(ctx, userID)
	if err != nil {
		return AccessOverview{}, fmt.Errorf("%w: %v", ErrInsightUnavailable, err)
	}
	entitlements, err := s.repo.ListActiveEntitlementsByUser(ctx, userID)
	if err != nil {
		return AccessOverview{}, fmt.Errorf("%w: %v", ErrInsightUnavailable, err)
	}
	activations, err := s.repo.ListActiveFreeActivationsByUser(ctx, userID)
	if err != nil {
		return AccessOverview{}, fmt.Errorf("%w: %v", ErrInsightUnavailable, err)
	}

	return accessOverviewFor(trainerID, assignments, entitlements, activations), nil
}

func (s *Service) GetStatistics(ctx context.Context, trainerID, userID string) (*statistics.ClientStatisticsResponse, error) {
	if err := s.verifyClient(ctx, trainerID, userID); err != nil {
		return nil, err
	}
	return s.getStatistics(ctx, userID)
}

func (s *Service) getStatistics(ctx context.Context, userID string) (*statistics.ClientStatisticsResponse, error) {
	stats, err := s.statistics.GetClientStatistics(ctx, userID)
	if err != nil {
		if errors.Is(err, statistics.ErrInvalidInput) {
			return nil, ErrInvalidInput
		}
		return nil, fmt.Errorf("%w: %v", ErrStatisticsInternal, err)
	}
	return stats, nil
}

func (s *Service) ListWorkoutSessions(ctx context.Context, trainerID, userID string, page, limit int) (SessionList, error) {
	if err := s.verifyClient(ctx, trainerID, userID); err != nil {
		return SessionList{}, err
	}
	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return SessionList{}, err
	}

	total, err := s.repo.CountSessions(ctx, userID)
	if err != nil {
		return SessionList{}, fmt.Errorf("%w: %v", ErrInsightUnavailable, err)
	}
	sessions, err := s.repo.ListSessions(ctx, userID, limit, (page-1)*limit)
	if err != nil {
		return SessionList{}, fmt.Errorf("%w: %v", ErrInsightUnavailable, err)
	}

	result := SessionList{
		Sessions:   make([]SessionSummary, 0, len(sessions)),
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages(total, limit),
	}
	for _, session := range sessions {
		result.Sessions = append(result.Sessions, s.toSessionSummary(session))
	}
	return result, nil
}

func (s *Service) GetWorkoutSession(ctx context.Context, trainerID, userID, sessionID string) (*workout_sessions.Session, error) {
	if err := s.verifyClient(ctx, trainerID, userID); err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(sessionID); err != nil {
		return nil, ErrInvalidInput
	}
	session, err := s.sessions.Get(ctx, userID, sessionID)
	if err != nil {
		if errors.Is(err, workout_sessions.ErrSessionNotFound) {
			return nil, ErrSessionNotFound
		}
		if errors.Is(err, workout_sessions.ErrInvalidInput) {
			return nil, ErrInvalidInput
		}
		return nil, fmt.Errorf("%w: %v", ErrSessionInternal, err)
	}
	return session, nil
}

func (s *Service) GetExerciseHistory(ctx context.Context, trainerID, userID, exerciseID string, limit int) (ExerciseHistory, error) {
	if err := s.verifyClient(ctx, trainerID, userID); err != nil {
		return ExerciseHistory{}, err
	}
	if _, err := uuid.Parse(exerciseID); err != nil {
		return ExerciseHistory{}, ErrInvalidInput
	}
	limit, err := normalizeLimit(limit)
	if err != nil {
		return ExerciseHistory{}, err
	}

	sessions, err := s.repo.ListExerciseHistory(ctx, userID, exerciseID, limit)
	if err != nil {
		return ExerciseHistory{}, fmt.Errorf("%w: %v", ErrInsightUnavailable, err)
	}

	history := ExerciseHistory{
		ExerciseID: exerciseID,
		Sessions:   make([]ExerciseHistorySession, 0, len(sessions)),
	}
	for _, session := range sessions {
		for _, exercise := range session.Exercises {
			if exercise.ExerciseID != exerciseID {
				continue
			}
			history.Name = exercise.Name
			history.Sessions = append(history.Sessions, exerciseHistorySession(session, exercise))
			break
		}
	}
	return history, nil
}

func (s *Service) verifyClient(ctx context.Context, trainerID, userID string) error {
	if _, err := uuid.Parse(trainerID); err != nil {
		return ErrClientNotFound
	}
	if _, err := uuid.Parse(userID); err != nil {
		return ErrClientNotFound
	}
	if _, err := s.clients.GetClient(ctx, trainerID, userID); err != nil {
		if errors.Is(err, trainer_clients.ErrClientNotFound) || errors.Is(err, trainer_clients.ErrInvalidInput) {
			return ErrClientNotFound
		}
		return fmt.Errorf("%w: %v", ErrInsightUnavailable, err)
	}
	return nil
}

func (s *Service) toSessionSummary(summary repositories.TrainerClientSessionSummary) SessionSummary {
	elapsed := summary.AccumulatedSeconds
	if summary.Status == models.WorkoutSessionStatusInProgress && summary.LastResumedAt != nil {
		live := int(s.now().Sub(*summary.LastResumedAt).Seconds())
		if live > 0 {
			elapsed += live
		}
	}
	result := SessionSummary{
		ID:              summary.ID,
		ProgramID:       summary.ProgramID,
		ProgramName:     summary.ProgramName,
		WeekNumber:      summary.WeekNumber,
		WorkoutPosition: summary.WorkoutPosition,
		Status:          summary.Status,
		StartedAt:       formatTime(summary.StartedAt),
		ElapsedSeconds:  elapsed,
		TotalSets:       summary.TotalSets,
		CompletedSets:   summary.CompletedSets,
		SkippedSets:     summary.SkippedSets,
	}
	if summary.CompletedAt != nil {
		value := formatTime(*summary.CompletedAt)
		result.CompletedAt = &value
	}
	if summary.AbandonedAt != nil {
		value := formatTime(*summary.AbandonedAt)
		result.AbandonedAt = &value
	}
	return result
}

func accessOverviewFor(trainerID string, assignments []models.ProgramAssignment, entitlements []models.Entitlement, activations []models.FreeProgramActivation) AccessOverview {
	overview := AccessOverview{
		Assignments:     make([]AssignmentAccess, 0, len(assignments)),
		Purchases:       make([]ProgramAccess, 0, len(entitlements)),
		FreeActivations: make([]ProgramAccess, 0, len(activations)),
	}
	for _, assignment := range assignments {
		programName, programStatus, accessStatus := programAccess(assignment.Program)
		item := AssignmentAccess{
			ProgramID:             assignment.ProgramID,
			ProgramName:           programName,
			ProgramStatus:         programStatus,
			AssignedAt:            formatTime(assignment.CreatedAt),
			OwnedByCurrentTrainer: assignment.TrainerID == trainerID,
			AccessStatus:          accessStatus,
		}
		if item.OwnedByCurrentTrainer {
			item.AssignmentID = assignment.ID
		}
		overview.Assignments = append(overview.Assignments, item)
	}
	for _, entitlement := range entitlements {
		programName, programStatus, accessStatus := programAccess(entitlement.Program)
		item := ProgramAccess{
			ProgramID:     entitlement.ProgramID,
			ProgramName:   programName,
			ProgramStatus: programStatus,
			GrantedAt:     formatTime(entitlement.CreatedAt),
			AccessStatus:  accessStatus,
		}
		if entitlement.ExpiresAt != nil {
			value := formatTime(*entitlement.ExpiresAt)
			item.ExpiresAt = &value
		}
		overview.Purchases = append(overview.Purchases, item)
	}
	for _, activation := range activations {
		programName, programStatus, accessStatus := programAccess(activation.Program)
		item := ProgramAccess{
			ProgramID:     activation.ProgramID,
			ProgramName:   programName,
			ProgramStatus: programStatus,
			GrantedAt:     formatTime(activation.ActivatedAt),
			AccessStatus:  accessStatus,
		}
		if activation.ExpiresAt != nil {
			value := formatTime(*activation.ExpiresAt)
			item.ExpiresAt = &value
		}
		overview.FreeActivations = append(overview.FreeActivations, item)
	}
	return overview
}

func programAccess(program models.Program) (string, string, string) {
	if program.ID == "" {
		return "Programme unavailable", "unavailable", "unavailable"
	}
	name := program.Name
	if name == "" {
		name = "Untitled programme"
	}
	if program.DeletedAt.Valid {
		return name, "unavailable", "unavailable"
	}
	return name, program.Status, "active"
}

func exerciseHistorySession(session models.WorkoutSession, exercise models.WorkoutSessionExercise) ExerciseHistorySession {
	result := ExerciseHistorySession{
		SessionID:       session.ID,
		ProgramID:       session.ProgramID,
		ProgramName:     session.ProgramName,
		WeekNumber:      session.WeekNumber,
		WorkoutPosition: session.WorkoutPosition,
		ExerciseID:      exercise.ExerciseID,
		ExerciseName:    exercise.Name,
		Instructions:    exercise.Instructions,
		Notes:           exercise.Notes,
		Sets:            make([]ExerciseHistorySet, 0, len(exercise.Sets)),
	}
	if session.CompletedAt != nil {
		value := formatTime(*session.CompletedAt)
		result.CompletedAt = &value
	}
	for _, set := range exercise.Sets {
		item := ExerciseHistorySet{
			ID:                    set.ID,
			SetNumber:             set.SetNumber,
			SetType:               set.SetType,
			PrescribedReps:        set.PrescribedReps,
			PrescribedWeightKg:    set.PrescribedWeightKg,
			PrescribedRIR:         set.PrescribedRIR,
			PrescribedRPE:         set.PrescribedRPE,
			PrescribedRestSeconds: set.PrescribedRestSeconds,
			PrescribedTempo:       set.PrescribedTempo,
			ActualReps:            set.ActualReps,
			ActualWeightKg:        set.ActualWeightKg,
			ActualRIR:             set.ActualRIR,
			ActualRPE:             set.ActualRPE,
			Status:                set.Status,
		}
		if set.CompletedAt != nil {
			value := formatTime(*set.CompletedAt)
			item.CompletedAt = &value
		}
		result.Sets = append(result.Sets, item)
	}
	return result
}

func normalizePagination(page, limit int) (int, int, error) {
	if page <= 0 || limit <= 0 {
		return 0, 0, ErrInvalidInput
	}
	if limit > MaxHistoryLimit {
		return 0, 0, ErrInvalidInput
	}
	return page, limit, nil
}

func normalizeLimit(limit int) (int, error) {
	if limit == 0 {
		return DefaultHistoryLimit, nil
	}
	if limit < 0 || limit > MaxHistoryLimit {
		return 0, ErrInvalidInput
	}
	return limit, nil
}

func totalPages(total int64, limit int) int {
	if total == 0 {
		return 0
	}
	return int((total + int64(limit) - 1) / int64(limit))
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
