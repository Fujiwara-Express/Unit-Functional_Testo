package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pickup-service/internal/domain"
	httphandler "github.com/pickup-service/internal/handler/http"
	"github.com/pickup-service/internal/handler/http/middleware"
	"github.com/pickup-service/internal/service"
	"github.com/pickup-service/test/mocks"
	"github.com/pickup-service/test/unit/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"
)

// injectRequestID injects a request_id into the request context (simulating middleware).
func injectRequestID(req *http.Request, id string) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.RequestIDKey, id)
	return req.WithContext(ctx)
}

// setPathValue sets a path value on the request using Go 1.22 r.SetPathValue.
func setPathValue(req *http.Request, key, value string) *http.Request {
	req.SetPathValue(key, value)
	return req
}

// --- 12.1 ---

func TestPickupHandler_RequestPickup_ValidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)
	handler := httphandler.NewPickupHandler(mockSvc)

	now := time.Now()
	mockSvc.EXPECT().RequestPickup(gomock.Any(), gomock.Any()).Return(&service.RequestPickupOutput{
		PickupID:            "pickup-123",
		OrderID:             "order-456",
		Status:              domain.StatusScheduled,
		EstimatedPickupTime: now,
		CreatedAt:           now,
	}, nil)

	body := `{"order_id":"order-456","user_id":"user-789","pickup_address":"123 Main St","pickup_city_code":"JKT","contact_name":"John Doe","contact_phone":"+62812345678"}`
	req := httptest.NewRequest(http.MethodPost, "/pickups", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	w := httptest.NewRecorder()

	handler.RequestPickup(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "pickup-123", resp["pickup_id"])
	assert.Equal(t, "order-456", resp["order_id"])
	assert.Equal(t, "SCHEDULED", resp["status"])
	assert.NotEmpty(t, resp["estimated_pickup_time"])
	assert.NotEmpty(t, resp["created_at"])
}

// --- 12.2 ---

func TestPickupHandler_RequestPickup_MissingRequiredField(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		missingField string
	}{
		{
			name:        "missing_order_id",
			body:        `{"user_id":"user-789","pickup_address":"123 Main St","pickup_city_code":"JKT","contact_name":"John Doe","contact_phone":"+62812345678"}`,
			missingField: "order_id",
		},
		{
			name:        "missing_user_id",
			body:        `{"order_id":"order-456","pickup_address":"123 Main St","pickup_city_code":"JKT","contact_name":"John Doe","contact_phone":"+62812345678"}`,
			missingField: "user_id",
		},
		{
			name:        "missing_pickup_address",
			body:        `{"order_id":"order-456","user_id":"user-789","pickup_city_code":"JKT","contact_name":"John Doe","contact_phone":"+62812345678"}`,
			missingField: "pickup_address",
		},
		{
			name:        "missing_pickup_city_code",
			body:        `{"order_id":"order-456","user_id":"user-789","pickup_address":"123 Main St","contact_name":"John Doe","contact_phone":"+62812345678"}`,
			missingField: "pickup_city_code",
		},
		{
			name:        "missing_contact_name",
			body:        `{"order_id":"order-456","user_id":"user-789","pickup_address":"123 Main St","pickup_city_code":"JKT","contact_phone":"+62812345678"}`,
			missingField: "contact_name",
		},
		{
			name:        "missing_contact_phone",
			body:        `{"order_id":"order-456","user_id":"user-789","pickup_address":"123 Main St","pickup_city_code":"JKT","contact_name":"John Doe"}`,
			missingField: "contact_phone",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockSvc := mocks.NewMockPickupService(ctrl)
			handler := httphandler.NewPickupHandler(mockSvc)

			req := httptest.NewRequest(http.MethodPost, "/pickups", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req = injectRequestID(req, "test-request-id")
			w := httptest.NewRecorder()

			handler.RequestPickup(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			var resp map[string]interface{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.NotEmpty(t, resp["code"])
			assert.NotEmpty(t, resp["message"])
			assert.NotEmpty(t, resp["request_id"])
			assert.NotEmpty(t, resp["timestamp"])
		})
	}
}

// --- 12.3 ---

func TestPickupHandler_AssignCourier_ValidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)
	handler := httphandler.NewPickupHandler(mockSvc)

	mockSvc.EXPECT().AssignCourier(gomock.Any(), "pickup-123", "courier-456").Return(&service.AssignCourierOutput{
		PickupID:  "pickup-123",
		CourierID: "courier-456",
		Status:    domain.StatusAssigned,
	}, nil)

	body := `{"courier_id":"courier-456"}`
	req := httptest.NewRequest(http.MethodPost, "/pickups/pickup-123/assign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	req = setPathValue(req, "pickup_id", "pickup-123")
	w := httptest.NewRecorder()

	handler.AssignCourier(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "pickup-123", resp["pickup_id"])
	assert.Equal(t, "courier-456", resp["courier_id"])
	assert.Equal(t, "ASSIGNED", resp["status"])
}

