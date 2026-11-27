package repositories_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

// A member with a plan who stopped training is reminded once a week at
// most, and never when they turned reminders off (v2.131.0).
func TestTrainingReminderCandidates(t *testing.T) {
	db := openInvoiceDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	ctx := context.Background()

	user := &models.User{Email: fmt.Sprintf("remind-%d@ryze.local", time.Now().UnixNano()), PasswordHash: "hash", FirstName: "R", LastName: "M"}
	if err := repositories.NewUserRepository(tx).Create(ctx, user); err != nil {
		t.Fatalf("user: %v", err)
	}
	plan := &models.Program{Name: "Plano", Type: models.ProgramTypeFree, ProductType: models.ProgramProductTypeGeneric,
		Status: models.ProgramStatusPublished, Currency: "EUR"}
	if err := repositories.NewGenericProgramRepository(tx).CreateFull(ctx, plan); err != nil {
		t.Fatalf("plan: %v", err)
	}
	for _, statement := range []string{
		`UPDATE users SET email_verified_at = NOW(), created_at = NOW() - INTERVAL '10 days' WHERE id = ?`,
		`INSERT INTO free_program_activations (id, user_id, program_id) VALUES (gen_random_uuid()::text, ?, '` + plan.ID + `')`,
	} {
		if err := tx.Exec(statement, user.ID).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	reminders := repositories.NewTrainingReminderRepository(tx)
	now := time.Now()
	isDue := func() bool {
		due, err := reminders.Due(ctx, now.Add(-4*24*time.Hour), now.Add(-7*24*time.Hour), 100000)
		if err != nil {
			t.Fatalf("due: %v", err)
		}
		for _, candidate := range due {
			if candidate.UserID == user.ID {
				return true
			}
		}
		return false
	}
	if !isDue() {
		t.Fatalf("a member with a plan and no training for 10 days is due")
	}
	if err := reminders.MarkReminded(ctx, user.ID, now); err != nil {
		t.Fatalf("mark: %v", err)
	}
	if isDue() {
		t.Fatalf("a member reminded today is not due again this week")
	}
	tx.Exec(`UPDATE users SET last_training_reminder_at = NULL WHERE id = ?`, user.ID)
	if err := reminders.SetEnabled(ctx, user.ID, false); err != nil {
		t.Fatalf("off: %v", err)
	}
	if isDue() {
		t.Fatalf("a member who turned reminders off is never due")
	}
}
