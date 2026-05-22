package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/handler"
	"order-service/mocks"
	"order-service/service"
	"order-service/types"
)

// TestGetOrderService validates Requirements 7.1, 7.2, 7.3, 13.6.
func TestGetOrderService(t *testing.T) {
	sampleOrder := &types.Order{
		OrderID:        "order-abc",
		TrackingNumber: "TRK123456",
		Status:         types.OrderStatusAwaitingPickup,
	}

	tests := []struct {
		name      string
		orderID   string
		repoSetup func(*mocks.MockOrderRepository)
		expectErr bool
		checkErr  func(*testing.T, error)
		checkResp func(*testing.T, *types.Order)
	}{
		{
			// Requirement 7.1: valid order_id → Order returned
			name:    "valid order_id returns the order from repo",
			orderID: "order-abc",
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("FindOrderByID", context.Background(), "order-abc").Return(sampleOrder, nil)
			},
			expectErr: false,
			checkResp: func(t *testing.T, order *types.Order) {
				require.NotNil(t, order)
				assert.Equal(t, "order-abc", order.OrderID)
				assert.Equal(t, "TRK123456", order.TrackingNumber)
			},
		},
		{
			// Requirement 7.2: repo not-found → not-found error propagated
			name:    "repo not-found error is propagated",
			orderID: "order-missing",
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("FindOrderByID", context.Background(), "order-missing").Return(nil, handler.ErrNotFound)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.True(t, errors.Is(err, handler.ErrNotFound), "expected not-found sentinel error")
			},
		},
		{
			// Requirement 7.3: repo unexpected error → error propagated
			name:    "repo unexpected error is propagated",
			orderID: "order-xyz",
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("FindOrderByID", context.Background(), "order-xyz").Return(nil, errors.New("db timeout"))
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "db timeout")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := new(mocks.MockOrderRepository)
			tc.repoSetup(mockRepo)

			svc := service.NewOrderService(mockRepo)
			order, err := svc.GetOrder(context.Background(), tc.orderID)

			if tc.expectErr {
				require.Error(t, err)
				if tc.checkErr != nil {
					tc.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
				if tc.checkResp != nil {
					tc.checkResp(t, order)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
