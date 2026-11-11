package member_feedback

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

const memberID = "11111111-1111-4111-8111-111111111111"

type fakeRepo struct {
	heldPlan  bool
	pending   *models.MemberFeedback
	created   []*models.MemberFeedback
	reviewErr error
	reviewed  struct {
		id, status, reviewer string
	}
	listStatus string
}

func (f *fakeRepo) Create(_ context.Context, feedback *models.MemberFeedback) error {
	if f.pending != nil {
		return repositories.ErrMemberFeedbackAlreadyPending
	}
	f.created = append(f.created, feedback)
	f.pending = feedback
	return nil
}

func (f *fakeRepo) FindPendingByUserID(_ context.Context, _ string) (*models.MemberFeedback, error) {
	if f.pending == nil {
		return nil, repositories.ErrMemberFeedbackNotFound
	}
	return f.pending, nil
}

func (f *fakeRepo) HasHeldPlan(_ context.Context, _ string) (bool, error) { return f.heldPlan, nil }

func (f *fakeRepo) ListPublished(_ context.Context, _, _ int) ([]models.MemberFeedback, int64, error) {
	return []models.MemberFeedback{{ID: "a"}}, 1, nil
}

func (f *fakeRepo) ListByStatus(_ context.Context, status string, _, _ int) ([]models.MemberFeedback, int64, error) {
	f.listStatus = status
	return nil, 0, nil
}

func (f *fakeRepo) Review(_ context.Context, id, status, reviewer string, _ time.Time) (*models.MemberFeedback, error) {
	if f.reviewErr != nil {
		return nil, f.reviewErr
	}
	f.reviewed.id, f.reviewed.status, f.reviewed.reviewer = id, status, reviewer
	return &models.MemberFeedback{ID: id, Status: status}, nil
}

type fakeUsers struct{ user *models.User }

func (f fakeUsers) FindByID(_ context.Context, _ string) (*models.User, error) {
	if f.user == nil {
		return nil, repositories.ErrUserNotFound
	}
	return f.user, nil
}

func newTestService(repo *fakeRepo) Service {
	return NewService(repo, fakeUsers{user: &models.User{ID: memberID, FirstName: "Alex", LastName: "doe"}})
}

func TestSubmitCreatesPendingFeedbackWithPublicName(t *testing.T) {
	repo := &fakeRepo{heldPlan: true}
	feedback, err := newTestService(repo).Submit(context.Background(), memberID, SubmitInput{Rating: 5, Message: "  Great programme,\r\nreally enjoyed it.  "})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if feedback.Status != models.MemberFeedbackStatusPending || feedback.UserID != memberID || feedback.Rating != 5 {
		t.Fatalf("unexpected feedback %+v", feedback)
	}
	if feedback.DisplayName != "Alex D." {
		t.Fatalf("display name = %q, want Alex D.", feedback.DisplayName)
	}
	if feedback.Message != "Great programme, \nreally enjoyed it." {
		t.Fatalf("message = %q", feedback.Message)
	}
}

func TestSubmitRequiresAPlan(t *testing.T) {
	_, err := newTestService(&fakeRepo{heldPlan: false}).Submit(context.Background(), memberID, SubmitInput{Rating: 4, Message: "Solid structure overall."})
	if !errors.Is(err, ErrNotEligible) {
		t.Fatalf("err = %v, want ErrNotEligible", err)
	}
}

func TestSubmitRejectsASecondPendingSubmission(t *testing.T) {
	repo := &fakeRepo{heldPlan: true, pending: &models.MemberFeedback{ID: "p"}}
	_, err := newTestService(repo).Submit(context.Background(), memberID, SubmitInput{Rating: 4, Message: "Another message here."})
	if !errors.Is(err, ErrAlreadyPending) {
		t.Fatalf("err = %v, want ErrAlreadyPending", err)
	}
}

func TestSubmitValidatesInput(t *testing.T) {
	cases := []SubmitInput{
		{Rating: 0, Message: "Long enough message."},
		{Rating: 6, Message: "Long enough message."},
		{Rating: 3, Message: "too short"},
		{Rating: 3, Message: strings.Repeat("a", MaxMessageLength+1)},
		{Rating: 3, Message: "\x00\x01\x02          "},
	}
	for _, input := range cases {
		if _, err := newTestService(&fakeRepo{heldPlan: true}).Submit(context.Background(), memberID, input); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("input %+v: err = %v, want ErrInvalidInput", input, err)
		}
	}
	if _, err := newTestService(&fakeRepo{heldPlan: true}).Submit(context.Background(), "not-a-uuid", SubmitInput{Rating: 3, Message: "Long enough message."}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid user id accepted: %v", err)
	}
}

