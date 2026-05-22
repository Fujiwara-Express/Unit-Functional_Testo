package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"pgregory.net/rapid"

	"routing-service/internal/handlers"
	"routing-service/internal/models"
	"routing-service/internal/services"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ── In-memory cache for testing ──────────────────────────────────────────────

type memCache struct {
	store map[string][]byte
}

func newMemCache() *memCache { return &memCache{store: map[string][]byte{}} }

func (m *memCache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	data, ok := m.store[key]
	if !ok {
		return false, nil
	}
	return true, json.Unmarshal(data, dest)
}

func (m *memCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	m.store[key] = data
	return nil
}

func (m *memCache) InvalidatePattern(ctx context.Context, pattern string) error {
	// Simple glob: "route:*" → delete all keys starting with "route:"
	prefix := pattern[:len(pattern)-1] // strip trailing *
	for k := range m.store {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(m.store, k)
		}
	}
	return nil
}

// ── Stub edge repository ─────────────────────────────────────────────────────

type stubEdgeRepo struct {
	edge *models.RouteEdge
}

func (s *stubEdgeRepo) GetAllEdges(_ context.Context, _ *string) ([]models.RouteEdge, error) {
	return nil, nil
}
func (s *stubEdgeRepo) CreateEdge(_ context.Context, _ models.CreateEdgeInput) (*models.RouteEdge, error) {
	return s.edge, nil
}
func (s *stubEdgeRepo) UpdateEdge(_ context.Context, _ string, _ models.UpdateEdgeInput) (*models.RouteEdge, error) {
	return s.edge, nil
}

// ── Generators ───────────────────────────────────────────────────────────────

func genEdge(t *rapid.T) models.RouteEdge {
	modes := []string{"DARAT", "UDARA", "LAUT"}
	// Use StringMatching to ensure edge_id is URL-safe (alphanumeric)
	edgeID := rapid.StringMatching(`[a-zA-Z0-9]{1,8}`).Draw(t, "edge_id")
	fromID := rapid.StringN(1, 8, 8).Draw(t, "from_node_id")
	toID := rapid.StringN(1, 8, 8).Draw(t, "to_node_id")
	return models.RouteEdge{
		EdgeID:          edgeID,
		FromNodeID:      fromID,
		ToNodeID:        toID,
		DistanceKm:      rapid.Float64Range(1, 5000).Draw(t, "dist"),
		AvgTransitHours: rapid.Float64Range(0.5, 200).Draw(t, "transit"),
		TransportMode:   modes[rapid.IntRange(0, 2).Draw(t, "mode")],
		IsActive:        rapid.Bool().Draw(t, "is_active"),
	}
}

func genCacheKeys(t *rapid.T) []string {
	n := rapid.IntRange(1, 5).Draw(t, "n_keys")
	keys := make([]string, n)
	for i := range keys {
		origin := rapid.StringN(1, 5, 5).Draw(t, fmt.Sprintf("origin_%d", i))
		dest := rapid.StringN(1, 5, 5).Draw(t, fmt.Sprintf("dest_%d", i))
		svc := []string{"REG", "EXP"}[rapid.IntRange(0, 1).Draw(t, fmt.Sprintf("svc_%d", i))]
		keys[i] = services.InterHubKey(origin, dest, svc)
	}
	return keys
}

// ── Property 6: Edge update invalidates affected cache ───────────────────────

// Feature: routing-service, Property 6: Edge update invalidates affected cache
// Validates: Requirements 3.4
func TestProperty6_EdgeUpdateInvalidatesCache(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		edge := genEdge(t)
		cacheKeys := genCacheKeys(t)

		cache := newMemCache()
		dummyRoute := models.InterHubRoute{Origin: "A", Destination: "B", TotalDistanceKm: 100}

		// Pre-populate cache
		for _, key := range cacheKeys {
			_ = cache.Set(context.Background(), key, dummyRoute, services.InterHubTTL)
		}

		// Verify all keys present before update
		for _, key := range cacheKeys {
			var r models.InterHubRoute
			found, _ := cache.Get(context.Background(), key, &r)
			if !found {
				t.Fatalf("key %s should be present before update", key)
			}
		}

		// Build handler and perform PATCH
		repo := &stubEdgeRepo{edge: &edge}
		handler := handlers.NewEdgesHandler(repo, cache)

		router := gin.New()
		router.PATCH("/routing/edges/:edge_id", handler.UpdateEdge)

		body, _ := json.Marshal(map[string]interface{}{"is_active": !edge.IsActive})
		req := httptest.NewRequest(http.MethodPatch,
			"/routing/edges/"+edge.EdgeID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		// All route:* cache entries must now be gone
		for _, key := range cacheKeys {
			var r models.InterHubRoute
			found, _ := cache.Get(context.Background(), key, &r)
			if found {
				t.Fatalf("key %s should be invalidated after edge update", key)
			}
		}
	})
}
