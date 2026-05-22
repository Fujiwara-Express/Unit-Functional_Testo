package router

import (
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"tracking-service/internal/handler"
	"tracking-service/internal/middleware"
	"tracking-service/internal/repository"
)

// New creates and returns a configured chi router with all routes and middleware.
// The middleware chain is applied in order: CorrelationIDMiddleware, LoggingMiddleware, RecoveryMiddleware.
// Routes are registered with /tracking/bulk before /tracking/{tracking_number} so chi
// matches the literal path first.
func New(repo repository.Repository, queryTimeout time.Duration, logWriter io.Writer) http.Handler {
	r := chi.NewRouter()

	// Middleware chain
	r.Use(middleware.CorrelationIDMiddleware)
	r.Use(middleware.NewLoggingMiddleware(logWriter))
	r.Use(middleware.RecoveryMiddleware)

	// Routes
	r.Post("/tracking/events", handler.NewEventHandler(repo, queryTimeout).ServeHTTP)
	r.Get("/tracking/bulk", handler.NewBulkHandler(repo, queryTimeout).ServeHTTP)
	r.Get("/tracking/{tracking_number}", handler.NewHistoryHandler(repo, queryTimeout).ServeHTTP)
	r.Get("/health", handler.NewHealthHandler(repo).ServeHTTP)

	return r
}
