package trainercontext

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ErrNoAuthenticatedTrainer is returned by TrainerIDFromContext and
// UserIDFromContext when the context does not carry a valid authenticated
// trainer identity.
var ErrNoAuthenticatedTrainer = errors.New("no authenticated trainer in context")

// trainerContextKey is an unexported type so other packages cannot forge
// the context key used to carry trainer authentication data.
type trainerContextKey struct{}

// TrainerContextKey is the Gin context key holding the authenticated trainer
// identity.
var TrainerContextKey = trainerContextKey{}

// Identity contains the authenticated trainer's user and trainer UUIDs.
//
// UserID identifies the underlying RYZE user account.
// TrainerID identifies the active trainer profile belonging to that user.
type Identity struct {
	UserID    string
	TrainerID string
}

// SetIdentity stores the authenticated trainer identity in the Gin context.
//
// This helper is intended for the TrainerAuthenticate middleware. Keeping the
// context write in this package ensures the context key remains centralized
// and prevents literal keys from being scattered through the application.
func SetIdentity(c *gin.Context, identity Identity) {
	c.Set(TrainerContextKey, identity)
}

// IdentityFromContext returns the authenticated trainer identity stored by
// TrainerAuthenticate.
//
// Both UUIDs must be valid and non-empty. Invalid, missing, or incorrectly
// typed context values are rejected safely.
func IdentityFromContext(c *gin.Context) (Identity, error) {
	value, exists := c.Get(TrainerContextKey)
	if !exists || value == nil {
		return Identity{}, ErrNoAuthenticatedTrainer
	}

	identity, ok := value.(Identity)
	if !ok || identity.UserID == "" || identity.TrainerID == "" {
		return Identity{}, ErrNoAuthenticatedTrainer
	}

	if _, err := uuid.Parse(identity.UserID); err != nil {
		return Identity{}, ErrNoAuthenticatedTrainer
	}

	if _, err := uuid.Parse(identity.TrainerID); err != nil {
		return Identity{}, ErrNoAuthenticatedTrainer
	}

	return identity, nil
}

// UserIDFromContext returns the authenticated user UUID associated with the
// trainer session.
func UserIDFromContext(c *gin.Context) (string, error) {
	identity, err := IdentityFromContext(c)
	if err != nil {
		return "", err
	}
	return identity.UserID, nil
}

// TrainerIDFromContext returns the authenticated trainer UUID associated with
// the trainer session.
func TrainerIDFromContext(c *gin.Context) (string, error) {
	identity, err := IdentityFromContext(c)
	if err != nil {
		return "", err
	}
	return identity.TrainerID, nil
}
