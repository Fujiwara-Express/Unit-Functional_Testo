// Package http wires together the Gin router and all HTTP handlers.
package http

import (
	"user-service/internal/middleware"
	"user-service/internal/service/auth"
	"user-service/internal/service/user"
	"user-service/internal/token"

	"github.com/gin-gonic/gin"
)

// NewRouter creates and returns a configured Gin engine with all routes registered.
// Requirements: 1.1, 2.1, 4.1, 6.4
func NewRouter(
	authSvc auth.AuthService,
	userSvc user.UserService,
	tokenMgr token.TokenManager,
) *gin.Engine {
	r := gin.Default()

	authHandler := NewAuthHandler(authSvc)
	userHandler := NewUserHandler(userSvc)
	adminHandler := NewAdminHandler(userSvc)

	authMiddleware := middleware.Auth(tokenMgr)

	// Public routes — no authentication required
	r.POST("/users/register", authHandler.Register)
	r.POST("/auth/login", authHandler.Login)
	r.POST("/auth/refresh", authHandler.RefreshToken)
	r.POST("/auth/logout", authHandler.Logout)

	// Protected routes — valid access token required
	protected := r.Group("/", authMiddleware)
	{
		protected.GET("/users/:user_id", userHandler.GetProfile)
		protected.PUT("/users/:user_id", userHandler.UpdateProfile)
		protected.PUT("/admin/users/:user_id/status", adminHandler.SetAccountStatus)
	}

	return r
}
