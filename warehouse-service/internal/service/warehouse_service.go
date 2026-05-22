package service

import (
	"context"

	"github.com/fujiwara-express/warehouse-service/internal/domain"
	"github.com/fujiwara-express/warehouse-service/internal/repository"
)

// WarehouseService adalah tugas-tugas yang bisa dilakukan oleh Kepala Gudang
type WarehouseService interface {
	ReceiveItem(ctx context.Context, item domain.Item) error                   // Barang Masuk
	DispatchItem(ctx context.Context, id string, quantity int) error           // Barang Keluar
	CheckStock(ctx context.Context, id string) (domain.Item, error)            // Cek Stok
}

type warehouseService struct {
	repo repository.WarehouseRepository
}

// NewWarehouseService menempatkan Kepala Gudang untuk menjaga Rak (Repository)
func NewWarehouseService(repo repository.WarehouseRepository) WarehouseService {
	return &warehouseService{repo: repo}
}

// ReceiveItem: Logika saat ada barang masuk
func (s *warehouseService) ReceiveItem(ctx context.Context, newItem domain.Item) error {
	// 1. Cek apakah barang ini sudah ada di rak sebelumnya
	existingItem, err := s.repo.GetItemByID(ctx, newItem.ID)
	
	if err == nil {
		// Jika sudah ada, cukup TANDA TAMBAH jumlah stoknya
		existingItem.Quantity += newItem.Quantity
		
		// Update lokasi jika barang dipindah ke rak baru
		if newItem.Location != "" {
			existingItem.Location = newItem.Location
		}
		return s.repo.SaveItem(ctx, existingItem)
	}

	// 2. Jika ini barang baru (belum ada di rak), langsung simpan saja
	return s.repo.SaveItem(ctx, newItem)
}

// DispatchItem: Logika saat ada barang mau dikirim/dikeluarkan
func (s *warehouseService) DispatchItem(ctx context.Context, id string, quantity int) error {
	// 1. Cari barangnya ada atau tidak
	item, err := s.repo.GetItemByID(ctx, id)
	if err != nil {
		return err // Error: Barang tidak ditemukan
	}

	// 2. Cek apakah stoknya cukup untuk dikeluarkan
	if item.Quantity < quantity {
		return domain.ErrOutOfStock // Error: Stok kurang!
	}

	// 3. Kurangi stok dan simpan kembali ke rak
	item.Quantity -= quantity
	return s.repo.SaveItem(ctx, item)
}

// CheckStock: Logika sekadar melihat data barang
func (s *warehouseService) CheckStock(ctx context.Context, id string) (domain.Item, error) {
	return s.repo.GetItemByID(ctx, id)
}