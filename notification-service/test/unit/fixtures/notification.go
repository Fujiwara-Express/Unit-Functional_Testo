package fixtures

import (
	"time"

	"github.com/notification-service/internal/domain"
)

// ValidNotificationLog returns a fully populated NotificationLog with sensible defaults.
func ValidNotificationLog() *domain.NotificationLog {
	return &domain.NotificationLog{
		NotifID:        "notif-123",
		UserID:         "user-456",
		TrackingNumber: "TRK123",
		Channel:        domain.ChannelPush,
		TemplateID:     "tmpl-001",
		Message:        "Your package is on the way.",
		Status:         domain.NotifStatusSent,
		SentAt:         time.Now(),
	}
}

// ValidNotificationTemplate returns a fully populated NotificationTemplate with sensible defaults.
func ValidNotificationTemplate() *domain.NotificationTemplate {
	return &domain.NotificationTemplate{
		TemplateID:   "tmpl-001",
		EventType:    "OUT_FOR_DELIVERY",
		Channel:      domain.ChannelPush,
		Subject:      "Paket Sedang Diantar",
		BodyTemplate: "Paket {{tracking_number}} sedang dalam perjalanan ke Anda.",
	}
}
