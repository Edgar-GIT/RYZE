package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/adminauthcontext"
	"ryze/backend/middleware/authcontext"
	"ryze/backend/models"
	"ryze/backend/services/trainer_applications"
)

// applicationResponse only exposes safe application information.
type applicationResponse struct {
	ID        string       `json:"id"`
	Status    string       `json:"status"`
	UserID    string       `json:"user_id"`
	User      userResponse `json:"user"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func newApplicationResponse(application *models.TrainerApplication) applicationResponse {
	return applicationResponse{
		ID:        application.ID,
		Status:    application.Status,
		UserID:    application.UserID,
		User:      newUserResponse(&application.User),
		CreatedAt: application.CreatedAt,
		UpdatedAt: application.UpdatedAt,
	}
}

// applicationDocumentResponse exposes the validated metadata of one evidence
// document. The storage key, the hash and the internal application link are
// never projected: the document is only reachable through the authenticated
// download endpoints.
type applicationDocumentResponse struct {
	ID               string    `json:"id"`
	Kind             string    `json:"kind"`
	OriginalFilename string    `json:"original_filename"`
	ContentType      string    `json:"content_type"`
	ByteSize         int64     `json:"byte_size"`
	CreatedAt        time.Time `json:"created_at"`
}

func newApplicationDocumentResponse(document *models.TrainerApplicationDocument) applicationDocumentResponse {
	return applicationDocumentResponse{
		ID:               document.ID,
		Kind:             document.Kind,
		OriginalFilename: document.OriginalFilename,
		ContentType:      document.ContentType,
		ByteSize:         document.ByteSize,
		CreatedAt:        document.CreatedAt,
	}
}

// applicationDecisionResponse exposes one append-only review decision to an
// administrator.
type applicationDecisionResponse struct {
	ID            string    `json:"id"`
	ApplicationID string    `json:"application_id"`
	AdminID       string    `json:"admin_id"`
	FromStatus    string    `json:"from_status"`
	ToStatus      string    `json:"to_status"`
	Reason        string    `json:"reason"`
	CreatedAt     time.Time `json:"created_at"`
}

func newApplicationDecisionResponse(decision *models.TrainerApplicationDecision) applicationDecisionResponse {
	return applicationDecisionResponse{
		ID:            decision.ID,
		ApplicationID: decision.ApplicationID,
		AdminID:       decision.AdminID,
		FromStatus:    decision.FromStatus,
		ToStatus:      decision.ToStatus,
		Reason:        decision.Reason,
		CreatedAt:     decision.CreatedAt,
	}
}

// AdminTrainerApplicationHandler exposes administrator trainer-application
// review operations. It never performs authentication or authorization itself:
// those are enforced by the AdminAuthenticate and RequireAdminPermission
// middleware mounted on the routes. All endpoints run under
// trainer_applications.read and trainer_applications.manage, both shared by
// every admin role.
type AdminTrainerApplicationHandler struct {
	service trainer_applications.Service
}

func NewAdminTrainerApplicationHandler(svc trainer_applications.Service) *AdminTrainerApplicationHandler {
	return &AdminTrainerApplicationHandler{service: svc}
}

// ListApplications returns one page of trainer applications, optionally
// filtered by status, with pagination metadata.
func (h *AdminTrainerApplicationHandler) ListApplications(c *gin.Context) {
	page, err := queryInt(c, "page", 1)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	limit, err := queryInt(c, "limit", 20)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}

	result, err := h.service.ListApplications(c.Request.Context(), page, limit, c.Query("status"))
	if err != nil {
		switch {
		case errors.Is(err, trainer_applications.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	applications := make([]applicationResponse, 0, len(result.Applications))
	for i := range result.Applications {
		applications = append(applications, newApplicationResponse(&result.Applications[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainer applications retrieved successfully.",
		"data": gin.H{
			"applications": applications,
			"pagination": gin.H{
				"page":        result.Page,
				"limit":       result.Limit,
				"total":       result.Total,
				"total_pages": totalPages(result.Total, result.Limit),
			},
		},
	})
}

// GetApplication returns one trainer application by UUID.
func (h *AdminTrainerApplicationHandler) GetApplication(c *gin.Context) {
	application, err := h.service.GetApplication(c.Request.Context(), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, trainer_applications.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, trainer_applications.ErrApplicationNotFound):
			RespondError(c, http.StatusNotFound, "APPLICATION_NOT_FOUND", "Trainer application not found.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainer application retrieved successfully.",
		"data":    newApplicationResponse(application),
	})
}

// ApproveApplication approves a PENDING application and creates the trainer
// profile for the applicant in the same database transaction. An optional
// {"reason": "..."} body is recorded with the reviewer decision.
func (h *AdminTrainerApplicationHandler) ApproveApplication(c *gin.Context) {
	adminID, err := adminauthcontext.AdminIdentityFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	reason, ok := reviewerReason(c)
	if !ok {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}

	application, err := h.service.ApproveApplication(c.Request.Context(), c.Param("id"), adminID, reason)
	if err != nil {
		switch {
		case errors.Is(err, trainer_applications.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, trainer_applications.ErrApplicationNotFound):
			RespondError(c, http.StatusNotFound, "APPLICATION_NOT_FOUND", "Trainer application not found.", nil)
		case errors.Is(err, trainer_applications.ErrApplicationStateConflict):
			RespondError(c, http.StatusConflict, "APPLICATION_STATE_CONFLICT", "Trainer application is not pending.", nil)
		case errors.Is(err, trainer_applications.ErrAlreadyTrainer):
			RespondError(c, http.StatusConflict, "USER_ALREADY_TRAINER", "User is already a trainer.", nil)
		case errors.Is(err, trainer_applications.ErrUserNotFound):
			RespondError(c, http.StatusConflict, "USER_NOT_ACTIVE", "User is not active.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainer application approved successfully.",
		"data":    newApplicationResponse(application),
	})
}

// RejectApplication rejects a PENDING application and records the reviewer
// decision. The application stays in history so the user can apply again. An
// optional {"reason": "..."} body is recorded with the decision.
func (h *AdminTrainerApplicationHandler) RejectApplication(c *gin.Context) {
	adminID, err := adminauthcontext.AdminIdentityFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	reason, ok := reviewerReason(c)
	if !ok {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}

	if err := h.service.RejectApplication(c.Request.Context(), c.Param("id"), adminID, reason); err != nil {
		switch {
		case errors.Is(err, trainer_applications.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, trainer_applications.ErrApplicationNotFound):
			RespondError(c, http.StatusNotFound, "APPLICATION_NOT_FOUND", "Trainer application not found.", nil)
		case errors.Is(err, trainer_applications.ErrApplicationStateConflict):
			RespondError(c, http.StatusConflict, "APPLICATION_STATE_CONFLICT", "Trainer application is not pending.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainer application rejected successfully.",
		"data":    gin.H{},
	})
}

// ListDocuments returns the active evidence documents of one application for a
// reviewer.
func (h *AdminTrainerApplicationHandler) ListDocuments(c *gin.Context) {
	documents, err := h.service.ListApplicationDocuments(c.Request.Context(), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, trainer_applications.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, trainer_applications.ErrApplicationNotFound):
			RespondError(c, http.StatusNotFound, "APPLICATION_NOT_FOUND", "Trainer application not found.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	responses := make([]applicationDocumentResponse, 0, len(documents))
	for i := range documents {
		responses = append(responses, newApplicationDocumentResponse(&documents[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainer application documents retrieved successfully.",
		"data": gin.H{
			"documents": responses,
		},
	})
}

// DownloadDocument streams one evidence document to an authenticated reviewer.
func (h *AdminTrainerApplicationHandler) DownloadDocument(c *gin.Context) {
	document, content, err := h.service.OpenApplicationDocument(c.Request.Context(), c.Param("id"), c.Param("documentID"))
	if err != nil {
		respondDocumentError(c, err)
		return
	}
	defer content.Close()
	respondDocumentDownload(c, document, content)
}

// ListDecisions returns the append-only review history of one application.
func (h *AdminTrainerApplicationHandler) ListDecisions(c *gin.Context) {
	decisions, err := h.service.ListDecisions(c.Request.Context(), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, trainer_applications.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, trainer_applications.ErrApplicationNotFound):
			RespondError(c, http.StatusNotFound, "APPLICATION_NOT_FOUND", "Trainer application not found.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	responses := make([]applicationDecisionResponse, 0, len(decisions))
	for i := range decisions {
		responses = append(responses, newApplicationDecisionResponse(&decisions[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainer application decisions retrieved successfully.",
		"data": gin.H{
			"decisions": responses,
		},
	})
}

// TrainerApplicationHandler exposes the trainer-application flow to regular
// authenticated users. The user identity always comes from the authenticated
// session; the request body can never influence it.
type TrainerApplicationHandler struct {
	service trainer_applications.Service
}

func NewTrainerApplicationHandler(svc trainer_applications.Service) *TrainerApplicationHandler {
	return &TrainerApplicationHandler{service: svc}
}

// ownApplicationResponse is the applicant's view of their own trainer
// application. It deliberately omits the user link: the caller is the user, so
// the user object would only echo data they already know.
type ownApplicationResponse struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// accessStatusResponse describes the authenticated caller's trainer access
// state. Application is null when the caller has never applied.
type accessStatusResponse struct {
	IsTrainer   bool                    `json:"is_trainer"`
	Application *ownApplicationResponse `json:"application"`
}

// Apply creates a PENDING trainer application for the authenticated user. The
// user ID is resolved exclusively from the authenticated session; any
// client-provided identity is ignored.
func (h *TrainerApplicationHandler) Apply(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	application, err := h.service.Apply(c.Request.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, trainer_applications.ErrUserNotFound):
			RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		case errors.Is(err, trainer_applications.ErrAlreadyTrainer):
			RespondError(c, http.StatusConflict, "ALREADY_TRAINER", "You are already a trainer.", nil)
		case errors.Is(err, trainer_applications.ErrApplicationAlreadyActive):
			RespondError(c, http.StatusConflict, "APPLICATION_ALREADY_EXISTS", "You already have an active trainer application.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Trainer application submitted successfully.",
		"data":    newApplicationResponse(application),
	})
}

// GetAccessStatus returns the authenticated user's trainer access state: the
// presence of an active trainer profile and their latest application, if any.
// The user ID is resolved exclusively from the authenticated session.
func (h *TrainerApplicationHandler) GetAccessStatus(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	status, err := h.service.GetAccessStatus(c.Request.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, trainer_applications.ErrInvalidInput), errors.Is(err, trainer_applications.ErrUserNotFound):
			RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	response := accessStatusResponse{IsTrainer: status.IsTrainer}
	if status.Application != nil {
		response.Application = &ownApplicationResponse{
			ID:        status.Application.ID,
			Status:    status.Application.Status,
			CreatedAt: status.Application.CreatedAt,
			UpdatedAt: status.Application.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainer access status retrieved successfully.",
		"data":    response,
	})
}

// UploadDocument accepts one multipart evidence document (fields: kind, file)
// for the caller's PENDING application. The file is validated server-side by
// its magic bytes and size; the client content type and filename are never
// trusted. The user id comes exclusively from the authenticated session.
func (h *TrainerApplicationHandler) UploadDocument(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	kind := c.PostForm("kind")
	fileHeader, err := c.FormFile("file")
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	defer file.Close()

	document, err := h.service.UploadDocument(c.Request.Context(), userID, kind, fileHeader.Filename, file)
	if err != nil {
		switch {
		case errors.Is(err, trainer_applications.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, trainer_applications.ErrApplicationNotFound):
			RespondError(c, http.StatusNotFound, "APPLICATION_NOT_FOUND", "Trainer application not found.", nil)
		case errors.Is(err, trainer_applications.ErrApplicationStateConflict):
			RespondError(c, http.StatusConflict, "APPLICATION_STATE_CONFLICT", "Trainer application is not pending.", nil)
		case errors.Is(err, trainer_applications.ErrDocumentUnsupportedType):
			RespondError(c, http.StatusBadRequest, "DOCUMENT_TYPE_NOT_ALLOWED", "Only PDF, JPEG and PNG documents are accepted.", nil)
		case errors.Is(err, trainer_applications.ErrDocumentTooLarge):
			RespondError(c, http.StatusRequestEntityTooLarge, "DOCUMENT_TOO_LARGE", "Document exceeds the maximum allowed size.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Trainer application document uploaded successfully.",
		"data":    newApplicationDocumentResponse(document),
	})
}

// ListOwnDocuments returns the caller's evidence documents for their latest
// application.
func (h *TrainerApplicationHandler) ListOwnDocuments(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	documents, err := h.service.ListOwnDocuments(c.Request.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, trainer_applications.ErrApplicationNotFound):
			RespondError(c, http.StatusNotFound, "APPLICATION_NOT_FOUND", "Trainer application not found.", nil)
		case errors.Is(err, trainer_applications.ErrUserNotFound), errors.Is(err, trainer_applications.ErrInvalidInput):
			RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	responses := make([]applicationDocumentResponse, 0, len(documents))
	for i := range documents {
		responses = append(responses, newApplicationDocumentResponse(&documents[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainer application documents retrieved successfully.",
		"data": gin.H{
			"documents": responses,
		},
	})
}

// DownloadOwnDocument streams one evidence document owned by the caller.
func (h *TrainerApplicationHandler) DownloadOwnDocument(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	document, content, err := h.service.OpenOwnDocument(c.Request.Context(), userID, c.Param("documentID"))
	if err != nil {
		respondDocumentError(c, err)
		return
	}
	defer content.Close()
	respondDocumentDownload(c, document, content)
}

// DeleteOwnDocument withdraws one evidence document while the application is
// still PENDING. After review the submission is locked.
func (h *TrainerApplicationHandler) DeleteOwnDocument(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	if err := h.service.DeleteOwnDocument(c.Request.Context(), userID, c.Param("documentID")); err != nil {
		switch {
		case errors.Is(err, trainer_applications.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, trainer_applications.ErrDocumentNotFound):
			RespondError(c, http.StatusNotFound, "DOCUMENT_NOT_FOUND", "Trainer application document not found.", nil)
		case errors.Is(err, trainer_applications.ErrApplicationStateConflict):
			RespondError(c, http.StatusConflict, "APPLICATION_STATE_CONFLICT", "Trainer application is not pending.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainer application document removed successfully.",
		"data":    gin.H{},
	})
}

// reviewerReason reads the optional {"reason": "..."} body of a review action.
// An absent or empty body means "no reason"; malformed content is a validation
// error.
func reviewerReason(c *gin.Context) (string, bool) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 64<<10))
	if err != nil {
		return "", false
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return "", true
	}
	var request struct {
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(body, &request); err != nil {
		return "", false
	}
	return request.Reason, true
}

func respondDocumentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, trainer_applications.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, trainer_applications.ErrDocumentNotFound):
		RespondError(c, http.StatusNotFound, "DOCUMENT_NOT_FOUND", "Trainer application document not found.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}

// respondDocumentDownload streams a validated document as a private attachment.
// The content type is always one of the allow-listed validated types, never a
// client-provided value, and nosniff plus an attachment disposition prevent the
// browser from interpreting the file as active content.
func respondDocumentDownload(c *gin.Context, document *models.TrainerApplicationDocument, content io.Reader) {
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": document.OriginalFilename})
	c.Header("Content-Type", document.ContentType)
	c.Header("Content-Disposition", disposition)
	c.Header("Content-Length", strconv.FormatInt(document.ByteSize, 10))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, no-store")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, content)
}
