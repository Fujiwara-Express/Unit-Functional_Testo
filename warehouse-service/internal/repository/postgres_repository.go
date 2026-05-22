package repository

import (
	"context"
	"database/sql"

	"github.com/fujiwara-express/warehouse-service/internal/domain"
)

type postgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository membuat koneksi repositori gudang ke PostgreSQL
func NewPostgresRepository(db *sql.DB) WarehouseRepository {
	return &postgresRepository{db: db}
}

// SaveItem akan menyimpan barang baru atau memperbarui data barang lama (Upsert)
func (r *postgresRepository) SaveItem(ctx context.Context, item domain.Item) error {
	query := `
		INSERT INTO items (item_id, name, quantity, location)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (item_id) 
		DO UPDATE SET name = EXCLUDED.name, quantity = EXCLUDED.quantity, location = EXCLUDED.location`
	
	_, err := r.db.ExecContext(ctx, query, item.ID, item.Name, item.Quantity, item.Location)
	return err
}

// GetItemByID mencari spesifik barang di database berdasarkan ID
func (r *postgresRepository) GetItemByID(ctx context.Context, id string) (domain.Item, error) {
	var item domain.Item
	query := `SELECT item_id, name, quantity, location FROM items WHERE item_id = $1`
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(&item.ID, &item.Name, &item.Quantity, &item.Location)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Item{}, domain.ErrItemNotFound
		}
		return domain.Item{}, err
	}
	return item, nil
}