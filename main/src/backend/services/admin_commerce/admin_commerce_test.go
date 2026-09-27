package admin_commerce_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/admin_commerce"
)

// gormDeletedAt returns a valid (soft-delete) GORM deletion marker.
func gormDeletedAt() gorm.DeletedAt {
	return gorm.DeletedAt{Time: time.Now(), Valid: true}
}

// stubAdminPurchaseRepository is a scripted fake for the admin purchase
// repository surface.
type stubAdminPurchaseRepository struct {
	lists      []models.Purchase
	total      int64
	listErr    error
	found      *models.Purchase
	findErr    error
	summary    repositories.AdminPurchaseSummary
	summaryErr error
	rows       []repositories.AdminProgramSalesRow
	gotFilter  repositories.AdminPurchaseFilter
	gotPage    int
	gotLimit   int
}

func (s *stubAdminPurchaseRepository) ListAdmin(_ context.Context, filter repositories.AdminPurchaseFilter, page, limit int) ([]models.Purchase, int64, error) {
	s.gotFilter = filter
	s.gotPage = page
	s.gotLimit = limit
	return s.lists, s.total, s.listErr
}

func (s *stubAdminPurchaseRepository) FindAdminByID(_ context.Context, _ string) (*models.Purchase, error) {
	return s.found, s.findErr
}

func (s *stubAdminPurchaseRepository) AdminPurchaseSummary(_ context.Context, filter repositories.AdminPurchaseFilter) (repositories.AdminPurchaseSummary, error) {
	s.gotFilter = filter
	return s.summary, s.summaryErr
}

func (s *stubAdminPurchaseRepository) ListAdminProgramSales(_ context.Context, filter repositories.AdminPurchaseFilter, page, limit int) ([]repositories.AdminProgramSalesRow, int64, error) {
	s.gotFilter = filter
	s.gotPage = page
	s.gotLimit = limit
	return s.rows, s.total, s.listErr
}

// stubAdminProgramRepository is a scripted fake for the program repository
// surface used by the admin commerce service.
type stubAdminProgramRepository struct {
	programs []models.Program
	err      error
}

func (s *stubAdminProgramRepository) FindAllByIDsUnscoped(_ context.Context, ids []string) ([]models.Program, error) {
	return s.programs, s.err
}

func newTestService(purchases *stubAdminPurchaseRepository, programs *stubAdminProgramRepository) admin_commerce.Service {
	return admin_commerce.NewService(purchases, programs)
}

func completedPublishedPurchase() models.Purchase {
	return models.Purchase{
		ID:              "00000000-0000-0000-0000-000000000001",
		UserID:          "11111111-1111-1111-1111-111111111111",
		ProgramID:       "22222222-2222-2222-2222-222222222222",
		PriceMinorUnits: 4500,
		Currency:        "EUR",
		Status:          models.PurchaseStatusCompleted,
		User: models.User{
			ID:        "11111111-1111-1111-1111-111111111111",
			Email:     "buyer@example.com",
			FirstName: "Ana",
			LastName:  "Sousa",
		},
		Program: models.Program{
			ID:              "22222222-2222-2222-2222-222222222222",
			Name:            "Muscle Builder",
			Type:            models.ProgramTypePremium,
			Status:          models.ProgramStatusPublished,
			PriceMinorUnits: 4500,
			Currency:        "EUR",
		},
	}
}

