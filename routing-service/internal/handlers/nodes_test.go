package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"routing-service/internal/handlers"
	"routing-service/internal/models"
	"routing-service/internal/repositories"
)

// ── stub node repository ─────────────────────────────────────────────────────

type stubNodeRepo struct {
	nodes []models.RouteNode
	node  *models.RouteNode
	err   error
}

func (s *stubNodeRepo) GetAllNodes(_ context.Context) ([]models.RouteNode, error) {
	return s.nodes, s.err
}
func (s *stubNodeRepo) CreateNode(_ context.Context, _ models.CreateNodeInput) (*models.RouteNode, error) {
	return s.node, s.err
}

// ── helpers ──────────────────────────────────────────────────────────────────

func newNodesRouter(repo *stubNodeRepo) *gin.Engine {
	r := gin.New()
	h := handlers.NewNodesHandler(repo)
	r.GET("/routing/nodes", h.GetNodes)
	r.POST("/routing/nodes", h.CreateNode)
	return r
}

// ── GET /routing/nodes ───────────────────────────────────────────────────────

func TestGetNodes_ReturnsAllNodes(t *testing.T) {
	repo := &stubNodeRepo{nodes: []models.RouteNode{
		{NodeID: "n1", HubID: "HUB_JKT", CityCode: "JKT", Latitude: -6.2, Longitude: 106.8},
	}}
	r := newNodesRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/routing/nodes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result []models.RouteNode
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(result) != 1 || result[0].HubID != "HUB_JKT" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestGetNodes_EmptyList(t *testing.T) {
	repo := &stubNodeRepo{nodes: nil}
	r := newNodesRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/routing/nodes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// Should return [] not null
	if w.Body.String() != "[]" {
		t.Errorf("expected empty array, got: %s", w.Body.String())
	}
}

// ── POST /routing/nodes ──────────────────────────────────────────────────────

func TestCreateNode_Success(t *testing.T) {
	created := &models.RouteNode{NodeID: "n-new", HubID: "HUB_BDG", CityCode: "BDG", Latitude: -6.9, Longitude: 107.6}
	repo := &stubNodeRepo{node: created}
	r := newNodesRouter(repo)

	body, _ := json.Marshal(models.CreateNodeInput{
		HubID: "HUB_BDG", CityCode: "BDG", Latitude: -6.9, Longitude: 107.6,
	})
	req := httptest.NewRequest(http.MethodPost, "/routing/nodes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "CREATED" {
		t.Errorf("expected status CREATED, got %v", resp["status"])
	}
	if resp["hub_id"] != "HUB_BDG" {
		t.Errorf("expected hub_id HUB_BDG, got %v", resp["hub_id"])
	}
}

func TestCreateNode_MissingFields_Returns400(t *testing.T) {
	repo := &stubNodeRepo{}
	r := newNodesRouter(repo)

	// Missing hub_id
	body := []byte(`{"city_code":"BDG","latitude":-6.9,"longitude":107.6}`)
	req := httptest.NewRequest(http.MethodPost, "/routing/nodes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateNode_DuplicateHub_Returns409(t *testing.T) {
	repo := &stubNodeRepo{err: repositories.ErrDuplicate}
	r := newNodesRouter(repo)

	body, _ := json.Marshal(models.CreateNodeInput{
		HubID: "HUB_JKT", CityCode: "JKT", Latitude: -6.2, Longitude: 106.8,
	})
	req := httptest.NewRequest(http.MethodPost, "/routing/nodes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "DUPLICATE_HUB" {
		t.Errorf("expected DUPLICATE_HUB error, got %v", resp["error"])
	}
}
