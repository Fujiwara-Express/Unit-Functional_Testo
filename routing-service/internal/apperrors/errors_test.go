package apperrors_test

import (
	"strings"
	"testing"

	"routing-service/internal/apperrors"
)

func TestValidationError_WithFields(t *testing.T) {
	err := &apperrors.ValidationError{Message: "bad input", Fields: []string{"hub_id", "city_code"}}
	msg := err.Error()
	if !strings.Contains(msg, "hub_id") || !strings.Contains(msg, "city_code") {
		t.Errorf("expected field names in error message, got: %s", msg)
	}
}

func TestValidationError_WithoutFields(t *testing.T) {
	err := &apperrors.ValidationError{Message: "bad input"}
	msg := err.Error()
	if !strings.Contains(msg, "bad input") {
		t.Errorf("expected message in error, got: %s", msg)
	}
}

func TestNotFoundError_WithID(t *testing.T) {
	err := &apperrors.NotFoundError{Resource: "edge", ID: "e-123"}
	msg := err.Error()
	if !strings.Contains(msg, "edge") || !strings.Contains(msg, "e-123") {
		t.Errorf("unexpected error message: %s", msg)
	}
}

func TestNotFoundError_WithoutID(t *testing.T) {
	err := &apperrors.NotFoundError{Resource: "route"}
	msg := err.Error()
	if !strings.Contains(msg, "route") {
		t.Errorf("unexpected error message: %s", msg)
	}
}

func TestDuplicateError(t *testing.T) {
	err := &apperrors.DuplicateError{Resource: "hub", Key: "HUB_JKT"}
	msg := err.Error()
	if !strings.Contains(msg, "HUB_JKT") {
		t.Errorf("unexpected error message: %s", msg)
	}
}

func TestUpstreamUnavailableError(t *testing.T) {
	cause := &apperrors.ValidationError{Message: "timeout"}
	err := &apperrors.UpstreamUnavailableError{Service: "DeliveryService", Cause: cause}
	msg := err.Error()
	if !strings.Contains(msg, "DeliveryService") {
		t.Errorf("unexpected error message: %s", msg)
	}
}
