package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/adminauthcontext"
	"ryze/backend/middleware/authcontext"
	"ryze/backend/models"
	"ryze/backend/services/member_feedback"
)

const feedbackMemberID = "11111111-1111-4111-8111-111111111111"

type fakeFeedbackService struct {
	submitErr   error
	submitted   member_feedback.SubmitInput
	submittedBy string
	reviewedBy  string
	decision    member_feedback.Decision
	eligibility member_feedback.Eligibility
	published   []models.MemberFeedback
}

func (f *fakeFeedbackService) GetEligibility(_ context.Context, _ string) (member_feedback.Eligibility, error) {
	return f.eligibility, nil
}

func (f *fakeFeedbackService) Submit(_ context.Context, userID string, input member_feedback.SubmitInput) (*models.MemberFeedback, error) {
	if f.submitErr != nil {
		return nil, f.submitErr
	}
	f.submitted, f.submittedBy = input, userID
	return &models.MemberFeedback{ID: "f1", UserID: userID, Rating: input.Rating, Message: input.Message, Status: models.MemberFeedbackStatusPending}, nil
}

func (f *fakeFeedbackService) ListPublished(_ context.Context, page, limit int) (member_feedback.ListResult, error) {
	return member_feedback.ListResult{Feedback: f.published, Total: int64(len(f.published)), Page: page, Limit: limit}, nil
}

func (f *fakeFeedbackService) ListForReview(_ context.Context, _ string, page, limit int) (member_feedback.ListResult, error) {
	return member_feedback.ListResult{Page: page, Limit: limit}, nil
}

func (f *fakeFeedbackService) Review(_ context.Context, id, adminID string, decision member_feedback.Decision) (*models.MemberFeedback, error) {
	f.reviewedBy, f.decision = adminID, decision
	return &models.MemberFeedback{ID: id, Status: models.MemberFeedbackStatusApproved}, nil
}

func feedbackRouter(service member_feedback.Service, userID, adminID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if userID != "" {
			c.Set(authcontext.UserIDContextKey, userID)
		}
		if adminID != "" {
			c.Set(adminauthcontext.AdminIdentityContextKey, adminID)
		}
	})
	member := NewMemberFeedbackHandler(service)
	admin := NewAdminMemberFeedbackHandler(service)
	router.GET("/api/v1/feedback", member.ListPublished)
	router.GET("/api/v1/me/feedback/eligibility", member.GetEligibility)
	router.POST("/api/v1/me/feedback", member.Submit)
	router.POST("/api/v1/admin/feedback/:id/approve", admin.Approve)
	router.POST("/api/v1/admin/feedback/:id/reject", admin.Reject)
	return router
}

func doFeedback(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestSubmitFeedbackUsesTheAuthenticatedMember(t *testing.T) {
	service := &fakeFeedbackService{}
	// A user_id in the body is ignored: identity only comes from the context.
	rec := doFeedback(feedbackRouter(service, feedbackMemberID, ""), http.MethodPost, "/api/v1/me/feedback", `{"rating":5,"message":"Great plan, clear sessions.","user_id":"someone-else"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if service.submittedBy != feedbackMemberID || service.submitted.Rating != 5 {
		t.Fatalf("submitted by %q with %+v", service.submittedBy, service.submitted)
	}
	if strings.Contains(rec.Body.String(), "user_id") {
		t.Fatalf("own response exposes the user id: %s", rec.Body.String())
	}
}

func TestSubmitFeedbackRequiresAuthentication(t *testing.T) {
	rec := doFeedback(feedbackRouter(&fakeFeedbackService{}, "", ""), http.MethodPost, "/api/v1/me/feedback", `{"rating":5,"message":"Great plan, clear sessions."}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestSubmitFeedbackMapsDomainErrors(t *testing.T) {
	cases := map[error]int{
		member_feedback.ErrInvalidInput:   http.StatusBadRequest,
		member_feedback.ErrNotEligible:    http.StatusForbidden,
		member_feedback.ErrAlreadyPending: http.StatusConflict,
		member_feedback.ErrUserNotFound:   http.StatusUnauthorized,
	}
	for err, want := range cases {
		rec := doFeedback(feedbackRouter(&fakeFeedbackService{submitErr: err}, feedbackMemberID, ""), http.MethodPost, "/api/v1/me/feedback", `{"rating":5,"message":"Great plan, clear sessions."}`)
		if rec.Code != want {
			t.Fatalf("%v: status = %d, want %d", err, rec.Code, want)
		}
	}
	rec := doFeedback(feedbackRouter(&fakeFeedbackService{}, feedbackMemberID, ""), http.MethodPost, "/api/v1/me/feedback", `{"rating":"five"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed body: status = %d", rec.Code)
	}
}

func TestPublicFeedbackShowsOnlyThePublicName(t *testing.T) {
	reviewed := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
	service := &fakeFeedbackService{published: []models.MemberFeedback{{
		ID: "f1", UserID: feedbackMemberID, DisplayName: "Alex D.", Rating: 5, Message: "Great.", Status: models.MemberFeedbackStatusApproved, ReviewedAt: &reviewed,
	}}}
	rec := doFeedback(feedbackRouter(service, "", ""), http.MethodGet, "/api/v1/feedback", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		Data struct {
			Feedback []map[string]any `json:"feedback"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || len(body.Data.Feedback) != 1 {
		t.Fatalf("decode: %v %s", err, rec.Body.String())
	}
	item := body.Data.Feedback[0]
	if item["display_name"] != "Alex D." || item["user_id"] != nil || item["status"] != nil {
		t.Fatalf("public item exposes internal fields: %v", item)
	}
}

func TestAdminReviewRecordsTheAdministrator(t *testing.T) {
	service := &fakeFeedbackService{}
	rec := doFeedback(feedbackRouter(service, "", "ADMIN_2"), http.MethodPost, "/api/v1/admin/feedback/22222222-2222-4222-8222-222222222222/approve", "")
	if rec.Code != http.StatusOK || service.reviewedBy != "ADMIN_2" || service.decision != member_feedback.DecisionApprove {
		t.Fatalf("approve: %d %q %q", rec.Code, service.reviewedBy, service.decision)
	}
	rec = doFeedback(feedbackRouter(service, "", "ADMIN_2"), http.MethodPost, "/api/v1/admin/feedback/22222222-2222-4222-8222-222222222222/reject", "")
	if rec.Code != http.StatusOK || service.decision != member_feedback.DecisionReject {
		t.Fatalf("reject: %d %q", rec.Code, service.decision)
	}
	rec = doFeedback(feedbackRouter(service, "", ""), http.MethodPost, "/api/v1/admin/feedback/22222222-2222-4222-8222-222222222222/approve", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing admin identity: status = %d", rec.Code)
	}
}
