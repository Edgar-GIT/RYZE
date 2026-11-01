package workout_sessions

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/program_access"
	"ryze/backend/services/program_content"
)

var (
	// ErrInvalidInput indicates malformed IDs or set values.
	ErrInvalidInput = errors.New("invalid workout session input")
	// ErrWorkoutNotFound indicates the workout is not part of an accessible
	// program. It is deliberately indistinguishable from a missing program.
	ErrWorkoutNotFound = errors.New("workout not found")
	// ErrSessionNotFound indicates the session is missing or belongs to another
	// user.
	ErrSessionNotFound = errors.New("workout session not found")
	// ErrSessionConflict indicates another workout is already active.
	ErrSessionConflict = errors.New("another workout session is already active")
	// ErrSessionNotActive indicates the session is already finished.
	ErrSessionNotActive = errors.New("workout session is not active")
	// ErrSetNotFound indicates the set does not belong to the session.
	ErrSetNotFound = errors.New("workout set not found")
)

// AccessService resolves the authenticated client's right to read a program.
type AccessService interface {
	GetProgramAccess(ctx context.Context, userID, programID string) (*program_content.Program, error)
}

// SessionRepository is the workout session data-access surface.
type SessionRepository interface {
	Create(ctx context.Context, session *models.WorkoutSession) error
	FindActiveByUser(ctx context.Context, userID string) (*models.WorkoutSession, error)
	FindByIDAndUser(ctx context.Context, userID, sessionID string) (*models.WorkoutSession, error)
	ListByUser(ctx context.Context, userID string, limit int) ([]models.WorkoutSession, error)
	UpdateSet(ctx context.Context, userID, sessionID, setID string, update repositories.WorkoutSetUpdate) (*models.WorkoutSession, error)
	Pause(ctx context.Context, userID, sessionID string, now time.Time) (*models.WorkoutSession, error)
	Resume(ctx context.Context, userID, sessionID string, now time.Time) (*models.WorkoutSession, error)
	Complete(ctx context.Context, userID, sessionID string, now time.Time) (*models.WorkoutSession, error)
	Abandon(ctx context.Context, userID, sessionID string, now time.Time) (*models.WorkoutSession, error)
}

// SetInput carries optional actual values plus the requested set status.
type SetInput struct {
	ActualReps     *int
	ActualWeightKg *float64
	ActualRIR      *int
	ActualRPE      *float64
	Status         string
}

// Set is one prescribed set with its actual performance.
type Set struct {
	ID                    string
	SetNumber             int
	SetType               string
	PrescribedReps        *int
	PrescribedWeightKg    *float64
	PrescribedRIR         *int
	PrescribedRPE         *float64
	PrescribedRestSeconds *int
	PrescribedTempo       string
	ActualReps            *int
	ActualWeightKg        *float64
	ActualRIR             *int
	ActualRPE             *float64
	Status                string
	CompletedAt           *time.Time
	RestEndsAt            *time.Time
}

// Exercise is one prescription snapshot with its sets.
type Exercise struct {
	ID           string
	ExerciseID   string
	Position     int
	Name         string
	Instructions string
	Notes        string
	Sets         []Set
}

// Session is the safe client view of one workout execution.
type Session struct {
	ID               string
	ProgramID        string
	ProgramWorkoutID string
	ProgramName      string
	WeekNumber       int
	WorkoutPosition  int
	Status           string
	StartedAt        time.Time
	PausedAt         *time.Time
	CompletedAt      *time.Time
	AbandonedAt      *time.Time
	ElapsedSeconds   int
	TotalSets        int
	CompletedSets    int
	SkippedSets      int
	Exercises        []Exercise
}

// Service implements the authenticated workout execution lifecycle.
type Service interface {
	Start(ctx context.Context, userID, programID, workoutID string) (*Session, error)
	Active(ctx context.Context, userID string) (*Session, error)
	Get(ctx context.Context, userID, sessionID string) (*Session, error)
	UpdateSet(ctx context.Context, userID, sessionID, setID string, input SetInput) (*Session, error)
	Pause(ctx context.Context, userID, sessionID string) (*Session, error)
	Resume(ctx context.Context, userID, sessionID string) (*Session, error)
	Complete(ctx context.Context, userID, sessionID string) (*Session, error)
	Abandon(ctx context.Context, userID, sessionID string) (*Session, error)
	History(ctx context.Context, userID string, limit int) ([]Session, error)
}

type service struct {
	access   AccessService
	sessions SessionRepository
	now      func() time.Time
}

