// Package service berisi unit test untuk logika bisnis pricing.
package service

import (
    "context"
    "testing"

    "github.com/fujiwara-express/pricing-service/internal/domain"
    "github.com/fujiwara-express/pricing-service/internal/repository/mocks"

    "go.uber.org/mock/gomock"
)

func TestCalculatePrice_Success_WithOversizeSurcharge(t *testing.T) {
	// 1. Siapkan Controller untuk Gomock
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 2. Buat Database Bohongan (Mock Repository)
	mockRepo := mocks.NewMockPricingRepository(ctrl)

	// 3. Masukkan database bohongan ke dalam Mesin kita
	svc := NewPricingService(mockRepo)

	// 4. Skenario: User mengirim paket SANGAT PANJANG (150cm) dengan berat 10kg
	// (Revisi Tahap 1: Ini harusnya kena denda Oversize)
	req := domain.CalculatePriceRequest{
		Origin:      "CGK",
		Destination: "BDO",
		ServiceType: "REG",
		Weight:      10,
		Length:      150, // Melebihi MaxLength (100)
		Width:       50,
		Height:      50,
	}

	ctx := context.Background()

	// 5. Atur "Janji" / Skenario Database Bohongan (MOCKING)
	// - Saat mesin mencari Zona, kembalikan ZoneID "Z1"
	mockRepo.EXPECT().
		GetZone(ctx, "CGK", "BDO").
		Return(domain.Zone{ZoneID: "Z1"}, nil)

	// - Saat mesin mencari Tarif, kembalikan harga 10.000/kg dengan MaxLength 100
	mockRepo.EXPECT().
		GetRate(ctx, "Z1", "REG").
		Return(domain.Rate{
			PricePerKG: 10000,
			MinWeight:  1,
			MaxLength:  100, // <-- Perhatikan: Paket user 150cm, jadi pasti melanggar ini
			MaxWidth:   100,
			MaxHeight:  100,
			MaxWeight:  50,
		}, nil)

	// - KARENA melanggar MaxLength, mesin PASTI akan memanggil GetSurcharge "OVERSIZE".
	// - Kita suruh database bohongan mengembalikan denda sebesar 50.000
	mockRepo.EXPECT().
		GetSurcharge(ctx, "OVERSIZE").
		Return(domain.Surcharge{Type: "OVERSIZE", Value: 50000}, nil)

	// 6. JALANKAN MESIN!
	resp, err := svc.CalculatePrice(ctx, req)

	// 7. VALIDASI HASILNYA
	if err != nil {
		t.Fatalf("Tidak diharapkan ada error, tapi dapat: %v", err)
	}

	// Hitung manual ekspektasinya:
	// Volumetrik: (150 * 50 * 50) / 6000 = 62.5 kg
	// Chargeable Weight (diambil yang terbesar antara 10kg vs 62.5kg) = 62.5 kg
	// Harga Dasar: 62.5 * 10000 = 625.000
	// Denda Oversize: 50.000
	// TOTAL HARGA: 675.000

	if resp.ChargeableWeight != 62.5 {
		t.Errorf("Ekspektasi berat penagihan 62.5, tapi dapat: %v", resp.ChargeableWeight)
	}

	if resp.Breakdown.OversizeSurcharge != 50000 {
		t.Errorf("Ekspektasi denda 50000, tapi dapat: %v", resp.Breakdown.OversizeSurcharge)
	}

	if resp.Price != 675000 {
		t.Errorf("Ekspektasi harga total 675000, tapi dapat: %v", resp.Price)
	}
}