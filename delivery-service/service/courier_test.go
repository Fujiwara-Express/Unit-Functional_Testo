package service_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"delivery-service/domain"
	"delivery-service/mocks"
	"delivery-service/service"

	"go.uber.org/mock/gomock"
)

// ---- Register Courier ----

func TestRegisterCourier_Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRepo.EXPECT().
		CreateCourier(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	svc := service.NewCourierService(mockRepo)
	handler := service.NewRegisterCourierHandler(svc)

	body, _ := json.Marshal(domain.RegisterCourierRequest{
		Name:        "Budi Santoso",
		Phone:       "081234567890",
		HubID:       "HUB-01",
		VehicleType: domain.VehicleTypeMotor,
	})

	req := httptest.NewRequest(http.MethodPost, "/delivery/couriers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	for _, field := range []string{"courier_id", "status"} {
		if _, ok := resp[field]; !ok {
			t.Errorf("response missing field %q", field)
		}
	}
}

func TestRegisterCourier_MissingFields(t *testing.T) {
	cases := []struct {
		name string
		req  domain.RegisterCourierRequest
	}{
		{"missing name", domain.RegisterCourierRequest{Phone: "081234567890", HubID: "HUB-01", VehicleType: domain.VehicleTypeMotor}},
		{"missing phone", domain.RegisterCourierRequest{Name: "Budi", HubID: "HUB-01", VehicleType: domain.VehicleTypeMotor}},
		{"missing hub_id", domain.RegisterCourierRequest{Name: "Budi", Phone: "081234567890", VehicleType: domain.VehicleTypeMotor}},
		{"missing vehicle_type", domain.RegisterCourierRequest{Name: "Budi", Phone: "081234567890", HubID: "HUB-01"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockDelivery_Repository(ctrl)
			// repo must NOT be called for any missing-field request
			mockRepo.EXPECT().CreateCourier(gomock.Any(), gomock.Any()).Times(0)

			svc := service.NewCourierService(mockRepo)
			handler := service.NewRegisterCourierHandler(svc)

			body, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/delivery/couriers", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", rec.Code)
			}
		})
	}
}

func TestRegisterCourier_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRepo.EXPECT().
		CreateCourier(gomock.Any(), gomock.Any()).
		Return(fmt.Errorf("db connection lost")).
		Times(1)

	svc := service.NewCourierService(mockRepo)
	handler := service.NewRegisterCourierHandler(svc)

	body, _ := json.Marshal(domain.RegisterCourierRequest{
		Name:        "Budi Santoso",
		Phone:       "081234567890",
		HubID:       "HUB-01",
		VehicleType: domain.VehicleTypeMotor,
	})

	req := httptest.NewRequest(http.MethodPost, "/delivery/couriers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

// ---- Update Courier ----

func TestUpdateCourier_Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRepo.EXPECT().
		UpdateCourier(gomock.Any(), "CR-001", gomock.Any()).
		Return(nil).
		Times(1)

	svc := service.NewCourierService(mockRepo)
	handler := service.NewUpdateCourierHandler(svc)

	isAvail := true
	lat := -6.2088
	lng := 106.8456
	body, _ := json.Marshal(domain.CourierUpdate{
		IsAvailable: &isAvail,
		CurrentLat:  &lat,
		CurrentLng:  &lng,
	})

	req := httptest.NewRequest(http.MethodPatch, "/delivery/couriers/CR-001", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	for _, field := range []string{"courier_id", "status"} {
		if _, ok := resp[field]; !ok {
			t.Errorf("response missing field %q", field)
		}
	}
}

func TestUpdateCourier_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRepo.EXPECT().
		UpdateCourier(gomock.Any(), "CR-999", gomock.Any()).
		Return(&service.NotFoundError{ID: "CR-999"}).
		Times(1)

	svc := service.NewCourierService(mockRepo)
	handler := service.NewUpdateCourierHandler(svc)

	isAvail := false
	body, _ := json.Marshal(domain.CourierUpdate{IsAvailable: &isAvail})

	req := httptest.NewRequest(http.MethodPatch, "/delivery/couriers/CR-999", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestUpdateCourier_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRepo.EXPECT().
		UpdateCourier(gomock.Any(), "CR-001", gomock.Any()).
		Return(fmt.Errorf("db error")).
		Times(1)

	svc := service.NewCourierService(mockRepo)
	handler := service.NewUpdateCourierHandler(svc)

	isAvail := true
	body, _ := json.Marshal(domain.CourierUpdate{IsAvailable: &isAvail})

	req := httptest.NewRequest(http.MethodPatch, "/delivery/couriers/CR-001", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

// ---- List Couriers ----

func TestListCouriers_WithFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	isAvail := true
	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRepo.EXPECT().
		ListCouriers(gomock.Any(), &filterMatcher{hubID: "HUB-01", isAvailable: &isAvail, cityCode: "JKT"}).
		Return([]*domain.Courier{
			{CourierID: "CR-001", Name: "Budi", HubID: "HUB-01", IsAvailable: true},
		}, nil).
		Times(1)

	svc := service.NewCourierService(mockRepo)
	handler := service.NewListCouriersHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/delivery/couriers?hub_id=HUB-01&is_available=true&city_code=JKT", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("expected 1 courier, got %d", len(resp))
	}
}

func TestListCouriers_NoFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRepo.EXPECT().
		ListCouriers(gomock.Any(), gomock.Any()).
		Return([]*domain.Courier{
			{CourierID: "CR-001", Name: "Budi"},
			{CourierID: "CR-002", Name: "Sari"},
		}, nil).
		Times(1)

	svc := service.NewCourierService(mockRepo)
	handler := service.NewListCouriersHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/delivery/couriers", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("expected 2 couriers, got %d", len(resp))
	}
}

