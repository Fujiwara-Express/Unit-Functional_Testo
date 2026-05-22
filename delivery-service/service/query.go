package service

import (
	"encoding/json"
	"net/http"
	"strings"

	"delivery-service/domain"
)

// QueryService handles delivery job and courier query logic.
type QueryService struct {
	repo domain.Delivery_Repository
}

// NewQueryService creates a new QueryService.
func NewQueryService(repo domain.Delivery_Repository) *QueryService {
	return &QueryService{repo: repo}
}

// --- Get Courier Jobs ---

// GetCourierJobsHandler is the HTTP handler for GET /delivery/courier/{courier_id}/jobs.
type GetCourierJobsHandler struct {
	svc *QueryService
}

// NewGetCourierJobsHandler creates a new GetCourierJobsHandler.
func NewGetCourierJobsHandler(svc *QueryService) *GetCourierJobsHandler {
	return &GetCourierJobsHandler{svc: svc}
}

func (h *GetCourierJobsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract courier_id from path: /delivery/courier/{courier_id}/jobs
	path := strings.TrimPrefix(r.URL.Path, "/delivery/courier/")
	courierID := strings.TrimSuffix(path, "/jobs")

	jobs, err := h.svc.repo.GetJobsByCourierID(r.Context(), courierID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if jobs == nil {
		jobs = []*domain.DeliveryJob{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(jobs)
}

// --- Get Delivery Detail ---

// GetDeliveryDetailHandler is the HTTP handler for GET /delivery/{delivery_id}.
type GetDeliveryDetailHandler struct {
	svc *QueryService
}

// NewGetDeliveryDetailHandler creates a new GetDeliveryDetailHandler.
func NewGetDeliveryDetailHandler(svc *QueryService) *GetDeliveryDetailHandler {
	return &GetDeliveryDetailHandler{svc: svc}
}

func (h *GetDeliveryDetailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract delivery_id from path: /delivery/{delivery_id}
	deliveryID := strings.TrimPrefix(r.URL.Path, "/delivery/")

	job, err := h.svc.repo.GetDeliveryJobByID(r.Context(), deliveryID)
	if err != nil || job == nil {
		http.Error(w, "delivery not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"delivery_id":     job.JobID,
		"tracking_number": job.TrackingNumber,
		"courier_id":      job.CourierID,
		"hub_id":          job.HubID,
		"status":          string(job.Status),
		"attempt_count":   job.AttemptCount,
		"proof_photo_url": job.ProofPhotoURL,
		"recipient_name":  job.RecipientName,
		"notes":           job.Notes,
		"assigned_at":     job.AssignedAt,
		"completed_at":    job.CompletedAt,
	})
}
