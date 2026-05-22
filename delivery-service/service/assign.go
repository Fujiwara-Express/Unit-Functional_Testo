package service

import (
	"encoding/json"
	"net/http"
	"time"

	"delivery-service/domain"
)

// RoutingError wraps a routing client failure.
type RoutingError struct{ Err error }

func (e *RoutingError) Error() string { return "routing error: " + e.Err.Error() }

// AssignService handles delivery job assignment logic.
type AssignService struct {
	repo    domain.Delivery_Repository
	routing domain.Routing_Client
}

// NewAssignService creates a new AssignService.
func NewAssignService(repo domain.Delivery_Repository, routing domain.Routing_Client) *AssignService {
	return &AssignService{repo: repo, routing: routing}
}

// AssignHandler is the HTTP handler for POST /delivery/assign.
type AssignHandler struct {
	svc *AssignService
}

// NewAssignHandler creates a new AssignHandler.
func NewAssignHandler(svc *AssignService) *AssignHandler {
	return &AssignHandler{svc: svc}
}

func (h *AssignHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req domain.AssignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.TrackingNumber == "" || req.HubID == "" || req.CourierID == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	// Call routing client first — if it fails, do NOT persist the job
	route, err := h.svc.routing.GetCourierRoute(r.Context(), req.CourierID)
	if err != nil {
		http.Error(w, "routing service error", http.StatusBadGateway)
		return
	}

	now := time.Now()
	job := &domain.DeliveryJob{
		JobID:          newDeliveryID(),
		TrackingNumber: req.TrackingNumber,
		CourierID:      req.CourierID,
		HubID:          req.HubID,
		Status:         domain.JobStatusOutForDelivery,
		AttemptCount:   0,
		AssignedAt:     now,
	}

	if err := h.svc.repo.CreateDeliveryJob(r.Context(), job); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"delivery_id":    job.JobID,
		"tracking_number": req.TrackingNumber,
		"courier_id":     req.CourierID,
		"status":         string(job.Status),
		"delivery_route": route,
	})
}
