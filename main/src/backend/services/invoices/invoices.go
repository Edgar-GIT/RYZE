// Package invoices issues the purchase document of every completed purchase
// (owner requirement, 2026-11-20): one per purchase, automatically, shown to
// the client and emailed without any client action. Real sales and Test Mode
// purchases are numbered in separate series; free activations are not
// purchases and get none.
//
// Until certified invoicing software is connected the document is a purchase
// receipt with no tax value; VAT follows the small-business exemption
// (article 53 of the Portuguese VAT Code). The document is a snapshot of the
// sale and is immutable once issued.
//
// Issuing never fails a purchase: it runs as a purchase fulfilment hook after
// the entitlement is committed, a failure is logged and retried by the next
// delivery of the purchase, by the client opening the document and by the
// background sweep.
package invoices

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"ryze/backend/config"
	"ryze/backend/email"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

// ErrNotFound covers an unknown purchase, another account's purchase and a
// purchase that is not completed alike.
var ErrNotFound = errors.New("invoice not found")

const (
	fulfilBatch = 10
	sweepBatch  = 50
	// emailRetryWindow bounds how far back the sweep re-sends a document whose
	// email was never recorded.
	emailRetryWindow = 7 * 24 * time.Hour
)

// Recipients resolves the mailbox of a Test Mode persona (the current
// session's validated recipient, or "" to suppress the email).
type Recipients interface {
	FindActiveRecipientByPersona(ctx context.Context, personaUserID string) (string, error)
}

// Service is the invoices contract.
type Service interface {
	// FulfillPurchase issues (and emails) the documents a completed purchase
	// of the program owes the user. It is idempotent.
	FulfillPurchase(ctx context.Context, userID, programID string) error
	// Sweep issues documents still missing for completed purchases and
	// re-records emails that were never recorded.
	Sweep(ctx context.Context) error
	// ForPurchase returns the owner's document for a purchase, issuing it on
	// demand when the purchase is completed and the document is missing.
	ForPurchase(ctx context.Context, userID, purchaseID string) (*models.Invoice, error)
	// List is the owner's documents, newest first.
	List(ctx context.Context, userID string) ([]models.Invoice, error)
	// ForPurchaseAdmin returns any purchase's document (administrators with
	// the commerce permission only; enforced by the route).
	ForPurchaseAdmin(ctx context.Context, purchaseID string) (*models.Invoice, error)
}

type service struct {
	repo       repositories.InvoiceRepository
	mailer     email.Mailer
	recipients Recipients
	seller     config.InvoiceConfig
	now        func() time.Time
}

// New builds the service. mailer may be nil (documents are still issued and
// shown in the account).
func New(repo repositories.InvoiceRepository, mailer email.Mailer, recipients Recipients, seller config.InvoiceConfig) Service {
	if strings.TrimSpace(seller.SellerName) == "" {
		seller.SellerName = "RYZE"
	}
	return &service{repo: repo, mailer: mailer, recipients: recipients, seller: seller, now: func() time.Time { return time.Now().UTC() }}
}

