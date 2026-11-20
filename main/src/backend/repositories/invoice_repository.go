package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ryze/backend/models"
)

// ErrInvoiceNotFound covers an unknown invoice and one that belongs to another
// account alike, so a reader can never learn that someone else's exists.
var ErrInvoiceNotFound = errors.New("invoice not found")

// InvoiceRepository stores purchase documents (migration 000048).
type InvoiceRepository interface {
	// Issue numbers and inserts an invoice for its purchase, atomically. When
	// the purchase already has one (a repeated delivery, a concurrent issue)
	// nothing is written, the existing invoice is returned and created is
	// false. Numbers are gapless per series and year.
	Issue(ctx context.Context, invoice *models.Invoice) (stored *models.Invoice, created bool, err error)
	FindByPurchaseID(ctx context.Context, purchaseID string) (*models.Invoice, error)
	// FindByPurchaseIDForUser returns the invoice only when userID owns it.
	FindByPurchaseIDForUser(ctx context.Context, purchaseID, userID string) (*models.Invoice, error)
	ListByUser(ctx context.Context, userID string) ([]models.Invoice, error)
	// CompletedWithoutInvoice lists completed purchases that have no invoice
	// yet, oldest first, with what an invoice needs. Empty userID and
	// programID mean every purchase (the background sweep).
	CompletedWithoutInvoice(ctx context.Context, userID, programID string, limit int) ([]models.InvoicePurchase, error)
	// LoadForIssue loads one purchase with its buyer and programme.
	LoadForIssue(ctx context.Context, purchaseID string) (*models.InvoicePurchase, error)
	// WithoutEmail lists invoices issued since `since` whose email was never
	// recorded in the outbox (dedupe key "invoice:<id>"), oldest first.
	WithoutEmail(ctx context.Context, since time.Time, limit int) ([]models.Invoice, error)
}

type invoiceRepository struct {
	db *gorm.DB
}

func NewInvoiceRepository(db *gorm.DB) InvoiceRepository {
	return &invoiceRepository{db: db}
}

// InvoiceEmailDedupeKey is the outbox dedupe key of an invoice's email.
func InvoiceEmailDedupeKey(invoiceID string) string {
	return "invoice:" + invoiceID
}

func (r *invoiceRepository) Issue(ctx context.Context, invoice *models.Invoice) (*models.Invoice, bool, error) {
	if existing, err := r.FindByPurchaseID(ctx, invoice.PurchaseID); err == nil {
		return existing, false, nil
	} else if !errors.Is(err, ErrInvoiceNotFound) {
		return nil, false, err
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Take the next number: the counter row is created on first use and
		// locked by the update, so concurrent issues in the same series and
		// year queue here, and a rollback gives the number back.
		var next struct{ LastNumber int }
		if err := tx.Raw(
			`INSERT INTO invoice_counters (series, year, last_number) VALUES (?, ?, 1)
			 ON CONFLICT (series, year) DO UPDATE SET last_number = invoice_counters.last_number + 1
			 RETURNING last_number`,
			invoice.Series, invoice.Year,
		).Scan(&next).Error; err != nil {
			return fmt.Errorf("failed to number invoice: %w", err)
		}
		invoice.Sequence = next.LastNumber
		invoice.Number = fmt.Sprintf("%s %d/%06d", invoice.Series, invoice.Year, next.LastNumber)
		return tx.Create(invoice).Error
	})
	if err != nil {
		if isDuplicateEntry(err) && violatedConstraint(err) == "uq_invoices_purchase" {
			// Another delivery issued it first; its number stands and ours
			// was rolled back with the counter.
			existing, findErr := r.FindByPurchaseID(ctx, invoice.PurchaseID)
			if findErr != nil {
				return nil, false, findErr
			}
			return existing, false, nil
		}
		return nil, false, fmt.Errorf("failed to issue invoice: %w", err)
	}
	return invoice, true, nil
}

func (r *invoiceRepository) FindByPurchaseID(ctx context.Context, purchaseID string) (*models.Invoice, error) {
	var invoice models.Invoice
	if err := r.db.WithContext(ctx).First(&invoice, "purchase_id = ?", purchaseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("failed to find invoice: %w", err)
	}
	return &invoice, nil
}

func (r *invoiceRepository) FindByPurchaseIDForUser(ctx context.Context, purchaseID, userID string) (*models.Invoice, error) {
	var invoice models.Invoice
	if err := r.db.WithContext(ctx).First(&invoice, "purchase_id = ? AND user_id = ?", purchaseID, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("failed to find invoice: %w", err)
	}
	return &invoice, nil
}

func (r *invoiceRepository) ListByUser(ctx context.Context, userID string) ([]models.Invoice, error) {
	var invoices []models.Invoice
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("issued_at DESC, sequence DESC").
		Find(&invoices).Error; err != nil {
		return nil, fmt.Errorf("failed to list invoices: %w", err)
	}
	return invoices, nil
}

func (r *invoiceRepository) CompletedWithoutInvoice(ctx context.Context, userID, programID string, limit int) ([]models.InvoicePurchase, error) {
	query := r.db.WithContext(ctx).
		Model(&models.Purchase{}).
		Select("purchases.id").
		Joins("LEFT JOIN invoices ON invoices.purchase_id = purchases.id").
		Where("purchases.status = ? AND invoices.id IS NULL", models.PurchaseStatusCompleted)
	if userID != "" {
		query = query.Where("purchases.user_id = ?", userID)
	}
	if programID != "" {
		query = query.Where("purchases.program_id = ?", programID)
	}
	var ids []string
	if err := query.Order("purchases.updated_at ASC, purchases.id ASC").Limit(limit).Pluck("purchases.id", &ids).Error; err != nil {
		return nil, fmt.Errorf("failed to list purchases without invoice: %w", err)
	}
	out := make([]models.InvoicePurchase, 0, len(ids))
	for _, id := range ids {
		loaded, err := r.LoadForIssue(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, *loaded)
	}
	return out, nil
}

func (r *invoiceRepository) LoadForIssue(ctx context.Context, purchaseID string) (*models.InvoicePurchase, error) {
	var loaded models.InvoicePurchase
	if err := r.db.WithContext(ctx).First(&loaded.Purchase, "id = ?", purchaseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPurchaseNotFound
		}
		return nil, fmt.Errorf("failed to load purchase: %w", err)
	}
	// The document records the sale as it was: the buyer and programme are
	// read even if they were removed since.
	if err := r.db.WithContext(ctx).Unscoped().First(&loaded.Buyer, "id = ?", loaded.Purchase.UserID).Error; err != nil {
		return nil, fmt.Errorf("failed to load invoice buyer: %w", err)
	}
	if err := r.db.WithContext(ctx).Unscoped().First(&loaded.Program, "id = ?", loaded.Purchase.ProgramID).Error; err != nil {
		return nil, fmt.Errorf("failed to load invoice programme: %w", err)
	}
	return &loaded, nil
}

func (r *invoiceRepository) WithoutEmail(ctx context.Context, since time.Time, limit int) ([]models.Invoice, error) {
	var invoices []models.Invoice
	if err := r.db.WithContext(ctx).
		Joins("LEFT JOIN email_outbox ON email_outbox.dedupe_key = 'invoice:' || invoices.id").
		Where("email_outbox.id IS NULL AND invoices.issued_at >= ?", since).
		Order(clause.OrderByColumn{Column: clause.Column{Table: "invoices", Name: "issued_at"}}).
		Limit(limit).
		Find(&invoices).Error; err != nil {
		return nil, fmt.Errorf("failed to list invoices without email: %w", err)
	}
	return invoices, nil
}
