//go:build functional

package functional_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"delivery-service/domain"
	"delivery-service/repository"
	"delivery-service/service"
	"delivery-service/testdb"
)

// stubRoutingClient is an in-process Routing_Client that does NOT make real HTTP calls.
type stubRoutingClient struct{}

func (s *stubRoutingClient) GetCourierRoute(_ context.Context, _ string) (*domain.DeliveryRoute, error) {
	return &domain.DeliveryRoute{
		TotalStops:                    1,
		TotalDistanceKm:               5.0,
		EstimatedTotalDurationMinutes: 30,
		OptimizedRoute:                []domain.RouteStop{},
	}, nil
}

// TestFunctional_Assign_PersistsDeliveryJob verifies that a valid assign request
// creates a delivery_jobs row with correct fields, assigned_at populated, and completed_at null.
// Validates: Requirements 10.1, 10.2
func TestFunctional_Assign_PersistsDeliveryJob(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() {
		if err := tdb.Truncate(); err != nil {
			t.Logf("cleanup truncate error: %v", err)
		}
	})

	// Seed a courier first (FK constraint)
	courier := &domain.Courier{
		CourierID:   "CR-ASSIGN-001",
		Name:        "Budi",
		Phone:       "081234567890",
		HubID:       "HUB-01",
		VehicleType: domain.VehicleTypeMotor,
		IsAvailable: true,
	}
	if err := tdb.SeedCourier(courier); err != nil {
		t.Fatalf("SeedCourier failed: %v", err)
	}

	repo := repository.NewPostgresRepository(tdb.DB())
	routing := &stubRoutingClient{}
	svc := service.NewAssignService(repo, routing)
	handler := service.NewAssignHandler(svc)
	server := httptest.NewServer(handler)
	defer server.Close()

	reqBody := domain.AssignRequest{
		TrackingNumber: "TRK-FUNC-001",
		HubID:          "HUB-01",
		CourierID:      "CR-ASSIGN-001",
	}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(server.URL+"/delivery/assign", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /delivery/assign failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
	}

	// Query DB directly and verify persisted fields
	job, err := tdb.GetDeliveryJobByTrackingNumber("TRK-FUNC-001")
	if err != nil {
		t.Fatalf("GetDeliveryJobByTrackingNumber failed: %v", err)
	}

	if job.TrackingNumber != reqBody.TrackingNumber {
		t.Errorf("tracking_number: got %q, want %q", job.TrackingNumber, reqBody.TrackingNumber)
	}
	if job.CourierID != reqBody.CourierID {
		t.Errorf("courier_id: got %q, want %q", job.CourierID, reqBody.CourierID)
	}
	if job.HubID != reqBody.HubID {
		t.Errorf("hub_id: got %q, want %q", job.HubID, reqBody.HubID)
	}
	if job.Status != domain.JobStatusOutForDelivery {
		t.Errorf("status: got %q, want OUT_FOR_DELIVERY", job.Status)
	}
	if job.AssignedAt.IsZero() {
		t.Error("assigned_at: expected non-zero, got zero")
	}
	if job.CompletedAt != nil {
		t.Errorf("completed_at: expected nil, got %v", job.CompletedAt)
	}
}
