package service_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"delivery-service/domain"
	"delivery-service/mocks"
	"delivery-service/service"

	"go.uber.org/mock/gomock"
)

// ---- Get Courier Jobs ----

func TestGetCourierJobs_Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRepo.EXPECT().
		GetJobsByCourierID(gomock.Any(), "CR-001").
		Return([]*domain.DeliveryJob{
			{JobID: "JOB-001", TrackingNumber: "TRK-001", CourierID: "CR-001", HubID: "HUB-01", Status: domain.JobStatusOutForDelivery},
			{JobID: "JOB-002", TrackingNumber: "TRK-002", CourierID: "CR-001", HubID: "HUB-01", Status: domain.JobStatusDelivered},
		}, nil).
		Times(1)

	svc := service.NewQueryService(mockRepo)
	handler := service.NewGetCourierJobsHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/delivery/courier/CR-001/jobs", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(resp))
	}
}

func TestGetCourierJobs_EmptyResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRepo.EXPECT().
		GetJobsByCourierID(gomock.Any(), "CR-002").
		Return([]*domain.DeliveryJob{}, nil).
		Times(1)

	svc := service.NewQueryService(mockRepo)
	handler := service.NewGetCourierJobsHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/delivery/courier/CR-002/jobs", nil)
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

func TestGetCourierJobs_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRepo.EXPECT().
		GetJobsByCourierID(gomock.Any(), "CR-001").
		Return(nil, fmt.Errorf("db error")).
		Times(1)

	svc := service.NewQueryService(mockRepo)
	handler := service.NewGetCourierJobsHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/delivery/courier/CR-001/jobs", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

// ---- Get Delivery Detail ----

func TestGetDeliveryDetail_Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Now()
	completed := now.Add(time.Hour)

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRepo.EXPECT().
		GetDeliveryJobByID(gomock.Any(), "JOB-001").
		Return(&domain.DeliveryJob{
			JobID:          "JOB-001",
			TrackingNumber: "TRK-001",
			CourierID:      "CR-001",
			HubID:          "HUB-01",
			Status:         domain.JobStatusDelivered,
			AttemptCount:   1,
			ProofPhotoURL:  "https://proof.jpg",
			RecipientName:  "Budi",
			Notes:          "left at door",
			AssignedAt:     now,
			CompletedAt:    &completed,
		}, nil).
		Times(1)

	svc := service.NewQueryService(mockRepo)
	handler := service.NewGetDeliveryDetailHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/delivery/JOB-001", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	for _, field := range []string{
		"delivery_id", "tracking_number", "courier_id", "hub_id", "status",
		"attempt_count", "proof_photo_url", "recipient_name", "notes", "assigned_at", "completed_at",
	} {
		if _, ok := resp[field]; !ok {
			t.Errorf("response missing field %q", field)
		}
	}
}

func TestGetDeliveryDetail_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRepo.EXPECT().
		GetDeliveryJobByID(gomock.Any(), "JOB-MISSING").
		Return(nil, nil).
		Times(1)

	svc := service.NewQueryService(mockRepo)
	handler := service.NewGetDeliveryDetailHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/delivery/JOB-MISSING", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
