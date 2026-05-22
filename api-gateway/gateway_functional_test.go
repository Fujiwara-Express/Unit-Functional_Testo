// Package main — functional tests for the API Gateway.
//
// These tests spin up real httptest.Server instances for both the gateway and
// the upstream backends, then send actual HTTP requests through the full stack.
// No mocks are used — everything is a real HTTP server.
package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── functional test environment ───────────────────────────────────────────────

// functionalEnv holds live httptest.Server instances for the gateway and both
// upstream backends.
type functionalEnv struct {
	gateway          *httptest.Server
	pricingBackend   *httptest.Server
	warehouseBackend *httptest.Server
}

// newFunctionalEnv starts three real HTTP servers and wires them together.
// pricingHandler and warehouseHandler define what each upstream returns.
func newFunctionalEnv(
	t *testing.T,
	pricingHandler http.HandlerFunc,
	warehouseHandler http.HandlerFunc,
) *functionalEnv {
	t.Helper()

	pricingBackend := httptest.NewServer(pricingHandler)
	warehouseBackend := httptest.NewServer(warehouseHandler)

	pricingURL, _ := url.Parse(pricingBackend.URL)
	warehouseURL, _ := url.Parse(warehouseBackend.URL)

	gw := NewGateway(pricingURL, warehouseURL)
	gatewayServer := httptest.NewServer(gw)

	t.Cleanup(func() {
		gatewayServer.Close()
		pricingBackend.Close()
		warehouseBackend.Close()
	})

	return &functionalEnv{
		gateway:          gatewayServer,
		pricingBackend:   pricingBackend,
		warehouseBackend: warehouseBackend,
	}
}

// get sends a real GET request to the gateway and returns the response.
func (e *functionalEnv) get(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(e.gateway.URL + path)
	require.NoError(t, err)
	return resp
}

// post sends a real POST request with a JSON body to the gateway.
func (e *functionalEnv) post(t *testing.T, path, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(e.gateway.URL+path, "application/json", strings.NewReader(body))
	require.NoError(t, err)
	return resp
}

// readBody reads and closes the response body, returning it as a string.
func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(b)
}

// ── /pricing routing ──────────────────────────────────────────────────────────

// TestFunctional_Pricing_CalculatePrice verifies the full round-trip for the
// primary pricing endpoint: gateway receives /pricing/calculate-price, strips
// the prefix, forwards to the pricing backend, and returns its response.
func TestFunctional_Pricing_CalculatePrice(t *testing.T) {
	env := newFunctionalEnv(t,
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/calculate-price", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"total_price":25000,"service_type":"REG"}`)) //nolint:errcheck
		},
		func(w http.ResponseWriter, _ *http.Request) {
			t.Error("warehouse backend must not be called for /pricing request")
		},
	)

	resp := env.get(t, "/pricing/calculate-price")
	body := readBody(t, resp)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body, "total_price")
	assert.Contains(t, body, "25000")
}

// TestFunctional_Pricing_POST verifies that POST requests to /pricing are
// forwarded correctly.
func TestFunctional_Pricing_POST(t *testing.T) {
	var receivedPath string
	env := newFunctionalEnv(t,
		func(w http.ResponseWriter, r *http.Request) {
			receivedPath = r.URL.Path
			assert.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok":true}`)) //nolint:errcheck
		},
		func(w http.ResponseWriter, _ *http.Request) {},
	)

	resp := env.post(t, "/pricing/calculate-price", `{"origin":"JKT","destination":"BDG","weight":2}`)
	resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "/calculate-price", receivedPath)
}

// TestFunctional_Pricing_ResponseHeadersPassthrough verifies that response
// headers set by the upstream are passed through to the client.
func TestFunctional_Pricing_ResponseHeadersPassthrough(t *testing.T) {
	env := newFunctionalEnv(t,
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Request-ID", "abc-123")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`)) //nolint:errcheck
		},
		func(w http.ResponseWriter, _ *http.Request) {},
	)

	resp := env.get(t, "/pricing/calculate-price")
	resp.Body.Close()

	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.Equal(t, "abc-123", resp.Header.Get("X-Request-ID"))
}

