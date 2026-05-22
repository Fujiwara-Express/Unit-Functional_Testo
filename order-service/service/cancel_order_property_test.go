package service_test

// Validates: Requirements 9.2

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	"order-service/mocks"
	"order-service/service"
	"order-service/types"
)

// genNonAwaitingPickupStatus generates a random OrderStatus that is NOT AWAITING_PICKUP.
func genNonAwaitingPickupStatus(t *rapid.T) types.OrderStatus {
	return rapid.SampledFrom([]types.OrderStatus{
		types.OrderStatusCreated,
		types.OrderStatusPickedUp,
		types.OrderStatusInTransit,
		types.OrderStatusDelivered,
		types.OrderStatusFailed,
		types.OrderStatusCancelled,
	}).Draw(t, "nonAwaitingPickupStatus")
}

// TestProperty5_CancelStatusGuard verifies that for any Order with Status != AWAITING_PICKUP,
// CancelOrder returns a conflict error and repo UpdateOrder is never called.
// Validates: Requirements 9.2
func TestProperty5_CancelStatusGuard(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		status := genNonAwaitingPickupStatus(t)

		mockRepo := new(mocks.MockOrderRepository)
		mockRepo.On("FindOrderByID", mock.Anything, "order-prop").Return(&types.Order{
			OrderID: "order-prop",
			Status:  status,
		}, nil)
		// UpdateOrder is intentionally NOT registered — AssertExpectations will fail if it is called

		svc := service.NewOrderService(mockRepo)
		err := svc.CancelOrder(context.Background(), "order-prop", "test reason")

		require.Error(t, err, "expected conflict error for status %s", status)
		require.True(t,
			strings.Contains(err.Error(), "order cannot be cancelled: status is not AWAITING_PICKUP"),
			"expected conflict error message for status %s, got: %s", status, err.Error(),
		)

		mockRepo.AssertExpectations(t)
	})
}
