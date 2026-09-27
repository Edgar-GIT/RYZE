package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ryze/backend/models"
)

var (
	// ErrPurchaseNotFound indicates the purchase does not exist, is
	// soft-deleted or does not belong to the requested user.
	ErrPurchaseNotFound = errors.New("purchase not found")
	// ErrCompletedPurchaseExists indicates a completed purchase already exists
	// for the (user, program) pair, so a new test purchase cannot be created.
	// It mirrors the completed_purchase uniqueness constraint at the service
	// layer so duplicate-completion races surface as a domain error.
	ErrCompletedPurchaseExists = errors.New("completed purchase already exists")
)

// AdminPurchaseFilter narrows the administrator purchase queries. Every value
// is optional: empty values are ignored, Test is a tri-state (nil means both)
// and the created range is half-open [CreatedFrom, CreatedTo). ProgramIDs
// narrows the query to a concrete set of interface programs; it is mutually
// exclusive with ProgramID and is used for the batch sale summary consumed by
// the program-management area.
type AdminPurchaseFilter struct {
	ProgramID   string
	ProgramIDs  []string
	Status      string
	Test        *bool
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

// AdminPurchaseSummary aggregates the administrator commerce dashboard metrics
// over the purchases matching a filter. Test purchases are counted separately
// and never contribute to real revenue.
type AdminPurchaseSummary struct {
	Total                 int64
	Completed             int64
	Pending               int64
	Failed                int64
	TestCount             int64
	RealRevenueMinorUnits int64
	ProgramsWithSales     int64
}

// AdminProgramSalesRow aggregates purchase counts and revenue per program over
// the purchases matching a filter. Test purchases are counted separately and
// never contribute to revenue. RealSales is the completed non-test purchase
// count (test purchases never inflate the real sales figure).
type AdminProgramSalesRow struct {
	ProgramID         string
	Total             int64
	Completed         int64
	RealSales         int64
	Pending           int64
	Failed            int64
	TestCount         int64
	RevenueMinorUnits int64
}

// PurchaseRepository defines the data-access operations for the purchase
// entity. Every operation is scoped to an explicit user id; the repository
// never obtains it from an HTTP context.
type PurchaseRepository interface {
	Create(ctx context.Context, purchase *models.Purchase) error
	FindByID(ctx context.Context, purchaseID string) (*models.Purchase, error)
	ListActiveByUser(ctx context.Context, userID string) ([]models.Purchase, error)
	FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.Purchase, error)
	Complete(ctx context.Context, purchaseID string) error
	CompleteWithEntitlement(ctx context.Context, purchaseID string, entitlement *models.Entitlement) error
	CompleteTestPurchase(ctx context.Context, purchase *models.Purchase, entitlement *models.Entitlement) error
	// SetPaymentMethod records the immutable payment method chosen at payment
	// initiation. It is only called after the method has been validated and a
	// configured provider has been resolved for it.
	SetPaymentMethod(ctx context.Context, purchaseID, method string) error

	// ListAdmin returns one page of purchases matching the admin filter, most
	// recently created first, with the customer and the program preloaded
	// (both unscoped so soft-deleted identities and retired programs keep
	// resolving for historical accuracy), plus the total count.
	ListAdmin(ctx context.Context, filter AdminPurchaseFilter, page, limit int) ([]models.Purchase, int64, error)
	// FindAdminByID returns one purchase by its id with the customer and the
	// program preloaded (both unscoped for historical accuracy).
	FindAdminByID(ctx context.Context, purchaseID string) (*models.Purchase, error)
	// AdminPurchaseSummary aggregates the commerce dashboard metrics over the
	// purchases matching the admin filter.
	AdminPurchaseSummary(ctx context.Context, filter AdminPurchaseFilter) (AdminPurchaseSummary, error)
	// ListAdminProgramSales aggregates purchase counts and revenue per program
	// over the purchases matching the admin filter, most sales first, plus the
	// total number of distinct programs with matching purchases. When the
	// filter carries a program id set, pagination is ignored and every
	// matching requested program is returned so the caller can zero-fill the
	// absent ones.
	ListAdminProgramSales(ctx context.Context, filter AdminPurchaseFilter, page, limit int) ([]AdminProgramSalesRow, int64, error)
}

type purchaseRepository struct {
	db *gorm.DB
}

func NewPurchaseRepository(db *gorm.DB) PurchaseRepository {
	return &purchaseRepository{db: db}
}

