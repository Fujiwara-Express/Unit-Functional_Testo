package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pickup-service/internal/client"
	"github.com/pickup-service/internal/domain"
	"github.com/pickup-service/internal/repository"
)

// pickupService implements the PickupService interface.
type pickupService struct {
	repo               repository.PickupRepository
	deliveryClient     client.DeliveryClient
	trackingClient     client.TrackingClient
	notificationClient client.NotificationClient
}

// NewPickupService creates a new instance of PickupService.
func NewPickupService(
	repo repository.PickupRepository,
	deliveryClient client.DeliveryClient,
	trackingClient client.TrackingClient,
	notificationClient client.NotificationClient,
) PickupService {
	return &pickupService{
		repo:               repo,
		deliveryClient:     deliveryClient,
		trackingClient:     trackingClient,
		notificationClient: notificationClient,
	}
}

// RequestPickup creates a new pickup request.
func (s *pickupService) RequestPickup(ctx context.Context, req RequestPickupInput) (*RequestPickupOutput, error) {
	now := time.Now()
	p := &domain.Pickup{
		PickupID:            uuid.New().String(),
		OrderID:             req.OrderID,
		UserID:              req.UserID,
		Status:              domain.StatusScheduled,
		PickupAddress:       req.PickupAddress,
		PickupCityCode:      req.PickupCityCode,
		ContactName:         req.ContactName,
		ContactPhone:        req.ContactPhone,
		EstimatedPickupTime: now,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	id, err := s.repo.CreatePickup(ctx, p)
	if err != nil {
		return nil, err
	}

	return &RequestPickupOutput{
		PickupID:            id,
		OrderID:             req.OrderID,
		Status:              domain.StatusScheduled,
		EstimatedPickupTime: now,
		CreatedAt:           now,
	}, nil
}

// AssignCourier assigns a courier to a pickup.
func (s *pickupService) AssignCourier(ctx context.Context, pickupID, courierID string) (*AssignCourierOutput, error) {
	p, err := s.repo.GetPickupByID(ctx, pickupID)
	if err != nil {
		return nil, err
	}

	if err := p.Transition(domain.StatusAssigned); err != nil {
		return nil, err
	}
	p.CourierID = courierID
	p.UpdatedAt = time.Now()

	if err := s.repo.UpdatePickup(ctx, p); err != nil {
		return nil, err
	}

	return &AssignCourierOutput{
		PickupID:  p.PickupID,
		CourierID: courierID,
		Status:    p.Status,
	}, nil
}

// UpdatePickupStatus updates the status of a pickup.
func (s *pickupService) UpdatePickupStatus(ctx context.Context, pickupID string, status domain.Status) (*UpdateStatusOutput, error) {
	p, err := s.repo.GetPickupByID(ctx, pickupID)
	if err != nil {
		return nil, err
	}

	if err := p.Transition(status); err != nil {
		return nil, err
	}
	now := time.Now()
	p.UpdatedAt = now

	if err := s.repo.UpdatePickup(ctx, p); err != nil {
		return nil, err
	}

	// Side effects based on new status
	switch status {
	case domain.StatusPickedUp:
		if err := s.trackingClient.PublishPickedUpEvent(ctx, p.PickupID, p.OrderID, now); err != nil {
			return nil, err
		}
	case domain.StatusFailedAttempt:
		if err := s.notificationClient.NotifyCourierEnRoute(ctx, p.ContactName, p.ContactPhone, p.CourierID); err != nil {
			return nil, err
		}
	}

	return &UpdateStatusOutput{
		PickupID:  p.PickupID,
		Status:    p.Status,
		Timestamp: now,
	}, nil
}

// GetPickup retrieves a pickup by ID.
func (s *pickupService) GetPickup(ctx context.Context, pickupID string) (*domain.Pickup, error) {
	return s.repo.GetPickupByID(ctx, pickupID)
}

// ListPickups retrieves a list of pickups based on filters.
func (s *pickupService) ListPickups(ctx context.Context, filters ListFilters) ([]*domain.Pickup, error) {
	return s.repo.ListPickups(ctx, repository.ListFilters{
		CourierID: filters.CourierID,
		Status:    filters.Status,
		Date:      filters.Date,
	})
}

// CancelPickup cancels a pickup.
func (s *pickupService) CancelPickup(ctx context.Context, pickupID string) (*CancelPickupOutput, error) {
	p, err := s.repo.GetPickupByID(ctx, pickupID)
	if err != nil {
		return nil, err
	}

	if p.Status != domain.StatusScheduled {
		return nil, fmt.Errorf("%w: can only cancel SCHEDULED pickups, current status is %s", domain.ErrConflict, p.Status)
	}

	if err := p.Transition(domain.StatusCancelled); err != nil {
		return nil, err
	}
	p.UpdatedAt = time.Now()

	if err := s.repo.UpdatePickup(ctx, p); err != nil {
		return nil, err
	}

	return &CancelPickupOutput{
		PickupID: p.PickupID,
		Status:   p.Status,
	}, nil
}
