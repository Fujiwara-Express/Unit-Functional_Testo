package client

import (
	"context"
	"time"
)

type kafkaTrackingClient struct {
	topic string
}

// NewKafkaTrackingClient returns a TrackingClient backed by Kafka.
func NewKafkaTrackingClient(topic string) TrackingClient {
	return &kafkaTrackingClient{topic: topic}
}

// PublishPickedUpEvent is a stub that will publish a picked-up event to Kafka.
func (k *kafkaTrackingClient) PublishPickedUpEvent(_ context.Context, _, _ string, _ time.Time) error {
	return nil
}
