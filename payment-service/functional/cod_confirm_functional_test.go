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

// TestFunctional_CodConfirm_CreatesCodCollectionsRow verifies that a valid COD
// confirm request inserts a cod_collections row with the correct fields.
// Validates: Requirements 6.1
func TestFunctional_CodConfirm_CreatesCodCollectionsRow(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() {
		if err := tdb.Truncate(); err != nil {
			t.Logf("cleanup truncate error: %v", err)
		}
	})

	const orderID = "order-cod-confirm-001"
	now := time.Now()
	err := tdb.SeedPayment(&domain.Payment{
		PaymentID: "pay-cod-001",
		OrderID:   orderID,
		UserID:    "user-001",
		Amount:    50000,
		Method:    domain.PaymentMethodCOD,
		Status:    domain.PaymentStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("SeedPayment failed: %v", err)
	}

	repo := repository.NewPostgresRepository(tdb.DB())
	svc := service.NewCodConfirmService(repo)
	handler := service.NewCodConfirmHandler(svc)
	server := httptest.NewServer(handler)
	defer server.Close()

	reqBody := service.CodConfirmRequest{
		OrderID:         orderID,
		CourierID:       "courier-001",
		AmountCollected: 50000,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(server.URL+"/cod/confirm", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /cod/confirm failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
	}

	col, err := tdb.GetCodCollectionByOrderID(orderID)
	if err != nil {
		t.Fatalf("GetCodCollectionByOrderID(%q) failed: %v", orderID, err)
	}

	if col.OrderID != orderID {
		t.Errorf("order_id: got %q, want %q", col.OrderID, orderID)
	}
	if col.CourierID != reqBody.CourierID {
		t.Errorf("courier_id: got %q, want %q", col.CourierID, reqBody.CourierID)
	}
	if col.AmountCollected != reqBody.AmountCollected {
		t.Errorf("amount_collected: got %v, want %v", col.AmountCollected, reqBody.AmountCollected)
	}
}

// TestFunctional_CodConfirm_UpdatesPaymentStatus verifies that a valid COD
// confirm request updates the corresponding payments row to status SUCCESS.
// Validates: Requirements 6.2
func TestFunctional_CodConfirm_UpdatesPaymentStatus(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() {
		if err := tdb.Truncate(); err != nil {
			t.Logf("cleanup truncate error: %v", err)
		}
	})

	const orderID = "order-cod-confirm-002"
	now := time.Now()
	err := tdb.SeedPayment(&domain.Payment{
		PaymentID: "pay-cod-002",
		OrderID:   orderID,
		UserID:    "user-002",
		Amount:    75000,
		Method:    domain.PaymentMethodCOD,
		Status:    domain.PaymentStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("SeedPayment failed: %v", err)
	}

	repo := repository.NewPostgresRepository(tdb.DB())
	svc := service.NewCodConfirmService(repo)
	handler := service.NewCodConfirmHandler(svc)
	server := httptest.NewServer(handler)
	defer server.Close()

	reqBody := service.CodConfirmRequest{
		OrderID:         orderID,
		CourierID:       "courier-002",
		AmountCollected: 75000,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(server.URL+"/cod/confirm", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /cod/confirm failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
	}

	payment, err := tdb.GetPaymentByOrderID(orderID)
	if err != nil {
		t.Fatalf("GetPaymentByOrderID(%q) failed: %v", orderID, err)
	}

	if payment.Status != domain.PaymentStatusSuccess {
		t.Errorf("status: got %q, want %q", payment.Status, domain.PaymentStatusSuccess)
	}
}
