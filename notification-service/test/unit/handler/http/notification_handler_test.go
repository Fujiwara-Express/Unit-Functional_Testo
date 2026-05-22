package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/notification-service/internal/domain"
	httphandler "github.com/notification-service/internal/handler/http"
	"github.com/notification-service/internal/handler/http/middleware"
	"github.com/notification-service/internal/service"
	"github.com/notification-service/test/mocks"
	"github.com/notification-service/test/unit/fixtures"
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

// Validates: Requirements 4.1
func TestNotificationHandler_SendNotification_ValidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)
	handler := httphandler.NewNotificationHandler(mockSvc)

	mockSvc.EXPECT().SendNotification(gomock.Any(), gomock.Any()).Return(&service.SendNotificationOutput{
		NotificationID: "notif-123",
		Status:         domain.NotifStatusSent,
		Channel:        domain.ChannelPush,
	}, nil)

	body := `{"user_id":"user-456","channel":"PUSH","template_id":"tmpl-001","variables":{"tracking_number":"TRK123"}}`
	req := httptest.NewRequest(http.MethodPost, "/notifications/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	w := httptest.NewRecorder()

	handler.SendNotification(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "notif-123", resp["notification_id"])
	assert.Equal(t, "SENT", resp["status"])
	assert.Equal(t, "PUSH", resp["channel"])
}

