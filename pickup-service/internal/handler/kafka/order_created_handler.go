package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/pickup-service/internal/service"
)

// OrderCreatedEvent represents the Kafka message payload for an order creation event.
type OrderCreatedEvent struct {
	OrderID        string `json:"order_id"`
	UserID         string `json:"user_id"`
	PickupAddress  string `json:"pickup_address"`
	PickupCityCode string `json:"pickup_city_code"`
	ContactName    string `json:"contact_name"`
	ContactPhone   string `json:"contact_phone"`
}

// validate checks that all required fields are present in the event.
func (e *OrderCreatedEvent) validate() error {
	if e.OrderID == "" {
		return fmt.Errorf("missing required field order_id")
	}
	if e.UserID == "" {
		return fmt.Errorf("missing required field user_id")
	}
	if e.PickupAddress == "" {
		return fmt.Errorf("missing required field pickup_address")
	}
	if e.PickupCityCode == "" {
		return fmt.Errorf("missing required field pickup_city_code")
	}
	if e.ContactName == "" {
		return fmt.Errorf("missing required field contact_name")
	}
	if e.ContactPhone == "" {
		return fmt.Errorf("missing required field contact_phone")
	}
	return nil
}

// OrderCreatedHandler handles order created Kafka events.
type OrderCreatedHandler struct {
	svc service.PickupService
}

// NewOrderCreatedHandler creates a new OrderCreatedHandler.
func NewOrderCreatedHandler(svc service.PickupService) *OrderCreatedHandler {
	return &OrderCreatedHandler{svc: svc}
}

// HandleOrderCreatedEvent processes an order created event message.
func (h *OrderCreatedHandler) HandleOrderCreatedEvent(msg []byte) error {
	var event OrderCreatedEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		log.Printf("failed to unmarshal order created event: %v", err)
		return err
	}

	if err := event.validate(); err != nil {
		log.Printf("invalid order created event: %v", err)
		return err
	}

	_, err := h.svc.RequestPickup(context.Background(), service.RequestPickupInput{
		OrderID:        event.OrderID,
		UserID:         event.UserID,
		PickupAddress:  event.PickupAddress,
		PickupCityCode: event.PickupCityCode,
		ContactName:    event.ContactName,
		ContactPhone:   event.ContactPhone,
	})
	return err
}
