package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"order-service/types"
)

// MockOrderService is a testify mock implementation of types.OrderService.
type MockOrderService struct {
	mock.Mock
}

func (m *MockOrderService) CreateOrder(ctx context.Context, req types.CreateOrderRequest) (*types.CreateOrderResponse, error) {
	args := m.Called(ctx, req)
	if v := args.Get(0); v != nil {
		return v.(*types.CreateOrderResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrderService) GetOrder(ctx context.Context, orderID string) (*types.Order, error) {
	args := m.Called(ctx, orderID)
	if v := args.Get(0); v != nil {
		return v.(*types.Order), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrderService) ListOrders(ctx context.Context, params types.ListOrdersParams) ([]*types.Order, error) {
	args := m.Called(ctx, params)
	if v := args.Get(0); v != nil {
		return v.([]*types.Order), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrderService) CancelOrder(ctx context.Context, orderID string, reason string) error {
	args := m.Called(ctx, orderID, reason)
	return args.Error(0)
}

func (m *MockOrderService) UpdateOrder(ctx context.Context, orderID string, req types.UpdateOrderRequest) error {
	args := m.Called(ctx, orderID, req)
	return args.Error(0)
}
