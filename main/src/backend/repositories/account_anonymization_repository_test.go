package repositories_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

// An account deleted more than 30 days ago is anonymised: nothing identifies
// the person any more, health data and their content are erased, the
// pictures are handed back for deletion (GDPR art. 17, v2.131.0).
func TestAccountAnonymization(t *testing.T) {
	db := openInvoiceDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	ctx := context.Background()

	email := fmt.Sprintf("anon-%d@ryze.local", time.Now().UnixNano())
	user := &models.User{Email: email, PasswordHash: "hash", FirstName: "Ana", LastName: "Silva"}
	if err := repositories.NewUserRepository(tx).Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	avatar := "avatars/0123abcd-0123-4abc-8abc-0123456789ab.jpg"
	for _, statement := range []string{
		`UPDATE users SET avatar_key = '` + avatar + `', display_name = 'Aninhas', deleted_at = NOW() - INTERVAL '31 days' WHERE id = ?`,
		`INSERT INTO body_measurements (id, user_id, kind, value, measured_on) VALUES (gen_random_uuid()::text, ?, 'weight', 60, CURRENT_DATE)`,
	} {
		if err := tx.Exec(statement, user.ID).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	anonymizer := repositories.NewAccountAnonymizationRepository(tx)
	due, err := anonymizer.Due(ctx, time.Now().Add(-30*24*time.Hour), 1000)
	if err != nil {
		t.Fatalf("due: %v", err)
	}
	found := false
	for _, id := range due {
		found = found || id == user.ID
	}
	if !found {
		t.Fatalf("an account deleted 31 days ago is due")
	}
	files, err := anonymizer.Anonymize(ctx, user.ID, time.Now())
	if err != nil {
		t.Fatalf("anonymize: %v", err)
	}
	if files.Avatar == nil || *files.Avatar != avatar {
		t.Fatalf("the avatar must be handed back for deletion: %+v", files)
	}

	var after struct {
		Email        string
		FirstName    string
		LastName     string
		Username     string
		DisplayName  *string
		AvatarKey    *string
		AnonymizedAt *time.Time
	}
	if err := tx.Raw(`SELECT email, first_name, last_name, username, display_name, avatar_key, anonymized_at FROM users WHERE id = ?`, user.ID).Scan(&after).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if after.Email == email || !strings.HasSuffix(after.Email, "@anonymized.invalid") || after.FirstName != "" || after.LastName != "" ||
		after.DisplayName != nil || after.AvatarKey != nil || after.AnonymizedAt == nil || !strings.HasPrefix(after.Username, "deleted_") {
		t.Fatalf("the account still identifies the person: %+v", after)
	}
	var measurements int64
	tx.Raw(`SELECT COUNT(*) FROM body_measurements WHERE user_id = ?`, user.ID).Scan(&measurements)
	if measurements != 0 {
		t.Fatalf("health data must be erased")
	}
	if again, err := anonymizer.Anonymize(ctx, user.ID, time.Now()); err != nil || again.Avatar != nil {
		t.Fatalf("anonymising twice is a no-op: %+v %v", again, err)
	}
}
