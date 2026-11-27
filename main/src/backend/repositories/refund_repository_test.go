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

// A refund ends the access the purchase gave, including a delivered Premium
// Level 2 personal plan, cancels the request and cannot happen twice.
func TestRefundRemovesAccess(t *testing.T) {
	db := openInvoiceDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	ctx := context.Background()

	user := &models.User{Email: fmt.Sprintf("refund-%d@ryze.local", time.Now().UnixNano()), PasswordHash: "hash", FirstName: "R", LastName: "F"}
	if err := repositories.NewUserRepository(tx).Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	generic := repositories.NewGenericProgramRepository(tx)
	offer := &models.Program{Name: "Personalizado", Type: models.ProgramTypePremium, ProductType: models.ProgramProductTypePremiumLevel2,
		Status: models.ProgramStatusPublished, PriceMinorUnits: 1949, Currency: "EUR"}
	personal := &models.Program{Name: "Plano pessoal", Type: models.ProgramTypeFree, ProductType: models.ProgramProductTypeGeneric,
		Status: models.ProgramStatusDraft, Currency: "EUR"}
	for _, program := range []*models.Program{offer, personal} {
		if err := generic.CreateFull(ctx, program); err != nil {
			t.Fatalf("create program: %v", err)
		}
	}
	method := "card"
	sale := &models.Purchase{UserID: user.ID, ProgramID: offer.ID, PriceMinorUnits: 1949, Currency: "EUR", PlatformAmount: 585,
		TrainerAmount: 1364, CommissionBPS: 3000, Status: models.PurchaseStatusCompleted, PaymentMethod: &method}
	if err := tx.Create(sale).Error; err != nil {
		t.Fatalf("sale: %v", err)
	}
	for _, programID := range []string{offer.ID, personal.ID} {
		if err := tx.Create(&models.Entitlement{UserID: user.ID, ProgramID: programID}).Error; err != nil {
			t.Fatalf("entitlement: %v", err)
		}
	}
	request := &models.PremiumLevel2Request{UserID: user.ID, ProgramID: offer.ID, TrainingProgramID: &personal.ID,
		Status: models.PremiumLevel2StatusDelivered, IntakeVersion: 1}
	if err := tx.Create(request).Error; err != nil {
		t.Fatalf("request: %v", err)
	}

	refunds := repositories.NewRefundRepository(tx)
	refunded, err := refunds.Refund(ctx, sale.ID, repositories.RefundRecord{Reason: "withdrawal", AdminID: "admin-2", At: time.Now()})
	if err != nil || refunded.Status != models.PurchaseStatusRefunded {
		t.Fatalf("refund: %+v %v", refunded, err)
	}
	entitlements := repositories.NewEntitlementRepository(tx)
	for _, programID := range []string{offer.ID, personal.ID} {
		if _, err := entitlements.FindActiveByUserAndProgram(ctx, user.ID, programID); !errors.Is(err, repositories.ErrEntitlementNotFound) {
			t.Fatalf("the access to %s must end, got %v", programID, err)
		}
	}
	var status string
	tx.Raw(`SELECT status FROM premium_level_2_requests WHERE id = ?`, request.ID).Scan(&status)
	if status != models.PremiumLevel2StatusCancelled {
		t.Fatalf("the service request must be cancelled, got %q", status)
	}
	if _, err := refunds.Refund(ctx, sale.ID, repositories.RefundRecord{Reason: "withdrawal", AdminID: "admin-2", At: time.Now()}); !errors.Is(err, repositories.ErrRefundNotAllowed) {
		t.Fatalf("a purchase is refunded once, got %v", err)
	}
}
