package repositories_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

// The database generates a unique @username from the email when none is
// chosen; a chosen one taken by another active account is refused.
func TestUsernameGenerationAndUniqueness(t *testing.T) {
	db := openInvoiceDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	users := repositories.NewUserRepository(tx)
	profiles := repositories.NewProfileRepository(tx)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)

	first := &models.User{Email: "handle" + suffix + "@example.com", PasswordHash: "hash", FirstName: "A", LastName: "B"}
	if err := users.Create(ctx, first); err != nil {
		t.Fatalf("create: %v", err)
	}
	if first.Username != "handle"+suffix {
		t.Fatalf("generated username = %q", first.Username)
	}
	second := &models.User{Email: "handle" + suffix + "@other.example", PasswordHash: "hash", FirstName: "C", LastName: "D"}
	if err := users.Create(ctx, second); err != nil {
		t.Fatalf("create second: %v", err)
	}
	if second.Username != "handle"+suffix+".2" {
		t.Fatalf("second generated username = %q", second.Username)
	}
	if err := profiles.UpdateProfile(ctx, second.ID, nil, first.Username, false); !errors.Is(err, repositories.ErrUsernameTaken) {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}
	name := "Edgar"
	if err := profiles.UpdateProfile(ctx, second.ID, &name, "edgar"+suffix, true); err != nil {
		t.Fatalf("update: %v", err)
	}
	found, err := profiles.FindByUsername(ctx, "edgar"+suffix)
	if err != nil || found.ID != second.ID || found.DisplayName == nil || *found.DisplayName != "Edgar" || found.PublicName() != "Edgar" {
		t.Fatalf("find by username: %+v (%v)", found, err)
	}
	chosen := &models.User{Email: "x" + suffix + "@example.com", PasswordHash: "hash", FirstName: "E", LastName: "F", Username: "edgar" + suffix}
	if err := users.Create(ctx, chosen); !errors.Is(err, repositories.ErrUsernameTaken) {
		t.Fatalf("a taken chosen username must be refused, got %v", err)
	}
}
