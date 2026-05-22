// Package service berisi logika bisnis (otak) dari aplikasi pricing.
package service

import (
	"context"
	"math"

	"github.com/fujiwara-express/pricing-service/internal/domain"
	"github.com/fujiwara-express/pricing-service/internal/repository"
)

// PricingService adalah kontrak layanan untuk diakses oleh Handler/Controller nanti.
type PricingService interface {
	CalculatePrice(ctx context.Context, req domain.CalculatePriceRequest) (domain.CalculatePriceResponse, error)
}

// pricingService adalah implementasi dari antarmuka PricingService.
type pricingService struct {
	repo repository.PricingRepository
}

// NewPricingService membuat mesin hitung baru dengan menyambungkan repository (database/mock).
func NewPricingService(repo repository.PricingRepository) PricingService {
	return &pricingService{repo: repo}
}

// CalculatePrice adalah fungsi utama yang memecahkan Revisi Tahap 1 (Logika Volume & Oversize).
func (s *pricingService) CalculatePrice(ctx context.Context, req domain.CalculatePriceRequest) (domain.CalculatePriceResponse, error) {
	// 1. Cari data Zona berdasarkan asal dan tujuan
	zone, err := s.repo.GetZone(ctx, req.Origin, req.Destination)
	if err != nil {
		return domain.CalculatePriceResponse{}, domain.ErrZoneNotFound
	}

	// 2. Cari data Tarif Dasar (termasuk batas dimensi dari database)
	rate, err := s.repo.GetRate(ctx, zone.ZoneID, req.ServiceType)
	if err != nil {
		return domain.CalculatePriceResponse{}, domain.ErrRateNotFound
	}

	// 3. Hitung Berat Volumetrik (P x L x T / 6000)
	volumetricWeight := (req.Length * req.Width * req.Height) / 6000

	// 4. Tentukan Chargeable Weight (Berat Penagihan)
	// Ambil yang paling besar antara berat asli vs volumetrik
	chargeableWeight := math.Max(req.Weight, volumetricWeight)
	// Pastikan tidak di bawah minimum berat layanan
	chargeableWeight = math.Max(chargeableWeight, rate.MinWeight)

	// 5. Hitung harga dasar (Base Rate)
	baseRate := chargeableWeight * rate.PricePerKG
	oversizeSurcharge := 0.0

	// 6. PENGECEKAN REVISI TAHAP 1: Validasi Dimensi Oversize
	// Jika panjang, lebar, tinggi, atau berat melebihi batas maksimal layanan...
	if req.Length > rate.MaxLength || req.Width > rate.MaxWidth || req.Height > rate.MaxHeight || req.Weight > rate.MaxWeight {

		// ...Maka ambil nilai denda OVERSIZE dari database
		surcharge, err := s.repo.GetSurcharge(ctx, "OVERSIZE")
		if err == nil {
			oversizeSurcharge = surcharge.Value
		}

	}

	// 7. Hitung Total Akhir
	totalPrice := baseRate + oversizeSurcharge

	// 8. Kembalikan Response sesuai format JSON yang diinginkan
	return domain.CalculatePriceResponse{
		ServiceType:      req.ServiceType,
		Price:            totalPrice,
		VolumetricWeight: volumetricWeight,
		ChargeableWeight: chargeableWeight,
		Breakdown: domain.PriceBreakdown{
			BaseRate:          baseRate,
			OversizeSurcharge: oversizeSurcharge,
		},
	}, nil
}
