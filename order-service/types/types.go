package types

import "time"

// OrderStatus represents the lifecycle state of an Order.
type OrderStatus string

const (
	OrderStatusCreated        OrderStatus = "CREATED"
	OrderStatusAwaitingPickup OrderStatus = "AWAITING_PICKUP"
	OrderStatusPickedUp       OrderStatus = "PICKED_UP"
	OrderStatusInTransit      OrderStatus = "IN_TRANSIT"
	OrderStatusDelivered      OrderStatus = "DELIVERED"
	OrderStatusFailed         OrderStatus = "FAILED"
	OrderStatusCancelled      OrderStatus = "CANCELLED"
)

// ServiceType represents the shipping service tier.
type ServiceType string

const (
	ServiceTypeREG     ServiceType = "REG"
	ServiceTypeYES     ServiceType = "YES"
	ServiceTypeOKE     ServiceType = "OKE"
	ServiceTypeSameDay ServiceType = "SAME_DAY"
)

// Order represents a shipping order entity.
type Order struct {
	OrderID          string
	TrackingNumber   string
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
	Price            float64
	IsCOD            bool
	CODAmount        float64
	Insurance        bool
	ItemDescription  string
	Status           OrderStatus
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
