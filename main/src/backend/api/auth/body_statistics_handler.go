package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/body_measurements"
	"ryze/backend/services/health_consent"
	"ryze/backend/services/muscle_recovery"
)

// BodyStatisticsHandler serves the member's muscle recovery estimate and
// body measurements (owner request, 2026-11-20). Identity comes only from
// the authentication context.
type BodyStatisticsHandler struct {
	recovery     *muscle_recovery.Service
	measurements *body_measurements.Service
	// consent gates the measurements, which are health data (v2.131.0).
	consent *health_consent.Service
}

// WithHealthConsent requires the explicit consent for health data before a
// measurement is saved.
func (h *BodyStatisticsHandler) WithHealthConsent(consent *health_consent.Service) *BodyStatisticsHandler {
	h.consent = consent
	return h
}

func NewBodyStatisticsHandler(recovery *muscle_recovery.Service, measurements *body_measurements.Service) *BodyStatisticsHandler {
	return &BodyStatisticsHandler{recovery: recovery, measurements: measurements}
}

// MuscleRecovery handles GET /me/statistics/muscles.
func (h *BodyStatisticsHandler) MuscleRecovery(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	report, err := h.recovery.Recovery(c.Request.Context(), userID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Muscle recovery retrieved successfully.", "data": report})
}

// ListMeasurements handles GET /me/measurements.
func (h *BodyStatisticsHandler) ListMeasurements(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	entries, err := h.measurements.List(c.Request.Context(), userID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Measurements retrieved successfully.", "data": gin.H{"measurements": entries}})
}

type measurementRequest struct {
	Kind       string   `json:"kind"`
	Value      *float64 `json:"value"`
	MeasuredOn string   `json:"measured_on"`
	// HealthConsent: the person gives the explicit consent for health data
	// with this measurement (needed once).
	HealthConsent bool `json:"health_consent"`
}

// SaveMeasurement handles POST /me/measurements: the same kind on the same
// day replaces the earlier value.
func (h *BodyStatisticsHandler) SaveMeasurement(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	var request measurementRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.Value == nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", []string{"value"})
		return
	}
	if h.consent != nil {
		if err := h.consent.Require(c.Request.Context(), userID, request.HealthConsent); err != nil {
			respondHealthConsentError(c, err)
			return
		}
	}
	entry, err := h.measurements.Save(c.Request.Context(), userID, request.Kind, *request.Value, request.MeasuredOn)
	switch {
	case errors.Is(err, body_measurements.ErrInvalidKind):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", []string{"kind"})
	case errors.Is(err, body_measurements.ErrInvalidValue):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", []string{"value"})
	case errors.Is(err, body_measurements.ErrInvalidDate):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", []string{"measured_on"})
	case err != nil:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	default:
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Measurement saved.", "data": entry})
	}
}

// DeleteMeasurement handles DELETE /me/measurements/:measurementID.
func (h *BodyStatisticsHandler) DeleteMeasurement(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	err = h.measurements.Delete(c.Request.Context(), userID, c.Param("measurementID"))
	switch {
	case errors.Is(err, body_measurements.ErrMeasurementMissing):
		RespondError(c, http.StatusNotFound, "MEASUREMENT_NOT_FOUND", "Measurement not found.", nil)
	case err != nil:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	default:
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Measurement deleted.", "data": nil})
	}
}
