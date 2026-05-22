// Package middleware provides Gin middleware for the routing service.
package middleware

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"routing-service/internal/apperrors"
	"routing-service/internal/clients"
	"routing-service/internal/repositories"
)

// ErrorHandler is a Gin middleware that catches errors set via c.Error() and
// maps them to appropriate HTTP responses.
//
// Error mapping (Requirements: 1.5, 2.3, 2.4, 3.5, 3.6, 3.7, 4.7, 6.3):
//   - *apperrors.ValidationError      → 400
//   - *apperrors.NotFoundError        → 404
//   - repositories.ErrNotFound        → 404
//   - *apperrors.DuplicateError       → 409
//   - repositories.ErrDuplicate       → 409
//   - *apperrors.UpstreamUnavailable  → 503
//   - *clients.ErrUpstreamUnavailable → 503
//   - everything else                 → 500 (no stack trace leaked)
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		// Use the last error attached to the context.
		err := c.Errors.Last().Err

		// If the handler already wrote a response, just log and return.
		if c.Writer.Written() {
			log.Printf("[ErrorHandler] response already written, suppressing error: %v", err)
			return
		}

		writeErrorResponse(c, err)
	}
}

// writeErrorResponse maps an error to an HTTP status + JSON body.
func writeErrorResponse(c *gin.Context, err error) {
	var (
		validationErr  *apperrors.ValidationError
		notFoundErr    *apperrors.NotFoundError
		duplicateErr   *apperrors.DuplicateError
		upstreamErr    *apperrors.UpstreamUnavailableError
		clientUpstream *clients.ErrUpstreamUnavailable
	)

	switch {
	case errors.As(err, &validationErr):
		body := gin.H{"error": "VALIDATION_ERROR", "message": validationErr.Message}
		if len(validationErr.Fields) > 0 {
			body["fields"] = validationErr.Fields
		}
		c.JSON(http.StatusBadRequest, body)

	case errors.As(err, &notFoundErr):
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "NOT_FOUND",
			"message": notFoundErr.Error(),
		})

	case errors.Is(err, repositories.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "NOT_FOUND",
			"message": err.Error(),
		})

	case errors.As(err, &duplicateErr):
		c.JSON(http.StatusConflict, gin.H{
			"error":   "DUPLICATE",
			"message": duplicateErr.Error(),
		})

	case errors.Is(err, repositories.ErrDuplicate):
		c.JSON(http.StatusConflict, gin.H{
			"error":   "DUPLICATE",
			"message": err.Error(),
		})

	case errors.As(err, &upstreamErr):
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "UPSTREAM_UNAVAILABLE",
			"message": upstreamErr.Error(),
		})

	case errors.As(err, &clientUpstream):
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "UPSTREAM_UNAVAILABLE",
			"message": clientUpstream.Error(),
		})

	default:
		// Log the real error server-side but never expose it to the client.
		log.Printf("[ErrorHandler] unhandled error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "INTERNAL_ERROR",
			"message": "an unexpected error occurred",
		})
	}
}
