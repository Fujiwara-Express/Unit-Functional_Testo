package client

import (
	"context"
	"time"
)

// TrackingClient defines the interface for publishing tracking events.
type TrackingClient interface {
	PublishPickedUpEvent(ctx context.Context, pickupID, orderID string, ts time.Time) error
}
