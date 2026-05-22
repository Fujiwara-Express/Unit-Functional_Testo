package domain

import "errors"

var (
	// ErrNotFound indicates the requested resource was not found.
	ErrNotFound = errors.New("not found")

	// ErrValidation indicates a validation error in query parameters.
	ErrValidation = errors.New("validation error")
)
