package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"routing-service/internal/handlers"
	"routing-service/internal/models"
)

// ── stub graph repository ────────────────────────────────────────────────────

type stubGraphRepo struct {
	graph *models.Graph
	err   error
}

func (s *stubGraphRepo) GetActiveGraph(_ context.Context) (*models.Graph, error) {
	return s.graph, s.err
}

// ── stub route cache ─────────────────────────────────────────────────────────

type stubRouteCache struct {
	data map[string][]byte
}

func newStubRouteCache() *stubRouteCache {
	return &stubRouteCache{data: map[string][]byte{}}
}

func (s *stubRouteCache) Get(_ context.Context, key string, dest interface{}) (bool, error) {
	raw, ok := s.data[key]
	if !ok {
		return false, nil
	}
	return true, json.Unmarshal(raw, dest)
}

func (s *stubRouteCache) Set(_ context.Context, key string, value interface{}, _ time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	s.data[key] = raw
	return nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

// twoNodeGraph builds a minimal A→B graph.
func twoNodeGraph() *models.Graph {
	return &models.Graph{
		Nodes: map[string]*models.RouteNode{
			"JKT": {NodeID: "n-JKT", HubID: "JKT", CityCode: "JKT"},
			"BDG": {NodeID: "n-BDG", HubID: "BDG", CityCode: "BDG"},
		},
		Edges: []models.RouteEdge{
			{EdgeID: "e1", FromNodeID: "n-JKT", ToNodeID: "n-BDG",
				DistanceKm: 120, AvgTransitHours: 3, TransportMode: "DARAT", IsActive: true},
		},
	}
}

func newRouteRouter(repo *stubGraphRepo, cache *stubRouteCache) *gin.Engine {
	r := gin.New()
	h := handlers.NewRouteHandler(repo, cache)
	r.GET("/routing/route", h.GetRoute)
	return r
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestGetRoute_Success(t *testing.T) {
	repo := &stubGraphRepo{graph: twoNodeGraph()}
	cache := newStubRouteCache()
	r := newRouteRouter(repo, cache)

	req := httptest.NewRequest(http.MethodGet, "/routing/route?origin=JKT&destination=BDG&service_type=REG", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result models.InterHubRoute
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.Origin != "JKT" || result.Destination != "BDG" {
		t.Errorf("unexpected route: %+v", result)
	}
}

func TestGetRoute_MissingParams_Returns400(t *testing.T) {
	repo := &stubGraphRepo{graph: twoNodeGraph()}
	cache := newStubRouteCache()
	r := newRouteRouter(repo, cache)

	// Missing destination and service_type
	req := httptest.NewRequest(http.MethodGet, "/routing/route?origin=JKT", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetRoute_NoPath_Returns404(t *testing.T) {
	// Graph with no edges between JKT and SBY
	g := &models.Graph{
		Nodes: map[string]*models.RouteNode{
			"JKT": {NodeID: "n-JKT", HubID: "JKT", CityCode: "JKT"},
			"SBY": {NodeID: "n-SBY", HubID: "SBY", CityCode: "SBY"},
		},
		Edges: []models.RouteEdge{},
	}
	repo := &stubGraphRepo{graph: g}
	cache := newStubRouteCache()
	r := newRouteRouter(repo, cache)

	req := httptest.NewRequest(http.MethodGet, "/routing/route?origin=JKT&destination=SBY&service_type=REG", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetRoute_CacheHit(t *testing.T) {
	repo := &stubGraphRepo{graph: twoNodeGraph()}
	cache := newStubRouteCache()

	// Pre-populate cache
	cached := models.InterHubRoute{Origin: "JKT", Destination: "BDG", TotalDistanceKm: 999}
	raw, _ := json.Marshal(cached)
	cache.data["route:JKT:BDG:REG"] = raw

	r := newRouteRouter(repo, cache)
	req := httptest.NewRequest(http.MethodGet, "/routing/route?origin=JKT&destination=BDG&service_type=REG", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result models.InterHubRoute
	json.Unmarshal(w.Body.Bytes(), &result)
	// Should return cached value (999), not recalculated (120)
	if result.TotalDistanceKm != 999 {
		t.Errorf("expected cached TotalDistanceKm=999, got %f", result.TotalDistanceKm)
	}
}
