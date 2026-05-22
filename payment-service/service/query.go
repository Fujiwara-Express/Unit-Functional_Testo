package service

import (
	"encoding/json"
	"net/http"
	"strings"

	"payment-service/domain"
)

// GetPaymentByIDHandler handles GET /payments/{payment_id}.
type GetPaymentByIDHandler struct {
	repo domain.Payment_Repository
}

// NewGetPaymentByIDHandler creates a new GetPaymentByIDHandler.
func NewGetPaymentByIDHandler(repo domain.Payment_Repository) *GetPaymentByIDHandler {
	return &GetPaymentByIDHandler{repo: repo}
}

func (h *GetPaymentByIDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract payment_id from path: /payments/{payment_id}
	paymentID := strings.TrimPrefix(r.URL.Path, "/payments/")
	if paymentID == "" {
		http.Error(w, "missing payment_id", http.StatusBadRequest)
		return
	}

	payment, err := h.repo.GetPaymentByID(r.Context(), paymentID)
	if err != nil || payment == nil {
		http.Error(w, "payment not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(payment)
}

// GetPaymentByOrderIDHandler handles GET /payments?order_id=...
type GetPaymentByOrderIDHandler struct {
	repo domain.Payment_Repository
}

// NewGetPaymentByOrderIDHandler creates a new GetPaymentByOrderIDHandler.
func NewGetPaymentByOrderIDHandler(repo domain.Payment_Repository) *GetPaymentByOrderIDHandler {
	return &GetPaymentByOrderIDHandler{repo: repo}
}

func (h *GetPaymentByOrderIDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	orderID := r.URL.Query().Get("order_id")
	if orderID == "" {
		http.Error(w, "missing required query param: order_id", http.StatusBadRequest)
		return
	}

	payment, err := h.repo.GetPaymentByOrderID(r.Context(), orderID)
	if err != nil || payment == nil {
		http.Error(w, "payment not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(payment)
}
