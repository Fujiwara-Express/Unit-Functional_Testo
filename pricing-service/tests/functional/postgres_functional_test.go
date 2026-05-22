// PostgreSQL functional tests for the pricing-service.
//
// Uses Testcontainers to spin up a real postgres:15-alpine container and
// exercises the full stack: PricingHandler → PricingService → PostgresRepository → DB.
// Docker must be running. No environment variables are required.
//
// Run with:
//
//	go test -v -tags postgres ./tests/functional/...
//
//go:build postgres

package functional_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	handler "github.com/fujiwara-express/pricing-service/internal/delivery/http"
	"github.com/fujiwara-express/pricing-service/internal/domain"
	"github.com/fujiwara-express/pricing-service/internal/repository"
	"github.com/fujiwara-express/pricing-service/internal/service"
)

var pgDB *sql.DB

func TestMain(m *testing.M) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "pricing_test",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}
	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start postgres container: %v\n", err)
		os.Exit(1)
	}
	defer pgContainer.Terminate(ctx) //nolint:errcheck

	host, err := pgContainer.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get container host: %v\n", err)
		os.Exit(1)
	}
	port, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get container port: %v\n", err)
		os.Exit(1)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=test password=test dbname=pricing_test sslmode=disable",
		host, port.Port(),
	)

	pgDB, err = sql.Open("postgres", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open db: %v\n", err)
		os.Exit(1)
	}
	defer pgDB.Close()

	if err := pgDB.PingContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to ping db: %v\n", err)
		os.Exit(1)
	}

	if err := applySchema(pgDB); err != nil {
		fmt.Fprintf(os.Stderr, "failed to apply schema: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// applySchema creates the tables and seeds the reference data used by all tests.
func applySchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS zones (
			id          SERIAL PRIMARY KEY,
			origin      VARCHAR(10),
			destination VARCHAR(10),
			zone_id     VARCHAR(10) NOT NULL
		);

		CREATE TABLE IF NOT EXISTS rates (
			id           SERIAL PRIMARY KEY,
			zone_id      VARCHAR(10),
			service_type VARCHAR(20),
			price_per_kg DECIMAL,
			min_weight   DECIMAL,
			max_weight   DECIMAL,
			max_length   DECIMAL,
			max_width    DECIMAL,
			max_height   DECIMAL
		);

		CREATE TABLE IF NOT EXISTS surcharges (
			id               SERIAL PRIMARY KEY,
			surcharge_type   VARCHAR(20),
			surcharge_value  DECIMAL
		);

		-- Seed reference data
		INSERT INTO zones (origin, destination, zone_id) VALUES
			('CGK', 'BDO', 'Z1'),
			('CGK', 'SBY', 'Z2'),
			('SBY', 'MKS', 'Z3');

		INSERT INTO rates (zone_id, service_type, price_per_kg, min_weight, max_weight, max_length, max_width, max_height) VALUES
			('Z1', 'REG',      10000, 1, 50,  100, 100, 100),
			('Z1', 'YES',      20000, 1, 30,   80,  80,  80),
			('Z2', 'REG',      15000, 1, 50,  100, 100, 100),
			('Z3', 'REG',      18000, 1, 50,  100, 100, 100);

		INSERT INTO surcharges (surcharge_type, surcharge_value) VALUES
			('OVERSIZE', 50000);
	`)
	return err
}

// newPGServer wires the PostgresRepository into the full stack.
func newPGServer(t *testing.T) *httptest.Server {
	t.Helper()
	repo := repository.NewPostgresRepository(pgDB)
	svc := service.NewPricingService(repo)
	h := handler.NewPricingHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("/calculate-price", h.CalculatePrice)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func pgPost(t *testing.T, srv *httptest.Server, req domain.CalculatePriceRequest) *http.Response {
	t.Helper()
	b, err := json.Marshal(req)
	require.NoError(t, err)
	resp, err := http.Post(srv.URL+"/calculate-price", "application/json", bytes.NewReader(b))
	require.NoError(t, err)
	return resp
}

func pgDecode(t *testing.T, resp *http.Response) domain.CalculatePriceResponse {
	t.Helper()
	defer resp.Body.Close()
	var out domain.CalculatePriceResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	return out
}

// ── PostgreSQL functional tests ───────────────────────────────────────────────

// TestPG_CalculatePrice_NormalPackage verifies the standard case against a real DB.
//
// CGK→BDO/Z1/REG: price_per_kg=10000, min_weight=1
// Actual 5 kg, volumetric (30×20×20)/6000 = 2 kg → chargeable = 5 kg
// Total = 5 × 10000 = 50000, no oversize
func TestPG_CalculatePrice_NormalPackage(t *testing.T) {
	srv := newPGServer(t)

	resp := pgPost(t, srv, domain.CalculatePriceRequest{
		Origin: "CGK", Destination: "BDO", ServiceType: "REG",
		Weight: 5, Length: 30, Width: 20, Height: 20,
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	out := pgDecode(t, resp)
	assert.Equal(t, "REG", out.ServiceType)
	assert.InDelta(t, 5.0, out.ChargeableWeight, 0.001)
	assert.InDelta(t, 50000.0, out.Price, 0.001)
	assert.InDelta(t, 0.0, out.Breakdown.OversizeSurcharge, 0.001)
}

// TestPG_CalculatePrice_VolumetricDominates verifies volumetric weight logic
// against a real DB.
//
// Volumetric: (60×60×60)/6000 = 36 kg > actual 2 kg → chargeable = 36 kg
// Total = 36 × 10000 = 360000
func TestPG_CalculatePrice_VolumetricDominates(t *testing.T) {
	srv := newPGServer(t)

	resp := pgPost(t, srv, domain.CalculatePriceRequest{
		Origin: "CGK", Destination: "BDO", ServiceType: "REG",
		Weight: 2, Length: 60, Width: 60, Height: 60,
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	out := pgDecode(t, resp)
	assert.InDelta(t, 36.0, out.ChargeableWeight, 0.001)
	assert.InDelta(t, 360000.0, out.Price, 0.001)
	assert.InDelta(t, 0.0, out.Breakdown.OversizeSurcharge, 0.001)
}

// TestPG_CalculatePrice_OversizeByLength verifies that exceeding MaxLength
// fetches the OVERSIZE surcharge from the DB and adds it to the total.
//
// Length 150 > MaxLength 100 → surcharge 50000 applied.
// Volumetric: (150×50×50)/6000 = 62.5 kg → chargeable = 62.5 kg
// Total = 62.5 × 10000 + 50000 = 675000
func TestPG_CalculatePrice_OversizeByLength(t *testing.T) {
	srv := newPGServer(t)

	resp := pgPost(t, srv, domain.CalculatePriceRequest{
		Origin: "CGK", Destination: "BDO", ServiceType: "REG",
		Weight: 10, Length: 150, Width: 50, Height: 50,
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	out := pgDecode(t, resp)
	assert.InDelta(t, 62.5, out.ChargeableWeight, 0.001)
	assert.InDelta(t, 50000.0, out.Breakdown.OversizeSurcharge, 0.001)
	assert.InDelta(t, 675000.0, out.Price, 0.001)
}

// TestPG_CalculatePrice_OversizeByWeight verifies that exceeding MaxWeight
// triggers the oversize surcharge from the DB.
func TestPG_CalculatePrice_OversizeByWeight(t *testing.T) {
	srv := newPGServer(t)

	resp := pgPost(t, srv, domain.CalculatePriceRequest{
		Origin: "CGK", Destination: "BDO", ServiceType: "REG",
		Weight: 55, Length: 30, Width: 20, Height: 20, // weight > MaxWeight 50
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	out := pgDecode(t, resp)
	assert.InDelta(t, 50000.0, out.Breakdown.OversizeSurcharge, 0.001)
}

// TestPG_CalculatePrice_DifferentZone verifies that a different origin/destination
// pair (CGK→SBY/Z2) uses the correct rate from the DB (15000/kg).
//
// Chargeable = 5 kg, Total = 5 × 15000 = 75000
func TestPG_CalculatePrice_DifferentZone(t *testing.T) {
	srv := newPGServer(t)

	resp := pgPost(t, srv, domain.CalculatePriceRequest{
		Origin: "CGK", Destination: "SBY", ServiceType: "REG",
		Weight: 5, Length: 30, Width: 20, Height: 20,
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	out := pgDecode(t, resp)
	assert.InDelta(t, 75000.0, out.Price, 0.001,
		"CGK→SBY uses Z2 rate (15000/kg), total should be 75000")
}

// TestPG_CalculatePrice_YESServiceType verifies that the YES service type uses
// its own rate (20000/kg, MaxLength 80) from the DB.
//
// Chargeable = 5 kg, Total = 5 × 20000 = 100000
func TestPG_CalculatePrice_YESServiceType(t *testing.T) {
	srv := newPGServer(t)

	resp := pgPost(t, srv, domain.CalculatePriceRequest{
		Origin: "CGK", Destination: "BDO", ServiceType: "YES",
		Weight: 5, Length: 30, Width: 20, Height: 20,
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	out := pgDecode(t, resp)
	assert.Equal(t, "YES", out.ServiceType)
	assert.InDelta(t, 100000.0, out.Price, 0.001,
		"YES service uses 20000/kg rate, total should be 100000")
}

// TestPG_CalculatePrice_YESServiceType_OversizeByLength verifies that the YES
// service type's tighter MaxLength (80) triggers oversize for a 90 cm package.
func TestPG_CalculatePrice_YESServiceType_OversizeByLength(t *testing.T) {
	srv := newPGServer(t)

	resp := pgPost(t, srv, domain.CalculatePriceRequest{
		Origin: "CGK", Destination: "BDO", ServiceType: "YES",
		Weight: 5, Length: 90, Width: 20, Height: 20, // 90 > MaxLength 80 for YES
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	out := pgDecode(t, resp)
	assert.InDelta(t, 50000.0, out.Breakdown.OversizeSurcharge, 0.001,
		"YES service MaxLength is 80, so 90 cm should trigger oversize")
}

// TestPG_CalculatePrice_ZoneNotFound verifies that an unknown route returns
// HTTP 500 with the zone-not-found error.
func TestPG_CalculatePrice_ZoneNotFound(t *testing.T) {
	srv := newPGServer(t)

	resp := pgPost(t, srv, domain.CalculatePriceRequest{
		Origin: "XXX", Destination: "YYY", ServiceType: "REG",
		Weight: 5, Length: 30, Width: 20, Height: 20,
	})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

// TestPG_CalculatePrice_RateNotFound verifies that a valid zone with no
// matching service type returns HTTP 500 with the rate-not-found error.
func TestPG_CalculatePrice_RateNotFound(t *testing.T) {
	srv := newPGServer(t)

	resp := pgPost(t, srv, domain.CalculatePriceRequest{
		Origin: "CGK", Destination: "BDO", ServiceType: "SAME_DAY", // not seeded
		Weight: 5, Length: 30, Width: 20, Height: 20,
	})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

// TestPG_FullLifecycle exercises multiple routes and service types in sequence
// to verify the full stack against a real PostgreSQL database.
func TestPG_FullLifecycle(t *testing.T) {
	srv := newPGServer(t)

	cases := []struct {
		name           string
		req            domain.CalculatePriceRequest
		wantStatus     int
		wantPrice      float64
		wantOversize   float64
	}{
		{
			name: "CGK→BDO REG normal",
			req: domain.CalculatePriceRequest{
				Origin: "CGK", Destination: "BDO", ServiceType: "REG",
				Weight: 5, Length: 30, Width: 20, Height: 20,
			},
			wantStatus: http.StatusOK, wantPrice: 50000, wantOversize: 0,
		},
		{
			name: "CGK→SBY REG different zone",
			req: domain.CalculatePriceRequest{
				Origin: "CGK", Destination: "SBY", ServiceType: "REG",
				Weight: 5, Length: 30, Width: 20, Height: 20,
			},
			wantStatus: http.StatusOK, wantPrice: 75000, wantOversize: 0,
		},
		{
			name: "CGK→BDO YES premium rate",
			req: domain.CalculatePriceRequest{
				Origin: "CGK", Destination: "BDO", ServiceType: "YES",
				Weight: 5, Length: 30, Width: 20, Height: 20,
			},
			wantStatus: http.StatusOK, wantPrice: 100000, wantOversize: 0,
		},
		{
			name: "CGK→BDO REG oversize",
			req: domain.CalculatePriceRequest{
				Origin: "CGK", Destination: "BDO", ServiceType: "REG",
				Weight: 10, Length: 150, Width: 50, Height: 50,
			},
			wantStatus: http.StatusOK, wantPrice: 675000, wantOversize: 50000,
		},
		{
			name: "unknown zone → 500",
			req: domain.CalculatePriceRequest{
				Origin: "ZZZ", Destination: "QQQ", ServiceType: "REG",
				Weight: 5, Length: 30, Width: 20, Height: 20,
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := pgPost(t, srv, tc.req)

			if tc.wantStatus != http.StatusOK {
				resp.Body.Close()
				assert.Equal(t, tc.wantStatus, resp.StatusCode)
				return
			}

			require.Equal(t, http.StatusOK, resp.StatusCode)
			out := pgDecode(t, resp)
			assert.InDelta(t, tc.wantPrice, out.Price, 0.001)
			assert.InDelta(t, tc.wantOversize, out.Breakdown.OversizeSurcharge, 0.001)
		})
	}
}
