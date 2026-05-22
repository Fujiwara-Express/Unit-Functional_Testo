package service_test

// Validates: Requirements 10.2, 10.3, 10.4

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

// genNonAwaitingPickupStatusForUpdate generates a random OrderStatus that is NOT AWAITING_PICKUP.
func genNonAwaitingPickupStatusForUpdate(t *rapid.T) types.OrderStatus {
	return rapid.SampledFrom([]types.OrderStatus{
		types.OrderStatusCreated,
		types.OrderStatusPickedUp,
		types.OrderStatusInTransit,
		types.OrderStatusDelivered,
		types.OrderStatusFailed,
		types.OrderStatusCancelled,
	}).Draw(t, "nonAwaitingPickupStatus")
}

// TestProperty6_UpdateStatusGuard verifies that for any Order with Status != AWAITING_PICKUP,
// UpdateOrder returns a conflict error and repo UpdateOrder is never called.
// Validates: Requirements 10.2
func TestProperty6_UpdateStatusGuard(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		status := genNonAwaitingPickupStatusForUpdate(t)

		mockRepo := new(mocks.MockOrderRepository)
		mockRepo.On("FindOrderByID", mock.Anything, "order-prop").Return(&types.Order{
			OrderID: "order-prop",
			Status:  status,
		}, nil)
		// UpdateOrder is intentionally NOT registered — AssertExpectations will fail if it is called

		svc := service.NewOrderService(mockRepo)
		req := types.UpdateOrderRequest{
			ReceiverName: func() *string { s := "Any Name"; return &s }(),
		}
		err := svc.UpdateOrder(context.Background(), "order-prop", req)

		require.Error(t, err, "expected conflict error for status %s", status)
		require.True(t,
			strings.Contains(err.Error(), "order cannot be updated: status is not AWAITING_PICKUP"),
			"expected conflict error message for status %s, got: %s", status, err.Error(),
		)

		mockRepo.AssertExpectations(t)
	})
}

// genUpdateOrderRequest generates an UpdateOrderRequest with a random non-empty subset of fields set.
func genUpdateOrderRequest(t *rapid.T) types.UpdateOrderRequest {
	setReceiverName := rapid.Bool().Draw(t, "setReceiverName")
	setReceiverAddress := rapid.Bool().Draw(t, "setReceiverAddress")
	setReceiverPhone := rapid.Bool().Draw(t, "setReceiverPhone")
	setReceiverCityCode := rapid.Bool().Draw(t, "setReceiverCityCode")
	setItemDescription := rapid.Bool().Draw(t, "setItemDescription")

	// Ensure at least one field is set
	if !setReceiverName && !setReceiverAddress && !setReceiverPhone && !setReceiverCityCode && !setItemDescription {
		setReceiverName = true
	}

	req := types.UpdateOrderRequest{}
	if setReceiverName {
		s := rapid.StringN(1, 50, -1).Draw(t, "receiverName")
		req.ReceiverName = &s
	}
	if setReceiverAddress {
		s := rapid.StringN(1, 100, -1).Draw(t, "receiverAddress")
		req.ReceiverAddress = &s
	}
	if setReceiverPhone {
		s := rapid.StringN(1, 20, -1).Draw(t, "receiverPhone")
		req.ReceiverPhone = &s
	}
	if setReceiverCityCode {
		s := rapid.StringN(1, 10, -1).Draw(t, "receiverCityCode")
		req.ReceiverCityCode = &s
	}
	if setItemDescription {
		s := rapid.StringN(1, 200, -1).Draw(t, "itemDescription")
		req.ItemDescription = &s
	}
	return req
}

// TestProperty7_PartialUpdateFieldIsolation verifies that for any UpdateOrderRequest setting only
// a subset of fields, exactly those fields are changed and all others remain equal to the original.
// Validates: Requirements 10.3, 10.4
func TestProperty7_PartialUpdateFieldIsolation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		req := genUpdateOrderRequest(t)

		original := types.Order{
			OrderID:          "order-prop",
			Status:           types.OrderStatusAwaitingPickup,
			ReceiverName:     "Original Receiver",
			ReceiverAddress:  "Original Address",
			ReceiverPhone:    "08000000000",
			ReceiverCityCode: "SBY",
			ItemDescription:  "Original Item",
			SenderName:       "Sender",
			SenderAddress:    "Sender Address",
			SenderPhone:      "07000000000",
			SenderCityCode:   "JKT",
			Weight:           1.5,
			Price:            15000.0,
		}

		var capturedOrder *types.Order

		mockRepo := new(mocks.MockOrderRepository)
		mockRepo.On("FindOrderByID", mock.Anything, "order-prop").Return(&original, nil)
		mockRepo.On("UpdateOrder", mock.Anything, mock.MatchedBy(func(o *types.Order) bool {
			capturedOrder = o
			return true
		})).Return(nil)

		svc := service.NewOrderService(mockRepo)
		err := svc.UpdateOrder(context.Background(), "order-prop", req)
		require.NoError(t, err)
		require.NotNil(t, capturedOrder)

		// Fields that were set in the request must equal the requested values
		if req.ReceiverName != nil {
			require.Equal(t, *req.ReceiverName, capturedOrder.ReceiverName,
				"ReceiverName should be updated to requested value")
		} else {
			require.Equal(t, original.ReceiverName, capturedOrder.ReceiverName,
				"ReceiverName should be unchanged when not in request")
		}

		if req.ReceiverAddress != nil {
			require.Equal(t, *req.ReceiverAddress, capturedOrder.ReceiverAddress,
				"ReceiverAddress should be updated to requested value")
		} else {
			require.Equal(t, original.ReceiverAddress, capturedOrder.ReceiverAddress,
				"ReceiverAddress should be unchanged when not in request")
		}

		if req.ReceiverPhone != nil {
			require.Equal(t, *req.ReceiverPhone, capturedOrder.ReceiverPhone,
				"ReceiverPhone should be updated to requested value")
		} else {
			require.Equal(t, original.ReceiverPhone, capturedOrder.ReceiverPhone,
				"ReceiverPhone should be unchanged when not in request")
		}

		if req.ReceiverCityCode != nil {
			require.Equal(t, *req.ReceiverCityCode, capturedOrder.ReceiverCityCode,
				"ReceiverCityCode should be updated to requested value")
		} else {
			require.Equal(t, original.ReceiverCityCode, capturedOrder.ReceiverCityCode,
				"ReceiverCityCode should be unchanged when not in request")
		}

		if req.ItemDescription != nil {
			require.Equal(t, *req.ItemDescription, capturedOrder.ItemDescription,
				"ItemDescription should be updated to requested value")
		} else {
			require.Equal(t, original.ItemDescription, capturedOrder.ItemDescription,
				"ItemDescription should be unchanged when not in request")
		}

		// Non-updatable fields must always remain unchanged
		require.Equal(t, original.SenderName, capturedOrder.SenderName, "SenderName must not change")
		require.Equal(t, original.SenderAddress, capturedOrder.SenderAddress, "SenderAddress must not change")
		require.Equal(t, original.SenderPhone, capturedOrder.SenderPhone, "SenderPhone must not change")
		require.Equal(t, original.SenderCityCode, capturedOrder.SenderCityCode, "SenderCityCode must not change")
		require.Equal(t, original.Weight, capturedOrder.Weight, "Weight must not change")
		require.Equal(t, original.Price, capturedOrder.Price, "Price must not change")
		require.Equal(t, original.Status, capturedOrder.Status, "Status must not change")

		mockRepo.AssertExpectations(t)
	})
}
