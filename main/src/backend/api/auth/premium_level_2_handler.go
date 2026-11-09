package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/adminauthcontext"
	"ryze/backend/middleware/authcontext"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/premium_level_2"
)

type clientPremiumLevel2RequestResponse struct {
	ID                   string     `json:"id"`
	ProgramID            string     `json:"program_id"`
	OfferName            string     `json:"offer_name"`
	OfferDescription     string     `json:"offer_description"`
	OfferPriceMinorUnits int64      `json:"offer_price_minor_units"`
	OfferCurrency        string     `json:"offer_currency"`
	Status               string     `json:"status"`
	IntakeVersion        int        `json:"intake_version"`
	TrainerID            *string    `json:"trainer_id"`
	TrainerName          string     `json:"trainer_name"`
	TrainingProgramID    *string    `json:"training_program_id"`
	TrainingProgramName  *string    `json:"training_program_name"`
	NutritionPlanVersion *int       `json:"nutrition_plan_version"`
	NutritionPlanStatus  *string    `json:"nutrition_plan_status"`
	DeliveredAt          *time.Time `json:"delivered_at"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type staffPremiumLevel2RequestResponse struct {
	clientPremiumLevel2RequestResponse
	ClientFirstName string `json:"client_first_name"`
	ClientLastName  string `json:"client_last_name"`
	ClientEmail     string `json:"client_email"`
}

type premiumLevel2IntakeResponse struct {
	ID          string          `json:"id"`
	Version     int             `json:"version"`
	Answers     json.RawMessage `json:"answers"`
	SubmittedAt time.Time       `json:"submitted_at"`
}

type premiumLevel2EventResponse struct {
	EventType  string    `json:"event_type"`
	ActorKind  string    `json:"actor_kind"`
	FromStatus *string   `json:"from_status"`
	ToStatus   *string   `json:"to_status"`
	CreatedAt  time.Time `json:"created_at"`
}

func newClientPremiumLevel2RequestResponse(request *models.PremiumLevel2Request) clientPremiumLevel2RequestResponse {
	response := clientPremiumLevel2RequestResponse{
		ID:                   request.ID,
		ProgramID:            request.ProgramID,
		OfferName:            request.OfferName,
		OfferDescription:     request.OfferDescription,
		OfferPriceMinorUnits: request.OfferPriceMinorUnits,
		OfferCurrency:        request.OfferCurrency,
		Status:               request.Status,
		IntakeVersion:        request.IntakeVersion,
		TrainerID:            request.TrainerID,
		TrainingProgramID:    request.TrainingProgramID,
		TrainingProgramName:  request.TrainingProgramName,
		NutritionPlanVersion: request.NutritionPlanVersion,
		NutritionPlanStatus:  request.NutritionPlanStatus,
		DeliveredAt:          request.DeliveredAt,
		CreatedAt:            request.CreatedAt,
		UpdatedAt:            request.UpdatedAt,
	}
	if request.TrainerFirstName != nil || request.TrainerLastName != nil {
		first, last := "", ""
		if request.TrainerFirstName != nil {
			first = *request.TrainerFirstName
		}
		if request.TrainerLastName != nil {
			last = *request.TrainerLastName
		}
		response.TrainerName = strings.TrimSpace(first + " " + last)
	}
	return response
}

func newStaffPremiumLevel2RequestResponse(request *models.PremiumLevel2Request) staffPremiumLevel2RequestResponse {
	return staffPremiumLevel2RequestResponse{
		clientPremiumLevel2RequestResponse: newClientPremiumLevel2RequestResponse(request),
		ClientFirstName:                    request.ClientFirstName,
		ClientLastName:                     request.ClientLastName,
		ClientEmail:                        request.ClientEmail,
	}
}

func newPremiumLevel2IntakeResponse(intake *models.PremiumLevel2Intake) *premiumLevel2IntakeResponse {
	if intake == nil {
		return nil
	}
	return &premiumLevel2IntakeResponse{
		ID:          intake.ID,
		Version:     intake.Version,
		Answers:     intake.Answers,
		SubmittedAt: intake.SubmittedAt,
	}
}

func newPremiumLevel2EventResponses(events []models.PremiumLevel2RequestEvent) []premiumLevel2EventResponse {
	responses := make([]premiumLevel2EventResponse, 0, len(events))
	for _, event := range events {
		responses = append(responses, premiumLevel2EventResponse{
			EventType:  event.EventType,
			ActorKind:  event.ActorKind,
			FromStatus: event.FromStatus,
			ToStatus:   event.ToStatus,
			CreatedAt:  event.CreatedAt,
		})
	}
	return responses
}

// PremiumLevel2Handler exposes the client, trainer and administration surfaces
// of the purchased Premium Level 2 service. Identity always comes from the
// authentication or trainer context; the body never selects the actor.
type PremiumLevel2Handler struct {
	service premium_level_2.Service
}

func NewPremiumLevel2Handler(service premium_level_2.Service) *PremiumLevel2Handler {
	return &PremiumLevel2Handler{service: service}
}

func (h *PremiumLevel2Handler) GetCatalog(c *gin.Context) {
	if _, ok := requireUserID(c); !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Premium Level 2 intake catalog retrieved successfully.",
		"data":    h.service.Catalog(),
	})
}

// GetStaffCatalog exposes the same server-owned intake catalog to the
// authenticated staff surfaces so they can render answers with the labels the
// client saw. The route middleware enforces the staff role and permission.
func (h *PremiumLevel2Handler) GetStaffCatalog(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Premium Level 2 intake catalog retrieved successfully.",
		"data":    h.service.Catalog(),
	})
}

func (h *PremiumLevel2Handler) ListClientRequests(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	requests, err := h.service.ListClientRequests(c.Request.Context(), userID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	responses := make([]clientPremiumLevel2RequestResponse, 0, len(requests))
	for i := range requests {
		responses = append(responses, newClientPremiumLevel2RequestResponse(&requests[i]))
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Premium Level 2 requests retrieved successfully.",
		"data":    gin.H{"requests": responses},
	})
}

func (h *PremiumLevel2Handler) GetClientRequest(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	request, intake, plan, err := h.service.GetClientRequest(c.Request.Context(), userID, c.Param("requestID"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Premium Level 2 request retrieved successfully.",
		"data": gin.H{
			"request":        newClientPremiumLevel2RequestResponse(request),
			"intake":         newPremiumLevel2IntakeResponse(intake),
			"nutrition_plan": plan,
		},
	})
}

type premiumLevel2IntakeRequest struct {
	Answers json.RawMessage `json:"answers"`
}

func (h *PremiumLevel2Handler) SubmitIntake(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var body premiumLevel2IntakeRequest
	if err := c.ShouldBindJSON(&body); err != nil || len(body.Answers) == 0 {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", []string{"answers is required"})
		return
	}
	request, intake, err := h.service.SubmitIntake(c.Request.Context(), userID, c.Param("requestID"), body.Answers)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Premium Level 2 intake saved successfully.",
		"data": gin.H{
			"request": newClientPremiumLevel2RequestResponse(request),
			"intake":  newPremiumLevel2IntakeResponse(intake),
		},
	})
}

func (h *PremiumLevel2Handler) GetDeliveredNutrition(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	request, plan, err := h.service.GetDeliveredNutrition(c.Request.Context(), userID, c.Param("requestID"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Premium Level 2 nutrition retrieved successfully.",
		"data": gin.H{
			"request": newClientPremiumLevel2RequestResponse(request),
			"plan":    plan,
		},
	})
}

func (h *PremiumLevel2Handler) ListAdminRequests(c *gin.Context) {
	page, limit := premiumLevel2Pagination(c)
	filter := repositories.PremiumLevel2Filter{
		Status: strings.TrimSpace(c.Query("status")),
		Query:  strings.TrimSpace(c.Query("q")),
	}
	requests, total, err := h.service.ListAdminRequests(c.Request.Context(), filter, page, limit)
	if err != nil {
		h.respondError(c, err)
		return
	}
	responses := make([]staffPremiumLevel2RequestResponse, 0, len(requests))
	for i := range requests {
		responses = append(responses, newStaffPremiumLevel2RequestResponse(&requests[i]))
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Premium Level 2 requests retrieved successfully.",
		"data": gin.H{
			"requests":   responses,
			"pagination": gin.H{"page": page, "limit": limit, "total": total},
		},
	})
}

func (h *PremiumLevel2Handler) GetAdminRequest(c *gin.Context) {
	request, intake, plan, events, err := h.service.GetAdminRequest(c.Request.Context(), c.Param("requestID"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Premium Level 2 request retrieved successfully.",
		"data": gin.H{
			"request":        newStaffPremiumLevel2RequestResponse(request),
			"intake":         newPremiumLevel2IntakeResponse(intake),
			"nutrition_plan": plan,
			"events":         newPremiumLevel2EventResponses(events),
		},
	})
}

type assignPremiumLevel2TrainerRequest struct {
	TrainerID string `json:"trainer_id"`
}

func (h *PremiumLevel2Handler) AssignTrainer(c *gin.Context) {
	adminID, err := adminauthcontext.AdminIdentityFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	var body assignPremiumLevel2TrainerRequest
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.TrainerID) == "" {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", []string{"trainer_id is required"})
		return
	}
	request, err := h.service.AssignTrainer(c.Request.Context(), c.Param("requestID"), strings.TrimSpace(body.TrainerID), adminID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainer assigned successfully.",
		"data":    newStaffPremiumLevel2RequestResponse(request),
	})
}

func (h *PremiumLevel2Handler) ListTrainerRequests(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}
	page, limit := premiumLevel2Pagination(c)
	filter := repositories.PremiumLevel2Filter{Status: strings.TrimSpace(c.Query("status"))}
	requests, total, err := h.service.ListTrainerRequests(c.Request.Context(), trainerID, filter, page, limit)
	if err != nil {
		h.respondError(c, err)
		return
	}
	responses := make([]staffPremiumLevel2RequestResponse, 0, len(requests))
	for i := range requests {
		responses = append(responses, newStaffPremiumLevel2RequestResponse(&requests[i]))
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Premium Level 2 requests retrieved successfully.",
		"data": gin.H{
			"requests":   responses,
			"pagination": gin.H{"page": page, "limit": limit, "total": total},
		},
	})
}

func (h *PremiumLevel2Handler) GetTrainerRequest(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}
	request, intake, plan, events, err := h.service.GetTrainerRequest(c.Request.Context(), trainerID, c.Param("requestID"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Premium Level 2 request retrieved successfully.",
		"data": gin.H{
			"request":        newStaffPremiumLevel2RequestResponse(request),
			"intake":         newPremiumLevel2IntakeResponse(intake),
			"nutrition_plan": plan,
			"events":         newPremiumLevel2EventResponses(events),
		},
	})
}

type attachPremiumLevel2ProgramRequest struct {
	ProgramID string `json:"program_id"`
}

func (h *PremiumLevel2Handler) AttachTrainingProgram(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}
	var body attachPremiumLevel2ProgramRequest
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.ProgramID) == "" {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", []string{"program_id is required"})
		return
	}
	request, err := h.service.AttachTrainingProgram(c.Request.Context(), trainerID, c.Param("requestID"), strings.TrimSpace(body.ProgramID))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Training programme attached successfully.",
		"data":    newStaffPremiumLevel2RequestResponse(request),
	})
}

func (h *PremiumLevel2Handler) SaveNutrition(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}
	var body premium_level_2.NutritionInput
	if err := c.ShouldBindJSON(&body); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	plan, err := h.service.SaveNutrition(c.Request.Context(), trainerID, c.Param("requestID"), body)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Nutrition plan saved successfully.",
		"data":    plan,
	})
}

func (h *PremiumLevel2Handler) Deliver(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}
	request, err := h.service.Deliver(c.Request.Context(), trainerID, c.Param("requestID"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Premium Level 2 request delivered successfully.",
		"data":    newStaffPremiumLevel2RequestResponse(request),
	})
}

func (h *PremiumLevel2Handler) respondError(c *gin.Context, err error) {
	var validation *premium_level_2.ValidationError
	switch {
	case errors.As(err, &validation):
		details := make([]string, 0, len(validation.Fields))
		keys := make([]string, 0, len(validation.Fields))
		for key := range validation.Fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			details = append(details, key+": "+validation.Fields[key])
		}
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", details)
	case errors.Is(err, premium_level_2.ErrRequestNotFound):
		RespondError(c, http.StatusNotFound, "PREMIUM_LEVEL_2_NOT_FOUND", "Premium Level 2 request not found.", nil)
	case errors.Is(err, premium_level_2.ErrNutritionMissing):
		RespondError(c, http.StatusNotFound, "NUTRITION_NOT_FOUND", "No nutrition plan is available yet.", nil)
	case errors.Is(err, premium_level_2.ErrInvalidState):
		RespondError(c, http.StatusConflict, "PREMIUM_LEVEL_2_NOT_EDITABLE", "This request can no longer be changed.", nil)
	case errors.Is(err, premium_level_2.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}

func requireUserID(c *gin.Context) (string, bool) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return "", false
	}
	return userID, true
}

func premiumLevel2Pagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > premium_level_2.MaxPageSize {
		limit = premium_level_2.MaxPageSize
	}
	return page, limit
}
