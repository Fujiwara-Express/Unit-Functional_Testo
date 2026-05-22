// Package domain defines the core types and constants for the user service.
package domain

import (
	"errors"
	"time"
)

// Sentinel errors used across repository and service layers.
var (
	ErrNotFound      = errors.New("not found")
	ErrEmailConflict = errors.New("email already exists")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
)

// Role represents the permission level of an account.
type Role string

const (
	RoleCustomer Role = "CUSTOMER"
	RoleCourier  Role = "COURIER"
	RoleAdmin    Role = "ADMIN"
)

// AvailabilityStatus represents a sender's current availability for deliveries.
type AvailabilityStatus string

const (
	StatusAvailable   AvailabilityStatus = "available"
	StatusUnavailable AvailabilityStatus = "unavailable"
)

// User is the core account entity stored in the database.
type User struct {
	ID           string    `db:"user_id"       json:"user_id"`
	Name         string    `db:"name"          json:"name"`
	Email        string    `db:"email"         json:"email"`
	Phone        string    `db:"phone"         json:"phone"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Role         Role      `db:"role"          json:"role"`
	IsActive     bool      `db:"is_active"     json:"is_active"`
	CreatedAt    time.Time `db:"created_at"    json:"created_at"`
}

// RefreshToken represents a persisted refresh token record.
type RefreshToken struct {
	TokenID   string    `db:"token_id"`
	UserID    string    `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	ExpiresAt time.Time `db:"expires_at"`
}
