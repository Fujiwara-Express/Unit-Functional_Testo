//go:build functional

package functional_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"delivery-service/domain"
	"delivery-service/repository"
	"delivery-service/service"
	"delivery-service/testdb"
)

// TestFunctional_RegisterCourier_PersistsRow verifies that a valid register
// courier request creates a couriers row with correct fields and is_available=true.
// Validates: Requirements 4.1, 4.2
func TestFunctional_RegisterCourier_PersistsRow(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() {
		if err := tdb.Truncate(); err != nil {
			t.Logf("cleanup truncate error: %v", err)
		}
	})

	repo := repository.NewPostgresRepository(tdb.DB())
	svc := service.NewCourierService(repo)
	handler := service.NewRegisterCourierHandler(svc)
	server := httptest.NewServer(handler)
	defer server.Close()

	reqBody := domain.RegisterCourierRequest{
		Name:        "Budi Santoso",
		Phone:       "081234567890",
		HubID:       "HUB-01",
		VehicleType: domain.VehicleTypeMotor,
	}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(server.URL+"/delivery/couriers", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /delivery/couriers failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected HTTP 201, got %d", resp.StatusCode)
	}

	var respBody map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	courierID := respBody["courier_id"]
	if courierID == "" {
		t.Fatal("response missing courier_id")
	}

	// Query DB directly and verify persisted fields
	courier, err := tdb.GetCourierByID(courierID)
	if err != nil {
		t.Fatalf("GetCourierByID(%q) failed: %v", courierID, err)
	}

	if courier.Name != reqBody.Name {
		t.Errorf("name: got %q, want %q", courier.Name, reqBody.Name)
	}
	if courier.Phone != reqBody.Phone {
		t.Errorf("phone: got %q, want %q", courier.Phone, reqBody.Phone)
	}
	if courier.HubID != reqBody.HubID {
		t.Errorf("hub_id: got %q, want %q", courier.HubID, reqBody.HubID)
	}
	if courier.VehicleType != reqBody.VehicleType {
		t.Errorf("vehicle_type: got %q, want %q", courier.VehicleType, reqBody.VehicleType)
	}
	if !courier.IsAvailable {
		t.Error("is_available: expected true by default, got false")
	}
}

// TestFunctional_UpdateCourier_PersistsFieldChanges verifies that a valid update
// courier request persists is_available, current_lat, and current_lng changes.
// Validates: Requirements 6.1
func TestFunctional_UpdateCourier_PersistsFieldChanges(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() {
		if err := tdb.Truncate(); err != nil {
			t.Logf("cleanup truncate error: %v", err)
		}
	})

	// Seed a courier
	seeded := &domain.Courier{
		CourierID:   "CR-FUNC-001",
		Name:        "Sari Dewi",
		Phone:       "089876543210",
		HubID:       "HUB-02",
		VehicleType: domain.VehicleTypeMotor,
		IsAvailable: true,
		CurrentLat:  0,
		CurrentLng:  0,
	}
	if err := tdb.SeedCourier(seeded); err != nil {
		t.Fatalf("SeedCourier failed: %v", err)
	}

	repo := repository.NewPostgresRepository(tdb.DB())
	svc := service.NewCourierService(repo)
	handler := service.NewUpdateCourierHandler(svc)
	server := httptest.NewServer(handler)
	defer server.Close()

	isAvail := false
	lat := -6.2088
	lng := 106.8456
	update := domain.CourierUpdate{
		IsAvailable: &isAvail,
		CurrentLat:  &lat,
		CurrentLng:  &lng,
	}
	body, _ := json.Marshal(update)

	req, _ := http.NewRequest(http.MethodPatch, server.URL+"/delivery/couriers/CR-FUNC-001", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH /delivery/couriers/CR-FUNC-001 failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
	}

	// Query DB and verify updated values
	courier, err := tdb.GetCourierByID("CR-FUNC-001")
	if err != nil {
		t.Fatalf("GetCourierByID failed: %v", err)
	}

	if courier.IsAvailable != isAvail {
		t.Errorf("is_available: got %v, want %v", courier.IsAvailable, isAvail)
	}
	if courier.CurrentLat != lat {
		t.Errorf("current_lat: got %v, want %v", courier.CurrentLat, lat)
	}
	if courier.CurrentLng != lng {
		t.Errorf("current_lng: got %v, want %v", courier.CurrentLng, lng)
	}
}

