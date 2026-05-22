//go:build functional

package functional_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"payment-service/domain"
	"payment-service/repository"
	"payment-service/service"
	"payment-service/testdb"
)

const functionalCallbackSecret = "functional-callback-secret"

// stubKafka is a no-op Kafka publisher for functional tests.
type stubKafka struct{}

func (s *stubKafka) Publish(_ context.Context, _ string, _ *domain.PaymentEvent) error {
	return nil
}

func newFunctionalCallbackServer(tdb *testdb.TestDB) (*httptest.Server, *service.SignatureValidator) {
	repo := repository.NewPostgresRepository(tdb.DB())
	validator := service.NewSignatureValidator(functionalCallbackSecret)
	kafka := &stubKafka{}
	svc := service.NewCallbackService(repo, kafka, validator)
	handler := service.NewCallbackHandler(svc)
	return httptest.NewServer(handler), validator
}

func callbackSig(validator *service.SignatureValidator, externalRef, orderID, status string) string {
	return validator.Compute(fmt.Sprintf("%s%s%s", externalRef, orderID, status))
}

// TestFunctional_Callback_SuccessUpdatesPaymentAndExternalRef verifies that a SUCCESS
// callback updates the payments row status to SUCCESS and persists external_ref.
// Validates: Requirements 13.1
func TestFunctional_Callback_SuccessUpdatesPaymentAndExternalRef(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() { tdb.Truncate() })

	const (
		orderID     = "order-cb-001"
		externalRef = "ext-cb-001"
	)
	now := time.Now()
	if err := tdb.SeedPayment(&domain.Payment{
		PaymentID: "pay-cb-001",
		OrderID:   orderID,
		UserID:    "user-001",
		Amount:    50000,
		Method:    domain.PaymentMethodVirtualAccount,
		Status:    domain.PaymentStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("SeedPayment failed: %v", err)
	}

	server, validator := newFunctionalCallbackServer(tdb)
	defer server.Close()

	sig := callbackSig(validator, externalRef, orderID, "SUCCESS")
	body, _ := json.Marshal(service.CallbackRequest{
		ExternalRef: externalRef,
		OrderID:     orderID,
		Status:      "SUCCESS",
		Signature:   sig,
	})

	resp, err := http.Post(server.URL+"/callback", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /callback failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	payment, err := tdb.GetPaymentByOrderID(orderID)
	if err != nil {
		t.Fatalf("GetPaymentByOrderID failed: %v", err)
	}
	if payment.Status != domain.PaymentStatusSuccess {
		t.Errorf("status: got %q, want SUCCESS", payment.Status)
	}
	if payment.ExternalRef != externalRef {
		t.Errorf("external_ref: got %q, want %q", payment.ExternalRef, externalRef)
	}
}

// TestFunctional_Callback_DuplicateIsIdempotent verifies that sending the same callback
// twice leaves the payments row unchanged after the second call.
// Validates: Requirements 13.2
func TestFunctional_Callback_DuplicateIsIdempotent(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() { tdb.Truncate() })

	const (
		orderID     = "order-cb-002"
		externalRef = "ext-cb-002"
	)
	now := time.Now()
	if err := tdb.SeedPayment(&domain.Payment{
		PaymentID: "pay-cb-002",
		OrderID:   orderID,
		UserID:    "user-002",
		Amount:    75000,
		Method:    domain.PaymentMethodQRIS,
		Status:    domain.PaymentStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("SeedPayment failed: %v", err)
	}

	server, validator := newFunctionalCallbackServer(tdb)
	defer server.Close()

	sig := callbackSig(validator, externalRef, orderID, "SUCCESS")
	reqBody, _ := json.Marshal(service.CallbackRequest{
		ExternalRef: externalRef, OrderID: orderID, Status: "SUCCESS", Signature: sig,
	})

	// First call
	resp1, err := http.Post(server.URL+"/callback", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("first POST failed: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first call: expected 200, got %d", resp1.StatusCode)
	}

	// Second call (duplicate)
	resp2, err := http.Post(server.URL+"/callback", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("second POST failed: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("second call: expected 200, got %d", resp2.StatusCode)
	}

	// Payment row should still be SUCCESS and unchanged
	payment, err := tdb.GetPaymentByOrderID(orderID)
	if err != nil {
		t.Fatalf("GetPaymentByOrderID failed: %v", err)
	}
	if payment.Status != domain.PaymentStatusSuccess {
		t.Errorf("status: got %q, want SUCCESS", payment.Status)
	}
}

// TestFunctional_Callback_FailedUpdatesPaymentStatus verifies that a FAILED callback
// updates the payments row status to FAILED.
// Validates: Requirements 13.3
func TestFunctional_Callback_FailedUpdatesPaymentStatus(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() { tdb.Truncate() })

	const (
		orderID     = "order-cb-003"
		externalRef = "ext-cb-003"
	)
	now := time.Now()
	if err := tdb.SeedPayment(&domain.Payment{
		PaymentID: "pay-cb-003",
		OrderID:   orderID,
		UserID:    "user-003",
		Amount:    30000,
		Method:    domain.PaymentMethodTransfer,
		Status:    domain.PaymentStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("SeedPayment failed: %v", err)
	}

	server, validator := newFunctionalCallbackServer(tdb)
	defer server.Close()

	sig := callbackSig(validator, externalRef, orderID, "FAILED")
	body, _ := json.Marshal(service.CallbackRequest{
		ExternalRef: externalRef, OrderID: orderID, Status: "FAILED", Signature: sig,
	})

	resp, err := http.Post(server.URL+"/callback", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /callback failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	payment, err := tdb.GetPaymentByOrderID(orderID)
	if err != nil {
		t.Fatalf("GetPaymentByOrderID failed: %v", err)
	}
	if payment.Status != domain.PaymentStatusFailed {
		t.Errorf("status: got %q, want FAILED", payment.Status)
	}
}
