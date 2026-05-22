package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"order-service/mocks"
	"order-service/service"
	"order-service/types"
)

func baseRequest() types.CreateOrderRequest {
	return types.CreateOrderRequest{
		SenderUserID:     "user-1",
		SenderName:       "Alice",
		SenderAddress:    "Jl. Merdeka 1",
		SenderPhone:      "081234567890",
		SenderCityCode:   "JKT",
		ReceiverName:     "Bob",
		ReceiverAddress:  "Jl. Sudirman 2",
		ReceiverPhone:    "089876543210",
		ReceiverCityCode: "SBY",
		Weight:           2.0,
		Length:           10,
		Width:            10,
		Height:           10,
		ServiceType:      types.ServiceTypeREG,
		IsCOD:            false,
		CODAmount:        0,
		Insurance:        false,
		ItemDescription:  "Electronics",
	}
}

func TestCreateOrderService(t *testing.T) {
	tests := []struct {
		name        string
		req         types.CreateOrderRequest
		repoSetup   func(*mocks.MockOrderRepository)
		expectErr   bool
		checkResult func(*testing.T, *types.CreateOrderResponse)
	}{
		{
			name: "valid request returns tracking number, AWAITING_PICKUP status, and positive price",
			req:  baseRequest(),
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("SaveOrder", mock.Anything, mock.Anything).Return(nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, resp *types.CreateOrderResponse) {
				assert.NotEmpty(t, resp.TrackingNumber, "tracking number must be non-empty")
				assert.Equal(t, types.OrderStatusAwaitingPickup, resp.Status)
				assert.Greater(t, resp.Price, 0.0, "price must be positive")
			},
		},
		{
			name: "REG service type produces non-zero price",
			req: func() types.CreateOrderRequest {
				r := baseRequest()
				r.ServiceType = types.ServiceTypeREG
				return r
			}(),
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("SaveOrder", mock.Anything, mock.Anything).Return(nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, resp *types.CreateOrderResponse) {
				assert.Greater(t, resp.Price, 0.0)
			},
		},
		{
			name: "YES service type produces non-zero price",
			req: func() types.CreateOrderRequest {
				r := baseRequest()
				r.ServiceType = types.ServiceTypeYES
				return r
			}(),
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("SaveOrder", mock.Anything, mock.Anything).Return(nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, resp *types.CreateOrderResponse) {
				assert.Greater(t, resp.Price, 0.0)
			},
		},
		{
			name: "OKE service type produces non-zero price",
			req: func() types.CreateOrderRequest {
				r := baseRequest()
				r.ServiceType = types.ServiceTypeOKE
				return r
			}(),
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("SaveOrder", mock.Anything, mock.Anything).Return(nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, resp *types.CreateOrderResponse) {
				assert.Greater(t, resp.Price, 0.0)
			},
		},
		{
			name: "SAME_DAY service type produces non-zero price",
			req: func() types.CreateOrderRequest {
				r := baseRequest()
				r.ServiceType = types.ServiceTypeSameDay
				return r
			}(),
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("SaveOrder", mock.Anything, mock.Anything).Return(nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, resp *types.CreateOrderResponse) {
				assert.Greater(t, resp.Price, 0.0)
			},
		},
		{
			name: "insurance=true results in higher price than insurance=false",
			req: func() types.CreateOrderRequest {
				r := baseRequest()
				r.Insurance = true
				return r
			}(),
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("SaveOrder", mock.Anything, mock.Anything).Return(nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, resp *types.CreateOrderResponse) {
				// Price without insurance for REG, weight=2: 9000*2 = 18000
				basePrice := 9000.0 * 2.0
				assert.Greater(t, resp.Price, basePrice, "insurance surcharge must increase price")
			},
		},
		{
			name: "is_cod=true results in higher price than is_cod=false",
			req: func() types.CreateOrderRequest {
				r := baseRequest()
				r.IsCOD = true
				r.CODAmount = 50000
				return r
			}(),
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("SaveOrder", mock.Anything, mock.Anything).Return(nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, resp *types.CreateOrderResponse) {
				// Price without COD for REG, weight=2: 9000*2 = 18000
				basePrice := 9000.0 * 2.0
				assert.Greater(t, resp.Price, basePrice, "COD surcharge must increase price")
			},
		},
		{
			name: "repo save error is returned to caller",
			req:  baseRequest(),
			repoSetup: func(m *mocks.MockOrderRepository) {
				m.On("SaveOrder", mock.Anything, mock.Anything).Return(errors.New("db connection failed"))
			},
			expectErr:   true,
			checkResult: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := new(mocks.MockOrderRepository)
			tc.repoSetup(mockRepo)

			svc := service.NewOrderService(mockRepo)
			result, err := svc.CreateOrder(context.Background(), tc.req)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tc.checkResult != nil {
					tc.checkResult(t, result)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
