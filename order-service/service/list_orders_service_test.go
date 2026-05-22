package service_test

// Validates: Requirements 8.1, 8.2, 8.3, 13.6

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/mocks"
	"order-service/service"
	"order-service/types"
)

// TestListOrdersService validates Requirements 8.1, 8.2, 8.3, 13.6.
func TestListOrdersService(t *testing.T) {
	sampleOrders := []*types.Order{
		{OrderID: "order-1", Status: types.OrderStatusCreated},
		{OrderID: "order-2", Status: types.OrderStatusInTransit},
	}

	tests := []struct {
		name        string
		params      types.ListOrdersParams
		repoSetup   func(*mocks.MockOrderRepository)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, []*types.Order)
	}{
		{
			// Requirement 8.1: all four params passed unchanged to repo
			name: "all four params are passed unchanged to the repository",
			params: types.ListOrdersParams{
				UserID: "user-42",
				Status: types.OrderStatusInTransit,
				Page:   3,
				Limit:  20,
			},
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("FindOrders", context.Background(), types.ListOrdersParams{
					UserID: "user-42",
					Status: types.OrderStatusInTransit,
					Page:   3,
					Limit:  20,
				}).Return(sampleOrders, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, orders []*types.Order) {
				require.Len(t, orders, 2)
				assert.Equal(t, "order-1", orders[0].OrderID)
				assert.Equal(t, "order-2", orders[1].OrderID)
			},
		},
		{
			// Requirement 8.2: repo returns empty slice → empty slice returned (no error)
			name: "repo returns empty slice results in empty slice with no error",
			params: types.ListOrdersParams{
				UserID: "user-no-orders",
				Status: types.OrderStatusDelivered,
				Page:   1,
				Limit:  10,
			},
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("FindOrders", context.Background(), types.ListOrdersParams{
					UserID: "user-no-orders",
					Status: types.OrderStatusDelivered,
					Page:   1,
					Limit:  10,
				}).Return([]*types.Order{}, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, orders []*types.Order) {
				require.NotNil(t, orders)
				assert.Empty(t, orders)
			},
		},
		{
			// Requirement 8.3: repo error → error propagated
			name: "repo error is propagated to the caller",
			params: types.ListOrdersParams{
				UserID: "user-err",
				Status: types.OrderStatusCreated,
				Page:   1,
				Limit:  5,
			},
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("FindOrders", context.Background(), types.ListOrdersParams{
					UserID: "user-err",
					Status: types.OrderStatusCreated,
					Page:   1,
					Limit:  5,
				}).Return(nil, errors.New("db unavailable"))
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "db unavailable")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := new(mocks.MockOrderRepository)
			tc.repoSetup(mockRepo)

			svc := service.NewOrderService(mockRepo)
			orders, err := svc.ListOrders(context.Background(), tc.params)

			if tc.expectErr {
				require.Error(t, err)
				if tc.checkErr != nil {
					tc.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
				if tc.checkResult != nil {
					tc.checkResult(t, orders)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