func TestSubmitRequiresAnActiveAccount(t *testing.T) {
	svc := NewService(&fakeRepo{heldPlan: true}, fakeUsers{})
	if _, err := svc.Submit(context.Background(), memberID, SubmitInput{Rating: 3, Message: "Long enough message."}); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("err = %v, want ErrUserNotFound", err)
	}
}

func TestEligibilityReportsPlanAndPendingSubmission(t *testing.T) {
	repo := &fakeRepo{heldPlan: true, pending: &models.MemberFeedback{ID: "p"}}
	result, err := newTestService(repo).GetEligibility(context.Background(), memberID)
	if err != nil {
		t.Fatalf("eligibility: %v", err)
	}
	if !result.Eligible || result.Pending == nil || result.Pending.ID != "p" {
		t.Fatalf("unexpected eligibility %+v", result)
	}
	result, err = newTestService(&fakeRepo{}).GetEligibility(context.Background(), memberID)
	if err != nil || result.Eligible || result.Pending != nil {
		t.Fatalf("member without plan: %+v %v", result, err)
	}
}

func TestReviewMapsDecisionsAndRecordsReviewer(t *testing.T) {
	repo := &fakeRepo{}
	id := "22222222-2222-4222-8222-222222222222"
	if _, err := newTestService(repo).Review(context.Background(), id, "ADMIN_2", DecisionApprove); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if repo.reviewed.status != models.MemberFeedbackStatusApproved || repo.reviewed.reviewer != "ADMIN_2" || repo.reviewed.id != id {
		t.Fatalf("unexpected review %+v", repo.reviewed)
	}
	if _, err := newTestService(repo).Review(context.Background(), id, "ADMIN_2", DecisionReject); err != nil || repo.reviewed.status != models.MemberFeedbackStatusRejected {
		t.Fatalf("reject: %v %+v", err, repo.reviewed)
	}
	if _, err := newTestService(repo).Review(context.Background(), id, "ADMIN_2", Decision("publish")); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("unknown decision accepted: %v", err)
	}
	conflict := &fakeRepo{reviewErr: repositories.ErrMemberFeedbackStateConflict}
	if _, err := newTestService(conflict).Review(context.Background(), id, "ADMIN_2", DecisionApprove); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("err = %v, want ErrStateConflict", err)
	}
	missing := &fakeRepo{reviewErr: repositories.ErrMemberFeedbackNotFound}
	if _, err := newTestService(missing).Review(context.Background(), id, "ADMIN_2", DecisionApprove); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestListsValidateAndClampPagination(t *testing.T) {
	repo := &fakeRepo{}
	result, err := newTestService(repo).ListPublished(context.Background(), 1, 500)
	if err != nil || result.Limit != MaxPageSize || result.Total != 1 {
		t.Fatalf("published: %+v %v", result, err)
	}
	if _, err := newTestService(repo).ListPublished(context.Background(), 0, 10); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("page 0 accepted: %v", err)
	}
	if _, err := newTestService(repo).ListForReview(context.Background(), "pending", 1, 0); err != nil || repo.listStatus != "pending" {
		t.Fatalf("review list: %v %q", err, repo.listStatus)
	}
	if _, err := newTestService(repo).ListForReview(context.Background(), "everything", 1, 10); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("unknown status accepted: %v", err)
	}
}

func TestPublicNameNeverShowsTheFullSurname(t *testing.T) {
	cases := map[[2]string]string{
		{"Alex", "Doe"}:    "Alex D.",
		{" ana ", "émile"}: "ana É.",
		{"Alex", ""}:       "Alex",
		{"", "Doe"}:        "D.",
		{"", ""}:           "RYZE member",
	}
	for in, want := range cases {
		if got := PublicName(in[0], in[1]); got != want {
			t.Fatalf("PublicName(%q, %q) = %q, want %q", in[0], in[1], got, want)
		}
	}
}
