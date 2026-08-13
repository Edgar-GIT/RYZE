package trainer_clients

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
	// ErrInvalidInput indicates the trainer-client input was malformed or
	// incomplete.
	ErrInvalidInput = errors.New("invalid trainer client input")
	// ErrClientNotFound indicates the client user does not exist or is not in a
	// state that allows the operation (for example, a soft-deleted user can
	// never be added or re-added as a client).
	ErrClientNotFound = errors.New("client not found")
	// ErrClientAlreadyActive indicates an active relationship already exists
	// between the trainer and the user.
	ErrClientAlreadyActive = errors.New("client already added")
	// ErrClientRelationNotFound indicates the relationship between the trainer
	// and the user does not exist or is not in the state required by the
	// operation.
	ErrClientRelationNotFound = errors.New("trainer-client relationship not found")
)

// MaxPageSize caps the number of clients returned in a single page. Larger
// limits are clamped to this value by the service.
const MaxPageSize = admin_users.MaxPageSize

// ClientRepository is the data-access surface required by the trainer-clients
// service. Every operation is scoped to an explicit trainer id; the repository
// never obtains it from an HTTP context.
type ClientRepository interface {
	Create(ctx context.Context, relation *models.TrainerClient) error
	FindActiveByTrainerAndUser(ctx context.Context, trainerID, userID string) (*models.TrainerClient, error)
	FindIncludingDeletedByTrainerAndUser(ctx context.Context, trainerID, userID string) (*models.TrainerClient, error)
	ListActiveClients(ctx context.Context, trainerID string, page, limit int) ([]models.TrainerClient, int64, error)
	SoftDelete(ctx context.Context, trainerID, userID string) error
	Reactivate(ctx context.Context, trainerID, userID string) error
}

// UserRepository is the data-access surface required for verifying that the
// client exists and is active.
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*models.User, error)
}

// Client is the safe representation of one trainer→client relationship. It
// only carries public user information and never exposes authentication data
// (password hash, session version), tokens or deletion markers.
type Client struct {
	RelationID        string
	TrainerID         string
	UserID            string
	Email             string
	FirstName         string
	LastName          string
	RelationCreatedAt time.Time
	RelationUpdatedAt time.Time
	UserCreatedAt     time.Time
	UserUpdatedAt     time.Time
}

// ListClientsResult carries one page of client relationships plus the
// pagination metadata needed to render the list.
type ListClientsResult struct {
	Clients []Client
	Total   int64
	Page    int
	Limit   int
}

// Service implements the trainer→client relationship flow. Authorization
// (which trainer may operate) is enforced by the route middleware; ownership
// is guaranteed because the trainer id always comes from the authenticated
// trainer context and is never accepted from the client. This service never
// knows about HTTP, Gin or the trainer context.
type Service interface {
	ListClients(ctx context.Context, trainerID string, page, limit int) (ListClientsResult, error)
	GetClient(ctx context.Context, trainerID, userID string) (*Client, error)
	AddClient(ctx context.Context, trainerID, userID string) (*Client, error)
	RemoveClient(ctx context.Context, trainerID, userID string) error
	ReactivateClient(ctx context.Context, trainerID, userID string) (*Client, error)
}

type service struct {
	clients ClientRepository
	users   UserRepository
}

func NewService(clients ClientRepository, users UserRepository) Service {
	return &service{clients: clients, users: users}
}

// AddClient creates the active relationship between the trainer and an
// existing active user. The trainer id is always provided by the caller from
// the authenticated trainer context. The client must exist and be active, must
// not be the trainer itself and must not already hold an active relationship
// with this trainer. The database enforces the one-active-relation rule
// through its unique index.
func (s *service) AddClient(ctx context.Context, trainerID, userID string) (*Client, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if trainerID == userID {
		return nil, fmt.Errorf("%w: a trainer cannot be their own client", ErrInvalidInput)
	}

	if _, err := s.users.FindByID(ctx, userID); err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrClientNotFound
		}
		return nil, fmt.Errorf("failed to check client user: %w", err)
	}

	relation := &models.TrainerClient{
		TrainerID: trainerID,
		UserID:    userID,
	}
	if err := s.clients.Create(ctx, relation); err != nil {
		if errors.Is(err, repositories.ErrClientRelationAlreadyActive) {
			return nil, ErrClientAlreadyActive
		}
		return nil, fmt.Errorf("failed to create trainer-client relationship: %w", err)
	}

	return s.getClient(ctx, trainerID, userID)
}

