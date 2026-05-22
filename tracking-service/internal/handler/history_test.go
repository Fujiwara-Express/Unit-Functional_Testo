package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"pgregory.net/rapid"

	"tracking-service/internal/middleware"
	"tracking-service/internal/models"
	"tracking-service/internal/repository"
)

// newHistoryTestServer creates an httptest.Server with a chi router that serves
// GET /tracking/{tracking_number} via HistoryHandler.
func newHistoryTestServer(repo *repository.MockRepository) *httptest.Server {
	r := chi.NewRouter()
	r.Use(middleware.CorrelationIDMiddleware)
	historyHandler := NewHistoryHandler(repo, 5*time.Second)
	r.Get("/tracking/{tracking_number}", historyHandler.ServeHTTP)
	return httptest.NewServer(r)
}

// newCombinedTestServer creates an httptest.Server with a chi router that serves
// both POST /tracking/events and GET /tracking/{tracking_number}.
func newCombinedTestServer(repo *repository.MockRepository) *httptest.Server {
	r := chi.NewRouter()
	r.Use(middleware.CorrelationIDMiddleware)
	r.Post("/tracking/events", NewEventHandler(repo, 5*time.Second).ServeHTTP)
	r.Get("/tracking/{tracking_number}", NewHistoryHandler(repo, 5*time.Second).ServeHTTP)
	return httptest.NewServer(r)
}

// postEvent posts a single CreateEventRequest to the given server URL and
// returns the HTTP response. The caller is responsible for closing the body.
func postEvent(serverURL string, req models.CreateEventRequest) (*http.Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	return http.Post(serverURL+"/tracking/events", "application/json", bytes.NewReader(body))
}

// getHistory performs GET /tracking/{trackingNumber} and returns the decoded
// TrackingHistoryResponse along with the raw HTTP response.
func getHistory(serverURL, trackingNumber string) (*http.Response, *models.TrackingHistoryResponse, error) {
	resp, err := http.Get(serverURL + "/tracking/" + trackingNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("GET history: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return resp, nil, nil
	}
	var histResp models.TrackingHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&histResp); err != nil {
		resp.Body.Close()
		return resp, nil, fmt.Errorf("decode history response: %w", err)
	}
	return resp, &histResp, nil
}

// Feature: tracking-service, Property 5: Summary projection reflects the most recent event
// Validates: Requirements 1.6, 2.1, 2.2, 2.5, 5.2
func TestProperty5_SummaryProjectionReflectsMostRecentEvent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := repository.NewMockRepository()
		server := newCombinedTestServer(repo)
		defer server.Close()

		trackingNumber := drawValidTrackingNumber(t)

		// Generate between 1 and 5 events for the same tracking number.
		eventCount := rapid.IntRange(1, 5).Draw(t, "event_count")

		type eventSpec struct {
			status    string
			timestamp time.Time
		}
		specs := make([]eventSpec, eventCount)

		// Draw distinct timestamps to avoid ambiguity in "most recent".
		usedOffsets := make(map[int]bool)
		baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < eventCount; i++ {
			var offsetHours int
			for {
				offsetHours = rapid.IntRange(0, 1000).Draw(t, fmt.Sprintf("offset_hours_%d", i))
				if !usedOffsets[offsetHours] {
					break
				}
			}
			usedOffsets[offsetHours] = true
			specs[i] = eventSpec{
				status:    rapid.SampledFrom(validStatusValues).Draw(t, fmt.Sprintf("status_%d", i)),
				timestamp: baseTime.Add(time.Duration(offsetHours) * time.Hour),
			}
		}

		// POST all events.
		for i, spec := range specs {
			req := models.CreateEventRequest{
				TrackingNumber: trackingNumber,
				Status:         spec.status,
				Timestamp:      spec.timestamp.Format(time.RFC3339),
			}
			resp, err := postEvent(server.URL, req)
			if err != nil {
				t.Fatalf("postEvent[%d] failed: %v", i, err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("postEvent[%d]: expected HTTP 201, got %d", i, resp.StatusCode)
			}
		}

		// GET history.
		resp, histResp, err := getHistory(server.URL, trackingNumber)
		if err != nil {
			t.Fatalf("getHistory failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
		}
		if histResp == nil {
			t.Fatal("history response is nil")
		}

		// Assert history length equals event count.
		if len(histResp.History) != eventCount {
			t.Fatalf("expected %d history entries, got %d", eventCount, len(histResp.History))
		}

		// Assert history is ordered ascending by timestamp.
		for i := 1; i < len(histResp.History); i++ {
			if histResp.History[i].Timestamp.Before(histResp.History[i-1].Timestamp) {
				t.Fatalf("history not sorted ascending: entry[%d].timestamp %v is before entry[%d].timestamp %v",
					i, histResp.History[i].Timestamp, i-1, histResp.History[i-1].Timestamp)
			}
		}

		// Find the event with the latest timestamp.
		latestSpec := specs[0]
		for _, spec := range specs[1:] {
			if spec.timestamp.After(latestSpec.timestamp) {
				latestSpec = spec
			}
		}

		// Assert current_status equals the status of the latest event.
		if string(histResp.CurrentStatus) != latestSpec.status {
			t.Fatalf("current_status mismatch: want %q (latest event), got %q",
				latestSpec.status, histResp.CurrentStatus)
		}

		// Assert history entries are sorted ascending (verify timestamps match posted events).
		sortedSpecs := make([]eventSpec, len(specs))
		copy(sortedSpecs, specs)
		sort.Slice(sortedSpecs, func(i, j int) bool {
			return sortedSpecs[i].timestamp.Before(sortedSpecs[j].timestamp)
		})
		for i, entry := range histResp.History {
			if !entry.Timestamp.Equal(sortedSpecs[i].timestamp) {
				t.Fatalf("history[%d].timestamp mismatch: want %v, got %v",
					i, sortedSpecs[i].timestamp, entry.Timestamp)
			}
			if string(entry.Status) != sortedSpecs[i].status {
				t.Fatalf("history[%d].status mismatch: want %q, got %q",
					i, sortedSpecs[i].status, entry.Status)
			}
		}
	})
}