// ── /warehouse routing ────────────────────────────────────────────────────────

// TestFunctional_Warehouse_Receive verifies the full round-trip for
// POST /warehouse/receive.
func TestFunctional_Warehouse_Receive(t *testing.T) {
	var receivedPath string
	var receivedBody string
	env := newFunctionalEnv(t,
		func(w http.ResponseWriter, _ *http.Request) {
			t.Error("pricing backend must not be called for /warehouse request")
		},
		func(w http.ResponseWriter, r *http.Request) {
			receivedPath = r.URL.Path
			b, _ := io.ReadAll(r.Body)
			receivedBody = string(b)
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"message":"Barang berhasil masuk gudang!"}`)) //nolint:errcheck
		},
	)

	payload := `{"item_id":"BRG-001","name":"Laptop","quantity":10,"location":"Rak-A1"}`
	resp := env.post(t, "/warehouse/receive", payload)
	body := readBody(t, resp)

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "/receive", receivedPath)
	assert.Equal(t, payload, receivedBody)
	assert.Contains(t, body, "berhasil masuk")
}

// TestFunctional_Warehouse_CheckStock verifies GET /warehouse/check-stock
// including query parameter preservation.
func TestFunctional_Warehouse_CheckStock(t *testing.T) {
	var receivedQuery string
	env := newFunctionalEnv(t,
		func(w http.ResponseWriter, _ *http.Request) {},
		func(w http.ResponseWriter, r *http.Request) {
			receivedQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"item_id":"BRG-001","name":"Laptop","quantity":50,"location":"Rak-A1"}`)) //nolint:errcheck
		},
	)

	resp := env.get(t, "/warehouse/check-stock?id=BRG-001")
	body := readBody(t, resp)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "id=BRG-001", receivedQuery, "query string must be preserved")

	var item map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(body), &item))
	assert.Equal(t, "BRG-001", item["item_id"])
	assert.Equal(t, float64(50), item["quantity"])
}

// TestFunctional_Warehouse_Dispatch verifies POST /warehouse/dispatch routing.
func TestFunctional_Warehouse_Dispatch(t *testing.T) {
	var receivedPath string
	env := newFunctionalEnv(t,
		func(w http.ResponseWriter, _ *http.Request) {},
		func(w http.ResponseWriter, r *http.Request) {
			receivedPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"Barang berhasil dikeluarkan!"}`)) //nolint:errcheck
		},
	)

	resp := env.post(t, "/warehouse/dispatch", `{"item_id":"BRG-001","quantity":5}`)
	body := readBody(t, resp)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "/dispatch", receivedPath)
	assert.Contains(t, body, "berhasil dikeluarkan")
}

// ── 404 for unknown routes ────────────────────────────────────────────────────

// TestFunctional_UnknownRoute_Returns404 verifies that a path matching neither
// /pricing nor /warehouse returns 404 without hitting any backend.
func TestFunctional_UnknownRoute_Returns404(t *testing.T) {
	env := newFunctionalEnv(t,
		func(w http.ResponseWriter, _ *http.Request) {
			t.Error("pricing backend must not be called for unknown route")
		},
		func(w http.ResponseWriter, _ *http.Request) {
			t.Error("warehouse backend must not be called for unknown route")
		},
	)

	resp := env.get(t, "/unknown/path")
	body := readBody(t, resp)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Contains(t, body, "tidak ditemukan")
}

// TestFunctional_RootPath_Returns404 verifies that a bare "/" returns 404.
func TestFunctional_RootPath_Returns404(t *testing.T) {
	env := newFunctionalEnv(t,
		func(w http.ResponseWriter, _ *http.Request) {
			t.Error("pricing backend must not be called for /")
		},
		func(w http.ResponseWriter, _ *http.Request) {
			t.Error("warehouse backend must not be called for /")
		},
	)

	resp := env.get(t, "/")
	resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ── upstream error propagation ────────────────────────────────────────────────

// TestFunctional_Pricing_UpstreamReturns500 verifies that a 500 from the
// pricing backend is propagated to the client.
func TestFunctional_Pricing_UpstreamReturns500(t *testing.T) {
	env := newFunctionalEnv(t,
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"internal error"}`)) //nolint:errcheck
		},
		func(w http.ResponseWriter, _ *http.Request) {},
	)

	resp := env.get(t, "/pricing/calculate-price")
	resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

