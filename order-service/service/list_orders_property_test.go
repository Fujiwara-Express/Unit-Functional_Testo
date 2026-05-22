package service_test

// Validates: Requirements 8.1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	"order-service/mocks"
	"order-service/service"
	"order-service/types"
)

// genOrderStatus returns a rapid generator for valid OrderStatus values.
func genOrderStatus(t *rapid.T) types.OrderStatus {
	return rapid.SampledFrom([]types.OrderStatus{
		types.OrderStatusCreated,
		types.OrderStatusAwaitingPickup,
		types.OrderStatusPickedUp,
		types.OrderStatusInTransit,
		types.OrderStatusDelivered,
		types.OrderStatusFailed,
		types.OrderStatusCancelled,
	}).Draw(t, "status")
}

// TestProperty8_ListOrdersParameterPassthrough verifies that for any combination of
// user_id, status, page, and limit, all four values reach the repository unchanged.
// Validates: Requirements 8.1
func TestProperty8_ListOrdersParameterPassthrough(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		userID := rapid.StringN(1, 64, -1).Draw(t, "userID")
		status := genOrderStatus(t)
		page := rapid.IntRange(1, 1000).Draw(t, "page")
		limit := rapid.IntRange(1, 100).Draw(t, "limit")

		params := types.ListOrdersParams{
			UserID: userID,
			Status: status,
			Page:   page,
			Limit:  limit,
		}

		mockRepo := new(mocks.MockOrderRepository)
		// Expect FindOrders to be called with exactly the params we generated.
		mockRepo.On("FindOrders", context.Background(), params).Return([]*types.Order{}, nil)

		svc := service.NewOrderService(mockRepo)
		_, err := svc.ListOrders(context.Background(), params)

		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}
