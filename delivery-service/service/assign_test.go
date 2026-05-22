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

func TestAssign_Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRouting := mocks.NewMockRouting_Client(ctrl)

	mockRouting.EXPECT().
		GetCourierRoute(gomock.Any(), "CR-001").
		Return(&domain.DeliveryRoute{
			TotalStops:                    1,
			TotalDistanceKm:               5.0,
			EstimatedTotalDurationMinutes: 30,
			OptimizedRoute:                []domain.RouteStop{},
		}, nil).
		Times(1)

	mockRepo.EXPECT().
		CreateDeliveryJob(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	svc := service.NewAssignService(mockRepo, mockRouting)
	handler := service.NewAssignHandler(svc)

	body, _ := json.Marshal(domain.AssignRequest{
		TrackingNumber: "TRK-001",
		HubID:          "HUB-01",
		CourierID:      "CR-001",
	})

	req := httptest.NewRequest(http.MethodPost, "/delivery/assign", bytes.NewReader(body))
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

	for _, field := range []string{"delivery_id", "tracking_number", "courier_id", "status", "delivery_route"} {
		if _, ok := resp[field]; !ok {
			t.Errorf("response missing field %q", field)
		}
	}
}

func TestAssign_RoutingClientError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRouting := mocks.NewMockRouting_Client(ctrl)

	mockRouting.EXPECT().
		GetCourierRoute(gomock.Any(), "CR-001").
		Return(nil, fmt.Errorf("routing service unavailable")).
		Times(1)

	// repo must NOT be called when routing fails
	mockRepo.EXPECT().CreateDeliveryJob(gomock.Any(), gomock.Any()).Times(0)

	svc := service.NewAssignService(mockRepo, mockRouting)
	handler := service.NewAssignHandler(svc)

	body, _ := json.Marshal(domain.AssignRequest{
		TrackingNumber: "TRK-001",
		HubID:          "HUB-01",
		CourierID:      "CR-001",
	})

	req := httptest.NewRequest(http.MethodPost, "/delivery/assign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", rec.Code)
	}
}

func TestAssign_MissingFields(t *testing.T) {
	cases := []struct {
		name string
		req  domain.AssignRequest
	}{
		{"missing tracking_number", domain.AssignRequest{HubID: "HUB-01", CourierID: "CR-001"}},
		{"missing hub_id", domain.AssignRequest{TrackingNumber: "TRK-001", CourierID: "CR-001"}},
		{"missing courier_id", domain.AssignRequest{TrackingNumber: "TRK-001", HubID: "HUB-01"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockDelivery_Repository(ctrl)
			mockRouting := mocks.NewMockRouting_Client(ctrl)

			// Neither mock should be called for missing-field requests
			mockRouting.EXPECT().GetCourierRoute(gomock.Any(), gomock.Any()).Times(0)
			mockRepo.EXPECT().CreateDeliveryJob(gomock.Any(), gomock.Any()).Times(0)

			svc := service.NewAssignService(mockRepo, mockRouting)
			handler := service.NewAssignHandler(svc)

			body, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/delivery/assign", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", rec.Code)
			}
		})
	}
}

func TestAssign_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRouting := mocks.NewMockRouting_Client(ctrl)

	mockRouting.EXPECT().
		GetCourierRoute(gomock.Any(), "CR-001").
		Return(&domain.DeliveryRoute{}, nil).
		Times(1)

	mockRepo.EXPECT().
		CreateDeliveryJob(gomock.Any(), gomock.Any()).
		Return(fmt.Errorf("db error")).
		Times(1)

	svc := service.NewAssignService(mockRepo, mockRouting)
	handler := service.NewAssignHandler(svc)

	body, _ := json.Marshal(domain.AssignRequest{
		TrackingNumber: "TRK-001",
		HubID:          "HUB-01",
		CourierID:      "CR-001",
	})

	req := httptest.NewRequest(http.MethodPost, "/delivery/assign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
