package admin_commerce

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/admin_users"
)

var (
	// ErrInvalidInput indicates the admin commerce input was malformed or
	// incomplete.
	ErrInvalidInput = errors.New("invalid admin commerce input")
	// ErrPurchaseNotFound indicates the purchase does not exist or is
	// soft-deleted.
	ErrPurchaseNotFound = errors.New("purchase not found")
)

// MaxPageSize caps the number of purchases or program sales returned in a
// single page. Larger limits are clamped to this value by the service.
const MaxPageSize = admin_users.MaxPageSize

// PurchaseRepository is the data-access surface required by the admin commerce
// service for the purchases table, including the administrative aggregates.
type PurchaseRepository interface {
	ListAdmin(ctx context.Context, filter repositories.AdminPurchaseFilter, page, limit int) ([]models.Purchase, int64, error)
	FindAdminByID(ctx context.Context, purchaseID string) (*models.Purchase, error)
	AdminPurchaseSummary(ctx context.Context, filter repositories.AdminPurchaseFilter) (repositories.AdminPurchaseSummary, error)
	ListAdminProgramSales(ctx context.Context, filter repositories.AdminPurchaseFilter, page, limit int) ([]repositories.AdminProgramSalesRow, int64, error)
}

// ProgramRepository is the data-access surface for loading program summaries,
// including soft-deleted programs so historical sales stay attributable.
type ProgramRepository interface {
	FindAllByIDsUnscoped(ctx context.Context, ids []string) ([]models.Program, error)
}

