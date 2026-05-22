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

func TestRefund_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)

	mockRepo.EXPECT().
		GetPaymentByOrderID(gomock.Any(), "ORD-001").
		Return(&domain.Payment{
			PaymentID: "PAY-001",
			OrderID:   "ORD-001",
			Status:    domain.PaymentStatusSuccess,
		}, nil).
		Times(1)

	mockRepo.EXPECT().
		UpdatePaymentStatus(gomock.Any(), "ORD-001", domain.PaymentStatusRefunded, "").
		Return(nil).
		Times(1)

	svc := service.NewRefundService(mockRepo)
	handler := service.NewRefundHandler(svc)

	body, _ := json.Marshal(service.RefundRequest{OrderID: "ORD-001", Reason: "ORDER_CANCELLED"})
	req := httptest.NewRequest(http.MethodPost, "/payments/refund", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRefund_IneligibleStatuses(t *testing.T) {
	ineligible := []domain.PaymentStatus{
		domain.PaymentStatusPending,
		domain.PaymentStatusFailed,
		domain.PaymentStatusRefunded,
	}

	for _, status := range ineligible {
		t.Run(string(status), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockPayment_Repository(ctrl)

			mockRepo.EXPECT().
				GetPaymentByOrderID(gomock.Any(), "ORD-001").
				Return(&domain.Payment{
					PaymentID: "PAY-001",
					OrderID:   "ORD-001",
					Status:    status,
				}, nil).
				Times(1)

			// UpdatePaymentStatus must NOT be called
			mockRepo.EXPECT().
				UpdatePaymentStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Times(0)

			svc := service.NewRefundService(mockRepo)
			handler := service.NewRefundHandler(svc)

			body, _ := json.Marshal(service.RefundRequest{OrderID: "ORD-001"})
			req := httptest.NewRequest(http.MethodPost, "/payments/refund", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status %s: expected 422, got %d", status, rec.Code)
			}
		})
	}
}

func TestRefund_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)

	mockRepo.EXPECT().
		GetPaymentByOrderID(gomock.Any(), "ORD-MISSING").
		Return(nil, sql.ErrNoRows).
		Times(1)

	svc := service.NewRefundService(mockRepo)
	handler := service.NewRefundHandler(svc)

	body, _ := json.Marshal(service.RefundRequest{OrderID: "ORD-MISSING"})
	req := httptest.NewRequest(http.MethodPost, "/payments/refund", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestRefund_InvalidRequestBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)
	mockRepo.EXPECT().GetPaymentByOrderID(gomock.Any(), gomock.Any()).Times(0)

	svc := service.NewRefundService(mockRepo)
	handler := service.NewRefundHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/payments/refund", bytes.NewReader([]byte("not-json")))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestRefund_UpdateStatusError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)

	mockRepo.EXPECT().
		GetPaymentByOrderID(gomock.Any(), "ORD-002").
		Return(&domain.Payment{
			PaymentID: "PAY-002",
			OrderID:   "ORD-002",
			Status:    domain.PaymentStatusSuccess,
		}, nil).
		Times(1)

	mockRepo.EXPECT().
		UpdatePaymentStatus(gomock.Any(), "ORD-002", domain.PaymentStatusRefunded, "").
		Return(fmt.Errorf("db error")).
		Times(1)

	svc := service.NewRefundService(mockRepo)
	handler := service.NewRefundHandler(svc)

	body, _ := json.Marshal(service.RefundRequest{OrderID: "ORD-002"})
	req := httptest.NewRequest(http.MethodPost, "/payments/refund", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestRefund_MissingOrderID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)
	// No repo calls expected
	mockRepo.EXPECT().GetPaymentByOrderID(gomock.Any(), gomock.Any()).Times(0)

	svc := service.NewRefundService(mockRepo)
	handler := service.NewRefundHandler(svc)

	body, _ := json.Marshal(service.RefundRequest{Reason: "ORDER_CANCELLED"})
	req := httptest.NewRequest(http.MethodPost, "/payments/refund", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
