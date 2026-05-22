package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode"

	"pgregory.net/rapid"

	"tracking-service/internal/middleware"
	"tracking-service/internal/models"
	"tracking-service/internal/repository"
)

// validStatusValues lists all known valid Status values for use in generators.
var validStatusValues = []string{
	"CREATED",
	"PICKED_UP",
	"ARRIVED_AT_HUB",
	"IN_TRANSIT",
	"OUT_FOR_DELIVERY",
	"DELIVERED",
	"FAILED_DELIVERY",
	"RETURNED",
}

// drawValidTrackingNumber generates a non-empty tracking number string.
func drawValidTrackingNumber(t *rapid.T) string {
	// Generate a non-empty ASCII string (letters and digits).
	return rapid.StringMatching(`[A-Za-z0-9\-]{1,30}`).Draw(t, "tracking_number")
}

// drawValidRFC3339Timestamp generates a valid RFC 3339 timestamp string.
func drawValidRFC3339Timestamp(t *rapid.T) string {
	// Generate a year in a reasonable range to avoid time.Time overflow.
	year := rapid.IntRange(2000, 2099).Draw(t, "year")
	month := rapid.IntRange(1, 12).Draw(t, "month")
	// Use day range 1-28 to avoid month-end edge cases.
	day := rapid.IntRange(1, 28).Draw(t, "day")
	hour := rapid.IntRange(0, 23).Draw(t, "hour")
	minute := rapid.IntRange(0, 59).Draw(t, "minute")
	second := rapid.IntRange(0, 59).Draw(t, "second")

	ts := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC)
	return ts.Format(time.RFC3339)
}

// drawValidCreateEventRequest generates a random valid CreateEventRequest.
func drawValidCreateEventRequest(t *rapid.T) models.CreateEventRequest {
	req := models.CreateEventRequest{
		TrackingNumber: drawValidTrackingNumber(t),
		Status:         rapid.SampledFrom(validStatusValues).Draw(t, "status"),
		Timestamp:      drawValidRFC3339Timestamp(t),
	}

	// Optionally include optional fields.
	if rapid.Bool().Draw(t, "has_location") {
		req.Location = rapid.StringMatching(`[A-Za-z ]{1,20}`).Draw(t, "location")
	}
	if rapid.Bool().Draw(t, "has_hub_id") {
		req.HubID = rapid.StringMatching(`[A-Za-z0-9\-]{1,10}`).Draw(t, "hub_id")
	}
	if rapid.Bool().Draw(t, "has_notes") {
		req.Notes = rapid.StringMatching(`[A-Za-z0-9 ]{1,30}`).Draw(t, "notes")
	}

	return req
}

// isUUIDLike checks that a string looks like a UUID v4 (contains hyphens, length ~36).
func isUUIDLike(s string) bool {
	if len(s) != 36 {
		return false
	}
	if !strings.Contains(s, "-") {
		return false
	}
	// Verify the hyphen positions: 8-4-4-4-12
	parts := strings.Split(s, "-")
	if len(parts) != 5 {
		return false
	}
	expectedLengths := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != expectedLengths[i] {
			return false
		}
		for _, ch := range part {
			if !unicode.Is(unicode.ASCII_Hex_Digit, ch) {
				return false
			}
		}
	}
	return true
}

// newTestServer creates an httptest.Server with CorrelationIDMiddleware wrapping
// an EventHandler backed by the given MockRepository.
func newTestServer(repo *repository.MockRepository) (*httptest.Server, *repository.MockRepository) {
	handler := NewEventHandler(repo, 5*time.Second)
	return httptest.NewServer(middleware.CorrelationIDMiddleware(handler)), repo
}

