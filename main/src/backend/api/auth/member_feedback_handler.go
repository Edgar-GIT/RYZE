package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/adminauthcontext"
	"ryze/backend/middleware/authcontext"
	"ryze/backend/models"
	"ryze/backend/services/member_feedback"
)

// publicFeedbackResponse is approved feedback as shown on the public Feedback
// page: the public name only ("First L."), never the account or its email.
type publicFeedbackResponse struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	Rating      int       `json:"rating"`
	Message     string    `json:"message"`
	PublishedAt time.Time `json:"published_at"`
}

func newPublicFeedbackResponse(feedback *models.MemberFeedback) publicFeedbackResponse {
	published := feedback.CreatedAt
	if feedback.ReviewedAt != nil {
		published = *feedback.ReviewedAt
	}
	return publicFeedbackResponse{
		ID:          feedback.ID,
		DisplayName: feedback.DisplayName,
		Rating:      feedback.Rating,
		Message:     feedback.Message,
		PublishedAt: published,
	}
}

// ownFeedbackResponse is the member's own submission.
type ownFeedbackResponse struct {
	ID        string    `json:"id"`
	Rating    int       `json:"rating"`
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func newOwnFeedbackResponse(feedback *models.MemberFeedback) ownFeedbackResponse {
	return ownFeedbackResponse{
		ID:        feedback.ID,
		Rating:    feedback.Rating,
		Message:   feedback.Message,
		Status:    feedback.Status,
		CreatedAt: feedback.CreatedAt,
	}
}

// adminFeedbackResponse is one submission as seen by a reviewer.
type adminFeedbackResponse struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	DisplayName string     `json:"display_name"`
	Rating      int        `json:"rating"`
	Message     string     `json:"message"`
	Status      string     `json:"status"`
	ReviewedBy  *string    `json:"reviewed_by"`
	ReviewedAt  *time.Time `json:"reviewed_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

func newAdminFeedbackResponse(feedback *models.MemberFeedback) adminFeedbackResponse {
	return adminFeedbackResponse{
		ID:          feedback.ID,
		UserID:      feedback.UserID,
		DisplayName: feedback.DisplayName,
		Rating:      feedback.Rating,
		Message:     feedback.Message,
		Status:      feedback.Status,
		ReviewedBy:  feedback.ReviewedBy,
		ReviewedAt:  feedback.ReviewedAt,
		CreatedAt:   feedback.CreatedAt,
	}
}

// MemberFeedbackHandler serves the public feedback list and the member's own
// eligibility and submission. Identity always comes from the authenticated
// context; the request body never names a user.
type MemberFeedbackHandler struct {
	service member_feedback.Service
}

func NewMemberFeedbackHandler(service member_feedback.Service) *MemberFeedbackHandler {
	return &MemberFeedbackHandler{service: service}
}

// ListPublished returns approved feedback, newest first.
func (h *MemberFeedbackHandler) ListPublished(c *gin.Context) {
	page, limit, ok := feedbackPage(c)
	if !ok {
		return
	}
	result, err := h.service.ListPublished(c.Request.Context(), page, limit)
	if err != nil {
		respondFeedbackError(c, err)
		return
	}
	items := make([]publicFeedbackResponse, 0, len(result.Feedback))
	for i := range result.Feedback {
		items = append(items, newPublicFeedbackResponse(&result.Feedback[i]))
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Feedback retrieved successfully.",
		"data":    gin.H{"feedback": items, "pagination": feedbackPagination(result)},
	})
}

// GetEligibility tells the member whether they can leave feedback now.
func (h *MemberFeedbackHandler) GetEligibility(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	result, err := h.service.GetEligibility(c.Request.Context(), userID)
	if err != nil {
		respondFeedbackError(c, err)
		return
	}
	var pending *ownFeedbackResponse
	if result.Pending != nil {
		response := newOwnFeedbackResponse(result.Pending)
		pending = &response
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Feedback eligibility retrieved successfully.",
		"data":    gin.H{"eligible": result.Eligible, "pending": pending},
	})
}

type submitFeedbackRequest struct {
	Rating  int    `json:"rating"`
	Message string `json:"message"`
}

// Submit stores the member's feedback as pending review.
func (h *MemberFeedbackHandler) Submit(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	var body submitFeedbackRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	feedback, err := h.service.Submit(c.Request.Context(), userID, member_feedback.SubmitInput{Rating: body.Rating, Message: body.Message})
	if err != nil {
		respondFeedbackError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Feedback received. It will appear after review.",
		"data":    newOwnFeedbackResponse(feedback),
	})
}

// AdminMemberFeedbackHandler exposes the review queue. Authentication and the
// feedback.read / feedback.manage permissions are enforced by the route
// middleware.
type AdminMemberFeedbackHandler struct {
	service member_feedback.Service
}

func NewAdminMemberFeedbackHandler(service member_feedback.Service) *AdminMemberFeedbackHandler {
	return &AdminMemberFeedbackHandler{service: service}
}

// List returns feedback for review, optionally filtered by status.
func (h *AdminMemberFeedbackHandler) List(c *gin.Context) {
	page, limit, ok := feedbackPage(c)
	if !ok {
		return
	}
	result, err := h.service.ListForReview(c.Request.Context(), c.Query("status"), page, limit)
	if err != nil {
		respondFeedbackError(c, err)
		return
	}
	items := make([]adminFeedbackResponse, 0, len(result.Feedback))
	for i := range result.Feedback {
		items = append(items, newAdminFeedbackResponse(&result.Feedback[i]))
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Feedback retrieved successfully.",
		"data":    gin.H{"feedback": items, "pagination": feedbackPagination(result)},
	})
}

// Approve publishes a pending submission.
func (h *AdminMemberFeedbackHandler) Approve(c *gin.Context) {
	h.review(c, member_feedback.DecisionApprove, "Feedback approved.")
}

// Reject keeps a pending submission off the public page.
func (h *AdminMemberFeedbackHandler) Reject(c *gin.Context) {
	h.review(c, member_feedback.DecisionReject, "Feedback rejected.")
}

func (h *AdminMemberFeedbackHandler) review(c *gin.Context, decision member_feedback.Decision, message string) {
	adminID, err := adminauthcontext.AdminIdentityFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	feedback, err := h.service.Review(c.Request.Context(), c.Param("id"), adminID, decision)
	if err != nil {
		respondFeedbackError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
		"data":    newAdminFeedbackResponse(feedback),
	})
}

func feedbackPage(c *gin.Context) (int, int, bool) {
	page, err := queryInt(c, "page", 1)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return 0, 0, false
	}
	limit, err := queryInt(c, "limit", member_feedback.DefaultPageSize)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return 0, 0, false
	}
	return page, limit, true
}

func feedbackPagination(result member_feedback.ListResult) gin.H {
	return gin.H{
		"page":        result.Page,
		"limit":       result.Limit,
		"total":       result.Total,
		"total_pages": totalPages(result.Total, result.Limit),
	}
}

func respondFeedbackError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, member_feedback.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, member_feedback.ErrUserNotFound):
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
	case errors.Is(err, member_feedback.ErrNotEligible):
		RespondError(c, http.StatusForbidden, "FEEDBACK_NOT_ELIGIBLE", "Feedback opens once you have a plan.", nil)
	case errors.Is(err, member_feedback.ErrAlreadyPending):
		RespondError(c, http.StatusConflict, "FEEDBACK_ALREADY_PENDING", "Your previous feedback is still being reviewed.", nil)
	case errors.Is(err, member_feedback.ErrNotFound):
		RespondError(c, http.StatusNotFound, "FEEDBACK_NOT_FOUND", "Feedback not found.", nil)
	case errors.Is(err, member_feedback.ErrStateConflict):
		RespondError(c, http.StatusConflict, "FEEDBACK_STATE_CONFLICT", "Feedback was already reviewed.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
