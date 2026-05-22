// Package http contains the HTTP handlers for the user service.
package http

import (
	"errors"
	"net/http"

	"user-service/internal/domain"
	"user-service/internal/service/auth"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	svc auth.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(svc auth.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Register handles POST /users/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req auth.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	resp, err := h.svc.Register(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// Login handles POST /auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req auth.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	resp, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// RefreshToken handles POST /auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	resp, err := h.svc.RefreshToken(c.Request.Context(), body.RefreshToken)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Logout handles POST /auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	if err := h.svc.Logout(c.Request.Context(), body.RefreshToken); err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "LOGGED_OUT"})
}

// writeServiceError maps domain sentinel errors to appropriate HTTP responses.
func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrEmailConflict):
		c.JSON(http.StatusConflict, errorResponse("CONFLICT", err.Error()))
	case errors.Is(err, domain.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, errorResponse("AUTHENTICATION_ERROR", "invalid credentials or token"))
	case errors.Is(err, domain.ErrForbidden):
		c.JSON(http.StatusForbidden, errorResponse("AUTHORIZATION_ERROR", "insufficient permissions"))
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "resource not found"))
	case isValidationError(err):
		c.JSON(http.StatusBadRequest, errorResponse("VALIDATION_ERROR", err.Error()))
	default:
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "an internal error occurred"))
	}
}

// isValidationError returns true for domain validation errors.
func isValidationError(err error) bool {
	return errors.Is(err, domain.ErrInvalidEmail) ||
		errors.Is(err, domain.ErrPasswordTooShort) ||
		errors.Is(err, domain.ErrInvalidPhone) ||
		errors.Is(err, domain.ErrInvalidRole)
}

// errorResponse builds the standard error envelope.
func errorResponse(code, message string) gin.H {
	return gin.H{"error": gin.H{"code": code, "message": message}}
}