func TestListCouriers_EmptyResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRepo.EXPECT().
		ListCouriers(gomock.Any(), gomock.Any()).
		Return([]*domain.Courier{}, nil).
		Times(1)

	svc := service.NewCourierService(mockRepo)
	handler := service.NewListCouriersHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/delivery/couriers", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected empty array, got %d items", len(resp))
	}
}

func TestListCouriers_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRepo.EXPECT().
		ListCouriers(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("db error")).
		Times(1)

	svc := service.NewCourierService(mockRepo)
	handler := service.NewListCouriersHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/delivery/couriers", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestRegisterCourier_InvalidRequestBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRepo.EXPECT().CreateCourier(gomock.Any(), gomock.Any()).Times(0)

	svc := service.NewCourierService(mockRepo)
	handler := service.NewRegisterCourierHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/delivery/couriers", bytes.NewReader([]byte("not-json")))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateCourier_InvalidRequestBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRepo.EXPECT().UpdateCourier(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	svc := service.NewCourierService(mockRepo)
	handler := service.NewUpdateCourierHandler(svc)

	req := httptest.NewRequest(http.MethodPatch, "/delivery/couriers/CR-001", bytes.NewReader([]byte("not-json")))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestNotFoundError_Message(t *testing.T) {
	err := &service.NotFoundError{ID: "CR-001"}
	if err.Error() == "" {
		t.Error("NotFoundError.Error() should not be empty")
	}
}

// filterMatcher is a gomock.Matcher that checks CourierFilter fields.
type filterMatcher struct {
	hubID       string
	isAvailable *bool
	cityCode    string
}

func (m *filterMatcher) Matches(x interface{}) bool {
	f, ok := x.(*domain.CourierFilter)
	if !ok {
		return false
	}
	if f.HubID != m.hubID || f.CityCode != m.cityCode {
		return false
	}
	if m.isAvailable == nil && f.IsAvailable != nil {
		return false
	}
	if m.isAvailable != nil && (f.IsAvailable == nil || *f.IsAvailable != *m.isAvailable) {
		return false
	}
	return true
}

func (m *filterMatcher) String() string {
	return fmt.Sprintf("filter{hub_id=%q, city_code=%q}", m.hubID, m.cityCode)
}
