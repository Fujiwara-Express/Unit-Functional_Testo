package client

import "context"

// Courier represents a courier available for delivery.
type Courier struct {
	CourierID string `json:"courier_id"`
	Name      string `json:"name"`
	CityCode  string `json:"city_code"`
}

// DeliveryClient defines the interface for interacting with the delivery service.
type DeliveryClient interface {
	GetAvailableCouriers(ctx context.Context, cityCode string) ([]Courier, error)
}