func NewService(access AccessService, sessions SessionRepository) Service {
	return &service{access: access, sessions: sessions, now: func() time.Time { return time.Now().UTC() }}
}

// Start begins a session for one workout in an accessible programme. Starting
// the workout that is already active returns it unchanged. Starting a different
// workout while another is active is a conflict, never a silent replacement.
func (s *service) Start(ctx context.Context, userID, programID, workoutID string) (*Session, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if err := validateID(programID, "program id"); err != nil {
		return nil, err
	}
	if err := validateID(workoutID, "workout id"); err != nil {
		return nil, err
	}

	program, err := s.access.GetProgramAccess(ctx, userID, programID)
	if err != nil {
		if errors.Is(err, program_access.ErrProgramNotAccessible) {
			return nil, ErrWorkoutNotFound
		}
		return nil, fmt.Errorf("failed to verify workout access: %w", err)
	}

	existing, err := s.sessions.FindActiveByUser(ctx, userID)
	if err == nil {
		if existing.ProgramWorkoutID == workoutID {
			return s.toSession(existing), nil
		}
		return nil, ErrSessionConflict
	}
	if !errors.Is(err, repositories.ErrWorkoutSessionNotFound) {
		return nil, fmt.Errorf("failed to check active workout session: %w", err)
	}

	week, workout, ok := findWorkout(program, workoutID)
	if !ok {
		return nil, ErrWorkoutNotFound
	}

	now := s.now()
	session := &models.WorkoutSession{
		UserID:           userID,
		ProgramID:        program.ID,
		ProgramWorkoutID: workout.ID,
		ProgramName:      program.Name,
		WeekNumber:       week.WeekNumber,
		WorkoutPosition:  workout.Position,
		Status:           models.WorkoutSessionStatusInProgress,
		StartedAt:        now,
		LastResumedAt:    &now,
	}
	for i := range workout.Exercises {
		exercise := &workout.Exercises[i]
		if exercise.Exercise.ID == "" {
			return nil, ErrWorkoutNotFound
		}
		snapshot := models.WorkoutSessionExercise{
			ExerciseID:   exercise.Exercise.ID,
			Position:     exercise.Position,
			Name:         exercise.Exercise.Name,
			Instructions: exercise.Instructions,
			Notes:        exercise.Notes,
			Sets:         make([]models.WorkoutSessionSet, 0, len(exercise.Sets)),
		}
		for k := range exercise.Sets {
			set := &exercise.Sets[k]
			snapshot.Sets = append(snapshot.Sets, models.WorkoutSessionSet{
				SetNumber:             set.SetNumber,
				SetType:               set.SetType,
				PrescribedReps:        set.Reps,
				PrescribedWeightKg:    set.WeightKg,
				PrescribedRIR:         set.RIR,
				PrescribedRPE:         set.RPE,
				PrescribedRestSeconds: set.RestSeconds,
				PrescribedTempo:       set.Tempo,
				Status:                models.WorkoutSetStatusPending,
			})
		}
		session.Exercises = append(session.Exercises, snapshot)
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		if errors.Is(err, repositories.ErrWorkoutSessionAlreadyActive) {
			active, findErr := s.sessions.FindActiveByUser(ctx, userID)
			if findErr == nil && active.ProgramWorkoutID == workoutID {
				return s.toSession(active), nil
			}
			return nil, ErrSessionConflict
		}
		return nil, fmt.Errorf("failed to start workout session: %w", err)
	}
	return s.toSession(session), nil
}

func (s *service) Active(ctx context.Context, userID string) (*Session, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	session, err := s.sessions.FindActiveByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrWorkoutSessionNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load active workout session: %w", err)
	}
	return s.toSession(session), nil
}

