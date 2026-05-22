package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"order-service/handler"
	"order-service/mocks"
	"order-service/types"
)

// sampleOrder returns a fully-populated Order for use in GetOrder tests.
func sampleOrder() *types.Order {
	now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	return &types.Order{
		OrderID:          "order-123",
		TrackingNumber:   "TRK999888",
		SenderUserID:     "user-1",
		SenderName:       "Alice",
		SenderAddress:    "123 Main St",
		SenderPhone:      "08001234567",
		SenderCityCode:   "JKT",
		ReceiverName:     "Bob",
		ReceiverAddress:  "456 Oak Ave",
		ReceiverPhone:    "08009876543",
		ReceiverCityCode: "SBY",
		Weight:           2.5,
		Length:           20.0,
		Width:            15.0,
		Height:           10.0,
		ServiceType:      types.ServiceTypeREG,
		Price:            35000.0,
		IsCOD:            false,
		CODAmount:        0.0,
		Insurance:        true,
		ItemDescription:  "electronics",
		Status:           types.OrderStatusAwaitingPickup,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func TestGetOrderHandler(t *testing.T) {
	tests := []struct {
		name           string
		orderID        string // path parameter value; empty string simulates missing param
		mockSetup      func(*mocks.MockOrderService)
		expectedStatus int
		checkBody      func(*testing.T, map[string]any)
	}{
		{
			name:    "valid order_id returns 200 with full Order JSON",
			orderID: "order-123",
			mockSetup: func(m *mocks.MockOrderService) {
				m.On("GetOrder", mock.Anything, "order-123").Return(sampleOrder(), nil)
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "order-123", body["order_id"])
				assert.Equal(t, "TRK999888", body["tracking_number"])
				assert.Equal(t, "user-1", body["sender_user_id"])
				assert.Equal(t, "Alice", body["sender_name"])
				assert.Equal(t, "123 Main St", body["sender_address"])
				assert.Equal(t, "08001234567", body["sender_phone"])
				assert.Equal(t, "JKT", body["sender_city_code"])
				assert.Equal(t, "Bob", body["receiver_name"])
				assert.Equal(t, "456 Oak Ave", body["receiver_address"])
				assert.Equal(t, "08009876543", body["receiver_phone"])
				assert.Equal(t, "SBY", body["receiver_city_code"])
				assert.Equal(t, 2.5, body["weight"])
				assert.Equal(t, 20.0, body["length"])
				assert.Equal(t, 15.0, body["width"])
				assert.Equal(t, 10.0, body["height"])
				assert.Equal(t, "REG", body["service_type"])
				assert.Equal(t, 35000.0, body["price"])
				assert.Equal(t, false, body["is_cod"])
				assert.Equal(t, 0.0, body["cod_amount"])
				assert.Equal(t, true, body["insurance"])
				assert.Equal(t, "electronics", body["item_description"])
				assert.Equal(t, "AWAITING_PICKUP", body["status"])
				assert.NotEmpty(t, body["created_at"])
				assert.NotEmpty(t, body["updated_at"])
			},
		},
		{
			name:    "service not-found error returns 404",
			orderID: "order-missing",
			mockSetup: func(m *mocks.MockOrderService) {
				m.On("GetOrder", mock.Anything, "order-missing").Return(nil, handler.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.NotEmpty(t, body["error"])
			},
		},
		{
			name:    "service unexpected error returns 500",
			orderID: "order-456",
			mockSetup: func(m *mocks.MockOrderService) {
				m.On("GetOrder", mock.Anything, "order-456").Return(nil, errors.New("database unavailable"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkBody: func(t *testing.T, body map[string]any) {
				assert.NotEmpty(t, body["error"])
			},
		},
		{
			name:           "empty order_id path param returns 400 without service call",
			orderID:        "", // empty — handler should reject before calling service
			mockSetup:      func(m *mocks.MockOrderService) { /* service must NOT be called */ },
			expectedStatus: http.StatusBadRequest,
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

			target := "/orders/" + tc.orderID
			req := httptest.NewRequest(http.MethodGet, target, nil)

			// Inject the path value so r.PathValue("order_id") returns tc.orderID.
			// Go 1.22 ServeMux sets path values; for direct handler calls we use
			// httptest + SetPathValue (available since Go 1.22).
			req.SetPathValue("order_id", tc.orderID)

			rec := httptest.NewRecorder()
			h.GetOrder(rec, req)

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
