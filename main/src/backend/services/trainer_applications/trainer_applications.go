package trainer_applications

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/admin_users"
)

var (
	// ErrInvalidInput indicates the trainer-application input was malformed or
	// incomplete.
	ErrInvalidInput = errors.New("invalid trainer application input")
	// ErrUserNotFound indicates the requesting user does not exist or is not in
	// a state that allows the operation (for example, a soft-deleted user can
	// never apply or be approved).
	ErrUserNotFound = errors.New("user not found")
	// ErrAlreadyTrainer indicates the user already owns an active trainer
	// profile, so no application can be created for them and no application can
	// be approved for them: a user can never hold two active trainer profiles.
	ErrAlreadyTrainer = errors.New("user is already a trainer")
	// ErrApplicationNotFound indicates the requested application does not exist
	// or is not in the state required by the operation.
	ErrApplicationNotFound = errors.New("trainer application not found")
	// ErrApplicationStateConflict indicates the application exists but is not
	// in the state required by the operation (for example, reviewing an
	// application that is no longer PENDING).
	ErrApplicationStateConflict = errors.New("trainer application is not in the required state")
	// ErrApplicationAlreadyActive indicates the user already owns an active
	// (PENDING or APPROVED) application.
	ErrApplicationAlreadyActive = errors.New("user already has an active trainer application")
)

// MaxPageSize caps the number of applications returned in a single page.
// Larger limits are clamped to this value by the service.
const MaxPageSize = admin_users.MaxPageSize

// ApplicationRepository is the data-access surface required by the
// trainer-application service.
type ApplicationRepository interface {
	Create(ctx context.Context, application *models.TrainerApplication) error
	FindActiveByUserID(ctx context.Context, userID string) (*models.TrainerApplication, error)
	FindByID(ctx context.Context, id string) (*models.TrainerApplication, error)
	List(ctx context.Context, page, limit int, status string) ([]models.TrainerApplication, int64, error)
	Approve(ctx context.Context, applicationID string) (*models.TrainerApplication, error)
	Reject(ctx context.Context, applicationID string) error
}

// UserRepository is the data-access surface required for verifying that the
// applying user exists and is active.
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*models.User, error)
}

// TrainerRepository is the data-access surface required for verifying that the
// user does not already own an active trainer profile.
type TrainerRepository interface {
	FindByUserID(ctx context.Context, userID string) (*models.Trainer, error)
}

// ListApplicationsResult carries one page of trainer applications plus the
// pagination metadata needed to render the list.
type ListApplicationsResult struct {
	Applications []models.TrainerApplication
	Total        int64
	Page         int
	Limit        int
}

// Service implements the trainer-application flow: users apply to become
// trainers and administrators review those applications. Authorization (which
// administrator may review applications) is enforced by the route middleware;
// this service only implements the domain rules.
type Service interface {
	Apply(ctx context.Context, userID string) (*models.TrainerApplication, error)
	ListApplications(ctx context.Context, page, limit int, status string) (ListApplicationsResult, error)
	GetApplication(ctx context.Context, id string) (*models.TrainerApplication, error)
	ApproveApplication(ctx context.Context, id string) (*models.TrainerApplication, error)
	RejectApplication(ctx context.Context, id string) error
}

type service struct {
	applications ApplicationRepository
	users        UserRepository
	trainers     TrainerRepository
}

func NewService(applications ApplicationRepository, users UserRepository, trainers TrainerRepository) Service {
	return &service{applications: applications, users: users, trainers: trainers}
}

// Apply creates a PENDING application for the authenticated user. The user ID
// is always taken from the authenticated session; the caller can never
// influence it. The user must exist, be active and not already own a trainer
// profile, and must not already hold an active application.
func (s *service) Apply(ctx context.Context, userID string) (*models.TrainerApplication, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	if _, err := s.users.FindByID(ctx, userID); err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to check user: %w", err)
	}

	if _, err := s.trainers.FindByUserID(ctx, userID); err == nil {
		return nil, ErrAlreadyTrainer
	} else if !errors.Is(err, repositories.ErrTrainerNotFound) {
		return nil, fmt.Errorf("failed to check trainer profile: %w", err)
	}

	application := &models.TrainerApplication{
		UserID: userID,
		Status: models.ApplicationStatusPending,
	}
	if err := s.applications.Create(ctx, application); err != nil {
		if errors.Is(err, repositories.ErrApplicationAlreadyActive) {
			return nil, ErrApplicationAlreadyActive
		}
		return nil, fmt.Errorf("failed to create trainer application: %w", err)
	}

	created, err := s.applications.FindByID(ctx, application.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload trainer application: %w", err)
	}
	return created, nil
}

