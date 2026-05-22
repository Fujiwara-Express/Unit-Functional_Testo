package service

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"payment-service/domain"
)

// NotFoundError is returned when a payment is not found or method doesn't match.
type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string { return e.Message }

// CodConfirmRequest is the request payload for confirming a COD payment.
type CodConfirmRequest struct {
	OrderID         string  `json:"order_id"`
	CourierID       string  `json:"courier_id"`
	AmountCollected float64 `json:"amount_collected"`
}

// CodConfirmService handles COD confirmation business logic.
type CodConfirmService struct {
	repo domain.Payment_Repository
}

// NewCodConfirmService creates a new CodConfirmService.
func NewCodConfirmService(repo domain.Payment_Repository) *CodConfirmService {
	return &CodConfirmService{repo: repo}
}

// Confirm processes a COD confirmation request.
func (s *CodConfirmService) Confirm(ctx context.Context, req *CodConfirmRequest) error {
	payment, err := s.repo.GetPaymentByOrderID(ctx, req.OrderID)
	if err != nil || payment == nil {
		return &NotFoundError{Message: "payment not found for order_id: " + req.OrderID}
	}

	if payment.Method != domain.PaymentMethodCOD {
		return &NotFoundError{Message: "payment method is not COD for order_id: " + req.OrderID}
	}

	collection := &domain.CodCollection{
		CollectionID:    newID(),
		OrderID:         req.OrderID,
		CourierID:       req.CourierID,
		AmountCollected: req.AmountCollected,
		CollectedAt:     time.Now(),
		RemittanceStatus: domain.RemittanceStatusPending,
	}
	if err := s.repo.CreateCodCollection(ctx, collection); err != nil {
		return err
	}

	return s.repo.UpdatePaymentStatus(ctx, req.OrderID, domain.PaymentStatusSuccess, "")
}

// CodConfirmHandler is the HTTP handler for the COD confirm endpoint.
type CodConfirmHandler struct {
	svc *CodConfirmService
}

// NewCodConfirmHandler creates a new CodConfirmHandler.
func NewCodConfirmHandler(svc *CodConfirmService) *CodConfirmHandler {
	return &CodConfirmHandler{svc: svc}
}

func (h *CodConfirmHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req CodConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.OrderID == "" || req.CourierID == "" || req.AmountCollected == 0 {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	if err := h.svc.Confirm(r.Context(), &req); err != nil {
		switch err.(type) {
		case *NotFoundError:
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}