// TestFunctional_ListCouriers_HubIDFilter verifies that the hub_id filter returns
// only couriers belonging to that hub.
// Validates: Requirements 8.1
func TestFunctional_ListCouriers_HubIDFilter(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() {
		if err := tdb.Truncate(); err != nil {
			t.Logf("cleanup truncate error: %v", err)
		}
	})

	// Seed couriers across two hubs
	couriers := []*domain.Courier{
		{CourierID: "CR-HUB1-001", Name: "A", Phone: "001", HubID: "HUB-01", VehicleType: domain.VehicleTypeMotor, IsAvailable: true},
		{CourierID: "CR-HUB1-002", Name: "B", Phone: "002", HubID: "HUB-01", VehicleType: domain.VehicleTypeMotor, IsAvailable: true},
		{CourierID: "CR-HUB2-001", Name: "C", Phone: "003", HubID: "HUB-02", VehicleType: domain.VehicleTypeMotor, IsAvailable: true},
	}
	for _, c := range couriers {
		if err := tdb.SeedCourier(c); err != nil {
			t.Fatalf("SeedCourier(%q) failed: %v", c.CourierID, err)
		}
	}

	repo := repository.NewPostgresRepository(tdb.DB())
	svc := service.NewCourierService(repo)
	handler := service.NewListCouriersHandler(svc)
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/delivery/couriers?hub_id=HUB-01")
	if err != nil {
		t.Fatalf("GET /delivery/couriers?hub_id=HUB-01 failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
	}

	var result []*domain.Courier
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 couriers for HUB-01, got %d", len(result))
	}
	for _, c := range result {
		if c.HubID != "HUB-01" {
			t.Errorf("courier %q has hub_id %q, want HUB-01", c.CourierID, c.HubID)
		}
	}
}

// TestFunctional_ListCouriers_IsAvailableFilter verifies that the is_available
// filter returns only available couriers.
// Validates: Requirements 8.2
func TestFunctional_ListCouriers_IsAvailableFilter(t *testing.T) {
	tdb := testdb.Setup(t)
	t.Cleanup(func() {
		if err := tdb.Truncate(); err != nil {
			t.Logf("cleanup truncate error: %v", err)
		}
	})

	// Seed mix of available and unavailable couriers
	couriers := []*domain.Courier{
		{CourierID: "CR-AVAIL-001", Name: "A", Phone: "001", HubID: "HUB-01", VehicleType: domain.VehicleTypeMotor, IsAvailable: true},
		{CourierID: "CR-AVAIL-002", Name: "B", Phone: "002", HubID: "HUB-01", VehicleType: domain.VehicleTypeMotor, IsAvailable: true},
		{CourierID: "CR-UNAVAIL-001", Name: "C", Phone: "003", HubID: "HUB-01", VehicleType: domain.VehicleTypeMotor, IsAvailable: false},
	}
	for _, c := range couriers {
		if err := tdb.SeedCourier(c); err != nil {
			t.Fatalf("SeedCourier(%q) failed: %v", c.CourierID, err)
		}
	}

	repo := repository.NewPostgresRepository(tdb.DB())
	svc := service.NewCourierService(repo)
	handler := service.NewListCouriersHandler(svc)
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/delivery/couriers?is_available=true")
	if err != nil {
		t.Fatalf("GET /delivery/couriers?is_available=true failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
	}

	var result []*domain.Courier
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 available couriers, got %d", len(result))
	}
	for _, c := range result {
		if !c.IsAvailable {
			t.Errorf("courier %q has is_available=false, expected true", c.CourierID)
		}
	}
}
