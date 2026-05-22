package httphandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/report-and-analytics/internal/domain"
	"github.com/report-and-analytics/internal/handler/http/middleware"
	"github.com/report-and-analytics/internal/service"
)

// ReportHandler handles HTTP requests for all report endpoints.
type ReportHandler struct {
	svc service.ReportService
}

// NewReportHandler creates a new ReportHandler.
func NewReportHandler(svc service.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

// writeError writes a standard JSON error response.
func writeError(w http.ResponseWriter, r *http.Request, code, message string, status int) {
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
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR"
	}
}

// GetOrderReport handles GET /reports/orders
func (h *ReportHandler) GetOrderReport(w http.ResponseWriter, r *http.Request) {
	f := domain.OrderReportFilter{
		DateFrom: r.URL.Query().Get("date_from"),
		DateTo:   r.URL.Query().Get("date_to"),
		HubID:    r.URL.Query().Get("hub_id"),
	}

	rep, err := h.svc.GetOrderReport(r.Context(), f)
	if err != nil {
		status, code := mapError(err)
		writeError(w, r, code, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(rep)
}

// GetDeliveryPerformanceReport handles GET /reports/delivery-performance
func (h *ReportHandler) GetDeliveryPerformanceReport(w http.ResponseWriter, r *http.Request) {
	f := domain.DeliveryPerformanceFilter{
		CourierID: r.URL.Query().Get("courier_id"),
		Period:    r.URL.Query().Get("period"),
	}

	rep, err := h.svc.GetDeliveryPerformanceReport(r.Context(), f)
	if err != nil {
		status, code := mapError(err)
		writeError(w, r, code, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(rep)
}

// GetRevenueReport handles GET /reports/revenue
func (h *ReportHandler) GetRevenueReport(w http.ResponseWriter, r *http.Request) {
	f := domain.RevenueFilter{
		Period:      r.URL.Query().Get("period"),
		ServiceType: r.URL.Query().Get("service_type"),
	}

	rep, err := h.svc.GetRevenueReport(r.Context(), f)
	if err != nil {
		status, code := mapError(err)
		writeError(w, r, code, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(rep)
}

// GetHubPerformanceReport handles GET /reports/hub-performance
func (h *ReportHandler) GetHubPerformanceReport(w http.ResponseWriter, r *http.Request) {
	f := domain.HubPerformanceFilter{
		HubID:  r.URL.Query().Get("hub_id"),
		Period: r.URL.Query().Get("period"),
	}

	rep, err := h.svc.GetHubPerformanceReport(r.Context(), f)
	if err != nil {
		status, code := mapError(err)
		writeError(w, r, code, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(rep)
}
