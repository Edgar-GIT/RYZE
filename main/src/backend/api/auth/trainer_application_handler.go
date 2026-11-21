package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/adminauthcontext"
	"ryze/backend/middleware/authcontext"
	"ryze/backend/models"
	"ryze/backend/services/trainer_applications"
)

// applicationResponse only exposes safe application information.
type applicationResponse struct {
	ID     string       `json:"id"`
	Status string       `json:"status"`
	UserID string       `json:"user_id"`
	User   userResponse `json:"user"`
	// Username is the applicant's @username (their account reference).
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// The applicant's details (complete application form, migration 000050);
	// null on older applications and administrator promotions.
	FullName        *string         `json:"full_name"`
	BirthDate       *string         `json:"birth_date"`
	City            *string         `json:"city"`
	Profession      *string         `json:"profession"`
	YearsExperience *int            `json:"years_experience"`
	WorkHistory     json.RawMessage `json:"work_history"`
	Education       json.RawMessage `json:"education"`
	About           *string         `json:"about"`
}

func newApplicationResponse(application *models.TrainerApplication) applicationResponse {
	response := applicationResponse{
		ID:              application.ID,
		Status:          application.Status,
		UserID:          application.UserID,
		User:            newUserResponse(&application.User),
		Username:        application.User.Username,
		CreatedAt:       application.CreatedAt,
		UpdatedAt:       application.UpdatedAt,
		FullName:        application.FullName,
		City:            application.City,
		Profession:      application.Profession,
		YearsExperience: application.YearsExperience,
		WorkHistory:     jsonOrEmptyList(application.WorkHistory),
		Education:       jsonOrEmptyList(application.Education),
		About:           application.About,
	}
	if application.BirthDate != nil {
		date := application.BirthDate.Format("2006-01-02")
		response.BirthDate = &date
	}
	return response
}

func jsonOrEmptyList(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("[]")
	}
	return raw
}

// reviewDocumentResponse is a document as the reviewers see it: the metadata
// plus the automatic screening result. Applicants never see the signals.
type reviewDocumentResponse struct {
	applicationDocumentResponse
	ScreeningStatus  string          `json:"screening_status"`
	ScreeningSignals json.RawMessage `json:"screening_signals"`
}