// ListApplications returns one page of trainer applications, optionally
// filtered by status and ordered by creation time, plus the total count of
// matching applications.
func (s *service) ListApplications(ctx context.Context, page, limit int, status string) (ListApplicationsResult, error) {
	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return ListApplicationsResult{}, err
	}
	if err := validateStatusFilter(status); err != nil {
		return ListApplicationsResult{}, err
	}

	applications, total, err := s.applications.List(ctx, page, limit, status)
	if err != nil {
		return ListApplicationsResult{}, fmt.Errorf("failed to list trainer applications: %w", err)
	}
	return ListApplicationsResult{Applications: applications, Total: total, Page: page, Limit: limit}, nil
}

// GetApplication returns one trainer application by its UUID. Soft-deleted
// applications are never returned.
func (s *service) GetApplication(ctx context.Context, id string) (*models.TrainerApplication, error) {
	if err := validateApplicationID(id); err != nil {
		return nil, err
	}

	application, err := s.applications.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrApplicationNotFound) {
			return nil, ErrApplicationNotFound
		}
		return nil, fmt.Errorf("failed to get trainer application: %w", err)
	}
	return application, nil
}

// ApproveApplication approves a PENDING application and creates the trainer
// profile for the applicant. The operation is atomic at the repository level:
// the application becomes APPROVED and the trainer profile is created in the
// same database transaction, so a failure can never leave a half-completed
// approval. The applicant must still exist and be active.
func (s *service) ApproveApplication(ctx context.Context, id string) (*models.TrainerApplication, error) {
	if err := validateApplicationID(id); err != nil {
		return nil, err
	}

	application, err := s.applications.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrApplicationNotFound) {
			return nil, ErrApplicationNotFound
		}
		return nil, fmt.Errorf("failed to get trainer application: %w", err)
	}
	if application.Status != models.ApplicationStatusPending {
		return nil, ErrApplicationStateConflict
	}

	if _, err := s.users.FindByID(ctx, application.UserID); err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to check user: %w", err)
	}

	approved, err := s.applications.Approve(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrApplicationNotFound):
			return nil, ErrApplicationNotFound
		case errors.Is(err, repositories.ErrApplicationStateConflict):
			return nil, ErrApplicationStateConflict
		case errors.Is(err, repositories.ErrTrainerAlreadyLinked):
			return nil, ErrAlreadyTrainer
		default:
			return nil, fmt.Errorf("failed to approve trainer application: %w", err)
		}
	}

	reloaded, err := s.applications.FindByID(ctx, approved.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload approved trainer application: %w", err)
	}
	return reloaded, nil
}

// RejectApplication rejects a PENDING application. The application stays in
// history so the user can apply again. Applications that do not exist or are
// not PENDING are rejected with the corresponding domain error.
func (s *service) RejectApplication(ctx context.Context, id string) error {
	if err := validateApplicationID(id); err != nil {
		return err
	}

	if err := s.applications.Reject(ctx, id); err != nil {
		switch {
		case errors.Is(err, repositories.ErrApplicationNotFound):
			return ErrApplicationNotFound
		case errors.Is(err, repositories.ErrApplicationStateConflict):
			return ErrApplicationStateConflict
		default:
			return fmt.Errorf("failed to reject trainer application: %w", err)
		}
	}
	return nil
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

// validateStatusFilter accepts an empty filter or one of the three application
// statuses.
func validateStatusFilter(status string) error {
	switch status {
	case "", models.ApplicationStatusPending, models.ApplicationStatusApproved, models.ApplicationStatusRejected:
		return nil
	default:
		return fmt.Errorf("%w: invalid status filter", ErrInvalidInput)
	}
}

// validateApplicationID rejects empty and malformed identifiers before any
// database access.
func validateApplicationID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: application id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid application id", ErrInvalidInput)
	}
	return nil
}

// validateUserID rejects empty and malformed identifiers before any database
// access.
func validateUserID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid user id", ErrInvalidInput)
	}
	return nil
}
