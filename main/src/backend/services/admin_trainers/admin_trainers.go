package admin_trainers

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/admin_users"
	"ryze/backend/services/password"
	"ryze/backend/services/registration"
)

var (
	// ErrInvalidInput indicates the admin trainer-management input was
	// malformed or incomplete.
	ErrInvalidInput = errors.New("invalid admin trainer management input")
	// ErrTrainerNotFound indicates the requested trainer does not exist or is
	// not in the state required by the operation (for example, an active-only
	// operation targeting a soft-deleted trainer).
	ErrTrainerNotFound = errors.New("trainer not found")
	// ErrAlreadyActive indicates a reactivation was attempted on a trainer
	// that is already active.
	ErrAlreadyActive = errors.New("trainer is already active")
	// ErrDuplicateEmail indicates the email is already owned by an existing
	// active identity.
	ErrDuplicateEmail = errors.New("email already in use")
	// ErrTrainerAlreadyLinked indicates the user already owns an active trainer
	// profile. A user can never hold two active trainer profiles.
	ErrTrainerAlreadyLinked = errors.New("user already has an active trainer")
	// ErrUserInactive indicates the linked user account is soft-deleted, which
	// blocks operations that require the user to exist and be active. The user
	// and trainer lifecycles stay independent: trainer operations never
	// reactivate a soft-deleted user.
	ErrUserInactive = errors.New("linked user is disabled")
)

// MaxPageSize caps the number of trainers returned in a single page. Larger
// limits are clamped to this value by the service.
const MaxPageSize = admin_users.MaxPageSize

// TrainerRepository is the data-access surface required by the admin trainer
// management service.
type TrainerRepository interface {
	Create(ctx context.Context, trainer *models.Trainer) error
	FindByID(ctx context.Context, id string) (*models.Trainer, error)
	FindByIDIncludingDeleted(ctx context.Context, id string) (*models.Trainer, error)
	FindByUserID(ctx context.Context, userID string) (*models.Trainer, error)
	ListActive(ctx context.Context, page, limit int) ([]models.Trainer, int64, error)
	ListDeleted(ctx context.Context, page, limit int) ([]models.Trainer, int64, error)
	SoftDelete(ctx context.Context, id string) error
	Reactivate(ctx context.Context, id string) error
}

// UserRepository is the data-access surface required by the trainer lifecycle:
// the reactivation guard inspects the linked user and a failed trainer creation
// compensates the just-created user.
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*models.User, error)
	SoftDelete(ctx context.Context, id string) error
}

// RegistrationService is the creation dependency required by the
// create-trainer operation. Reusing the public registration flow guarantees
// that admin-created users behave exactly like the public registration
// lifecycle (email validation, Argon2id hashing and reactivation of a
// soft-deleted email).
type RegistrationService interface {
	Register(ctx context.Context, input registration.RegisterInput) (*models.User, error)
}

// UserProfileUpdater is the profile-update dependency required by the trainer
// update operation. It reuses the admin user-management service so trainer
// profile updates apply the exact same whitelist and validation as user
// updates.
type UserProfileUpdater interface {
	UpdateUser(ctx context.Context, id string, input admin_users.UpdateUserInput) (*models.User, error)
}

// TrainerResult carries one trainer together with its linked user. The user
// may be empty when the linked user account is soft-deleted (the two lifecycles
// are independent).
type TrainerResult struct {
	Trainer models.Trainer
	User    models.User
}

// ListTrainersResult carries one page of trainers plus the pagination metadata
// needed to render the list.
type ListTrainersResult struct {
	Trainers []TrainerResult
	Total    int64
	Page     int
	Limit    int
}

// CreateTrainerInput carries the admin-provided data for creating a trainer.
// A new user account is created alongside the trainer profile. The plaintext
// password is never logged, stored or returned, and the caller can never
// influence the trainer id, the user id, deleted_at or session_version.
type CreateTrainerInput struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
}

// UpdateTrainerInput carries the admin-provided fields for updating a trainer.
// Only safe user-profile fields are accepted; nil means "leave unchanged".
type UpdateTrainerInput struct {
	Email     *string
	FirstName *string
	LastName  *string
}

// AdminTrainerService manages trainer profiles on behalf of administrators.
// Authorization (which administrator may perform each operation) is enforced by
// the route middleware; this service only implements the domain rules.
type AdminTrainerService interface {
	ListTrainers(ctx context.Context, page, limit int) (ListTrainersResult, error)
	ListDeletedTrainers(ctx context.Context, page, limit int) (ListTrainersResult, error)
	GetTrainer(ctx context.Context, id string) (*TrainerResult, error)
	CreateTrainer(ctx context.Context, input CreateTrainerInput) (*TrainerResult, error)
	UpdateTrainer(ctx context.Context, id string, input UpdateTrainerInput) (*TrainerResult, error)
	SoftDeleteTrainer(ctx context.Context, id string) error
	ReactivateTrainer(ctx context.Context, id string) (*TrainerResult, error)
}

