package config

import (
	"fmt"
	"os"
)

// Config holds all runtime configuration for the pickup-service.
type Config struct {
	// HTTP server
	Port string

	// PostgreSQL
	DatabaseURL string

	// Downstream services
	DeliveryServiceURL    string
	NotificationServiceURL string

	// Kafka
	TrackingKafkaTopic string
}

// Load reads configuration from environment variables, falling back to
// defaults suitable for local development.
func Load() (*Config, error) {
	cfg := &Config{
		Port:                   getEnv("PORT", "8080"),
		DatabaseURL:            getEnv("DATABASE_URL", "host=localhost port=5432 user=postgres password=postgres dbname=pickup sslmode=disable"),
		DeliveryServiceURL:     getEnv("DELIVERY_SERVICE_URL", "http://localhost:8081"),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8082"),
		TrackingKafkaTopic:     getEnv("TRACKING_KAFKA_TOPIC", "tracking.events"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks that required fields are non-empty.
func (c *Config) validate() error {
	required := map[string]string{
		"DATABASE_URL":            c.DatabaseURL,
		"DELIVERY_SERVICE_URL":    c.DeliveryServiceURL,
		"NOTIFICATION_SERVICE_URL": c.NotificationServiceURL,
		"TRACKING_KAFKA_TOPIC":    c.TrackingKafkaTopic,
	}
	for name, val := range required {
		if val == "" {
			return fmt.Errorf("config: %s must not be empty", name)
		}
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
