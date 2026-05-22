package service_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"payment-service/domain"
	"payment-service/mocks"
	"payment-service/service"

	"go.uber.org/mock/gomock"
)

const callbackSecret = "callback-secret"

func buildCallbackPayload(externalRef, orderID, status string) string {
	return fmt.Sprintf("%s%s%s", externalRef, orderID, status)
}

func newCallbackValidator() *service.SignatureValidator {
	return service.NewSignatureValidator(callbackSecret)
}

func TestCallback_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)
	mockKafka := mocks.NewMockKafka_Publisher(ctrl)
	validator := newCallbackValidator()

	const (
		externalRef = "EXT-001"
		orderID     = "ORD-001"
	)
	sig := validator.Compute(buildCallbackPayload(externalRef, orderID, "SUCCESS"))

	// Idempotency check — not yet processed
	mockRepo.EXPECT().
		GetPaymentByExternalRef(gomock.Any(), externalRef).
		Return(nil, sql.ErrNoRows).
		Times(1)

	// Update status to SUCCESS
	mockRepo.EXPECT().
		UpdatePaymentStatus(gomock.Any(), orderID, domain.PaymentStatusSuccess, externalRef).
		Return(nil).
		Times(1)

	// Fetch updated payment
	mockRepo.EXPECT().
		GetPaymentByOrderID(gomock.Any(), orderID).
		Return(&domain.Payment{
			PaymentID: "PAY-001",
			OrderID:   orderID,
			UserID:    "USR-001",
			Amount:    50000,
			Method:    domain.PaymentMethodVirtualAccount,
			Status:    domain.PaymentStatusSuccess,
			UpdatedAt: time.Now(),
		}, nil).
		Times(1)

	// Kafka publish PAYMENT_SUCCESS
	mockKafka.EXPECT().
		Publish(gomock.Any(), string(domain.PaymentEventSuccess), gomock.Any()).
		Return(nil).
		Times(1)

	svc := service.NewCallbackService(mockRepo, mockKafka, validator)
	handler := service.NewCallbackHandler(svc)

	body, _ := json.Marshal(service.CallbackRequest{
		ExternalRef: externalRef,
		OrderID:     orderID,
		Status:      "SUCCESS",
		Signature:   sig,
	})
	req := httptest.NewRequest(http.MethodPost, "/payments/callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	for _, field := range []string{"payment_id", "order_id", "status", "updated_at"} {
		if _, ok := resp[field]; !ok {
			t.Errorf("response missing field %q", field)
		}
	}
}

