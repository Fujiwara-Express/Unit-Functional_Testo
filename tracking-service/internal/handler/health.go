package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
	"tracking-service/internal/middleware"
	"tracking-service/internal/models"
	"tracking-service/internal/repository"
)

// HealthHandler handles GET /health.
type HealthHandler struct {
	repo repository.Repository
}

// NewHealthHandler returns a new HealthHandler backed by the given repository.
func NewHealthHandler(repo repository.Repository) *HealthHandler {
	return &HealthHandler{repo: repo}
}

// ServeHTTP calls repo.Ping with a 5-second deadline. It returns HTTP 200
// {"status":"ok"} on success and HTTP 503 ErrorResponse on failure.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	w.Header().Set("Content-Type", "application/json")

	if err := h.repo.Ping(ctx); err != nil {
		correlationID := middleware.GetCorrelationID(r.Context())
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:         "database unavailable",
			CorrelationID: correlationID,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
