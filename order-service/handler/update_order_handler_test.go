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
	"order-service/types"
)

func strPtr(s string) *string { return &s }

func TestUpdateOrderHandler(t *testing.T) {
	tests := []struct {
		name           string
		orderID        string
		body           string
		mockSetup      func(*mocks.MockOrderService)
		expectedStatus int
		checkBody      func(*testing.T, map[string]any)
	}{
		{
			// Requirements 5.1, 5.2
			name:    "valid partial update returns 200 with order_id and UPDATED status",
			orderID: "order-123",
			body:    `{"receiver_name":"Alice","receiver_address":"123 Main St"}`,
			mockSetup: func(m *mocks.MockOrderService) {
				m.On("UpdateOrder", mock.Anything, "order-123", types.UpdateOrderRequest{
					ReceiverName:    strPtr("Alice"),
					ReceiverAddress: strPtr("123 Main St"),
				}).Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "order-123", body["order_id"])
				assert.Equal(t, "UPDATED", body["status"])
			},
		},
		{
			// Requirement 5.3
			name:    "service conflict error returns 409",
			orderID: "order-456",
			body:    `{"item_description":"fragile goods"}`,
			mockSetup: func(m *mocks.MockOrderService) {
				m.On("UpdateOrder", mock.Anything, "order-456", types.UpdateOrderRequest{
					ItemDescription: strPtr("fragile goods"),
				}).Return(handler.ErrConflict)
			},
			expectedStatus: http.StatusConflict,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.NotEmpty(t, body["error"])
			},
		},
		{
			// Requirement 5.4
			name:    "service not-found error returns 404",
			orderID: "order-999",
			body:    `{"receiver_phone":"08123456789"}`,
			mockSetup: func(m *mocks.MockOrderService) {
				m.On("UpdateOrder", mock.Anything, "order-999", types.UpdateOrderRequest{
					ReceiverPhone: strPtr("08123456789"),
				}).Return(handler.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.NotEmpty(t, body["error"])
			},
		},
		{
			// Requirement 5.5
			name:           "no updatable fields returns 422 without service call",
			orderID:        "order-123",
			body:           `{}`,
			mockSetup:      func(m *mocks.MockOrderService) { /* must NOT be called */ },
			expectedStatus: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.NotEmpty(t, body["error"])
			},
		},
		{
			// Requirement 5.5 — explicit null fields still count as no updatable fields
			name:           "all null fields returns 422 without service call",
			orderID:        "order-123",
			body:           `{"receiver_name":null,"item_description":null}`,
			mockSetup:      func(m *mocks.MockOrderService) { /* must NOT be called */ },
			expectedStatus: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.NotEmpty(t, body["error"])
			},
		},
		{
			// Requirement 13.6 — service unexpected error returns 500
			name:    "service unexpected error returns 500",
			orderID: "order-789",
			body:    `{"receiver_city_code":"JKT"}`,
			mockSetup: func(m *mocks.MockOrderService) {
				m.On("UpdateOrder", mock.Anything, "order-789", types.UpdateOrderRequest{
					ReceiverCityCode: strPtr("JKT"),
				}).Return(errors.New("database error"))
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

			req := httptest.NewRequest(http.MethodPatch, "/orders/"+tc.orderID, bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.SetPathValue("order_id", tc.orderID)

			rec := httptest.NewRecorder()
			h.UpdateOrder(rec, req)

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
