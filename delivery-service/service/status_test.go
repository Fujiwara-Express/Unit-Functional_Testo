package service_test

import (
	"bytes"
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

func seedJob() *domain.DeliveryJob {
	return &domain.DeliveryJob{
		JobID:          "JOB-001",
		TrackingNumber: "TRK-001",
		CourierID:      "CR-001",
		HubID:          "HUB-01",
		Status:         domain.JobStatusOutForDelivery,
		AttemptCount:   0,
		AssignedAt:     time.Now(),
	}
}

func TestStatusUpdate_Delivered(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	job := seedJob()

	mockRepo.EXPECT().
		GetDeliveryJobByTrackingNumber(gomock.Any(), "TRK-001").
		Return(job, nil).
		Times(1)

	mockRepo.EXPECT().
		UpdateDeliveryJobStatus(gomock.Any(), "JOB-001", gomock.Any()).
		DoAndReturn(func(_ interface{}, _ string, u *domain.JobStatusUpdate) error {
			if u.Status != domain.JobStatusDelivered {
				t.Errorf("expected status DELIVERED, got %s", u.Status)
			}
			if u.ProofPhotoURL != "https://proof.jpg" {
				t.Errorf("expected proof_photo_url, got %q", u.ProofPhotoURL)
			}
			if u.RecipientName != "Budi" {
				t.Errorf("expected recipient_name, got %q", u.RecipientName)
			}
			if u.CompletedAt == nil {
				t.Error("expected completed_at to be set")
			}
			return nil
		}).
		Times(1)

	svc := service.NewStatusService(mockRepo)
	handler := service.NewStatusUpdateHandler(svc)

	body, _ := json.Marshal(domain.StatusUpdateRequest{
		TrackingNumber: "TRK-001",
		CourierID:      "CR-001",
		Status:         "DELIVERED",
		ProofPhotoURL:  "https://proof.jpg",
		RecipientName:  "Budi",
		Notes:          "left at door",
	})

	req := httptest.NewRequest(http.MethodPost, "/delivery/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStatusUpdate_FailedAttempt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	job := seedJob()
	job.AttemptCount = 1

	mockRepo.EXPECT().
		GetDeliveryJobByTrackingNumber(gomock.Any(), "TRK-001").
		Return(job, nil).
		Times(1)

	mockRepo.EXPECT().
		UpdateDeliveryJobStatus(gomock.Any(), "JOB-001", gomock.Any()).
		DoAndReturn(func(_ interface{}, _ string, u *domain.JobStatusUpdate) error {
			if u.Status != domain.JobStatusFailed {
				t.Errorf("expected status FAILED, got %s", u.Status)
			}
			if u.AttemptCount != 2 {
				t.Errorf("expected attempt_count=2, got %d", u.AttemptCount)
			}
			return nil
		}).
		Times(1)

	svc := service.NewStatusService(mockRepo)
	handler := service.NewStatusUpdateHandler(svc)

	body, _ := json.Marshal(domain.StatusUpdateRequest{
		TrackingNumber: "TRK-001",
		CourierID:      "CR-001",
		Status:         "FAILED_ATTEMPT",
	})

	req := httptest.NewRequest(http.MethodPost, "/delivery/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStatusUpdate_Returned(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	job := seedJob()

	mockRepo.EXPECT().
		GetDeliveryJobByTrackingNumber(gomock.Any(), "TRK-001").
		Return(job, nil).
		Times(1)

	mockRepo.EXPECT().
		UpdateDeliveryJobStatus(gomock.Any(), "JOB-001", gomock.Any()).
		DoAndReturn(func(_ interface{}, _ string, u *domain.JobStatusUpdate) error {
			if u.Status != domain.JobStatusReturned {
				t.Errorf("expected status RETURNED, got %s", u.Status)
			}
			if u.CompletedAt == nil {
				t.Error("expected completed_at to be set")
			}
			return nil
		}).
		Times(1)

	svc := service.NewStatusService(mockRepo)
	handler := service.NewStatusUpdateHandler(svc)

	body, _ := json.Marshal(domain.StatusUpdateRequest{
		TrackingNumber: "TRK-001",
		CourierID:      "CR-001",
		Status:         "RETURNED",
	})

	req := httptest.NewRequest(http.MethodPost, "/delivery/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStatusUpdate_TrackingNumberNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)

	mockRepo.EXPECT().
		GetDeliveryJobByTrackingNumber(gomock.Any(), "TRK-MISSING").
		Return(nil, fmt.Errorf("not found")).
		Times(1)

	mockRepo.EXPECT().UpdateDeliveryJobStatus(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	svc := service.NewStatusService(mockRepo)
	handler := service.NewStatusUpdateHandler(svc)

	body, _ := json.Marshal(domain.StatusUpdateRequest{
		TrackingNumber: "TRK-MISSING",
		CourierID:      "CR-001",
		Status:         "DELIVERED",
	})

	req := httptest.NewRequest(http.MethodPost, "/delivery/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestStatusUpdate_UpdateJobError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	job := seedJob()

	mockRepo.EXPECT().
		GetDeliveryJobByTrackingNumber(gomock.Any(), "TRK-001").
		Return(job, nil).
		Times(1)

	mockRepo.EXPECT().
		UpdateDeliveryJobStatus(gomock.Any(), "JOB-001", gomock.Any()).
		Return(fmt.Errorf("db error")).
		Times(1)

	svc := service.NewStatusService(mockRepo)
	handler := service.NewStatusUpdateHandler(svc)

	body, _ := json.Marshal(domain.StatusUpdateRequest{
		TrackingNumber: "TRK-001",
		CourierID:      "CR-001",
		Status:         "DELIVERED",
		ProofPhotoURL:  "https://proof.jpg",
		RecipientName:  "Budi",
	})

	req := httptest.NewRequest(http.MethodPost, "/delivery/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestStatusUpdate_InvalidRequestBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockDelivery_Repository(ctrl)
	mockRepo.EXPECT().GetDeliveryJobByTrackingNumber(gomock.Any(), gomock.Any()).Times(0)

	svc := service.NewStatusService(mockRepo)
	handler := service.NewStatusUpdateHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/delivery/status", bytes.NewReader([]byte("not-json")))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestRoutingError_Message(t *testing.T) {
	err := &service.RoutingError{Err: fmt.Errorf("timeout")}
	if err.Error() == "" {
		t.Error("RoutingError.Error() should not be empty")
	}
}

func TestStatusUpdate_MissingFields(t *testing.T) {
	cases := []struct {
		name string
		req  domain.StatusUpdateRequest
	}{
		{"missing tracking_number", domain.StatusUpdateRequest{CourierID: "CR-001", Status: "DELIVERED"}},
		{"missing courier_id", domain.StatusUpdateRequest{TrackingNumber: "TRK-001", Status: "DELIVERED"}},
		{"missing status", domain.StatusUpdateRequest{TrackingNumber: "TRK-001", CourierID: "CR-001"}},
		{"invalid status", domain.StatusUpdateRequest{TrackingNumber: "TRK-001", CourierID: "CR-001", Status: "UNKNOWN"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockDelivery_Repository(ctrl)
			mockRepo.EXPECT().GetDeliveryJobByTrackingNumber(gomock.Any(), gomock.Any()).Times(0)
			mockRepo.EXPECT().UpdateDeliveryJobStatus(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			svc := service.NewStatusService(mockRepo)
			handler := service.NewStatusUpdateHandler(svc)

			body, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/delivery/status", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", rec.Code)
			}
		})
	}
}