// Validates: Requirements 4.2
func TestNotificationHandler_SendNotification_MissingRequiredField(t *testing.T) {
	cases := []struct {
		name         string
		body         string
		missingField string
	}{
		{
			name:         "missing_user_id",
			body:         `{"channel":"PUSH","template_id":"tmpl-001"}`,
			missingField: "user_id",
		},
		{
			name:         "missing_channel",
			body:         `{"user_id":"user-456","template_id":"tmpl-001"}`,
			missingField: "channel",
		},
		{
			name:         "missing_template_id",
			body:         `{"user_id":"user-456","channel":"PUSH"}`,
			missingField: "template_id",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockSvc := mocks.NewMockNotificationService(ctrl)
			handler := httphandler.NewNotificationHandler(mockSvc)

			req := httptest.NewRequest(http.MethodPost, "/notifications/send", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req = injectRequestID(req, "test-request-id")
			w := httptest.NewRecorder()

			handler.SendNotification(w, req)

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

// Validates: Requirements 4.3
func TestNotificationHandler_SendNotification_InvalidChannel(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)
	handler := httphandler.NewNotificationHandler(mockSvc)

	body := `{"user_id":"user-456","channel":"INVALID","template_id":"tmpl-001"}`
	req := httptest.NewRequest(http.MethodPost, "/notifications/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	w := httptest.NewRecorder()

	handler.SendNotification(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["code"])
	assert.NotEmpty(t, resp["message"])
	assert.NotEmpty(t, resp["request_id"])
	assert.NotEmpty(t, resp["timestamp"])
}

// Validates: Requirements 4.4
func TestNotificationHandler_ListTemplates_ValidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)
	handler := httphandler.NewNotificationHandler(mockSvc)

	expected := []*domain.NotificationTemplate{fixtures.ValidNotificationTemplate()}
	mockSvc.EXPECT().ListTemplates(gomock.Any()).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/notifications/templates", nil)
	req = injectRequestID(req, "test-request-id")
	w := httptest.NewRecorder()

	handler.ListTemplates(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp, 1)
	assert.NotEmpty(t, resp[0]["template_id"])
	assert.NotEmpty(t, resp[0]["event_type"])
	assert.NotEmpty(t, resp[0]["channel"])
	assert.NotEmpty(t, resp[0]["subject"])
	assert.NotEmpty(t, resp[0]["body_template"])
}

// Validates: Requirements 4.5
func TestNotificationHandler_CreateTemplate_ValidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)
	handler := httphandler.NewNotificationHandler(mockSvc)

	mockSvc.EXPECT().CreateTemplate(gomock.Any(), gomock.Any()).Return(&service.CreateTemplateOutput{
		TemplateID: "tmpl-new",
		Status:     "CREATED",
	}, nil)

	body := `{"event_type":"OUT_FOR_DELIVERY","channel":"PUSH","subject":"Package Update","body_template":"Your package {{tracking_number}} is on the way."}`
	req := httptest.NewRequest(http.MethodPost, "/notifications/templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	w := httptest.NewRecorder()

	handler.CreateTemplate(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "tmpl-new", resp["template_id"])
	assert.Equal(t, "CREATED", resp["status"])
}

// Validates: Requirements 4.6
func TestNotificationHandler_CreateTemplate_MissingField(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"missing_event_type", `{"channel":"PUSH","subject":"Subject","body_template":"Body"}`},
		{"missing_channel", `{"event_type":"OUT_FOR_DELIVERY","subject":"Subject","body_template":"Body"}`},
		{"missing_subject", `{"event_type":"OUT_FOR_DELIVERY","channel":"PUSH","body_template":"Body"}`},
		{"missing_body_template", `{"event_type":"OUT_FOR_DELIVERY","channel":"PUSH","subject":"Subject"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockSvc := mocks.NewMockNotificationService(ctrl)
			handler := httphandler.NewNotificationHandler(mockSvc)

			req := httptest.NewRequest(http.MethodPost, "/notifications/templates", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req = injectRequestID(req, "test-request-id")
			w := httptest.NewRecorder()

			handler.CreateTemplate(w, req)

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

// Validates: Requirements 4.7
func TestNotificationHandler_UpdateTemplate_ValidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)
	handler := httphandler.NewNotificationHandler(mockSvc)

	mockSvc.EXPECT().UpdateTemplate(gomock.Any(), "tmpl-001", gomock.Any()).Return(&service.UpdateTemplateOutput{
		TemplateID: "tmpl-001",
		Status:     "UPDATED",
	}, nil)

	body := `{"subject":"New Subject","body_template":"New body"}`
	req := httptest.NewRequest(http.MethodPut, "/notifications/templates/tmpl-001", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	req.SetPathValue("template_id", "tmpl-001")
	w := httptest.NewRecorder()

	handler.UpdateTemplate(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "tmpl-001", resp["template_id"])
	assert.Equal(t, "UPDATED", resp["status"])
}

// Validates: Requirements 4.8
func TestNotificationHandler_UpdateTemplate_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)
	handler := httphandler.NewNotificationHandler(mockSvc)

	mockSvc.EXPECT().UpdateTemplate(gomock.Any(), "nonexistent", gomock.Any()).Return(nil, domain.ErrNotFound)

	body := `{"subject":"New Subject","body_template":"New body"}`
	req := httptest.NewRequest(http.MethodPut, "/notifications/templates/nonexistent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	req.SetPathValue("template_id", "nonexistent")
	w := httptest.NewRecorder()

	handler.UpdateTemplate(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["code"])
	assert.NotEmpty(t, resp["message"])
	assert.NotEmpty(t, resp["request_id"])
	assert.NotEmpty(t, resp["timestamp"])
}

// Validates: Requirements 4.9
func TestNotificationHandler_ServiceError_Returns500(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)
	handler := httphandler.NewNotificationHandler(mockSvc)

	mockSvc.EXPECT().SendNotification(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

	body := `{"user_id":"user-456","channel":"PUSH","template_id":"tmpl-001"}`
	req := httptest.NewRequest(http.MethodPost, "/notifications/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	w := httptest.NewRecorder()

	handler.SendNotification(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["code"])
	assert.NotEmpty(t, resp["message"])
	assert.NotEmpty(t, resp["request_id"])
	assert.NotEmpty(t, resp["timestamp"])
}

// Validates: Requirements 4.4 — service error path
func TestNotificationHandler_ListTemplates_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)
	handler := httphandler.NewNotificationHandler(mockSvc)

	mockSvc.EXPECT().ListTemplates(gomock.Any()).Return(nil, domain.ErrServiceUnavailable)

	req := httptest.NewRequest(http.MethodGet, "/notifications/templates", nil)
	req = injectRequestID(req, "test-request-id")
	w := httptest.NewRecorder()

	handler.ListTemplates(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["code"])
	assert.NotEmpty(t, resp["request_id"])
}

// Validates: Requirements 4.4 — not found error path
func TestNotificationHandler_ListTemplates_NotFoundError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)
	handler := httphandler.NewNotificationHandler(mockSvc)

	mockSvc.EXPECT().ListTemplates(gomock.Any()).Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/notifications/templates", nil)
	req = injectRequestID(req, "test-request-id")
	w := httptest.NewRecorder()

	handler.ListTemplates(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Validates: Requirements 4.5 — service error path
func TestNotificationHandler_CreateTemplate_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)
	handler := httphandler.NewNotificationHandler(mockSvc)

	mockSvc.EXPECT().CreateTemplate(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

	body := `{"event_type":"OUT_FOR_DELIVERY","channel":"PUSH","subject":"Subject","body_template":"Body"}`
	req := httptest.NewRequest(http.MethodPost, "/notifications/templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	w := httptest.NewRecorder()

	handler.CreateTemplate(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// Validates: Requirements 4.5 — invalid JSON body
func TestNotificationHandler_CreateTemplate_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)
	handler := httphandler.NewNotificationHandler(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/notifications/templates", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	w := httptest.NewRecorder()

	handler.CreateTemplate(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Validates: Requirements 4.7 — missing template_id path param
func TestNotificationHandler_UpdateTemplate_MissingTemplateID(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)
	handler := httphandler.NewNotificationHandler(mockSvc)

	body := `{"subject":"New Subject","body_template":"New body"}`
	req := httptest.NewRequest(http.MethodPut, "/notifications/templates/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	// template_id path value intentionally not set
	w := httptest.NewRecorder()

	handler.UpdateTemplate(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Validates: Requirements 4.7 — invalid JSON body for update
func TestNotificationHandler_UpdateTemplate_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)
	handler := httphandler.NewNotificationHandler(mockSvc)

	req := httptest.NewRequest(http.MethodPut, "/notifications/templates/tmpl-001", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	req.SetPathValue("template_id", "tmpl-001")
	w := httptest.NewRecorder()

	handler.UpdateTemplate(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Validates: Requirements 4.7 — conflict error path (covers mapError ErrConflict branch)
func TestNotificationHandler_UpdateTemplate_ConflictError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)
	handler := httphandler.NewNotificationHandler(mockSvc)

	mockSvc.EXPECT().UpdateTemplate(gomock.Any(), "tmpl-001", gomock.Any()).Return(nil, domain.ErrConflict)

	body := `{"subject":"Subject","body_template":"Body"}`
	req := httptest.NewRequest(http.MethodPut, "/notifications/templates/tmpl-001", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	req.SetPathValue("template_id", "tmpl-001")
	w := httptest.NewRecorder()

	handler.UpdateTemplate(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// Validates: Requirements 4.9 — ErrValidation maps to 400 (covers mapError ErrValidation branch)
func TestNotificationHandler_SendNotification_ValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)
	handler := httphandler.NewNotificationHandler(mockSvc)

	mockSvc.EXPECT().SendNotification(gomock.Any(), gomock.Any()).Return(nil, domain.ErrValidation)

	body := `{"user_id":"user-456","channel":"PUSH","template_id":"tmpl-001"}`
	req := httptest.NewRequest(http.MethodPost, "/notifications/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	w := httptest.NewRecorder()

	handler.SendNotification(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Validates: Requirements 4.9 — ErrServiceUnavailable maps to 503 (covers mapError branch)
func TestNotificationHandler_SendNotification_ServiceUnavailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)
	handler := httphandler.NewNotificationHandler(mockSvc)

	mockSvc.EXPECT().SendNotification(gomock.Any(), gomock.Any()).Return(nil, domain.ErrServiceUnavailable)

	body := `{"user_id":"user-456","channel":"PUSH","template_id":"tmpl-001"}`
	req := httptest.NewRequest(http.MethodPost, "/notifications/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	w := httptest.NewRecorder()

	handler.SendNotification(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// Validates: Requirements 4.1 — invalid JSON body for send
func TestNotificationHandler_SendNotification_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)
	handler := httphandler.NewNotificationHandler(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/notifications/send", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	req = injectRequestID(req, "test-request-id")
	w := httptest.NewRecorder()

	handler.SendNotification(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Feature: notification-service-unit-tests, Property 11: Handler error responses always contain non-empty request_id
// Validates: Requirements 4.9, 4.11, 7.6
func TestNotificationHandler_ErrorResponse_AlwaysHasRequestID(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		ctrl := gomock.NewController(t)
		mockSvc := mocks.NewMockNotificationService(ctrl)
		handler := httphandler.NewNotificationHandler(mockSvc)

		scenarios := []struct {
			name    string
			setupFn func()
			makeReq func() *http.Request
			callFn  func(w http.ResponseWriter, r *http.Request)
		}{
			{
				name: "send_not_found",
				setupFn: func() {
					mockSvc.EXPECT().SendNotification(gomock.Any(), gomock.Any()).Return(nil, domain.ErrNotFound)
				},
				makeReq: func() *http.Request {
					req := httptest.NewRequest(http.MethodPost, "/notifications/send",
						strings.NewReader(`{"user_id":"u","channel":"PUSH","template_id":"t"}`))
					req.Header.Set("Content-Type", "application/json")
					return req
				},
				callFn: handler.SendNotification,
			},
			{
				name:    "send_missing_field",
				setupFn: func() {},
				makeReq: func() *http.Request {
					req := httptest.NewRequest(http.MethodPost, "/notifications/send",
						strings.NewReader(`{}`))
					req.Header.Set("Content-Type", "application/json")
					return req
				},
				callFn: handler.SendNotification,
			},
			{
				name: "update_not_found",
				setupFn: func() {
					mockSvc.EXPECT().UpdateTemplate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, domain.ErrNotFound)
				},
				makeReq: func() *http.Request {
					req := httptest.NewRequest(http.MethodPut, "/notifications/templates/t",
						strings.NewReader(`{"subject":"s","body_template":"b"}`))
					req.Header.Set("Content-Type", "application/json")
					req.SetPathValue("template_id", "t")
					return req
				},
				callFn: handler.UpdateTemplate,
			},
		}

		idx := rapid.IntRange(0, len(scenarios)-1).Draw(rt, "scenario_idx")
		scenario := scenarios[idx]

		scenario.setupFn()
		req := scenario.makeReq()
		requestID := rapid.StringMatching(`[a-z0-9-]{4,36}`).Draw(rt, "request_id")
		req = injectRequestID(req, requestID)
		w := httptest.NewRecorder()

		scenario.callFn(w, req)

		assert.True(t, w.Code >= 400, "expected error status code, got %d", w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		reqID, ok := resp["request_id"].(string)
		assert.True(t, ok, "request_id should be a string")
		assert.NotEmpty(t, reqID, "request_id should not be empty")
	})
}

// Feature: notification-service-unit-tests, Property 12: Handler POST /notifications/send missing field returns 400 with field-named error
// Validates: Requirements 4.2
func TestNotificationHandler_SendNotification_MissingFieldNamedInError(t *testing.T) {
	requiredFields := []struct {
		fieldName string
		body      string
	}{
		{"user_id", `{"channel":"PUSH","template_id":"tmpl-001"}`},
		{"channel", `{"user_id":"user-456","template_id":"tmpl-001"}`},
		{"template_id", `{"user_id":"user-456","channel":"PUSH"}`},
	}

	rapid.Check(t, func(rt *rapid.T) {
		ctrl := gomock.NewController(t)
		mockSvc := mocks.NewMockNotificationService(ctrl)
		handler := httphandler.NewNotificationHandler(mockSvc)

		idx := rapid.IntRange(0, len(requiredFields)-1).Draw(rt, "field_idx")
		tc := requiredFields[idx]

		req := httptest.NewRequest(http.MethodPost, "/notifications/send", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		req = injectRequestID(req, "test-request-id")
		w := httptest.NewRecorder()

		handler.SendNotification(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		message, ok := resp["message"].(string)
		assert.True(t, ok, "message should be a string")
		assert.Contains(t, message, tc.fieldName, "error message should contain the missing field name")
	})
}
