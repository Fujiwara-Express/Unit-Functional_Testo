package model

// Ini adalah model Rate yang sudah direvisi
type Rate struct {
	RateID        string  `json:"rate_id"`
	ZoneID        string  `json:"zone_id"`
	ServiceType   string  `json:"service_type"`
	PricePerKG    float64 `json:"price_per_kg"`
	MinWeight     float64 `json:"min_weight"`
	MaxLength     float64 `json:"max_length"`   // <-- Tambahan revisi
	MaxWidth      float64 `json:"max_width"`    // <-- Tambahan revisi
	MaxHeight     float64 `json:"max_height"`   // <-- Tambahan revisi
	MaxWeight     float64 `json:"max_weight"`   // <-- Tambahan revisi
	EstimatedDays int     `json:"estimated_days"`
}

// Model Zone (Tetap sama seperti rancangan awal)
type Zone struct {
	ZoneID              string `json:"zone_id"`
	OriginCityCode      string `json:"origin_city_code"`
	DestinationCityCode string `json:"destination_city_code"`
	ZoneCategory        string `json:"zone_category"`
}

// Model Surcharge (Tetap sama seperti rancangan awal)
type Surcharge struct {
	Type  string  `json:"type"`  // contoh: "FUEL", "OVERSIZE"
	Value float64 `json:"value"`
}