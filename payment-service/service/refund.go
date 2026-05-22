package service

import (
	"context"
	"encoding/json"
	"net/http"

	"payment-service/domain"
)

// IneligibleStatusError is returned when a payment cannot be refunded due to its status.
type IneligibleStatusError struct {
	Status domain.PaymentStatus
}

func (e *IneligibleStatusError) Error() string {
	return "payment status " + string(e.Status) + " is not eligible for refund"
}

// RefundRequest is the request payload for refunding a payment.
type RefundRequest struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

// RefundService handles payment refund business logic.
type RefundService struct {
	repo domain.Payment_Repository
}

// NewRefundService creates a new RefundService.
func NewRefundService(repo domain.Payment_Repository) *RefundService {
	return &RefundService{repo: repo}
}

// Refund processes a refund request.
func (s *RefundService) Refund(ctx context.Context, req *RefundRequest) error {
	payment, err := s.repo.GetPaymentByOrderID(ctx, req.OrderID)
	if err != nil || payment == nil {
		return &NotFoundError{Message: "payment not found for order_id: " + req.OrderID}
	}

	switch payment.Status {
	case domain.PaymentStatusPending, domain.PaymentStatusFailed, domain.PaymentStatusRefunded:
		return &IneligibleStatusError{Status: payment.Status}
	case domain.PaymentStatusSuccess:
		return s.repo.UpdatePaymentStatus(ctx, req.OrderID, domain.PaymentStatusRefunded, "")
	default:
		return &IneligibleStatusError{Status: payment.Status}
	}
}

// RefundHandler is the HTTP handler for the refund endpoint.
type RefundHandler struct {
	svc *RefundService
}

// NewRefundHandler creates a new RefundHandler.
func NewRefundHandler(svc *RefundService) *RefundHandler {
	return &RefundHandler{svc: svc}
}

func (h *RefundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req RefundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.OrderID == "" {
		http.Error(w, "missing required field: order_id", http.StatusBadRequest)
		return
	}

	if err := h.svc.Refund(r.Context(), &req); err != nil {
		switch err.(type) {
		case *NotFoundError:
			http.Error(w, err.Error(), http.StatusNotFound)
		case *IneligibleStatusError:
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}
