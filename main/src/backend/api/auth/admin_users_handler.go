package auth

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/admin_users"
	"ryze/backend/services/password"
)

// AdminUserHandler exposes administrator user-management operations. It never
// performs authentication or authorization itself: those are enforced by the
// AdminAuthenticate and RequireAdminPermission middleware mounted on the
// routes. Read endpoints run under users.read (both roles); lifecycle
// endpoints run under users.manage (Technical Administrator only).
type AdminUserHandler struct {
	service admin_users.AdminUserService
}

func NewAdminUserHandler(svc admin_users.AdminUserService) *AdminUserHandler {
	return &AdminUserHandler{service: svc}
}

// createAdminUserRequest is the request DTO for creating a user. It never
// accepts id, deleted_at or session_version: those are controlled exclusively
// by the backend.
type createAdminUserRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// updateAdminUserRequest is the request DTO for updating a user. Only safe
// whitelisted fields are accepted; password changes use the dedicated
// administrative password-reset operation.
type updateAdminUserRequest struct {
	Email     *string `json:"email"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
}

// adminPasswordResetRequest is the request DTO for the administrative password
// reset. It requires no current password: the operation is performed by an
// authorized administrator.
type adminPasswordResetRequest struct {
	NewPassword string `json:"new_password"`
}

// ListUsers returns one page of active users with pagination metadata. Only
// safe public user information is exposed.
func (h *AdminUserHandler) ListUsers(c *gin.Context) {
	result, ok := h.listUsers(c, false)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Users retrieved successfully.",
		"data":    listUsersData(result),
	})
}

// ListDeletedUsers returns one page of soft-deleted users with pagination
// metadata. It is a clearly separated view for lifecycle management; active
// and deleted users are never mixed in a single list. The same safe
// representation is used and deleted_at is never exposed.
func (h *AdminUserHandler) ListDeletedUsers(c *gin.Context) {
	result, ok := h.listUsers(c, true)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Deleted users retrieved successfully.",
		"data":    listUsersData(result),
	})
}

func (h *AdminUserHandler) listUsers(c *gin.Context, deleted bool) (admin_users.ListUsersResult, bool) {
	page, err := queryInt(c, "page", 1)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return admin_users.ListUsersResult{}, false
	}
	limit, err := queryInt(c, "limit", 20)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return admin_users.ListUsersResult{}, false
	}

	var result admin_users.ListUsersResult
	if deleted {
		result, err = h.service.ListDeletedUsers(c.Request.Context(), page, limit)
	} else {
		result, err = h.service.ListUsers(c.Request.Context(), page, limit)
	}
	if err != nil {
		switch {
		case errors.Is(err, admin_users.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return admin_users.ListUsersResult{}, false
	}
	return result, true
}

func listUsersData(result admin_users.ListUsersResult) gin.H {
	users := make([]userResponse, 0, len(result.Users))
	for i := range result.Users {
		users = append(users, newUserResponse(&result.Users[i]))
	}
	return gin.H{
		"users": users,
		"pagination": gin.H{
			"page":        result.Page,
			"limit":       result.Limit,
			"total":       result.Total,
			"total_pages": totalPages(result.Total, result.Limit),
		},
	}
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

// CreateUser creates a new active user, or reactivates a soft-deleted identity
// when the email belongs to a previously deleted account.
func (h *AdminUserHandler) CreateUser(c *gin.Context) {
	var req createAdminUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	user, err := h.service.CreateUser(c.Request.Context(), admin_users.CreateUserInput{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		switch {
		case errors.Is(err, admin_users.ErrInvalidInput), errors.Is(err, password.ErrEmptyPassword):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, admin_users.ErrDuplicateEmail):
			RespondError(c, http.StatusConflict, "EMAIL_ALREADY_REGISTERED", "Email already in use.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "User created successfully.",
		"data":    newUserResponse(user),
	})
}

// UpdateUser updates the whitelisted safe fields of an active user.
func (h *AdminUserHandler) UpdateUser(c *gin.Context) {
	var req updateAdminUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	user, err := h.service.UpdateUser(c.Request.Context(), c.Param("id"), admin_users.UpdateUserInput{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		switch {
		case errors.Is(err, admin_users.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, admin_users.ErrUserNotFound):
			RespondError(c, http.StatusNotFound, "USER_NOT_FOUND", "User not found.", nil)
		case errors.Is(err, admin_users.ErrDuplicateEmail):
			RespondError(c, http.StatusConflict, "EMAIL_ALREADY_REGISTERED", "Email already in use.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User updated successfully.",
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

// ReactivateUser restores a soft-deleted user: the account becomes active again
// and can authenticate again while every pre-deletion session stays invalid.
func (h *AdminUserHandler) ReactivateUser(c *gin.Context) {
	user, err := h.service.ReactivateUser(c.Request.Context(), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, admin_users.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, admin_users.ErrUserNotFound):
			RespondError(c, http.StatusNotFound, "USER_NOT_FOUND", "User not found.", nil)
		case errors.Is(err, admin_users.ErrAlreadyActive):
			RespondError(c, http.StatusConflict, "USER_ALREADY_ACTIVE", "User is already active.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User reactivated successfully.",
		"data":    newUserResponse(user),
	})
}

// ResetUserPassword replaces an active user's password (without requiring the
// current password) and invalidates every previously issued session.
func (h *AdminUserHandler) ResetUserPassword(c *gin.Context) {
	var req adminPasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	if err := h.service.ResetUserPassword(c.Request.Context(), c.Param("id"), req.NewPassword); err != nil {
		switch {
		case errors.Is(err, admin_users.ErrInvalidInput), errors.Is(err, password.ErrEmptyPassword):
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
		"message": "Password reset successfully.",
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