// Create persists a new purchase record. The caller must verify that the user
// and program exist and that no duplicate active purchase already exists.
func (r *purchaseRepository) Create(ctx context.Context, purchase *models.Purchase) error {
	if err := r.db.WithContext(ctx).Create(purchase).Error; err != nil {
		return fmt.Errorf("failed to create purchase: %w", err)
	}
	return nil
}

// FindByID returns one active (non-deleted) purchase by its id without user
// scoping. An unknown or soft-deleted purchase maps to ErrPurchaseNotFound.
func (r *purchaseRepository) FindByID(ctx context.Context, purchaseID string) (*models.Purchase, error) {
	var purchase models.Purchase
	if err := r.db.WithContext(ctx).
		First(&purchase, "id = ?", purchaseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPurchaseNotFound
		}
		return nil, fmt.Errorf("failed to find purchase: %w", err)
	}
	return &purchase, nil
}

// Complete atomically transitions a pending purchase to the completed status.
// It only touches rows currently in "pending" state; the WHERE clause ensures
// exactly one row is updated. When the purchase does not exist, is
// soft-deleted or is already in a non-pending state, ErrPurchaseNotFound is
// returned because the caller cannot distinguish between "not found" and
// "already processed" at this layer — the service is responsible for the
// idempotency semantics.
func (r *purchaseRepository) Complete(ctx context.Context, purchaseID string) error {
	result := r.db.WithContext(ctx).
		Model(&models.Purchase{}).
		Where("id = ? AND status = ?", purchaseID, models.PurchaseStatusPending).
		Update("status", models.PurchaseStatusCompleted)
	if result.Error != nil {
		return fmt.Errorf("failed to complete purchase: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrPurchaseNotFound
	}
	return nil
}

// CompleteWithEntitlement atomically transitions a pending purchase to
// completed and creates the entitlement row inside a single database
// transaction. If the purchase is not pending or does not exist,
// ErrPurchaseNotFound is returned and no row is modified. Duplicate entry
// errors on the entitlement are mapped to ErrEntitlementAlreadyExists so the
// service can react with an integrity error rather than exposing raw driver
// details.
func (r *purchaseRepository) CompleteWithEntitlement(ctx context.Context, purchaseID string, entitlement *models.Entitlement) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.Purchase{}).
			Where("id = ? AND status = ?", purchaseID, models.PurchaseStatusPending).
			Update("status", models.PurchaseStatusCompleted)
		if result.Error != nil {
			return fmt.Errorf("failed to complete purchase: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrPurchaseNotFound
		}

		if err := tx.Create(entitlement).Error; err != nil {
			if isDuplicateEntry(err) {
				return ErrEntitlementAlreadyExists
			}
			return fmt.Errorf("failed to create entitlement: %w", err)
		}
		return nil
	})
}

// CompleteTestPurchase atomically creates a completed test purchase and its
// entitlement inside a single database transaction. It is the exclusive
// persistence path for Test Mode purchases: the purchase is created directly
// in the completed state with a zero price and the test marker set, and it
// never contacts a payment provider. A duplicate completed purchase or a
// duplicate entitlement both surface as domain errors so the service can treat
// them as an already-owned program.
func (r *purchaseRepository) CompleteTestPurchase(ctx context.Context, purchase *models.Purchase, entitlement *models.Entitlement) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(purchase).Error; err != nil {
			if isDuplicateEntry(err) {
				return ErrCompletedPurchaseExists
			}
			return fmt.Errorf("failed to create test purchase: %w", err)
		}

		if err := tx.Create(entitlement).Error; err != nil {
			if isDuplicateEntry(err) {
				return ErrEntitlementAlreadyExists
			}
			return fmt.Errorf("failed to create entitlement: %w", err)
		}
		return nil
	})
}

