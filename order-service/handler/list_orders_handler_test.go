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

// sampleOrders returns a small slice of orders for list tests.
func sampleOrders() []*types.Order {
	now := time.Date(2024, 3, 1, 9, 0, 0, 0, time.UTC)
	return []*types.Order{
		{
			OrderID:        "order-1",
			TrackingNumber: "TRK000001",
			SenderUserID:   "user-42",
			ReceiverName:   "Charlie",
			ServiceType:    types.ServiceTypeREG,
			Price:          15000.0,
			Status:         types.OrderStatusAwaitingPickup,
			CreatedAt:      now,
		},
		{
			OrderID:        "order-2",
			TrackingNumber: "TRK000002",
			SenderUserID:   "user-42",
			ReceiverName:   "Diana",
			ServiceType:    types.ServiceTypeYES,
			Price:          30000.0,
			Status:         types.OrderStatusInTransit,
			CreatedAt:      now,
		},
	}
}

func TestListOrdersHandler(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string // raw query string, e.g. "user_id=u1&status=CREATED&page=1&limit=10"
		mockSetup      func(*mocks.MockOrderService)
		expectedStatus int
		checkBody      func(*testing.T, []byte)
	}{
		{
			// Requirements 3.1, 3.2: valid params → 200 with order array, all four params passed
			name:        "valid params returns 200 with order array",
			queryParams: "user_id=user-42&status=AWAITING_PICKUP&page=1&limit=10",
			mockSetup: func(m *mocks.MockOrderService) {
				expected := types.ListOrdersParams{
					UserID: "user-42",
					Status: types.OrderStatusAwaitingPickup,
					Page:   1,
					Limit:  10,
				}
				m.On("ListOrders", mock.Anything, expected).Return(sampleOrders(), nil)
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var items []map[string]any
				require.NoError(t, json.Unmarshal(body, &items))
				assert.Len(t, items, 2)
				assert.Equal(t, "order-1", items[0]["order_id"])
				assert.Equal(t, "TRK000001", items[0]["tracking_number"])
				assert.Equal(t, "user-42", items[0]["sender_user_id"])
				assert.Equal(t, "Charlie", items[0]["receiver_name"])
				assert.Equal(t, "REG", items[0]["service_type"])
				assert.Equal(t, 15000.0, items[0]["price"])
				assert.Equal(t, "AWAITING_PICKUP", items[0]["status"])
				assert.NotEmpty(t, items[0]["created_at"])
			},
		},
		{
			// Requirement 3.3: non-positive page → 400
			name:           "page=0 returns 400",
			queryParams:    "user_id=u1&status=CREATED&page=0&limit=10",
			mockSetup:      func(m *mocks.MockOrderService) { /* must NOT be called */ },
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body []byte) {
				var resp map[string]any
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.NotEmpty(t, resp["error"])
			},
		},
		{
			// Requirement 3.3: negative page → 400
			name:           "page=-1 returns 400",
			queryParams:    "user_id=u1&status=CREATED&page=-1&limit=10",
			mockSetup:      func(m *mocks.MockOrderService) { /* must NOT be called */ },
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body []byte) {
				var resp map[string]any
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.NotEmpty(t, resp["error"])
			},
		},
		{
			// Requirement 3.3: non-positive limit → 400
			name:           "limit=0 returns 400",
			queryParams:    "user_id=u1&status=CREATED&page=1&limit=0",
			mockSetup:      func(m *mocks.MockOrderService) { /* must NOT be called */ },
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body []byte) {
				var resp map[string]any
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.NotEmpty(t, resp["error"])
			},
		},
		{
			// Requirement 3.4: invalid status enum → 400
			name:           "invalid status returns 400",
			queryParams:    "user_id=u1&status=INVALID_STATUS&page=1&limit=10",
			mockSetup:      func(m *mocks.MockOrderService) { /* must NOT be called */ },
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body []byte) {
				var resp map[string]any
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.NotEmpty(t, resp["error"])
			},
		},
		{
			// Requirement 3.5: empty list → 200 with []
			name:        "empty list returns 200 with empty array",
			queryParams: "user_id=user-99&status=DELIVERED&page=2&limit=5",
			mockSetup: func(m *mocks.MockOrderService) {
				expected := types.ListOrdersParams{
					UserID: "user-99",
					Status: types.OrderStatusDelivered,
					Page:   2,
					Limit:  5,
				}
				m.On("ListOrders", mock.Anything, expected).Return([]*types.Order{}, nil)
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var items []map[string]any
				require.NoError(t, json.Unmarshal(body, &items))
				assert.Empty(t, items)
			},
		},
		{
			// Requirement 3.6: service error → 500
			name:        "service error returns 500",
			queryParams: "user_id=u1&status=CREATED&page=1&limit=10",
			mockSetup: func(m *mocks.MockOrderService) {
				m.On("ListOrders", mock.Anything, mock.Anything).Return(nil, errors.New("db failure"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkBody: func(t *testing.T, body []byte) {
				var resp map[string]any
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.NotEmpty(t, resp["error"])
			},
		},
		{
			// Requirement 3.1: all four params passed correctly to mock service
			name:        "all four params passed to service",
			queryParams: "user_id=user-7&status=IN_TRANSIT&page=3&limit=20",
			mockSetup: func(m *mocks.MockOrderService) {
				expected := types.ListOrdersParams{
					UserID: "user-7",
					Status: types.OrderStatusInTransit,
					Page:   3,
					Limit:  20,
				}
				m.On("ListOrders", mock.Anything, expected).Return(sampleOrders(), nil)
			},
			expectedStatus: http.StatusOK,
			checkBody:      nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockOrderService)
			tc.mockSetup(mockSvc)

			h := handler.New(mockSvc)

			req := httptest.NewRequest(http.MethodGet, "/orders?"+tc.queryParams, nil)
			rec := httptest.NewRecorder()

			h.ListOrders(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)

			if tc.checkBody != nil {
				tc.checkBody(t, rec.Body.Bytes())
			}

			mockSvc.AssertExpectations(t)
		})
	}
}
