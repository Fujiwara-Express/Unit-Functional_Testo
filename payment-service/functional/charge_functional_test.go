//go:build functional

package functional_test

import (
	"bytes"
	"context"
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

// stubGateway is a minimal in-process Payment_Gateway_Client for functional tests.
type stubGateway struct{}

func (s *stubGateway) Charge(_ context.Context, req *domain.ChargeRequest) (*domain.ChargeResponse, error) {
	return &domain.ChargeResponse{
		ExternalRef: "ext-ref-" + req.OrderID,
		Status:      "PENDING",
		VANumber:    "8881234567890",
		ExpiredAt:   time.Now().Add(24 * time.Hour),
	}, nil
}

// TestFunctional_Charge_CreatesPaymentsRow verifies that a valid charge request
// results in a payments row with the correct fields.
// Validates: Requirements 4.1
func TestFunctional_Charge_CreatesPaymentsRow(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() {
		if err := tdb.Truncate(); err != nil {
			t.Logf("cleanup truncate error: %v", err)
		}
	})

	repo := repository.NewPostgresRepository(tdb.DB())
	gw := &stubGateway{}
	svc := service.NewChargeService(repo, gw)
	handler := service.NewChargeHandler(svc)
	server := httptest.NewServer(handler)
	defer server.Close()

	tests := []struct {
		name           string
		req            domain.ChargeRequest
		wantStatus     domain.PaymentStatus
		wantStatusAlt  domain.PaymentStatus // for methods that may return PENDING or SUCCESS
	}{
		{
			name: "TRANSFER creates PENDING payment",
			req: domain.ChargeRequest{
				OrderID: "order-transfer-001",
				UserID:  "user-001",
				Amount:  150000,
				Method:  domain.PaymentMethodTransfer,
			},
			wantStatus: domain.PaymentStatusPending,
		},
		{
			name: "COD creates SUCCESS payment",
			req: domain.ChargeRequest{
				OrderID: "order-cod-001",
				UserID:  "user-002",
				Amount:  75000,
				Method:  domain.PaymentMethodCOD,
			},
			wantStatus: domain.PaymentStatusSuccess,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(tc.req)
			if err != nil {
				t.Fatalf("failed to marshal request: %v", err)
			}

			resp, err := http.Post(server.URL+"/charge", "application/json", bytes.NewReader(body))
			if err != nil {
				t.Fatalf("POST /charge failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
			}

			payment, err := tdb.GetPaymentByOrderID(tc.req.OrderID)
			if err != nil {
				t.Fatalf("GetPaymentByOrderID(%q) failed: %v", tc.req.OrderID, err)
			}

			if payment.OrderID != tc.req.OrderID {
				t.Errorf("order_id: got %q, want %q", payment.OrderID, tc.req.OrderID)
			}
			if payment.UserID != tc.req.UserID {
				t.Errorf("user_id: got %q, want %q", payment.UserID, tc.req.UserID)
			}
			if payment.Amount != tc.req.Amount {
				t.Errorf("amount: got %v, want %v", payment.Amount, tc.req.Amount)
			}
			if payment.Method != tc.req.Method {
				t.Errorf("method: got %q, want %q", payment.Method, tc.req.Method)
			}

			wantStatuses := []domain.PaymentStatus{tc.wantStatus}
			if tc.wantStatusAlt != "" {
				wantStatuses = append(wantStatuses, tc.wantStatusAlt)
			}
			statusOK := false
			for _, s := range wantStatuses {
				if payment.Status == s {
					statusOK = true
					break
				}
			}
			if !statusOK {
				t.Errorf("status: got %q, want one of %v", payment.Status, wantStatuses)
			}
		})
	}
}

// TestFunctional_Charge_DuplicateOrderID verifies that POSTing a charge with an
// already-existing order_id returns HTTP 409 and leaves exactly one payments row.
// Validates: Requirements 4.2
func TestFunctional_Charge_DuplicateOrderID(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() {
		if err := tdb.Truncate(); err != nil {
			t.Logf("cleanup truncate error: %v", err)
		}
	})

	const orderID = "order-dup-001"

	// Seed an existing payment with the target order_id.
	now := time.Now()
	err := tdb.SeedPayment(&domain.Payment{
		PaymentID: "pay-existing-001",
		OrderID:   orderID,
		UserID:    "user-001",
		Amount:    100000,
		Method:    domain.PaymentMethodTransfer,
		Status:    domain.PaymentStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("SeedPayment failed: %v", err)
	}

	repo := repository.NewPostgresRepository(tdb.DB())
	gw := &stubGateway{}
	svc := service.NewChargeService(repo, gw)
	handler := service.NewChargeHandler(svc)
	server := httptest.NewServer(handler)
	defer server.Close()

	// POST a duplicate charge with the same order_id.
	req := domain.ChargeRequest{
		OrderID: orderID,
		UserID:  "user-001",
		Amount:  100000,
		Method:  domain.PaymentMethodTransfer,
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(server.URL+"/charge", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /charge failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected HTTP 409, got %d", resp.StatusCode)
	}

	// Verify exactly one row exists for this order_id.
	payment, err := tdb.GetPaymentByOrderID(orderID)
	if err != nil {
		t.Fatalf("GetPaymentByOrderID(%q) failed: %v", orderID, err)
	}
	if payment.PaymentID != "pay-existing-001" {
		t.Errorf("expected original payment_id %q, got %q", "pay-existing-001", payment.PaymentID)
	}
}
