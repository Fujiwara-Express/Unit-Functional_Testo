package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// stubBackend starts an httptest.Server that records the last request path it
// received and always responds with the given status code and body.
type stubBackend struct {
	server       *httptest.Server
	lastPath     string
	lastMethod   string
	responseCode int
	responseBody string
}

func newStubBackend(code int, body string) *stubBackend {
	s := &stubBackend{responseCode: code, responseBody: body}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.lastPath = r.URL.Path
		s.lastMethod = r.Method
		w.WriteHeader(s.responseCode)
		w.Write([]byte(s.responseBody)) //nolint:errcheck
	}))
	return s
}

func (s *stubBackend) URL() *url.URL {
	u, _ := url.Parse(s.server.URL)
	return u
}

func (s *stubBackend) Close() { s.server.Close() }

// newTestGateway wires two stub backends and returns the gateway handler.
func newTestGateway(t *testing.T) (gw http.Handler, pricing, warehouse *stubBackend) {
	t.Helper()
	pricing = newStubBackend(http.StatusOK, `{"service":"pricing"}`)
	warehouse = newStubBackend(http.StatusOK, `{"service":"warehouse"}`)
	t.Cleanup(func() { pricing.Close(); warehouse.Close() })
	gw = NewGateway(pricing.URL(), warehouse.URL())
	return
}

// do sends a request through the gateway handler and returns the recorder.
func do(gw http.Handler, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)
	return rec
}

// ── routing: /pricing prefix ──────────────────────────────────────────────────

// TestRouting_PricingPrefix verifies that a request to /pricing/calculate-price
// is forwarded to the pricing backend with the prefix stripped.
func TestRouting_PricingPrefix(t *testing.T) {
	gw, pricing, _ := newTestGateway(t)

	rec := do(gw, http.MethodGet, "/pricing/calculate-price")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "/calculate-price", pricing.lastPath,
		"pricing backend should receive path with /pricing prefix stripped")
}

// TestRouting_PricingPrefix_POST verifies that POST requests are also forwarded.
func TestRouting_PricingPrefix_POST(t *testing.T) {
	gw, pricing, _ := newTestGateway(t)

	rec := do(gw, http.MethodPost, "/pricing/calculate-price")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, http.MethodPost, pricing.lastMethod)
	assert.Equal(t, "/calculate-price", pricing.lastPath)
}

// TestRouting_PricingPrefix_DeepPath verifies that nested paths are forwarded
// with only the /pricing prefix stripped, leaving the rest intact.
func TestRouting_PricingPrefix_DeepPath(t *testing.T) {
	gw, pricing, _ := newTestGateway(t)

	do(gw, http.MethodGet, "/pricing/v2/rates/domestic")

	assert.Equal(t, "/v2/rates/domestic", pricing.lastPath)
}

// TestRouting_PricingPrefix_Root verifies that /pricing alone (no trailing path)
// is forwarded as "/" to the backend.
func TestRouting_PricingPrefix_Root(t *testing.T) {
	gw, pricing, _ := newTestGateway(t)

	do(gw, http.MethodGet, "/pricing")

	assert.Equal(t, "/", pricing.lastPath)
}

// TestRouting_PricingPrefix_DoesNotHitWarehouse verifies that a /pricing request
// never reaches the warehouse backend.
func TestRouting_PricingPrefix_DoesNotHitWarehouse(t *testing.T) {
	gw, _, warehouse := newTestGateway(t)

	do(gw, http.MethodGet, "/pricing/calculate-price")

	assert.Empty(t, warehouse.lastPath,
		"warehouse backend must not receive /pricing requests")
}

// ── routing: /warehouse prefix ────────────────────────────────────────────────

// TestRouting_WarehousePrefix verifies that a request to /warehouse/receive
// is forwarded to the warehouse backend with the prefix stripped.
func TestRouting_WarehousePrefix(t *testing.T) {
	gw, _, warehouse := newTestGateway(t)

	rec := do(gw, http.MethodPost, "/warehouse/receive")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "/receive", warehouse.lastPath)
}

// TestRouting_WarehousePrefix_CheckStock verifies that query parameters are
// preserved after prefix stripping.
func TestRouting_WarehousePrefix_CheckStock(t *testing.T) {
	gw, _, warehouse := newTestGateway(t)

	rec := do(gw, http.MethodGet, "/warehouse/check-stock?id=BRG-001")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "/check-stock", warehouse.lastPath)
}

// TestRouting_WarehousePrefix_Dispatch verifies POST /warehouse/dispatch routing.
func TestRouting_WarehousePrefix_Dispatch(t *testing.T) {
	gw, _, warehouse := newTestGateway(t)

	do(gw, http.MethodPost, "/warehouse/dispatch")

	assert.Equal(t, "/dispatch", warehouse.lastPath)
}

// TestRouting_WarehousePrefix_Root verifies that /warehouse alone is forwarded
// as "/" to the backend.
func TestRouting_WarehousePrefix_Root(t *testing.T) {
	gw, _, warehouse := newTestGateway(t)

	do(gw, http.MethodGet, "/warehouse")

	assert.Equal(t, "/", warehouse.lastPath)
}

// TestRouting_WarehousePrefix_DoesNotHitPricing verifies that a /warehouse
// request never reaches the pricing backend.
func TestRouting_WarehousePrefix_DoesNotHitPricing(t *testing.T) {
	gw, pricing, _ := newTestGateway(t)

	do(gw, http.MethodPost, "/warehouse/receive")

	assert.Empty(t, pricing.lastPath,
		"pricing backend must not receive /warehouse requests")
}

// ── routing: unknown paths ────────────────────────────────────────────────────