// Feature: tracking-service, Property 6: Non-existent tracking number returns HTTP 404
// Validates: Requirements 2.3
func TestProperty6_NonExistentTrackingNumberReturns404(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := repository.NewMockRepository()
		server := newHistoryTestServer(repo)
		defer server.Close()

		// Generate a tracking number that was never inserted.
		trackingNumber := drawValidTrackingNumber(t)

		resp, err := http.Get(server.URL + "/tracking/" + trackingNumber)
		if err != nil {
			t.Fatalf("GET request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected HTTP 404 for non-existent tracking number %q, got %d",
				trackingNumber, resp.StatusCode)
		}

		// Decode and verify the error response has a descriptive message.
		var errResp models.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}
		if errResp.Error == "" {
			t.Fatal("error response has empty error message")
		}
	})
}

// Feature: tracking-service, Property 7: estimated_delivery serialization
// Validates: Requirements 2.6
func TestProperty7_EstimatedDeliverySerialization(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := repository.NewMockRepository()
		server := newHistoryTestServer(repo)
		defer server.Close()

		trackingNumber := drawValidTrackingNumber(t)
		hasEstimatedDelivery := rapid.Bool().Draw(t, "has_estimated_delivery")

		// Build a summary directly in the mock.
		summary := models.TrackingSummary{
			TrackingNumber: trackingNumber,
			CurrentStatus:  models.Status(rapid.SampledFrom(validStatusValues).Draw(t, "status")),
			UpdatedAt:      time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
		}

		// Also insert a matching event so GetEventsByTrackingNumber doesn't return ErrNotFound.
		event := models.TrackingEvent{
			EventID:          "test-event-id",
			TrackingNumber:   trackingNumber,
			Status:           summary.CurrentStatus,
			CreatedByService: "tracking-service",
			Timestamp:        summary.UpdatedAt,
		}
		if err := repo.InsertEventAndUpsertSummary(t.Context(), event); err != nil {
			t.Fatalf("InsertEventAndUpsertSummary failed: %v", err)
		}

		if hasEstimatedDelivery {
			// Generate a future delivery time.
			year := rapid.IntRange(2025, 2030).Draw(t, "delivery_year")
			month := rapid.IntRange(1, 12).Draw(t, "delivery_month")
			day := rapid.IntRange(1, 28).Draw(t, "delivery_day")
			deliveryTime := time.Date(year, time.Month(month), day, 12, 0, 0, 0, time.UTC)
			summary.EstimatedDelivery = &deliveryTime
		}

		// Set the summary directly in the mock (overrides the one set by InsertEventAndUpsertSummary).
		repo.SetSummaryForTest(summary)

		// GET history.
		resp, err := http.Get(server.URL + "/tracking/" + trackingNumber)
		if err != nil {
			t.Fatalf("GET request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
		}

		// Decode the raw JSON to inspect estimated_delivery as a raw value.
		var rawResp map[string]json.RawMessage
		if err := json.NewDecoder(resp.Body).Decode(&rawResp); err != nil {
			t.Fatalf("failed to decode response as raw JSON: %v", err)
		}

		rawED, ok := rawResp["estimated_delivery"]
		if !ok {
			t.Fatal("response missing 'estimated_delivery' field")
		}

		if hasEstimatedDelivery {
			// Should be a non-null RFC 3339 string.
			if string(rawED) == "null" {
				t.Fatal("expected non-null estimated_delivery, got null")
			}
			// Unmarshal as a string and parse as RFC 3339.
			var edStr string
			if err := json.Unmarshal(rawED, &edStr); err != nil {
				t.Fatalf("estimated_delivery is not a JSON string: %v (raw: %s)", err, rawED)
			}
			if _, err := time.Parse(time.RFC3339, edStr); err != nil {
				t.Fatalf("estimated_delivery %q is not a valid RFC 3339 string: %v", edStr, err)
			}
		} else {
			// Should be null.
			if string(rawED) != "null" {
				t.Fatalf("expected null estimated_delivery, got %s", rawED)
			}
		}
	})
}

