package service_test

// Validates: Requirements 6.7, 6.1, 6.2, 6.4, 6.5

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	"order-service/mocks"
	"order-service/service"
	"order-service/types"
)

// genServiceType returns a rapid generator for valid ServiceType values.
func genServiceType(t *rapid.T) types.ServiceType {
	return rapid.SampledFrom([]types.ServiceType{
		types.ServiceTypeREG,
		types.ServiceTypeYES,
		types.ServiceTypeOKE,
		types.ServiceTypeSameDay,
	}).Draw(t, "serviceType")
}

// genValidCreateOrderRequest generates a CreateOrderRequest with positive weight and dimensions.
func genValidCreateOrderRequest(t *rapid.T) types.CreateOrderRequest {
	weight := rapid.Float64Range(0.1, 100.0).Draw(t, "weight")
	length := rapid.Float64Range(1.0, 200.0).Draw(t, "length")
	width := rapid.Float64Range(1.0, 200.0).Draw(t, "width")
	height := rapid.Float64Range(1.0, 200.0).Draw(t, "height")
	serviceType := genServiceType(t)

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
		Weight:           weight,
		Length:           length,
		Width:            width,
		Height:           height,
		ServiceType:      serviceType,
		IsCOD:            false,
		CODAmount:        0,
		Insurance:        false,
		ItemDescription:  "Test item",
	}
}

// TestProperty1_PriceAndTrackingInvariants verifies that for any valid CreateOrderRequest,
// the response has price > 0 and a non-empty TrackingNumber.
// Validates: Requirements 6.7, 6.1
func TestProperty1_PriceAndTrackingInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		req := genValidCreateOrderRequest(t)

		mockRepo := new(mocks.MockOrderRepository)
		mockRepo.On("SaveOrder", mock.Anything, mock.Anything).Return(nil)

		svc := service.NewOrderService(mockRepo)
		resp, err := svc.CreateOrder(context.Background(), req)

		require.NoError(t, err)
		require.Greater(t, resp.Price, 0.0, "price must be > 0 for any valid request")
		require.NotEmpty(t, resp.TrackingNumber, "tracking number must be non-empty")
	})
}

// TestProperty2_InitialStatusIsAwaitingPickup verifies that for any valid CreateOrderRequest,
// the persisted order has Status == AWAITING_PICKUP.
// Validates: Requirements 6.2
func TestProperty2_InitialStatusIsAwaitingPickup(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		req := genValidCreateOrderRequest(t)

		mockRepo := new(mocks.MockOrderRepository)
		mockRepo.On("SaveOrder", mock.Anything, mock.Anything).Return(nil)

		svc := service.NewOrderService(mockRepo)
		resp, err := svc.CreateOrder(context.Background(), req)

		require.NoError(t, err)
		require.Equal(t, types.OrderStatusAwaitingPickup, resp.Status,
			"initial order status must be AWAITING_PICKUP")
	})
}

// TestProperty3_InsuranceSurchargeMonotonicity verifies that price with insurance=true
// is strictly greater than price with insurance=false, all other fields equal.
// Validates: Requirements 6.4
func TestProperty3_InsuranceSurchargeMonotonicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		base := genValidCreateOrderRequest(t)

		withInsurance := base
		withInsurance.Insurance = true

		withoutInsurance := base
		withoutInsurance.Insurance = false

		mockRepoWith := new(mocks.MockOrderRepository)
		mockRepoWith.On("SaveOrder", mock.Anything, mock.Anything).Return(nil)

		mockRepoWithout := new(mocks.MockOrderRepository)
		mockRepoWithout.On("SaveOrder", mock.Anything, mock.Anything).Return(nil)

		svcWith := service.NewOrderService(mockRepoWith)
		respWith, err := svcWith.CreateOrder(context.Background(), withInsurance)
		require.NoError(t, err)

		svcWithout := service.NewOrderService(mockRepoWithout)
		respWithout, err := svcWithout.CreateOrder(context.Background(), withoutInsurance)
		require.NoError(t, err)

		require.Greater(t, respWith.Price, respWithout.Price,
			"price with insurance must be strictly greater than price without insurance")
	})
}

// TestProperty4_CODSurchargeMonotonicity verifies that price with is_cod=true and cod_amount>0
// is strictly greater than price without COD, all other fields equal.
// Validates: Requirements 6.5
func TestProperty4_CODSurchargeMonotonicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		base := genValidCreateOrderRequest(t)
		codAmount := rapid.Float64Range(1.0, 10000000.0).Draw(t, "codAmount")

		withCOD := base
		withCOD.IsCOD = true
		withCOD.CODAmount = codAmount

		withoutCOD := base
		withoutCOD.IsCOD = false
		withoutCOD.CODAmount = 0

		mockRepoWith := new(mocks.MockOrderRepository)
		mockRepoWith.On("SaveOrder", mock.Anything, mock.Anything).Return(nil)

		mockRepoWithout := new(mocks.MockOrderRepository)
		mockRepoWithout.On("SaveOrder", mock.Anything, mock.Anything).Return(nil)

		svcWith := service.NewOrderService(mockRepoWith)
		respWith, err := svcWith.CreateOrder(context.Background(), withCOD)
		require.NoError(t, err)

		svcWithout := service.NewOrderService(mockRepoWithout)
		respWithout, err := svcWithout.CreateOrder(context.Background(), withoutCOD)
		require.NoError(t, err)

		require.Greater(t, respWith.Price, respWithout.Price,
			"price with COD must be strictly greater than price without COD")
	})
}
