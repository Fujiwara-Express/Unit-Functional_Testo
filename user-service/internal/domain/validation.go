package domain

import (
	"errors"
	"regexp"
)

// Sentinel validation errors.
var (
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrInvalidPhone     = errors.New("invalid phone number format")
	ErrInvalidRole      = errors.New("invalid role: must be CUSTOMER, COURIER, or ADMIN")
)

// emailRegex matches a basic RFC-5321-style email address.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// phoneRegex matches E.164-style phone numbers: optional leading +, then 7–15 digits.
var phoneRegex = regexp.MustCompile(`^\+?[0-9]{7,15}$`)

// ValidateEmail returns ErrInvalidEmail when s does not match a valid email format.
func ValidateEmail(s string) error {
	if !emailRegex.MatchString(s) {
		return ErrInvalidEmail
	}
	return nil
}

// ValidatePassword returns ErrPasswordTooShort when s has fewer than 8 characters.
func ValidatePassword(s string) error {
	if len(s) < 8 {
		return ErrPasswordTooShort
	}
	return nil
}

// ValidatePhone returns ErrInvalidPhone when s does not match a valid phone format.
func ValidatePhone(s string) error {
	if s == "" {
		return nil // phone is optional on some paths; callers enforce presence when required
	}
	if !phoneRegex.MatchString(s) {
		return ErrInvalidPhone
	}
	return nil
}

// ValidateRole returns ErrInvalidRole when r is not one of the recognised Role constants.
func ValidateRole(r Role) error {
	switch r {
	case RoleCustomer, RoleCourier, RoleAdmin:
		return nil
	default:
		return ErrInvalidRole
	}
}
