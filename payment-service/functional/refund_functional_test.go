//go:build functional

package functional_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"payment-service/domain"
	"payment-service/repository"
	"payment-service/service"
	"payment-service/testdb"
)

// TestFunctional_Refund_UpdatesStatusToRefunded verifies that refunding a SUCCESS
// payment updates the payments row status to REFUNDED.
// Validates: Requirements 8.1
func TestFunctional_Refund_UpdatesStatusToRefunded(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() {
		if err := tdb.Truncate(); err != nil {
			t.Logf("cleanup truncate error: %v", err)
		}
	})

	const orderID = "order-refund-001"
	now := time.Now()
	if err := tdb.SeedPayment(&domain.Payment{
		PaymentID: "pay-refund-001",
		OrderID:   orderID,
		UserID:    "user-001",
		Amount:    100000,
		Method:    domain.PaymentMethodTransfer,
		Status:    domain.PaymentStatusSuccess,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("SeedPayment failed: %v", err)
	}

	repo := repository.NewPostgresRepository(tdb.DB())
	svc := service.NewRefundService(repo)
	handler := service.NewRefundHandler(svc)
	server := httptest.NewServer(handler)
	defer server.Close()

	body, _ := json.Marshal(service.RefundRequest{OrderID: orderID, Reason: "ORDER_CANCELLED"})
	resp, err := http.Post(server.URL+"/refund", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /refund failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
	}

	payment, err := tdb.GetPaymentByOrderID(orderID)
	if err != nil {
		t.Fatalf("GetPaymentByOrderID failed: %v", err)
	}
	if payment.Status != domain.PaymentStatusRefunded {
		t.Errorf("status: got %q, want %q", payment.Status, domain.PaymentStatusRefunded)
	}
}

// TestFunctional_Refund_NonSuccessStatusUnchanged verifies that refunding a non-SUCCESS
// payment leaves the status unchanged.
// Validates: Requirements 8.2
func TestFunctional_Refund_NonSuccessStatusUnchanged(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() {
		if err := tdb.Truncate(); err != nil {
			t.Logf("cleanup truncate error: %v", err)
		}
	})

	cases := []struct {
		name   string
		status domain.PaymentStatus
		payID  string
		ordID  string
	}{
		{"PENDING", domain.PaymentStatusPending, "pay-refund-002", "order-refund-002"},
		{"FAILED", domain.PaymentStatusFailed, "pay-refund-003", "order-refund-003"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now()
			if err := tdb.SeedPayment(&domain.Payment{
				PaymentID: tc.payID,
				OrderID:   tc.ordID,
				UserID:    "user-001",
				Amount:    50000,
				Method:    domain.PaymentMethodTransfer,
				Status:    tc.status,
				CreatedAt: now,
				UpdatedAt: now,
			}); err != nil {
				t.Fatalf("SeedPayment failed: %v", err)
			}

			repo := repository.NewPostgresRepository(tdb.DB())
			svc := service.NewRefundService(repo)
			handler := service.NewRefundHandler(svc)
			server := httptest.NewServer(handler)
			defer server.Close()

			body, _ := json.Marshal(service.RefundRequest{OrderID: tc.ordID})
			resp, err := http.Post(server.URL+"/refund", "application/json", bytes.NewReader(body))
			if err != nil {
				t.Fatalf("POST /refund failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Errorf("expected 422, got %d", resp.StatusCode)
			}

			payment, err := tdb.GetPaymentByOrderID(tc.ordID)
			if err != nil {
				t.Fatalf("GetPaymentByOrderID failed: %v", err)
			}
			if payment.Status != tc.status {
				t.Errorf("status changed: got %q, want %q", payment.Status, tc.status)
			}
		})
	}
}