func TestListPurchasesSuccess(t *testing.T) {
	purchase := completedPublishedPurchase()
	purchase.Test = false
	stub := &stubAdminPurchaseRepository{
		lists: []models.Purchase{purchase},
		total: 7,
		summary: repositories.AdminPurchaseSummary{
			Total:                 7,
			Completed:             5,
			Pending:               1,
			Failed:                1,
			TestCount:             2,
			RealRevenueMinorUnits: 22500,
			ProgramsWithSales:     3,
		},
	}
	svc := newTestService(stub, &stubAdminProgramRepository{})
	filter := admin_commerce.PurchaseFilter{Status: models.PurchaseStatusCompleted}

	result, err := svc.ListPurchases(context.Background(), filter, 1, 10)
	if err != nil {
		t.Fatalf("ListPurchases: %v", err)
	}
	if result.Page != 1 || result.Limit != 10 || result.Total != 7 {
		t.Fatalf("pagination metadata mismatch: %+v", result)
	}
	if len(result.Purchases) != 1 {
		t.Fatalf("expected 1 purchase, got %d", len(result.Purchases))
	}
	got := result.Purchases[0]
	if got.Access != true {
		t.Fatalf("completed purchase of published program must grant access")
	}
	if got.Customer.Name != "Ana Sousa" || got.Customer.Email != "buyer@example.com" {
		t.Fatalf("customer not mapped: %+v", got.Customer)
	}
	if got.Program.Name != "Muscle Builder" || got.Program.Deleted {
		t.Fatalf("program not mapped: %+v", got.Program)
	}
	if result.Summary.RealRevenueMinorUnits != 22500 || result.Summary.CompletedCount != 5 || result.Summary.ProgramsWithSales != 3 {
		t.Fatalf("summary not forwarded: %+v", result.Summary)
	}
	if stub.gotFilter.Status != models.PurchaseStatusCompleted {
		t.Fatalf("filter not forwarded: %+v", stub.gotFilter)
	}
}

func TestListPurchasesAccessReflectsProgramState(t *testing.T) {
	base := completedPublishedPurchase()

	draft := base
	draft.Status = models.PurchaseStatusCompleted
	draft.Program.Status = models.ProgramStatusDraft

	deleted := base
	deleted.Program.DeletedAt = gormDeletedAt()

	pending := base
	pending.Status = models.PurchaseStatusPending

	stub := &stubAdminPurchaseRepository{lists: []models.Purchase{pending, draft, deleted}}
	svc := newTestService(stub, &stubAdminProgramRepository{})

	result, err := svc.ListPurchases(context.Background(), admin_commerce.PurchaseFilter{}, 1, 20)
	if err != nil {
		t.Fatalf("ListPurchases: %v", err)
	}
	for i, want := range []bool{false, false, false} {
		if result.Purchases[i].Access != want {
			t.Fatalf("purchase %d: expected access %v, got %v", i, want, result.Purchases[i].Access)
		}
	}
	if !result.Purchases[2].Program.Deleted {
		t.Fatalf("soft-deleted program must be flagged as deleted")
	}
}

