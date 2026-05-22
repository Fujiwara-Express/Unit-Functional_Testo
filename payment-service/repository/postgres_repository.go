package repository

import (
	"context"
	"database/sql"
	"time"

	"payment-service/domain"
	"payment-service/service"

	"github.com/lib/pq"
)

// PostgresRepository implements domain.Payment_Repository using PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreatePayment inserts a new payment record into the payments table.
func (r *PostgresRepository) CreatePayment(ctx context.Context, p *domain.Payment) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO payments
			(payment_id, order_id, user_id, amount, method, status, external_ref, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		p.PaymentID,
		p.OrderID,
		p.UserID,
		p.Amount,
		string(p.Method),
		string(p.Status),
		p.ExternalRef,
		p.CreatedAt,
		p.UpdatedAt,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return &service.DuplicateOrderError{OrderID: p.OrderID}
		}
		return err
	}
	return nil
}

// GetPaymentByID retrieves a payment by its primary key.
func (r *PostgresRepository) GetPaymentByID(ctx context.Context, paymentID string) (*domain.Payment, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT payment_id, order_id, user_id, amount, method, status, external_ref, created_at, updated_at
		FROM payments WHERE payment_id = $1`, paymentID)
	return scanPayment(row)
}

// GetPaymentByOrderID retrieves a payment by its associated order ID.
func (r *PostgresRepository) GetPaymentByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT payment_id, order_id, user_id, amount, method, status, external_ref, created_at, updated_at
		FROM payments WHERE order_id = $1`, orderID)
	return scanPayment(row)
}

// GetPaymentByExternalRef retrieves a payment by its external gateway reference.
func (r *PostgresRepository) GetPaymentByExternalRef(ctx context.Context, externalRef string) (*domain.Payment, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT payment_id, order_id, user_id, amount, method, status, external_ref, created_at, updated_at
		FROM payments WHERE external_ref = $1`, externalRef)
	return scanPayment(row)
}

// UpdatePaymentStatus updates the status and external reference of a payment.
func (r *PostgresRepository) UpdatePaymentStatus(ctx context.Context, orderID string, status domain.PaymentStatus, externalRef string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE payments SET status = $1, external_ref = $2, updated_at = $3
		WHERE order_id = $4`,
		string(status), externalRef, time.Now(), orderID,
	)
	return err
}

// CreateCodCollection inserts a new COD collection record.
func (r *PostgresRepository) CreateCodCollection(ctx context.Context, col *domain.CodCollection) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO cod_collections
			(collection_id, order_id, courier_id, amount_collected, collected_at, remittance_status)
		VALUES
			($1, $2, $3, $4, $5, $6)`,
		col.CollectionID,
		col.OrderID,
		col.CourierID,
		col.AmountCollected,
		col.CollectedAt,
		string(col.RemittanceStatus),
	)
	return err
}

func scanPayment(row *sql.Row) (*domain.Payment, error) {
	p := &domain.Payment{}
	var method, status string
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&p.PaymentID,
		&p.OrderID,
		&p.UserID,
		&p.Amount,
		&method,
		&status,
		&p.ExternalRef,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	p.Method = domain.PaymentMethod(method)
	p.Status = domain.PaymentStatus(status)
	p.CreatedAt = createdAt
	p.UpdatedAt = updatedAt
	return p, nil
}
