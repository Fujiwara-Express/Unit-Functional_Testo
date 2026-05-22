package models

// RouteNode represents a hub in the routing graph.
type RouteNode struct {
	NodeID    string  `json:"node_id" db:"node_id"`
	HubID     string  `json:"hub_id" db:"hub_id"`
	CityCode  string  `json:"city_code" db:"city_code"`
	Latitude  float64 `json:"latitude" db:"latitude"`
	Longitude float64 `json:"longitude" db:"longitude"`
}

// RouteEdge represents a directed connection between two hubs.
type RouteEdge struct {
	EdgeID          string  `json:"edge_id" db:"edge_id"`
	FromNodeID      string  `json:"from_node_id" db:"from_node_id"`
	ToNodeID        string  `json:"to_node_id" db:"to_node_id"`
	DistanceKm      float64 `json:"distance_km" db:"distance_km"`
	AvgTransitHours float64 `json:"avg_transit_hours" db:"avg_transit_hours"`
	TransportMode   string  `json:"transport_mode" db:"transport_mode"` // DARAT | UDARA | LAUT
	IsActive        bool    `json:"is_active" db:"is_active"`
}

// Graph is the in-memory representation used by the Dijkstra calculator.
type Graph struct {
	Nodes map[string]*RouteNode // keyed by hub_id
	Edges []RouteEdge
}

// InterHubRouteStop is one hub in the inter-hub route sequence.
type InterHubRouteStop struct {
	HubID    string `json:"hub_id"`
	City     string `json:"city"`
	Sequence int    `json:"sequence"`
}

// InterHubRoute is the result of an inter-hub route calculation.
type InterHubRoute struct {
	Origin                string              `json:"origin"`
	Destination           string              `json:"destination"`
	Route                 []InterHubRouteStop `json:"route"`
	TotalDistanceKm       float64             `json:"total_distance_km"`
	EstimatedTransitHours float64             `json:"estimated_transit_hours"`
}

// DeliveryPoint is a single package delivery destination for a courier.
type DeliveryPoint struct {
	DeliveryID     string  `json:"delivery_id"`
	TrackingNumber string  `json:"tracking_number"`
	RecipientName  string  `json:"recipient_name"`
	Address        string  `json:"address"`
	Lat            float64 `json:"lat"`
	Lng            float64 `json:"lng"`
}

// HubOrigin is the courier's starting hub.
type HubOrigin struct {
	HubID string  `json:"hub_id"`
	Lat   float64 `json:"lat"`
	Lng   float64 `json:"lng"`
	Label string  `json:"label"`
}

// CourierRouteStop is one delivery stop in the optimized courier route.
type CourierRouteStop struct {
	Sequence           int     `json:"sequence"`
	TrackingNumber     string  `json:"tracking_number"`
	DeliveryID         string  `json:"delivery_id"`
	RecipientName      string  `json:"recipient_name"`
	Address            string  `json:"address"`
	Lat                float64 `json:"lat"`
	Lng                float64 `json:"lng"`
	EstimatedArrival   string  `json:"estimated_arrival"`
	DistanceFromPrevKm float64 `json:"distance_from_prev_km"`
}

// CourierRoute is the full optimized daily delivery route for a courier.
type CourierRoute struct {
	CourierID                     string             `json:"courier_id"`
	HubID                         string             `json:"hub_id"`
	Origin                        HubOrigin          `json:"origin"`
	OptimizedRoute                []CourierRouteStop `json:"optimized_route"`
	TotalStops                    int                `json:"total_stops"`
	TotalDistanceKm               float64            `json:"total_distance_km"`
	EstimatedTotalDurationMinutes float64            `json:"estimated_total_duration_minutes"`
}

// --- DTOs for API request bodies ---

// CreateNodeInput is the request body for POST /routing/nodes.
type CreateNodeInput struct {
	HubID     string  `json:"hub_id" binding:"required"`
	CityCode  string  `json:"city_code" binding:"required"`
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
}

// CreateEdgeInput is the request body for POST /routing/edges.
type CreateEdgeInput struct {
	FromNodeID      string  `json:"from_node_id" binding:"required"`
	ToNodeID        string  `json:"to_node_id" binding:"required"`
	DistanceKm      float64 `json:"distance_km" binding:"required,gt=0"`
	AvgTransitHours float64 `json:"avg_transit_hours" binding:"required,gt=0"`
	TransportMode   string  `json:"transport_mode" binding:"required,oneof=DARAT UDARA LAUT"`
	IsActive        *bool   `json:"is_active"`
}

// UpdateEdgeInput is the request body for PATCH /routing/edges/:edge_id.
type UpdateEdgeInput struct {
	DistanceKm      *float64 `json:"distance_km" binding:"omitempty,gt=0"`
	AvgTransitHours *float64 `json:"avg_transit_hours" binding:"omitempty,gt=0"`
	TransportMode   *string  `json:"transport_mode" binding:"omitempty,oneof=DARAT UDARA LAUT"`
	IsActive        *bool    `json:"is_active"`
}
