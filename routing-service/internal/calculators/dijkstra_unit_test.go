package calculators_test

import (
	"testing"

	"routing-service/internal/calculators"
	"routing-service/internal/models"
)

// buildSimpleGraph builds a small hand-crafted graph for deterministic tests.
//
//	A --10--> B --5--> C
//	A --100-> C          (direct but expensive)
func buildSimpleGraph() *models.Graph {
	nodes := map[string]*models.RouteNode{
		"A": {NodeID: "n-A", HubID: "A", CityCode: "CA"},
		"B": {NodeID: "n-B", HubID: "B", CityCode: "CB"},
		"C": {NodeID: "n-C", HubID: "C", CityCode: "CC"},
	}
	edges := []models.RouteEdge{
		{EdgeID: "e1", FromNodeID: "n-A", ToNodeID: "n-B", DistanceKm: 10, AvgTransitHours: 1, TransportMode: "DARAT", IsActive: true},
		{EdgeID: "e2", FromNodeID: "n-B", ToNodeID: "n-C", DistanceKm: 5, AvgTransitHours: 0.5, TransportMode: "DARAT", IsActive: true},
		{EdgeID: "e3", FromNodeID: "n-A", ToNodeID: "n-C", DistanceKm: 100, AvgTransitHours: 10, TransportMode: "DARAT", IsActive: true},
	}
	return &models.Graph{Nodes: nodes, Edges: edges}
}

func TestDijkstra_ShortestPath(t *testing.T) {
	g := buildSimpleGraph()
	result, err := calculators.Dijkstra(g, "A", "C")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected a route, got nil")
	}
	// Optimal: A→B→C (cost 15+1.5=16.5) vs A→C direct (cost 110)
	if len(result.Route) != 3 {
		t.Errorf("expected 3 stops, got %d", len(result.Route))
	}
	if result.Route[0].HubID != "A" || result.Route[1].HubID != "B" || result.Route[2].HubID != "C" {
		t.Errorf("unexpected route: %+v", result.Route)
	}
	if result.TotalDistanceKm != 15 {
		t.Errorf("expected TotalDistanceKm=15, got %f", result.TotalDistanceKm)
	}
}

func TestDijkstra_SameOriginDestination(t *testing.T) {
	g := buildSimpleGraph()
	result, err := calculators.Dijkstra(g, "A", "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected a route, got nil")
	}
	if len(result.Route) != 1 {
		t.Errorf("expected 1 stop for same origin/dest, got %d", len(result.Route))
	}
}

func TestDijkstra_NoPath(t *testing.T) {
	// D is isolated — no edges connect to it
	g := buildSimpleGraph()
	g.Nodes["D"] = &models.RouteNode{NodeID: "n-D", HubID: "D", CityCode: "CD"}

	result, err := calculators.Dijkstra(g, "A", "D")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil (no path), got %+v", result)
	}
}

func TestDijkstra_UnknownOrigin(t *testing.T) {
	g := buildSimpleGraph()
	_, err := calculators.Dijkstra(g, "UNKNOWN", "C")
	if err == nil {
		t.Error("expected error for unknown origin, got nil")
	}
}

func TestDijkstra_UnknownDestination(t *testing.T) {
	g := buildSimpleGraph()
	_, err := calculators.Dijkstra(g, "A", "UNKNOWN")
	if err == nil {
		t.Error("expected error for unknown destination, got nil")
	}
}

func TestDijkstra_InactiveEdgesIgnored(t *testing.T) {
	// Only inactive edge A→B; only active edge A→C
	nodes := map[string]*models.RouteNode{
		"A": {NodeID: "n-A", HubID: "A", CityCode: "CA"},
		"B": {NodeID: "n-B", HubID: "B", CityCode: "CB"},
		"C": {NodeID: "n-C", HubID: "C", CityCode: "CC"},
	}
	edges := []models.RouteEdge{
		{EdgeID: "e1", FromNodeID: "n-A", ToNodeID: "n-B", DistanceKm: 1, AvgTransitHours: 0.1, IsActive: false},
		{EdgeID: "e2", FromNodeID: "n-A", ToNodeID: "n-C", DistanceKm: 50, AvgTransitHours: 2, IsActive: true},
	}
	g := &models.Graph{Nodes: nodes, Edges: edges}

	// A→B should be unreachable (edge inactive)
	result, err := calculators.Dijkstra(g, "A", "B")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil (inactive edge), got route with %d stops", len(result.Route))
	}
}

func TestDijkstra_SequenceNumbers(t *testing.T) {
	g := buildSimpleGraph()
	result, _ := calculators.Dijkstra(g, "A", "C")
	for i, stop := range result.Route {
		if stop.Sequence != i+1 {
			t.Errorf("stop[%d].Sequence = %d, want %d", i, stop.Sequence, i+1)
		}
	}
}

func TestEdgeWeight(t *testing.T) {
	edge := models.RouteEdge{DistanceKm: 100, AvgTransitHours: 2}
	got := calculators.EdgeWeight(edge, 1, 1)
	if got != 102 {
		t.Errorf("EdgeWeight = %f, want 102", got)
	}
	got2 := calculators.EdgeWeight(edge, 0.5, 2)
	if got2 != 54 {
		t.Errorf("EdgeWeight(0.5,2) = %f, want 54", got2)
	}
}