// ListClients returns one page of the authenticated trainer's active clients,
// ordered by creation time, plus the total count. The trainer id is always
// explicit; a client-supplied trainer_id can never change which clients are
// listed.
func (s *service) ListClients(ctx context.Context, trainerID string, page, limit int) (ListClientsResult, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return ListClientsResult{}, err
	}

	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return ListClientsResult{}, err
	}

	relations, total, err := s.clients.ListActiveClients(ctx, trainerID, page, limit)
	if err != nil {
		return ListClientsResult{}, fmt.Errorf("failed to list trainer clients: %w", err)
	}

	clients := make([]Client, 0, len(relations))
	for i := range relations {
		clients = append(clients, newClient(&relations[i]))
	}
	return ListClientsResult{Clients: clients, Total: total, Page: page, Limit: limit}, nil
}

// GetClient returns the safe profile of one of the trainer's active clients.
// The active relationship and the linked user are verified together in a single
// query scoped by both identifiers, so a user that does not exist, is
// soft-deleted, is not a client of this trainer or whose relationship was
// soft-deleted is indistinguishable and never revealed. Only the trainer who
// owns the active relationship can ever receive the linked user's public data.
func (s *service) GetClient(ctx context.Context, trainerID, userID string) (*Client, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	relation, err := s.clients.FindActiveByTrainerAndUser(ctx, trainerID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrClientRelationNotFound) {
			return nil, ErrClientRelationNotFound
		}
		return nil, fmt.Errorf("failed to load trainer-client relationship: %w", err)
	}
	// A missing or soft-deleted linked user must be hidden behind the same
	// not-found error: the profile read never reveals whether the requested
	// user exists.
	if relation.User.ID == "" || relation.User.ID != userID {
		return nil, ErrClientRelationNotFound
	}

	client := newClient(relation)
	return &client, nil
}

// RemoveClient soft-deletes the relationship between the trainer and the user.
// Only the relationship row is touched: the user account and the trainer
// profile are never deleted and remain active.
func (s *service) RemoveClient(ctx context.Context, trainerID, userID string) error {
	if err := validateTrainerID(trainerID); err != nil {
		return err
	}
	if err := validateUserID(userID); err != nil {
		return err
	}

	if err := s.clients.SoftDelete(ctx, trainerID, userID); err != nil {
		switch {
		case errors.Is(err, repositories.ErrClientRelationNotFound):
			return ErrClientRelationNotFound
		default:
			return fmt.Errorf("failed to remove trainer-client relationship: %w", err)
		}
	}
	return nil
}

// ReactivateClient restores a previously removed relationship, reusing the
// exact same row. The client user must still exist and be active, and an
// already-active relationship can never be reactivated.
func (s *service) ReactivateClient(ctx context.Context, trainerID, userID string) (*Client, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	if _, err := s.users.FindByID(ctx, userID); err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrClientNotFound
		}
		return nil, fmt.Errorf("failed to check client user: %w", err)
	}

	if err := s.clients.Reactivate(ctx, trainerID, userID); err != nil {
		switch {
		case errors.Is(err, repositories.ErrClientRelationNotFound):
			return nil, ErrClientRelationNotFound
		case errors.Is(err, repositories.ErrClientRelationAlreadyActive):
			return nil, ErrClientAlreadyActive
		default:
			return nil, fmt.Errorf("failed to reactivate trainer-client relationship: %w", err)
		}
	}

	return s.getClient(ctx, trainerID, userID)
}

// getClient loads the active relationship for the pair and maps it to its safe
// representation. The linked user is always present because the relationship
// is only created against active users; a missing linked user is an impossible
// inconsistency and is treated as an internal error without exposing details.
func (s *service) getClient(ctx context.Context, trainerID, userID string) (*Client, error) {
	relation, err := s.clients.FindActiveByTrainerAndUser(ctx, trainerID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrClientRelationNotFound) {
			return nil, ErrClientRelationNotFound
		}
		return nil, fmt.Errorf("failed to reload trainer-client relationship: %w", err)
	}
	if relation.User.ID == "" || relation.User.ID != userID {
		return nil, fmt.Errorf("trainer-client relationship inconsistent: linked user is missing")
	}
	client := newClient(relation)
	return &client, nil
}

func newClient(relation *models.TrainerClient) Client {
	return Client{
		RelationID:        relation.ID,
		TrainerID:         relation.TrainerID,
		UserID:            relation.UserID,
		Email:             relation.User.Email,
		FirstName:         relation.User.FirstName,
		LastName:          relation.User.LastName,
		RelationCreatedAt: relation.CreatedAt,
		RelationUpdatedAt: relation.UpdatedAt,
		UserCreatedAt:     relation.User.CreatedAt,
		UserUpdatedAt:     relation.User.UpdatedAt,
	}
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
