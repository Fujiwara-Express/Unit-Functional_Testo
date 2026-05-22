package calculators_test

import (
	"math"
	"testing"
	"time"

	"pgregory.net/rapid"

	"routing-service/internal/calculators"
	"routing-service/internal/models"
)

// ── Generators ───────────────────────────────────────────────────────────────

func genHubOrigin(t *rapid.T) models.HubOrigin {
	return models.HubOrigin{
		HubID: rapid.StringN(1, 5, 5).Draw(t, "hub_id"),
		Lat:   rapid.Float64Range(-90, 90).Draw(t, "hub_lat"),
		Lng:   rapid.Float64Range(-180, 180).Draw(t, "hub_lng"),
		Label: rapid.StringN(1, 10, 10).Draw(t, "hub_label"),
	}
}

func genDeliveryPoint(t *rapid.T, idx int) models.DeliveryPoint {
	return models.DeliveryPoint{
		DeliveryID:     rapid.StringN(1, 8, 8).Draw(t, "delivery_id"),
		TrackingNumber: rapid.StringN(1, 8, 8).Draw(t, "tracking"),
		RecipientName:  rapid.StringN(1, 10, 10).Draw(t, "recipient"),
		Address:        rapid.StringN(1, 20, 20).Draw(t, "address"),
		Lat:            rapid.Float64Range(-90, 90).Draw(t, "lat"),
		Lng:            rapid.Float64Range(-180, 180).Draw(t, "lng"),
	}
}

func genDeliveryPoints(t *rapid.T, minN, maxN int) []models.DeliveryPoint {
	n := rapid.IntRange(minN, maxN).Draw(t, "n_points")
	points := make([]models.DeliveryPoint, n)
	seen := map[string]bool{}
	for i := range points {
		for {
			p := genDeliveryPoint(t, i)
			if !seen[p.DeliveryID] {
				seen[p.DeliveryID] = true
				points[i] = p
				break
			}
		}
	}
	return points
}

var fixedStart = time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC)

// ── Property 3: Nearest neighbor route starts at hub ─────────────────────────

// Feature: routing-service, Property 3: Nearest neighbor route starts at hub
// Validates: Requirements 4.2
func TestProperty3_RouteStartsAtHub(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hub := genHubOrigin(t)
		points := genDeliveryPoints(t, 1, 10)

		result := calculators.CalculateCourierRoute("c1", points, hub, fixedStart)

		first := result.OptimizedRoute[0]

		// Find the delivery point actually closest to the hub
		closestID := points[0].DeliveryID
		closestDist := calculators.HaversineDistance(hub.Lat, hub.Lng, points[0].Lat, points[0].Lng)
		for _, p := range points {
			d := calculators.HaversineDistance(hub.Lat, hub.Lng, p.Lat, p.Lng)
			if d < closestDist {
				closestDist = d
				closestID = p.DeliveryID
			}
		}

		if first.DeliveryID != closestID {
			t.Fatalf("first stop is %s, want closest %s", first.DeliveryID, closestID)
		}

		expectedDist := calculators.HaversineDistance(hub.Lat, hub.Lng, first.Lat, first.Lng)
		if math.Abs(first.DistanceFromPrevKm-expectedDist) > 1e-9 {
			t.Fatalf("distance_from_prev_km %f != haversine %f", first.DistanceFromPrevKm, expectedDist)
		}
	})
}

// ── Property 4: Courier route covers all delivery points ─────────────────────

// Feature: routing-service, Property 4: Courier route covers all delivery points
// Validates: Requirements 4.2, 4.3
func TestProperty4_RouteCoversAllDeliveryPoints(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hub := genHubOrigin(t)
		points := genDeliveryPoints(t, 1, 10)

		result := calculators.CalculateCourierRoute("c1", points, hub, fixedStart)

		inputIDs := make(map[string]bool, len(points))
		for _, p := range points {
			inputIDs[p.DeliveryID] = true
		}
		outputIDs := make(map[string]bool, len(result.OptimizedRoute))
		for _, s := range result.OptimizedRoute {
			outputIDs[s.DeliveryID] = true
		}

		if len(inputIDs) != len(outputIDs) {
			t.Fatalf("input has %d delivery IDs, output has %d", len(inputIDs), len(outputIDs))
		}
		for id := range inputIDs {
			if !outputIDs[id] {
				t.Fatalf("delivery_id %s missing from output", id)
			}
		}
	})
}

// ── Property 5: Total distance consistency ───────────────────────────────────

// Feature: routing-service, Property 5: Total distance consistency
// Validates: Requirements 4.3
func TestProperty5_TotalDistanceConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hub := genHubOrigin(t)
		points := genDeliveryPoints(t, 1, 10)

		result := calculators.CalculateCourierRoute("c1", points, hub, fixedStart)

		sum := 0.0
		for _, s := range result.OptimizedRoute {
			sum += s.DistanceFromPrevKm
		}
		if math.Abs(result.TotalDistanceKm-sum) > 1e-9 {
			t.Fatalf("total_distance_km %f != sum of stops %f", result.TotalDistanceKm, sum)
		}
	})
}

// ── Property 7: Empty delivery points returns zero totals ────────────────────

// Feature: routing-service, Property 7: Empty delivery points returns zero totals
// Validates: Requirements 4.6
func TestProperty7_EmptyDeliveryPointsZeroTotals(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hub := genHubOrigin(t)

		result := calculators.CalculateCourierRoute("c1", nil, hub, fixedStart)

		if len(result.OptimizedRoute) != 0 {
			t.Fatalf("expected empty route, got %d stops", len(result.OptimizedRoute))
		}
		if result.TotalStops != 0 {
			t.Fatalf("expected TotalStops=0, got %d", result.TotalStops)
		}
		if result.TotalDistanceKm != 0 {
			t.Fatalf("expected TotalDistanceKm=0, got %f", result.TotalDistanceKm)
		}
		if result.EstimatedTotalDurationMinutes != 0 {
			t.Fatalf("expected EstimatedTotalDurationMinutes=0, got %f", result.EstimatedTotalDurationMinutes)
		}
	})
}
