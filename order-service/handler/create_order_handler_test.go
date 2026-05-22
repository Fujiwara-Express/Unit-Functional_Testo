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

// validCreateOrderBody returns a minimal valid JSON body for POST /orders.
func validCreateOrderBody() map[string]any {
	return map[string]any{
		"sender_user_id":   "user-1",
		"sender_name":      "Alice",
		"sender_address":   "123 Main St",
		"sender_phone":     "08001234567",
		"sender_city_code": "JKT",
		"receiver_name":    "Bob",
		"receiver_address": "456 Oak Ave",
		"receiver_phone":   "08009876543",
		"receiver_city_code": "SBY",
		"weight":           1.5,
		"length":           10.0,
		"width":            10.0,
		"height":           10.0,
		"service_type":     "REG",
		"is_cod":           false,
		"cod_amount":       0.0,
		"insurance":        false,
		"item_description": "books",
	}
}

func TestCreateOrderHandler(t *testing.T) {
	tests := []struct {
		name           string
		body           any    // will be JSON-encoded; use string for raw/malformed JSON
		rawBody        string // if non-empty, used as-is (for malformed JSON)
		mockSetup      func(*mocks.MockOrderService)
		expectedStatus int
		checkBody      func(*testing.T, map[string]any)
	}{
		{
			name: "valid request returns 201 with all required fields",
			body: validCreateOrderBody(),
			mockSetup: func(m *mocks.MockOrderService) {
				m.On("CreateOrder", mock.Anything, mock.Anything).Return(&types.CreateOrderResponse{
					OrderID:        "order-abc",
					TrackingNumber: "TRK123456",
					Price:          25000.0,
					EstimatedDays:  3,
					Status:         types.OrderStatusAwaitingPickup,
				}, nil)
			},
			expectedStatus: http.StatusCreated,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "order-abc", body["order_id"])
				assert.Equal(t, "TRK123456", body["tracking_number"])
				assert.Equal(t, 25000.0, body["price"])
				assert.Equal(t, float64(3), body["estimated_days"])
				assert.Equal(t, "AWAITING_PICKUP", body["status"])
				assert.NotEmpty(t, body["created_at"])
			},
		},
		{
			name:           "malformed JSON returns 400",
			rawBody:        `{not valid json`,
			mockSetup:      func(m *mocks.MockOrderService) { /* service must NOT be called */ },
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.NotEmpty(t, body["error"])
			},
		},
		{
			name: "missing required field returns 422",
			body: func() map[string]any {
				b := validCreateOrderBody()
				delete(b, "sender_user_id")
				return b
			}(),
			mockSetup:      func(m *mocks.MockOrderService) { /* service must NOT be called */ },
			expectedStatus: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.NotEmpty(t, body["error"])
			},
		},
		{
			name: "invalid service_type returns 422",
			body: func() map[string]any {
				b := validCreateOrderBody()
				b["service_type"] = "INVALID"
				return b
			}(),
			mockSetup:      func(m *mocks.MockOrderService) { /* service must NOT be called */ },
			expectedStatus: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.NotEmpty(t, body["error"])
			},
		},
		{
			name: "is_cod=true with cod_amount<=0 returns 422",
			body: func() map[string]any {
				b := validCreateOrderBody()
				b["is_cod"] = true
				b["cod_amount"] = 0.0
				return b
			}(),
			mockSetup:      func(m *mocks.MockOrderService) { /* service must NOT be called */ },
			expectedStatus: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.NotEmpty(t, body["error"])
			},
		},
		{
			name: "service error returns 500",
			body: validCreateOrderBody(),
			mockSetup: func(m *mocks.MockOrderService) {
				m.On("CreateOrder", mock.Anything, mock.Anything).Return(nil, errors.New("database unavailable"))
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

			var bodyBytes []byte
			if tc.rawBody != "" {
				bodyBytes = []byte(tc.rawBody)
			} else {
				var err error
				bodyBytes, err = json.Marshal(tc.body)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			h.CreateOrder(rec, req)

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
