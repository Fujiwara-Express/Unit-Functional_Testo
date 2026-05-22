package repository

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"payment-service/domain"

	"github.com/segmentio/kafka-go"
)

// KafkaPublisher implements domain.Kafka_Publisher using segmentio/kafka-go.
type KafkaPublisher struct {
	writer *kafka.Writer
}

// NewKafkaPublisher creates a new KafkaPublisher with optional TLS support.
func NewKafkaPublisher(brokers string, tlsConfig *tls.Config) *KafkaPublisher {
	brokerList := strings.Split(brokers, ",")

	transport := &kafka.Transport{
		TLS:         tlsConfig,
		DialTimeout: 10 * time.Second,
	}

	return &KafkaPublisher{
		writer: &kafka.Writer{
			Addr:      kafka.TCP(brokerList...),
			Balancer:  &kafka.LeastBytes{},
			Transport: transport,
			Logger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
				log.Printf("[KAFKA] "+msg, args...)
			}),
		},
	}
}

// Publish sends the payment event to the specified Kafka topic.
func (p *KafkaPublisher) Publish(ctx context.Context, topic string, event *domain.PaymentEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(event.OrderID),
		Value: payload,
	})

	if err != nil {
		log.Printf("[KAFKA-ERROR] Failed to publish message to %s: %v\n", topic, err)
		return err
	}

	log.Printf("[KAFKA] Successfully published event %s to topic %s\n", event.EventType, topic)
	return nil
}

// Close closes the underlying Kafka writer.
func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}

// LoadTLSConfig helper to create a tls.Config from Aiven certificate files.
func LoadTLSConfig(caPath, certPath, keyPath string) (*tls.Config, error) {
	if caPath == "" || certPath == "" || keyPath == "" {
		return nil, nil // Fallback to plaintext
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load client cert/key: %w", err)
	}

	caCert, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert: %w", err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		// For some cloud providers, you might need to set InsecureSkipVerify: true 
		// if the broker's hostname doesn't match the cert, but it's safer to keep it false.
		MinVersion: tls.VersionTLS12,
	}, nil
}
