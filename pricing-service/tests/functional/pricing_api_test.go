// Package functional_test contains end-to-end functional tests for the pricing-service.
//
// Two test suites live here:
//
//  1. In-memory suite (always runs) — wires the real PricingHandler, real
//     PricingService, and the built-in memoryRepository together.  No database
//     or Docker required.  The memory repo has one hardcoded zone (CGK→BDO/Z1)
//     and one rate (Z1/REG: 10 000/kg, max dims 100×100×100, max weight 50).
//
//  2. PostgreSQL suite (build tag "postgres") — uses Testcontainers to spin up
//     a real postgres:15-alpine container and exercises the PostgresRepository.
//     Run with: go test -v -tags postgres ./tests/functional/...
package functional_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	handler "github.com/fujiwara-express/pricing-service/internal/delivery/http"
	"github.com/fujiwara-express/pricing-service/internal/domain"
	"github.com/fujiwara-express/pricing-service/internal/repository"
	"github.com/fujiwara-express/pricing-service/internal/service"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// newMemoryServer wires the full in-memory stack and returns an httptest.Server.
func newMemoryServer(t *testing.T) *httptest.Server {
	t.Helper()
	repo := repository.NewMemoryRepository()
	svc := service.NewPricingService(repo)
	h := handler.NewPricingHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("/calculate-price", h.CalculatePrice)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// postPrice sends a POST /calculate-price request and returns the response.
func postPrice(t *testing.T, srv *httptest.Server, req domain.CalculatePriceRequest) *http.Response {
	t.Helper()
	b, err := json.Marshal(req)
	require.NoError(t, err)
	resp, err := http.Post(srv.URL+"/calculate-price", "application/json", bytes.NewReader(b))
	require.NoError(t, err)
	return resp
}

// decodePrice decodes the response body into CalculatePriceResponse.
func decodePrice(t *testing.T, resp *http.Response) domain.CalculatePriceResponse {
	t.Helper()
	defer resp.Body.Close()
	var out domain.CalculatePriceResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	return out
}

// baseReq returns a minimal valid request for the memory repo's seeded data.
// CGK→BDO, REG, 5 kg, 30×20×20 cm (all within limits → no oversize surcharge).
func baseReq() domain.CalculatePriceRequest {
	return domain.CalculatePriceRequest{
		Origin:      "CGK",
		Destination: "BDO",
		ServiceType: "REG",
		Weight:      5,
		Length:      30,
		Width:       20,
		Height:      20,
	}
}

// ── POST /calculate-price — happy path ───────────────────────────────────────

// TestFunctional_CalculatePrice_NormalPackage verifies the standard case:
// actual weight > volumetric weight, no oversize, correct total price.
//
// Volumetric: (30×20×20)/6000 = 2 kg  →  chargeable = max(5, 2) = 5 kg
// Base rate:  5 × 10 000 = 50 000
// Oversize:   0
// Total:      50 000
func TestFunctional_CalculatePrice_NormalPackage(t *testing.T) {
	srv := newMemoryServer(t)

	resp := postPrice(t, srv, baseReq())
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	out := decodePrice(t, resp)
	assert.Equal(t, "REG", out.ServiceType)
	assert.InDelta(t, 2.0, out.VolumetricWeight, 0.001)
	assert.InDelta(t, 5.0, out.ChargeableWeight, 0.001)
	assert.InDelta(t, 50000.0, out.Breakdown.BaseRate, 0.001)
	assert.InDelta(t, 0.0, out.Breakdown.OversizeSurcharge, 0.001)
	assert.InDelta(t, 50000.0, out.Price, 0.001)
}

// TestFunctional_CalculatePrice_VolumetricDominates verifies that when the
// volumetric weight exceeds the actual weight, the volumetric weight is used
// as the chargeable weight.
//
// Volumetric: (60×60×60)/6000 = 36 kg  →  chargeable = max(5, 36) = 36 kg
// Base rate:  36 × 10 000 = 360 000
// Oversize:   0  (dims 60 < max 100)
// Total:      360 000
func TestFunctional_CalculatePrice_VolumetricDominates(t *testing.T) {
	srv := newMemoryServer(t)

	req := baseReq()
	req.Weight = 5
	req.Length = 60
	req.Width = 60
	req.Height = 60

	resp := postPrice(t, srv, req)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	out := decodePrice(t, resp)
	assert.InDelta(t, 36.0, out.VolumetricWeight, 0.001)
	assert.InDelta(t, 36.0, out.ChargeableWeight, 0.001)
	assert.InDelta(t, 360000.0, out.Price, 0.001)
	assert.InDelta(t, 0.0, out.Breakdown.OversizeSurcharge, 0.001)
}

// TestFunctional_CalculatePrice_MinWeightApplied verifies that when both actual
// and volumetric weights are below the minimum weight (1 kg), the minimum is
// used as the chargeable weight.
//
// Actual: 0.3 kg, Volumetric: (10×10×10)/6000 ≈ 0.167 kg
// Chargeable = max(max(0.3, 0.167), 1) = 1 kg
// Base rate: 1 × 10 000 = 10 000
func TestFunctional_CalculatePrice_MinWeightApplied(t *testing.T) {
	srv := newMemoryServer(t)

	req := baseReq()
	req.Weight = 0.3
	req.Length = 10
	req.Width = 10
	req.Height = 10

	resp := postPrice(t, srv, req)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	out := decodePrice(t, resp)
	assert.InDelta(t, 1.0, out.ChargeableWeight, 0.001)
	assert.InDelta(t, 10000.0, out.Price, 0.001)
}

// TestFunctional_CalculatePrice_OversizeByLength verifies that a package
// exceeding MaxLength triggers the oversize surcharge.
//
// Length 150 > MaxLength 100 → oversize surcharge 50 000 applied.
// Volumetric: (150×50×50)/6000 = 62.5 kg  →  chargeable = max(10, 62.5) = 62.5
// Base rate:  62.5 × 10 000 = 625 000
// Oversize:   50 000
// Total:      675 000
func TestFunctional_CalculatePrice_OversizeByLength(t *testing.T) {
	srv := newMemoryServer(t)

	req := baseReq()
	req.Weight = 10
	req.Length = 150
	req.Width = 50
	req.Height = 50

	resp := postPrice(t, srv, req)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	out := decodePrice(t, resp)
	assert.InDelta(t, 62.5, out.ChargeableWeight, 0.001)
	assert.InDelta(t, 50000.0, out.Breakdown.OversizeSurcharge, 0.001)
	assert.InDelta(t, 675000.0, out.Price, 0.001)
}

// TestFunctional_CalculatePrice_OversizeByWidth verifies that exceeding MaxWidth
// also triggers the oversize surcharge.
func TestFunctional_CalculatePrice_OversizeByWidth(t *testing.T) {
	srv := newMemoryServer(t)

	req := baseReq()
	req.Width = 120 // > MaxWidth 100

	resp := postPrice(t, srv, req)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	out := decodePrice(t, resp)
	assert.InDelta(t, 50000.0, out.Breakdown.OversizeSurcharge, 0.001)
	assert.Greater(t, out.Price, out.Breakdown.BaseRate)
}

// TestFunctional_CalculatePrice_OversizeByHeight verifies that exceeding
// MaxHeight triggers the oversize surcharge.
func TestFunctional_CalculatePrice_OversizeByHeight(t *testing.T) {
	srv := newMemoryServer(t)

	req := baseReq()
	req.Height = 110 // > MaxHeight 100

	resp := postPrice(t, srv, req)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	out := decodePrice(t, resp)
	assert.InDelta(t, 50000.0, out.Breakdown.OversizeSurcharge, 0.001)
}

// TestFunctional_CalculatePrice_OversizeByWeight verifies that exceeding
// MaxWeight triggers the oversize surcharge.
func TestFunctional_CalculatePrice_OversizeByWeight(t *testing.T) {
	srv := newMemoryServer(t)

	req := baseReq()
	req.Weight = 55 // > MaxWeight 50

	resp := postPrice(t, srv, req)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	out := decodePrice(t, resp)
	assert.InDelta(t, 50000.0, out.Breakdown.OversizeSurcharge, 0.001)
}

// TestFunctional_CalculatePrice_ExactlyAtLimit verifies that a package exactly
// at the dimension limits does NOT trigger the oversize surcharge.
func TestFunctional_CalculatePrice_ExactlyAtLimit(t *testing.T) {
	srv := newMemoryServer(t)

	req := baseReq()
	req.Weight = 50  // == MaxWeight
	req.Length = 100 // == MaxLength
	req.Width = 100  // == MaxWidth
	req.Height = 100 // == MaxHeight

	resp := postPrice(t, srv, req)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	out := decodePrice(t, resp)
	assert.InDelta(t, 0.0, out.Breakdown.OversizeSurcharge, 0.001,
		"package exactly at limits must not incur oversize surcharge")
}

// TestFunctional_CalculatePrice_ResponseFields verifies that all expected JSON
// fields are present in the response.
func TestFunctional_CalculatePrice_ResponseFields(t *testing.T) {
	srv := newMemoryServer(t)

	resp := postPrice(t, srv, baseReq())
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	var raw map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&raw))

	for _, field := range []string{"service_type", "price", "volumetric_weight", "chargeable_weight", "breakdown"} {
		assert.Contains(t, raw, field, "response must contain field %q", field)
	}

	breakdown, ok := raw["breakdown"].(map[string]interface{})
	require.True(t, ok, "breakdown must be an object")
	assert.Contains(t, breakdown, "base_rate")
	assert.Contains(t, breakdown, "oversize_surcharge")
}

