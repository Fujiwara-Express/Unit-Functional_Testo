package service_test

// Validates: Requirements 10.1, 10.2, 10.3, 10.4, 10.5, 13.6

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"order-service/handler"
	"order-service/mocks"
	"order-service/service"
	"order-service/types"
)

func ptr(s string) *string { return &s }

func TestUpdateOrderService(t *testing.T) {
	tests := []struct {
		name      string
		orderID   string
		req       types.UpdateOrderRequest
		repoSetup func(*mocks.MockOrderRepository)
		expectErr bool
		checkErr  func(*testing.T, error)
	}{
		{
			// Requirement 10.1: order in AWAITING_PICKUP → fields applied, repo update called
			name:    "order in AWAITING_PICKUP has fields applied and repo update is called",
			orderID: "order-1",
			req: types.UpdateOrderRequest{
				ReceiverName:     ptr("New Name"),
				ReceiverAddress:  ptr("New Address"),
				ReceiverPhone:    ptr("08123456789"),
				ReceiverCityCode: ptr("JKT"),
				ItemDescription:  ptr("New Item"),
			},
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("FindOrderByID", mock.Anything, "order-1").Return(&types.Order{
					OrderID:          "order-1",
					Status:           types.OrderStatusAwaitingPickup,
					ReceiverName:     "Old Name",
					ReceiverAddress:  "Old Address",
					ReceiverPhone:    "08000000000",
					ReceiverCityCode: "SBY",
					ItemDescription:  "Old Item",
				}, nil)
				m.On("UpdateOrder", mock.Anything, mock.MatchedBy(func(o *types.Order) bool {
					return o.OrderID == "order-1" &&
						o.ReceiverName == "New Name" &&
						o.ReceiverAddress == "New Address" &&
						o.ReceiverPhone == "08123456789" &&
						o.ReceiverCityCode == "JKT" &&
						o.ItemDescription == "New Item"
				})).Return(nil)
			},
			expectErr: false,
		},
		{
			// Requirement 10.2: order not in AWAITING_PICKUP → conflict error, repo update NOT called
			name:    "order in PICKED_UP status returns conflict error without calling update",
			orderID: "order-2",
			req: types.UpdateOrderRequest{
				ReceiverName: ptr("New Name"),
			},
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("FindOrderByID", mock.Anything, "order-2").Return(&types.Order{
					OrderID: "order-2",
					Status:  types.OrderStatusPickedUp,
				}, nil)
				// UpdateOrder must NOT be called
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.True(t,
					strings.Contains(err.Error(), "order cannot be updated: status is not AWAITING_PICKUP"),
					"expected conflict error message, got: %s", err.Error(),
				)
			},
		},
		{
			// Requirement 10.3: only receiver fields provided → only receiver fields changed
			name:    "only receiver fields provided leaves item_description unchanged",
			orderID: "order-3",
			req: types.UpdateOrderRequest{
				ReceiverName:     ptr("Updated Name"),
				ReceiverAddress:  ptr("Updated Address"),
				ReceiverPhone:    ptr("08999999999"),
				ReceiverCityCode: ptr("BDG"),
			},
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("FindOrderByID", mock.Anything, "order-3").Return(&types.Order{
					OrderID:         "order-3",
					Status:          types.OrderStatusAwaitingPickup,
					ReceiverName:    "Old Name",
					ReceiverAddress: "Old Address",
					ReceiverPhone:   "08000000000",
					ItemDescription: "Original Description",
				}, nil)
				m.On("UpdateOrder", mock.Anything, mock.MatchedBy(func(o *types.Order) bool {
					return o.OrderID == "order-3" &&
						o.ReceiverName == "Updated Name" &&
						o.ItemDescription == "Original Description" // unchanged
				})).Return(nil)
			},
			expectErr: false,
		},
		{
			// Requirement 10.4: only item_description provided → only that field changed
			name:    "only item_description provided leaves receiver fields unchanged",
			orderID: "order-4",
			req: types.UpdateOrderRequest{
				ItemDescription: ptr("Updated Description"),
			},
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("FindOrderByID", mock.Anything, "order-4").Return(&types.Order{
					OrderID:          "order-4",
					Status:           types.OrderStatusAwaitingPickup,
					ReceiverName:     "Original Receiver",
					ReceiverAddress:  "Original Address",
					ReceiverPhone:    "08111111111",
					ReceiverCityCode: "MLG",
					ItemDescription:  "Old Description",
				}, nil)
				m.On("UpdateOrder", mock.Anything, mock.MatchedBy(func(o *types.Order) bool {
					return o.OrderID == "order-4" &&
						o.ItemDescription == "Updated Description" &&
						o.ReceiverName == "Original Receiver" && // unchanged
						o.ReceiverAddress == "Original Address" && // unchanged
						o.ReceiverPhone == "08111111111" && // unchanged
						o.ReceiverCityCode == "MLG" // unchanged
				})).Return(nil)
			},
			expectErr: false,
		},
		{
			// Requirement 10.5: repo update error → error propagated
			name:    "repo update error is propagated",
			orderID: "order-5",
			req: types.UpdateOrderRequest{
				ReceiverName: ptr("New Name"),
			},
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("FindOrderByID", mock.Anything, "order-5").Return(&types.Order{
					OrderID: "order-5",
					Status:  types.OrderStatusAwaitingPickup,
				}, nil)
				m.On("UpdateOrder", mock.Anything, mock.Anything).Return(errors.New("db write failed"))
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "db write failed")
			},
		},
		{
			// Requirement 10.2: repo fetch not-found → not-found propagated
			name:    "repo fetch not-found error is propagated",
			orderID: "order-missing",
			req:     types.UpdateOrderRequest{},
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("FindOrderByID", mock.Anything, "order-missing").Return(nil, handler.ErrNotFound)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.True(t, errors.Is(err, handler.ErrNotFound), "expected not-found sentinel error")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := new(mocks.MockOrderRepository)
			tc.repoSetup(mockRepo)

			svc := service.NewOrderService(mockRepo)
			err := svc.UpdateOrder(context.Background(), tc.orderID, tc.req)

			if tc.expectErr {
				require.Error(t, err)
				if tc.checkErr != nil {
					tc.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
