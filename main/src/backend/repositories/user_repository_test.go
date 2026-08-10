package repositories_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

func TestUserRepository(t *testing.T) {
	config.LoadEnvFile()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}

	tx := db.Begin()
	defer tx.Rollback()

	repo := repositories.NewUserRepository(tx)
	ctx := context.Background()
	email := fmt.Sprintf("repo-test-%d@ryze.local", time.Now().UnixNano())
	updatedEmail := fmt.Sprintf("repo-test-updated-%d@ryze.local", time.Now().UnixNano())

	// 1. Create user.
	user := &models.User{
		Email:        email,
		PasswordHash: "prepared-hash-outside-repository-scope",
		FirstName:    "John",
		LastName:     "Doe",
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.ID == "" {
		t.Fatal("create user: expected generated UUID id")
	}

	// 2. Duplicate email is rejected while the user is active.
	duplicate := &models.User{
		Email:     email,
		FirstName: "Duplicate",
		LastName:  "Entry",
	}
	if err := repo.Create(ctx, duplicate); !errors.Is(err, repositories.ErrDuplicateEmail) {
		t.Fatalf("duplicate email: expected ErrDuplicateEmail, got %v", err)
	}

	// 3. Find the user by ID.
	byID, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("find user by id: %v", err)
	}
	if byID.Email != email {
		t.Fatalf("find by id: expected email %q, got %q", email, byID.Email)
	}

	// 4. Find the user by email.
	byEmail, err := repo.FindByEmail(ctx, email)
	if err != nil {
		t.Fatalf("find user by email: %v", err)
	}
	if byEmail.ID != user.ID {
		t.Fatalf("find by email: expected id %q, got %q", user.ID, byEmail.ID)
	}

	// 5. Update basic information.
	user.FirstName = "Jane"
	user.LastName = "Smith"
	user.Email = updatedEmail
	if err := repo.Update(ctx, user); err != nil {
		t.Fatalf("update user: %v", err)
	}

	updated, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("find updated user by id: %v", err)
	}
	if updated.FirstName != "Jane" || updated.LastName != "Smith" || updated.Email != updatedEmail {
		t.Fatalf("update: got %s %s <%s>", updated.FirstName, updated.LastName, updated.Email)
	}

	// 6. Soft delete the user.
	if err := repo.SoftDelete(ctx, user.ID); err != nil {
		t.Fatalf("soft delete user: %v", err)
	}

	// 7. Deleted user is no longer returned by normal lookups.
	if _, err := repo.FindByID(ctx, user.ID); !errors.Is(err, repositories.ErrUserNotFound) {
		t.Fatalf("find by id after delete: expected ErrUserNotFound, got %v", err)
	}
	if _, err := repo.FindByEmail(ctx, updatedEmail); !errors.Is(err, repositories.ErrUserNotFound) {
		t.Fatalf("find by email after delete: expected ErrUserNotFound, got %v", err)
	}

	// 8. Row still exists with deleted_at populated.
	var raw models.User
	if err := tx.Unscoped().Where("id = ?", user.ID).First(&raw).Error; err != nil {
		t.Fatalf("row should still exist after soft delete: %v", err)
	}
	if !raw.DeletedAt.Valid {
		t.Fatal("expected deleted_at to be populated on the soft-deleted row")
	}
}
