//pintu masuk aplikasi & setup awal

// Package domain mendefinisikan struktur data inti untuk pricing-service.
package domain

import "errors"

// Sentinel errors khusus untuk layanan pricing
var (
	ErrZoneNotFound      = errors.New("zone not found for the given origin and destination")
	ErrRateNotFound      = errors.New("rate not found for the given zone and service type")
	ErrInvalidDimensions = errors.New("invalid weight or dimensions provided")
)

// Rate merepresentasikan data tarif dan batas maksimal dari database.
// Ini menjawab Revisi Tahap 1 (Penambahan MaxLength dkk).
type Rate struct {
	RateID      string  `db:"rate_id"`
	ZoneID      string  `db:"zone_id"`
	ServiceType string  `db:"service_type"`
	PricePerKG  float64 `db:"price_per_kg"`
	MinWeight   float64 `db:"min_weight"`
	MaxLength   float64 `db:"max_length"`
	MaxWidth    float64 `db:"max_width"`
	MaxHeight   float64 `db:"max_height"`
	MaxWeight   float64 `db:"max_weight"`
}

// Zone merepresentasikan rute pengiriman.
type Zone struct {
	ZoneID              string `db:"zone_id"`
	OriginCityCode      string `db:"origin_city_code"`
	DestinationCityCode string `db:"destination_city_code"`
}

// Surcharge merepresentasikan denda tambahan (seperti OVERSIZE).
type Surcharge struct {
	Type  string  `db:"type"`
	Value float64 `db:"value"`
}

// =====================================================================
// DATA TRANSFER OBJECTS (Bentuk Request & Response untuk HTTP)
// =====================================================================

// CalculatePriceRequest adalah data yang ditangkap dari URL query (Postman).
type CalculatePriceRequest struct {
	Origin      string  `form:"origin" binding:"required"`
	Destination string  `form:"destination" binding:"required"`
	Weight      float64 `form:"weight" binding:"required,gt=0"`
	ServiceType string  `form:"service_type" binding:"required"`
	Length      float64 `form:"length" binding:"required,gt=0"`
	Width       float64 `form:"width" binding:"required,gt=0"`
	Height      float64 `form:"height" binding:"required,gt=0"`
}

// CalculatePriceResponse adalah JSON final yang akan dikembalikan ke User.
type CalculatePriceResponse struct {
	ServiceType      string         `json:"service_type"`
	Price            float64        `json:"price"`
	VolumetricWeight float64        `json:"volumetric_weight"`
	ChargeableWeight float64        `json:"chargeable_weight"`
	Breakdown        PriceBreakdown `json:"breakdown"`
}

// PriceBreakdown berisi detail perhitungan (termasuk denda oversize Revisi Tahap 1).
type PriceBreakdown struct {
	BaseRate          float64 `json:"base_rate"`
	OversizeSurcharge float64 `json:"oversize_surcharge"`
}
