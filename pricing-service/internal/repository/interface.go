// Package repository mendefinisikan antarmuka (kontrak) untuk akses data.
package repository

import (
    "context"
    "github.com/fujiwara-express/pricing-service/internal/domain"
)

// PricingRepository mendefinisikan operasi untuk mengambil data tarif, zona, dan denda.
// Karena kita menggunakan interface, kita bisa dengan mudah membuat "Mock" 
// untuk Unit Test (Tahap 2) tanpa harus menyalakan database asli.
type PricingRepository interface {
	// GetZone mencari data Zona berdasarkan asal dan tujuan
	GetZone(ctx context.Context, originCode string, destCode string) (domain.Zone, error)

	// GetRate mencari data Tarif berdasarkan ID Zona dan Jenis Layanan.
	// Data yang dikembalikan sudah mencakup batas dimensi (Revisi Tahap 1).
	GetRate(ctx context.Context, zoneID string, serviceType string) (domain.Rate, error)

	// GetSurcharge mencari besaran denda berdasarkan tipenya (contoh: "OVERSIZE")
	GetSurcharge(ctx context.Context, surchargeType string) (domain.Surcharge, error)
}