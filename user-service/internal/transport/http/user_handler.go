package http

import (
	"net/http"

	"user-service/internal/middleware"
	"user-service/internal/service/user"

	"github.com/gin-gonic/gin"
)

// UserHandler handles user profile HTTP requests.
type UserHandler struct {
	svc user.UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(svc user.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// GetProfile handles GET /users/:user_id
// Requirements: 4.1
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.Param("user_id")

	profile, err := h.svc.GetProfile(c.Request.Context(), userID)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateProfile handles PUT /users/:user_id
// Requirements: 4.2, 4.3, 4.4
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := c.Param("user_id")

	// Ensure the authenticated user can only update their own profile.
	callerID, _ := c.Get(middleware.ContextKeyUserID)
	if callerID != userID {
		c.JSON(http.StatusForbidden, errorResponse("AUTHORIZATION_ERROR", "cannot update another user's profile"))
		return
	}

	var req user.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	resp, err := h.svc.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