// Feature: tracking-service, Property 1: Valid event creation round-trip
// Validates: Requirements 1.1, 1.7, 4.2
func TestProperty1_ValidEventCreationRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := repository.NewMockRepository()
		server, _ := newTestServer(repo)
		defer server.Close()

		req := drawValidCreateEventRequest(t)

		// Encode the request body.
		body, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("failed to marshal request: %v", err)
		}

		// POST to the test server.
		resp, err := http.Post(server.URL, "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("POST request failed: %v", err)
		}
		defer resp.Body.Close()

		// Assert HTTP 201.
		if resp.StatusCode != http.StatusCreated {
			var errResp models.ErrorResponse
			json.NewDecoder(resp.Body).Decode(&errResp) //nolint:errcheck
			t.Fatalf("expected HTTP 201, got %d; error: %s", resp.StatusCode, errResp.Error)
		}

		// Decode the CreateEventResponse.
		var createResp models.CreateEventResponse
		if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
			t.Fatalf("failed to decode CreateEventResponse: %v", err)
		}

		// Assert event_id is non-empty and looks like a UUID.
		if createResp.EventID == "" {
			t.Fatal("event_id in response is empty")
		}
		if !isUUIDLike(createResp.EventID) {
			t.Fatalf("event_id %q does not look like a UUID (expected 8-4-4-4-12 hex format)", createResp.EventID)
		}

		// Retrieve the stored event from the mock repository.
		ctx := t.Context()
		events, err := repo.GetEventsByTrackingNumber(ctx, req.TrackingNumber)
		if err != nil {
			t.Fatalf("GetEventsByTrackingNumber(%q) failed: %v", req.TrackingNumber, err)
		}
		if len(events) == 0 {
			t.Fatalf("no events stored for tracking_number %q", req.TrackingNumber)
		}

		// Find the event matching the returned event_id.
		var stored *models.TrackingEvent
		for i := range events {
			if events[i].EventID == createResp.EventID {
				stored = &events[i]
				break
			}
		}
		if stored == nil {
			t.Fatalf("event with event_id %q not found in repository", createResp.EventID)
		}

		// Assert all fields are preserved.
		if stored.TrackingNumber != req.TrackingNumber {
			t.Fatalf("tracking_number mismatch: want %q, got %q", req.TrackingNumber, stored.TrackingNumber)
		}
		if string(stored.Status) != req.Status {
			t.Fatalf("status mismatch: want %q, got %q", req.Status, stored.Status)
		}
		if stored.Location != req.Location {
			t.Fatalf("location mismatch: want %q, got %q", req.Location, stored.Location)
		}
		if stored.HubID != req.HubID {
			t.Fatalf("hub_id mismatch: want %q, got %q", req.HubID, stored.HubID)
		}
		if stored.Notes != req.Notes {
			t.Fatalf("notes mismatch: want %q, got %q", req.Notes, stored.Notes)
		}

		// Assert created_by_service is "tracking-service".
		if stored.CreatedByService != "tracking-service" {
			t.Fatalf("created_by_service mismatch: want %q, got %q", "tracking-service", stored.CreatedByService)
		}

		// Assert timestamp is preserved (compare as RFC 3339 strings to avoid timezone issues).
		expectedTS, err := time.Parse(time.RFC3339, req.Timestamp)
		if err != nil {
			t.Fatalf("failed to parse expected timestamp %q: %v", req.Timestamp, err)
		}
		if !stored.Timestamp.Equal(expectedTS) {
			t.Fatalf("timestamp mismatch: want %v, got %v", expectedTS, stored.Timestamp)
		}
	})
}

// Feature: tracking-service, Property 11: All responses carry Content-Type: application/json
// Validates: Requirements 6.2
func TestProperty11_ContentTypeOnAllResponses(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := repository.NewMockRepository()
		server, _ := newTestServer(repo)
		defer server.Close()

		// Generate a mix of valid and invalid requests.
		requestKind := rapid.IntRange(0, 4).Draw(t, "request_kind")

		var (
			body        []byte
			contentType string
			err         error
		)
		contentType = "application/json"

		switch requestKind {
		case 0:
			// Valid request.
			req := drawValidCreateEventRequest(t)
			body, err = json.Marshal(req)
			if err != nil {
				t.Fatalf("failed to marshal valid request: %v", err)
			}

		case 1:
			// Missing required fields (tracking_number absent).
			req := models.CreateEventRequest{
				Status:    rapid.SampledFrom(validStatusValues).Draw(t, "status"),
				Timestamp: drawValidRFC3339Timestamp(t),
			}
			body, err = json.Marshal(req)
			if err != nil {
				t.Fatalf("failed to marshal request: %v", err)
			}

		case 2:
			// Invalid status value.
			req := models.CreateEventRequest{
				TrackingNumber: drawValidTrackingNumber(t),
				Status:         "INVALID_STATUS_" + rapid.StringMatching(`[A-Z]{3}`).Draw(t, "suffix"),
				Timestamp:      drawValidRFC3339Timestamp(t),
			}
			body, err = json.Marshal(req)
			if err != nil {
				t.Fatalf("failed to marshal request: %v", err)
			}

		case 3:
			// Invalid timestamp format.
			req := models.CreateEventRequest{
				TrackingNumber: drawValidTrackingNumber(t),
				Status:         rapid.SampledFrom(validStatusValues).Draw(t, "status"),
				Timestamp:      "not-a-timestamp-" + rapid.StringMatching(`[0-9]{4}`).Draw(t, "suffix"),
			}
			body, err = json.Marshal(req)
			if err != nil {
				t.Fatalf("failed to marshal request: %v", err)
			}

		case 4:
			// Malformed JSON body.
			body = []byte(`{invalid json`)
			contentType = "text/plain"
		}

		resp, err := http.Post(server.URL, contentType, bytes.NewReader(body))
		if err != nil {
			t.Fatalf("POST request failed: %v", err)
		}
		defer resp.Body.Close()

		// Assert Content-Type header contains "application/json".
		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "application/json") {
			t.Fatalf("expected Content-Type to contain 'application/json', got %q (status %d)", ct, resp.StatusCode)
		}
	})
}
