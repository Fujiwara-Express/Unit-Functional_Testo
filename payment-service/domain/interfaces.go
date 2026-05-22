package domain

import "context"

// Payment_Repository defines the persistence layer for payments and COD collections.
//
//go:generate mockgen -source=interfaces.go -destination=../mocks/mock_repository.go -package=mocks
type Payment_Repository interface {
	// CreatePayment persists a new payment record.
	CreatePayment(ctx context.Context, payment *Payment) error

	// GetPaymentByID retrieves a payment by its primary key.
	GetPaymentByID(ctx context.Context, paymentID string) (*Payment, error)

	// GetPaymentByOrderID retrieves a payment by its associated order ID.
	GetPaymentByOrderID(ctx context.Context, orderID string) (*Payment, error)

	// UpdatePaymentStatus updates the status and external reference of a payment.
	UpdatePaymentStatus(ctx context.Context, orderID string, status PaymentStatus, externalRef string) error

	// CreateCodCollection persists a new COD collection record.
	CreateCodCollection(ctx context.Context, collection *CodCollection) error

	// GetPaymentByExternalRef retrieves a payment by its external gateway reference.
	GetPaymentByExternalRef(ctx context.Context, externalRef string) (*Payment, error)
}

// Payment_Gateway_Client defines the interface for communicating with an external payment gateway.
//
//go:generate mockgen -source=interfaces.go -destination=../mocks/mock_gateway_client.go -package=mocks
type Payment_Gateway_Client interface {
	// Charge initiates a charge request with the payment gateway.
	Charge(ctx context.Context, req *ChargeRequest) (*ChargeResponse, error)
}

// Kafka_Publisher defines the interface for publishing domain events to Kafka.
//
//go:generate mockgen -source=interfaces.go -destination=../mocks/mock_kafka_publisher.go -package=mocks
type Kafka_Publisher interface {
	// Publish sends a payment event to the specified Kafka topic.
	Publish(ctx context.Context, topic string, event *PaymentEvent) error
}