func TestListPurchasesInvalidInputs(t *testing.T) {
	svc := newTestService(&stubAdminPurchaseRepository{}, &stubAdminProgramRepository{})

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name   string
		filter admin_commerce.PurchaseFilter
		page   int
		limit  int
	}{
		{name: "page zero", page: 0, limit: 20},
		{name: "limit zero", page: 1, limit: 0},
		{name: "invalid status", filter: admin_commerce.PurchaseFilter{Status: "refunded"}},
		{name: "invalid program id", filter: admin_commerce.PurchaseFilter{ProgramID: "not-a-uuid"}},
		{name: "inverted date range", filter: admin_commerce.PurchaseFilter{CreatedFrom: &to, CreatedTo: &from}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.ListPurchases(context.Background(), tc.filter, tc.page, tc.limit); !errors.Is(err, admin_commerce.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestListPurchasesClampsLimit(t *testing.T) {
	stub := &stubAdminPurchaseRepository{}
	svc := newTestService(stub, &stubAdminProgramRepository{})
	result, err := svc.ListPurchases(context.Background(), admin_commerce.PurchaseFilter{}, 1, 999_999)
	if err != nil {
		t.Fatalf("ListPurchases: %v", err)
	}
	if result.Limit != admin_commerce.MaxPageSize {
		t.Fatalf("expected limit clamped to %d, got %d", admin_commerce.MaxPageSize, result.Limit)
	}
	if stub.gotLimit != admin_commerce.MaxPageSize {
		t.Fatalf("expected repository limit %d, got %d", admin_commerce.MaxPageSize, stub.gotLimit)
	}
}

func TestGetPurchaseSuccess(t *testing.T) {
	purchase := completedPublishedPurchase()
	purchase.Test = false
	stub := &stubAdminPurchaseRepository{found: &purchase}
	svc := newTestService(stub, &stubAdminProgramRepository{})

	got, err := svc.GetPurchase(context.Background(), purchase.ID)
	if err != nil {
		t.Fatalf("GetPurchase: %v", err)
	}
	if got.ID != purchase.ID || got.Access != true || got.Customer.Name != "Ana Sousa" {
		t.Fatalf("purchase not mapped: %+v", got)
	}
}

func TestGetPurchaseNotFound(t *testing.T) {
	stub := &stubAdminPurchaseRepository{findErr: repositories.ErrPurchaseNotFound}
	svc := newTestService(stub, &stubAdminProgramRepository{})

	if _, err := svc.GetPurchase(context.Background(), "00000000-0000-0000-0000-000000000001"); !errors.Is(err, admin_commerce.ErrPurchaseNotFound) {
		t.Fatalf("expected ErrPurchaseNotFound, got %v", err)
	}
}

func TestGetPurchaseInvalidID(t *testing.T) {
	svc := newTestService(&stubAdminPurchaseRepository{}, &stubAdminProgramRepository{})
	for _, id := range []string{"", "not-a-uuid"} {
		if _, err := svc.GetPurchase(context.Background(), id); !errors.Is(err, admin_commerce.ErrInvalidInput) {
			t.Fatalf("id %q: expected ErrInvalidInput, got %v", id, err)
		}
	}
}

func TestListProgramSalesSuccess(t *testing.T) {
	rows := []repositories.AdminProgramSalesRow{
		{
			ProgramID:         "22222222-2222-2222-2222-222222222222",
			Total:             5,
			Completed:         3,
			RealSales:         2,
			Pending:           1,
			Failed:            1,
			TestCount:         1,
			RevenueMinorUnits: 9000,
		},
	}
	stub := &stubAdminPurchaseRepository{rows: rows, total: 1}
	programs := &stubAdminProgramRepository{
		programs: []models.Program{{
			ID:              "22222222-2222-2222-2222-222222222222",
			Name:            "Muscle Builder",
			Type:            models.ProgramTypePremium,
			Status:          models.ProgramStatusPublished,
			PriceMinorUnits: 4500,
		}},
	}
	svc := newTestService(stub, programs)

	result, err := svc.ListProgramSales(context.Background(), admin_commerce.PurchaseFilter{}, 1, 20)
	if err != nil {
		t.Fatalf("ListProgramSales: %v", err)
	}
	if result.Total != 1 || len(result.Sales) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	sale := result.Sales[0]
	if sale.Program.Name != "Muscle Builder" || sale.CompletedSales != 3 || sale.RealSales != 2 || sale.RevenueMinorUnits != 9000 || sale.TestPurchases != 1 {
		t.Fatalf("sales row not mapped: %+v", sale)
	}
	if len(programs.programs) != 1 {
		t.Fatalf("programs were not requested")
	}
}

func TestListProgramSalesKeepsDeletedProgram(t *testing.T) {
	rows := []repositories.AdminProgramSalesRow{
		{
			ProgramID:         "22222222-2222-2222-2222-222222222222",
			Total:             1,
			Completed:         1,
			RevenueMinorUnits: 4500,
		},
	}
	program := models.Program{
		ID:              "22222222-2222-2222-2222-222222222222",
		Name:            "Retired Pack",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		DeletedAt:       gormDeletedAt(),
		PriceMinorUnits: 4500,
	}
	stub := &stubAdminPurchaseRepository{rows: rows, total: 1}
	svc := newTestService(stub, &stubAdminProgramRepository{programs: []models.Program{program}})

	result, err := svc.ListProgramSales(context.Background(), admin_commerce.PurchaseFilter{}, 1, 20)
	if err != nil {
		t.Fatalf("ListProgramSales: %v", err)
	}
	if !result.Sales[0].Program.Deleted {
		t.Fatalf("soft-deleted program must be flagged as deleted")
	}
	if result.Sales[0].Program.Name != "Retired Pack" {
		t.Fatalf("deleted program must keep its name: %+v", result.Sales[0].Program)
	}
}

func TestListProgramSalesMissingProgramIsDefensive(t *testing.T) {
	rows := []repositories.AdminProgramSalesRow{
		{ProgramID: "22222222-2222-2222-2222-222222222222", Total: 1, Completed: 1},
	}
	stub := &stubAdminPurchaseRepository{rows: rows, total: 1}
	svc := newTestService(stub, &stubAdminProgramRepository{programs: nil})

	result, err := svc.ListProgramSales(context.Background(), admin_commerce.PurchaseFilter{}, 1, 20)
	if err != nil {
		t.Fatalf("ListProgramSales: %v", err)
	}
	if result.Sales[0].Program.ID != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("sale must keep its attributed program id: %+v", result.Sales[0].Program)
	}
}

func TestListProgramSalesBatchReturnsEveryRequestedProgram(t *testing.T) {
	const (
		idA = "22222222-2222-2222-2222-222222222222"
		idB = "33333333-3333-3333-3333-333333333333"
		idC = "44444444-4444-4444-4444-444444444444"
	)
	rows := []repositories.AdminProgramSalesRow{
		{
			ProgramID:         idB,
			Total:             4,
			Completed:         3,
			RealSales:         2,
			Pending:           1,
			TestCount:         1,
			RevenueMinorUnits: 9000,
		},
	}
	stub := &stubAdminPurchaseRepository{rows: rows}
	programs := &stubAdminProgramRepository{
		programs: []models.Program{
			{ID: idA, Name: "Program A"},
			{ID: idB, Name: "Program B"},
			{ID: idC, Name: "Program C"},
		},
	}
	svc := newTestService(stub, programs)

	filter := admin_commerce.PurchaseFilter{ProgramIDs: []string{idA, idB, idC}}
	result, err := svc.ListProgramSales(context.Background(), filter, 1, 20)
	if err != nil {
		t.Fatalf("ListProgramSales: %v", err)
	}
	if result.Total != 3 || len(result.Sales) != 3 {
		t.Fatalf("expected one row per requested program, got %+v", result)
	}
	// Request order must be preserved while absent programs are zero-filled.
	if result.Sales[0].Program.ID != idA || result.Sales[0].Program.Name != "Program A" {
		t.Fatalf("first row must be program A: %+v", result.Sales[0])
	}
	if result.Sales[0].CompletedSales != 0 || result.Sales[0].RealSales != 0 || result.Sales[0].RevenueMinorUnits != 0 {
		t.Fatalf("program A has no purchases and must be zero-filled: %+v", result.Sales[0])
	}
	if result.Sales[1].Program.ID != idB || result.Sales[1].CompletedSales != 3 || result.Sales[1].RealSales != 2 || result.Sales[1].RevenueMinorUnits != 9000 {
		t.Fatalf("second row must carry program B counts: %+v", result.Sales[1])
	}
	if result.Sales[2].Program.ID != idC || result.Sales[2].CompletedSales != 0 {
		t.Fatalf("third row must be zero-filled program C: %+v", result.Sales[2])
	}
}

func TestListProgramSalesBatchSeparatesRealFromTest(t *testing.T) {
	const id = "22222222-2222-2222-2222-222222222222"
	rows := []repositories.AdminProgramSalesRow{
		{
			ProgramID:         id,
			Total:             6,
			Completed:         4,
			RealSales:         3,
			Pending:           1,
			Failed:            1,
			TestCount:         2,
			RevenueMinorUnits: 13500,
		},
	}
	stub := &stubAdminPurchaseRepository{rows: rows}
	svc := newTestService(stub, &stubAdminProgramRepository{programs: []models.Program{{ID: id}}})

	result, err := svc.ListProgramSales(context.Background(), admin_commerce.PurchaseFilter{ProgramIDs: []string{id}}, 1, 20)
	if err != nil {
		t.Fatalf("ListProgramSales: %v", err)
	}
	sale := result.Sales[0]
	if sale.RealSales != 3 {
		t.Fatalf("real sales must exclude test purchases, got %d", sale.RealSales)
	}
	if sale.CompletedSales != 4 {
		t.Fatalf("completed sales keeps the inclusive count, got %d", sale.CompletedSales)
	}
	if sale.TestPurchases != 2 || sale.PendingPurchases != 1 || sale.FailedPurchases != 1 {
		t.Fatalf("pending/failed/test not separated: %+v", sale)
	}
	if sale.RevenueMinorUnits != 13500 {
		t.Fatalf("revenue must exclude test purchases, got %d", sale.RevenueMinorUnits)
	}
}

func TestListProgramSalesBatchDeduplicatesRequestedIDs(t *testing.T) {
	const id = "22222222-2222-2222-2222-222222222222"
	stub := &stubAdminPurchaseRepository{}
	svc := newTestService(stub, &stubAdminProgramRepository{programs: []models.Program{{ID: id}}})

	result, err := svc.ListProgramSales(context.Background(), admin_commerce.PurchaseFilter{ProgramIDs: []string{id, id, id}}, 1, 20)
	if err != nil {
		t.Fatalf("ListProgramSales: %v", err)
	}
	if len(result.Sales) != 1 || result.Total != 1 {
		t.Fatalf("duplicated ids must collapse to one row, got %+v", result)
	}
	if len(stub.gotFilter.ProgramIDs) != 1 {
		t.Fatalf("repository must receive the deduplicated set, got %v", stub.gotFilter.ProgramIDs)
	}
}

func TestListProgramSalesBatchInvalidIDs(t *testing.T) {
	svc := newTestService(&stubAdminPurchaseRepository{}, &stubAdminProgramRepository{})
	filter := admin_commerce.PurchaseFilter{ProgramIDs: []string{"00000000-0000-0000-0000-000000000001", "not-a-uuid"}}
	if _, err := svc.ListProgramSales(context.Background(), filter, 1, 20); !errors.Is(err, admin_commerce.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for malformed program ids, got %v", err)
	}

	tooMany := make([]string, admin_commerce.MaxPageSize+1)
	for i := range tooMany {
		tooMany[i] = "00000000-0000-0000-0000-000000000001"
	}
	filter = admin_commerce.PurchaseFilter{ProgramIDs: tooMany}
	if _, err := svc.ListProgramSales(context.Background(), filter, 1, 20); !errors.Is(err, admin_commerce.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for an oversized program id set, got %v", err)
	}
}

func TestListProgramSalesBatchKeepsRetiredProgramIdentity(t *testing.T) {
	const id = "22222222-2222-2222-2222-222222222222"
	rows := []repositories.AdminProgramSalesRow{
		{ProgramID: id, Total: 1, Completed: 1, RealSales: 1, RevenueMinorUnits: 4500},
	}
	program := models.Program{ID: id, Name: "Retired Pack", Status: models.ProgramStatusPublished, DeletedAt: gormDeletedAt()}
	stub := &stubAdminPurchaseRepository{rows: rows}
	svc := newTestService(stub, &stubAdminProgramRepository{programs: []models.Program{program}})

	result, err := svc.ListProgramSales(context.Background(), admin_commerce.PurchaseFilter{ProgramIDs: []string{id}}, 1, 20)
	if err != nil {
		t.Fatalf("ListProgramSales: %v", err)
	}
	sale := result.Sales[0]
	if !sale.Program.Deleted || sale.Program.Name != "Retired Pack" {
		t.Fatalf("retired program must keep its identity: %+v", sale.Program)
	}
	if sale.RealSales != 1 || sale.RevenueMinorUnits != 4500 {
		t.Fatalf("retired program must keep its historical sales: %+v", sale)
	}
}
