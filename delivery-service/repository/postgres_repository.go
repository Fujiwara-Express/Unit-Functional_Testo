package repository

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"delivery-service/domain"
	"delivery-service/service"
)

// PostgresRepository implements domain.Delivery_Repository using PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateCourier inserts a new courier record.
func (r *PostgresRepository) CreateCourier(ctx context.Context, c *domain.Courier) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO couriers (courier_id, name, phone, hub_id, vehicle_type, is_available, current_lat, current_lng)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		c.CourierID, c.Name, c.Phone, c.HubID, string(c.VehicleType), c.IsAvailable, c.CurrentLat, c.CurrentLng,
	)
	return err
}

// GetCourierByID retrieves a courier by its primary key.
func (r *PostgresRepository) GetCourierByID(ctx context.Context, courierID string) (*domain.Courier, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT courier_id, name, phone, hub_id, vehicle_type, is_available, current_lat, current_lng
		FROM couriers WHERE courier_id = $1`, courierID)
	return scanCourier(row)
}

// UpdateCourier updates fields on an existing courier record.
func (r *PostgresRepository) UpdateCourier(ctx context.Context, courierID string, update *domain.CourierUpdate) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE couriers
		SET is_available = COALESCE($1, is_available),
		    current_lat  = COALESCE($2, current_lat),
		    current_lng  = COALESCE($3, current_lng)
		WHERE courier_id = $4`,
		update.IsAvailable, update.CurrentLat, update.CurrentLng, courierID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return &service.NotFoundError{ID: courierID}
	}
	return nil
}

// ListCouriers retrieves couriers matching the given filter.
func (r *PostgresRepository) ListCouriers(ctx context.Context, filter *domain.CourierFilter) ([]*domain.Courier, error) {
	query := `SELECT courier_id, name, phone, hub_id, vehicle_type, is_available, current_lat, current_lng FROM couriers WHERE 1=1`
	args := []interface{}{}
	i := 1

	if filter.HubID != "" {
		query += ` AND hub_id = $` + itoa(i)
		args = append(args, filter.HubID)
		i++
	}
	if filter.IsAvailable != nil {
		query += ` AND is_available = $` + itoa(i)
		args = append(args, *filter.IsAvailable)
		i++
	}
	if filter.CityCode != "" {
		query += ` AND city_code = $` + itoa(i)
		args = append(args, filter.CityCode)
		i++
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var couriers []*domain.Courier
	for rows.Next() {
		c := &domain.Courier{}
		var vt string
		if err := rows.Scan(&c.CourierID, &c.Name, &c.Phone, &c.HubID, &vt, &c.IsAvailable, &c.CurrentLat, &c.CurrentLng); err != nil {
			return nil, err
		}
		c.VehicleType = domain.VehicleType(vt)
		couriers = append(couriers, c)
	}
	return couriers, rows.Err()
}

// CreateDeliveryJob inserts a new delivery job record.
func (r *PostgresRepository) CreateDeliveryJob(ctx context.Context, j *domain.DeliveryJob) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO delivery_jobs
			(job_id, tracking_number, courier_id, hub_id, status, attempt_count,
			 proof_photo_url, recipient_name, notes, assigned_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		j.JobID, j.TrackingNumber, j.CourierID, j.HubID, string(j.Status), j.AttemptCount,
		j.ProofPhotoURL, j.RecipientName, j.Notes, j.AssignedAt, j.CompletedAt,
	)
	return err
}

// GetDeliveryJobByID retrieves a delivery job by its primary key.
func (r *PostgresRepository) GetDeliveryJobByID(ctx context.Context, jobID string) (*domain.DeliveryJob, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT job_id, tracking_number, courier_id, hub_id, status, attempt_count,
		       proof_photo_url, recipient_name, notes, assigned_at, completed_at
		FROM delivery_jobs WHERE job_id = $1`, jobID)
	return scanDeliveryJob(row)
}

// GetDeliveryJobByTrackingNumber retrieves a delivery job by tracking number.
func (r *PostgresRepository) GetDeliveryJobByTrackingNumber(ctx context.Context, trackingNumber string) (*domain.DeliveryJob, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT job_id, tracking_number, courier_id, hub_id, status, attempt_count,
		       proof_photo_url, recipient_name, notes, assigned_at, completed_at
		FROM delivery_jobs WHERE tracking_number = $1`, trackingNumber)
	return scanDeliveryJob(row)
}

// UpdateDeliveryJobStatus updates the status and related fields of a delivery job.
func (r *PostgresRepository) UpdateDeliveryJobStatus(ctx context.Context, jobID string, update *domain.JobStatusUpdate) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE delivery_jobs
		SET status         = $1,
		    attempt_count  = $2,
		    proof_photo_url = $3,
		    recipient_name = $4,
		    notes          = $5,
		    completed_at   = $6
		WHERE job_id = $7`,
		string(update.Status), update.AttemptCount, update.ProofPhotoURL,
		update.RecipientName, update.Notes, update.CompletedAt, jobID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return &service.NotFoundError{ID: jobID}
	}
	return nil
}

// GetJobsByCourierID retrieves all delivery jobs assigned to a courier.
func (r *PostgresRepository) GetJobsByCourierID(ctx context.Context, courierID string) ([]*domain.DeliveryJob, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT job_id, tracking_number, courier_id, hub_id, status, attempt_count,
		       proof_photo_url, recipient_name, notes, assigned_at, completed_at
		FROM delivery_jobs WHERE courier_id = $1`, courierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*domain.DeliveryJob
	for rows.Next() {
		j, err := scanDeliveryJobRow(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func scanCourier(row *sql.Row) (*domain.Courier, error) {
	c := &domain.Courier{}
	var vt string
	if err := row.Scan(&c.CourierID, &c.Name, &c.Phone, &c.HubID, &vt, &c.IsAvailable, &c.CurrentLat, &c.CurrentLng); err != nil {
		return nil, err
	}
	c.VehicleType = domain.VehicleType(vt)
	return c, nil
}

func scanDeliveryJob(row *sql.Row) (*domain.DeliveryJob, error) {
	j := &domain.DeliveryJob{}
	var status string
	var assignedAt time.Time
	var completedAt *time.Time
	if err := row.Scan(
		&j.JobID, &j.TrackingNumber, &j.CourierID, &j.HubID, &status, &j.AttemptCount,
		&j.ProofPhotoURL, &j.RecipientName, &j.Notes, &assignedAt, &completedAt,
	); err != nil {
		return nil, err
	}
	j.Status = domain.JobStatus(status)
	j.AssignedAt = assignedAt
	j.CompletedAt = completedAt
	return j, nil
}

func scanDeliveryJobRow(rows *sql.Rows) (*domain.DeliveryJob, error) {
	j := &domain.DeliveryJob{}
	var status string
	var assignedAt time.Time
	var completedAt *time.Time
	if err := rows.Scan(
		&j.JobID, &j.TrackingNumber, &j.CourierID, &j.HubID, &status, &j.AttemptCount,
		&j.ProofPhotoURL, &j.RecipientName, &j.Notes, &assignedAt, &completedAt,
	); err != nil {
		return nil, err
	}
	j.Status = domain.JobStatus(status)
	j.AssignedAt = assignedAt
	j.CompletedAt = completedAt
	return j, nil
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
