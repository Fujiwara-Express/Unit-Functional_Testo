// Package middleware provides Gin HTTP middleware for the user service.
package middleware

import (
	"net/http"
	"strings"

	"user-service/internal/token"

	"github.com/gin-gonic/gin"
)

// Context keys for values injected by the auth middleware.
const (
	ContextKeyUserID = "user_id"
	ContextKeyRole   = "role"
)

// Auth returns a Gin middleware that validates the Authorization: Bearer <token>
// header, injects user_id and role into the context, and aborts with 401 on failure.
func Auth(tm token.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "AUTHENTICATION_ERROR",
					"message": "authorization header is required",
				},
			})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "AUTHENTICATION_ERROR",
					"message": "authorization header must be in the format: Bearer <token>",
				},
			})
			return
		}

		claims, err := tm.ValidateAccessToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "AUTHENTICATION_ERROR",
					"message": "invalid or expired token",
				},
			})
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyRole, claims.Role)
		c.Next()
	}
}
