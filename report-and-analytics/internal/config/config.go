package config

import (
	"fmt"
	"os"
)

// Config holds all runtime configuration for the report-and-analytics service.
type Config struct {
	// HTTP server
	Port string

	// PostgreSQL data warehouse (read replica / OLAP)
	DatabaseURL string
}

// Load reads configuration from environment variables, falling back to
// defaults suitable for local development.
func Load() (*Config, error) {
	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "host=localhost port=5432 user=postgres password=postgres dbname=analytics sslmode=disable"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks that required fields are non-empty.
func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("config: DATABASE_URL must not be empty")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
