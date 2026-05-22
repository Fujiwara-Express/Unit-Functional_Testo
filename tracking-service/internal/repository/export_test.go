package repository

import "database/sql"

// GetTestDB returns the *sql.DB set up by TestMain in integration_test.go.
// This function is compiled only during tests and is used by the external
// test package (package repository_test) in postgres_integration_test.go
// to access the shared test database without creating an import cycle.
func GetTestDB() *sql.DB {
	return testDB
}
