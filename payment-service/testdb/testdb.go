//go:build functional

package testdb

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"payment-service/domain"

	_ "github.com/lib/pq"
)

// TestDB wraps a *sql.DB with helper methods for functional tests.
type TestDB struct {
	db *sql.DB
}

// Setup opens a DB connection using environment variables and registers cleanup.
// Environment variables:
//   - TEST_DB_HOST     (default: localhost)
//   - TEST_DB_PORT     (default: 5432)
//   - TEST_DB_USER     (default: postgres)
//   - TEST_DB_PASSWORD (default: "")
//   - TEST_DB_NAME     (default: payment_test)
func Setup(t *testing.T) *TestDB {
	t.Helper()

	host := envOrDefault("TEST_DB_HOST", "localhost")
	port := envOrDefault("TEST_DB_PORT", "5432")
	user := envOrDefault("TEST_DB_USER", "postgres")
	password := envOrDefault("TEST_DB_PASSWORD", "")
	dbname := envOrDefault("TEST_DB_NAME", "payment_test")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("testdb: failed to open DB connection: %v", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("testdb: failed to ping DB: %v", err)
	}

	tdb := &TestDB{db: db}

	t.Cleanup(func() {
		db.Close()
	})

	return tdb
}

// DB returns the underlying *sql.DB connection.
func (t *TestDB) DB() *sql.DB {
	return t.db
}

// Truncate removes all rows from payments and cod_collections tables.
func (t *TestDB) Truncate() error {
	_, err := t.db.Exec(`TRUNCATE TABLE payments, cod_collections RESTART IDENTITY CASCADE`)
	return err
}

// SeedPayment inserts a payment row into the payments table.
func (t *TestDB) SeedPayment(payment *domain.Payment) error {
	_, err := t.db.Exec(`
		INSERT INTO payments
			(payment_id, order_id, user_id, amount, method, status, external_ref, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		payment.PaymentID,
		payment.OrderID,
		payment.UserID,
		payment.Amount,
		string(payment.Method),
		string(payment.Status),
		payment.ExternalRef,
		payment.CreatedAt,
		payment.UpdatedAt,
	)
	return err
}

// SeedCodCollection inserts a cod_collection row into the cod_collections table.
func (t *TestDB) SeedCodCollection(col *domain.CodCollection) error {
	_, err := t.db.Exec(`
		INSERT INTO cod_collections
			(collection_id, order_id, courier_id, amount_collected, collected_at, remittance_status)
		VALUES
			($1, $2, $3, $4, $5, $6)`,
		col.CollectionID,
		col.OrderID,
		col.CourierID,
		col.AmountCollected,
		col.CollectedAt,
		string(col.RemittanceStatus),
	)
	return err
}

// GetPaymentByOrderID retrieves a payment row by order_id.
func (t *TestDB) GetPaymentByOrderID(orderID string) (*domain.Payment, error) {
	row := t.db.QueryRow(`
		SELECT payment_id, order_id, user_id, amount, method, status, external_ref, created_at, updated_at
		FROM payments
		WHERE order_id = $1`, orderID)

	p := &domain.Payment{}
	var method, status string
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&p.PaymentID,
		&p.OrderID,
		&p.UserID,
		&p.Amount,
		&method,
		&status,
		&p.ExternalRef,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	p.Method = domain.PaymentMethod(method)
	p.Status = domain.PaymentStatus(status)
	p.CreatedAt = createdAt
	p.UpdatedAt = updatedAt

	return p, nil
}

// GetCodCollectionByOrderID retrieves a cod_collection row by order_id.
func (t *TestDB) GetCodCollectionByOrderID(orderID string) (*domain.CodCollection, error) {
	row := t.db.QueryRow(`
		SELECT collection_id, order_id, courier_id, amount_collected, collected_at, remittance_status
		FROM cod_collections
		WHERE order_id = $1`, orderID)

	c := &domain.CodCollection{}
	var remittanceStatus string
	var collectedAt time.Time

	err := row.Scan(
		&c.CollectionID,
		&c.OrderID,
		&c.CourierID,
		&c.AmountCollected,
		&collectedAt,
		&remittanceStatus,
	)
	if err != nil {
		return nil, err
	}

	c.CollectedAt = collectedAt
	c.RemittanceStatus = domain.RemittanceStatus(remittanceStatus)

	return c, nil
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
