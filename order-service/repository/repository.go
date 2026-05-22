package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"order-service/types"
)

// OrderRepository implements types.OrderRepository using database/sql.
type OrderRepository struct {
	db *sql.DB
}

// NewOrderRepository creates a new OrderRepository with the given database connection.
func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// SaveOrder inserts a new order into the database.
func (r *OrderRepository) SaveOrder(ctx context.Context, order *types.Order) error {
	const query = `INSERT INTO orders (
		order_id, tracking_number, sender_user_id, sender_name, sender_address, sender_phone, sender_city_code,
		receiver_name, receiver_address, receiver_phone, receiver_city_code,
		weight, length, width, height, service_type, price, is_cod, cod_amount, insurance,
		item_description, status, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)`

	_, err := r.db.ExecContext(ctx, query,
		order.OrderID,
		order.TrackingNumber,
		order.SenderUserID,
		order.SenderName,
		order.SenderAddress,
		order.SenderPhone,
		order.SenderCityCode,
		order.ReceiverName,
		order.ReceiverAddress,
		order.ReceiverPhone,
		order.ReceiverCityCode,
		order.Weight,
		order.Length,
		order.Width,
		order.Height,
		string(order.ServiceType),
		order.Price,
		order.IsCOD,
		order.CODAmount,
		order.Insurance,
		order.ItemDescription,
		string(order.Status),
		order.CreatedAt,
		order.UpdatedAt,
	)
	return err
}

// FindOrderByID retrieves an order by its ID.
func (r *OrderRepository) FindOrderByID(ctx context.Context, orderID string) (*types.Order, error) {
	const query = `SELECT order_id, tracking_number, sender_user_id, sender_name, sender_address, sender_phone, sender_city_code,
		receiver_name, receiver_address, receiver_phone, receiver_city_code,
		weight, length, width, height, service_type, price, is_cod, cod_amount, insurance,
		item_description, status, created_at, updated_at
	FROM orders WHERE order_id = $1`

	row := r.db.QueryRowContext(ctx, query, orderID)

	var order types.Order
	var serviceType string
	var status string

	err := row.Scan(
		&order.OrderID,
		&order.TrackingNumber,
		&order.SenderUserID,
		&order.SenderName,
		&order.SenderAddress,
		&order.SenderPhone,
		&order.SenderCityCode,
		&order.ReceiverName,
		&order.ReceiverAddress,
		&order.ReceiverPhone,
		&order.ReceiverCityCode,
		&order.Weight,
		&order.Length,
		&order.Width,
		&order.Height,
		&serviceType,
		&order.Price,
		&order.IsCOD,
		&order.CODAmount,
		&order.Insurance,
		&order.ItemDescription,
		&status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("order %s: %w", orderID, types.ErrNotFound)
		}
		return nil, err
	}

	order.ServiceType = types.ServiceType(serviceType)
	order.Status = types.OrderStatus(status)

	return &order, nil
}

// FindOrders retrieves orders matching the given filter parameters.
func (r *OrderRepository) FindOrders(ctx context.Context, params types.ListOrdersParams) ([]*types.Order, error) {
	const baseQuery = `SELECT order_id, tracking_number, sender_user_id, sender_name, sender_address, sender_phone, sender_city_code,
		receiver_name, receiver_address, receiver_phone, receiver_city_code,
		weight, length, width, height, service_type, price, is_cod, cod_amount, insurance,
		item_description, status, created_at, updated_at
	FROM orders`

	var conditions []string
	var args []interface{}
	argIdx := 1

	if params.UserID != "" {
		conditions = append(conditions, fmt.Sprintf("sender_user_id = $%d", argIdx))
		args = append(args, params.UserID)
		argIdx++
	}

	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(params.Status))
		argIdx++
	}

	query := baseQuery
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 10
	}

	offset := 0
	if params.Page > 0 {
		offset = (params.Page - 1) * limit
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*types.Order
	for rows.Next() {
		var order types.Order
		var serviceType string
		var status string

		err := rows.Scan(
			&order.OrderID,
			&order.TrackingNumber,
			&order.SenderUserID,
			&order.SenderName,
			&order.SenderAddress,
			&order.SenderPhone,
			&order.SenderCityCode,
			&order.ReceiverName,
			&order.ReceiverAddress,
			&order.ReceiverPhone,
			&order.ReceiverCityCode,
			&order.Weight,
			&order.Length,
			&order.Width,
			&order.Height,
			&serviceType,
			&order.Price,
			&order.IsCOD,
			&order.CODAmount,
			&order.Insurance,
			&order.ItemDescription,
			&status,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		order.ServiceType = types.ServiceType(serviceType)
		order.Status = types.OrderStatus(status)
		orders = append(orders, &order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

// UpdateOrder updates an existing order in the database.
func (r *OrderRepository) UpdateOrder(ctx context.Context, order *types.Order) error {
	const query = `UPDATE orders SET
		tracking_number=$1, sender_user_id=$2, sender_name=$3, sender_address=$4, sender_phone=$5, sender_city_code=$6,
		receiver_name=$7, receiver_address=$8, receiver_phone=$9, receiver_city_code=$10,
		weight=$11, length=$12, width=$13, height=$14, service_type=$15, price=$16,
		is_cod=$17, cod_amount=$18, insurance=$19, item_description=$20, status=$21, updated_at=$22
	WHERE order_id=$23`

	_, err := r.db.ExecContext(ctx, query,
		order.TrackingNumber,
		order.SenderUserID,
		order.SenderName,
		order.SenderAddress,
		order.SenderPhone,
		order.SenderCityCode,
		order.ReceiverName,
		order.ReceiverAddress,
		order.ReceiverPhone,
		order.ReceiverCityCode,
		order.Weight,
		order.Length,
		order.Width,
		order.Height,
		string(order.ServiceType),
		order.Price,
		order.IsCOD,
		order.CODAmount,
		order.Insurance,
		order.ItemDescription,
		string(order.Status),
		order.UpdatedAt,
		order.OrderID,
	)
	return err
}
