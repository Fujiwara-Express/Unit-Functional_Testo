package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"order-service/types"
)

// MockOrderRepository is a testify mock implementation of types.OrderRepository.
type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) SaveOrder(ctx context.Context, order *types.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderRepository) FindOrderByID(ctx context.Context, orderID string) (*types.Order, error) {
	args := m.Called(ctx, orderID)
	if v := args.Get(0); v != nil {
		return v.(*types.Order), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrderRepository) FindOrders(ctx context.Context, params types.ListOrdersParams) ([]*types.Order, error) {
	args := m.Called(ctx, params)
	if v := args.Get(0); v != nil {
		return v.([]*types.Order), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrderRepository) UpdateOrder(ctx context.Context, order *types.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}
