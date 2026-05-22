package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"order-service/types"
)

// Price base rates per kg by service type.
const (
	baseRateREG     = 9000.0
	baseRateYES     = 19000.0
	baseRateOKE     = 7000.0
	baseRateSameDay = 25000.0

	insuranceSurchargeRate = 0.2  // 20% of base price
	codSurchargeRate       = 0.02 // 2% of base price
)

// OrderService implements types.OrderService.
type OrderService struct {
	repo types.OrderRepository
}

// NewOrderService creates a new OrderService with the given repository.
func NewOrderService(repo types.OrderRepository) *OrderService {
	return &OrderService{repo: repo}
}

// CreateOrder creates a new shipping order, calculates price, and persists it.
func (s *OrderService) CreateOrder(ctx context.Context, req types.CreateOrderRequest) (*types.CreateOrderResponse, error) {
	trackingNumber := generateTrackingNumber()

	price := calculatePrice(req)

	now := time.Now()
	order := &types.Order{
		OrderID:          newUUID(),
		TrackingNumber:   trackingNumber,
		SenderUserID:     req.SenderUserID,
		SenderName:       req.SenderName,
		SenderAddress:    req.SenderAddress,
		SenderPhone:      req.SenderPhone,
		SenderCityCode:   req.SenderCityCode,
		ReceiverName:     req.ReceiverName,
		ReceiverAddress:  req.ReceiverAddress,
		ReceiverPhone:    req.ReceiverPhone,
		ReceiverCityCode: req.ReceiverCityCode,
		Weight:           req.Weight,
		Length:           req.Length,
		Width:            req.Width,
		Height:           req.Height,
		ServiceType:      req.ServiceType,
		Price:            price,
		IsCOD:            req.IsCOD,
		CODAmount:        req.CODAmount,
		Insurance:        req.Insurance,
		ItemDescription:  req.ItemDescription,
		Status:           types.OrderStatusAwaitingPickup,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.repo.SaveOrder(ctx, order); err != nil {
		return nil, fmt.Errorf("save order: %w", err)
	}

	return &types.CreateOrderResponse{
		OrderID:        order.OrderID,
		TrackingNumber: order.TrackingNumber,
		Price:          order.Price,
		EstimatedDays:  estimatedDays(req.ServiceType),
		Status:         order.Status,
	}, nil
}

// GetOrder retrieves an order by ID.
func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*types.Order, error) {
	return s.repo.FindOrderByID(ctx, orderID)
}

// ListOrders returns orders matching the given parameters.
func (s *OrderService) ListOrders(ctx context.Context, params types.ListOrdersParams) ([]*types.Order, error) {
	return s.repo.FindOrders(ctx, params)
}

// CancelOrder cancels an order that is in AWAITING_PICKUP status.
func (s *OrderService) CancelOrder(ctx context.Context, orderID string, reason string) error {
	order, err := s.repo.FindOrderByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status != types.OrderStatusAwaitingPickup {
		return fmt.Errorf("order cannot be cancelled: status is not AWAITING_PICKUP: %w", types.ErrConflict)
	}
	order.Status = types.OrderStatusCancelled
	order.UpdatedAt = time.Now()
	return s.repo.UpdateOrder(ctx, order)
}

// UpdateOrder applies partial updates to an order in AWAITING_PICKUP status.
func (s *OrderService) UpdateOrder(ctx context.Context, orderID string, req types.UpdateOrderRequest) error {
	order, err := s.repo.FindOrderByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status != types.OrderStatusAwaitingPickup {
		return fmt.Errorf("order cannot be updated: status is not AWAITING_PICKUP: %w", types.ErrConflict)
	}
	if req.ReceiverName != nil {
		order.ReceiverName = *req.ReceiverName
	}
	if req.ReceiverAddress != nil {
		order.ReceiverAddress = *req.ReceiverAddress
	}
	if req.ReceiverPhone != nil {
		order.ReceiverPhone = *req.ReceiverPhone
	}
	if req.ReceiverCityCode != nil {
		order.ReceiverCityCode = *req.ReceiverCityCode
	}
	if req.ItemDescription != nil {
		order.ItemDescription = *req.ItemDescription
	}
	order.UpdatedAt = time.Now()
	return s.repo.UpdateOrder(ctx, order)
}

// generateTrackingNumber produces a unique tracking number using random bytes.
func generateTrackingNumber() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

// newUUID generates a random UUID string.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// calculatePrice computes the shipping price based on service type, weight, and surcharges.
func calculatePrice(req types.CreateOrderRequest) float64 {
	var baseRate float64
	switch req.ServiceType {
	case types.ServiceTypeREG:
		baseRate = baseRateREG
	case types.ServiceTypeYES:
		baseRate = baseRateYES
	case types.ServiceTypeOKE:
		baseRate = baseRateOKE
	case types.ServiceTypeSameDay:
		baseRate = baseRateSameDay
	default:
		baseRate = baseRateREG
	}

	price := baseRate * req.Weight
	if price <= 0 {
		price = baseRate // minimum 1 unit
	}

	if req.Insurance {
		price += price * insuranceSurchargeRate
	}
	if req.IsCOD {
		price += price * codSurchargeRate
	}

	return price
}

// estimatedDays returns the estimated delivery days for a service type.
func estimatedDays(st types.ServiceType) int {
	switch st {
	case types.ServiceTypeSameDay:
		return 1
	case types.ServiceTypeYES:
		return 1
	case types.ServiceTypeREG:
		return 3
	case types.ServiceTypeOKE:
		return 5
	default:
		return 3
	}
}