func (s *service) FulfillPurchase(ctx context.Context, userID, programID string) error {
	pending, err := s.repo.CompletedWithoutInvoice(ctx, userID, programID, fulfilBatch)
	if err != nil {
		return fmt.Errorf("invoice: %w", err)
	}
	var failures []error
	for index := range pending {
		if _, err := s.issue(ctx, &pending[index]); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (s *service) Sweep(ctx context.Context) error {
	pending, err := s.repo.CompletedWithoutInvoice(ctx, "", "", sweepBatch)
	if err != nil {
		return fmt.Errorf("invoice sweep: %w", err)
	}
	for index := range pending {
		if _, err := s.issue(ctx, &pending[index]); err != nil {
			slog.Error("invoice not issued", slog.String("purchase_id", pending[index].Purchase.ID), slog.String("error", err.Error()))
		}
	}
	if s.mailer == nil {
		return nil
	}
	unsent, err := s.repo.WithoutEmail(ctx, s.now().Add(-emailRetryWindow), sweepBatch)
	if err != nil {
		return fmt.Errorf("invoice email sweep: %w", err)
	}
	for index := range unsent {
		invoice := &unsent[index]
		// A Test Mode document is emailed only when it is issued, inside the
		// session whose recipient was just validated: a later sweep could
		// reach a different session's mailbox.
		if invoice.Test {
			continue
		}
		loaded, err := s.repo.LoadForIssue(ctx, invoice.PurchaseID)
		if err != nil {
			slog.Error("invoice email not retried", slog.String("invoice_id", invoice.ID), slog.String("error", err.Error()))
			continue
		}
		s.sendEmail(ctx, invoice, &loaded.Buyer)
	}
	return nil
}

func (s *service) ForPurchase(ctx context.Context, userID, purchaseID string) (*models.Invoice, error) {
	if !validID(userID) || !validID(purchaseID) {
		return nil, ErrNotFound
	}
	invoice, err := s.repo.FindByPurchaseIDForUser(ctx, purchaseID, userID)
	if err == nil {
		return invoice, nil
	}
	if !errors.Is(err, repositories.ErrInvoiceNotFound) {
		return nil, err
	}
	// Missing: issue it now if this is the owner's completed purchase (an
	// earlier issue failed and no retry has run yet).
	loaded, err := s.repo.LoadForIssue(ctx, purchaseID)
	if err != nil {
		if errors.Is(err, repositories.ErrPurchaseNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if loaded.Purchase.UserID != userID || loaded.Purchase.Status != models.PurchaseStatusCompleted {
		return nil, ErrNotFound
	}
	return s.issue(ctx, loaded)
}

func (s *service) List(ctx context.Context, userID string) ([]models.Invoice, error) {
	if !validID(userID) {
		return nil, ErrNotFound
	}
	return s.repo.ListByUser(ctx, userID)
}

func (s *service) ForPurchaseAdmin(ctx context.Context, purchaseID string) (*models.Invoice, error) {
	if !validID(purchaseID) {
		return nil, ErrNotFound
	}
	invoice, err := s.repo.FindByPurchaseID(ctx, purchaseID)
	if errors.Is(err, repositories.ErrInvoiceNotFound) {
		return nil, ErrNotFound
	}
	return invoice, err
}

// issue numbers and stores the document of one completed purchase (or returns
// the one already issued) and records its email.
func (s *service) issue(ctx context.Context, loaded *models.InvoicePurchase) (*models.Invoice, error) {
	purchase := &loaded.Purchase
	if purchase.Status != models.PurchaseStatusCompleted {
		return nil, ErrNotFound
	}
	now := s.now()
	series := models.InvoiceSeriesSales
	if purchase.Test {
		series = models.InvoiceSeriesTest
	}
	exemption := models.InvoiceTaxExemptionSmallBusiness
	purchasedAt := purchase.UpdatedAt
	if purchasedAt.IsZero() {
		purchasedAt = purchase.CreatedAt
	}
	invoice := &models.Invoice{
		Series:              series,
		Year:                now.Year(),
		PurchaseID:          purchase.ID,
		UserID:              purchase.UserID,
		ProgramID:           purchase.ProgramID,
		Test:                purchase.Test,
		Status:              models.InvoiceStatusIssued,
		IssuedAt:            now,
		PurchasedAt:         purchasedAt.UTC(),
		SellerName:          bounded(s.seller.SellerName, 160),
		SellerTaxID:         optional(s.seller.SellerTaxID, 32),
		SellerAddress:       optional(s.seller.SellerAddress, 255),
		BuyerName:           bounded(buyerName(&loaded.Buyer), 255),
		BuyerTaxID:          purchase.BuyerTaxID,
		Description:         bounded(loaded.Program.Name, 255),
		ProductType:         bounded(loaded.Program.ProductType, 32),
		Quantity:            1,
		UnitPriceMinorUnits: purchase.PriceMinorUnits,
		TaxExemptionCode:    &exemption,
		TotalMinorUnits:     purchase.PriceMinorUnits,
		Currency:            purchase.Currency,
		PaymentMethod:       purchase.PaymentMethod,
	}
	if invoice.Description == "" {
		invoice.Description = "RYZE"
	}
	stored, created, err := s.repo.Issue(ctx, invoice)
	if err != nil {
		return nil, err
	}
	if created {
		slog.Info("invoice issued",
			slog.String("invoice_id", stored.ID),
			slog.String("number", stored.Number),
			slog.String("purchase_id", stored.PurchaseID),
		)
		s.sendEmail(ctx, stored, &loaded.Buyer)
	}
	return stored, nil
}

// sendEmail records the document's email in the outbox (once per document,
// by its dedupe key). A Test Mode persona's goes to the current session's
// validated recipient, never to the persona's synthetic address; without one
// it is suppressed. Failures are logged; the sweep records it later.
func (s *service) sendEmail(ctx context.Context, invoice *models.Invoice, buyer *models.User) {
	if s.mailer == nil || buyer == nil || buyer.DeletedAt.Valid {
		return
	}
	recipient := buyer.Email
	if buyer.TestPersona != nil {
		if s.recipients == nil {
			return
		}
		resolved, err := s.recipients.FindActiveRecipientByPersona(ctx, buyer.ID)
		if err != nil {
			slog.Error("invoice email recipient not resolved", slog.String("invoice_id", invoice.ID), slog.String("error", err.Error()))
			return
		}
		recipient = resolved
	}
	if recipient == "" {
		return
	}
	locale := email.LocaleEnglish
	if buyer.PreferredLanguage != nil {
		locale = *buyer.PreferredLanguage
	}
	data := &email.InvoiceData{
		Number:      invoice.Number,
		IssuedAt:    invoice.IssuedAt.UTC().Format("2006-01-02"),
		Test:        invoice.Test,
		SellerName:  invoice.SellerName,
		BuyerName:   invoice.BuyerName,
		Description: invoice.Description,
		Quantity:    invoice.Quantity,
		UnitPrice:   invoice.UnitPriceMinorUnits,
		Tax:         invoice.TaxMinorUnits,
		Total:       invoice.TotalMinorUnits,
		Currency:    invoice.Currency,
	}
	if invoice.SellerTaxID != nil {
		data.SellerTaxID = *invoice.SellerTaxID
	}
	if invoice.SellerAddress != nil {
		data.SellerAddress = *invoice.SellerAddress
	}
	if invoice.BuyerTaxID != nil {
		data.BuyerTaxID = *invoice.BuyerTaxID
	}
	if invoice.TaxExemptionCode != nil {
		data.TaxExemption = *invoice.TaxExemptionCode
	}
	if err := s.mailer.Enqueue(ctx, email.Request{
		UserID:    buyer.ID,
		To:        recipient,
		Template:  email.TemplateInvoiceIssued,
		Locale:    locale,
		Name:      email.GreetingName(buyer),
		Program:   invoice.Description,
		Path:      ReceiptPath(invoice.PurchaseID),
		DedupeKey: repositories.InvoiceEmailDedupeKey(invoice.ID),
		Invoice:   data,
	}); err != nil {
		slog.Error("invoice email not recorded", slog.String("invoice_id", invoice.ID), slog.String("error", err.Error()))
	}
}

// ReceiptPath is the document's page in the web app.
func ReceiptPath(purchaseID string) string {
	return "/account/purchases/" + purchaseID + "/receipt"
}

// buyerName is the account holder's name at issue time, or their account
// address when no name was given.
func buyerName(user *models.User) string {
	name := strings.TrimSpace(strings.TrimSpace(user.FirstName) + " " + strings.TrimSpace(user.LastName))
	if name == "" {
		return user.Email
	}
	return name
}

func bounded(value string, limit int) string {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) > limit {
		value = string([]rune(value)[:limit])
	}
	return value
}

func optional(value string, limit int) *string {
	value = bounded(value, limit)
	if value == "" {
		return nil
	}
	return &value
}

func validID(id string) bool {
	return id != "" && len(id) <= 36 && !strings.ContainsAny(id, " \t\r\n")
}
