package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/pickup-service/internal/client"
	"github.com/stretchr/testify/assert"
)

// TestTrackingClient_PublishPickedUpEvent_ValidPayload verifies that the Kafka
// tracking client stub accepts a valid payload (pickup_id, order_id, timestamp)
// without returning an error.
func TestTrackingClient_PublishPickedUpEvent_ValidPayload(t *testing.T) {
	tc := client.NewKafkaTrackingClient("tracking-events")

	pickupID := "pickup-123"
	orderID := "order-456"
	ts := time.Now()

	err := tc.PublishPickedUpEvent(context.Background(), pickupID, orderID, ts)
	assert.NoError(t, err)
}
