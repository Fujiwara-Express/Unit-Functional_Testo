package service

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"delivery-service/domain"
)

// StatusService handles delivery status update logic.
type StatusService struct {
	repo domain.Delivery_Repository
}

// NewStatusService creates a new StatusService.
func NewStatusService(repo domain.Delivery_Repository) *StatusService {
	return &StatusService{repo: repo}
}

// StatusUpdateHandler is the HTTP handler for POST /delivery/status.
type StatusUpdateHandler struct {
	svc *StatusService
}

// NewStatusUpdateHandler creates a new StatusUpdateHandler.
func NewStatusUpdateHandler(svc *StatusService) *StatusUpdateHandler {
	return &StatusUpdateHandler{svc: svc}
}

func (h *StatusUpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req domain.StatusUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.TrackingNumber == "" || req.CourierID == "" || req.Status == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	validStatuses := map[string]bool{
		"DELIVERED":     true,
		"FAILED_ATTEMPT": true,
		"RETURNED":      true,
	}
	if !validStatuses[strings.ToUpper(req.Status)] {
		http.Error(w, "invalid status value", http.StatusBadRequest)
		return
	}

	// Look up the job by tracking number
	job, err := h.svc.repo.GetDeliveryJobByTrackingNumber(r.Context(), req.TrackingNumber)
	if err != nil {
		http.Error(w, "tracking number not found", http.StatusNotFound)
		return
	}

	now := time.Now()
	update := &domain.JobStatusUpdate{}

	switch strings.ToUpper(req.Status) {
	case "DELIVERED":
		update.Status = domain.JobStatusDelivered
		update.ProofPhotoURL = req.ProofPhotoURL
		update.RecipientName = req.RecipientName
		update.Notes = req.Notes
		update.CompletedAt = &now
		update.AttemptCount = job.AttemptCount

	case "FAILED_ATTEMPT":
		update.Status = domain.JobStatusFailed
		update.AttemptCount = job.AttemptCount + 1
		update.Notes = req.Notes

	case "RETURNED":
		update.Status = domain.JobStatusReturned
		update.CompletedAt = &now
		update.AttemptCount = job.AttemptCount
	}

	if err := h.svc.repo.UpdateDeliveryJobStatus(r.Context(), job.JobID, update); err != nil {
		switch err.(type) {
		case *NotFoundError:
			http.Error(w, "delivery job not found", http.StatusNotFound)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"tracking_number": req.TrackingNumber,
		"status":          update.Status.String(),
	})
}
