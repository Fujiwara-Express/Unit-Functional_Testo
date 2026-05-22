// Package apperrors defines typed application errors used across the routing service.
// Handlers return these errors; the global middleware maps them to HTTP responses.
package apperrors

import "fmt"

// ValidationError is returned when request input fails validation.
// Maps to HTTP 400.
type ValidationError struct {
	Message string
	Fields  []string // optional list of invalid field names
}

func (e *ValidationError) Error() string {
	if len(e.Fields) > 0 {
		return fmt.Sprintf("validation error on fields %v: %s", e.Fields, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// NotFoundError is returned when a requested resource does not exist.
// Maps to HTTP 404.
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s '%s' not found", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

// DuplicateError is returned when a resource with the same unique key already exists.
// Maps to HTTP 409.
type DuplicateError struct {
	Resource string
	Key      string
}

func (e *DuplicateError) Error() string {
	return fmt.Sprintf("%s '%s' already exists", e.Resource, e.Key)
}

// UpstreamUnavailableError is returned when an external dependency is unreachable.
// Maps to HTTP 503.
type UpstreamUnavailableError struct {
	Service string
	Cause   error
}

func (e *UpstreamUnavailableError) Error() string {
	return fmt.Sprintf("%s is unavailable: %v", e.Service, e.Cause)
}
