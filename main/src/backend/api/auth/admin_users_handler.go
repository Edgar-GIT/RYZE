package auth

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/admin_users"
)

// AdminUserHandler exposes administrator user-management operations. It never
// performs authentication or authorization itself: those are enforced by the
// AdminAuthenticate and RequireAdminRole middleware mounted on the routes.
type AdminUserHandler struct {
	service admin_users.AdminUserService
}

func NewAdminUserHandler(svc admin_users.AdminUserService) *AdminUserHandler {
	return &AdminUserHandler{service: svc}
}

// ListUsers returns one page of active users with pagination metadata. Only
// safe public user information is exposed.
func (h *AdminUserHandler) ListUsers(c *gin.Context) {
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

	result, err := h.service.ListUsers(c.Request.Context(), page, limit)
	if err != nil {
		switch {
		case errors.Is(err, admin_users.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	users := make([]userResponse, 0, len(result.Users))
	for i := range result.Users {
		users = append(users, newUserResponse(&result.Users[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Users retrieved successfully.",
		"data": gin.H{
			"users": users,
			"pagination": gin.H{
				"page":        result.Page,
				"limit":       result.Limit,
				"total":       result.Total,
				"total_pages": totalPages(result.Total, result.Limit),
			},
		},
	})
}

// GetUser returns one active user by UUID. Soft-deleted users are never
// returned.
func (h *AdminUserHandler) GetUser(c *gin.Context) {
	user, err := h.service.GetUser(c.Request.Context(), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, admin_users.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, admin_users.ErrUserNotFound):
			RespondError(c, http.StatusNotFound, "USER_NOT_FOUND", "User not found.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User retrieved successfully.",
		"data":    newUserResponse(user),
	})
}

// SoftDeleteUser soft-deletes an active user by UUID. The row is preserved and
// the user's existing sessions are invalidated immediately.
func (h *AdminUserHandler) SoftDeleteUser(c *gin.Context) {
	if err := h.service.SoftDeleteUser(c.Request.Context(), c.Param("id")); err != nil {
		switch {
		case errors.Is(err, admin_users.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, admin_users.ErrUserNotFound):
			RespondError(c, http.StatusNotFound, "USER_NOT_FOUND", "User not found.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User disabled successfully.",
		"data":    gin.H{},
	})
}

// queryInt parses an integer query parameter, returning the fallback when the
// parameter is absent. Any non-integer value is reported as an error.
func queryInt(c *gin.Context, key string, fallback int) (int, error) {
	value := c.Query(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

// totalPages computes the number of pages required to hold total records at the
// given page size.
func totalPages(total int64, limit int) int {
	if limit <= 0 {
		return 0
	}
	return int((total + int64(limit) - 1) / int64(limit))
}
