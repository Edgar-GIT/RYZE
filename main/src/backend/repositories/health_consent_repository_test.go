package repositories_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

// Withdrawing the consent for health data erases the body measurements
// (GDPR art. 9(2)(a), v2.131.0).
func TestHealthConsentGiveAndWithdraw(t *testing.T) {
	db := openInvoiceDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	ctx := context.Background()

	user := &models.User{Email: fmt.Sprintf("health-consent-%d@ryze.local", time.Now().UnixNano()), PasswordHash: "hash", FirstName: "H", LastName: "C"}
	if err := repositories.NewUserRepository(tx).Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	consents := repositories.NewHealthConsentRepository(tx)
	if at, err := consents.ConsentedAt(ctx, user.ID); err != nil || at != nil {
		t.Fatalf("no consent at first: %v %v", at, err)
	}
	first := time.Now().UTC().Truncate(time.Second)
	if err := consents.Give(ctx, user.ID, first); err != nil {
		t.Fatalf("give: %v", err)
	}
	if err := consents.Give(ctx, user.ID, first.Add(time.Hour)); err != nil {
		t.Fatalf("give again: %v", err)
	}
	if at, err := consents.ConsentedAt(ctx, user.ID); err != nil || at == nil || !at.Equal(first) {
		t.Fatalf("the first consent stays: %v %v", at, err)
	}
	if err := tx.Exec(`INSERT INTO body_measurements (id, user_id, kind, value, measured_on) VALUES (gen_random_uuid()::text, ?, 'weight', 80, CURRENT_DATE)`, user.ID).Error; err != nil {
		t.Fatalf("measurement: %v", err)
	}
	if err := consents.Withdraw(ctx, user.ID); err != nil {
		t.Fatalf("withdraw: %v", err)
	}
	var left int64
	tx.Raw(`SELECT COUNT(*) FROM body_measurements WHERE user_id = ?`, user.ID).Scan(&left)
	if left != 0 {
		t.Fatalf("withdrawing must erase the measurements, %d left", left)
	}
	if at, err := consents.ConsentedAt(ctx, user.ID); err != nil || at != nil {
		t.Fatalf("withdrawn consent: %v %v", at, err)
	}
}
