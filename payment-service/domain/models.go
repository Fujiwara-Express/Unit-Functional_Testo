package domain

import "time"

// PaymentStatus represents the status of a payment.
type PaymentStatus string

const (
	PaymentStatusPending  PaymentStatus = "PENDING"
	PaymentStatusSuccess  PaymentStatus = "SUCCESS"
	PaymentStatusFailed   PaymentStatus = "FAILED"
	PaymentStatusRefunded PaymentStatus = "REFUNDED"
)

// PaymentMethod represents the method used for a payment.
type PaymentMethod string

const (
	PaymentMethodTransfer       PaymentMethod = "TRANSFER"
	PaymentMethodVirtualAccount PaymentMethod = "VIRTUAL_ACCOUNT"
	PaymentMethodQRIS           PaymentMethod = "QRIS"
	PaymentMethodCOD            PaymentMethod = "COD"
)

// RemittanceStatus represents the remittance status of a COD collection.
type RemittanceStatus string

const (
	RemittanceStatusPending  RemittanceStatus = "PENDING"
	RemittanceStatusRemitted RemittanceStatus = "REMITTED"
)

// Payment represents a payment record.
type Payment struct {
	PaymentID   string        `json:"payment_id"`
	OrderID     string        `json:"order_id"`
	UserID      string        `json:"user_id"`
	Amount      float64       `json:"amount"`
	Method      PaymentMethod `json:"method"`
	Status      PaymentStatus `json:"status"`
	ExternalRef string        `json:"external_ref,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// CodCollection represents a COD collection record.
type CodCollection struct {
	CollectionID     string           `json:"collection_id"`
	OrderID          string           `json:"order_id"`
	CourierID        string           `json:"courier_id"`
	AmountCollected  float64          `json:"amount_collected"`
	CollectedAt      time.Time        `json:"collected_at"`
	RemittanceStatus RemittanceStatus `json:"remittance_status"`
}

// ChargeRequest is the request payload for charging a payment.
type ChargeRequest struct {
	OrderID string        `json:"order_id"`
	Amount  float64       `json:"amount"`
	Method  PaymentMethod `json:"method"`
	UserID  string        `json:"user_id"`
}

// ChargeResponse is the response from the payment gateway for a charge.
type ChargeResponse struct {
	ExternalRef string    `json:"external_ref"`
	Status      string    `json:"status"`
	VANumber    string    `json:"va_number,omitempty"`
	ExpiredAt   time.Time `json:"expired_at,omitempty"`
}

// PaymentEventType represents the type of a payment event published to Kafka.
type PaymentEventType string

const (
	PaymentEventSuccess PaymentEventType = "PAYMENT_SUCCESS"
	PaymentEventFailed  PaymentEventType = "PAYMENT_FAILED"
)

// PaymentEvent is the event published to Kafka after a payment status change.
type PaymentEvent struct {
	EventType   PaymentEventType `json:"event_type"`
	PaymentID   string           `json:"payment_id"`
	OrderID     string           `json:"order_id"`
	UserID      string           `json:"user_id"`
	Amount      float64          `json:"amount"`
	Method      PaymentMethod    `json:"method"`
	Status      PaymentStatus    `json:"status"`
	ExternalRef string           `json:"external_ref,omitempty"`
	OccurredAt  time.Time        `json:"occurred_at"`
}
