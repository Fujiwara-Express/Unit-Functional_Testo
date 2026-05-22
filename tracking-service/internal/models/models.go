package models

import "time"

// Status is the enumerated lifecycle state of a packet.
type Status string

const (
	StatusCreated        Status = "CREATED"
	StatusPickedUp       Status = "PICKED_UP"
	StatusArrivedAtHub   Status = "ARRIVED_AT_HUB"
	StatusInTransit      Status = "IN_TRANSIT"
	StatusOutForDelivery Status = "OUT_FOR_DELIVERY"
	StatusDelivered      Status = "DELIVERED"
	StatusFailedDelivery Status = "FAILED_DELIVERY"
	StatusReturned       Status = "RETURNED"
)

// TrackingEvent is the immutable domain record stored in tracking_events.
type TrackingEvent struct {
	EventID          string    `json:"event_id"`
	TrackingNumber   string    `json:"tracking_number"`
	Status           Status    `json:"status"`
	Location         string    `json:"location,omitempty"`
	HubID            string    `json:"hub_id,omitempty"`
	Notes            string    `json:"notes,omitempty"`
	CreatedByService string    `json:"created_by_service"`
	Timestamp        time.Time `json:"timestamp"`
}

// TrackingSummary is the current-state projection stored in tracking_summary.
type TrackingSummary struct {
	TrackingNumber    string     `json:"tracking_number"`
	CurrentStatus     Status     `json:"current_status"`
	LastLocation      string     `json:"last_location,omitempty"`
	EstimatedDelivery *time.Time `json:"estimated_delivery"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// CreateEventRequest is the body for POST /tracking/events.
type CreateEventRequest struct {
	TrackingNumber string `json:"tracking_number"`
	Status         string `json:"status"`
	Location       string `json:"location,omitempty"`
	HubID          string `json:"hub_id,omitempty"`
	Notes          string `json:"notes,omitempty"`
	Timestamp      string `json:"timestamp"` // RFC 3339
}

// CreateEventResponse is the 201 body for POST /tracking/events.
type CreateEventResponse struct {
	EventID string `json:"event_id"`
}

// HistoryEntry is one element in the history array.
type HistoryEntry struct {
	Status    Status    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// TrackingHistoryResponse is the 200 body for GET /tracking/{tracking_number}.
type TrackingHistoryResponse struct {
	TrackingNumber    string         `json:"tracking_number"`
	CurrentStatus     Status         `json:"current_status"`
	EstimatedDelivery *time.Time     `json:"estimated_delivery"`
	History           []HistoryEntry `json:"history"`
}

// ErrorResponse is the body for all 4xx/5xx responses.
type ErrorResponse struct {
	Error         string `json:"error"`
	CorrelationID string `json:"correlation_id"`
}

// ValidationError holds a field-level validation failure.
type ValidationError struct {
	Field   string
	Message string
}
