package auth

import (
	"errors"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/nutrition_questionnaire"
	"ryze/backend/services/questionnaires"
)

// questionnaireResponse is the client-safe view of the intake state for one
// program. It deliberately carries no answer content: the client only needs to
// know whether it must still answer, and the answers themselves are never read
// back from the server.
type questionnaireResponse struct {
	Success          bool   `json:"success"`
	ProgramID        string `json:"program_id"`
	Required         bool   `json:"required"`
	Submitted        bool   `json:"submitted"`
	Version          int    `json:"version"`
	SchemaVersion    int    `json:"schema_version"`
	MinSchemaVersion int    `json:"min_schema_version"`
	SubmittedAt      string `json:"submitted_at,omitempty"`
}

// questionnaireQuestionsResponse serves the server-owned question catalog. The
// client renders the intake from this contract, so the form and the validation
// rules can never drift apart.
type questionnaireQuestionsResponse struct {
	Success   bool                               `json:"success"`
	Version   int                                `json:"version"`
	Questions []nutrition_questionnaire.Question `json:"questions"`
}

// QuestionnaireHandler exposes the authenticated Premium Level 1 intake. Every
// operation is scoped to the caller resolved by the authentication middleware;
// no handler accepts a user id from the client.
type QuestionnaireHandler struct {
	questionnaires questionnaires.Service
}

// NewQuestionnaireHandler wires the intake use cases.
func NewQuestionnaireHandler(questionnaires questionnaires.Service) *QuestionnaireHandler {
	return &QuestionnaireHandler{questionnaires: questionnaires}
}

// GetRequirement reports whether the program needs an intake and, when the
// caller already submitted one, which revision is stored.
func (h *QuestionnaireHandler) GetRequirement(c *gin.Context) {
	userID, programID, ok := h.resolve(c)
	if !ok {
		return
	}

	requirement, err := h.questionnaires.GetRequirement(c.Request.Context(), userID, programID)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, newQuestionnaireResponse(requirement))
}

// GetQuestions returns the server-owned question catalog. It requires
// authentication so the intake contract is never served anonymously.
func (h *QuestionnaireHandler) GetQuestions(c *gin.Context) {
	if _, err := authcontext.UserIDFromContext(c); err != nil {
		RespondError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}
	c.JSON(http.StatusOK, questionnaireQuestionsResponse{
		Success:   true,
		Version:   nutrition_questionnaire.SchemaVersion,
		Questions: nutrition_questionnaire.Catalog(),
	})
}

// Submit validates and stores the intake for the authenticated caller. The whole
// submission is accepted or rejected as a unit, and per-field rejection reasons
// are returned without echoing any submitted value.
func (h *QuestionnaireHandler) Submit(c *gin.Context) {
	userID, programID, ok := h.resolve(c)
	if !ok {
		return
	}

	var request struct {
		Answers nutrition_questionnaire.Answers `json:"answers"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}

	requirement, err := h.questionnaires.Submit(c.Request.Context(), userID, programID, request.Answers)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, newQuestionnaireResponse(requirement))
}

// resolve reads the authenticated caller and the program identifier from the
// request. The user id is taken exclusively from the authentication context, so
// a client cannot act on behalf of another account.
func (h *QuestionnaireHandler) resolve(c *gin.Context) (string, string, bool) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return "", "", false
	}

	programID := c.Param("programID")
	if programID == "" {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return "", "", false
	}
	return userID, programID, true
}

// respondError maps a service error onto the API contract. Internal failure
// details and the stored failure reason of a nutrition assignment are never
// included.
func (h *QuestionnaireHandler) respondError(c *gin.Context, err error) {
	var fieldErrors *questionnaires.FieldValidationError
	switch {
	case errors.As(err, &fieldErrors):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", flattenFieldReasons(fieldErrors.FieldReasons()))
	case errors.Is(err, questionnaires.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, questionnaires.ErrLocked):
		// A conflict, not a validation error: the request was well formed, the
		// resource is simply immutable now that the purchase completed.
		RespondError(c, http.StatusConflict, "QUESTIONNAIRE_LOCKED", "This questionnaire can no longer be changed because the purchase is already complete.", nil)
	case errors.Is(err, questionnaires.ErrProgramNotFound):
		RespondError(c, http.StatusNotFound, "PROGRAM_NOT_FOUND", "Program not found.", nil)
	case errors.Is(err, nutrition_questionnaire.ErrNotSubmitted):
		RespondError(c, http.StatusNotFound, "QUESTIONNAIRE_NOT_SUBMITTED", "No questionnaire has been submitted for this program.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "The request could not be completed.", nil)
	}
}

// newQuestionnaireResponse maps the intake state onto the response contract.
func newQuestionnaireResponse(requirement *questionnaires.Requirement) questionnaireResponse {
	response := questionnaireResponse{
		Success:          true,
		ProgramID:        requirement.ProgramID,
		Required:         requirement.Required,
		Submitted:        requirement.Submitted,
		Version:          requirement.Version,
		SchemaVersion:    requirement.SchemaVersion,
		MinSchemaVersion: requirement.MinSchemaVersion,
	}
	if requirement.SubmittedAt != nil {
		response.SubmittedAt = requirement.SubmittedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return response
}

// flattenFieldReasons converts per-field rejection reasons into the flat detail
// list the error contract uses, in a stable order. Only field names and reasons
// are included; the value that failed is never echoed back.
func flattenFieldReasons(reasons map[string]string) []string {
	details := make([]string, 0, len(reasons))
	for field, reason := range reasons {
		details = append(details, field+": "+reason)
	}
	sort.Strings(details)
	return details
}
