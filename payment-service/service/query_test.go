package service_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"payment-service/domain"
	"payment-service/mocks"
	"payment-service/service"

	"go.uber.org/mock/gomock"
)

// --- GetPaymentByID tests ---

func TestGetPaymentByID_Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)
	mockRepo.EXPECT().
		GetPaymentByID(gomock.Any(), "PAY-001").
		Return(&domain.Payment{
			PaymentID: "PAY-001",
			OrderID:   "ORD-001",
			UserID:    "USR-001",
			Amount:    50000,
			Method:    domain.PaymentMethodTransfer,
			Status:    domain.PaymentStatusPending,
		}, nil).
		Times(1)

	handler := service.NewGetPaymentByIDHandler(mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/payments/PAY-001", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["payment_id"] != "PAY-001" {
		t.Errorf("expected payment_id PAY-001, got %v", resp["payment_id"])
	}
}

func TestGetPaymentByID_MissingID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)
	mockRepo.EXPECT().GetPaymentByID(gomock.Any(), gomock.Any()).Times(0)

	handler := service.NewGetPaymentByIDHandler(mockRepo)

	// Path with no ID after /payments/
	req := httptest.NewRequest(http.MethodGet, "/payments/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetPaymentByID_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)
	mockRepo.EXPECT().
		GetPaymentByID(gomock.Any(), "PAY-MISSING").
		Return(nil, sql.ErrNoRows).
		Times(1)

	handler := service.NewGetPaymentByIDHandler(mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/payments/PAY-MISSING", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// --- GetPaymentByOrderID tests ---

func TestGetPaymentByOrderID_Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)
	mockRepo.EXPECT().
		GetPaymentByOrderID(gomock.Any(), "ORD-001").
		Return(&domain.Payment{
			PaymentID: "PAY-001",
			OrderID:   "ORD-001",
			Status:    domain.PaymentStatusSuccess,
			Method:    domain.PaymentMethodVirtualAccount,
			Amount:    75000,
		}, nil).
		Times(1)

	handler := service.NewGetPaymentByOrderIDHandler(mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/payments?order_id=ORD-001", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["order_id"] != "ORD-001" {
		t.Errorf("expected order_id ORD-001, got %v", resp["order_id"])
	}
}

func TestGetPaymentByOrderID_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)
	mockRepo.EXPECT().
		GetPaymentByOrderID(gomock.Any(), "ORD-MISSING").
		Return(nil, sql.ErrNoRows).
		Times(1)

	handler := service.NewGetPaymentByOrderIDHandler(mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/payments?order_id=ORD-MISSING", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestGetPaymentByOrderID_MissingParam(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockPayment_Repository(ctrl)
	mockRepo.EXPECT().GetPaymentByOrderID(gomock.Any(), gomock.Any()).Times(0)

	handler := service.NewGetPaymentByOrderIDHandler(mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/payments", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
