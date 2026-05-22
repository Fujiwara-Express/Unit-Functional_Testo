package domain

import "errors"

var (
	// ErrNotFound indicates the requested resource was not found.
	ErrNotFound = errors.New("not found")

	// ErrValidation indicates a validation error.
	ErrValidation = errors.New("validation error")

	// ErrConflict indicates a conflict with the current state.
	ErrConflict = errors.New("conflict")

	// ErrServiceUnavailable indicates an external service is unavailable.
	ErrServiceUnavailable = errors.New("service unavailable")
)
