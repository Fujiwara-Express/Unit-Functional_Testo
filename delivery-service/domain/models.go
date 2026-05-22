package domain

import "time"

// JobStatus represents the status of a delivery job.
type JobStatus string

const (
	JobStatusAssigned       JobStatus = "ASSIGNED"
	JobStatusOutForDelivery JobStatus = "OUT_FOR_DELIVERY"
	JobStatusDelivered      JobStatus = "DELIVERED"
	JobStatusFailed         JobStatus = "FAILED"
	JobStatusReturned       JobStatus = "RETURNED"
)

// String returns the string representation of a JobStatus.
func (s JobStatus) String() string { return string(s) }

// VehicleType represents the type of vehicle a courier uses.
type VehicleType string

const (
	VehicleTypeMotor VehicleType = "MOTOR"
)

// Courier represents a courier record.
type Courier struct {
	CourierID   string      `json:"courier_id"`
	Name        string      `json:"name"`
	Phone       string      `json:"phone"`
	HubID       string      `json:"hub_id"`
	VehicleType VehicleType `json:"vehicle_type"`
	IsAvailable bool        `json:"is_available"`
	CurrentLat  float64     `json:"current_lat"`
	CurrentLng  float64     `json:"current_lng"`
}

// DeliveryJob represents a delivery job record.
type DeliveryJob struct {
	JobID          string     `json:"job_id"`
	TrackingNumber string     `json:"tracking_number"`
	CourierID      string     `json:"courier_id"`
	HubID          string     `json:"hub_id"`
	Status         JobStatus  `json:"status"`
	AttemptCount   int        `json:"attempt_count"`
	ProofPhotoURL  string     `json:"proof_photo_url,omitempty"`
	RecipientName  string     `json:"recipient_name,omitempty"`
	Notes          string     `json:"notes,omitempty"`
	AssignedAt     time.Time  `json:"assigned_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// DeliveryRoute is the optimized route response returned by Routing_Client.
type DeliveryRoute struct {
	TotalStops                    int         `json:"total_stops"`
	TotalDistanceKm               float64     `json:"total_distance_km"`
	EstimatedTotalDurationMinutes int         `json:"estimated_total_duration_minutes"`
	OptimizedRoute                []RouteStop `json:"optimized_route"`
}

// RouteStop represents a single stop in an optimized delivery route.
type RouteStop struct {
	Sequence           int       `json:"sequence"`
	TrackingNumber     string    `json:"tracking_number"`
	DeliveryID         string    `json:"delivery_id"`
	RecipientName      string    `json:"recipient_name"`
	Address            string    `json:"address"`
	Lat                float64   `json:"lat"`
	Lng                float64   `json:"lng"`
	EstimatedArrival   time.Time `json:"estimated_arrival"`
	DistanceFromPrevKm float64   `json:"distance_from_prev_km"`
}

// RegisterCourierRequest is the request payload for registering a new courier.
type RegisterCourierRequest struct {
	Name        string      `json:"name"`
	Phone       string      `json:"phone"`
	HubID       string      `json:"hub_id"`
	VehicleType VehicleType `json:"vehicle_type"`
}

// CourierUpdate holds the fields that can be updated on a courier.
type CourierUpdate struct {
	IsAvailable *bool    `json:"is_available,omitempty"`
	CurrentLat  *float64 `json:"current_lat,omitempty"`
	CurrentLng  *float64 `json:"current_lng,omitempty"`
}

// CourierFilter holds filter parameters for listing couriers.
type CourierFilter struct {
	HubID       string `json:"hub_id,omitempty"`
	IsAvailable *bool  `json:"is_available,omitempty"`
	CityCode    string `json:"city_code,omitempty"`
}

// AssignRequest is the request payload for assigning a courier to a delivery job.
type AssignRequest struct {
	TrackingNumber string `json:"tracking_number"`
	HubID          string `json:"hub_id"`
	CourierID      string `json:"courier_id"`
}

// StatusUpdateRequest is the request payload for updating a delivery job status.
type StatusUpdateRequest struct {
	TrackingNumber string    `json:"tracking_number"`
	CourierID      string    `json:"courier_id"`
	Status         string    `json:"status"`
	Notes          string    `json:"notes,omitempty"`
	ProofPhotoURL  string    `json:"proof_photo_url,omitempty"`
	RecipientName  string    `json:"recipient_name,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// JobStatusUpdate holds the fields to update on a delivery job status change.
type JobStatusUpdate struct {
	Status        JobStatus
	AttemptCount  int
	ProofPhotoURL string
	RecipientName string
	Notes         string
	CompletedAt   *time.Time
}
