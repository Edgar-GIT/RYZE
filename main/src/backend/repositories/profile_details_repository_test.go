package repositories_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

// The details are absent until saved, then read back as saved; saving again
// replaces them.
func TestProfileDetailsSaveAndGet(t *testing.T) {
	db := openInvoiceDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	ctx := context.Background()
	user := &models.User{Email: fmt.Sprintf("details%d@example.com", time.Now().UnixNano()), PasswordHash: "hash", FirstName: "A", LastName: "B"}
	if err := repositories.NewUserRepository(tx).Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	repo := repositories.NewProfileDetailsRepository(tx)
	empty, err := repo.Get(ctx, user.ID)
	if err != nil || empty.Bio != nil || len(empty.SocialLinks) != 0 {
		t.Fatalf("empty details: %+v %v", empty, err)
	}
	bio, since, weight := "Powerlifter", 2019, 81.5
	birth := time.Date(1995, 3, 2, 0, 0, 0, 0, time.UTC)
	if err := repo.Save(ctx, user.ID, repositories.ProfileDetailsRow{Bio: &bio, BirthDate: &birth, TrainingSince: &since, WeightKg: &weight,
		SocialLinks: map[string]string{"instagram": "rita"}}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := repo.Save(ctx, user.ID, repositories.ProfileDetailsRow{Bio: &bio, WeightKg: &weight, SocialLinks: map[string]string{"x": "rita"}}); err != nil {
		t.Fatalf("save again: %v", err)
	}
	got, err := repo.Get(ctx, user.ID)
	if err != nil || got.Bio == nil || *got.Bio != bio || got.BirthDate != nil || got.WeightKg == nil || *got.WeightKg != 81.5 ||
		got.SocialLinks["x"] != "rita" || got.SocialLinks["instagram"] != "" {
		t.Fatalf("replaced details: %+v %v", got, err)
	}
}
