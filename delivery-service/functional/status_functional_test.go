//go:build functional

package functional_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"delivery-service/domain"
	"delivery-service/repository"
	"delivery-service/service"
	"delivery-service/testdb"
)

func seedJobForStatus(t *testing.T, tdb *testdb.TestDB, jobID, trackingNumber string, attemptCount int) {
	t.Helper()
	courier := &domain.Courier{
		CourierID:   "CR-STATUS-001",
		Name:        "Sari",
		Phone:       "089999999999",
		HubID:       "HUB-01",
		VehicleType: domain.VehicleTypeMotor,
		IsAvailable: true,
	}
	// Ignore error — courier may already exist from a previous sub-test
	_ = tdb.SeedCourier(courier)

	job := &domain.DeliveryJob{
		JobID:          jobID,
		TrackingNumber: trackingNumber,
		CourierID:      "CR-STATUS-001",
		HubID:          "HUB-01",
		Status:         domain.JobStatusOutForDelivery,
		AttemptCount:   attemptCount,
		AssignedAt:     time.Now(),
	}
	if err := tdb.SeedDeliveryJob(job); err != nil {
		t.Fatalf("SeedDeliveryJob failed: %v", err)
	}
}

// TestFunctional_StatusUpdate_Delivered verifies DELIVERED persists terminal fields.
// Validates: Requirements 12.1
func TestFunctional_StatusUpdate_Delivered(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() { _ = tdb.Truncate() })

	seedJobForStatus(t, tdb, "JOB-DEL-001", "TRK-DEL-001", 0)

	repo := repository.NewPostgresRepository(tdb.DB())
	svc := service.NewStatusService(repo)
	handler := service.NewStatusUpdateHandler(svc)
	server := httptest.NewServer(handler)
	defer server.Close()

	body, _ := json.Marshal(domain.StatusUpdateRequest{
		TrackingNumber: "TRK-DEL-001",
		CourierID:      "CR-STATUS-001",
		Status:         "DELIVERED",
		ProofPhotoURL:  "https://proof.jpg",
		RecipientName:  "Budi Santoso",
		Notes:          "left at door",
	})

	resp, err := http.Post(server.URL+"/delivery/status", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /delivery/status failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
	}

	job, err := tdb.GetDeliveryJobByTrackingNumber("TRK-DEL-001")
	if err != nil {
		t.Fatalf("GetDeliveryJobByTrackingNumber failed: %v", err)
	}

	if job.Status != domain.JobStatusDelivered {
		t.Errorf("status: got %q, want DELIVERED", job.Status)
	}
	if job.CompletedAt == nil {
		t.Error("completed_at: expected non-nil")
	}
	if job.ProofPhotoURL != "https://proof.jpg" {
		t.Errorf("proof_photo_url: got %q, want %q", job.ProofPhotoURL, "https://proof.jpg")
	}
	if job.RecipientName != "Budi Santoso" {
		t.Errorf("recipient_name: got %q, want %q", job.RecipientName, "Budi Santoso")
	}
}

// TestFunctional_StatusUpdate_FailedAttempt verifies FAILED_ATTEMPT increments attempt_count.
// Validates: Requirements 12.2
func TestFunctional_StatusUpdate_FailedAttempt(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() { _ = tdb.Truncate() })

	seedJobForStatus(t, tdb, "JOB-FAIL-001", "TRK-FAIL-001", 1)

	repo := repository.NewPostgresRepository(tdb.DB())
	svc := service.NewStatusService(repo)
	handler := service.NewStatusUpdateHandler(svc)
	server := httptest.NewServer(handler)
	defer server.Close()

	body, _ := json.Marshal(domain.StatusUpdateRequest{
		TrackingNumber: "TRK-FAIL-001",
		CourierID:      "CR-STATUS-001",
		Status:         "FAILED_ATTEMPT",
	})

	resp, err := http.Post(server.URL+"/delivery/status", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /delivery/status failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
	}

	job, err := tdb.GetDeliveryJobByTrackingNumber("TRK-FAIL-001")
	if err != nil {
		t.Fatalf("GetDeliveryJobByTrackingNumber failed: %v", err)
	}

	if job.Status != domain.JobStatusFailed {
		t.Errorf("status: got %q, want FAILED", job.Status)
	}
	if job.AttemptCount != 2 {
		t.Errorf("attempt_count: got %d, want 2 (previous 1 + 1)", job.AttemptCount)
	}
}

// TestFunctional_StatusUpdate_Returned verifies RETURNED sets completed_at.
// Validates: Requirements 12.3
func TestFunctional_StatusUpdate_Returned(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() { _ = tdb.Truncate() })

	seedJobForStatus(t, tdb, "JOB-RET-001", "TRK-RET-001", 0)

	repo := repository.NewPostgresRepository(tdb.DB())
	svc := service.NewStatusService(repo)
	handler := service.NewStatusUpdateHandler(svc)
	server := httptest.NewServer(handler)
	defer server.Close()

	body, _ := json.Marshal(domain.StatusUpdateRequest{
		TrackingNumber: "TRK-RET-001",
		CourierID:      "CR-STATUS-001",
		Status:         "RETURNED",
	})

	resp, err := http.Post(server.URL+"/delivery/status", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /delivery/status failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
	}

	job, err := tdb.GetDeliveryJobByTrackingNumber("TRK-RET-001")
	if err != nil {
		t.Fatalf("GetDeliveryJobByTrackingNumber failed: %v", err)
	}

	if job.Status != domain.JobStatusReturned {
		t.Errorf("status: got %q, want RETURNED", job.Status)
	}
	if job.CompletedAt == nil {
		t.Error("completed_at: expected non-nil")
	}
}
