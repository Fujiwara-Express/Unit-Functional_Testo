package calculators

import (
	"container/heap"
	"fmt"
	"math"
	"os"
	"strconv"

	"routing-service/internal/models"
)

// defaultAlpha and defaultBeta are the Dijkstra weight coefficients.
// They can be overridden via ROUTE_ALPHA / ROUTE_BETA environment variables.
var (
	defaultAlpha = envFloat("ROUTE_ALPHA", 1.0)
	defaultBeta  = envFloat("ROUTE_BETA", 1.0)
)

// EdgeWeight computes the combined Dijkstra weight for an edge.
// weight = alpha * distance_km + beta * avg_transit_hours
func EdgeWeight(edge models.RouteEdge, alpha, beta float64) float64 {
	return alpha*edge.DistanceKm + beta*edge.AvgTransitHours
}

// Dijkstra runs Dijkstra's algorithm on the active-edge graph and returns the
// optimal InterHubRoute from origin to destination, or an error if no path exists.
func Dijkstra(graph *models.Graph, origin, destination string) (*models.InterHubRoute, error) {
	return DijkstraWeighted(graph, origin, destination, defaultAlpha, defaultBeta)
}

// DijkstraWeighted is the same as Dijkstra but with explicit alpha/beta coefficients.
func DijkstraWeighted(graph *models.Graph, origin, destination string, alpha, beta float64) (*models.InterHubRoute, error) {
	if _, ok := graph.Nodes[origin]; !ok {
		return nil, fmt.Errorf("origin hub '%s' not found", origin)
	}
	if _, ok := graph.Nodes[destination]; !ok {
		return nil, fmt.Errorf("destination hub '%s' not found", destination)
	}

	// Build adjacency list: hub_id → [{toHubID, edge}]
	type adjEntry struct {
		toHubID string
		edge    models.RouteEdge
	}
	adj := make(map[string][]adjEntry, len(graph.Nodes))
	for hubID := range graph.Nodes {
		adj[hubID] = nil
	}

	// Build a nodeID → hubID lookup
	nodeIDToHubID := make(map[string]string, len(graph.Nodes))
	for hubID, node := range graph.Nodes {
		nodeIDToHubID[node.NodeID] = hubID
	}

	for _, edge := range graph.Edges {
		if !edge.IsActive {
			continue
		}
		fromHub, ok1 := nodeIDToHubID[edge.FromNodeID]
		toHub, ok2 := nodeIDToHubID[edge.ToNodeID]
		if !ok1 || !ok2 {
			continue
		}
		adj[fromHub] = append(adj[fromHub], adjEntry{toHubID: toHub, edge: edge})
	}

	// Dijkstra with a binary min-heap
	dist := make(map[string]float64, len(graph.Nodes))
	prev := make(map[string]string, len(graph.Nodes))
	prevEdge := make(map[string]*models.RouteEdge, len(graph.Nodes))

	for hubID := range graph.Nodes {
		dist[hubID] = math.Inf(1)
	}
	dist[origin] = 0

	pq := &priorityQueue{}
	heap.Push(pq, &pqItem{hubID: origin, cost: 0})

	for pq.Len() > 0 {
		item := heap.Pop(pq).(*pqItem)
		u, cost := item.hubID, item.cost

		if cost > dist[u] {
			continue // stale entry
		}
		if u == destination {
			break
		}

		for _, entry := range adj[u] {
			w := EdgeWeight(entry.edge, alpha, beta)
			newCost := cost + w
			if newCost < dist[entry.toHubID] {
				dist[entry.toHubID] = newCost
				prev[entry.toHubID] = u
				e := entry.edge
				prevEdge[entry.toHubID] = &e
				heap.Push(pq, &pqItem{hubID: entry.toHubID, cost: newCost})
			}
		}
	}

	if math.IsInf(dist[destination], 1) {
		return nil, nil // no path — caller returns 404
	}

	// Reconstruct path
	var hubPath []string
	for cur := destination; cur != ""; cur = prev[cur] {
		hubPath = append([]string{cur}, hubPath...)
		if cur == origin {
			break
		}
	}

	// Accumulate totals
	var totalDistance, totalTransit float64
	for i := 1; i < len(hubPath); i++ {
		e := prevEdge[hubPath[i]]
		totalDistance += e.DistanceKm
		totalTransit += e.AvgTransitHours
	}

	route := make([]models.InterHubRouteStop, len(hubPath))
	for i, hubID := range hubPath {
		node := graph.Nodes[hubID]
		route[i] = models.InterHubRouteStop{
			HubID:    hubID,
			City:     node.CityCode,
			Sequence: i + 1,
		}
	}

	return &models.InterHubRoute{
		Origin:                origin,
		Destination:           destination,
		Route:                 route,
		TotalDistanceKm:       totalDistance,
		EstimatedTransitHours: totalTransit,
	}, nil
}

// ── priority queue (min-heap) ────────────────────────────────────────────────

type pqItem struct {
	hubID string
	cost  float64
	index int
}

type priorityQueue []*pqItem

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].cost < pq[j].cost }
func (pq priorityQueue) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i]; pq[i].index = i; pq[j].index = j }
func (pq *priorityQueue) Push(x interface{}) {
	item := x.(*pqItem)
	item.index = len(*pq)
	*pq = append(*pq, item)
}
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*pq = old[:n-1]
	return item
}

// ── helpers ──────────────────────────────────────────────────────────────────

func envFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}