// PurchaseFilter narrows the administrator commerce queries. Every value is
// optional; Test is a tri-state (nil means both) and the created range is
// half-open [CreatedFrom, CreatedTo). ProgramIDs restricts a program sale
// lookup to a concrete set of programs and is mutually exclusive with
// ProgramID.
type PurchaseFilter struct {
	ProgramID   string
	ProgramIDs  []string
	Status      string
	Test        *bool
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

// Customer is the safe buyer identity exposed to administrators. It carries the
// display name and contact email only — never the user id, session data or
// internal fields.
type Customer struct {
	Name  string
	Email string
}

// Program is the safe program summary exposed inside admin commerce views. It
// carries public product metadata plus a deleted flag so a retired program
// keeps its historical identity without leaking deletion timestamps.
type Program struct {
	ID               string
	Name             string
	Type             string
	Status           string
	Level            *string
	DurationWeeks    *int
	FrequencyPerWeek *int
	TrainingType     *string
	PriceMinorUnits  int64
	Currency         string
	Deleted          bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Purchase is the safe representation of one purchase record in the admin
// commerce view. Commission and payout data remain internal to the backend and
// are never exposed to the administrator.
type Purchase struct {
	ID              string
	ProgramID       string
	PriceMinorUnits int64
	Currency        string
	Status          string
	Test            bool
	Access          bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Customer        Customer
	Program         Program
}

// SalesSummary aggregates the commerce dashboard metrics over the filtered
// purchases. Test purchases are counted separately and excluded from revenue.
type SalesSummary struct {
	CompletedCount        int64
	PendingCount          int64
	FailedCount           int64
	RealRevenueMinorUnits int64
	TestPurchaseCount     int64
	ProgramsWithSales     int64
}

// ProgramSale aggregates the purchase counts and revenue of a single program.
// RealSales is the completed non-test purchase count: Test Mode purchases are
// reported separately and never inflate the real sales figure.
type ProgramSale struct {
	Program           Program
	CompletedSales    int64
	RealSales         int64
	RevenueMinorUnits int64
	TestPurchases     int64
	PendingPurchases  int64
	FailedPurchases   int64
}

// ListPurchasesResult carries one page of purchases, the matching summary and
// the pagination metadata needed to render the commerce list.
type ListPurchasesResult struct {
	Purchases []Purchase
	Summary   SalesSummary
	Total     int64
	Page      int
	Limit     int
}

// ListProgramSalesResult carries one page of per-program sales plus the
// pagination metadata.
type ListProgramSalesResult struct {
	Sales []ProgramSale
	Total int64
	Page  int
	Limit int
}

// Service implements the administrative commerce view over the purchases data.
// Authorization (which administrator may read commerce data) is enforced by the
// route middleware; this service only implements the domain rules.
type Service interface {
	ListPurchases(ctx context.Context, filter PurchaseFilter, page, limit int) (ListPurchasesResult, error)
	GetPurchase(ctx context.Context, purchaseID string) (*Purchase, error)
	ListProgramSales(ctx context.Context, filter PurchaseFilter, page, limit int) (ListProgramSalesResult, error)
}

type service struct {
	purchases PurchaseRepository
	programs  ProgramRepository
}

func NewService(purchases PurchaseRepository, programs ProgramRepository) Service {
	return &service{purchases: purchases, programs: programs}
}

// ListPurchases returns one page of purchases matching the filter, most
// recently created first, plus the filtered sales summary and pagination
// metadata.
func (s *service) ListPurchases(ctx context.Context, filter PurchaseFilter, page, limit int) (ListPurchasesResult, error) {
	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return ListPurchasesResult{}, err
	}
	if err := validateFilter(filter); err != nil {
		return ListPurchasesResult{}, err
	}

	repoFilter := toRepositoryFilter(filter)
	records, total, err := s.purchases.ListAdmin(ctx, repoFilter, page, limit)
	if err != nil {
		return ListPurchasesResult{}, fmt.Errorf("failed to list purchases: %w", err)
	}
	summary, err := s.purchases.AdminPurchaseSummary(ctx, repoFilter)
	if err != nil {
		return ListPurchasesResult{}, fmt.Errorf("failed to aggregate purchases: %w", err)
	}

	purchases := make([]Purchase, 0, len(records))
	for i := range records {
		purchases = append(purchases, *newPurchase(&records[i]))
	}

	return ListPurchasesResult{
		Purchases: purchases,
		Summary: SalesSummary{
			CompletedCount:        summary.Completed,
			PendingCount:          summary.Pending,
			FailedCount:           summary.Failed,
			RealRevenueMinorUnits: summary.RealRevenueMinorUnits,
			TestPurchaseCount:     summary.TestCount,
			ProgramsWithSales:     summary.ProgramsWithSales,
		},
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

// GetPurchase returns the safe representation of one purchase by its id,
// including the buyer identity and the program summary (soft-deleted programs
// keep resolving so the record stays attributable).
func (s *service) GetPurchase(ctx context.Context, purchaseID string) (*Purchase, error) {
	if err := validatePurchaseID(purchaseID); err != nil {
		return nil, err
	}

	record, err := s.purchases.FindAdminByID(ctx, purchaseID)
	if err != nil {
		if errors.Is(err, repositories.ErrPurchaseNotFound) {
			return nil, ErrPurchaseNotFound
		}
		return nil, fmt.Errorf("failed to get purchase: %w", err)
	}
	return newPurchase(record), nil
}

// ListProgramSales returns one page of per-program sales matching the filter,
// most completed sales first, plus the total number of distinct programs with
// matching purchases. When the filter carries a program id set the response
// instead returns one sale row per requested program (in request order, absent
// ones zero-filled) so the program-management area receives a compact summary
// in a single aggregation.
func (s *service) ListProgramSales(ctx context.Context, filter PurchaseFilter, page, limit int) (ListProgramSalesResult, error) {
	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return ListProgramSalesResult{}, err
	}
	if err := validateFilter(filter); err != nil {
		return ListProgramSalesResult{}, err
	}

	requestedIDs := dedupeProgramIDs(filter.ProgramIDs)
	if len(requestedIDs) > 0 {
		filter.ProgramIDs = requestedIDs
	}

	rows, total, err := s.purchases.ListAdminProgramSales(ctx, toRepositoryFilter(filter), page, limit)
	if err != nil {
		return ListProgramSalesResult{}, fmt.Errorf("failed to aggregate program sales: %w", err)
	}

	if len(requestedIDs) > 0 {
		return s.batchProgramSales(ctx, filter, rows)
	}

	programMap := map[string]models.Program{}
	if len(rows) > 0 {
		ids := make([]string, 0, len(rows))
		for i := range rows {
			ids = append(ids, rows[i].ProgramID)
		}
		programs, err := s.programs.FindAllByIDsUnscoped(ctx, ids)
		if err != nil {
			return ListProgramSalesResult{}, fmt.Errorf("failed to load programs: %w", err)
		}
		for i := range programs {
			programMap[programs[i].ID] = programs[i]
		}
	}

	sales := make([]ProgramSale, 0, len(rows))
	for i := range rows {
		program, ok := programMap[rows[i].ProgramID]
		if !ok {
			// A program id with no matching row is defensive: the platform only
			// soft-deletes programs, so the sale keeps its attributed id.
			program = models.Program{ID: rows[i].ProgramID}
		}
		sales = append(sales, mapProgramSale(&program, &rows[i]))
	}

	return ListProgramSalesResult{
		Sales: sales,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

// batchProgramSales assembles the program sale summary for an explicit set of
// programs. Every requested id produces exactly one row in request order;
// programs without matching purchases are zero-filled so the caller never
// guesses whether a missing row means "no sales" or "request failed".
func (s *service) batchProgramSales(ctx context.Context, filter PurchaseFilter, rows []repositories.AdminProgramSalesRow) (ListProgramSalesResult, error) {
	ids := filter.ProgramIDs

	present := make(map[string]repositories.AdminProgramSalesRow, len(rows))
	for i := range rows {
		present[rows[i].ProgramID] = rows[i]
	}

	programMap := map[string]models.Program{}
	if len(ids) > 0 {
		programs, err := s.programs.FindAllByIDsUnscoped(ctx, ids)
		if err != nil {
			return ListProgramSalesResult{}, fmt.Errorf("failed to load programs: %w", err)
		}
		for i := range programs {
			programMap[programs[i].ID] = programs[i]
		}
	}

	sales := make([]ProgramSale, 0, len(ids))
	for _, id := range ids {
		program, ok := programMap[id]
		if !ok {
			// A requested program without a metadata row keeps its identity
			// so the summary stays attributable.
			program = models.Program{ID: id}
		}
		row := present[id]
		sales = append(sales, mapProgramSale(&program, &row))
	}

	return ListProgramSalesResult{
		Sales: sales,
		Total: int64(len(ids)),
		Page:  1,
		Limit: len(ids),
	}, nil
}

// mapProgramSale converts one aggregated repository row into a program sale.
func mapProgramSale(program *models.Program, row *repositories.AdminProgramSalesRow) ProgramSale {
	return ProgramSale{
		Program:           *newProgram(program),
		CompletedSales:    row.Completed,
		RealSales:         row.RealSales,
		RevenueMinorUnits: row.RevenueMinorUnits,
		TestPurchases:     row.TestCount,
		PendingPurchases:  row.Pending,
		FailedPurchases:   row.Failed,
	}
}

func newCustomer(user *models.User) Customer {
	name := strings.TrimSpace(strings.TrimSpace(user.FirstName) + " " + strings.TrimSpace(user.LastName))
	return Customer{Name: name, Email: user.Email}
}

func newProgram(model *models.Program) *Program {
	return &Program{
		ID:               model.ID,
		Name:             model.Name,
		Type:             model.Type,
		Status:           model.Status,
		Level:            model.Level,
		DurationWeeks:    model.DurationWeeks,
		FrequencyPerWeek: model.FrequencyPerWeek,
		TrainingType:     model.TrainingType,
		PriceMinorUnits:  model.PriceMinorUnits,
		Currency:         model.Currency,
		Deleted:          model.DeletedAt.Valid,
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
	}
}

func newPurchase(model *models.Purchase) *Purchase {
	return &Purchase{
		ID:              model.ID,
		ProgramID:       model.ProgramID,
		PriceMinorUnits: model.PriceMinorUnits,
		Currency:        model.Currency,
		Status:          model.Status,
		Test:            model.Test,
		Access:          purchaseHasAccess(model),
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
		Customer:        newCustomer(&model.User),
		Program:         *newProgram(&model.Program),
	}
}

// purchaseHasAccess reports whether a purchase currently grants access to its
// program. This is the same business rule as the client purchase service:
// access only exists for completed purchases of published programs that have
// not been soft-deleted.
func purchaseHasAccess(model *models.Purchase) bool {
	if model.Status != models.PurchaseStatusCompleted {
		return false
	}
	if model.Program.DeletedAt.Valid {
		return false
	}
	return model.Program.Status == models.ProgramStatusPublished
}

// toRepositoryFilter converts the service filter to the repository filter. The
// service accepts the same optional semantics; the repository is where the
// WHERE clauses are built.
func toRepositoryFilter(filter PurchaseFilter) repositories.AdminPurchaseFilter {
	return repositories.AdminPurchaseFilter{
		ProgramID:   filter.ProgramID,
		ProgramIDs:  filter.ProgramIDs,
		Status:      filter.Status,
		Test:        filter.Test,
		CreatedFrom: filter.CreatedFrom,
		CreatedTo:   filter.CreatedTo,
	}
}

// normalizePagination validates the pagination parameters and clamps oversized
// limits to MaxPageSize.
func normalizePagination(page, limit int) (int, int, error) {
	if page < 1 {
		return 0, 0, fmt.Errorf("%w: page must be at least 1", ErrInvalidInput)
	}
	if limit < 1 {
		return 0, 0, fmt.Errorf("%w: limit must be at least 1", ErrInvalidInput)
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	return page, limit, nil
}

// validateFilter rejects malformed filter values before any database access.
func validateFilter(filter PurchaseFilter) error {
	if filter.ProgramID != "" {
		if _, err := uuid.Parse(filter.ProgramID); err != nil {
			return fmt.Errorf("%w: invalid program id", ErrInvalidInput)
		}
	}
	if filter.ProgramID != "" && len(filter.ProgramIDs) > 0 {
		return fmt.Errorf("%w: program id and program id set are mutually exclusive", ErrInvalidInput)
	}
	if len(filter.ProgramIDs) > MaxPageSize {
		return fmt.Errorf("%w: program id set exceeds the maximum size", ErrInvalidInput)
	}
	for _, id := range filter.ProgramIDs {
		if _, err := uuid.Parse(id); err != nil {
			return fmt.Errorf("%w: invalid program id", ErrInvalidInput)
		}
	}
	switch filter.Status {
	case "", models.PurchaseStatusPending, models.PurchaseStatusCompleted, models.PurchaseStatusFailed:
	default:
		return fmt.Errorf("%w: invalid purchase status", ErrInvalidInput)
	}
	if filter.CreatedFrom != nil && filter.CreatedTo != nil && !filter.CreatedTo.After(*filter.CreatedFrom) {
		return fmt.Errorf("%w: invalid date range", ErrInvalidInput)
	}
	return nil
}

// dedupeProgramIDs returns the program ids in request order without
// duplicates, so a repeated id never produces repeated summary rows.
func dedupeProgramIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// validatePurchaseID rejects empty and malformed purchase identifiers.
func validatePurchaseID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: purchase id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid purchase id", ErrInvalidInput)
	}
	return nil
}