// SetPaymentMethod records the immutable payment method chosen for a purchase.
// It is called at payment initiation, only after the service has validated the
// method and resolved a configured provider for it. The update is scoped to
// the purchase id; an unknown or soft-deleted purchase maps to
// ErrPurchaseNotFound.
func (r *purchaseRepository) SetPaymentMethod(ctx context.Context, purchaseID, method string) error {
	result := r.db.WithContext(ctx).
		Model(&models.Purchase{}).
		Where("id = ?", purchaseID).
		Update("payment_method", method)
	if result.Error != nil {
		return fmt.Errorf("failed to set payment method: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrPurchaseNotFound
	}
	return nil
}

// ListActiveByUser returns every active (non-deleted) purchase belonging to
// the given user, most recently created first, with the associated program
// preloaded. The preload is unscoped so a soft-deleted program still resolves
// for historical accuracy — the purchase history must keep showing the program
// name even after the program is retired. The user id always comes from the
// caller; the repository never derives it from an HTTP context.
func (r *purchaseRepository) ListActiveByUser(ctx context.Context, userID string) ([]models.Purchase, error) {
	var purchases []models.Purchase
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Preload("Program", func(db *gorm.DB) *gorm.DB {
			return db.Unscoped()
		}).
		Order("created_at DESC, id DESC").
		Find(&purchases).Error; err != nil {
		return nil, fmt.Errorf("failed to list purchases: %w", err)
	}
	return purchases, nil
}

// FindActiveByUserAndProgram returns the most recently created active
// (non-deleted) purchase for the given user and program pair, or
// ErrPurchaseNotFound when none exists. This method is used to detect duplicate
// purchase intent before creating a new purchase.
func (r *purchaseRepository) FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.Purchase, error) {
	var purchase models.Purchase
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND program_id = ?", userID, programID).
		Order("created_at DESC, id DESC").
		First(&purchase).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPurchaseNotFound
		}
		return nil, fmt.Errorf("failed to find active purchase: %w", err)
	}
	return &purchase, nil
}

// ListAdmin returns one page of purchases matching the admin filter, most
// recently created first, plus the total number of matching purchases. The
// customer and the program are preloaded unscoped so soft-deleted accounts and
// retired programs keep resolving for historical accuracy; the caller decides
// how much of that data is exposed.
func (r *purchaseRepository) ListAdmin(ctx context.Context, filter AdminPurchaseFilter, page, limit int) ([]models.Purchase, int64, error) {
	var purchases []models.Purchase
	var total int64

	if err := scopedAdminPurchases(r.db.WithContext(ctx), filter).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count purchases: %w", err)
	}

	query := r.db.WithContext(ctx).
		Model(&models.Purchase{}).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Unscoped()
		}).
		Preload("Program", func(db *gorm.DB) *gorm.DB {
			return db.Unscoped()
		}).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Offset((page - 1) * limit)
	query = applyAdminPurchaseFilter(query, filter)
	if err := query.Find(&purchases).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list purchases: %w", err)
	}
	return purchases, total, nil
}

// FindAdminByID returns one purchase by its id with the customer and the
// program preloaded (both unscoped for historical accuracy). An unknown or
// soft-deleted purchase maps to ErrPurchaseNotFound.
func (r *purchaseRepository) FindAdminByID(ctx context.Context, purchaseID string) (*models.Purchase, error) {
	var purchase models.Purchase
	if err := r.db.WithContext(ctx).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Unscoped()
		}).
		Preload("Program", func(db *gorm.DB) *gorm.DB {
			return db.Unscoped()
		}).
		First(&purchase, "id = ?", purchaseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPurchaseNotFound
		}
		return nil, fmt.Errorf("failed to find purchase: %w", err)
	}
	return &purchase, nil
}

// AdminPurchaseSummary aggregates the commerce dashboard metrics over the
// purchases matching the admin filter. Test purchases are counted and reported
// separately and excluded from real revenue. ProgramsWithSales is the number of
// distinct programs with at least one completed non-test purchase.
func (r *purchaseRepository) AdminPurchaseSummary(ctx context.Context, filter AdminPurchaseFilter) (AdminPurchaseSummary, error) {
	var summary AdminPurchaseSummary
	db := scopedAdminPurchases(r.db.WithContext(ctx), filter)
	if err := db.Select(`
		COUNT(*) AS total,
		COALESCE(SUM(status = '` + models.PurchaseStatusCompleted + `'), 0) AS completed,
		COALESCE(SUM(status = '` + models.PurchaseStatusPending + `'), 0) AS pending,
		COALESCE(SUM(status = '` + models.PurchaseStatusFailed + `'), 0) AS failed,
		COALESCE(SUM(test), 0) AS test_count,
		COALESCE(SUM(CASE WHEN status = '` + models.PurchaseStatusCompleted + `' AND test = 0 THEN price_minor_units END), 0) AS real_revenue_minor_units
	`).Scan(&summary).Error; err != nil {
		return AdminPurchaseSummary{}, fmt.Errorf("failed to aggregate purchase summary: %w", err)
	}

	var programsWithSales int64
	if err := scopedAdminPurchases(r.db.WithContext(ctx), filter).
		Where("status = ?", models.PurchaseStatusCompleted).
		Where("test = ?", false).
		Distinct("program_id").
		Count(&programsWithSales).Error; err != nil {
		return AdminPurchaseSummary{}, fmt.Errorf("failed to count programs with sales: %w", err)
	}
	summary.ProgramsWithSales = programsWithSales
	return summary, nil
}

