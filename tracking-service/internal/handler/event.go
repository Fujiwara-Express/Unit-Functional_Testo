package handler

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"tracking-service/internal/middleware"
	"tracking-service/internal/models"
	"tracking-service/internal/repository"
	"tracking-service/internal/validator"
)

// EventHandler handles POST /tracking/events.
type EventHandler struct {
	repo         repository.Repository
	queryTimeout time.Duration
}

// NewEventHandler returns a new EventHandler backed by the given repository
// and using queryTimeout for database operation deadlines.
func NewEventHandler(repo repository.Repository, queryTimeout time.Duration) *EventHandler {
	return &EventHandler{repo: repo, queryTimeout: queryTimeout}
}

// generateEventUUID generates a random UUID v4 using crypto/rand.
func generateEventUUID() string {
	var uuid [16]byte
	rand.Read(uuid[:]) //nolint:errcheck // crypto/rand.Read never returns an error on supported platforms
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

// writeJSON writes a JSON-encoded value with the given HTTP status code.
// Content-Type is always set to application/json.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// writeError writes an ErrorResponse JSON body with the given status code.
func writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
	correlationID := middleware.GetCorrelationID(r.Context())
	writeJSON(w, status, models.ErrorResponse{
		Error:         message,
		CorrelationID: correlationID,
	})
}

// ServeHTTP handles POST /tracking/events.
//
// It decodes the JSON body, validates the request, builds a TrackingEvent,
// persists it via the repository, and returns HTTP 201 with the new event ID.
func (h *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Decode JSON body.
	var req models.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "malformed request body")
		return
	}

	// 2. Validate the request.
	validationErrs := validator.ValidateCreateEventRequest(req)
	if len(validationErrs) > 0 {
		// Separate 400 (missing fields) from 422 (invalid format/enum).
		var missing400 []string
		var first422 string

		for _, ve := range validationErrs {
			switch ve.Field {
			case "tracking_number":
				// tracking_number is always a missing-field (400) error.
				missing400 = append(missing400, ve.Field)
			case "status":
				if ve.Message == "status is required" {
					missing400 = append(missing400, ve.Field)
				} else {
					// Invalid enum value → 422.
					if first422 == "" {
						first422 = ve.Message
					}
				}
			case "timestamp":
				if ve.Message == "timestamp is required" {
					missing400 = append(missing400, ve.Field)
				} else {
					// Invalid format → 422.
					if first422 == "" {
						first422 = ve.Message
					}
				}
			}
		}

		// 400 takes priority over 422.
		if len(missing400) > 0 {
			writeError(w, r, http.StatusBadRequest,
				"missing fields: ["+strings.Join(missing400, ", ")+"]")
			return
		}
		if first422 != "" {
			writeError(w, r, http.StatusUnprocessableEntity, first422)
			return
		}
	}

	// 3. Parse the validated RFC 3339 timestamp.
	ts, err := time.Parse(time.RFC3339, req.Timestamp)
	if err != nil {
		// Should not happen after validation, but guard defensively.
		writeError(w, r, http.StatusUnprocessableEntity,
			fmt.Sprintf("timestamp must be RFC 3339; got '%s'", req.Timestamp))
		return
	}

	// 4. Build the domain event.
	event := models.TrackingEvent{
		EventID:          generateEventUUID(),
		TrackingNumber:   req.TrackingNumber,
		Status:           models.Status(req.Status),
		Location:         req.Location,
		HubID:            req.HubID,
		Notes:            req.Notes,
		CreatedByService: "tracking-service",
		Timestamp:        ts,
	}

	// 5. Persist via repository with a deadline context.
	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	if err := h.repo.InsertEventAndUpsertSummary(ctx, event); err != nil {
		switch {
		case errors.Is(err, repository.ErrPoolExhausted):
			writeError(w, r, http.StatusServiceUnavailable, "service temporarily unavailable")
		case errors.Is(err, context.DeadlineExceeded):
			writeError(w, r, http.StatusGatewayTimeout, "upstream timeout")
		default:
			writeError(w, r, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	// 6. Return HTTP 201 with the new event ID.
	writeJSON(w, http.StatusCreated, models.CreateEventResponse{EventID: event.EventID})
}
