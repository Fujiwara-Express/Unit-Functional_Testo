package http

import (
	"net/http"

	"user-service/internal/middleware"
	"user-service/internal/service/user"

	"github.com/gin-gonic/gin"
)

// AdminHandler handles admin-only HTTP requests.
type AdminHandler struct {
	svc user.UserService
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(svc user.UserService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

// SetAccountStatus handles PUT /admin/users/:user_id/status
// Requirements: 6.1, 6.2, 6.4
func (h *AdminHandler) SetAccountStatus(c *gin.Context) {
	targetID := c.Param("user_id")
	callerRole, _ := c.Get(middleware.ContextKeyRole)

	var body struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	role, _ := callerRole.(string)
	if err := h.svc.SetAccountStatus(c.Request.Context(), role, targetID, body.IsActive); err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "UPDATED"})
}
