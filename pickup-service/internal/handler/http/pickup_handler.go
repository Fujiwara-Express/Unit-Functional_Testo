package httphandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/pickup-service/internal/domain"
	"github.com/pickup-service/internal/handler/http/middleware"
	"github.com/pickup-service/internal/service"
)

// PickupHandler handles HTTP requests for pickup operations.
type PickupHandler struct {
	svc service.PickupService
}

// NewPickupHandler creates a new PickupHandler with the given service.
func NewPickupHandler(svc service.PickupService) *PickupHandler {
	return &PickupHandler{svc: svc}
}

// writeError writes a standard error response.
func writeError(w http.ResponseWriter, r *http.Request, code string, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Code:      code,
		Message:   message,
		RequestID: middleware.GetRequestID(r.Context()),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// mapError maps a domain error to an HTTP status code and error code string.
func mapError(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "NOT_FOUND"
	case errors.Is(err, domain.ErrValidation):
		return http.StatusBadRequest, "VALIDATION_ERROR"
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, "CONFLICT"
	case errors.Is(err, domain.ErrInvalidTransition):
		return http.StatusBadRequest, "INVALID_TRANSITION"
	case errors.Is(err, domain.ErrServiceUnavailable):
		return http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR"
	}
}

// requestPickupBody is the JSON body for POST /pickups.
type requestPickupBody struct {
	OrderID        string `json:"order_id"`
	UserID         string `json:"user_id"`
	PickupAddress  string `json:"pickup_address"`
	PickupCityCode string `json:"pickup_city_code"`
	ContactName    string `json:"contact_name"`
	ContactPhone   string `json:"contact_phone"`
}

// RequestPickup handles POST /pickups
func (h *PickupHandler) RequestPickup(w http.ResponseWriter, r *http.Request) {
	var body requestPickupBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, "BAD_REQUEST", "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	switch {
	case body.OrderID == "":
		writeError(w, r, "VALIDATION_ERROR", "missing required field order_id", http.StatusBadRequest)
		return
	case body.UserID == "":
		writeError(w, r, "VALIDATION_ERROR", "missing required field user_id", http.StatusBadRequest)
		return
	case body.PickupAddress == "":
		writeError(w, r, "VALIDATION_ERROR", "missing required field pickup_address", http.StatusBadRequest)
		return
	case body.PickupCityCode == "":
		writeError(w, r, "VALIDATION_ERROR", "missing required field pickup_city_code", http.StatusBadRequest)
		return
	case body.ContactName == "":
		writeError(w, r, "VALIDATION_ERROR", "missing required field contact_name", http.StatusBadRequest)
		return
	case body.ContactPhone == "":
		writeError(w, r, "VALIDATION_ERROR", "missing required field contact_phone", http.StatusBadRequest)
		return
	}

	out, err := h.svc.RequestPickup(r.Context(), service.RequestPickupInput{
		OrderID:        body.OrderID,
		UserID:         body.UserID,
		PickupAddress:  body.PickupAddress,
		PickupCityCode: body.PickupCityCode,
		ContactName:    body.ContactName,
		ContactPhone:   body.ContactPhone,
	})
	if err != nil {
		status, code := mapError(err)
		writeError(w, r, code, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"pickup_id":             out.PickupID,
		"order_id":              out.OrderID,
		"status":                string(out.Status),
		"estimated_pickup_time": out.EstimatedPickupTime,
		"created_at":            out.CreatedAt,
	})
}

// assignCourierBody is the JSON body for POST /pickups/{pickup_id}/assign.
type assignCourierBody struct {
	CourierID string `json:"courier_id"`
}

// AssignCourier handles POST /pickups/{pickup_id}/assign
func (h *PickupHandler) AssignCourier(w http.ResponseWriter, r *http.Request) {
	pickupID := r.PathValue("pickup_id")
	if pickupID == "" {
		writeError(w, r, "BAD_REQUEST", "missing pickup_id", http.StatusBadRequest)
		return
	}

	var body assignCourierBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, "BAD_REQUEST", "invalid request body", http.StatusBadRequest)
		return
	}

	out, err := h.svc.AssignCourier(r.Context(), pickupID, body.CourierID)
	if err != nil {
		status, code := mapError(err)
		writeError(w, r, code, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"pickup_id":  out.PickupID,
		"courier_id": out.CourierID,
		"status":     string(out.Status),
	})
}

// updateStatusBody is the JSON body for POST /pickups/{pickup_id}/status.
type updateStatusBody struct {
	Status string `json:"status"`
}

// UpdatePickupStatus handles POST /pickups/{pickup_id}/status
func (h *PickupHandler) UpdatePickupStatus(w http.ResponseWriter, r *http.Request) {
	pickupID := r.PathValue("pickup_id")
	if pickupID == "" {
		writeError(w, r, "BAD_REQUEST", "missing pickup_id", http.StatusBadRequest)
		return
	}

	var body updateStatusBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, "BAD_REQUEST", "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate status value
	status := domain.Status(body.Status)
	validStatuses := map[domain.Status]bool{
		domain.StatusScheduled:     true,
		domain.StatusAssigned:      true,
		domain.StatusPickedUp:      true,
		domain.StatusFailedAttempt: true,
		domain.StatusCancelled:     true,
	}
	if !validStatuses[status] {
		writeError(w, r, "VALIDATION_ERROR", "invalid status value: "+body.Status, http.StatusBadRequest)
		return
	}

	out, err := h.svc.UpdatePickupStatus(r.Context(), pickupID, status)
	if err != nil {
		httpStatus, code := mapError(err)
		writeError(w, r, code, err.Error(), httpStatus)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"pickup_id": out.PickupID,
		"status":    string(out.Status),
		"timestamp": out.Timestamp,
	})
}

// GetPickup handles GET /pickups/{pickup_id}
func (h *PickupHandler) GetPickup(w http.ResponseWriter, r *http.Request) {
	pickupID := r.PathValue("pickup_id")
	if pickupID == "" {
		writeError(w, r, "BAD_REQUEST", "missing pickup_id", http.StatusBadRequest)
		return
	}

	pickup, err := h.svc.GetPickup(r.Context(), pickupID)
	if err != nil {
		status, code := mapError(err)
		writeError(w, r, code, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(pickup)
}

// ListPickups handles GET /pickups
func (h *PickupHandler) ListPickups(w http.ResponseWriter, r *http.Request) {
	filters := service.ListFilters{
		CourierID: r.URL.Query().Get("courier_id"),
		Status:    r.URL.Query().Get("status"),
		Date:      r.URL.Query().Get("date"),
	}

	pickups, err := h.svc.ListPickups(r.Context(), filters)
	if err != nil {
		status, code := mapError(err)
		writeError(w, r, code, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(pickups)
}

// CancelPickup handles POST /pickups/{pickup_id}/cancel
func (h *PickupHandler) CancelPickup(w http.ResponseWriter, r *http.Request) {
	pickupID := r.PathValue("pickup_id")
	if pickupID == "" {
		writeError(w, r, "BAD_REQUEST", "missing pickup_id", http.StatusBadRequest)
		return
	}

	out, err := h.svc.CancelPickup(r.Context(), pickupID)
	if err != nil {
		httpStatus, code := mapError(err)
		writeError(w, r, code, err.Error(), httpStatus)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"pickup_id": out.PickupID,
		"status":    string(out.Status),
	})
}
