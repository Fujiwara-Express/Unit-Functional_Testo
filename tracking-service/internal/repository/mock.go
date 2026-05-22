package repository

import (
	"context"
	"sort"
	"sync"
	"tracking-service/internal/models"
)

// MockRepository is an in-memory implementation of Repository for use in unit
// and property-based tests. It requires no database connection.
type MockRepository struct {
	mu        sync.RWMutex
	events    map[string][]models.TrackingEvent
	summaries map[string]models.TrackingSummary

	// InjectError, when non-nil, is returned by all write operations so that
	// error-path tests can simulate repository failures without a real database.
	InjectError error
}

// NewMockRepository returns an initialised *MockRepository ready for use.
func NewMockRepository() *MockRepository {
	return &MockRepository{
		events:    make(map[string][]models.TrackingEvent),
		summaries: make(map[string]models.TrackingSummary),
	}
}

// InsertEventAndUpsertSummary appends the event to the in-memory event log and
// upserts the summary so that it always reflects the event with the latest
// Timestamp.
func (m *MockRepository) InsertEventAndUpsertSummary(_ context.Context, event models.TrackingEvent) error {
	if m.InjectError != nil {
		return m.InjectError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Append the event.
	m.events[event.TrackingNumber] = append(m.events[event.TrackingNumber], event)

	// Upsert the summary: update only when no summary exists yet or when this
	// event is strictly newer than the current summary.
	existing, ok := m.summaries[event.TrackingNumber]
	if !ok || event.Timestamp.After(existing.UpdatedAt) {
		m.summaries[event.TrackingNumber] = models.TrackingSummary{
			TrackingNumber: event.TrackingNumber,
			CurrentStatus:  event.Status,
			LastLocation:   event.Location,
			UpdatedAt:      event.Timestamp,
			// EstimatedDelivery is not set by events; callers may set it directly
			// on the summary map if needed for tests.
		}
	}

	return nil
}

// GetEventsByTrackingNumber returns all events for the given tracking number
// sorted by Timestamp ascending. Returns ErrNotFound if no events exist.
func (m *MockRepository) GetEventsByTrackingNumber(_ context.Context, trackingNumber string) ([]models.TrackingEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	evts, ok := m.events[trackingNumber]
	if !ok || len(evts) == 0 {
		return nil, ErrNotFound
	}

	// Return a sorted copy so callers cannot mutate internal state.
	result := make([]models.TrackingEvent, len(evts))
	copy(result, evts)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result, nil
}

// GetSummaryByTrackingNumber returns the summary for the given tracking number.
// Returns ErrNotFound if no summary exists.
func (m *MockRepository) GetSummaryByTrackingNumber(_ context.Context, trackingNumber string) (models.TrackingSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	summary, ok := m.summaries[trackingNumber]
	if !ok {
		return models.TrackingSummary{}, ErrNotFound
	}

	return summary, nil
}

// GetSummariesByTrackingNumbers returns summaries for all provided tracking
// numbers that exist. Missing numbers are silently omitted.
func (m *MockRepository) GetSummariesByTrackingNumbers(_ context.Context, numbers []string) ([]models.TrackingSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]models.TrackingSummary, 0, len(numbers))
	for _, n := range numbers {
		if s, ok := m.summaries[n]; ok {
			result = append(result, s)
		}
	}

	return result, nil
}

// Ping always returns nil (or InjectError when set) to satisfy the Repository
// interface without requiring a real database connection.
func (m *MockRepository) Ping(_ context.Context) error {
	return m.InjectError
}

// SetSummaryForTest directly sets a TrackingSummary in the mock's internal map.
// This is intended for use in tests that need to set up specific summary state
// (e.g., to test estimated_delivery serialization) without going through the
// normal event-insertion path.
func (m *MockRepository) SetSummaryForTest(s models.TrackingSummary) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.summaries[s.TrackingNumber] = s
}
