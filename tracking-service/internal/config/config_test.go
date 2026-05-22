package config

import (
	"testing"
	"time"
)



// TestLoadConfig_Defaults verifies that defaults are applied when no env vars
// are set (except the required DATABASE_DSN).
func TestLoadConfig_Defaults(t *testing.T) {
	t.Setenv("HTTP_PORT", "")
	t.Setenv("DATABASE_DSN", "postgres://user:pass@localhost/db")
	t.Setenv("DB_MAX_OPEN_CONNS", "")
	t.Setenv("DB_MAX_IDLE_CONNS", "")
	t.Setenv("DB_QUERY_TIMEOUT", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.HTTPPort != 8080 {
		t.Errorf("HTTPPort: got %d, want 8080", cfg.HTTPPort)
	}
	if cfg.DBMaxOpenConns != 50 {
		t.Errorf("DBMaxOpenConns: got %d, want 50", cfg.DBMaxOpenConns)
	}
	if cfg.DBMaxIdleConns != 10 {
		t.Errorf("DBMaxIdleConns: got %d, want 10", cfg.DBMaxIdleConns)
	}
	if cfg.DBQueryTimeout != 5*time.Second {
		t.Errorf("DBQueryTimeout: got %v, want 5s", cfg.DBQueryTimeout)
	}
	if cfg.DatabaseDSN != "postgres://user:pass@localhost/db" {
		t.Errorf("DatabaseDSN: got %q, want postgres://user:pass@localhost/db", cfg.DatabaseDSN)
	}
}

// TestLoadConfig_MissingDSN verifies that an error is returned when
// DATABASE_DSN is not set.
func TestLoadConfig_MissingDSN(t *testing.T) {
	t.Setenv("DATABASE_DSN", "")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when DATABASE_DSN is empty, got nil")
	}
}

// TestLoadConfig_CustomValues verifies that all environment variables are read
// and override the defaults correctly.
func TestLoadConfig_CustomValues(t *testing.T) {
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("DATABASE_DSN", "postgres://custom:secret@db.example.com/tracking")
	t.Setenv("DB_MAX_OPEN_CONNS", "100")
	t.Setenv("DB_MAX_IDLE_CONNS", "20")
	t.Setenv("DB_QUERY_TIMEOUT", "10s")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.HTTPPort != 9090 {
		t.Errorf("HTTPPort: got %d, want 9090", cfg.HTTPPort)
	}
	if cfg.DatabaseDSN != "postgres://custom:secret@db.example.com/tracking" {
		t.Errorf("DatabaseDSN: got %q", cfg.DatabaseDSN)
	}
	if cfg.DBMaxOpenConns != 100 {
		t.Errorf("DBMaxOpenConns: got %d, want 100", cfg.DBMaxOpenConns)
	}
	if cfg.DBMaxIdleConns != 20 {
		t.Errorf("DBMaxIdleConns: got %d, want 20", cfg.DBMaxIdleConns)
	}
	if cfg.DBQueryTimeout != 10*time.Second {
		t.Errorf("DBQueryTimeout: got %v, want 10s", cfg.DBQueryTimeout)
	}
}

// TestLoadConfig_InvalidHTTPPort verifies that a non-integer HTTP_PORT returns
// an error.
func TestLoadConfig_InvalidHTTPPort(t *testing.T) {
	t.Setenv("HTTP_PORT", "not-a-number")
	t.Setenv("DATABASE_DSN", "postgres://user:pass@localhost/db")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for invalid HTTP_PORT, got nil")
	}
}

// TestLoadConfig_InvalidDBQueryTimeout verifies that a malformed
// DB_QUERY_TIMEOUT returns an error.
func TestLoadConfig_InvalidDBQueryTimeout(t *testing.T) {
	t.Setenv("DATABASE_DSN", "postgres://user:pass@localhost/db")
	t.Setenv("DB_QUERY_TIMEOUT", "not-a-duration")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for invalid DB_QUERY_TIMEOUT, got nil")
	}
}
