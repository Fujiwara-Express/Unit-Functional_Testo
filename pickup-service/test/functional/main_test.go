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

	"github.com/pickup-service/internal/client"
	httphandler "github.com/pickup-service/internal/handler/http"
	"github.com/pickup-service/internal/handler/http/middleware"
	kafkahandler "github.com/pickup-service/internal/handler/kafka"
	"github.com/pickup-service/internal/repository"
	"github.com/pickup-service/internal/service"
)

var (
	testDB       *sql.DB
	testServer   *httptest.Server
	notifStub    *notificationStubServer
	deliveryStub *deliveryStubServer
	kafkaHandler *kafkahandler.OrderCreatedHandler
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start Postgres container
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "pickup_test",
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
	defer func() { _ = pgContainer.Terminate(ctx) }()

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
		"host=%s port=%s user=test password=test dbname=pickup_test sslmode=disable",
		host, port.Port(),
	)

	testDB, err = sql.Open("postgres", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open db: %v\n", err)
		os.Exit(1)
	}
	defer testDB.Close()

	if err := testDB.PingContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to ping db: %v\n", err)
		os.Exit(1)
	}

	if err := applySchema(testDB); err != nil {
		fmt.Fprintf(os.Stderr, "failed to apply schema: %v\n", err)
		os.Exit(1)
	}

	// Build stubs
	notifStub = newNotificationStub()
	defer notifStub.Close()
	deliveryStub = newDeliveryStub()
	defer deliveryStub.Close()

	// Build shared SUT with a no-op tracking client (for HTTP-only tests)
	repo := repository.NewPickupRepository(testDB)
	noopTracker := client.NewKafkaTrackingClient("noop")
	svc := service.NewPickupService(
		repo,
		client.New(deliveryStub.URL()),
		noopTracker,
		client.NewHTTPNotificationClient(notifStub.URL()),
	)
	handler := httphandler.NewPickupHandler(svc)
	kafkaHandler = kafkahandler.NewOrderCreatedHandler(svc)

	mux := buildMux(handler)
	testServer = httptest.NewServer(mux)
	defer testServer.Close()

	os.Exit(m.Run())
}

// buildMux registers all six pickup routes with RequestID and Auth middleware.
func buildMux(h *httphandler.PickupHandler) *http.ServeMux {
	mux := http.NewServeMux()
	chain := func(next http.HandlerFunc) http.Handler {
		return middleware.RequestID(middleware.Auth(http.HandlerFunc(next)))
	}
	mux.Handle("POST /pickups", chain(h.RequestPickup))
	mux.Handle("POST /pickups/{pickup_id}/assign", chain(h.AssignCourier))
	mux.Handle("POST /pickups/{pickup_id}/status", chain(h.UpdatePickupStatus))
	mux.Handle("GET /pickups/{pickup_id}", chain(h.GetPickup))
	mux.Handle("GET /pickups", chain(h.ListPickups))
	mux.Handle("POST /pickups/{pickup_id}/cancel", chain(h.CancelPickup))
	return mux
}

// applySchema creates the pickups and pickup_attempts tables.
func applySchema(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS pickups (
    pickup_id             TEXT        PRIMARY KEY,
    order_id              TEXT        NOT NULL,
    user_id               TEXT        NOT NULL,
    courier_id            TEXT        NOT NULL DEFAULT '',
    status                TEXT        NOT NULL,
    pickup_address        TEXT        NOT NULL,
    pickup_city_code      TEXT        NOT NULL,
    contact_name          TEXT        NOT NULL,
    contact_phone         TEXT        NOT NULL,
    attempt_count         INTEGER     NOT NULL DEFAULT 0,
    estimated_pickup_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pickup_attempts (
    attempt_id  TEXT        PRIMARY KEY,
    pickup_id   TEXT        NOT NULL REFERENCES pickups(pickup_id),
    reason      TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`
	_, err := db.Exec(schema)
	return err
}
