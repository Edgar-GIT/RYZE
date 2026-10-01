package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/nutrition_assignment"
)

// NutritionHandler exposes the authenticated nutrition plan of a purchased
// Premium Level 1 program.
//
// The plan is private to the account that bought the program: the entitlement is
// the gate, and a missing, unowned or unavailable program is reported
// identically so the endpoint never reveals who bought what. The internal
// failure reason of an assignment is never included in a response.
type NutritionHandler struct {
	assignments nutrition_assignment.Service
}

// NewNutritionHandler wires the nutrition assignment use cases.
func NewNutritionHandler(assignments nutrition_assignment.Service) *NutritionHandler {
	return &NutritionHandler{assignments: assignments}
}

// GetStatus returns the current state of the caller's nutrition plan, including
// the plan itself once generation completed. A plan generated from a superseded
// intake is reported as out of date rather than served as current.
func (h *NutritionHandler) GetStatus(c *gin.Context) {
	userID, programID, ok := h.resolve(c)
	if !ok {
		return
	}

	status, err := h.assignments.GetStatus(c.Request.Context(), userID, programID)
	if err != nil {
		h.respondError(c, err)
		return
	}

	h.respondStatus(c, status)
}

// Generate runs the deterministic generation step for the caller's plan.
//
// The client never marks a plan as generated: generation is a server operation
// over the stored, validated intake. The call is idempotent, so a client that
// retries after a network failure converges on the same single plan.
func (h *NutritionHandler) Generate(c *gin.Context) {
	userID, programID, ok := h.resolve(c)
	if !ok {
		return
	}

	status, err := h.assignments.Run(c.Request.Context(), userID, programID)
	if err != nil {
		h.respondError(c, err)
		return
	}

	h.respondStatus(c, status)
}

// respondStatus renders the assignment state. Both endpoints answer with the same
// body so a client can treat generation as a refresh of the status it already
// knows how to read.
//
// Plan is emitted as an explicit null while no plan exists: the field's presence is
// part of the contract, and omitting it would leave a client guessing whether the
// plan is absent or the response is malformed.
func (h *NutritionHandler) respondStatus(c *gin.Context, status *nutrition_assignment.Status) {
	c.JSON(http.StatusOK, gin.H{
		"success":               true,
		"program_id":            status.ProgramID,
		"status":                status.Status,
		"version":               status.Version,
		"questionnaire_version": status.QuestionnaireVersion,
		"out_of_date":           status.OutOfDate,
		"plan":                  status.Plan,
	})
}

// resolve reads the authenticated caller and the program identifier from the
// request. The user id is taken exclusively from the authentication context, so
// a client cannot read or generate a plan for another account.
func (h *NutritionHandler) resolve(c *gin.Context) (string, string, bool) {
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

// respondError maps a service error onto the API contract.
func (h *NutritionHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, nutrition_assignment.ErrAssignmentNotFound):
		// A missing plan, an unowned program and a product without a plan are
		// reported identically, so this endpoint is not a purchase oracle.
		RespondError(c, http.StatusNotFound, "NUTRITION_NOT_FOUND", "No nutrition plan is available for this program.", nil)
	case errors.Is(err, nutrition_assignment.ErrInvalidInput):
		RespondError(c, http.StatusConflict, "NUTRITION_NOT_READY", "The nutrition plan is not ready. Please try again.", nil)
	case errors.Is(err, nutrition_assignment.ErrNoEligibleFood):
		// The stored restrictions cannot be satisfied by the catalog. That is a
		// fact about the intake, not a server fault, and it is not retryable, so
		// it gets its own code instead of being hidden behind a 500. The wrapped
		// detail names catalog internals and is deliberately not forwarded.
		RespondError(c, http.StatusUnprocessableEntity, "NUTRITION_RESTRICTIONS_UNSATISFIABLE", "Your dietary restrictions cannot be satisfied for this program yet.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "The request could not be completed.", nil)
	}
}
