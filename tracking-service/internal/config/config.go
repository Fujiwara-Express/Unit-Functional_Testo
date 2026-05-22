package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the tracking-service.
type Config struct {
	HTTPPort       int
	DatabaseDSN    string
	DBMaxOpenConns int
	DBMaxIdleConns int
	DBQueryTimeout time.Duration
}

// LoadConfig reads configuration from environment variables and applies
// defaults for any variables that are absent. It returns an error if
// DATABASE_DSN is empty.
func LoadConfig() (Config, error) {
	cfg := Config{
		HTTPPort:       8080,
		DBMaxOpenConns: 50,
		DBMaxIdleConns: 10,
		DBQueryTimeout: 5 * time.Second,
	}

	if port := os.Getenv("HTTP_PORT"); port != "" {
		v, err := strconv.Atoi(port)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HTTP_PORT %q: %w", port, err)
		}
		cfg.HTTPPort = v
	}

	cfg.DatabaseDSN = os.Getenv("DATABASE_DSN")
	if cfg.DatabaseDSN == "" {
		return Config{}, errors.New("DATABASE_DSN is required but not set")
	}

	if v := os.Getenv("DB_MAX_OPEN_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid DB_MAX_OPEN_CONNS %q: %w", v, err)
		}
		cfg.DBMaxOpenConns = n
	}

	if v := os.Getenv("DB_MAX_IDLE_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid DB_MAX_IDLE_CONNS %q: %w", v, err)
		}
		cfg.DBMaxIdleConns = n
	}

	if v := os.Getenv("DB_QUERY_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid DB_QUERY_TIMEOUT %q: %w", v, err)
		}
		cfg.DBQueryTimeout = d
	}

	return cfg, nil
}
