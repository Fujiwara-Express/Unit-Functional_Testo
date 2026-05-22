package models

import (
	"encoding/json"
	"testing"
	"time"
	"unicode"

	"pgregory.net/rapid"
)

// Feature: tracking-service, Property 10: JSON serialization round-trip
func TestProperty10_JSONRoundTrip(t *testing.T) {
	validStatuses := []string{
		"CREATED",
		"PICKED_UP",
		"ARRIVED_AT_HUB",
		"IN_TRANSIT",
		"OUT_FOR_DELIVERY",
		"DELIVERED",
		"FAILED_DELIVERY",
		"RETURNED",
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate a random non-empty tracking number using unicode letters
		trackingNumber := rapid.StringOf(rapid.RuneFrom(nil, unicode.Letter)).Filter(func(s string) bool {
			return len(s) > 0
		}).Draw(t, "tracking_number")

		// Generate a random valid status
		status := rapid.SampledFrom(validStatuses).Draw(t, "status")

		// Generate a random timestamp as RFC 3339 string
		// Use a fixed epoch range to keep times reasonable
		unixSec := rapid.Int64Range(0, 9999999999).Draw(t, "unix_sec")
		ts := time.Unix(unixSec, 0).UTC()
		timestamp := ts.Format(time.RFC3339)

		// Generate optional string fields (can be empty)
		location := rapid.String().Draw(t, "location")
		hubID := rapid.String().Draw(t, "hub_id")
		notes := rapid.String().Draw(t, "notes")

		original := CreateEventRequest{
			TrackingNumber: trackingNumber,
			Status:         status,
			Timestamp:      timestamp,
			Location:       location,
			HubID:          hubID,
			Notes:          notes,
		}

		// First cycle: marshal → unmarshal
		data1, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("first marshal failed: %v", err)
		}

		var decoded1 CreateEventRequest
		if err := json.Unmarshal(data1, &decoded1); err != nil {
			t.Fatalf("first unmarshal failed: %v", err)
		}

		// Second cycle: marshal → unmarshal
		data2, err := json.Marshal(decoded1)
		if err != nil {
			t.Fatalf("second marshal failed: %v", err)
		}

		var decoded2 CreateEventRequest
		if err := json.Unmarshal(data2, &decoded2); err != nil {
			t.Fatalf("second unmarshal failed: %v", err)
		}

		// Assert all fields are equal after two round-trips
		if decoded2.TrackingNumber != original.TrackingNumber {
			t.Fatalf("TrackingNumber mismatch: got %q, want %q", decoded2.TrackingNumber, original.TrackingNumber)
		}
		if decoded2.Status != original.Status {
			t.Fatalf("Status mismatch: got %q, want %q", decoded2.Status, original.Status)
		}
		if decoded2.Timestamp != original.Timestamp {
			t.Fatalf("Timestamp mismatch: got %q, want %q", decoded2.Timestamp, original.Timestamp)
		}
		if decoded2.Location != original.Location {
			t.Fatalf("Location mismatch: got %q, want %q", decoded2.Location, original.Location)
		}
		if decoded2.HubID != original.HubID {
			t.Fatalf("HubID mismatch: got %q, want %q", decoded2.HubID, original.HubID)
		}
		if decoded2.Notes != original.Notes {
			t.Fatalf("Notes mismatch: got %q, want %q", decoded2.Notes, original.Notes)
		}
	})
}
