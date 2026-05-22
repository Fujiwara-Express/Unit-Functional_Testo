package validator

import (
	"fmt"
	"time"
	"tracking-service/internal/models"
)

// ValidateStatus returns true if s is a known Status value.
func ValidateStatus(s string) bool {
	switch models.Status(s) {
	case models.StatusCreated,
		models.StatusPickedUp,
		models.StatusArrivedAtHub,
		models.StatusInTransit,
		models.StatusOutForDelivery,
		models.StatusDelivered,
		models.StatusFailedDelivery,
		models.StatusReturned:
		return true
	}
	return false
}

// ValidateTimestamp returns true if s is a valid RFC 3339 string.
func ValidateTimestamp(s string) bool {
	_, err := time.Parse(time.RFC3339, s)
	return err == nil
}

// ValidateCreateEventRequest validates the POST /tracking/events body.
// Returns a list of field-level errors.
func ValidateCreateEventRequest(req models.CreateEventRequest) []models.ValidationError {
	var errs []models.ValidationError

	// Presence checks
	if req.TrackingNumber == "" {
		errs = append(errs, models.ValidationError{
			Field:   "tracking_number",
			Message: "tracking_number is required",
		})
	}
	if req.Status == "" {
		errs = append(errs, models.ValidationError{
			Field:   "status",
			Message: "status is required",
		})
	}
	if req.Timestamp == "" {
		errs = append(errs, models.ValidationError{
			Field:   "timestamp",
			Message: "timestamp is required",
		})
	}

	// Format/enum checks (only when field is non-empty)
	if req.Status != "" && !ValidateStatus(req.Status) {
		errs = append(errs, models.ValidationError{
			Field:   "status",
			Message: fmt.Sprintf("invalid status '%s'; valid values: CREATED, PICKED_UP, ARRIVED_AT_HUB, IN_TRANSIT, OUT_FOR_DELIVERY, DELIVERED, FAILED_DELIVERY, RETURNED", req.Status),
		})
	}
	if req.Timestamp != "" && !ValidateTimestamp(req.Timestamp) {
		errs = append(errs, models.ValidationError{
			Field:   "timestamp",
			Message: fmt.Sprintf("timestamp must be RFC 3339; got '%s'", req.Timestamp),
		})
	}

	return errs
}

// ValidateBulkNumbers validates the numbers slice (1–100 items, non-empty strings).
func ValidateBulkNumbers(numbers []string) []models.ValidationError {
	if len(numbers) == 0 {
		return []models.ValidationError{
			{Field: "numbers", Message: "query parameter 'numbers' is required"},
		}
	}

	for _, n := range numbers {
		if n == "" {
			return []models.ValidationError{
				{Field: "numbers", Message: "tracking numbers must not be empty strings"},
			}
		}
	}

	if len(numbers) > 100 {
		return []models.ValidationError{
			{Field: "numbers", Message: fmt.Sprintf("maximum 100 tracking numbers per request; got %d", len(numbers))},
		}
	}

	return nil
}
