package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"user-service/internal/domain"
)

// PostgresRefreshTokenRepository is a PostgreSQL-backed implementation of RefreshTokenRepository.
type PostgresRefreshTokenRepository struct {
	db *sqlx.DB
}

// NewPostgresRefreshTokenRepository creates a new PostgresRefreshTokenRepository.
func NewPostgresRefreshTokenRepository(db *sqlx.DB) *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{db: db}
}

// Save persists a new refresh token record.
func (r *PostgresRefreshTokenRepository) Save(ctx context.Context, token domain.RefreshToken) error {
	const q = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`

	_, err := r.db.ExecContext(ctx, q, token.UserID, token.TokenHash, token.ExpiresAt)
	if err != nil {
		return fmt.Errorf("token_repo.Save: %w", err)
	}
	return nil
}

// FindByTokenHash retrieves a refresh token record by its hash.
func (r *PostgresRefreshTokenRepository) FindByTokenHash(ctx context.Context, hash string) (domain.RefreshToken, error) {
	const q = `SELECT token_id, user_id, token_hash, expires_at
	           FROM refresh_tokens WHERE token_hash = $1`

	var token domain.RefreshToken
	if err := r.db.GetContext(ctx, &token, q, hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.RefreshToken{}, domain.ErrNotFound
		}
		return domain.RefreshToken{}, fmt.Errorf("token_repo.FindByTokenHash: %w", err)
	}
	return token, nil
}

// Revoke deletes a refresh token record by its hash, effectively revoking it.
func (r *PostgresRefreshTokenRepository) Revoke(ctx context.Context, tokenHash string) error {
	const q = `DELETE FROM refresh_tokens WHERE token_hash = $1`

	res, err := r.db.ExecContext(ctx, q, tokenHash)
	if err != nil {
		return fmt.Errorf("token_repo.Revoke: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("token_repo.Revoke rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// DeleteExpired removes all refresh token records that have passed their expiry time.
func (r *PostgresRefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	const q = `DELETE FROM refresh_tokens WHERE expires_at < NOW()`

	_, err := r.db.ExecContext(ctx, q)
	if err != nil {
		return fmt.Errorf("token_repo.DeleteExpired: %w", err)
	}
	return nil
}