func TestCallback_Failed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)
	mockKafka := mocks.NewMockKafka_Publisher(ctrl)
	validator := newCallbackValidator()

	const (
		externalRef = "EXT-002"
		orderID     = "ORD-002"
	)
	sig := validator.Compute(buildCallbackPayload(externalRef, orderID, "FAILED"))

	mockRepo.EXPECT().GetPaymentByExternalRef(gomock.Any(), externalRef).Return(nil, sql.ErrNoRows).Times(1)
	mockRepo.EXPECT().UpdatePaymentStatus(gomock.Any(), orderID, domain.PaymentStatusFailed, externalRef).Return(nil).Times(1)
	mockRepo.EXPECT().GetPaymentByOrderID(gomock.Any(), orderID).Return(&domain.Payment{
		PaymentID: "PAY-002", OrderID: orderID, Status: domain.PaymentStatusFailed, UpdatedAt: time.Now(),
	}, nil).Times(1)
	mockKafka.EXPECT().Publish(gomock.Any(), string(domain.PaymentEventFailed), gomock.Any()).Return(nil).Times(1)

	svc := service.NewCallbackService(mockRepo, mockKafka, validator)
	handler := service.NewCallbackHandler(svc)

	body, _ := json.Marshal(service.CallbackRequest{ExternalRef: externalRef, OrderID: orderID, Status: "FAILED", Signature: sig})
	req := httptest.NewRequest(http.MethodPost, "/payments/callback", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCallback_Idempotent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)
	mockKafka := mocks.NewMockKafka_Publisher(ctrl)
	validator := newCallbackValidator()

	const (
		externalRef = "EXT-003"
		orderID     = "ORD-003"
	)
	sig := validator.Compute(buildCallbackPayload(externalRef, orderID, "SUCCESS"))

	// Already processed — return existing payment
	mockRepo.EXPECT().
		GetPaymentByExternalRef(gomock.Any(), externalRef).
		Return(&domain.Payment{
			PaymentID: "PAY-003", OrderID: orderID, Status: domain.PaymentStatusSuccess, UpdatedAt: time.Now(),
		}, nil).
		Times(1)

	// UpdatePaymentStatus and Kafka must NOT be called
	mockRepo.EXPECT().UpdatePaymentStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockKafka.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	svc := service.NewCallbackService(mockRepo, mockKafka, validator)
	handler := service.NewCallbackHandler(svc)

	body, _ := json.Marshal(service.CallbackRequest{ExternalRef: externalRef, OrderID: orderID, Status: "SUCCESS", Signature: sig})
	req := httptest.NewRequest(http.MethodPost, "/payments/callback", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCallback_InvalidSignature(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)
	mockKafka := mocks.NewMockKafka_Publisher(ctrl)
	validator := newCallbackValidator()

	// No repo or kafka calls expected
	mockRepo.EXPECT().UpdatePaymentStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockKafka.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	svc := service.NewCallbackService(mockRepo, mockKafka, validator)
	handler := service.NewCallbackHandler(svc)

	body, _ := json.Marshal(service.CallbackRequest{
		ExternalRef: "EXT-004", OrderID: "ORD-004", Status: "SUCCESS", Signature: "bad-sig",
	})
	req := httptest.NewRequest(http.MethodPost, "/payments/callback", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestCallback_RepoFailureAfterSignature(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)
	mockKafka := mocks.NewMockKafka_Publisher(ctrl)
	validator := newCallbackValidator()

	const (
		externalRef = "EXT-005"
		orderID     = "ORD-005"
	)
	sig := validator.Compute(buildCallbackPayload(externalRef, orderID, "SUCCESS"))

	mockRepo.EXPECT().GetPaymentByExternalRef(gomock.Any(), externalRef).Return(nil, sql.ErrNoRows).Times(1)
	mockRepo.EXPECT().
		UpdatePaymentStatus(gomock.Any(), orderID, domain.PaymentStatusSuccess, externalRef).
		Return(fmt.Errorf("db error")).
		Times(1)

	// Kafka must NOT be called when repo fails
	mockKafka.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	svc := service.NewCallbackService(mockRepo, mockKafka, validator)
	handler := service.NewCallbackHandler(svc)

	body, _ := json.Marshal(service.CallbackRequest{ExternalRef: externalRef, OrderID: orderID, Status: "SUCCESS", Signature: sig})
	req := httptest.NewRequest(http.MethodPost, "/payments/callback", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCallback_KafkaPublishFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)
	mockKafka := mocks.NewMockKafka_Publisher(ctrl)
	validator := newCallbackValidator()

	const (
		externalRef = "EXT-006"
		orderID     = "ORD-006"
	)
	sig := validator.Compute(buildCallbackPayload(externalRef, orderID, "SUCCESS"))

	mockRepo.EXPECT().GetPaymentByExternalRef(gomock.Any(), externalRef).Return(nil, sql.ErrNoRows).Times(1)
	mockRepo.EXPECT().UpdatePaymentStatus(gomock.Any(), orderID, domain.PaymentStatusSuccess, externalRef).Return(nil).Times(1)
	mockRepo.EXPECT().GetPaymentByOrderID(gomock.Any(), orderID).Return(&domain.Payment{
		PaymentID: "PAY-006", OrderID: orderID, UserID: "USR-006",
		Amount: 50000, Method: domain.PaymentMethodVirtualAccount,
		Status: domain.PaymentStatusSuccess, UpdatedAt: time.Now(),
	}, nil).Times(1)
	mockKafka.EXPECT().
		Publish(gomock.Any(), string(domain.PaymentEventSuccess), gomock.Any()).
		Return(fmt.Errorf("kafka broker unavailable")).
		Times(1)

	svc := service.NewCallbackService(mockRepo, mockKafka, validator)
	handler := service.NewCallbackHandler(svc)

	body, _ := json.Marshal(service.CallbackRequest{ExternalRef: externalRef, OrderID: orderID, Status: "SUCCESS", Signature: sig})
	req := httptest.NewRequest(http.MethodPost, "/payments/callback", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCallback_UnknownStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)
	mockKafka := mocks.NewMockKafka_Publisher(ctrl)
	validator := newCallbackValidator()

	const (
		externalRef = "EXT-007"
		orderID     = "ORD-007"
	)
	sig := validator.Compute(buildCallbackPayload(externalRef, orderID, "UNKNOWN"))

	mockRepo.EXPECT().GetPaymentByExternalRef(gomock.Any(), externalRef).Return(nil, sql.ErrNoRows).Times(1)
	mockRepo.EXPECT().UpdatePaymentStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockKafka.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	svc := service.NewCallbackService(mockRepo, mockKafka, validator)
	handler := service.NewCallbackHandler(svc)

	body, _ := json.Marshal(service.CallbackRequest{ExternalRef: externalRef, OrderID: orderID, Status: "UNKNOWN", Signature: sig})
	req := httptest.NewRequest(http.MethodPost, "/payments/callback", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCallback_GetPaymentAfterUpdateNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)
	mockKafka := mocks.NewMockKafka_Publisher(ctrl)
	validator := newCallbackValidator()

	const (
		externalRef = "EXT-008"
		orderID     = "ORD-008"
	)
	sig := validator.Compute(buildCallbackPayload(externalRef, orderID, "SUCCESS"))

	mockRepo.EXPECT().GetPaymentByExternalRef(gomock.Any(), externalRef).Return(nil, sql.ErrNoRows).Times(1)
	mockRepo.EXPECT().UpdatePaymentStatus(gomock.Any(), orderID, domain.PaymentStatusSuccess, externalRef).Return(nil).Times(1)
	mockRepo.EXPECT().GetPaymentByOrderID(gomock.Any(), orderID).Return(nil, sql.ErrNoRows).Times(1)
	mockKafka.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	svc := service.NewCallbackService(mockRepo, mockKafka, validator)
	handler := service.NewCallbackHandler(svc)

	body, _ := json.Marshal(service.CallbackRequest{ExternalRef: externalRef, OrderID: orderID, Status: "SUCCESS", Signature: sig})
	req := httptest.NewRequest(http.MethodPost, "/payments/callback", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestCallback_InvalidRequestBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)
	mockKafka := mocks.NewMockKafka_Publisher(ctrl)
	validator := newCallbackValidator()

	svc := service.NewCallbackService(mockRepo, mockKafka, validator)
	handler := service.NewCallbackHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/payments/callback", bytes.NewReader([]byte("not-json")))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
