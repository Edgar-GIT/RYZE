package repositories_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

// Allocation by offer: the request goes to the available trainer with the
// fewest open requests, one open offer at a time; a declined or expired offer
// lets the next trainer be picked; accepting allocates the request.
func TestPremiumLevel2OfferFlow(t *testing.T) {
	db := openInvoiceDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	ctx := context.Background()
	suffix := time.Now().UnixNano()

	users := repositories.NewUserRepository(tx)
	trainers := repositories.NewTrainerRepository(tx)
	generic := repositories.NewGenericProgramRepository(tx)
	requests := repositories.NewPremiumLevel2Repository(tx)
	offers := repositories.NewPremiumLevel2OfferRepository(tx)

	seedUser := func(label string) *models.User {
		user := &models.User{Email: fmt.Sprintf("pl2-offer-%s-%d@ryze.local", label, suffix), PasswordHash: "hash", FirstName: "T", LastName: label}
		if err := users.Create(ctx, user); err != nil {
			t.Fatalf("create user: %v", err)
		}
		return user
	}
	seedTrainer := func(label string) *models.Trainer {
		trainer := &models.Trainer{UserID: seedUser(label).ID}
		if err := trainers.Create(ctx, trainer); err != nil {
			t.Fatalf("create trainer: %v", err)
		}
		if err := offers.SetTrainerAvailability(ctx, trainer.ID, true); err != nil {
			t.Fatalf("availability: %v", err)
		}
		return trainer
	}
	// Other trainers in the database must not be picked by this test.
	if err := tx.Exec(`UPDATE trainers SET accepts_level2_requests = false`).Error; err != nil {
		t.Fatalf("reset availability: %v", err)
	}
	first := seedTrainer("first")
	time.Sleep(2 * time.Millisecond)
	second := seedTrainer("second")

	offer := &models.Program{Name: "Premium Level 2", Type: models.ProgramTypePremium, ProductType: models.ProgramProductTypePremiumLevel2,
		Status: models.ProgramStatusPublished, Currency: string(models.ProgramCurrencyEUR)}
	if err := generic.CreateFull(ctx, offer); err != nil {
		t.Fatalf("create offer programme: %v", err)
	}
	newRequest := func(label string) string {
		request, err := requests.CreateRequest(ctx, seedUser(label).ID, offer.ID)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
		if _, _, err := requests.CreateIntake(ctx, request.ID, json.RawMessage(`{}`)); err != nil {
			t.Fatalf("intake: %v", err)
		}
		return request.ID
	}

	requestID := newRequest("client-a")
	picked, err := offers.PickTrainer(ctx, requestID)
	if err != nil || picked != first.ID {
		t.Fatalf("first pick = %q, %v (want the older trainer)", picked, err)
	}
	opened, created, err := offers.CreateOffer(ctx, requestID, picked, time.Now().Add(time.Hour))
	if err != nil || !created {
		t.Fatalf("create offer: %v %v", created, err)
	}
	if _, again, err := offers.CreateOffer(ctx, requestID, second.ID, time.Now().Add(time.Hour)); err != nil || again {
		t.Fatalf("a second open offer must not be created: %v %v", again, err)
	}
	if _, err := offers.DeclineOffer(ctx, opened.ID, first.ID, first.UserID, time.Now()); err != nil {
		t.Fatalf("decline: %v", err)
	}
	if picked, err = offers.PickTrainer(ctx, requestID); err != nil || picked != second.ID {
		t.Fatalf("after a decline the next trainer is picked, got %q %v", picked, err)
	}
	expiring, _, err := offers.CreateOffer(ctx, requestID, second.ID, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("offer to second: %v", err)
	}
	expired, err := offers.ExpireOffers(ctx, expiring.ExpiresAt.Add(time.Second))
	if err != nil || len(expired) != 1 || expired[0] != requestID {
		t.Fatalf("expire: %v %v", expired, err)
	}
	if picked, err = offers.PickTrainer(ctx, requestID); err != nil || picked != "" {
		t.Fatalf("with every trainer asked, nobody is picked (ADMIN_2 allocates): %q %v", picked, err)
	}

	// Accepting allocates the request to the trainer.
	otherID := newRequest("client-b")
	accepted, _, err := offers.CreateOffer(ctx, otherID, first.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("offer: %v", err)
	}
	if views, err := offers.ListOpenOffers(ctx, first.ID); err != nil || len(views) != 1 || views[0].OfferName != "Premium Level 2" {
		t.Fatalf("open offers: %+v %v", views, err)
	}
	request, err := offers.AcceptOffer(ctx, accepted.ID, first.ID, first.UserID, time.Now())
	if err != nil || request.TrainerID == nil || *request.TrainerID != first.ID || request.Status != models.PremiumLevel2StatusInProgress {
		t.Fatalf("accept: %+v %v", request, err)
	}
	if _, err := offers.AcceptOffer(ctx, accepted.ID, first.ID, first.UserID, time.Now()); err == nil {
		t.Fatalf("an answered offer cannot be accepted again")
	}
}
