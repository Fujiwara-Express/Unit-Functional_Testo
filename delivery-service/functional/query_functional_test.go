//go:build functional

package functional_test

import (
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

// TestFunctional_GetCourierJobs_ReturnsAllJobs verifies all jobs for a courier are returned.
// Validates: Requirements 14.1
func TestFunctional_GetCourierJobs_ReturnsAllJobs(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() { _ = tdb.Truncate() })

	// Seed two couriers
	for _, c := range []*domain.Courier{
		{CourierID: "CR-Q-001", Name: "A", Phone: "001", HubID: "HUB-01", VehicleType: domain.VehicleTypeMotor, IsAvailable: true},
		{CourierID: "CR-Q-002", Name: "B", Phone: "002", HubID: "HUB-01", VehicleType: domain.VehicleTypeMotor, IsAvailable: true},
	} {
		if err := tdb.SeedCourier(c); err != nil {
			t.Fatalf("SeedCourier(%q) failed: %v", c.CourierID, err)
		}
	}

	now := time.Now()
	// Seed 2 jobs for CR-Q-001 and 1 job for CR-Q-002
	jobs := []*domain.DeliveryJob{
		{JobID: "JOB-Q-001", TrackingNumber: "TRK-Q-001", CourierID: "CR-Q-001", HubID: "HUB-01", Status: domain.JobStatusOutForDelivery, AssignedAt: now},
		{JobID: "JOB-Q-002", TrackingNumber: "TRK-Q-002", CourierID: "CR-Q-001", HubID: "HUB-01", Status: domain.JobStatusDelivered, AssignedAt: now},
		{JobID: "JOB-Q-003", TrackingNumber: "TRK-Q-003", CourierID: "CR-Q-002", HubID: "HUB-01", Status: domain.JobStatusOutForDelivery, AssignedAt: now},
	}
	for _, j := range jobs {
		if err := tdb.SeedDeliveryJob(j); err != nil {
			t.Fatalf("SeedDeliveryJob(%q) failed: %v", j.JobID, err)
		}
	}

	repo := repository.NewPostgresRepository(tdb.DB())
	svc := service.NewQueryService(repo)
	handler := service.NewGetCourierJobsHandler(svc)
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/delivery/courier/CR-Q-001/jobs")
	if err != nil {
		t.Fatalf("GET /delivery/courier/CR-Q-001/jobs failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
	}

	var result []*domain.DeliveryJob
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 jobs for CR-Q-001, got %d", len(result))
	}

	for _, j := range result {
		if j.CourierID != "CR-Q-001" {
			t.Errorf("job %q has courier_id %q, want CR-Q-001", j.JobID, j.CourierID)
		}
	}

	// Verify correct tracking numbers and hub_ids
	trackingNums := map[string]bool{"TRK-Q-001": false, "TRK-Q-002": false}
	for _, j := range result {
		if _, ok := trackingNums[j.TrackingNumber]; ok {
			trackingNums[j.TrackingNumber] = true
		}
		if j.HubID != "HUB-01" {
			t.Errorf("job %q has hub_id %q, want HUB-01", j.JobID, j.HubID)
		}
	}
	for tn, found := range trackingNums {
		if !found {
			t.Errorf("expected tracking_number %q in result, not found", tn)
		}
	}
}

// TestFunctional_GetCourierJobs_EmptyForCourierWithNoJobs verifies empty array for courier with no jobs.
// Validates: Requirements 14.2
func TestFunctional_GetCourierJobs_EmptyForCourierWithNoJobs(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() { _ = tdb.Truncate() })

	if err := tdb.SeedCourier(&domain.Courier{
		CourierID:   "CR-NOJOBS-001",
		Name:        "Empty",
		Phone:       "000",
		HubID:       "HUB-01",
		VehicleType: domain.VehicleTypeMotor,
		IsAvailable: true,
	}); err != nil {
		t.Fatalf("SeedCourier failed: %v", err)
	}

	repo := repository.NewPostgresRepository(tdb.DB())
	svc := service.NewQueryService(repo)
	handler := service.NewGetCourierJobsHandler(svc)
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/delivery/courier/CR-NOJOBS-001/jobs")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
	}

	var result []*domain.DeliveryJob
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d jobs", len(result))
	}
}

// TestFunctional_GetDeliveryDetail_RoundTrip verifies all fields match the seeded row.
// Validates: Requirements 16.1
func TestFunctional_GetDeliveryDetail_RoundTrip(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() { _ = tdb.Truncate() })

	if err := tdb.SeedCourier(&domain.Courier{
		CourierID:   "CR-DETAIL-001",
		Name:        "Detail",
		Phone:       "111",
		HubID:       "HUB-01",
		VehicleType: domain.VehicleTypeMotor,
		IsAvailable: true,
	}); err != nil {
		t.Fatalf("SeedCourier failed: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	completed := now.Add(2 * time.Hour)
	seeded := &domain.DeliveryJob{
		JobID:          "JOB-DETAIL-001",
		TrackingNumber: "TRK-DETAIL-001",
		CourierID:      "CR-DETAIL-001",
		HubID:          "HUB-01",
		Status:         domain.JobStatusDelivered,
		AttemptCount:   1,
		ProofPhotoURL:  "https://proof.jpg",
		RecipientName:  "Budi",
		Notes:          "left at door",
		AssignedAt:     now,
		CompletedAt:    &completed,
	}
	if err := tdb.SeedDeliveryJob(seeded); err != nil {
		t.Fatalf("SeedDeliveryJob failed: %v", err)
	}

	repo := repository.NewPostgresRepository(tdb.DB())
	svc := service.NewQueryService(repo)
	handler := service.NewGetDeliveryDetailHandler(svc)
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/delivery/JOB-DETAIL-001")
	if err != nil {
		t.Fatalf("GET /delivery/JOB-DETAIL-001 failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	checks := map[string]interface{}{
		"delivery_id":     seeded.JobID,
		"tracking_number": seeded.TrackingNumber,
		"courier_id":      seeded.CourierID,
		"hub_id":          seeded.HubID,
		"status":          string(seeded.Status),
		"proof_photo_url": seeded.ProofPhotoURL,
		"recipient_name":  seeded.RecipientName,
		"notes":           seeded.Notes,
	}
	for field, want := range checks {
		got, ok := result[field]
		if !ok {
			t.Errorf("response missing field %q", field)
			continue
		}
		if got != want {
			t.Errorf("field %q: got %v, want %v", field, got, want)
		}
	}
	if result["completed_at"] == nil {
		t.Error("completed_at: expected non-nil")
	}
}

// TestFunctional_GetDeliveryDetail_NotFound verifies 404 for non-existent delivery_id.
// Validates: Requirements 16.2
func TestFunctional_GetDeliveryDetail_NotFound(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() { _ = tdb.Truncate() })

	repo := repository.NewPostgresRepository(tdb.DB())
	svc := service.NewQueryService(repo)
	handler := service.NewGetDeliveryDetailHandler(svc)
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/delivery/JOB-NONEXISTENT")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected HTTP 404, got %d", resp.StatusCode)
	}
}