// ListAdminProgramSales aggregates purchase counts and revenue per program over
// the purchases matching the admin filter, most sales first, plus the total
// number of distinct programs with matching purchases. When the filter carries
// a program id set, pagination is ignored and every matching requested program
// is returned so the caller can zero-fill the absent ones.
func (r *purchaseRepository) ListAdminProgramSales(ctx context.Context, filter AdminPurchaseFilter, page, limit int) ([]AdminProgramSalesRow, int64, error) {
	var rows []AdminProgramSalesRow
	var total int64

	// The set mode is never paginated: the caller asks for a bounded slice of
	// concrete programs and needs every aggregated row in one round trip.
	if len(filter.ProgramIDs) == 0 {
		sub := scopedAdminPurchases(r.db.WithContext(ctx), filter).
			Select("program_id", "COUNT(*) AS total").
			Group("program_id")
		if err := r.db.WithContext(ctx).
			Table("(?) AS program_sales", sub).
			Count(&total).Error; err != nil {
			return nil, 0, fmt.Errorf("failed to count program sales: %w", err)
		}
	}

	db := scopedAdminPurchases(r.db.WithContext(ctx), filter).
		Select(
			"program_id",
			"COUNT(*) AS total",
			"COALESCE(SUM(status = '"+models.PurchaseStatusCompleted+"'), 0) AS completed",
			"COALESCE(SUM(CASE WHEN status = '"+models.PurchaseStatusCompleted+"' AND test = 0 THEN 1 ELSE 0 END), 0) AS real_sales",
			"COALESCE(SUM(status = '"+models.PurchaseStatusPending+"'), 0) AS pending",
			"COALESCE(SUM(status = '"+models.PurchaseStatusFailed+"'), 0) AS failed",
			"COALESCE(SUM(test), 0) AS test_count",
			"COALESCE(SUM(CASE WHEN status = '"+models.PurchaseStatusCompleted+"' AND test = 0 THEN price_minor_units END), 0) AS revenue_minor_units",
		).
		Group("program_id").
		Order("completed DESC, revenue_minor_units DESC, program_id ASC")
	if len(filter.ProgramIDs) == 0 {
		db = db.Limit(limit).Offset((page - 1) * limit)
	}

	if err := db.Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to aggregate program sales: %w", err)
	}
	return rows, total, nil
}

// scopedAdminPurchases returns a query over the purchases table with the admin
// filter conditions applied. Test is a tri-state: nil means both test and
// non-test purchases. The model is set explicitly so count and aggregate
// queries never depend on the caller having issued a previous statement on the
// same connection.
func scopedAdminPurchases(db *gorm.DB, filter AdminPurchaseFilter) *gorm.DB {
	return applyAdminPurchaseFilter(db.Model(&models.Purchase{}), filter)
}

// applyAdminPurchaseFilter applies the admin filter WHERE clauses to a query.
func applyAdminPurchaseFilter(db *gorm.DB, filter AdminPurchaseFilter) *gorm.DB {
	for _, clause := range adminPurchaseConditions(filter) {
		db = db.Where(clause.Query, clause.Args...)
	}
	return db
}

type adminWhereClause struct {
	Query string
	Args  []any
}

// adminPurchaseConditions converts an admin filter into GORM WHERE clauses.
func adminPurchaseConditions(filter AdminPurchaseFilter) []adminWhereClause {
	var clauses []adminWhereClause
	if filter.ProgramID != "" {
		clauses = append(clauses, adminWhereClause{Query: "program_id = ?", Args: []any{filter.ProgramID}})
	} else if len(filter.ProgramIDs) > 0 {
		clauses = append(clauses, adminWhereClause{Query: "program_id IN ?", Args: []any{filter.ProgramIDs}})
	}
	if filter.Status != "" {
		clauses = append(clauses, adminWhereClause{Query: "status = ?", Args: []any{filter.Status}})
	}
	if filter.Test != nil {
		clauses = append(clauses, adminWhereClause{Query: "test = ?", Args: []any{*filter.Test}})
	}
	if filter.CreatedFrom != nil {
		clauses = append(clauses, adminWhereClause{Query: "created_at >= ?", Args: []any{filter.CreatedFrom}})
	}
	if filter.CreatedTo != nil {
		clauses = append(clauses, adminWhereClause{Query: "created_at < ?", Args: []any{filter.CreatedTo}})
	}
	return clauses
}