func newReviewDocumentResponse(document *models.TrainerApplicationDocument) reviewDocumentResponse {
	return reviewDocumentResponse{
		applicationDocumentResponse: newApplicationDocumentResponse(document),
		ScreeningStatus:             document.ScreeningStatus,
		ScreeningSignals:            jsonOrEmptyList(document.ScreeningSignals),
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

	responses := make([]reviewDocumentResponse, 0, len(documents))
	for i := range documents {
		responses = append(responses, newReviewDocumentResponse(&documents[i]))
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
	// maxDocumentBytes bounds each attached file read for the screening.
	maxDocumentBytes int64
}

func NewTrainerApplicationHandler(svc trainer_applications.Service) *TrainerApplicationHandler {
	return &TrainerApplicationHandler{service: svc, maxDocumentBytes: 10 << 20}
}

// WithMaxDocumentBytes sets the per-file bound (TRAINER_EVIDENCE_MAX_BYTES).
func (h *TrainerApplicationHandler) WithMaxDocumentBytes(limit int64) *TrainerApplicationHandler {
	if limit > 0 {
		h.maxDocumentBytes = limit
	}
	return h
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

	// The complete application (owner request, 2026-11-20): the form fields
	// and the evidence files in one multipart submission.
	form, documents, ok := h.readApplication(c)
	if !ok {
		return
	}
	application, err := h.service.SubmitApplication(c.Request.Context(), userID, form, documents)
	if err != nil {
		var formErr *trainer_applications.FormError
		var rejected *trainer_applications.DocumentsRejectedError
		switch {
		case errors.As(err, &formErr):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Some fields are missing or invalid.", formErr.Fields)
		case errors.As(err, &rejected):
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"success": false,
				"message": "Some documents were refused.",
				"error": gin.H{
					"code":       "DOCUMENTS_REJECTED",
					"details":    []string{},
					"rejections": rejected.Rejections,
				},
			})
		case errors.Is(err, trainer_applications.ErrMissingDocuments):
			RespondError(c, http.StatusBadRequest, "DOCUMENTS_REQUIRED", "Attach at least one certification and one identification document.", nil)
		case errors.Is(err, trainer_applications.ErrDocumentUnsupportedType):
			RespondError(c, http.StatusBadRequest, "DOCUMENT_TYPE_NOT_ALLOWED", "Documents must be PDF, JPEG or PNG files.", nil)
		case errors.Is(err, trainer_applications.ErrDocumentTooLarge):
			RespondError(c, http.StatusRequestEntityTooLarge, "DOCUMENT_TOO_LARGE", "A document is too large.", nil)
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
		var rejected *trainer_applications.DocumentsRejectedError
		switch {
		case errors.Is(err, trainer_applications.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, trainer_applications.ErrApplicationNotFound):
			RespondError(c, http.StatusNotFound, "APPLICATION_NOT_FOUND", "Trainer application not found.", nil)
		case errors.Is(err, trainer_applications.ErrApplicationStateConflict):
			RespondError(c, http.StatusConflict, "APPLICATION_STATE_CONFLICT", "Trainer application is not pending.", nil)
		case errors.As(err, &rejected):
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"success": false,
				"message": "The document was refused.",
				"error":   gin.H{"code": "DOCUMENTS_REJECTED", "details": []string{}, "rejections": rejected.Rejections},
			})
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

// applicationDocumentFields are the multipart file fields, one per kind.
var applicationDocumentFields = []string{"certification", "identification", "insurance", "other"}

// readApplication reads the multipart application: the form fields and the
// files (each read in full, bounded, for the screening). It answers the
// request itself when the body is unusable.
func (h *TrainerApplicationHandler) readApplication(c *gin.Context) (trainer_applications.ApplicationForm, []trainer_applications.SubmittedDocument, bool) {
	var form trainer_applications.ApplicationForm
	multipartForm, err := c.MultipartForm()
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Send the application form with its documents.", nil)
		return form, nil, false
	}
	value := func(key string) string {
		if values := multipartForm.Value[key]; len(values) > 0 {
			return values[0]
		}
		return ""
	}
	form.FullName = value("full_name")
	form.BirthDate = value("birth_date")
	form.City = value("city")
	form.Profession = value("profession")
	form.About = value("about")
	years, err := strconv.Atoi(strings.TrimSpace(value("years_experience")))
	if err != nil {
		years = -1
	}
	form.YearsExperience = years
	if raw := value("work_history"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &form.WorkHistory); err != nil {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Some fields are missing or invalid.", []string{"work_history"})
			return form, nil, false
		}
	}
	if raw := value("education"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &form.Education); err != nil {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Some fields are missing or invalid.", []string{"education"})
			return form, nil, false
		}
	}

	var documents []trainer_applications.SubmittedDocument
	for _, kind := range applicationDocumentFields {
		for _, header := range multipartForm.File[kind] {
			if len(documents) >= trainer_applications.MaxSubmittedDocuments {
				RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Too many documents.", []string{"documents"})
				return form, nil, false
			}
			if header.Size > h.maxDocumentBytes {
				RespondError(c, http.StatusRequestEntityTooLarge, "DOCUMENT_TOO_LARGE", "A document is too large.", nil)
				return form, nil, false
			}
			file, err := header.Open()
			if err != nil {
				RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "A document could not be read.", nil)
				return form, nil, false
			}
			data, err := io.ReadAll(io.LimitReader(file, h.maxDocumentBytes+1))
			file.Close()
			if err != nil || int64(len(data)) > h.maxDocumentBytes {
				RespondError(c, http.StatusRequestEntityTooLarge, "DOCUMENT_TOO_LARGE", "A document is too large.", nil)
				return form, nil, false
			}
			documents = append(documents, trainer_applications.SubmittedDocument{Kind: kind, Filename: header.Filename, Data: data})
		}
	}
	return form, documents, true
}

type promoteTrainerRequest struct {
	Username string `json:"username"`
	Reason   string `json:"reason"`
}

// PromoteUser makes the account with the given @username a trainer directly
// (administrators who manage trainer applications).
func (h *AdminTrainerApplicationHandler) PromoteUser(c *gin.Context) {
	adminID, err := adminauthcontext.AdminIdentityFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	var req promoteTrainerRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Username) == "" || len(req.Reason) > 500 {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	application, err := h.service.PromoteUser(c.Request.Context(), req.Username, adminID, req.Reason)
	if err != nil {
		switch {
		case errors.Is(err, trainer_applications.ErrUserNotFound):
			RespondError(c, http.StatusNotFound, "USER_NOT_FOUND", "No active account has this username.", nil)
		case errors.Is(err, trainer_applications.ErrAlreadyTrainer):
			RespondError(c, http.StatusConflict, "USER_ALREADY_TRAINER", "User is already a trainer.", nil)
		case errors.Is(err, trainer_applications.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "User promoted to trainer.",
		"data":    newApplicationResponse(application),
	})
}