func (s *service) Get(ctx context.Context, userID, sessionID string) (*Session, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if err := validateID(sessionID, "session id"); err != nil {
		return nil, err
	}
	session, err := s.sessions.FindByIDAndUser(ctx, userID, sessionID)
	if err != nil {
		if errors.Is(err, repositories.ErrWorkoutSessionNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to load workout session: %w", err)
	}
	return s.toSession(session), nil
}

func (s *service) UpdateSet(ctx context.Context, userID, sessionID, setID string, input SetInput) (*Session, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if err := validateID(sessionID, "session id"); err != nil {
		return nil, err
	}
	if err := validateID(setID, "set id"); err != nil {
		return nil, err
	}

	session, err := s.sessions.FindByIDAndUser(ctx, userID, sessionID)
	if err != nil {
		if errors.Is(err, repositories.ErrWorkoutSessionNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to load workout session: %w", err)
	}
	if session.Status != models.WorkoutSessionStatusInProgress && session.Status != models.WorkoutSessionStatusPaused {
		return nil, ErrSessionNotActive
	}

	set, ok := findSet(session, setID)
	if !ok {
		return nil, ErrSetNotFound
	}

	status := input.Status
	if status == "" {
		status = set.Status
	}
	switch status {
	case models.WorkoutSetStatusPending, models.WorkoutSetStatusCompleted, models.WorkoutSetStatusSkipped:
	default:
		return nil, fmt.Errorf("%w: invalid set status", ErrInvalidInput)
	}

	reps := firstInt(input.ActualReps, set.ActualReps)
	weight := firstFloat(input.ActualWeightKg, set.ActualWeightKg)
	rir := firstInt(input.ActualRIR, set.ActualRIR)
	rpe := firstFloat(input.ActualRPE, set.ActualRPE)
	if status == models.WorkoutSetStatusPending {
		reps, weight, rir, rpe = nil, nil, nil, nil
	}
	if err := validateSetValues(reps, weight, rir, rpe); err != nil {
		return nil, err
	}
	if status == models.WorkoutSetStatusCompleted && (reps == nil || *reps <= 0) {
		return nil, fmt.Errorf("%w: actual reps are required to complete a set", ErrInvalidInput)
	}

	var completedAt *time.Time
	if status == models.WorkoutSetStatusCompleted {
		if set.CompletedAt != nil {
			completedAt = set.CompletedAt
		} else {
			now := s.now()
			completedAt = &now
		}
	}

	updated, err := s.sessions.UpdateSet(ctx, userID, sessionID, setID, repositories.WorkoutSetUpdate{
		ActualReps:     reps,
		ActualWeightKg: weight,
		ActualRIR:      rir,
		ActualRPE:      rpe,
		Status:         status,
		CompletedAt:    completedAt,
	})
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrWorkoutSessionNotFound):
			return nil, ErrSessionNotFound
		case errors.Is(err, repositories.ErrWorkoutSetNotFound):
			return nil, ErrSetNotFound
		case errors.Is(err, repositories.ErrWorkoutSessionNotActive):
			return nil, ErrSessionNotActive
		default:
			return nil, fmt.Errorf("failed to update workout set: %w", err)
		}
	}
	return s.toSession(updated), nil
}

func (s *service) Pause(ctx context.Context, userID, sessionID string) (*Session, error) {
	return s.transition(ctx, userID, sessionID, s.sessions.Pause)
}

func (s *service) Resume(ctx context.Context, userID, sessionID string) (*Session, error) {
	return s.transition(ctx, userID, sessionID, s.sessions.Resume)
}

func (s *service) Complete(ctx context.Context, userID, sessionID string) (*Session, error) {
	return s.transition(ctx, userID, sessionID, s.sessions.Complete)
}

func (s *service) Abandon(ctx context.Context, userID, sessionID string) (*Session, error) {
	return s.transition(ctx, userID, sessionID, s.sessions.Abandon)
}

type transitionFunc func(context.Context, string, string, time.Time) (*models.WorkoutSession, error)

func (s *service) transition(ctx context.Context, userID, sessionID string, transition transitionFunc) (*Session, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if err := validateID(sessionID, "session id"); err != nil {
		return nil, err
	}
	updated, err := transition(ctx, userID, sessionID, s.now())
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrWorkoutSessionNotFound):
			return nil, ErrSessionNotFound
		case errors.Is(err, repositories.ErrWorkoutSessionNotActive):
			return nil, ErrSessionNotActive
		default:
			return nil, fmt.Errorf("failed to update workout session: %w", err)
		}
	}
	return s.toSession(updated), nil
}

func (s *service) History(ctx context.Context, userID string, limit int) ([]Session, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := s.sessions.ListByUser(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list workout history: %w", err)
	}
	sessions := make([]Session, 0, len(rows))
	for i := range rows {
		sessions = append(sessions, *s.toSession(&rows[i]))
	}
	return sessions, nil
}