// Feature: tracking-service, Property 12: Timestamp fields in responses are valid RFC 3339 strings
// Validates: Requirements 6.3
func TestProperty12_TimestampFieldsAreRFC3339(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := repository.NewMockRepository()
		server := newCombinedTestServer(repo)
		defer server.Close()

		trackingNumber := drawValidTrackingNumber(t)
		eventCount := rapid.IntRange(1, 5).Draw(t, "event_count")

		// POST events with distinct timestamps.
		usedOffsets := make(map[int]bool)
		baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < eventCount; i++ {
			var offsetHours int
			for {
				offsetHours = rapid.IntRange(0, 1000).Draw(t, fmt.Sprintf("ts_offset_%d", i))
				if !usedOffsets[offsetHours] {
					break
				}
			}
			usedOffsets[offsetHours] = true

			req := models.CreateEventRequest{
				TrackingNumber: trackingNumber,
				Status:         rapid.SampledFrom(validStatusValues).Draw(t, fmt.Sprintf("ts_status_%d", i)),
				Timestamp:      baseTime.Add(time.Duration(offsetHours) * time.Hour).Format(time.RFC3339),
			}
			resp, err := postEvent(server.URL, req)
			if err != nil {
				t.Fatalf("postEvent[%d] failed: %v", i, err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("postEvent[%d]: expected HTTP 201, got %d", i, resp.StatusCode)
			}
		}

		// GET history.
		resp, err := http.Get(server.URL + "/tracking/" + trackingNumber)
		if err != nil {
			t.Fatalf("GET request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
		}

		// Decode the response.
		var histResp models.TrackingHistoryResponse
		if err := json.NewDecoder(resp.Body).Decode(&histResp); err != nil {
			t.Fatalf("failed to decode history response: %v", err)
		}

		// Assert each history entry's timestamp is a valid RFC 3339 string by
		// re-encoding and parsing it.
		for i, entry := range histResp.History {
			// time.Time is marshalled to RFC 3339 by encoding/json; verify by
			// round-tripping through JSON.
			tsBytes, err := json.Marshal(entry.Timestamp)
			if err != nil {
				t.Fatalf("history[%d].timestamp marshal failed: %v", i, err)
			}
			// JSON marshals time.Time as a quoted RFC 3339 string.
			var tsStr string
			if err := json.Unmarshal(tsBytes, &tsStr); err != nil {
				t.Fatalf("history[%d].timestamp is not a JSON string: %v", i, err)
			}
			if _, err := time.Parse(time.RFC3339, tsStr); err != nil {
				t.Fatalf("history[%d].timestamp %q is not a valid RFC 3339 string: %v", i, tsStr, err)
			}
		}

		// Also verify the raw JSON response has RFC 3339 timestamp strings.
		// Re-fetch to get the raw JSON.
		resp2, err := http.Get(server.URL + "/tracking/" + trackingNumber)
		if err != nil {
			t.Fatalf("second GET request failed: %v", err)
		}
		defer resp2.Body.Close()

		var rawResp struct {
			History []struct {
				Timestamp string `json:"timestamp"`
			} `json:"history"`
		}
		if err := json.NewDecoder(resp2.Body).Decode(&rawResp); err != nil {
			t.Fatalf("failed to decode raw history response: %v", err)
		}

		for i, entry := range rawResp.History {
			if _, err := time.Parse(time.RFC3339, entry.Timestamp); err != nil {
				t.Fatalf("raw history[%d].timestamp %q is not a valid RFC 3339 string: %v",
					i, entry.Timestamp, err)
			}
		}
	})
}
