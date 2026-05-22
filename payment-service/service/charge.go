package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"payment-service/domain"
)

// ChargeService handles payment charge business logic.
type ChargeService struct {
	repo    domain.Payment_Repository
	gateway domain.Payment_Gateway_Client
}

// NewChargeService creates a new ChargeService.
func NewChargeService(repo domain.Payment_Repository, gateway domain.Payment_Gateway_Client) *ChargeService {
	return &ChargeService{repo: repo, gateway: gateway}
}

// ChargeResult is the response returned by the service layer.
type ChargeResult struct {
	PaymentID string              `json:"payment_id"`
	OrderID   string              `json:"order_id"`
	Status    domain.PaymentStatus `json:"status"`
	Method    domain.PaymentMethod `json:"method"`
	VANumber  string              `json:"va_number,omitempty"`
	ExpiredAt *time.Time          `json:"expired_at,omitempty"`
}

// Charge processes a payment charge request.
func (s *ChargeService) Charge(ctx context.Context, req *domain.ChargeRequest) (*ChargeResult, error) {
	paymentID := newID()
	now := time.Now()

	switch req.Method {
	case domain.PaymentMethodTransfer, domain.PaymentMethodVirtualAccount, domain.PaymentMethodQRIS:
		gwResp, err := s.gateway.Charge(ctx, req)
		if err != nil {
			return nil, &GatewayError{Err: err}
		}

		payment := &domain.Payment{
			PaymentID:   paymentID,
			OrderID:     req.OrderID,
			UserID:      req.UserID,
			Amount:      req.Amount,
			Method:      req.Method,
			Status:      domain.PaymentStatusPending,
			ExternalRef: gwResp.ExternalRef,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := s.repo.CreatePayment(ctx, payment); err != nil {
			return nil, err
		}

		result := &ChargeResult{
			PaymentID: paymentID,
			OrderID:   req.OrderID,
			Status:    domain.PaymentStatusPending,
			Method:    req.Method,
			VANumber:  gwResp.VANumber,
		}
		if !gwResp.ExpiredAt.IsZero() {
			result.ExpiredAt = &gwResp.ExpiredAt
		}
		return result, nil

	case domain.PaymentMethodCOD:
		payment := &domain.Payment{
			PaymentID: paymentID,
			OrderID:   req.OrderID,
			UserID:    req.UserID,
			Amount:    req.Amount,
			Method:    req.Method,
			Status:    domain.PaymentStatusSuccess,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := s.repo.CreatePayment(ctx, payment); err != nil {
			return nil, err
		}
		return &ChargeResult{
			PaymentID: paymentID,
			OrderID:   req.OrderID,
			Status:    domain.PaymentStatusSuccess,
			Method:    req.Method,
		}, nil

	default:
		return nil, &ValidationError{Message: "unsupported payment method"}
	}
}

// newID generates a random hex ID.
func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GatewayError wraps a gateway failure.
type GatewayError struct {
	Err error
}

func (e *GatewayError) Error() string { return "gateway error: " + e.Err.Error() }

// ValidationError represents a bad request.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// DuplicateOrderError is returned when an order_id already exists.
type DuplicateOrderError struct {
	OrderID string
}

func (e *DuplicateOrderError) Error() string { return "duplicate order_id: " + e.OrderID }

// ChargeHandler is the HTTP handler for the charge endpoint.
type ChargeHandler struct {
	svc *ChargeService
}

// NewChargeHandler creates a new ChargeHandler.
func NewChargeHandler(svc *ChargeService) *ChargeHandler {
	return &ChargeHandler{svc: svc}
}

func (h *ChargeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req domain.ChargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.OrderID == "" || req.Amount == 0 || req.Method == "" || req.UserID == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	result, err := h.svc.Charge(r.Context(), &req)
	if err != nil {
		switch err.(type) {
		case *GatewayError:
			http.Error(w, "gateway error", http.StatusBadGateway)
		case *ValidationError:
			http.Error(w, err.Error(), http.StatusBadRequest)
		case *DuplicateOrderError:
			http.Error(w, "duplicate order_id", http.StatusConflict)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}