func TestPickupHandler_AssignCourier_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)
	handler := httphandler.NewPickupHandler(mockSvc)

	mockSvc.EXPECT().AssignCourier(gomock.Any(), "nonexistent", gomock.Any()).Return(nil, domain.ErrNotFound)

	body := `{"courier_id":"courier-456"}`
	req := httptest.NewRequest(http.MethodPost, "/pickups/nonexistent/assign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	req = setPathValue(req, "pickup_id", "nonexistent")
	w := httptest.NewRecorder()

	handler.AssignCourier(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["code"])
	assert.NotEmpty(t, resp["message"])
	assert.NotEmpty(t, resp["request_id"])
	assert.NotEmpty(t, resp["timestamp"])
}

// --- 12.4 ---

func TestPickupHandler_UpdatePickupStatus_ValidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)
	handler := httphandler.NewPickupHandler(mockSvc)

	now := time.Now()
	mockSvc.EXPECT().UpdatePickupStatus(gomock.Any(), "pickup-123", domain.StatusAssigned).Return(&service.UpdateStatusOutput{
		PickupID:  "pickup-123",
		Status:    domain.StatusAssigned,
		Timestamp: now,
	}, nil)

	body := `{"status":"ASSIGNED"}`
	req := httptest.NewRequest(http.MethodPost, "/pickups/pickup-123/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	req = setPathValue(req, "pickup_id", "pickup-123")
	w := httptest.NewRecorder()

	handler.UpdatePickupStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "pickup-123", resp["pickup_id"])
	assert.Equal(t, "ASSIGNED", resp["status"])
}

func TestPickupHandler_UpdatePickupStatus_InvalidStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)
	handler := httphandler.NewPickupHandler(mockSvc)

	body := `{"status":"INVALID_STATUS"}`
	req := httptest.NewRequest(http.MethodPost, "/pickups/pickup-123/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	req = setPathValue(req, "pickup_id", "pickup-123")
	w := httptest.NewRecorder()

	handler.UpdatePickupStatus(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["code"])
	assert.NotEmpty(t, resp["message"])
	assert.NotEmpty(t, resp["request_id"])
	assert.NotEmpty(t, resp["timestamp"])
}

// --- 12.5 ---

func TestPickupHandler_GetPickup_ValidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)
	handler := httphandler.NewPickupHandler(mockSvc)

	expected := fixtures.ValidPickup()
	mockSvc.EXPECT().GetPickup(gomock.Any(), "pickup-123").Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/pickups/pickup-123", nil)
	req = injectRequestID(req, "test-request-id")
	req = setPathValue(req, "pickup_id", "pickup-123")
	w := httptest.NewRecorder()

	handler.GetPickup(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, expected.PickupID, resp["pickup_id"])
	assert.Equal(t, expected.OrderID, resp["order_id"])
	assert.Equal(t, string(expected.Status), resp["status"])
	assert.Equal(t, expected.PickupAddress, resp["pickup_address"])
	assert.Equal(t, expected.ContactName, resp["contact_name"])
}

