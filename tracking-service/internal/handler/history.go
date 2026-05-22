package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"tracking-service/internal/models"
	"tracking-service/internal/repository"
)

// HistoryHandler handles GET /tracking/{tracking_number}.
type HistoryHandler struct {
	repo         repository.Repository
	queryTimeout time.Duration
}

// NewHistoryHandler returns a new HistoryHandler backed by the given repository
// and using queryTimeout for database operation deadlines.
func NewHistoryHandler(repo repository.Repository, queryTimeout time.Duration) *HistoryHandler {
	return &HistoryHandler{repo: repo, queryTimeout: queryTimeout}
}

// ServeHTTP handles GET /tracking/{tracking_number}.
//
// It extracts the tracking number from the URL path, fetches events and summary
// from the repository, and returns a TrackingHistoryResponse as JSON.
func (h *HistoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	trackingNumber := chi.URLParam(r, "tracking_number")

	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	// Get events (sorted ascending by the repository).
	events, err := h.repo.GetEventsByTrackingNumber(ctx, trackingNumber)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, fmt.Sprintf("tracking number '%s' not found", trackingNumber))
			return
		}
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	// Get summary.
	summary, err := h.repo.GetSummaryByTrackingNumber(ctx, trackingNumber)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, fmt.Sprintf("tracking number '%s' not found", trackingNumber))
			return
		}
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	// Build history entries from events (already sorted ascending by repo).
	history := make([]models.HistoryEntry, len(events))
	for i, e := range events {
		history[i] = models.HistoryEntry{
			Status:    e.Status,
			Timestamp: e.Timestamp,
		}
	}

	resp := models.TrackingHistoryResponse{
		TrackingNumber:    trackingNumber,
		CurrentStatus:     summary.CurrentStatus,
		EstimatedDelivery: summary.EstimatedDelivery,
		History:           history,
	}

	writeJSON(w, http.StatusOK, resp)
}
