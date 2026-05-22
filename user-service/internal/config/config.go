package config

import "os"

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	Port      string
	DBDSN     string
	JWTSecret string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Port:      getEnv("PORT", "8080"),
		DBDSN:     getEnv("DB_DSN", "postgres://postgres:postgres@localhost:5432/userservice?sslmode=disable"),
		JWTSecret: getEnv("JWT_SECRET", "change-me-in-production"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
