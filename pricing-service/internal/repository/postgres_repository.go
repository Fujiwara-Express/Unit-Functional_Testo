package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fujiwara-express/pricing-service/internal/domain"
)

// postgresRepository adalah implementasi database asli
type postgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository membuat koneksi repositori ke PostgreSQL
func NewPostgresRepository(db *sql.DB) PricingRepository {
	return &postgresRepository{db: db}
}

// GetZone mencari zona dari tabel 'zones'
func (r *postgresRepository) GetZone(ctx context.Context, origin, destination string) (domain.Zone, error) {
	var zone domain.Zone
	query := `SELECT zone_id FROM zones WHERE origin = $1 AND destination = $2`
	
	err := r.db.QueryRowContext(ctx, query, origin, destination).Scan(&zone.ZoneID)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Zone{}, domain.ErrZoneNotFound
		}
		return domain.Zone{}, err
	}
	return zone, nil
}

// GetRate mencari tarif dasar dari tabel 'rates'
func (r *postgresRepository) GetRate(ctx context.Context, zoneID, serviceType string) (domain.Rate, error) {
	var rate domain.Rate
	query := `
		SELECT price_per_kg, min_weight, max_weight, max_length, max_width, max_height 
		FROM rates 
		WHERE zone_id = $1 AND service_type = $2`
	
	err := r.db.QueryRowContext(ctx, query, zoneID, serviceType).Scan(
		&rate.PricePerKG,
		&rate.MinWeight,
		&rate.MaxWeight,
		&rate.MaxLength,
		&rate.MaxWidth,
		&rate.MaxHeight,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Rate{}, domain.ErrRateNotFound
		}
		return domain.Rate{}, err
	}
	return rate, nil
}

// GetSurcharge mencari nilai denda dari tabel 'surcharges'
func (r *postgresRepository) GetSurcharge(ctx context.Context, sType string) (domain.Surcharge, error) {
	var surcharge domain.Surcharge
	surcharge.Type = sType
	
	query := `SELECT surcharge_value FROM surcharges WHERE surcharge_type = $1`
	err := r.db.QueryRowContext(ctx, query, sType).Scan(&surcharge.Value)
	
	if err != nil {
		if err == sql.ErrNoRows {
			// Jika denda tidak ada, kita kembalikan error agar service tahu
			return domain.Surcharge{}, errors.New("surcharge not found")
		}
		return domain.Surcharge{}, err
	}
	return surcharge, nil
}