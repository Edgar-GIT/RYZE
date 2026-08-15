package workout_history

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/admin_users"
)

var (
	// ErrInvalidInput indicates the input was malformed or incomplete.
	ErrInvalidInput = errors.New("invalid workout history input")
	// ErrWorkoutNotFound indicates the user has no active program assignment or
	// the requested workout does not belong to the assigned program's active
	// structure. Both cases are deliberately indistinguishable to the client.
	ErrWorkoutNotFound = errors.New("workout not found in the assigned program")
)

const (
	// MaxPageSize caps the number of history entries returned in a single page.
	MaxPageSize = admin_users.MaxPageSize
)

// WorkoutHistoryRepository is the data-access surface required by the workout
// history service. Every operation is scoped to the authenticated user id, which
// always comes from the authentication context and is never accepted from the
// client.
type WorkoutHistoryRepository interface {
	Create(ctx context.Context, userID, programWorkoutID string, entry *models.WorkoutHistory) error
	ListByUser(ctx context.Context, userID string, page, limit int) ([]models.WorkoutHistory, int64, error)
}

// HistoryEntry is the safe representation of one completed workout. It carries
// only the completed workout reference and the lifecycle timestamps; the owning
// user id and deletion markers are never exposed.
type HistoryEntry struct {
	ID               string
	ProgramWorkoutID string
	CompletedAt      time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ListHistoryResult carries one page of history entries plus the pagination
// metadata needed to render the list.
type ListHistoryResult struct {
	Entries []HistoryEntry
	Total   int64
	Page    int
	Limit   int
}

// Service implements the client-facing workout history flow. The requesting user
// identity always comes from the authentication context and is never accepted
// from the client. This service never knows about HTTP, Gin or the
// authentication context.
type Service interface {
	// CompleteWorkout records that the authenticated user completed one workout
	// of their currently assigned program. The workout is only executable when
	// it belongs to the active structure of the assigned program; any break in
	// that chain maps to a single indistinguishable ErrWorkoutNotFound.
	CompleteWorkout(ctx context.Context, userID, workoutID string) (*HistoryEntry, error)
	// ListHistory returns one page of the authenticated user's own completed
	// workouts, newest first.
	ListHistory(ctx context.Context, userID string, page, limit int) (ListHistoryResult, error)
}

type service struct {
	history WorkoutHistoryRepository
}

func NewService(history WorkoutHistoryRepository) Service {
	return &service{history: history}
}

func (s *service) CompleteWorkout(ctx context.Context, userID, workoutID string) (*HistoryEntry, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if err := validateWorkoutID(workoutID); err != nil {
		return nil, err
	}

	entry := &models.WorkoutHistory{}
	if err := s.history.Create(ctx, userID, workoutID, entry); err != nil {
		switch {
		case errors.Is(err, repositories.ErrWorkoutNotFound):
			return nil, ErrWorkoutNotFound
		default:
			return nil, fmt.Errorf("failed to record the completed workout: %w", err)
		}
	}
	return newEntry(entry), nil
}

func (s *service) ListHistory(ctx context.Context, userID string, page, limit int) (ListHistoryResult, error) {
	if err := validateUserID(userID); err != nil {
		return ListHistoryResult{}, err
	}
	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return ListHistoryResult{}, err
	}

	entries, total, err := s.history.ListByUser(ctx, userID, page, limit)
	if err != nil {
		return ListHistoryResult{}, fmt.Errorf("failed to list workout history entries: %w", err)
	}

	return ListHistoryResult{
		Entries: toSafeList(entries),
		Total:   total,
		Page:    page,
		Limit:   limit,
	}, nil
}

func newEntry(model *models.WorkoutHistory) *HistoryEntry {
	return &HistoryEntry{
		ID:               model.ID,
		ProgramWorkoutID: model.ProgramWorkoutID,
		CompletedAt:      model.CompletedAt,
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
	}
}

func toSafeList(models []models.WorkoutHistory) []HistoryEntry {
	list := make([]HistoryEntry, 0, len(models))
	for i := range models {
		list = append(list, *newEntry(&models[i]))
	}
	return list
}

// normalizePagination validates the pagination parameters and clamps oversized
// limits to MaxPageSize.
func normalizePagination(page, limit int) (int, int, error) {
	if page < 1 {
		return 0, 0, fmt.Errorf("%w: page must be at least 1", ErrInvalidInput)
	}
	if limit < 1 {
		return 0, 0, fmt.Errorf("%w: limit must be at least 1", ErrInvalidInput)
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	return page, limit, nil
}

func validateUserID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid user id", ErrInvalidInput)
	}
	return nil
}

func validateWorkoutID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: workout id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid workout id", ErrInvalidInput)
	}
	return nil
}