type adminTrainerService struct {
	trainerRepo TrainerRepository
	userRepo    UserRepository
	registrar   RegistrationService
	userUpdater UserProfileUpdater
}

func NewAdminTrainerService(
	trainerRepo TrainerRepository,
	userRepo UserRepository,
	registrar RegistrationService,
	userUpdater UserProfileUpdater,
) AdminTrainerService {
	return &adminTrainerService{
		trainerRepo: trainerRepo,
		userRepo:    userRepo,
		registrar:   registrar,
		userUpdater: userUpdater,
	}
}

// ListTrainers returns one page of active trainers (soft-deleted trainers are
// never listed) plus the total count of active trainers.
func (s *adminTrainerService) ListTrainers(ctx context.Context, page, limit int) (ListTrainersResult, error) {
	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return ListTrainersResult{}, err
	}

	trainers, total, err := s.trainerRepo.ListActive(ctx, page, limit)
	if err != nil {
		return ListTrainersResult{}, fmt.Errorf("failed to list trainers: %w", err)
	}
	return ListTrainersResult{Trainers: toResults(trainers), Total: total, Page: page, Limit: limit}, nil
}

// ListDeletedTrainers returns one page of soft-deleted trainers (a clearly
// separated view from the normal active-trainer listing, used for lifecycle
// management) plus the total count of soft-deleted trainers.
func (s *adminTrainerService) ListDeletedTrainers(ctx context.Context, page, limit int) (ListTrainersResult, error) {
	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return ListTrainersResult{}, err
	}

	trainers, total, err := s.trainerRepo.ListDeleted(ctx, page, limit)
	if err != nil {
		return ListTrainersResult{}, fmt.Errorf("failed to list deleted trainers: %w", err)
	}
	return ListTrainersResult{Trainers: toResults(trainers), Total: total, Page: page, Limit: limit}, nil
}

// GetTrainer returns one active trainer by its UUID. Soft-deleted trainers are
// never returned.
func (s *adminTrainerService) GetTrainer(ctx context.Context, id string) (*TrainerResult, error) {
	if err := validateTrainerID(id); err != nil {
		return nil, err
	}

	trainer, err := s.trainerRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrTrainerNotFound) {
			return nil, ErrTrainerNotFound
		}
		return nil, fmt.Errorf("failed to get trainer: %w", err)
	}
	return &TrainerResult{Trainer: *trainer, User: trainer.User}, nil
}

// CreateTrainer creates a new user account and its trainer profile. The user is
// created through the registration flow (same validation, Argon2id hashing and
// soft-deleted-email reactivation rules as public registration). If the trainer
// profile cannot be created, the just-created user is soft-deleted as a
// compensation so no active user account is ever left without a trainer
// profile: there are never partial accounts.
func (s *adminTrainerService) CreateTrainer(ctx context.Context, input CreateTrainerInput) (*TrainerResult, error) {
	user, err := s.registrar.Register(ctx, registration.RegisterInput{
		Email:     input.Email,
		Password:  input.Password,
		FirstName: input.FirstName,
		LastName:  input.LastName,
	})
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrDuplicateEmail):
			return nil, ErrDuplicateEmail
		case errors.Is(err, registration.ErrInvalidInput), errors.Is(err, password.ErrEmptyPassword):
			return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
		default:
			return nil, fmt.Errorf("failed to create trainer user: %w", err)
		}
	}

	if _, err := s.trainerRepo.FindByUserID(ctx, user.ID); err == nil {
		s.compensateUser(ctx, user.ID)
		return nil, ErrTrainerAlreadyLinked
	} else if !errors.Is(err, repositories.ErrTrainerNotFound) {
		s.compensateUser(ctx, user.ID)
		return nil, fmt.Errorf("failed to check existing trainer: %w", err)
	}

	trainer := &models.Trainer{UserID: user.ID}
	if err := s.trainerRepo.Create(ctx, trainer); err != nil {
		s.compensateUser(ctx, user.ID)
		if errors.Is(err, repositories.ErrTrainerAlreadyLinked) {
			return nil, ErrTrainerAlreadyLinked
		}
		return nil, fmt.Errorf("failed to create trainer: %w", err)
	}

	return &TrainerResult{Trainer: *trainer, User: *user}, nil
}

