package functional_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	httphandler "github.com/report-and-analytics/internal/handler/http"
	"github.com/report-and-analytics/internal/handler/http/middleware"
	"github.com/report-and-analytics/internal/repository"
	"github.com/report-and-analytics/internal/service"
)

var (
	testDB     *sql.DB
	testServer *httptest.Server
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start Postgres container (data warehouse stand-in)
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "analytics_test",
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
		"host=%s port=%s user=test password=test dbname=analytics_test sslmode=disable",
		host, port.Port(),
	)

	testDB, err = sql.Open("postgres", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open db: %v\n", err)
		os.Exit(1)
	}
	if err := testDB.PingContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to ping db: %v\n", err)
		os.Exit(1)
	}

	if err := applySchema(testDB); err != nil {
		fmt.Fprintf(os.Stderr, "failed to apply schema: %v\n", err)
		os.Exit(1)
	}

	// Wire SUT
	repo := repository.NewReportRepository(testDB)
	svc := service.NewReportService(repo)
	handler := httphandler.NewReportHandler(svc)

	mux := buildMux(handler)
	testServer = httptest.NewServer(mux)
	defer testServer.Close()

	os.Exit(m.Run())
}

// applySchema creates the data warehouse tables used by the report queries.
func applySchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS fact_orders (
			order_id     TEXT PRIMARY KEY,
			order_date   DATE NOT NULL,
			hub_id       TEXT NOT NULL DEFAULT '',
			status       TEXT NOT NULL,
			service_type TEXT NOT NULL DEFAULT '',
			period_type  TEXT NOT NULL DEFAULT '',
			revenue      BIGINT NOT NULL DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS fact_deliveries (
			delivery_id         TEXT PRIMARY KEY,
			courier_id          TEXT NOT NULL,
			period_type         TEXT NOT NULL,
			status              TEXT NOT NULL,
			delivery_time_hours FLOAT NOT NULL DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS fact_hub_performance (
			id                      SERIAL PRIMARY KEY,
			hub_id                  TEXT NOT NULL,
			period_type             TEXT NOT NULL,
			inbound_count           BIGINT NOT NULL DEFAULT 0,
			outbound_count          BIGINT NOT NULL DEFAULT 0,
			dispatched_count        BIGINT NOT NULL DEFAULT 0,
			avg_dwell_time_hours    FLOAT NOT NULL DEFAULT 0,
			capacity_utilization_pct FLOAT NOT NULL DEFAULT 0
		);
	`)
	return err
}

// buildMux registers all four report routes wrapped with RequestID and Auth middleware.
func buildMux(h *httphandler.ReportHandler) *http.ServeMux {
	mux := http.NewServeMux()
	chain := func(next http.HandlerFunc) http.Handler {
		return middleware.RequestID(middleware.Auth(next))
	}
	mux.Handle("GET /reports/orders", chain(h.GetOrderReport))
	mux.Handle("GET /reports/delivery-performance", chain(h.GetDeliveryPerformanceReport))
	mux.Handle("GET /reports/revenue", chain(h.GetRevenueReport))
	mux.Handle("GET /reports/hub-performance", chain(h.GetHubPerformanceReport))
	return mux
}