func TestPickupHandler_GetPickup_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)
	handler := httphandler.NewPickupHandler(mockSvc)

	mockSvc.EXPECT().GetPickup(gomock.Any(), "nonexistent").Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/pickups/nonexistent", nil)
	req = injectRequestID(req, "test-request-id")
	req = setPathValue(req, "pickup_id", "nonexistent")
	w := httptest.NewRecorder()

	handler.GetPickup(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["code"])
	assert.NotEmpty(t, resp["message"])
	assert.NotEmpty(t, resp["request_id"])
	assert.NotEmpty(t, resp["timestamp"])
}

// --- 12.6 ---

func TestPickupHandler_ListPickups_WithQueryParams(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)
	handler := httphandler.NewPickupHandler(mockSvc)

	expected := []*domain.Pickup{fixtures.ValidPickup()}
	mockSvc.EXPECT().ListPickups(gomock.Any(), service.ListFilters{
		CourierID: "courier-456",
		Status:    "SCHEDULED",
		Date:      "2024-01-15",
	}).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/pickups?courier_id=courier-456&status=SCHEDULED&date=2024-01-15", nil)
	req = injectRequestID(req, "test-request-id")
	w := httptest.NewRecorder()

	handler.ListPickups(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
}

// --- 12.7 ---

func TestPickupHandler_CancelPickup_Scheduled(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)
	handler := httphandler.NewPickupHandler(mockSvc)

	mockSvc.EXPECT().CancelPickup(gomock.Any(), "pickup-123").Return(&service.CancelPickupOutput{
		PickupID: "pickup-123",
		Status:   domain.StatusCancelled,
	}, nil)

	req := httptest.NewRequest(http.MethodPost, "/pickups/pickup-123/cancel", nil)
	req = injectRequestID(req, "test-request-id")
	req = setPathValue(req, "pickup_id", "pickup-123")
	w := httptest.NewRecorder()

	handler.CancelPickup(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "pickup-123", resp["pickup_id"])
	assert.Equal(t, "CANCELLED", resp["status"])
}

func TestPickupHandler_CancelPickup_Conflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)
	handler := httphandler.NewPickupHandler(mockSvc)

	mockSvc.EXPECT().CancelPickup(gomock.Any(), "pickup-123").Return(nil, domain.ErrConflict)

	req := httptest.NewRequest(http.MethodPost, "/pickups/pickup-123/cancel", nil)
	req = injectRequestID(req, "test-request-id")
	req = setPathValue(req, "pickup_id", "pickup-123")
	w := httptest.NewRecorder()

	handler.CancelPickup(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["code"])
	assert.NotEmpty(t, resp["message"])
	assert.NotEmpty(t, resp["request_id"])
	assert.NotEmpty(t, resp["timestamp"])
}

// --- 12.8 ---

