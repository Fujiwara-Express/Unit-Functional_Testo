package repository

import (
	"context"

	"github.com/pickup-service/internal/domain"
)

// ListFilters contains optional filters for listing pickups
type ListFilters struct {
	CourierID string
	Status    string
	Date      string
}

// PickupRepository defines the interface for pickup data operations
type PickupRepository interface {
	CreatePickup(ctx context.Context, p *domain.Pickup) (string, error)
	GetPickupByID(ctx context.Context, id string) (*domain.Pickup, error)
	UpdatePickup(ctx context.Context, p *domain.Pickup) error
	ListPickups(ctx context.Context, filters ListFilters) ([]*domain.Pickup, error)
	CreatePickupAttempt(ctx context.Context, a *domain.PickupAttempt) error
}
