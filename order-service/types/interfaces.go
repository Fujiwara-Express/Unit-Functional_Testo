package types

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned when an operation is rejected due to the resource's current state.
var ErrConflict = errors.New("conflict")

// OrderService defines the business logic interface for order operations.
type OrderService interface {
	CreateOrder(ctx context.Context, req CreateOrderRequest) (*CreateOrderResponse, error)
	GetOrder(ctx context.Context, orderID string) (*Order, error)
	ListOrders(ctx context.Context, params ListOrdersParams) ([]*Order, error)
	CancelOrder(ctx context.Context, orderID string, reason string) error
	UpdateOrder(ctx context.Context, orderID string, req UpdateOrderRequest) error
}

// OrderRepository defines the data access interface for order persistence.
type OrderRepository interface {
	SaveOrder(ctx context.Context, order *Order) error
	FindOrderByID(ctx context.Context, orderID string) (*Order, error)
	FindOrders(ctx context.Context, params ListOrdersParams) ([]*Order, error)
	UpdateOrder(ctx context.Context, order *Order) error
}

// CreateOrderRequest holds the input fields for creating a new order.
type CreateOrderRequest struct {
	SenderUserID     string
	SenderName       string
	SenderAddress    string
	SenderPhone      string
	SenderCityCode   string
	ReceiverName     string
	ReceiverAddress  string
	ReceiverPhone    string
	ReceiverCityCode string
	Weight           float64
	Length           float64
	Width            float64
	Height           float64
	ServiceType      ServiceType
	IsCOD            bool
	CODAmount        float64
	Insurance        bool
	ItemDescription  string
}

// UpdateOrderRequest holds the optional fields that can be updated on an existing order.
type UpdateOrderRequest struct {
	ReceiverName     *string
	ReceiverAddress  *string
	ReceiverPhone    *string
	ReceiverCityCode *string
	ItemDescription  *string
}

// ListOrdersParams holds the filter and pagination parameters for listing orders.
type ListOrdersParams struct {
	UserID string
	Status OrderStatus
	Page   int
	Limit  int
}

// CreateOrderResponse holds the result returned after successfully creating an order.
type CreateOrderResponse struct {
	OrderID        string
	TrackingNumber string
	Price          float64
	EstimatedDays  int
	Status         OrderStatus
}
