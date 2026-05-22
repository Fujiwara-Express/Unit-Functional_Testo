package service

import (
	"context"
	"time"

	"github.com/pickup-service/internal/domain"
)

// RequestPickupInput contains the input for requesting a pickup.
type RequestPickupInput struct {
	OrderID        string
	UserID         string
	PickupAddress  string
	PickupCityCode string
	ContactName    string
	ContactPhone   string
}

// RequestPickupOutput contains the result of a pickup request.
type RequestPickupOutput struct {
	PickupID            string
	OrderID             string
	Status              domain.Status
	EstimatedPickupTime time.Time
	CreatedAt           time.Time
}

// AssignCourierOutput contains the result of assigning a courier.
type AssignCourierOutput struct {
	PickupID  string
	CourierID string
	Status    domain.Status
}

// UpdateStatusOutput contains the result of updating a pickup status.
type UpdateStatusOutput struct {
	PickupID  string
	Status    domain.Status
	Timestamp time.Time
}

// CancelPickupOutput contains the result of cancelling a pickup.
type CancelPickupOutput struct {
	PickupID string
	Status   domain.Status
}

// ListFilters contains optional filters for listing pickups.
type ListFilters struct {
	CourierID string
	Status    string
	Date      string
}

// PickupService defines the business logic interface for pickup operations.
type PickupService interface {
	RequestPickup(ctx context.Context, req RequestPickupInput) (*RequestPickupOutput, error)
	AssignCourier(ctx context.Context, pickupID, courierID string) (*AssignCourierOutput, error)
	UpdatePickupStatus(ctx context.Context, pickupID string, status domain.Status) (*UpdateStatusOutput, error)
	GetPickup(ctx context.Context, pickupID string) (*domain.Pickup, error)
	ListPickups(ctx context.Context, filters ListFilters) ([]*domain.Pickup, error)
	CancelPickup(ctx context.Context, pickupID string) (*CancelPickupOutput, error)
}
