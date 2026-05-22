package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"tracking-service/internal/models"
	"tracking-service/internal/repository"
	"tracking-service/internal/validator"
)

// BulkHandler handles GET /tracking/bulk.
type BulkHandler struct {
	repo         repository.Repository
	queryTimeout time.Duration
}

// NewBulkHandler returns a new BulkHandler backed by the given repository
// and using queryTimeout for database operation deadlines.
func NewBulkHandler(repo repository.Repository, queryTimeout time.Duration) *BulkHandler {
	return &BulkHandler{repo: repo, queryTimeout: queryTimeout}
}

// ServeHTTP handles GET /tracking/bulk?numbers=TRK1,TRK2,...
//
// It parses the comma-separated `numbers` query parameter, validates the list,
// fetches summaries from the repository, and returns a JSON array of
// TrackingSummary objects for the tracking numbers that exist.
func (h *BulkHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	numbersParam := r.URL.Query().Get("numbers")
	if numbersParam == "" {
		writeError(w, r, http.StatusBadRequest, "query parameter 'numbers' is required")
		return
	}

	numbers := strings.Split(numbersParam, ",")

	// Validate the numbers slice (checks for empty strings and >100 count).
	if errs := validator.ValidateBulkNumbers(numbers); len(errs) > 0 {
		// We already checked for empty param above, so any error here is either
		// an empty string element (400) or too many numbers (422).
		// ValidateBulkNumbers returns a "required" error for empty slice (already
		// handled), an "empty strings" error for blank elements, and a count error
		// for >100. The count error maps to 422; empty-string error maps to 400.
		firstErr := errs[0]
		if strings.Contains(firstErr.Message, "maximum 100") {
			writeError(w, r, http.StatusUnprocessableEntity, firstErr.Message)
		} else {
			writeError(w, r, http.StatusBadRequest, firstErr.Message)
		}
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	summaries, err := h.repo.GetSummariesByTrackingNumbers(ctx, numbers)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	// Return empty array instead of null when no summaries are found.
	if summaries == nil {
		summaries = []models.TrackingSummary{}
	}

	writeJSON(w, http.StatusOK, summaries)
}
