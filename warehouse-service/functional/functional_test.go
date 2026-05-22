// Package functional contains end-to-end functional tests for the warehouse-service.
//
// These tests wire the real WarehouseHandler, real WarehouseService, and the
// in-memory WarehouseRepository together — no database or Docker required.
// They exercise the full HTTP request → handler → service → repository path,
// verifying that all three layers integrate correctly.
package functional

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	handler "github.com/fujiwara-express/warehouse-service/internal/delivery/http"
	"github.com/fujiwara-express/warehouse-service/internal/domain"
	"github.com/fujiwara-express/warehouse-service/internal/repository"
	"github.com/fujiwara-express/warehouse-service/internal/service"
)

// ── test helpers ─────────────────────────────────────────────────────────────

// newTestServer wires up a fresh in-memory stack and returns an httptest.Server.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	repo := repository.NewMemoryRepository()
	svc := service.NewWarehouseService(repo)
	h := handler.NewWarehouseHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("/receive", h.HandleReceiveItem)
	mux.HandleFunc("/dispatch", h.HandleDispatchItem)
	mux.HandleFunc("/check-stock", h.HandleCheckStock)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func postJSON(t *testing.T, srv *httptest.Server, path string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	resp, err := http.Post(srv.URL+path, "application/json", bytes.NewReader(b))
	require.NoError(t, err)
	return resp
}

func getJSON(t *testing.T, srv *httptest.Server, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(srv.URL + path)
	require.NoError(t, err)
	return resp
}

// ── POST /receive ─────────────────────────────────────────────────────────────

// TestFunctional_ReceiveItem_NewItem verifies that a new item is stored and
// the API returns HTTP 201.
func TestFunctional_ReceiveItem_NewItem(t *testing.T) {
	srv := newTestServer(t)

	resp := postJSON(t, srv, "/receive", domain.Item{
		ID: "BRG-001", Name: "Laptop Gaming", Quantity: 50, Location: "Rak-A1",
	})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

// TestFunctional_ReceiveItem_StockAccumulates verifies that receiving the same
// item twice accumulates the quantity correctly.
func TestFunctional_ReceiveItem_StockAccumulates(t *testing.T) {
	srv := newTestServer(t)

	// First receive: 20 units
	r1 := postJSON(t, srv, "/receive", domain.Item{ID: "BRG-002", Name: "Mouse", Quantity: 20, Location: "Rak-B1"})
	r1.Body.Close()
	require.Equal(t, http.StatusCreated, r1.StatusCode)

	// Second receive: 15 more units
	r2 := postJSON(t, srv, "/receive", domain.Item{ID: "BRG-002", Name: "Mouse", Quantity: 15, Location: ""})
	r2.Body.Close()
	require.Equal(t, http.StatusCreated, r2.StatusCode)

	// Check stock: should be 35
	resp := getJSON(t, srv, "/check-stock?id=BRG-002")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var item domain.Item
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&item))
	assert.Equal(t, 35, item.Quantity)
	assert.Equal(t, "Rak-B1", item.Location, "location should be unchanged when incoming location is empty")
}

// TestFunctional_ReceiveItem_LocationOverwritten verifies that providing a new
// location on a subsequent receive updates the stored location.
func TestFunctional_ReceiveItem_LocationOverwritten(t *testing.T) {
	srv := newTestServer(t)

	postJSON(t, srv, "/receive", domain.Item{ID: "BRG-003", Name: "Keyboard", Quantity: 10, Location: "Rak-C1"}).Body.Close()
	postJSON(t, srv, "/receive", domain.Item{ID: "BRG-003", Name: "Keyboard", Quantity: 5, Location: "Rak-D2"}).Body.Close()

	resp := getJSON(t, srv, "/check-stock?id=BRG-003")
	defer resp.Body.Close()

	var item domain.Item
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&item))
	assert.Equal(t, "Rak-D2", item.Location)
	assert.Equal(t, 15, item.Quantity)
}

