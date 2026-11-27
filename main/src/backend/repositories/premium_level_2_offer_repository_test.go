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

// Allocation by offer (owner decision, 2026-11-24): the request is offered
// at once to every available trainer; the first to accept takes it, the other
// offers are withdrawn and a late accept fails. RYZE itself can take a
// waiting request instead.
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
	// Other trainers in the database must not be offered this test's requests.
	if err := tx.Exec(`UPDATE trainers SET accepts_level2_requests = false`).Error; err != nil {
		t.Fatalf("reset availability: %v", err)
	}
	first := seedTrainer("first")
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
	offerOf := func(created []models.PremiumLevel2Offer, trainerID string) string {
		for _, offer := range created {
			if offer.TrainerID == trainerID {
				return offer.ID
			}
		}
		t.Fatalf("trainer %s was not offered the request: %+v", trainerID, created)
		return ""
	}

	requestID := newRequest("client-a")
	created, err := offers.OfferToAvailableTrainers(ctx, requestID, time.Now().Add(time.Hour))
	if err != nil || len(created) != 2 {
		t.Fatalf("every available trainer is offered the request: %+v %v", created, err)
	}
	if again, err := offers.OfferToAvailableTrainers(ctx, requestID, time.Now().Add(time.Hour)); err != nil || len(again) != 0 {
		t.Fatalf("nobody is offered the same request twice: %+v %v", again, err)
	}
	if views, err := offers.ListOpenOffers(ctx, second.ID); err != nil || len(views) != 1 || views[0].OfferName != "Premium Level 2" {
		t.Fatalf("open offers: %+v %v", views, err)
	}
	request, err := offers.AcceptOffer(ctx, offerOf(created, second.ID), second.ID, second.UserID, time.Now())
	if err != nil || request.TrainerID == nil || *request.TrainerID != second.ID || request.Status != models.PremiumLevel2StatusInProgress {
		t.Fatalf("first accept takes the request: %+v %v", request, err)
	}
	if _, err := offers.AcceptOffer(ctx, offerOf(created, first.ID), first.ID, first.UserID, time.Now()); err == nil {
		t.Fatalf("a late accept must fail once the request is taken")
	}
	if views, err := offers.ListOpenOffers(ctx, first.ID); err != nil || len(views) != 0 {
		t.Fatalf("the other offers are withdrawn: %+v %v", views, err)
	}

	// A declined offer leaves the others open; RYZE can take the request.
	otherID := newRequest("client-b")
	created, err = offers.OfferToAvailableTrainers(ctx, otherID, time.Now().Add(time.Hour))
	if err != nil || len(created) != 2 {
		t.Fatalf("offer: %+v %v", created, err)
	}
	if _, err := offers.DeclineOffer(ctx, offerOf(created, first.ID), first.ID, first.UserID, time.Now()); err != nil {
		t.Fatalf("decline: %v", err)
	}
	if views, err := offers.ListOpenOffers(ctx, second.ID); err != nil || len(views) != 1 {
		t.Fatalf("a decline leaves the other offers open: %+v %v", views, err)
	}
	taken, err := requests.TakeAsRyze(ctx, otherID, "admin-1")
	if err != nil || !taken.HandledByRyze || taken.TrainerID != nil || taken.Status != models.PremiumLevel2StatusInProgress {
		t.Fatalf("RYZE takes the request: %+v %v", taken, err)
	}
	if views, err := offers.ListOpenOffers(ctx, second.ID); err != nil || len(views) != 0 {
		t.Fatalf("taking the request withdraws the offers: %+v %v", views, err)
	}
	if _, err := requests.TakeAsRyze(ctx, otherID, "admin-1"); err == nil {
		t.Fatalf("a taken request cannot be taken again")
	}

	// A trainer who stops accepting requests loses their open offers.
	thirdID := newRequest("client-c")
	if _, err := offers.OfferToAvailableTrainers(ctx, thirdID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("offer: %v", err)
	}
	if err := offers.SetTrainerAvailability(ctx, first.ID, false); err != nil {
		t.Fatalf("availability off: %v", err)
	}
	if views, err := offers.ListOpenOffers(ctx, first.ID); err != nil || len(views) != 0 {
		t.Fatalf("availability off withdraws the offers: %+v %v", views, err)
	}
}
