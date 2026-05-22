package service_test

// Validates: Requirements 9.1, 9.2, 9.3, 9.4, 13.6

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

func TestCancelOrderService(t *testing.T) {
	tests := []struct {
		name        string
		orderID     string
		reason      string
		repoSetup   func(*mocks.MockOrderRepository)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkUpdate func(*testing.T, *mocks.MockOrderRepository)
	}{
		{
			// Requirement 9.1: order in AWAITING_PICKUP → status set to CANCELLED, repo update called
			name:    "order in AWAITING_PICKUP is cancelled and repo update is called",
			orderID: "order-1",
			reason:  "customer request",
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("FindOrderByID", mock.Anything, "order-1").Return(&types.Order{
					OrderID: "order-1",
					Status:  types.OrderStatusAwaitingPickup,
				}, nil)
				m.On("UpdateOrder", mock.Anything, mock.MatchedBy(func(o *types.Order) bool {
					return o.OrderID == "order-1" && o.Status == types.OrderStatusCancelled
				})).Return(nil)
			},
			expectErr: false,
		},
		{
			// Requirement 9.2: order not in AWAITING_PICKUP → conflict error, repo update NOT called
			name:    "order in PICKED_UP status returns conflict error without calling update",
			orderID: "order-2",
			reason:  "customer request",
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("FindOrderByID", mock.Anything, "order-2").Return(&types.Order{
					OrderID: "order-2",
					Status:  types.OrderStatusPickedUp,
				}, nil)
				// UpdateOrder must NOT be called — AssertExpectations will verify this
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.True(t,
					strings.Contains(err.Error(), "order cannot be cancelled: status is not AWAITING_PICKUP"),
					"expected conflict error message, got: %s", err.Error(),
				)
			},
		},
		{
			// Requirement 9.2: order in DELIVERED status → conflict error, repo update NOT called
			name:    "order in DELIVERED status returns conflict error without calling update",
			orderID: "order-3",
			reason:  "customer request",
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("FindOrderByID", mock.Anything, "order-3").Return(&types.Order{
					OrderID: "order-3",
					Status:  types.OrderStatusDelivered,
				}, nil)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.True(t,
					strings.Contains(err.Error(), "order cannot be cancelled: status is not AWAITING_PICKUP"),
					"expected conflict error message, got: %s", err.Error(),
				)
			},
		},
		{
			// Requirement 9.3: repo fetch not-found → not-found propagated
			name:    "repo fetch not-found error is propagated",
			orderID: "order-missing",
			reason:  "customer request",
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("FindOrderByID", mock.Anything, "order-missing").Return(nil, handler.ErrNotFound)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.True(t, errors.Is(err, handler.ErrNotFound), "expected not-found sentinel error")
			},
		},
		{
			// Requirement 9.4: repo update error → error propagated
			name:    "repo update error is propagated",
			orderID: "order-4",
			reason:  "customer request",
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("FindOrderByID", mock.Anything, "order-4").Return(&types.Order{
					OrderID: "order-4",
					Status:  types.OrderStatusAwaitingPickup,
				}, nil)
				m.On("UpdateOrder", mock.Anything, mock.Anything).Return(errors.New("db write failed"))
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "db write failed")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := new(mocks.MockOrderRepository)
			tc.repoSetup(mockRepo)

			svc := service.NewOrderService(mockRepo)
			err := svc.CancelOrder(context.Background(), tc.orderID, tc.reason)

			if tc.expectErr {
				require.Error(t, err)
				if tc.checkErr != nil {
					tc.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
			}

			// AssertExpectations verifies UpdateOrder was NOT called in conflict/not-found cases
			mockRepo.AssertExpectations(t)
		})
	}
}
