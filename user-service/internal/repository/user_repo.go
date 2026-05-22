package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"user-service/internal/domain"
)

// PostgresUserRepository is a PostgreSQL-backed implementation of UserRepository.
type PostgresUserRepository struct {
	db *sqlx.DB
}

// NewPostgresUserRepository creates a new PostgresUserRepository.
func NewPostgresUserRepository(db *sqlx.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

// Create inserts a new user record and returns the persisted user (with generated fields).
func (r *PostgresUserRepository) Create(ctx context.Context, user domain.User) (domain.User, error) {
	const q = `
		INSERT INTO users (name, email, phone, password_hash, role, is_active)
		VALUES (:name, :email, :phone, :password_hash, :role, :is_active)
		RETURNING user_id, name, email, phone, password_hash, role, is_active, created_at`

	rows, err := r.db.NamedQueryContext(ctx, q, user)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return domain.User{}, domain.ErrEmailConflict
		}
		return domain.User{}, fmt.Errorf("user_repo.Create: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return domain.User{}, fmt.Errorf("user_repo.Create: no row returned")
	}
	var created domain.User
	if err := rows.StructScan(&created); err != nil {
		return domain.User{}, fmt.Errorf("user_repo.Create scan: %w", err)
	}
	return created, nil
}

// FindByEmail retrieves a user by email address.
func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	const q = `SELECT user_id, name, email, phone, password_hash, role, is_active, created_at
	           FROM users WHERE email = $1`

	var user domain.User
	if err := r.db.GetContext(ctx, &user, q, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, fmt.Errorf("user_repo.FindByEmail: %w", err)
	}
	return user, nil
}

// FindByID retrieves a user by their UUID.
func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (domain.User, error) {
	const q = `SELECT user_id, name, email, phone, password_hash, role, is_active, created_at
	           FROM users WHERE user_id = $1`

	var user domain.User
	if err := r.db.GetContext(ctx, &user, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, fmt.Errorf("user_repo.FindByID: %w", err)
	}
	return user, nil
}

// Update persists changes to an existing user record and returns the updated user.
func (r *PostgresUserRepository) Update(ctx context.Context, user domain.User) (domain.User, error) {
	const q = `
		UPDATE users
		SET name = :name, phone = :phone, password_hash = :password_hash
		WHERE user_id = :user_id
		RETURNING user_id, name, email, phone, password_hash, role, is_active, created_at`

	rows, err := r.db.NamedQueryContext(ctx, q, user)
	if err != nil {
		return domain.User{}, fmt.Errorf("user_repo.Update: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return domain.User{}, domain.ErrNotFound
	}
	var updated domain.User
	if err := rows.StructScan(&updated); err != nil {
		return domain.User{}, fmt.Errorf("user_repo.Update scan: %w", err)
	}
	return updated, nil
}

// SetStatus sets the is_active flag for the given user ID.
func (r *PostgresUserRepository) SetStatus(ctx context.Context, id string, active bool) error {
	const q = `UPDATE users SET is_active = $1 WHERE user_id = $2`

	res, err := r.db.ExecContext(ctx, q, active, id)
	if err != nil {
		return fmt.Errorf("user_repo.SetStatus: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("user_repo.SetStatus rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