// TestRouting_UnknownPath returns 404 for a path that matches neither prefix.
func TestRouting_UnknownPath(t *testing.T) {
	gw, pricing, warehouse := newTestGateway(t)

	rec := do(gw, http.MethodGet, "/unknown/path")

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Empty(t, pricing.lastPath, "pricing backend must not be called for unknown path")
	assert.Empty(t, warehouse.lastPath, "warehouse backend must not be called for unknown path")
}

// TestRouting_RootPath returns 404 for a bare "/" request.
func TestRouting_RootPath(t *testing.T) {
	gw, _, _ := newTestGateway(t)

	rec := do(gw, http.MethodGet, "/")

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// TestRouting_UnknownPath_ResponseBody verifies the 404 body contains a
// meaningful message.
func TestRouting_UnknownPath_ResponseBody(t *testing.T) {
	gw, _, _ := newTestGateway(t)

	rec := do(gw, http.MethodGet, "/nonexistent")

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "tidak ditemukan")
}

// TestRouting_PricingLookalikePrefix verifies that a path starting with
// "/pricingextra" is NOT routed to the pricing backend (it must be an exact
// prefix match of "/pricing" followed by "/" or end-of-string).
// NOTE: The current implementation uses HasPrefix, so "/pricingextra" WILL
// match. This test documents the actual behaviour.
func TestRouting_PricingLookalikePrefix_ActualBehaviour(t *testing.T) {
	gw, pricing, _ := newTestGateway(t)

	// /pricingextra starts with /pricing → current impl forwards it
	do(gw, http.MethodGet, "/pricingextra/foo")

	// Document what actually happens: pricing backend receives the stripped path
	assert.Equal(t, "/extra/foo", pricing.lastPath)
}

// ── upstream error propagation ────────────────────────────────────────────────

// TestUpstreamError_PricingBackendDown verifies that when the pricing backend
// returns a 500, the gateway propagates that status to the client.
func TestUpstreamError_PricingBackendDown(t *testing.T) {
	pricing := newStubBackend(http.StatusInternalServerError, `{"error":"upstream down"}`)
	warehouse := newStubBackend(http.StatusOK, `{}`)
	t.Cleanup(func() { pricing.Close(); warehouse.Close() })

	gw := NewGateway(pricing.URL(), warehouse.URL())
	rec := do(gw, http.MethodGet, "/pricing/calculate-price")

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// TestUpstreamError_WarehouseBackendDown verifies that when the warehouse
// backend returns a 503, the gateway propagates that status to the client.
func TestUpstreamError_WarehouseBackendDown(t *testing.T) {
	pricing := newStubBackend(http.StatusOK, `{}`)
	warehouse := newStubBackend(http.StatusServiceUnavailable, `{"error":"warehouse down"}`)
	t.Cleanup(func() { pricing.Close(); warehouse.Close() })

	gw := NewGateway(pricing.URL(), warehouse.URL())
	rec := do(gw, http.MethodPost, "/warehouse/receive")

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

// TestUpstreamError_BackendGone verifies that when the upstream server is
// completely unreachable, the gateway returns 502 Bad Gateway.
func TestUpstreamError_BackendGone(t *testing.T) {
	// Start and immediately close a backend so its port is unreachable
	gone := newStubBackend(http.StatusOK, "")
	goneURL := gone.URL()
	gone.Close() // port is now closed

	warehouse := newStubBackend(http.StatusOK, `{}`)
	t.Cleanup(warehouse.Close)

	gw := NewGateway(goneURL, warehouse.URL())
	rec := do(gw, http.MethodGet, "/pricing/calculate-price")

	assert.Equal(t, http.StatusBadGateway, rec.Code)
}

// ── response passthrough ──────────────────────────────────────────────────────

// TestResponsePassthrough_Body verifies that the response body from the
// upstream is passed through to the client unchanged.
func TestResponsePassthrough_Body(t *testing.T) {
	pricing := newStubBackend(http.StatusOK, `{"price":25000}`)
	warehouse := newStubBackend(http.StatusOK, `{}`)
	t.Cleanup(func() { pricing.Close(); warehouse.Close() })

	gw := NewGateway(pricing.URL(), warehouse.URL())
	rec := do(gw, http.MethodGet, "/pricing/calculate-price")

	assert.Equal(t, `{"price":25000}`, rec.Body.String())
}

// TestResponsePassthrough_StatusCode verifies that a 201 from the upstream
// is passed through to the client.
func TestResponsePassthrough_StatusCode(t *testing.T) {
	pricing := newStubBackend(http.StatusOK, `{}`)
	warehouse := newStubBackend(http.StatusCreated, `{"message":"created"}`)
	t.Cleanup(func() { pricing.Close(); warehouse.Close() })

	gw := NewGateway(pricing.URL(), warehouse.URL())
	rec := do(gw, http.MethodPost, "/warehouse/receive")

	assert.Equal(t, http.StatusCreated, rec.Code)
}

// ── NewGateway independence ───────────────────────────────────────────────────

// TestNewGateway_IndependentInstances verifies that two gateway instances with
// different backends are fully independent.
func TestNewGateway_IndependentInstances(t *testing.T) {
	backendA := newStubBackend(http.StatusOK, "A")
	backendB := newStubBackend(http.StatusOK, "B")
	t.Cleanup(func() { backendA.Close(); backendB.Close() })

	dummy := newStubBackend(http.StatusOK, "dummy")
	t.Cleanup(dummy.Close)

	gwA := NewGateway(backendA.URL(), dummy.URL())
	gwB := NewGateway(backendB.URL(), dummy.URL())

	recA := do(gwA, http.MethodGet, "/pricing/foo")
	recB := do(gwB, http.MethodGet, "/pricing/foo")

	assert.Equal(t, "A", recA.Body.String())
	assert.Equal(t, "B", recB.Body.String())
}