// ── POST /calculate-price — error paths ──────────────────────────────────────

// TestFunctional_CalculatePrice_ZoneNotFound verifies that an unknown
// origin/destination pair returns HTTP 500 with the zone-not-found error.
func TestFunctional_CalculatePrice_ZoneNotFound(t *testing.T) {
	srv := newMemoryServer(t)

	req := baseReq()
	req.Origin = "SBY"
	req.Destination = "MKS" // not seeded in memory repo

	resp := postPrice(t, srv, req)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

// TestFunctional_CalculatePrice_InvalidJSON verifies that a malformed request
// body returns HTTP 400.
func TestFunctional_CalculatePrice_InvalidJSON(t *testing.T) {
	srv := newMemoryServer(t)

	resp, err := http.Post(srv.URL+"/calculate-price", "application/json",
		bytes.NewReader([]byte("{not valid json")))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestFunctional_CalculatePrice_WrongMethod verifies that GET requests to
// /calculate-price return HTTP 405.
func TestFunctional_CalculatePrice_WrongMethod(t *testing.T) {
	srv := newMemoryServer(t)

	resp, err := http.Get(srv.URL + "/calculate-price")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

// ── full lifecycle ────────────────────────────────────────────────────────────

// TestFunctional_FullLifecycle exercises a sequence of price calculations that
// covers the normal, volumetric-dominant, and oversize scenarios in one test.
func TestFunctional_FullLifecycle(t *testing.T) {
	srv := newMemoryServer(t)

	cases := []struct {
		name              string
		req               domain.CalculatePriceRequest
		wantPrice         float64
		wantOversize      float64
		wantChargeable    float64
	}{
		{
			name:           "normal package",
			req:            baseReq(), // 5 kg, 30×20×20
			wantPrice:      50000,
			wantOversize:   0,
			wantChargeable: 5,
		},
		{
			name: "volumetric dominates",
			req: domain.CalculatePriceRequest{
				Origin: "CGK", Destination: "BDO", ServiceType: "REG",
				Weight: 2, Length: 60, Width: 60, Height: 60,
			},
			// Volumetric: (60×60×60)/6000 = 36 kg
			wantPrice:      360000,
			wantOversize:   0,
			wantChargeable: 36,
		},
		{
			name: "oversize by length",
			req: domain.CalculatePriceRequest{
				Origin: "CGK", Destination: "BDO", ServiceType: "REG",
				Weight: 10, Length: 150, Width: 50, Height: 50,
			},
			// Volumetric: (150×50×50)/6000 = 62.5 kg
			wantPrice:      675000,
			wantOversize:   50000,
			wantChargeable: 62.5,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := postPrice(t, srv, tc.req)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			out := decodePrice(t, resp)
			assert.InDelta(t, tc.wantChargeable, out.ChargeableWeight, 0.001)
			assert.InDelta(t, tc.wantOversize, out.Breakdown.OversizeSurcharge, 0.001)
			assert.InDelta(t, tc.wantPrice, out.Price, 0.001)
		})
	}
}
