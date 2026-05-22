package service_test

import (
	"bytes"
	"database/sql"
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

func TestCodConfirm_Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)

	// Mock GetPaymentByOrderID to return a COD payment
	mockRepo.EXPECT().
		GetPaymentByOrderID(gomock.Any(), "ORD-COD-001").
		Return(&domain.Payment{
			PaymentID: "PAY-001",
			OrderID:   "ORD-COD-001",
			UserID:    "USR-001",
			Amount:    50000,
			Method:    domain.PaymentMethodCOD,
			Status:    domain.PaymentStatusPending,
		}, nil).
		Times(1)

	// Expect CreateCodCollection called once
	mockRepo.EXPECT().
		CreateCodCollection(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	// Expect UpdatePaymentStatus called once with SUCCESS
	mockRepo.EXPECT().
		UpdatePaymentStatus(gomock.Any(), "ORD-COD-001", domain.PaymentStatusSuccess, gomock.Any()).
		Return(nil).
		Times(1)

	svc := service.NewCodConfirmService(mockRepo)
	handler := service.NewCodConfirmHandler(svc)

	body, _ := json.Marshal(service.CodConfirmRequest{
		OrderID:         "ORD-COD-001",
		CourierID:       "CRR-001",
		AmountCollected: 50000,
	})

	req := httptest.NewRequest(http.MethodPost, "/payments/cod/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// TestCodConfirm_NotFound verifies HTTP 404 when order_id does not exist (Req 5.3).
func TestCodConfirm_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)
	mockRepo.EXPECT().
		GetPaymentByOrderID(gomock.Any(), "ORD-MISSING").
		Return(nil, sql.ErrNoRows).
		Times(1)

	svc := service.NewCodConfirmService(mockRepo)
	handler := service.NewCodConfirmHandler(svc)

	body, _ := json.Marshal(service.CodConfirmRequest{
		OrderID:         "ORD-MISSING",
		CourierID:       "CRR-001",
		AmountCollected: 50000,
	})

	req := httptest.NewRequest(http.MethodPost, "/payments/cod/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// TestCodConfirm_NonCODMethod verifies HTTP 404 when the payment method is not COD (Req 5.3).
func TestCodConfirm_NonCODMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)
	mockRepo.EXPECT().
		GetPaymentByOrderID(gomock.Any(), "ORD-TRANSFER-001").
		Return(&domain.Payment{
			PaymentID: "PAY-002",
			OrderID:   "ORD-TRANSFER-001",
			UserID:    "USR-001",
			Amount:    50000,
			Method:    domain.PaymentMethodTransfer,
			Status:    domain.PaymentStatusPending,
		}, nil).
		Times(1)

	svc := service.NewCodConfirmService(mockRepo)
	handler := service.NewCodConfirmHandler(svc)

	body, _ := json.Marshal(service.CodConfirmRequest{
		OrderID:         "ORD-TRANSFER-001",
		CourierID:       "CRR-001",
		AmountCollected: 50000,
	})

	req := httptest.NewRequest(http.MethodPost, "/payments/cod/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestCodConfirm_CreateCollectionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)

	mockRepo.EXPECT().
		GetPaymentByOrderID(gomock.Any(), "ORD-COD-002").
		Return(&domain.Payment{
			PaymentID: "PAY-002",
			OrderID:   "ORD-COD-002",
			Method:    domain.PaymentMethodCOD,
			Status:    domain.PaymentStatusPending,
		}, nil).
		Times(1)

	mockRepo.EXPECT().
		CreateCodCollection(gomock.Any(), gomock.Any()).
		Return(fmt.Errorf("db error")).
		Times(1)

	mockRepo.EXPECT().UpdatePaymentStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	svc := service.NewCodConfirmService(mockRepo)
	handler := service.NewCodConfirmHandler(svc)

	body, _ := json.Marshal(service.CodConfirmRequest{
		OrderID:         "ORD-COD-002",
		CourierID:       "CRR-001",
		AmountCollected: 50000,
	})

	req := httptest.NewRequest(http.MethodPost, "/payments/cod/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestCodConfirm_InvalidRequestBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)

	svc := service.NewCodConfirmService(mockRepo)
	handler := service.NewCodConfirmHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/payments/cod/confirm", bytes.NewReader([]byte("not-json")))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// TestCodConfirm_MissingFields verifies HTTP 400 when required fields are absent (Req 5.4).
func TestCodConfirm_MissingFields(t *testing.T) {
	cases := []struct {
		name string
		body service.CodConfirmRequest
	}{
		{"missing order_id", service.CodConfirmRequest{CourierID: "CRR-001", AmountCollected: 50000}},
		{"missing courier_id", service.CodConfirmRequest{OrderID: "ORD-COD-001", AmountCollected: 50000}},
		{"missing amount_collected", service.CodConfirmRequest{OrderID: "ORD-COD-001", CourierID: "CRR-001"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// No repo calls expected — handler should reject before reaching service.
			mockRepo := mocks.NewMockPayment_Repository(ctrl)

			svc := service.NewCodConfirmService(mockRepo)
			handler := service.NewCodConfirmHandler(svc)

			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/payments/cod/confirm", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("%s: expected 400, got %d", tc.name, rec.Code)
			}
		})
	}
}
