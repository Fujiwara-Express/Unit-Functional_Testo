package service_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"payment-service/domain"
	"payment-service/mocks"
	"payment-service/service"

	"go.uber.org/mock/gomock"
)

func TestCharge_TRANSFER(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGateway := mocks.NewMockPayment_Gateway_Client(ctrl)
	mockRepo := mocks.NewMockPayment_Repository(ctrl)

	mockGateway.EXPECT().
		Charge(gomock.Any(), gomock.Any()).
		Return(&domain.ChargeResponse{ExternalRef: "EXT-001", Status: "PENDING"}, nil).
		Times(1)

	mockRepo.EXPECT().
		CreatePayment(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	svc := service.NewChargeService(mockRepo, mockGateway)
	handler := service.NewChargeHandler(svc)

	body, _ := json.Marshal(domain.ChargeRequest{
		OrderID: "ORD-001",
		Amount:  50000,
		Method:  domain.PaymentMethodTransfer,
		UserID:  "USR-001",
	})

	req := httptest.NewRequest(http.MethodPost, "/payments/charge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	for _, field := range []string{"payment_id", "order_id", "status", "method"} {
		if _, ok := resp[field]; !ok {
			t.Errorf("response missing field %q", field)
		}
	}
}

func TestCharge_VIRTUAL_ACCOUNT(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGateway := mocks.NewMockPayment_Gateway_Client(ctrl)
	mockRepo := mocks.NewMockPayment_Repository(ctrl)

	mockGateway.EXPECT().
		Charge(gomock.Any(), gomock.Any()).
		Return(&domain.ChargeResponse{ExternalRef: "EXT-002", Status: "PENDING", VANumber: "8808123456789"}, nil).
		Times(1)

	mockRepo.EXPECT().
		CreatePayment(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	svc := service.NewChargeService(mockRepo, mockGateway)
	handler := service.NewChargeHandler(svc)

	body, _ := json.Marshal(domain.ChargeRequest{
		OrderID: "ORD-002",
		Amount:  75000,
		Method:  domain.PaymentMethodVirtualAccount,
		UserID:  "USR-002",
	})

	req := httptest.NewRequest(http.MethodPost, "/payments/charge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	for _, field := range []string{"payment_id", "order_id", "status", "method"} {
		if _, ok := resp[field]; !ok {
			t.Errorf("response missing field %q", field)
		}
	}
}

func TestCharge_QRIS(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGateway := mocks.NewMockPayment_Gateway_Client(ctrl)
	mockRepo := mocks.NewMockPayment_Repository(ctrl)

	mockGateway.EXPECT().
		Charge(gomock.Any(), gomock.Any()).
		Return(&domain.ChargeResponse{ExternalRef: "EXT-003", Status: "PENDING"}, nil).
		Times(1)

	mockRepo.EXPECT().
		CreatePayment(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	svc := service.NewChargeService(mockRepo, mockGateway)
	handler := service.NewChargeHandler(svc)

	body, _ := json.Marshal(domain.ChargeRequest{
		OrderID: "ORD-003",
		Amount:  25000,
		Method:  domain.PaymentMethodQRIS,
		UserID:  "USR-003",
	})

	req := httptest.NewRequest(http.MethodPost, "/payments/charge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	for _, field := range []string{"payment_id", "order_id", "status", "method"} {
		if _, ok := resp[field]; !ok {
			t.Errorf("response missing field %q", field)
		}
	}
}

func TestCharge_COD(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGateway := mocks.NewMockPayment_Gateway_Client(ctrl)
	mockRepo := mocks.NewMockPayment_Repository(ctrl)

	// Gateway must NOT be called for COD
	mockGateway.EXPECT().
		Charge(gomock.Any(), gomock.Any()).
		Times(0)

	// Repo must be called exactly once
	mockRepo.EXPECT().
		CreatePayment(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	svc := service.NewChargeService(mockRepo, mockGateway)
	handler := service.NewChargeHandler(svc)

	body, _ := json.Marshal(domain.ChargeRequest{
		OrderID: "ORD-004",
		Amount:  30000,
		Method:  domain.PaymentMethodCOD,
		UserID:  "USR-004",
	})

	req := httptest.NewRequest(http.MethodPost, "/payments/charge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	for _, field := range []string{"payment_id", "order_id", "status", "method"} {
		if _, ok := resp[field]; !ok {
			t.Errorf("response missing field %q", field)
		}
	}

	if resp["status"] != string(domain.PaymentStatusSuccess) {
		t.Errorf("expected status %q, got %q", domain.PaymentStatusSuccess, resp["status"])
	}
}

func TestCharge_GatewayError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGateway := mocks.NewMockPayment_Gateway_Client(ctrl)
	mockRepo := mocks.NewMockPayment_Repository(ctrl)

	mockGateway.EXPECT().
		Charge(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("gateway unavailable")).
		Times(1)

	// repo must NOT be called when gateway fails
	mockRepo.EXPECT().
		CreatePayment(gomock.Any(), gomock.Any()).
		Times(0)

	svc := service.NewChargeService(mockRepo, mockGateway)
	handler := service.NewChargeHandler(svc)

	body, _ := json.Marshal(domain.ChargeRequest{
		OrderID: "ORD-005",
		Amount:  50000,
		Method:  domain.PaymentMethodTransfer,
		UserID:  "USR-005",
	})

	req := httptest.NewRequest(http.MethodPost, "/payments/charge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", rec.Code)
	}
}

func TestCharge_DuplicateOrderID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGateway := mocks.NewMockPayment_Gateway_Client(ctrl)
	mockRepo := mocks.NewMockPayment_Repository(ctrl)

	mockGateway.EXPECT().
		Charge(gomock.Any(), gomock.Any()).
		Return(&domain.ChargeResponse{ExternalRef: "EXT-006", Status: "PENDING"}, nil).
		Times(1)

	mockRepo.EXPECT().
		CreatePayment(gomock.Any(), gomock.Any()).
		Return(&service.DuplicateOrderError{OrderID: "ORD-001"}).
		Times(1)

	svc := service.NewChargeService(mockRepo, mockGateway)
	handler := service.NewChargeHandler(svc)

	body, _ := json.Marshal(domain.ChargeRequest{
		OrderID: "ORD-001",
		Amount:  50000,
		Method:  domain.PaymentMethodTransfer,
		UserID:  "USR-001",
	})

	req := httptest.NewRequest(http.MethodPost, "/payments/charge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestCharge_UnsupportedMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGateway := mocks.NewMockPayment_Gateway_Client(ctrl)
	mockRepo := mocks.NewMockPayment_Repository(ctrl)

	mockGateway.EXPECT().Charge(gomock.Any(), gomock.Any()).Times(0)
	mockRepo.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).Times(0)

	svc := service.NewChargeService(mockRepo, mockGateway)
	handler := service.NewChargeHandler(svc)

	body, _ := json.Marshal(domain.ChargeRequest{
		OrderID: "ORD-007",
		Amount:  50000,
		Method:  "CRYPTO",
		UserID:  "USR-007",
	})

	req := httptest.NewRequest(http.MethodPost, "/payments/charge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCharge_RepoInternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGateway := mocks.NewMockPayment_Gateway_Client(ctrl)
	mockRepo := mocks.NewMockPayment_Repository(ctrl)

	mockGateway.EXPECT().
		Charge(gomock.Any(), gomock.Any()).
		Return(&domain.ChargeResponse{ExternalRef: "EXT-008", Status: "PENDING"}, nil).
		Times(1)

	mockRepo.EXPECT().
		CreatePayment(gomock.Any(), gomock.Any()).
		Return(fmt.Errorf("unexpected db error")).
		Times(1)

	svc := service.NewChargeService(mockRepo, mockGateway)
	handler := service.NewChargeHandler(svc)

	body, _ := json.Marshal(domain.ChargeRequest{
		OrderID: "ORD-008",
		Amount:  50000,
		Method:  domain.PaymentMethodTransfer,
		UserID:  "USR-008",
	})

	req := httptest.NewRequest(http.MethodPost, "/payments/charge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCharge_InvalidRequestBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGateway := mocks.NewMockPayment_Gateway_Client(ctrl)
	mockRepo := mocks.NewMockPayment_Repository(ctrl)

	svc := service.NewChargeService(mockRepo, mockGateway)
	handler := service.NewChargeHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/payments/charge", bytes.NewReader([]byte("not-json")))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCharge_ErrorMessages(t *testing.T) {
	// Ensure the Error() methods on error types return non-empty strings.
	ge := &service.GatewayError{Err: fmt.Errorf("upstream down")}
	if ge.Error() == "" {
		t.Error("GatewayError.Error() should not be empty")
	}

	ve := &service.ValidationError{Message: "bad method"}
	if ve.Error() == "" {
		t.Error("ValidationError.Error() should not be empty")
	}

	de := &service.DuplicateOrderError{OrderID: "ORD-001"}
	if de.Error() == "" {
		t.Error("DuplicateOrderError.Error() should not be empty")
	}
}

func TestCharge_MissingFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGateway := mocks.NewMockPayment_Gateway_Client(ctrl)
	mockRepo := mocks.NewMockPayment_Repository(ctrl)

	// Neither gateway nor repo should be called for missing-field requests
	mockGateway.EXPECT().Charge(gomock.Any(), gomock.Any()).Times(0)
	mockRepo.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).Times(0)

	svc := service.NewChargeService(mockRepo, mockGateway)
	handler := service.NewChargeHandler(svc)

	cases := []domain.ChargeRequest{
		{OrderID: "", Amount: 50000, Method: domain.PaymentMethodTransfer, UserID: "USR-001"},
		{OrderID: "ORD-001", Amount: 0, Method: domain.PaymentMethodTransfer, UserID: "USR-001"},
		{OrderID: "ORD-001", Amount: 50000, Method: "", UserID: "USR-001"},
		{OrderID: "ORD-001", Amount: 50000, Method: domain.PaymentMethodTransfer, UserID: ""},
	}

	for _, tc := range cases {
		body, _ := json.Marshal(tc)
		req := httptest.NewRequest(http.MethodPost, "/payments/charge", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for request %+v, got %d", tc, rec.Code)
		}
	}
}