// compensateUser soft-deletes a user that was created for a trainer profile
// whose creation failed. The user row is never physically removed, so the
// account remains recoverable; it simply never becomes an active account
// without a trainer profile.
func (s *adminTrainerService) compensateUser(ctx context.Context, userID string) {
	_ = s.userRepo.SoftDelete(ctx, userID)
}

// UpdateTrainer updates the whitelisted safe profile fields of the user linked
// to an active trainer. The trainer profile itself has no editable fields;
// id, user_id and deleted_at can never be modified through this operation.
func (s *adminTrainerService) UpdateTrainer(ctx context.Context, id string, input UpdateTrainerInput) (*TrainerResult, error) {
	if err := validateTrainerID(id); err != nil {
		return nil, err
	}
	if input.Email == nil && input.FirstName == nil && input.LastName == nil {
		return nil, fmt.Errorf("%w: at least one field is required", ErrInvalidInput)
	}

	trainer, err := s.trainerRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrTrainerNotFound) {
			return nil, ErrTrainerNotFound
		}
		return nil, fmt.Errorf("failed to get trainer: %w", err)
	}

	user, err := s.userUpdater.UpdateUser(ctx, trainer.UserID, admin_users.UpdateUserInput{
		Email:     input.Email,
		FirstName: input.FirstName,
		LastName:  input.LastName,
	})
	if err != nil {
		switch {
		case errors.Is(err, admin_users.ErrInvalidInput):
			return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
		case errors.Is(err, admin_users.ErrUserNotFound):
			return nil, ErrUserInactive
		case errors.Is(err, admin_users.ErrDuplicateEmail):
			return nil, ErrDuplicateEmail
		default:
			return nil, fmt.Errorf("failed to update trainer: %w", err)
		}
	}

	return &TrainerResult{Trainer: *trainer, User: *user}, nil
}

// SoftDeleteTrainer soft-deletes an active trainer by its UUID. The row is
// never physically removed and the linked user account is never touched: the
// trainer lifecycle is fully independent from the user lifecycle.
func (s *adminTrainerService) SoftDeleteTrainer(ctx context.Context, id string) error {
	if err := validateTrainerID(id); err != nil {
		return err
	}

	if err := s.trainerRepo.SoftDelete(ctx, id); err != nil {
		if errors.Is(err, repositories.ErrTrainerNotFound) {
			return ErrTrainerNotFound
		}
		return fmt.Errorf("failed to soft delete trainer: %w", err)
	}
	return nil
}

// ReactivateTrainer restores a soft-deleted trainer: the same trainer UUID,
// the same user link and the same created_at are preserved, deleted_at is
// cleared and the trainer becomes active again. The reactivation always fails
// when the linked user account is soft-deleted: the user lifecycle is never
// altered by a trainer operation.
func (s *adminTrainerService) ReactivateTrainer(ctx context.Context, id string) (*TrainerResult, error) {
	if err := validateTrainerID(id); err != nil {
		return nil, err
	}

	trainer, err := s.trainerRepo.FindByIDIncludingDeleted(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrTrainerNotFound) {
			return nil, ErrTrainerNotFound
		}
		return nil, fmt.Errorf("failed to get trainer: %w", err)
	}
	if !trainer.DeletedAt.Valid {
		return nil, ErrAlreadyActive
	}

	if _, err := s.userRepo.FindByID(ctx, trainer.UserID); err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrUserInactive
		}
		return nil, fmt.Errorf("failed to check linked user: %w", err)
	}

	if err := s.trainerRepo.Reactivate(ctx, id); err != nil {
		switch {
		case errors.Is(err, repositories.ErrTrainerNotFound):
			return nil, ErrTrainerNotFound
		case errors.Is(err, repositories.ErrTrainerAlreadyLinked):
			return nil, ErrTrainerAlreadyLinked
		default:
			return nil, fmt.Errorf("failed to reactivate trainer: %w", err)
		}
	}

	active, err := s.trainerRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to reload trainer: %w", err)
	}
	return &TrainerResult{Trainer: *active, User: active.User}, nil
}

func toResults(trainers []models.Trainer) []TrainerResult {
	results := make([]TrainerResult, 0, len(trainers))
	for i := range trainers {
		results = append(results, TrainerResult{Trainer: trainers[i], User: trainers[i].User})
	}
	return results
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

// validateTrainerID rejects empty and malformed identifiers before any database
// access.
func validateTrainerID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: trainer id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid trainer id", ErrInvalidInput)
	}
	return nil
}