// TestFunctional_ReceiveItem_InvalidJSON verifies that malformed JSON returns 400.
func TestFunctional_ReceiveItem_InvalidJSON(t *testing.T) {
	srv := newTestServer(t)

	resp, err := http.Post(srv.URL+"/receive", "application/json", bytes.NewReader([]byte("{bad json")))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ── POST /dispatch ────────────────────────────────────────────────────────────

// TestFunctional_DispatchItem_Valid verifies that dispatching a valid quantity
// reduces the stock and returns HTTP 200.
func TestFunctional_DispatchItem_Valid(t *testing.T) {
	srv := newTestServer(t)

	// Seed stock
	postJSON(t, srv, "/receive", domain.Item{ID: "BRG-010", Name: "Monitor", Quantity: 30, Location: "Rak-E1"}).Body.Close()

	// Dispatch 10 units
	resp := postJSON(t, srv, "/dispatch", domain.StockRequest{ItemID: "BRG-010", Quantity: 10})
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify remaining stock
	checkResp := getJSON(t, srv, "/check-stock?id=BRG-010")
	defer checkResp.Body.Close()

	var item domain.Item
	require.NoError(t, json.NewDecoder(checkResp.Body).Decode(&item))
	assert.Equal(t, 20, item.Quantity)
}

// TestFunctional_DispatchItem_ExactStock verifies that dispatching exactly the
// available quantity reduces stock to zero.
func TestFunctional_DispatchItem_ExactStock(t *testing.T) {
	srv := newTestServer(t)

	postJSON(t, srv, "/receive", domain.Item{ID: "BRG-011", Name: "Headset", Quantity: 5, Location: "Rak-F1"}).Body.Close()

	resp := postJSON(t, srv, "/dispatch", domain.StockRequest{ItemID: "BRG-011", Quantity: 5})
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	checkResp := getJSON(t, srv, "/check-stock?id=BRG-011")
	defer checkResp.Body.Close()

	var item domain.Item
	require.NoError(t, json.NewDecoder(checkResp.Body).Decode(&item))
	assert.Equal(t, 0, item.Quantity)
}

// TestFunctional_DispatchItem_OutOfStock verifies that dispatching more than
// available returns HTTP 400.
func TestFunctional_DispatchItem_OutOfStock(t *testing.T) {
	srv := newTestServer(t)

	postJSON(t, srv, "/receive", domain.Item{ID: "BRG-012", Name: "Webcam", Quantity: 3, Location: "Rak-G1"}).Body.Close()

	resp := postJSON(t, srv, "/dispatch", domain.StockRequest{ItemID: "BRG-012", Quantity: 100})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Stock must remain unchanged
	checkResp := getJSON(t, srv, "/check-stock?id=BRG-012")
	defer checkResp.Body.Close()

	var item domain.Item
	require.NoError(t, json.NewDecoder(checkResp.Body).Decode(&item))
	assert.Equal(t, 3, item.Quantity)
}

// TestFunctional_DispatchItem_ItemNotFound verifies that dispatching a
// non-existent item returns HTTP 400.
func TestFunctional_DispatchItem_ItemNotFound(t *testing.T) {
	srv := newTestServer(t)

	resp := postJSON(t, srv, "/dispatch", domain.StockRequest{ItemID: "BRG-GHOST", Quantity: 1})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestFunctional_DispatchItem_InvalidJSON verifies that malformed JSON returns 400.
func TestFunctional_DispatchItem_InvalidJSON(t *testing.T) {
	srv := newTestServer(t)

	resp, err := http.Post(srv.URL+"/dispatch", "application/json", bytes.NewReader([]byte("{bad")))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ── GET /check-stock ──────────────────────────────────────────────────────────

// TestFunctional_CheckStock_Found verifies that checking stock for an existing
// item returns HTTP 200 with the correct JSON fields.
func TestFunctional_CheckStock_Found(t *testing.T) {
	srv := newTestServer(t)

	postJSON(t, srv, "/receive", domain.Item{ID: "BRG-020", Name: "SSD 1TB", Quantity: 25, Location: "Rak-H1"}).Body.Close()

	resp := getJSON(t, srv, "/check-stock?id=BRG-020")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	var item domain.Item
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&item))
	assert.Equal(t, "BRG-020", item.ID)
	assert.Equal(t, "SSD 1TB", item.Name)
	assert.Equal(t, 25, item.Quantity)
	assert.Equal(t, "Rak-H1", item.Location)
}

// TestFunctional_CheckStock_NotFound verifies that checking a non-existent
// item returns HTTP 404.
func TestFunctional_CheckStock_NotFound(t *testing.T) {
	srv := newTestServer(t)

	resp := getJSON(t, srv, "/check-stock?id=BRG-GHOST")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestFunctional_CheckStock_MissingID verifies that omitting the id query
// parameter returns HTTP 400.
func TestFunctional_CheckStock_MissingID(t *testing.T) {
	srv := newTestServer(t)

	resp := getJSON(t, srv, "/check-stock")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ── Full lifecycle ────────────────────────────────────────────────────────────

// TestFunctional_FullLifecycle exercises the complete happy-path lifecycle:
// Receive → CheckStock → Dispatch → CheckStock, verifying state at each step.
func TestFunctional_FullLifecycle(t *testing.T) {
	srv := newTestServer(t)

	// 1. Receive 100 units of a new item
	r1 := postJSON(t, srv, "/receive", domain.Item{
		ID: "BRG-100", Name: "RAM 16GB", Quantity: 100, Location: "Rak-Z1",
	})
	r1.Body.Close()
	require.Equal(t, http.StatusCreated, r1.StatusCode)

	// 2. Check stock — should be 100
	c1 := getJSON(t, srv, "/check-stock?id=BRG-100")
	var item domain.Item
	require.NoError(t, json.NewDecoder(c1.Body).Decode(&item))
	c1.Body.Close()
	assert.Equal(t, 100, item.Quantity)
	assert.Equal(t, "Rak-Z1", item.Location)

	// 3. Receive 50 more units (location unchanged)
	r2 := postJSON(t, srv, "/receive", domain.Item{ID: "BRG-100", Name: "RAM 16GB", Quantity: 50})
	r2.Body.Close()
	require.Equal(t, http.StatusCreated, r2.StatusCode)

	// 4. Check stock — should be 150
	c2 := getJSON(t, srv, "/check-stock?id=BRG-100")
	require.NoError(t, json.NewDecoder(c2.Body).Decode(&item))
	c2.Body.Close()
	assert.Equal(t, 150, item.Quantity)

	// 5. Dispatch 30 units
	d1 := postJSON(t, srv, "/dispatch", domain.StockRequest{ItemID: "BRG-100", Quantity: 30})
	d1.Body.Close()
	require.Equal(t, http.StatusOK, d1.StatusCode)

	// 6. Check stock — should be 120
	c3 := getJSON(t, srv, "/check-stock?id=BRG-100")
	require.NoError(t, json.NewDecoder(c3.Body).Decode(&item))
	c3.Body.Close()
	assert.Equal(t, 120, item.Quantity)

	// 7. Attempt to dispatch more than available — should fail
	d2 := postJSON(t, srv, "/dispatch", domain.StockRequest{ItemID: "BRG-100", Quantity: 9999})
	d2.Body.Close()
	assert.Equal(t, http.StatusBadRequest, d2.StatusCode)

	// 8. Stock must still be 120 after failed dispatch
	c4 := getJSON(t, srv, "/check-stock?id=BRG-100")
	require.NoError(t, json.NewDecoder(c4.Body).Decode(&item))
	c4.Body.Close()
	assert.Equal(t, 120, item.Quantity)
}

// TestFunctional_MultipleItems verifies that multiple independent items are
// tracked separately without interfering with each other.
func TestFunctional_MultipleItems(t *testing.T) {
	srv := newTestServer(t)

	items := []domain.Item{
		{ID: "BRG-A", Name: "Item A", Quantity: 10, Location: "Rak-1"},
		{ID: "BRG-B", Name: "Item B", Quantity: 20, Location: "Rak-2"},
		{ID: "BRG-C", Name: "Item C", Quantity: 30, Location: "Rak-3"},
	}

	for _, item := range items {
		r := postJSON(t, srv, "/receive", item)
		r.Body.Close()
		require.Equal(t, http.StatusCreated, r.StatusCode)
	}

	// Dispatch from B only
	d := postJSON(t, srv, "/dispatch", domain.StockRequest{ItemID: "BRG-B", Quantity: 5})
	d.Body.Close()
	require.Equal(t, http.StatusOK, d.StatusCode)

	// Verify A and C are untouched
	for _, tc := range []struct {
		id   string
		want int
	}{
		{"BRG-A", 10},
		{"BRG-B", 15},
		{"BRG-C", 30},
	} {
		resp := getJSON(t, srv, "/check-stock?id="+tc.id)
		var got domain.Item
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		resp.Body.Close()
		assert.Equal(t, tc.want, got.Quantity, "item %s quantity mismatch", tc.id)
	}
}