// Feature: pickup-service-unit-tests, Property 11: Handler error responses always contain non-empty request_id
func TestPickupHandler_ErrorResponse_AlwaysHasRequestID(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		ctrl := gomock.NewController(t)
		mockSvc := mocks.NewMockPickupService(ctrl)
		handler := httphandler.NewPickupHandler(mockSvc)

		// Pick a random error scenario
		scenarios := []struct {
			name    string
			setupFn func()
			makeReq func() *http.Request
		}{
			{
				name: "not_found",
				setupFn: func() {
					mockSvc.EXPECT().GetPickup(gomock.Any(), gomock.Any()).Return(nil, domain.ErrNotFound)
				},
				makeReq: func() *http.Request {
					req := httptest.NewRequest(http.MethodGet, "/pickups/some-id", nil)
					req = setPathValue(req, "pickup_id", "some-id")
					return req
				},
			},
			{
				name: "conflict",
				setupFn: func() {
					mockSvc.EXPECT().CancelPickup(gomock.Any(), gomock.Any()).Return(nil, domain.ErrConflict)
				},
				makeReq: func() *http.Request {
					req := httptest.NewRequest(http.MethodPost, "/pickups/some-id/cancel", nil)
					req = setPathValue(req, "pickup_id", "some-id")
					return req
				},
			},
			{
				name: "validation_error",
				setupFn: func() {
					// No service call expected — handler validates before calling service
				},
				makeReq: func() *http.Request {
					// Missing all required fields
					req := httptest.NewRequest(http.MethodPost, "/pickups", strings.NewReader(`{}`))
					req.Header.Set("Content-Type", "application/json")
					return req
				},
			},
		}

		idx := rapid.IntRange(0, len(scenarios)-1).Draw(rt, "scenario_idx")
		scenario := scenarios[idx]

		scenario.setupFn()
		req := scenario.makeReq()
		requestID := rapid.StringMatching(`[a-z0-9-]{8,36}`).Draw(rt, "request_id")
		req = injectRequestID(req, requestID)
		w := httptest.NewRecorder()

		switch idx {
		case 0:
			handler.GetPickup(w, req)
		case 1:
			handler.CancelPickup(w, req)
		case 2:
			handler.RequestPickup(w, req)
		}

		assert.True(t, w.Code >= 400, "expected error status code, got %d", w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		reqID, ok := resp["request_id"].(string)
		assert.True(t, ok, "request_id should be a string")
		assert.NotEmpty(t, reqID, "request_id should not be empty")
	})
}

// --- 12.9 ---

