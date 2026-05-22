package validator

import (
	"testing"
	"time"

	"pgregory.net/rapid"
	"tracking-service/internal/models"
)

// validStatuses lists all known valid Status values.
var validStatuses = []string{
	string(models.StatusCreated),
	string(models.StatusPickedUp),
	string(models.StatusArrivedAtHub),
	string(models.StatusInTransit),
	string(models.StatusOutForDelivery),
	string(models.StatusDelivered),
	string(models.StatusFailedDelivery),
	string(models.StatusReturned),
}

func isValidStatus(s string) bool {
	for _, v := range validStatuses {
		if s == v {
			return true
		}
	}
	return false
}

// Feature: tracking-service, Property 2: Missing required fields return HTTP 400 identifying the missing field(s)
func TestProperty2_MissingRequiredFields(t *testing.T) {
	// Validates: Requirements 1.2, 1.3
	rapid.Check(t, func(t *rapid.T) {
		// Generate a bitmask 0–6 (exclude 7 = all present, which is a valid request)
		bitmask := rapid.IntRange(0, 6).Draw(t, "bitmask")

		// Bit 0 = tracking_number present, bit 1 = status present, bit 2 = timestamp present
		trackingNumberPresent := (bitmask & 1) != 0
		statusPresent := (bitmask & 2) != 0
		timestampPresent := (bitmask & 4) != 0

		req := models.CreateEventRequest{}

		if trackingNumberPresent {
			req.TrackingNumber = "TRK-001"
		}
		if statusPresent {
			req.Status = string(models.StatusCreated)
		}
		if timestampPresent {
			req.Timestamp = time.Now().UTC().Format(time.RFC3339)
		}

		errs := ValidateCreateEventRequest(req)

		// There must be at least one error since at least one required field is missing
		if len(errs) == 0 {
			t.Fatalf("expected validation errors for bitmask=%d, got none", bitmask)
		}

		// Build a set of error fields for quick lookup
		errorFields := make(map[string]bool)
		for _, e := range errs {
			errorFields[e.Field] = true
		}

		// Every absent field must appear in the errors
		if !trackingNumberPresent && !errorFields["tracking_number"] {
			t.Fatalf("expected error for missing tracking_number, got errors: %v", errs)
		}
		if !statusPresent && !errorFields["status"] {
			t.Fatalf("expected error for missing status, got errors: %v", errs)
		}
		if !timestampPresent && !errorFields["timestamp"] {
			t.Fatalf("expected error for missing timestamp, got errors: %v", errs)
		}
	})
}

// Feature: tracking-service, Property 3: Invalid status value returns HTTP 422
func TestProperty3_InvalidStatus(t *testing.T) {
	// Validates: Requirements 1.4
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.String().Draw(t, "status")

		// Skip valid status strings — we only want to test invalid ones
		if isValidStatus(s) {
			t.Skip()
		}

		if ValidateStatus(s) {
			t.Fatalf("expected ValidateStatus(%q) to return false, got true", s)
		}
	})
}

// Feature: tracking-service, Property 4: Invalid timestamp format returns HTTP 422
func TestProperty4_InvalidTimestamp(t *testing.T) {
	// Validates: Requirements 1.5
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.String().Draw(t, "timestamp")

		// Skip strings that happen to be valid RFC 3339
		if _, err := time.Parse(time.RFC3339, s); err == nil {
			t.Skip()
		}

		if ValidateTimestamp(s) {
			t.Fatalf("expected ValidateTimestamp(%q) to return false, got true", s)
		}
	})
}

// Feature: tracking-service, Property 9: Bulk query count boundary
func TestProperty9_BulkCountBoundary(t *testing.T) {
	// Validates: Requirements 3.2, 3.4

	// Sub-property: 1–100 non-empty strings → no error
	t.Run("valid_range", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			count := rapid.IntRange(1, 100).Draw(t, "count")
			numbers := make([]string, count)
			for i := range numbers {
				// Generate non-empty strings
				numbers[i] = rapid.StringMatching(`[a-zA-Z0-9]{1,20}`).Draw(t, "number")
			}

			errs := ValidateBulkNumbers(numbers)
			if errs != nil {
				t.Fatalf("expected no errors for %d numbers, got: %v", count, errs)
			}
		})
	})

	// Sub-property: 101+ non-empty strings → error
	t.Run("over_limit", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			count := rapid.IntRange(101, 200).Draw(t, "count")
			numbers := make([]string, count)
			for i := range numbers {
				numbers[i] = rapid.StringMatching(`[a-zA-Z0-9]{1,20}`).Draw(t, "number")
			}

			errs := ValidateBulkNumbers(numbers)
			if errs == nil {
				t.Fatalf("expected an error for %d numbers (over limit), got nil", count)
			}
		})
	})
}
