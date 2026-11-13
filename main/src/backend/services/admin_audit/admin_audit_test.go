package admin_audit

import (
	"context"
	"errors"
	"testing"

	"ryze/backend/config"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

type fakeAuditRepo struct {
	created []*models.AdminAuditLog
	filter  repositories.AdminAuditFilter
}

func (f *fakeAuditRepo) Create(_ context.Context, entry *models.AdminAuditLog) error {
	f.created = append(f.created, entry)
	return nil
}

func (f *fakeAuditRepo) List(_ context.Context, filter repositories.AdminAuditFilter, _, _ int) ([]models.AdminAuditLog, int64, error) {
	f.filter = filter
	target := "u-1"
	return []models.AdminAuditLog{{ID: "a", AdminID: config.Admin1ID, Action: "user.disable", Method: "PATCH", Route: "/r", TargetID: &target, StatusCode: 200, Outcome: "success"}}, 1, nil
}

func TestRecordDerivesActionOutcomeAndNormalisesIP(t *testing.T) {
	repo := &fakeAuditRepo{}
	svc := New(repo)
	err := svc.Record(context.Background(), Entry{
		AdminID:    config.Admin2ID,
		Method:     "post",
		Route:      "/api/v1/admin/feedback/:id/approve",
		TargetID:   "f-1",
		StatusCode: 409,
		ClientIP:   "::ffff:127.0.0.1",
	})
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	got := repo.created[0]
	if got.Action != "feedback.approve" || got.Method != "POST" || got.Outcome != models.AdminAuditOutcomeFailed {
		t.Fatalf("unexpected record: %+v", got)
	}
	if got.TargetID == nil || *got.TargetID != "f-1" || got.ClientIP == nil || *got.ClientIP != "127.0.0.1" {
		t.Fatalf("unexpected target/ip: %+v", got)
	}
}

func TestRecordRejectsUnknownIdentityAndIncompleteEntries(t *testing.T) {
	svc := New(&fakeAuditRepo{})
	for _, entry := range []Entry{
		{AdminID: "ADMIN_3", Method: "POST", Route: "/r", StatusCode: 200},
		{AdminID: config.Admin1ID, Method: "", Route: "/r", StatusCode: 200},
		{AdminID: config.Admin1ID, Method: "POST", Route: "/r", StatusCode: 0},
	} {
		if err := svc.Record(context.Background(), entry); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("Record(%+v) = %v, want ErrInvalidInput", entry, err)
		}
	}
}

func TestRecordDropsAnUnparseableAddress(t *testing.T) {
	repo := &fakeAuditRepo{}
	if err := New(repo).Record(context.Background(), Entry{AdminID: config.Admin1ID, Method: "DELETE", Route: "/x", StatusCode: 204, ClientIP: "not an ip"}); err != nil {
		t.Fatalf("record: %v", err)
	}
	if repo.created[0].ClientIP != nil {
		t.Fatalf("an unparseable address must not be stored: %v", *repo.created[0].ClientIP)
	}
}

func TestListValidatesFilters(t *testing.T) {
	repo := &fakeAuditRepo{}
	svc := New(repo)
	ctx := context.Background()
	for _, tc := range []struct {
		filter      ListFilter
		page, limit int
	}{
		{ListFilter{}, 0, 10},
		{ListFilter{}, 1, MaxPageSize + 1},
		{ListFilter{AdminID: "root"}, 1, 10},
		{ListFilter{Outcome: "maybe"}, 1, 10},
	} {
		if _, err := svc.List(ctx, tc.filter, tc.page, tc.limit); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("List(%+v, %d, %d) = %v, want ErrInvalidInput", tc.filter, tc.page, tc.limit, err)
		}
	}
	result, err := svc.List(ctx, ListFilter{AdminID: config.Admin1ID, Outcome: "success"}, 1, 10)
	if err != nil || len(result.Records) != 1 || result.Records[0].TargetID != "u-1" {
		t.Fatalf("List = %+v, %v", result, err)
	}
	if repo.filter.AdminID != config.Admin1ID || repo.filter.Outcome != "success" {
		t.Fatalf("filter not forwarded: %+v", repo.filter)
	}
}

func TestActionOutcomeAndScope(t *testing.T) {
	if got := ActionFor("POST", "/api/v1/programs/generic/:programID/publish"); got != "generic_program.publish" {
		t.Fatalf("ActionFor = %q", got)
	}
	if got := ActionFor("PUT", "/api/v1/admin/unknown"); got != "put /api/v1/admin/unknown" {
		t.Fatalf("fallback ActionFor = %q", got)
	}
	for status, want := range map[int]string{200: "success", 204: "success", 302: "success", 401: "denied", 403: "denied", 400: "failed", 404: "failed", 500: "failed"} {
		if got := OutcomeFor(status); got != want {
			t.Fatalf("OutcomeFor(%d) = %q, want %q", status, got, want)
		}
	}
	if ShouldAudit("GET", "/api/v1/admin/users") || !ShouldAudit("GET", "/api/v1/admin/x/download") || !ShouldAudit("DELETE", "/api/v1/admin/x") {
		t.Fatal("ShouldAudit scope is wrong")
	}
}
