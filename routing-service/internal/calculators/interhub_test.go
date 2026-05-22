package calculators_test

import (
	"math"
	"testing"

	"pgregory.net/rapid"

	"routing-service/internal/calculators"
	"routing-service/internal/models"
)

// ── Generators ───────────────────────────────────────────────────────────────

// genGraph generates a random connected graph with 2–8 nodes.
// A spanning chain H0→H1→...→H(n-1) guarantees a path always exists.
func genGraph(t *rapid.T) (*models.Graph, int) {
	n := rapid.IntRange(2, 8).Draw(t, "n")

	nodes := make(map[string]*models.RouteNode, n)
	for i := 0; i < n; i++ {
		hubID := rapid.StringMatching(`H\d`).Draw(t, "unused") // just use manual IDs
		_ = hubID
		id := nodeID(i)
		hubID = hubIDStr(i)
		nodes[hubID] = &models.RouteNode{
			NodeID:   id,
			HubID:    hubID,
			CityCode: "C" + hubIDStr(i),
		}
	}

	// Spanning chain ensures connectivity
	edges := make([]models.RouteEdge, 0, n-1)
	for i := 0; i < n-1; i++ {
		edges = append(edges, models.RouteEdge{
			EdgeID:          "chain-" + hubIDStr(i),
			FromNodeID:      nodeID(i),
			ToNodeID:        nodeID(i + 1),
			DistanceKm:      100,
			AvgTransitHours: 2,
			TransportMode:   "DARAT",
			IsActive:        true,
		})
	}

	// Extra random edges
	extraCount := rapid.IntRange(0, n*2).Draw(t, "extra_edges")
	seen := map[string]bool{}
	for _, e := range edges {
		seen[e.FromNodeID+":"+e.ToNodeID] = true
	}
	for range extraCount {
		from := rapid.IntRange(0, n-1).Draw(t, "from")
		to := rapid.IntRange(0, n-1).Draw(t, "to")
		if from == to {
			continue
		}
		key := nodeID(from) + ":" + nodeID(to)
		if seen[key] {
			continue
		}
		seen[key] = true
		dist := rapid.Float64Range(1, 1000).Draw(t, "dist")
		transit := rapid.Float64Range(0.5, 48).Draw(t, "transit")
		edges = append(edges, models.RouteEdge{
			EdgeID:          "e-" + key,
			FromNodeID:      nodeID(from),
			ToNodeID:        nodeID(to),
			DistanceKm:      dist,
			AvgTransitHours: transit,
			TransportMode:   "DARAT",
			IsActive:        true,
		})
	}

	return &models.Graph{Nodes: nodes, Edges: edges}, n
}

// ── Property 8: Dijkstra optimality ─────────────────────────────────────────

// Feature: routing-service, Property 8: Dijkstra optimality — no shorter path exists
// Validates: Requirements 1.2
func TestProperty8_DijkstraOptimality(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		graph, n := genGraph(t)
		origin := hubIDStr(0)
		dest := hubIDStr(n - 1)

		result, _ := calculators.Dijkstra(graph, origin, dest)
		bruteMin := bruteForceShortestPath(graph, origin, dest, 1, 1)

		if result == nil {
			if !math.IsInf(bruteMin, 1) {
				t.Fatalf("Dijkstra returned nil but brute force found path with cost %f", bruteMin)
			}
			return
		}

		// Compute cost of Dijkstra's path
		dijkstraCost := pathCost(graph, result.Route)
		if math.Abs(dijkstraCost-bruteMin) > 1e-6 {
			t.Fatalf("Dijkstra cost %f != brute force min %f", dijkstraCost, bruteMin)
		}
	})
}

// ── Property 1: Inter-hub route sequence integrity ───────────────────────────

// Feature: routing-service, Property 1: Inter-hub route sequence integrity
// Validates: Requirements 1.1, 1.2
func TestProperty1_RouteSequenceIntegrity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		graph, n := genGraph(t)
		origin := hubIDStr(0)
		dest := hubIDStr(n - 1)

		result, _ := calculators.Dijkstra(graph, origin, dest)
		if result == nil {
			return // no path — nothing to check
		}

		route := result.Route
		if len(route) == 0 {
			t.Fatal("route is empty")
		}
		if route[0].HubID != origin {
			t.Fatalf("route does not start at origin: got %s", route[0].HubID)
		}
		if route[len(route)-1].HubID != dest {
			t.Fatalf("route does not end at destination: got %s", route[len(route)-1].HubID)
		}

		// Each consecutive pair must be connected by an active edge
		nodeIDMap := buildNodeIDMap(graph)
		for i := 1; i < len(route); i++ {
			fromHub := route[i-1].HubID
			toHub := route[i].HubID
			if !hasActiveEdge(graph, nodeIDMap, fromHub, toHub) {
				t.Fatalf("no active edge from %s to %s", fromHub, toHub)
			}
		}

		// Sequence numbers must be 1-based and contiguous
		for i, stop := range route {
			if stop.Sequence != i+1 {
				t.Fatalf("sequence[%d] = %d, want %d", i, stop.Sequence, i+1)
			}
		}
	})
}

// ── helpers ──────────────────────────────────────────────────────────────────

func nodeID(i int) string   { return "node-" + hubIDStr(i) }
func hubIDStr(i int) string { return "H" + string(rune('0'+i)) }

func buildNodeIDMap(graph *models.Graph) map[string]string {
	m := make(map[string]string)
	for hubID, node := range graph.Nodes {
		m[node.NodeID] = hubID
	}
	return m
}

func hasActiveEdge(graph *models.Graph, nodeIDMap map[string]string, fromHub, toHub string) bool {
	for _, e := range graph.Edges {
		if !e.IsActive {
			continue
		}
		if nodeIDMap[e.FromNodeID] == fromHub && nodeIDMap[e.ToNodeID] == toHub {
			return true
		}
	}
	return false
}

func pathCost(graph *models.Graph, route []models.InterHubRouteStop) float64 {
	nodeIDMap := buildNodeIDMap(graph)
	total := 0.0
	for i := 1; i < len(route); i++ {
		fromHub := route[i-1].HubID
		toHub := route[i].HubID
		for _, e := range graph.Edges {
			if nodeIDMap[e.FromNodeID] == fromHub && nodeIDMap[e.ToNodeID] == toHub {
				total += calculators.EdgeWeight(e, 1, 1)
				break
			}
		}
	}
	return total
}

// bruteForceShortestPath finds the minimum-weight path via DFS over all simple paths.
func bruteForceShortestPath(graph *models.Graph, origin, dest string, alpha, beta float64) float64 {
	adj := make(map[string][]struct {
		toHub string
		edge  models.RouteEdge
	})
	nodeIDMap := buildNodeIDMap(graph)
	for _, e := range graph.Edges {
		from := nodeIDMap[e.FromNodeID]
		to := nodeIDMap[e.ToNodeID]
		adj[from] = append(adj[from], struct {
			toHub string
			edge  models.RouteEdge
		}{to, e})
	}

	best := math.Inf(1)
	visited := map[string]bool{origin: true}

	var dfs func(cur string, cost float64)
	dfs = func(cur string, cost float64) {
		if cur == dest {
			if cost < best {
				best = cost
			}
			return
		}
		for _, entry := range adj[cur] {
			if visited[entry.toHub] {
				continue
			}
			visited[entry.toHub] = true
			dfs(entry.toHub, cost+calculators.EdgeWeight(entry.edge, alpha, beta))
			visited[entry.toHub] = false
		}
	}
	dfs(origin, 0)
	return best
}
