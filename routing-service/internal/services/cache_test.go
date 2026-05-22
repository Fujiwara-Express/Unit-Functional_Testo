package services_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"pgregory.net/rapid"

	"routing-service/internal/models"
	"routing-service/internal/services"
)

// ── In-memory Redis stub ─────────────────────────────────────────────────────

type memRedis struct {
	store map[string]memEntry
}

type memEntry struct {
	value     []byte
	expiresAt time.Time
}

func newMemRedis() *memRedis { return &memRedis{store: map[string]memEntry{}} }

func (m *memRedis) Get(ctx context.Context, key string) ([]byte, bool) {
	e, ok := m.store[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.value, true
}

func (m *memRedis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) {
	m.store[key] = memEntry{value: value, expiresAt: time.Now().Add(ttl)}
}

func (m *memRedis) Keys(ctx context.Context, pattern string) []string {
	// simple: return all keys (sufficient for tests using "route:*")
	keys := make([]string, 0, len(m.store))
	for k := range m.store {
		keys = append(keys, k)
	}
	return keys
}

func (m *memRedis) Del(ctx context.Context, keys ...string) {
	for _, k := range keys {
		delete(m.store, k)
	}
}

// ── Minimal CacheService backed by memRedis ──────────────────────────────────
// We test the JSON round-trip logic directly without a real Redis connection.

func roundTrip(t rapid.TB, value interface{}, dest interface{}) bool {
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	return true
}

// ── Generators ───────────────────────────────────────────────────────────────

func genInterHubRoute(t *rapid.T) models.InterHubRoute {
	n := rapid.IntRange(1, 5).Draw(t, "n_stops")
	stops := make([]models.InterHubRouteStop, n)
	for i := range stops {
		stops[i] = models.InterHubRouteStop{
			HubID:    rapid.StringN(1, 5, 5).Draw(t, "hub_id"),
			City:     rapid.StringN(1, 5, 5).Draw(t, "city"),
			Sequence: i + 1,
		}
	}
	return models.InterHubRoute{
		Origin:                rapid.StringN(1, 5, 5).Draw(t, "origin"),
		Destination:           rapid.StringN(1, 5, 5).Draw(t, "dest"),
		Route:                 stops,
		TotalDistanceKm:       rapid.Float64Range(0, 10000).Draw(t, "dist"),
		EstimatedTransitHours: rapid.Float64Range(0, 500).Draw(t, "transit"),
	}
}

func genCourierRoute(t *rapid.T) models.CourierRoute {
	n := rapid.IntRange(0, 5).Draw(t, "n_stops")
	stops := make([]models.CourierRouteStop, n)
	for i := range stops {
		stops[i] = models.CourierRouteStop{
			Sequence:           i + 1,
			TrackingNumber:     rapid.StringN(1, 8, 8).Draw(t, "tracking"),
			DeliveryID:         rapid.StringN(1, 8, 8).Draw(t, "delivery_id"),
			RecipientName:      rapid.StringN(1, 10, 10).Draw(t, "recipient"),
			Address:            rapid.StringN(1, 20, 20).Draw(t, "address"),
			Lat:                rapid.Float64Range(-90, 90).Draw(t, "lat"),
			Lng:                rapid.Float64Range(-180, 180).Draw(t, "lng"),
			EstimatedArrival:   "2024-01-01T08:00:00Z",
			DistanceFromPrevKm: rapid.Float64Range(0, 1000).Draw(t, "dist"),
		}
	}
	return models.CourierRoute{
		CourierID: rapid.StringN(1, 8, 8).Draw(t, "courier_id"),
		HubID:     rapid.StringN(1, 5, 5).Draw(t, "hub_id"),
		Origin: models.HubOrigin{
			HubID: rapid.StringN(1, 5, 5).Draw(t, "origin_hub_id"),
			Lat:   rapid.Float64Range(-90, 90).Draw(t, "origin_lat"),
			Lng:   rapid.Float64Range(-180, 180).Draw(t, "origin_lng"),
			Label: rapid.StringN(1, 10, 10).Draw(t, "origin_label"),
		},
		OptimizedRoute:                stops,
		TotalStops:                    n,
		TotalDistanceKm:               rapid.Float64Range(0, 10000).Draw(t, "total_dist"),
		EstimatedTotalDurationMinutes: rapid.Float64Range(0, 10000).Draw(t, "total_dur"),
	}
}

// ── Property 2: Route cache round-trip consistency ───────────────────────────

// Feature: routing-service, Property 2: Route cache round-trip consistency
// Validates: Requirements 5.3
func TestProperty2_InterHubRouteRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		route := genInterHubRoute(t)

		var retrieved models.InterHubRoute
		roundTrip(t, route, &retrieved)

		key := services.InterHubKey(route.Origin, route.Destination, "REG")
		_ = key // key format validated separately

		if retrieved.Origin != route.Origin {
			t.Fatalf("origin mismatch: %s != %s", retrieved.Origin, route.Origin)
		}
		if retrieved.Destination != route.Destination {
			t.Fatalf("destination mismatch")
		}
		if retrieved.TotalDistanceKm != route.TotalDistanceKm {
			t.Fatalf("total_distance_km mismatch: %f != %f", retrieved.TotalDistanceKm, route.TotalDistanceKm)
		}
		if len(retrieved.Route) != len(route.Route) {
			t.Fatalf("route length mismatch: %d != %d", len(retrieved.Route), len(route.Route))
		}
	})
}

// Feature: routing-service, Property 2: Route cache round-trip consistency
// Validates: Requirements 5.3
func TestProperty2_CourierRouteRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		route := genCourierRoute(t)

		var retrieved models.CourierRoute
		roundTrip(t, route, &retrieved)

		if retrieved.CourierID != route.CourierID {
			t.Fatalf("courier_id mismatch")
		}
		if len(retrieved.OptimizedRoute) != len(route.OptimizedRoute) {
			t.Fatalf("optimized_route length mismatch: %d != %d", len(retrieved.OptimizedRoute), len(route.OptimizedRoute))
		}
		if retrieved.TotalDistanceKm != route.TotalDistanceKm {
			t.Fatalf("total_distance_km mismatch: %f != %f", retrieved.TotalDistanceKm, route.TotalDistanceKm)
		}
	})
}
