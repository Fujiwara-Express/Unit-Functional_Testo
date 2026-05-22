package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"order-service/handler"
	"order-service/mocks"
)

func TestCancelOrderHandler(t *testing.T) {
	tests := []struct {
		name           string
		orderID        string
		body           string
		mockSetup      func(*mocks.MockOrderService)
		expectedStatus int
		checkBody      func(*testing.T, map[string]any)
	}{
		{
			name:    "valid request returns 200",
			orderID: "order-123",
			body:    `{"reason":"customer request"}`,
			mockSetup: func(m *mocks.MockOrderService) {
				m.On("CancelOrder", mock.Anything, "order-123", "customer request").Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "cancelled", body["status"])
			},
		},
		{
			name:    "service conflict error returns 409",
			orderID: "order-456",
			body:    `{"reason":"duplicate"}`,
			mockSetup: func(m *mocks.MockOrderService) {
				m.On("CancelOrder", mock.Anything, "order-456", "duplicate").Return(handler.ErrConflict)
			},
			expectedStatus: http.StatusConflict,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.NotEmpty(t, body["error"])
			},
		},
		{
			name:    "service not-found error returns 404",
			orderID: "order-999",
			body:    `{"reason":"wrong order"}`,
			mockSetup: func(m *mocks.MockOrderService) {
				m.On("CancelOrder", mock.Anything, "order-999", "wrong order").Return(handler.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.NotEmpty(t, body["error"])
			},
		},
		{
			name:           "missing reason returns 422 without service call",
			orderID:        "order-123",
			body:           `{}`,
			mockSetup:      func(m *mocks.MockOrderService) { /* must NOT be called */ },
			expectedStatus: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.NotEmpty(t, body["error"])
			},
		},
		{
			name:           "empty reason returns 422 without service call",
			orderID:        "order-123",
			body:           `{"reason":""}`,
			mockSetup:      func(m *mocks.MockOrderService) { /* must NOT be called */ },
			expectedStatus: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.NotEmpty(t, body["error"])
			},
		},
		{
			name:    "service unexpected error returns 500",
			orderID: "order-789",
			body:    `{"reason":"test"}`,
			mockSetup: func(m *mocks.MockOrderService) {
				m.On("CancelOrder", mock.Anything, "order-789", "test").Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.NotEmpty(t, body["error"])
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockOrderService)
			tc.mockSetup(mockSvc)

			h := handler.New(mockSvc)

			req := httptest.NewRequest(http.MethodPost, "/orders/"+tc.orderID+"/cancel", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.SetPathValue("order_id", tc.orderID)

			rec := httptest.NewRecorder()
			h.CancelOrder(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)

			if tc.checkBody != nil {
				var respBody map[string]any
				err := json.Unmarshal(rec.Body.Bytes(), &respBody)
				require.NoError(t, err, "response body should be valid JSON")
				tc.checkBody(t, respBody)
			}

			mockSvc.AssertExpectations(t)
		})
	}
}
