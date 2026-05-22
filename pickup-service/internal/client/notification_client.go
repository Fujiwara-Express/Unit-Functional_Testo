package client

import "context"

// NotificationClient defines the interface for sending notifications.
type NotificationClient interface {
	NotifyCourierEnRoute(ctx context.Context, contactName, contactPhone, courierID string) error
}
