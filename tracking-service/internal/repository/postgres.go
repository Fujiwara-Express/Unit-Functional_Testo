package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"tracking-service/internal/models"
)

// PostgresRepository implements Repository against a PostgreSQL database using
// the standard database/sql interface backed by the pgx driver.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository returns a *PostgresRepository that uses the provided
// *sql.DB for all database operations.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// InsertEventAndUpsertSummary persists the event and upserts the summary in a
// single transaction. It acquires a row-level lock on the tracking_summary row
// before writing to prevent concurrent summary corruption.
//
// If the context deadline is exceeded, ErrPoolExhausted is returned so that the
// handler layer can map it to HTTP 503.
func (r *PostgresRepository) InsertEventAndUpsertSummary(ctx context.Context, event models.TrackingEvent) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("begin transaction: %w", ErrPoolExhausted)
		}
		return fmt.Errorf("begin transaction: %w", err)
	}
	// Ensure the transaction is rolled back on any error path.
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Step 1: Acquire a row-level lock on the tracking_summary row (if it
	// exists). This prevents concurrent transactions from producing a stale
	// summary. ErrNoRows is expected when the row does not yet exist.
	var locked string
	err = tx.QueryRowContext(ctx,
		`SELECT tracking_number FROM tracking_summary WHERE tracking_number = $1 FOR UPDATE`,
		event.TrackingNumber,
	).Scan(&locked)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("lock tracking_summary row: %w", ErrPoolExhausted)
		}
		return fmt.Errorf("lock tracking_summary row: %w", err)
	}
	// Reset err so the deferred rollback does not fire on ErrNoRows.
	err = nil

	// Step 2: Insert the event into the append-only event log.
	_, err = tx.ExecContext(ctx,
		`INSERT INTO tracking_events
			(event_id, tracking_number, status, location, hub_id, notes, created_by_service, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		event.EventID,
		event.TrackingNumber,
		string(event.Status),
		event.Location,
		event.HubID,
		event.Notes,
		event.CreatedByService,
		event.Timestamp,
	)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("insert tracking_event: %w", ErrPoolExhausted)
		}
		return fmt.Errorf("insert tracking_event: %w", err)
	}

	// Step 3: Upsert the summary, updating only when the incoming event is
	// strictly newer than the stored one (prevents out-of-order updates from
	// overwriting a more recent status).
	_, err = tx.ExecContext(ctx,
		`INSERT INTO tracking_summary
			(tracking_number, current_status, last_location, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (tracking_number) DO UPDATE
			SET current_status = EXCLUDED.current_status,
			    last_location   = EXCLUDED.last_location,
			    updated_at      = EXCLUDED.updated_at
			WHERE tracking_summary.updated_at < EXCLUDED.updated_at`,
		event.TrackingNumber,
		string(event.Status),
		event.Location,
		event.Timestamp,
	)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("upsert tracking_summary: %w", ErrPoolExhausted)
		}
		return fmt.Errorf("upsert tracking_summary: %w", err)
	}

	// Step 4: Commit the transaction.
	if err = tx.Commit(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("commit transaction: %w", ErrPoolExhausted)
		}
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetEventsByTrackingNumber returns all events for the given tracking number
// ordered by timestamp ascending. Returns ErrNotFound if no events exist.
func (r *PostgresRepository) GetEventsByTrackingNumber(ctx context.Context, trackingNumber string) ([]models.TrackingEvent, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT event_id, tracking_number, status, location, hub_id, notes, created_by_service, timestamp
		FROM tracking_events
		WHERE tracking_number = $1
		ORDER BY timestamp ASC`,
		trackingNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("query tracking_events: %w", err)
	}
	defer rows.Close()

	var events []models.TrackingEvent
	for rows.Next() {
		var e models.TrackingEvent
		var status string
		if err := rows.Scan(
			&e.EventID,
			&e.TrackingNumber,
			&status,
			&e.Location,
			&e.HubID,
			&e.Notes,
			&e.CreatedByService,
			&e.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("scan tracking_event row: %w", err)
		}
		e.Status = models.Status(status)
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tracking_event rows: %w", err)
	}

	if len(events) == 0 {
		return nil, ErrNotFound
	}

	return events, nil
}

// GetSummaryByTrackingNumber returns the summary for the given tracking number.
// Returns ErrNotFound if no summary row exists.
func (r *PostgresRepository) GetSummaryByTrackingNumber(ctx context.Context, trackingNumber string) (models.TrackingSummary, error) {
	var s models.TrackingSummary
	var status string

	err := r.db.QueryRowContext(ctx,
		`SELECT tracking_number, current_status, last_location, estimated_delivery, updated_at
		FROM tracking_summary
		WHERE tracking_number = $1`,
		trackingNumber,
	).Scan(
		&s.TrackingNumber,
		&status,
		&s.LastLocation,
		&s.EstimatedDelivery,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.TrackingSummary{}, ErrNotFound
		}
		return models.TrackingSummary{}, fmt.Errorf("query tracking_summary: %w", err)
	}

	s.CurrentStatus = models.Status(status)
	return s, nil
}

// GetSummariesByTrackingNumbers returns summaries for all provided tracking
// numbers in a single query using PostgreSQL's ANY($1) array operator.
// Numbers that do not exist in the database are silently omitted.
func (r *PostgresRepository) GetSummariesByTrackingNumbers(ctx context.Context, numbers []string) ([]models.TrackingSummary, error) {
	// Build a pgx TextArray so the driver can bind []string as a PostgreSQL
	// text[] parameter accepted by ANY($1).
	arr := pgtype.FlatArray[string](numbers)

	rows, err := r.db.QueryContext(ctx,
		`SELECT tracking_number, current_status, last_location, estimated_delivery, updated_at
		FROM tracking_summary
		WHERE tracking_number = ANY($1)`,
		arr,
	)
	if err != nil {
		return nil, fmt.Errorf("query tracking_summary bulk: %w", err)
	}
	defer rows.Close()

	var summaries []models.TrackingSummary
	for rows.Next() {
		var s models.TrackingSummary
		var status string
		if err := rows.Scan(
			&s.TrackingNumber,
			&status,
			&s.LastLocation,
			&s.EstimatedDelivery,
			&s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan tracking_summary row: %w", err)
		}
		s.CurrentStatus = models.Status(status)
		summaries = append(summaries, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tracking_summary rows: %w", err)
	}

	return summaries, nil
}

// Ping checks database connectivity by delegating to db.PingContext.
func (r *PostgresRepository) Ping(ctx context.Context) error {
	if err := r.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}
