package domain

import "errors"

var (
	// ErrNotFound indicates the requested resource was not found.
	ErrNotFound = errors.New("not found")

	// ErrInvalidTransition indicates an invalid status transition was attempted.
	ErrInvalidTransition = errors.New("invalid status transition")

	// ErrConflict indicates a conflict with the current state.
	ErrConflict = errors.New("conflict")

	// ErrValidation indicates a validation error.
	ErrValidation = errors.New("validation error")

	// ErrServiceUnavailable indicates an external service is unavailable.
	ErrServiceUnavailable = errors.New("service unavailable")
)
