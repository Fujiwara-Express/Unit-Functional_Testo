package calculators

import (
	"fmt"
	"time"

	"routing-service/internal/models"
)

const avgSpeedKmh = 30.0 // assumed average courier speed

// CalculateCourierRoute computes the optimized daily delivery route for a courier
// using the nearest neighbor heuristic (TSP approximation).
// The route always starts from the hub origin.
func CalculateCourierRoute(
	courierID string,
	deliveryPoints []models.DeliveryPoint,
	hub models.HubOrigin,
	startTime time.Time,
) *models.CourierRoute {
	// Requirement 4.6: empty delivery points → zero totals
	if len(deliveryPoints) == 0 {
		return &models.CourierRoute{
			CourierID:                     courierID,
			HubID:                         hub.HubID,
			Origin:                        hub,
			OptimizedRoute:                []models.CourierRouteStop{},
			TotalStops:                    0,
			TotalDistanceKm:               0,
			EstimatedTotalDurationMinutes: 0,
		}
	}

	unvisited := make([]models.DeliveryPoint, len(deliveryPoints))
	copy(unvisited, deliveryPoints)

	var optimizedRoute []models.CourierRouteStop
	curLat, curLng := hub.Lat, hub.Lng
	var totalDistance, elapsedMinutes float64

	for seq := 1; len(unvisited) > 0; seq++ {
		// Find nearest unvisited point from current position
		nearestIdx := 0
		nearestDist := HaversineDistance(curLat, curLng, unvisited[0].Lat, unvisited[0].Lng)
		for i := 1; i < len(unvisited); i++ {
			d := HaversineDistance(curLat, curLng, unvisited[i].Lat, unvisited[i].Lng)
			if d < nearestDist {
				nearestDist = d
				nearestIdx = i
			}
		}

		point := unvisited[nearestIdx]
		unvisited = append(unvisited[:nearestIdx], unvisited[nearestIdx+1:]...)

		travelMinutes := (nearestDist / avgSpeedKmh) * 60
		elapsedMinutes += travelMinutes
		totalDistance += nearestDist

		arrival := startTime.Add(time.Duration(elapsedMinutes * float64(time.Minute)))

		optimizedRoute = append(optimizedRoute, models.CourierRouteStop{
			Sequence:           seq,
			TrackingNumber:     point.TrackingNumber,
			DeliveryID:         point.DeliveryID,
			RecipientName:      point.RecipientName,
			Address:            point.Address,
			Lat:                point.Lat,
			Lng:                point.Lng,
			EstimatedArrival:   arrival.Format(time.RFC3339),
			DistanceFromPrevKm: nearestDist,
		})

		curLat, curLng = point.Lat, point.Lng
	}

	return &models.CourierRoute{
		CourierID:                     courierID,
		HubID:                         hub.HubID,
		Origin:                        hub,
		OptimizedRoute:                optimizedRoute,
		TotalStops:                    len(optimizedRoute),
		TotalDistanceKm:               totalDistance,
		EstimatedTotalDurationMinutes: elapsedMinutes,
	}
}

// DefaultStartTime returns 08:00 today in local time.
func DefaultStartTime() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, now.Location())
}

// TodayDate returns today's date as YYYY-MM-DD.
func TodayDate() string {
	return fmt.Sprintf("%s", time.Now().Format("2006-01-02"))
}
