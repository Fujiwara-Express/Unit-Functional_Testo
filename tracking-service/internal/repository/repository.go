package repository

import (
	"context"
	"errors"
	"tracking-service/internal/models"
)

var ErrNotFound = errors.New("not found")
var ErrPoolExhausted = errors.New("connection pool exhausted")

type Repository interface {
	// InsertEventAndUpsertSummary persists the event and upserts the summary
	// in a single transaction with a row-level lock on the summary row.
	InsertEventAndUpsertSummary(ctx context.Context, event models.TrackingEvent) error

	// GetEventsByTrackingNumber returns all events for a tracking number,
	// ordered by timestamp ascending. Returns ErrNotFound if none exist.
	GetEventsByTrackingNumber(ctx context.Context, trackingNumber string) ([]models.TrackingEvent, error)

	// GetSummaryByTrackingNumber returns the summary for a tracking number.
	// Returns ErrNotFound if it does not exist.
	GetSummaryByTrackingNumber(ctx context.Context, trackingNumber string) (models.TrackingSummary, error)

	// GetSummariesByTrackingNumbers returns summaries for all provided tracking
	// numbers in a single query. Missing numbers are silently omitted.
	GetSummariesByTrackingNumbers(ctx context.Context, numbers []string) ([]models.TrackingSummary, error)

	// Ping checks database connectivity.
	Ping(ctx context.Context) error
}
