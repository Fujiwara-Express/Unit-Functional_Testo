// Package repository defines the data-access interfaces for the user service.
package repository

import (
	"context"

	"user-service/internal/domain"
)

// UserRepository defines persistence operations for user accounts.
type UserRepository interface {
	Create(ctx context.Context, user domain.User) (domain.User, error)
	FindByEmail(ctx context.Context, email string) (domain.User, error)
	FindByID(ctx context.Context, id string) (domain.User, error)
	Update(ctx context.Context, user domain.User) (domain.User, error)
	SetStatus(ctx context.Context, id string, active bool) error
}

// RefreshTokenRepository defines persistence operations for refresh tokens.
type RefreshTokenRepository interface {
	Save(ctx context.Context, token domain.RefreshToken) error
	FindByTokenHash(ctx context.Context, hash string) (domain.RefreshToken, error)
	Revoke(ctx context.Context, tokenHash string) error
	DeleteExpired(ctx context.Context) error
}