// TestFunctional_Warehouse_UpstreamReturns400 verifies that a 400 from the
// warehouse backend (e.g. out-of-stock) is propagated to the client.
func TestFunctional_Warehouse_UpstreamReturns400(t *testing.T) {
	env := newFunctionalEnv(t,
		func(w http.ResponseWriter, _ *http.Request) {},
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"stok barang tidak mencukupi"}`)) //nolint:errcheck
		},
	)

	resp := env.post(t, "/warehouse/dispatch", `{"item_id":"BRG-001","quantity":9999}`)
	body := readBody(t, resp)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, body, "stok")
}

// TestFunctional_Pricing_BackendUnreachable verifies that when the pricing
// backend is completely unreachable, the gateway returns 502 Bad Gateway.
func TestFunctional_Pricing_BackendUnreachable(t *testing.T) {
	// Start a backend and immediately shut it down so the port is closed
	gone := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	goneURL, _ := url.Parse(gone.URL)
	gone.Close()

	warehouse := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	t.Cleanup(warehouse.Close)
	warehouseURL, _ := url.Parse(warehouse.URL)

	gw := NewGateway(goneURL, warehouseURL)
	gwServer := httptest.NewServer(gw)
	t.Cleanup(gwServer.Close)

	resp, err := http.Get(gwServer.URL + "/pricing/calculate-price")
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
}

// ── full lifecycle ────────────────────────────────────────────────────────────

// TestFunctional_FullLifecycle exercises a realistic sequence of requests
// through the gateway: price calculation, then warehouse receive + check-stock.
func TestFunctional_FullLifecycle(t *testing.T) {
	// Pricing backend: returns a price
	pricingCalls := 0
	// Warehouse backend: simple in-memory store
	stock := map[string]int{}

	env := newFunctionalEnv(t,
		func(w http.ResponseWriter, r *http.Request) {
			pricingCalls++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"total_price":35000,"service_type":"YES"}`)) //nolint:errcheck
		},
		func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/receive":
				var payload map[string]interface{}
				json.NewDecoder(r.Body).Decode(&payload) //nolint:errcheck
				id, _ := payload["item_id"].(string)
				qty, _ := payload["quantity"].(float64)
				stock[id] += int(qty)
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"message":"ok"}`)) //nolint:errcheck
			case "/check-stock":
				id := r.URL.Query().Get("id")
				qty := stock[id]
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"item_id":"` + id + `","quantity":` + itoa(qty) + `}`)) //nolint:errcheck
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		},
	)

	// Step 1: Calculate price via /pricing
	r1 := env.get(t, "/pricing/calculate-price?origin=JKT&destination=BDG&weight=2")
	body1 := readBody(t, r1)
	require.Equal(t, http.StatusOK, r1.StatusCode)
	assert.Contains(t, body1, "35000")
	assert.Equal(t, 1, pricingCalls)

	// Step 2: Receive stock via /warehouse
	r2 := env.post(t, "/warehouse/receive", `{"item_id":"BRG-001","name":"Laptop","quantity":50,"location":"Rak-A1"}`)
	r2.Body.Close()
	require.Equal(t, http.StatusCreated, r2.StatusCode)

	// Step 3: Check stock via /warehouse
	r3 := env.get(t, "/warehouse/check-stock?id=BRG-001")
	body3 := readBody(t, r3)
	require.Equal(t, http.StatusOK, r3.StatusCode)
	assert.Contains(t, body3, "50")

	// Step 4: Unknown route returns 404
	r4 := env.get(t, "/tracking/TRK-001")
	r4.Body.Close()
	assert.Equal(t, http.StatusNotFound, r4.StatusCode)
}

// itoa is a minimal int-to-string helper to avoid importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}
