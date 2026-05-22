package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pickup-service/internal/domain"
)

// pickupRepository implements PickupRepository using database/sql
type pickupRepository struct {
	db *sql.DB
}

// NewPickupRepository creates a new PickupRepository instance
func NewPickupRepository(db *sql.DB) PickupRepository {
	return &pickupRepository{
		db: db,
	}
}

// CreatePickup inserts a new pickup and returns the generated pickup_id
func (r *pickupRepository) CreatePickup(ctx context.Context, p *domain.Pickup) (string, error) {
	query := `INSERT INTO pickups (pickup_id, order_id, user_id, courier_id, status, pickup_address, pickup_city_code, contact_name, contact_phone, attempt_count, estimated_pickup_time, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING pickup_id`
	row := r.db.QueryRowContext(ctx, query,
		p.PickupID, p.OrderID, p.UserID, p.CourierID, string(p.Status),
		p.PickupAddress, p.PickupCityCode, p.ContactName, p.ContactPhone,
		p.AttemptCount, p.EstimatedPickupTime, p.CreatedAt, p.UpdatedAt,
	)
	var pickupID string
	if err := row.Scan(&pickupID); err != nil {
		return "", fmt.Errorf("CreatePickup: %w", err)
	}
	return pickupID, nil
}

// GetPickupByID retrieves a pickup by its ID
func (r *pickupRepository) GetPickupByID(ctx context.Context, id string) (*domain.Pickup, error) {
	query := `SELECT pickup_id, order_id, user_id, courier_id, status, pickup_address, pickup_city_code, contact_name, contact_phone, attempt_count, estimated_pickup_time, created_at, updated_at FROM pickups WHERE pickup_id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	p := &domain.Pickup{}
	var status string
	err := row.Scan(
		&p.PickupID, &p.OrderID, &p.UserID, &p.CourierID, &status,
		&p.PickupAddress, &p.PickupCityCode, &p.ContactName, &p.ContactPhone,
		&p.AttemptCount, &p.EstimatedPickupTime, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: pickup %s", domain.ErrNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("GetPickupByID: %w", err)
	}
	p.Status = domain.Status(status)
	return p, nil
}

// UpdatePickup updates an existing pickup
func (r *pickupRepository) UpdatePickup(ctx context.Context, p *domain.Pickup) error {
	query := `UPDATE pickups SET order_id=$1, user_id=$2, courier_id=$3, status=$4, pickup_address=$5, pickup_city_code=$6, contact_name=$7, contact_phone=$8, attempt_count=$9, estimated_pickup_time=$10, updated_at=$11 WHERE pickup_id=$12`
	_, err := r.db.ExecContext(ctx, query,
		p.OrderID, p.UserID, p.CourierID, string(p.Status),
		p.PickupAddress, p.PickupCityCode, p.ContactName, p.ContactPhone,
		p.AttemptCount, p.EstimatedPickupTime, p.UpdatedAt, p.PickupID,
	)
	if err != nil {
		return fmt.Errorf("UpdatePickup: %w", err)
	}
	return nil
}

// ListPickups retrieves pickups based on optional filters
func (r *pickupRepository) ListPickups(ctx context.Context, filters ListFilters) ([]*domain.Pickup, error) {
	query := `SELECT pickup_id, order_id, user_id, courier_id, status, pickup_address, pickup_city_code, contact_name, contact_phone, attempt_count, estimated_pickup_time, created_at, updated_at FROM pickups`
	args := []interface{}{}

	if filters.CourierID != "" || filters.Status != "" || filters.Date != "" {
		query += " WHERE"
		first := true
		argIdx := 1
		if filters.CourierID != "" {
			query += fmt.Sprintf(" courier_id = $%d", argIdx)
			args = append(args, filters.CourierID)
			argIdx++
			first = false
		}
		if filters.Status != "" {
			if !first {
				query += " AND"
			}
			query += fmt.Sprintf(" status = $%d", argIdx)
			args = append(args, filters.Status)
			argIdx++
			first = false
		}
		if filters.Date != "" {
			if !first {
				query += " AND"
			}
			query += fmt.Sprintf(" DATE(created_at) = $%d", argIdx)
			args = append(args, filters.Date)
			_ = argIdx
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListPickups: %w", err)
	}
	defer rows.Close()

	var pickups []*domain.Pickup
	for rows.Next() {
		p := &domain.Pickup{}
		var status string
		if err := rows.Scan(
			&p.PickupID, &p.OrderID, &p.UserID, &p.CourierID, &status,
			&p.PickupAddress, &p.PickupCityCode, &p.ContactName, &p.ContactPhone,
			&p.AttemptCount, &p.EstimatedPickupTime, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ListPickups scan: %w", err)
		}
		p.Status = domain.Status(status)
		pickups = append(pickups, p)
	}
	return pickups, rows.Err()
}

// CreatePickupAttempt inserts a new pickup attempt record
func (r *pickupRepository) CreatePickupAttempt(ctx context.Context, a *domain.PickupAttempt) error {
	query := `INSERT INTO pickup_attempts (attempt_id, pickup_id, reason, created_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, a.AttemptID, a.PickupID, a.Reason, a.CreatedAt)
	if err != nil {
		return fmt.Errorf("CreatePickupAttempt: %w", err)
	}
	return nil
}
