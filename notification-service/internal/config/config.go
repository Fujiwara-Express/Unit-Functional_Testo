package config

import (
	"fmt"
	"os"
)

// Config holds all runtime configuration for the notification-service.
type Config struct {
	// HTTP server
	Port string

	// PostgreSQL
	DatabaseURL string

	// Firebase (PUSH notifications)
	FirebaseBaseURL string
	FirebaseAPIKey  string

	// SendGrid (EMAIL notifications)
	SendGridBaseURL string
	SendGridAPIKey  string

	// WhatsApp (WHATSAPP notifications)
	WhatsAppBaseURL string
	WhatsAppAPIKey  string
}

// Load reads configuration from environment variables, falling back to
// defaults suitable for local development.
func Load() (*Config, error) {
	cfg := &Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "host=localhost port=5432 user=postgres password=postgres dbname=notifications sslmode=disable"),
		FirebaseBaseURL: getEnv("FIREBASE_BASE_URL", "https://fcm.googleapis.com"),
		FirebaseAPIKey:  getEnv("FIREBASE_API_KEY", ""),
		SendGridBaseURL: getEnv("SENDGRID_BASE_URL", "https://api.sendgrid.com"),
		SendGridAPIKey:  getEnv("SENDGRID_API_KEY", ""),
		WhatsAppBaseURL: getEnv("WHATSAPP_BASE_URL", "https://graph.facebook.com"),
		WhatsAppAPIKey:  getEnv("WHATSAPP_API_KEY", ""),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks that required fields are non-empty.
func (c *Config) validate() error {
	required := map[string]string{
		"DATABASE_URL":     c.DatabaseURL,
		"FIREBASE_BASE_URL": c.FirebaseBaseURL,
		"SENDGRID_BASE_URL": c.SendGridBaseURL,
		"WHATSAPP_BASE_URL": c.WhatsAppBaseURL,
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
