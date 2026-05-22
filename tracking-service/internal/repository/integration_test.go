package repository

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	if os.Getenv("INTEGRATION") != "true" {
		// Skip all integration tests when INTEGRATION env var is not set
		os.Exit(0)
	}

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = "postgres://tracking:tracking_secret@localhost:5432/tracking_test"
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		panic("failed to open test database: " + err.Error())
	}

	// Apply migrations
	migrationSQL, err := os.ReadFile("../../migrations/001_initial_schema.sql")
	if err != nil {
		panic("failed to read migration file: " + err.Error())
	}

	if _, err := db.Exec(string(migrationSQL)); err != nil {
		// Ignore errors if tables already exist
		_ = err
	}

	testDB = db

	code := m.Run()

	// Teardown: drop tables
	db.Exec("DROP TABLE IF EXISTS tracking_events CASCADE")
	db.Exec("DROP TABLE IF EXISTS tracking_summary CASCADE")
	db.Close()

	os.Exit(code)
}
