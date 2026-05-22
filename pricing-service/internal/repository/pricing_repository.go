package repository

import (
	"context"
	"errors"
	"github.com/fujiwara-express/pricing-service/internal/domain"
)

type memoryRepository struct{}

func NewMemoryRepository() PricingRepository {
	return &memoryRepository{}
}

func (r *memoryRepository) GetZone(ctx context.Context, origin, destination string) (domain.Zone, error) {
	// Data simulasi: CGK ke BDO ada di zona Z1
	if origin == "CGK" && destination == "BDO" {
		return domain.Zone{ZoneID: "Z1"}, nil
	}
	return domain.Zone{}, errors.New("zona tidak ditemukan")
}

func (r *memoryRepository) GetRate(ctx context.Context, zoneID, serviceType string) (domain.Rate, error) {
	// Data tarif simulasi
	return domain.Rate{
		PricePerKG: 10000,
		MinWeight:  1,
		MaxWeight:  50,
		MaxLength:  100,
		MaxWidth:   100,
		MaxHeight:  100,
	}, nil
}

func (r *memoryRepository) GetSurcharge(ctx context.Context, sType string) (domain.Surcharge, error) {
	// Data denda simulasi
	if sType == "OVERSIZE" {
		return domain.Surcharge{Type: "OVERSIZE", Value: 50000}, nil
	}
	return domain.Surcharge{}, nil
}