package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/notification-service/internal/domain"
	"github.com/notification-service/internal/service"
)

// TrackingStatusUpdatedEvent represents the Kafka message payload for a tracking status update.
type TrackingStatusUpdatedEvent struct {
	UserID         string `json:"user_id"`
	TrackingNumber string `json:"tracking_number"`
	EventType      string `json:"event_type"`
}

// TrackingStatusHandler handles tracking.status_updated Kafka events.
type TrackingStatusHandler struct {
	svc service.NotificationService
}

// NewTrackingStatusHandler creates a new TrackingStatusHandler.
func NewTrackingStatusHandler(svc service.NotificationService) *TrackingStatusHandler {
	return &TrackingStatusHandler{svc: svc}
}

// HandleTrackingStatusUpdated processes a tracking status updated event message.
func (h *TrackingStatusHandler) HandleTrackingStatusUpdated(msg []byte) error {
	var event TrackingStatusUpdatedEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		log.Printf("failed to unmarshal tracking status updated event: %v", err)
		return err
	}

	_, err := h.svc.SendNotification(context.Background(), service.SendNotificationInput{
		UserID:     event.UserID,
		Channel:    domain.ChannelPush,
		TemplateID: event.EventType,
		Variables: map[string]string{
			"tracking_number": event.TrackingNumber,
			"event_type":      event.EventType,
		},
	})
	return err
}
