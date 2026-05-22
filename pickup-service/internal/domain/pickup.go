package domain

import "time"

// Status represents the current state of a pickup
type Status string

const (
	StatusScheduled     Status = "SCHEDULED"
	StatusAssigned      Status = "ASSIGNED"
	StatusPickedUp      Status = "PICKED_UP"
	StatusFailedAttempt Status = "FAILED_ATTEMPT"
	StatusCancelled     Status = "CANCELLED"
)

// Pickup represents a package pickup request
type Pickup struct {
	PickupID            string    `json:"pickup_id"`
	OrderID             string    `json:"order_id"`
	UserID              string    `json:"user_id"`
	CourierID           string    `json:"courier_id,omitempty"`
	Status              Status    `json:"status"`
	PickupAddress       string    `json:"pickup_address"`
	PickupCityCode      string    `json:"pickup_city_code"`
	ContactName         string    `json:"contact_name"`
	ContactPhone        string    `json:"contact_phone"`
	AttemptCount        int       `json:"attempt_count"`
	EstimatedPickupTime time.Time `json:"estimated_pickup_time"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// PickupAttempt represents a failed pickup attempt
type PickupAttempt struct {
	AttemptID string    `json:"attempt_id"`
	PickupID  string    `json:"pickup_id"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}