// Feature: pickup-service-unit-tests, Property 12: Handler POST /pickups missing field returns 400 with field-named error
func TestPickupHandler_RequestPickup_MissingFieldNamedInError(t *testing.T) {
	requiredFields := []struct {
		fieldName string
		body      string
	}{
		{
			fieldName: "order_id",
			body:      `{"user_id":"user-789","pickup_address":"123 Main St","pickup_city_code":"JKT","contact_name":"John Doe","contact_phone":"+62812345678"}`,
		},
		{
			fieldName: "user_id",
			body:      `{"order_id":"order-456","pickup_address":"123 Main St","pickup_city_code":"JKT","contact_name":"John Doe","contact_phone":"+62812345678"}`,
		},
		{
			fieldName: "pickup_address",
			body:      `{"order_id":"order-456","user_id":"user-789","pickup_city_code":"JKT","contact_name":"John Doe","contact_phone":"+62812345678"}`,
		},
		{
			fieldName: "pickup_city_code",
			body:      `{"order_id":"order-456","user_id":"user-789","pickup_address":"123 Main St","contact_name":"John Doe","contact_phone":"+62812345678"}`,
		},
		{
			fieldName: "contact_name",
			body:      `{"order_id":"order-456","user_id":"user-789","pickup_address":"123 Main St","pickup_city_code":"JKT","contact_phone":"+62812345678"}`,
		},
		{
			fieldName: "contact_phone",
			body:      `{"order_id":"order-456","user_id":"user-789","pickup_address":"123 Main St","pickup_city_code":"JKT","contact_name":"John Doe"}`,
		},
	}

	rapid.Check(t, func(rt *rapid.T) {
		ctrl := gomock.NewController(t)
		mockSvc := mocks.NewMockPickupService(ctrl)
		handler := httphandler.NewPickupHandler(mockSvc)

		idx := rapid.IntRange(0, len(requiredFields)-1).Draw(rt, "field_idx")
		tc := requiredFields[idx]

		req := httptest.NewRequest(http.MethodPost, "/pickups", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		req = injectRequestID(req, "test-request-id")
		w := httptest.NewRecorder()

		handler.RequestPickup(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		message, ok := resp["message"].(string)
		assert.True(t, ok, "message should be a string")
		assert.Contains(t, message, tc.fieldName, "error message should contain the missing field name")
	})
}

// --- Additional tests to improve HTTP handler coverage ---

func TestPickupHandler_RequestPickup_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)
	handler := httphandler.NewPickupHandler(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/pickups", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	w := httptest.NewRecorder()

	handler.RequestPickup(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPickupHandler_RequestPickup_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)
	handler := httphandler.NewPickupHandler(mockSvc)

	mockSvc.EXPECT().RequestPickup(gomock.Any(), gomock.Any()).Return(nil, domain.ErrServiceUnavailable)

	body := `{"order_id":"o1","user_id":"u1","pickup_address":"addr","pickup_city_code":"JKT","contact_name":"John","contact_phone":"+628123"}`
	req := httptest.NewRequest(http.MethodPost, "/pickups", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	w := httptest.NewRecorder()

	handler.RequestPickup(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestPickupHandler_RequestPickup_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)
	handler := httphandler.NewPickupHandler(mockSvc)

	mockSvc.EXPECT().RequestPickup(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

	body := `{"order_id":"o1","user_id":"u1","pickup_address":"addr","pickup_city_code":"JKT","contact_name":"John","contact_phone":"+628123"}`
	req := httptest.NewRequest(http.MethodPost, "/pickups", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	w := httptest.NewRecorder()

	handler.RequestPickup(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPickupHandler_AssignCourier_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)
	handler := httphandler.NewPickupHandler(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/pickups/pickup-123/assign", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	req = setPathValue(req, "pickup_id", "pickup-123")
	w := httptest.NewRecorder()

	handler.AssignCourier(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPickupHandler_UpdatePickupStatus_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)
	handler := httphandler.NewPickupHandler(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/pickups/pickup-123/status", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	req = setPathValue(req, "pickup_id", "pickup-123")
	w := httptest.NewRecorder()

	handler.UpdatePickupStatus(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPickupHandler_UpdatePickupStatus_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)
	handler := httphandler.NewPickupHandler(mockSvc)

	mockSvc.EXPECT().UpdatePickupStatus(gomock.Any(), "pickup-123", domain.StatusAssigned).Return(nil, domain.ErrInvalidTransition)

	body := `{"status":"ASSIGNED"}`
	req := httptest.NewRequest(http.MethodPost, "/pickups/pickup-123/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	req = setPathValue(req, "pickup_id", "pickup-123")
	w := httptest.NewRecorder()

	handler.UpdatePickupStatus(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPickupHandler_ListPickups_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)
	handler := httphandler.NewPickupHandler(mockSvc)

	mockSvc.EXPECT().ListPickups(gomock.Any(), gomock.Any()).Return(nil, domain.ErrServiceUnavailable)

	req := httptest.NewRequest(http.MethodGet, "/pickups", nil)
	req = injectRequestID(req, "test-request-id")
	w := httptest.NewRecorder()

	handler.ListPickups(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestPickupHandler_AssignCourier_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)
	handler := httphandler.NewPickupHandler(mockSvc)

	mockSvc.EXPECT().AssignCourier(gomock.Any(), "pickup-123", gomock.Any()).Return(nil, domain.ErrConflict)

	body := `{"courier_id":"courier-1"}`
	req := httptest.NewRequest(http.MethodPost, "/pickups/pickup-123/assign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	req = setPathValue(req, "pickup_id", "pickup-123")
	w := httptest.NewRecorder()

	handler.AssignCourier(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestPickupHandler_GetPickup_ValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)
	handler := httphandler.NewPickupHandler(mockSvc)

	mockSvc.EXPECT().GetPickup(gomock.Any(), "pickup-123").Return(nil, domain.ErrValidation)

	req := httptest.NewRequest(http.MethodGet, "/pickups/pickup-123", nil)
	req = injectRequestID(req, "test-request-id")
	req = setPathValue(req, "pickup_id", "pickup-123")
	w := httptest.NewRecorder()

	handler.GetPickup(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
