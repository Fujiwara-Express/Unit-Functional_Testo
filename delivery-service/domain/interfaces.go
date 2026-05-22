package domain

import "context"

// Delivery_Repository defines the persistence layer for couriers and delivery jobs.
//
//go:generate mockgen -source=interfaces.go -destination=../mocks/mock_repository.go -package=mocks
type Delivery_Repository interface {
	// CreateCourier persists a new courier record.
	CreateCourier(ctx context.Context, courier *Courier) error

	// GetCourierByID retrieves a courier by its primary key.
	GetCourierByID(ctx context.Context, courierID string) (*Courier, error)

	// UpdateCourier updates fields on an existing courier record.
	UpdateCourier(ctx context.Context, courierID string, update *CourierUpdate) error

	// ListCouriers retrieves couriers matching the given filter.
	ListCouriers(ctx context.Context, filter *CourierFilter) ([]*Courier, error)

	// CreateDeliveryJob persists a new delivery job record.
	CreateDeliveryJob(ctx context.Context, job *DeliveryJob) error

	// GetDeliveryJobByID retrieves a delivery job by its primary key.
	GetDeliveryJobByID(ctx context.Context, jobID string) (*DeliveryJob, error)

	// GetDeliveryJobByTrackingNumber retrieves a delivery job by its tracking number.
	GetDeliveryJobByTrackingNumber(ctx context.Context, trackingNumber string) (*DeliveryJob, error)

	// UpdateDeliveryJobStatus updates the status and related fields of a delivery job.
	UpdateDeliveryJobStatus(ctx context.Context, jobID string, update *JobStatusUpdate) error

	// GetJobsByCourierID retrieves all delivery jobs assigned to a courier.
	GetJobsByCourierID(ctx context.Context, courierID string) ([]*DeliveryJob, error)
}

// Routing_Client defines the interface for the external routing service.
//
//go:generate mockgen -source=interfaces.go -destination=../mocks/mock_routing_client.go -package=mocks
type Routing_Client interface {
	// GetCourierRoute retrieves the optimized delivery route for a courier.
	GetCourierRoute(ctx context.Context, courierID string) (*DeliveryRoute, error)
}
