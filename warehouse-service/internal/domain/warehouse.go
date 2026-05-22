package domain

import "errors"

// Item adalah bentuk data barang di gudang kita
type Item struct {
	ID       string `json:"item_id"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Location string `json:"location"` // Contoh: Rak-A1
}

// StockRequest adalah bentuk data saat ada permintaan barang keluar
type StockRequest struct {
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

// Berbagai error yang mungkin terjadi di gudang
var (
	ErrItemNotFound = errors.New("barang tidak ditemukan di gudang")
	ErrOutOfStock   = errors.New("stok barang tidak mencukupi")
)