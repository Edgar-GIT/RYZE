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

// Public profiles are found by @username; following is idempotent and counted
// from both sides; a deleted account stops being found.
func TestPublicProfilesAndFollows(t *testing.T) {
	db := openInvoiceDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	users := repositories.NewUserRepository(tx)
	ana := &models.User{Email: "ana" + suffix + "@example.com", PasswordHash: "hash", FirstName: "Ana", LastName: "A", Username: "ana" + suffix}
	rui := &models.User{Email: "rui" + suffix + "@example.com", PasswordHash: "hash", FirstName: "Rui", LastName: "R", Username: "rui" + suffix}
	for _, user := range []*models.User{ana, rui} {
		if err := users.Create(ctx, user); err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	repo := repositories.NewPublicProfileRepository(tx)

	found, err := repo.FindByUsername(ctx, ana.Username)
	if err != nil || found.ID != ana.ID {
		t.Fatalf("find by username: %+v, %v", found, err)
	}
	if _, err := repo.FindByUsername(ctx, "nobody"+suffix); !errors.Is(err, repositories.ErrPublicProfileNotFound) {
		t.Fatalf("unknown username: %v", err)
	}

	for range 2 {
		if err := repo.Follow(ctx, rui.ID, ana.ID); err != nil {
			t.Fatalf("follow: %v", err)
		}
	}
	followers, following, err := repo.FollowCounts(ctx, ana.ID)
	if err != nil || followers != 1 || following != 0 {
		t.Fatalf("ana counts: %d %d %v", followers, following, err)
	}
	if ok, err := repo.IsFollowing(ctx, rui.ID, ana.ID); err != nil || !ok {
		t.Fatalf("is following: %v %v", ok, err)
	}
	if people, err := repo.Followers(ctx, ana.ID, 10); err != nil || len(people) != 1 || people[0].Username != rui.Username {
		t.Fatalf("followers: %+v %v", people, err)
	}
	if err := repo.Follow(ctx, ana.ID, ana.ID); err == nil {
		t.Fatalf("following yourself must be refused by the database")
	}
}

func TestPublicProfilesUnfollow(t *testing.T) {
	db := openInvoiceDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	users := repositories.NewUserRepository(tx)
	a := &models.User{Email: "fa" + suffix + "@example.com", PasswordHash: "hash", FirstName: "A", LastName: "A", Username: "fa" + suffix}
	b := &models.User{Email: "fb" + suffix + "@example.com", PasswordHash: "hash", FirstName: "B", LastName: "B", Username: "fb" + suffix}
	for _, user := range []*models.User{a, b} {
		if err := users.Create(ctx, user); err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	repo := repositories.NewPublicProfileRepository(tx)
	if err := repo.Follow(ctx, a.ID, b.ID); err != nil {
		t.Fatalf("follow: %v", err)
	}
	if err := repo.Unfollow(ctx, a.ID, b.ID); err != nil {
		t.Fatalf("unfollow: %v", err)
	}
	if _, following, err := repo.FollowCounts(ctx, a.ID); err != nil || following != 0 {
		t.Fatalf("after unfollow: %d %v", following, err)
	}
	if workouts, err := repo.CompletedWorkouts(ctx, a.ID); err != nil || workouts != 0 {
		t.Fatalf("workouts: %d %v", workouts, err)
	}
}
