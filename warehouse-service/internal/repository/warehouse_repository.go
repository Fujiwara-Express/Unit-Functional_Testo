package repository

import (
	"context"
	"sync"

	"github.com/fujiwara-express/warehouse-service/internal/domain"
)

// WarehouseRepository adalah kontrak (aturan) untuk interaksi dengan "database" gudang
type WarehouseRepository interface {
	SaveItem(ctx context.Context, item domain.Item) error
	GetItemByID(ctx context.Context, id string) (domain.Item, error)
}

// memoryRepository adalah implementasi gudang bohong-bohongan (disimpan di RAM)
type memoryRepository struct {
	data map[string]domain.Item
	mu   sync.RWMutex // Mencegah error jika diakses banyak orang bersamaan
}

// NewMemoryRepository membuat "database" gudang baru yang masih kosong
func NewMemoryRepository() WarehouseRepository {
	return &memoryRepository{
		data: make(map[string]domain.Item),
	}
}

// SaveItem bertugas menyimpan barang baru atau mengupdate jumlah barang lama
func (r *memoryRepository) SaveItem(ctx context.Context, item domain.Item) error {
	r.mu.Lock()         // Kunci rak sebentar biar gak ada yang rebutan
	defer r.mu.Unlock() // Buka lagi kuncinya setelah selesai

	r.data[item.ID] = item
	return nil
}

// GetItemByID bertugas mencari barang di rak berdasarkan ID-nya
func (r *memoryRepository) GetItemByID(ctx context.Context, id string) (domain.Item, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, exists := r.data[id]
	if !exists {
		return domain.Item{}, domain.ErrItemNotFound
	}
	return item, nil
}