func (s *service) toSession(model *models.WorkoutSession) *Session {
	session := &Session{
		ID:               model.ID,
		ProgramID:        model.ProgramID,
		ProgramWorkoutID: model.ProgramWorkoutID,
		ProgramName:      model.ProgramName,
		WeekNumber:       model.WeekNumber,
		WorkoutPosition:  model.WorkoutPosition,
		Status:           model.Status,
		StartedAt:        model.StartedAt,
		PausedAt:         model.PausedAt,
		CompletedAt:      model.CompletedAt,
		AbandonedAt:      model.AbandonedAt,
		ElapsedSeconds:   model.AccumulatedSeconds,
		Exercises:        make([]Exercise, 0, len(model.Exercises)),
	}
	if model.Status == models.WorkoutSessionStatusInProgress && model.LastResumedAt != nil {
		seconds := int(s.now().Sub(*model.LastResumedAt).Seconds())
		if seconds > 0 {
			session.ElapsedSeconds += seconds
		}
	}

	for i := range model.Exercises {
		exercise := &model.Exercises[i]
		view := Exercise{
			ID:           exercise.ID,
			ExerciseID:   exercise.ExerciseID,
			Position:     exercise.Position,
			Name:         exercise.Name,
			Instructions: exercise.Instructions,
			Notes:        exercise.Notes,
			Sets:         make([]Set, 0, len(exercise.Sets)),
		}
		for k := range exercise.Sets {
			set := &exercise.Sets[k]
			view.Sets = append(view.Sets, s.toSet(set))
			session.TotalSets++
			switch set.Status {
			case models.WorkoutSetStatusCompleted:
				session.CompletedSets++
			case models.WorkoutSetStatusSkipped:
				session.SkippedSets++
			}
		}
		session.Exercises = append(session.Exercises, view)
	}
	return session
}

func (s *service) toSet(model *models.WorkoutSessionSet) Set {
	set := Set{
		ID:                    model.ID,
		SetNumber:             model.SetNumber,
		SetType:               model.SetType,
		PrescribedReps:        model.PrescribedReps,
		PrescribedWeightKg:    model.PrescribedWeightKg,
		PrescribedRIR:         model.PrescribedRIR,
		PrescribedRPE:         model.PrescribedRPE,
		PrescribedRestSeconds: model.PrescribedRestSeconds,
		PrescribedTempo:       model.PrescribedTempo,
		ActualReps:            model.ActualReps,
		ActualWeightKg:        model.ActualWeightKg,
		ActualRIR:             model.ActualRIR,
		ActualRPE:             model.ActualRPE,
		Status:                model.Status,
		CompletedAt:           model.CompletedAt,
	}
	if model.CompletedAt != nil && model.PrescribedRestSeconds != nil {
		restEndsAt := model.CompletedAt.Add(time.Duration(*model.PrescribedRestSeconds) * time.Second)
		set.RestEndsAt = &restEndsAt
	}
	return set
}

func findWorkout(program *program_content.Program, workoutID string) (*program_content.Week, *program_content.Workout, bool) {
	for i := range program.Weeks {
		week := &program.Weeks[i]
		for k := range week.Workouts {
			if week.Workouts[k].ID == workoutID {
				return week, &week.Workouts[k], true
			}
		}
	}
	return nil, nil, false
}

func findSet(session *models.WorkoutSession, setID string) (*models.WorkoutSessionSet, bool) {
	for i := range session.Exercises {
		for k := range session.Exercises[i].Sets {
			if session.Exercises[i].Sets[k].ID == setID {
				return &session.Exercises[i].Sets[k], true
			}
		}
	}
	return nil, false
}

func firstInt(value, fallback *int) *int {
	if value != nil {
		return value
	}
	return fallback
}

func firstFloat(value, fallback *float64) *float64 {
	if value != nil {
		return value
	}
	return fallback
}

func validateSetValues(reps *int, weight *float64, rir *int, rpe *float64) error {
	if reps != nil && (*reps < 0 || *reps > 1000) {
		return fmt.Errorf("%w: invalid actual reps", ErrInvalidInput)
	}
	if weight != nil && (*weight < 0 || *weight > 10000) {
		return fmt.Errorf("%w: invalid actual weight", ErrInvalidInput)
	}
	if rir != nil && (*rir < 0 || *rir > 10) {
		return fmt.Errorf("%w: invalid actual RIR", ErrInvalidInput)
	}
	if rpe != nil && (*rpe < 0 || *rpe > 10) {
		return fmt.Errorf("%w: invalid actual RPE", ErrInvalidInput)
	}
	return nil
}

func validateUserID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid user id", ErrInvalidInput)
	}
	return nil
}

func validateID(id, field string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: %s is required", ErrInvalidInput, field)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid %s", ErrInvalidInput, field)
	}
	return nil
}
