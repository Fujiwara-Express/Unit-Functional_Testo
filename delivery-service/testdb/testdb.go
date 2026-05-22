//go:build functional

package testdb

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"delivery-service/domain"

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
//   - TEST_DB_NAME     (default: delivery_test)
func Setup(t *testing.T) *TestDB {
	t.Helper()

	host := envOrDefault("TEST_DB_HOST", "localhost")
	port := envOrDefault("TEST_DB_PORT", "5432")
	user := envOrDefault("TEST_DB_USER", "postgres")
	password := envOrDefault("TEST_DB_PASSWORD", "")
	dbname := envOrDefault("TEST_DB_NAME", "delivery_test")

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

// Truncate removes all rows from couriers and delivery_jobs tables.
func (t *TestDB) Truncate() error {
	_, err := t.db.Exec(`TRUNCATE TABLE couriers, delivery_jobs RESTART IDENTITY CASCADE`)
	return err
}

// SeedCourier inserts a courier row into the couriers table.
func (t *TestDB) SeedCourier(c *domain.Courier) error {
	_, err := t.db.Exec(`
		INSERT INTO couriers
			(courier_id, name, phone, hub_id, vehicle_type, is_available, current_lat, current_lng)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8)`,
		c.CourierID,
		c.Name,
		c.Phone,
		c.HubID,
		string(c.VehicleType),
		c.IsAvailable,
		c.CurrentLat,
		c.CurrentLng,
	)
	return err
}

// SeedDeliveryJob inserts a delivery_job row into the delivery_jobs table.
func (t *TestDB) SeedDeliveryJob(j *domain.DeliveryJob) error {
	_, err := t.db.Exec(`
		INSERT INTO delivery_jobs
			(job_id, tracking_number, courier_id, hub_id, status, attempt_count,
			 proof_photo_url, recipient_name, notes, assigned_at, completed_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		j.JobID,
		j.TrackingNumber,
		j.CourierID,
		j.HubID,
		string(j.Status),
		j.AttemptCount,
		j.ProofPhotoURL,
		j.RecipientName,
		j.Notes,
		j.AssignedAt,
		j.CompletedAt,
	)
	return err
}

// GetCourierByID retrieves a courier row by courier_id.
func (t *TestDB) GetCourierByID(id string) (*domain.Courier, error) {
	row := t.db.QueryRow(`
		SELECT courier_id, name, phone, hub_id, vehicle_type, is_available, current_lat, current_lng
		FROM couriers
		WHERE courier_id = $1`, id)

	c := &domain.Courier{}
	var vehicleType string

	err := row.Scan(
		&c.CourierID,
		&c.Name,
		&c.Phone,
		&c.HubID,
		&vehicleType,
		&c.IsAvailable,
		&c.CurrentLat,
		&c.CurrentLng,
	)
	if err != nil {
		return nil, err
	}

	c.VehicleType = domain.VehicleType(vehicleType)
	return c, nil
}

// GetDeliveryJobByID retrieves a delivery_job row by job_id.
func (t *TestDB) GetDeliveryJobByID(id string) (*domain.DeliveryJob, error) {
	return t.scanDeliveryJob(t.db.QueryRow(`
		SELECT job_id, tracking_number, courier_id, hub_id, status, attempt_count,
		       proof_photo_url, recipient_name, notes, assigned_at, completed_at
		FROM delivery_jobs
		WHERE job_id = $1`, id))
}

// GetDeliveryJobByTrackingNumber retrieves a delivery_job row by tracking_number.
func (t *TestDB) GetDeliveryJobByTrackingNumber(tn string) (*domain.DeliveryJob, error) {
	return t.scanDeliveryJob(t.db.QueryRow(`
		SELECT job_id, tracking_number, courier_id, hub_id, status, attempt_count,
		       proof_photo_url, recipient_name, notes, assigned_at, completed_at
		FROM delivery_jobs
		WHERE tracking_number = $1`, tn))
}

func (t *TestDB) scanDeliveryJob(row *sql.Row) (*domain.DeliveryJob, error) {
	j := &domain.DeliveryJob{}
	var status string
	var assignedAt time.Time
	var completedAt *time.Time

	err := row.Scan(
		&j.JobID,
		&j.TrackingNumber,
		&j.CourierID,
		&j.HubID,
		&status,
		&j.AttemptCount,
		&j.ProofPhotoURL,
		&j.RecipientName,
		&j.Notes,
		&assignedAt,
		&completedAt,
	)
	if err != nil {
		return nil, err
	}

	j.Status = domain.JobStatus(status)
	j.AssignedAt = assignedAt
	j.CompletedAt = completedAt
	return j, nil
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
