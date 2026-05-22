package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"payment-service/domain"
)

// CallbackRequest is the payload sent by the payment gateway.
type CallbackRequest struct {
	ExternalRef string    `json:"external_ref"`
	OrderID     string    `json:"order_id"`
	Status      string    `json:"status"`
	Method      string    `json:"method"`
	Amount      float64   `json:"amount"`
	PaidAt      time.Time `json:"paid_at"`
	Signature   string    `json:"signature"`
}

// CallbackResponse is returned after processing a callback.
type CallbackResponse struct {
	PaymentID string    `json:"payment_id"`
	OrderID   string    `json:"order_id"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CallbackService handles payment gateway callback processing.
type CallbackService struct {
	repo      domain.Payment_Repository
	kafka     domain.Kafka_Publisher
	validator *SignatureValidator
}

// NewCallbackService creates a new CallbackService.
func NewCallbackService(repo domain.Payment_Repository, kafka domain.Kafka_Publisher, validator *SignatureValidator) *CallbackService {
	return &CallbackService{repo: repo, kafka: kafka, validator: validator}
}

// Process handles a validated callback request.
func (s *CallbackService) Process(ctx context.Context, req *CallbackRequest) (*CallbackResponse, error) {
	// Build the payload string for signature validation (external_ref + order_id + status)
	payload := fmt.Sprintf("%s%s%s", req.ExternalRef, req.OrderID, req.Status)
	if !s.validator.Validate(payload, req.Signature) {
		return nil, &UnauthorizedError{Message: "invalid signature"}
	}

	// Idempotency: check if external_ref already processed
	existing, err := s.repo.GetPaymentByExternalRef(ctx, req.ExternalRef)
	if err == nil && existing != nil {
		return &CallbackResponse{
			PaymentID: existing.PaymentID,
			OrderID:   existing.OrderID,
			Status:    string(existing.Status),
			UpdatedAt: existing.UpdatedAt,
		}, nil
	}

	// Determine new status
	var newStatus domain.PaymentStatus
	var eventType domain.PaymentEventType
	switch req.Status {
	case "SUCCESS":
		newStatus = domain.PaymentStatusSuccess
		eventType = domain.PaymentEventSuccess
	case "FAILED", "EXPIRED":
		newStatus = domain.PaymentStatusFailed
		eventType = domain.PaymentEventFailed
	default:
		return nil, &ValidationError{Message: "unknown callback status: " + req.Status}
	}

	// Update payment status
	if err := s.repo.UpdatePaymentStatus(ctx, req.OrderID, newStatus, req.ExternalRef); err != nil {
		return nil, err
	}

	// Fetch updated payment for response
	payment, err := s.repo.GetPaymentByOrderID(ctx, req.OrderID)
	if err != nil || payment == nil {
		return nil, &NotFoundError{Message: "payment not found after update"}
	}

	// Publish Kafka event
	event := &domain.PaymentEvent{
		EventType:   eventType,
		PaymentID:   payment.PaymentID,
		OrderID:     payment.OrderID,
		UserID:      payment.UserID,
		Amount:      payment.Amount,
		Method:      payment.Method,
		Status:      newStatus,
		ExternalRef: req.ExternalRef,
		OccurredAt:  time.Now(),
	}
	if err := s.kafka.Publish(ctx, string(eventType), event); err != nil {
		return nil, err
	}

	return &CallbackResponse{
		PaymentID: payment.PaymentID,
		OrderID:   payment.OrderID,
		Status:    string(newStatus),
		UpdatedAt: payment.UpdatedAt,
	}, nil
}

// UnauthorizedError is returned when signature validation fails.
type UnauthorizedError struct {
	Message string
}

func (e *UnauthorizedError) Error() string { return e.Message }

// CallbackHandler is the HTTP handler for the callback endpoint.
type CallbackHandler struct {
	svc *CallbackService
}

// NewCallbackHandler creates a new CallbackHandler.
func NewCallbackHandler(svc *CallbackService) *CallbackHandler {
	return &CallbackHandler{svc: svc}
}

func (h *CallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req CallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.svc.Process(r.Context(), &req)
	if err != nil {
		switch err.(type) {
		case *UnauthorizedError:
			http.Error(w, err.Error(), http.StatusUnauthorized)
		case *NotFoundError:
			http.Error(w, err.Error(), http.StatusNotFound)
		case *ValidationError:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}
