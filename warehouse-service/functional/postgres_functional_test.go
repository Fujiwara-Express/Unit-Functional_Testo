// PostgreSQL functional tests for the warehouse-service.
//
// These tests use Testcontainers to spin up a real postgres:15-alpine container
// and exercise the full stack (handler → service → PostgresRepository → DB).
// Docker must be running. No environment variables are required.
//
// Run with the "postgres" build tag:
//
//	go test -v -tags postgres ./functional/...
//
//go:build postgres

package functional

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

	handler "github.com/fujiwara-express/warehouse-service/internal/delivery/http"
	"github.com/fujiwara-express/warehouse-service/internal/domain"
	"github.com/fujiwara-express/warehouse-service/internal/repository"
	"github.com/fujiwara-express/warehouse-service/internal/service"
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
			"POSTGRES_DB":       "warehouse_test",
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
		"host=%s port=%s user=test password=test dbname=warehouse_test sslmode=disable",
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

func applySchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			item_id  VARCHAR(50) PRIMARY KEY,
			name     VARCHAR(100) NOT NULL,
			quantity INT NOT NULL DEFAULT 0,
			location VARCHAR(50)
		)
	`)
	return err
}

func truncate(t *testing.T) {
	t.Helper()
	_, err := pgDB.Exec("TRUNCATE TABLE items")
	require.NoError(t, err)
}

// newPGTestServer wires the real PostgresRepository into the full stack.
func newPGTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	repo := repository.NewPostgresRepository(pgDB)
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

func pgPost(t *testing.T, srv *httptest.Server, path string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	resp, err := http.Post(srv.URL+path, "application/json", bytes.NewReader(b))
	require.NoError(t, err)
	return resp
}

func pgGet(t *testing.T, srv *httptest.Server, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(srv.URL + path)
	require.NoError(t, err)
	return resp
}

// TestPG_ReceiveItem_PersistsRow verifies that a received item is stored in
// the PostgreSQL items table with the correct fields.
func TestPG_ReceiveItem_PersistsRow(t *testing.T) {
	truncate(t)
	srv := newPGTestServer(t)

	resp := pgPost(t, srv, "/receive", domain.Item{
		ID: "BRG-PG-001", Name: "Laptop", Quantity: 50, Location: "Rak-A1",
	})
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Query DB directly
	var item domain.Item
	err := pgDB.QueryRow(
		"SELECT item_id, name, quantity, location FROM items WHERE item_id = $1",
		"BRG-PG-001",
	).Scan(&item.ID, &item.Name, &item.Quantity, &item.Location)
	require.NoError(t, err)

	assert.Equal(t, "BRG-PG-001", item.ID)
	assert.Equal(t, "Laptop", item.Name)
	assert.Equal(t, 50, item.Quantity)
	assert.Equal(t, "Rak-A1", item.Location)
}

// TestPG_ReceiveItem_Upsert verifies that receiving the same item twice
// accumulates the quantity via the ON CONFLICT upsert.
func TestPG_ReceiveItem_Upsert(t *testing.T) {
	truncate(t)
	srv := newPGTestServer(t)

	pgPost(t, srv, "/receive", domain.Item{ID: "BRG-PG-002", Name: "Mouse", Quantity: 20, Location: "Rak-B1"}).Body.Close()
	pgPost(t, srv, "/receive", domain.Item{ID: "BRG-PG-002", Name: "Mouse", Quantity: 10, Location: ""}).Body.Close()

	var qty int
	err := pgDB.QueryRow("SELECT quantity FROM items WHERE item_id = $1", "BRG-PG-002").Scan(&qty)
	require.NoError(t, err)
	assert.Equal(t, 30, qty)
}

// TestPG_DispatchItem_UpdatesRow verifies that dispatching stock reduces the
// quantity in the PostgreSQL items table.
func TestPG_DispatchItem_UpdatesRow(t *testing.T) {
	truncate(t)
	srv := newPGTestServer(t)

	pgPost(t, srv, "/receive", domain.Item{ID: "BRG-PG-003", Name: "Monitor", Quantity: 40, Location: "Rak-C1"}).Body.Close()

	resp := pgPost(t, srv, "/dispatch", domain.StockRequest{ItemID: "BRG-PG-003", Quantity: 15})
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var qty int
	err := pgDB.QueryRow("SELECT quantity FROM items WHERE item_id = $1", "BRG-PG-003").Scan(&qty)
	require.NoError(t, err)
	assert.Equal(t, 25, qty)
}

// TestPG_DispatchItem_OutOfStock verifies that dispatching more than available
// returns HTTP 400 and leaves the DB row unchanged.
func TestPG_DispatchItem_OutOfStock(t *testing.T) {
	truncate(t)
	srv := newPGTestServer(t)

	pgPost(t, srv, "/receive", domain.Item{ID: "BRG-PG-004", Name: "Keyboard", Quantity: 5, Location: "Rak-D1"}).Body.Close()

	resp := pgPost(t, srv, "/dispatch", domain.StockRequest{ItemID: "BRG-PG-004", Quantity: 100})
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var qty int
	err := pgDB.QueryRow("SELECT quantity FROM items WHERE item_id = $1", "BRG-PG-004").Scan(&qty)
	require.NoError(t, err)
	assert.Equal(t, 5, qty, "quantity must be unchanged after failed dispatch")
}

// TestPG_CheckStock_ReturnsCorrectData verifies that GET /check-stock returns
// the data that was persisted in PostgreSQL.
func TestPG_CheckStock_ReturnsCorrectData(t *testing.T) {
	truncate(t)
	srv := newPGTestServer(t)

	pgPost(t, srv, "/receive", domain.Item{ID: "BRG-PG-005", Name: "SSD 512GB", Quantity: 100, Location: "Rak-E1"}).Body.Close()

	resp := pgGet(t, srv, "/check-stock?id=BRG-PG-005")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var item domain.Item
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&item))
	assert.Equal(t, "BRG-PG-005", item.ID)
	assert.Equal(t, "SSD 512GB", item.Name)
	assert.Equal(t, 100, item.Quantity)
	assert.Equal(t, "Rak-E1", item.Location)
}

// TestPG_CheckStock_NotFound verifies that querying a non-existent item
// returns HTTP 404.
func TestPG_CheckStock_NotFound(t *testing.T) {
	truncate(t)
	srv := newPGTestServer(t)

	resp := pgGet(t, srv, "/check-stock?id=BRG-GHOST")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestPG_FullLifecycle exercises the complete lifecycle against PostgreSQL:
// Receive → CheckStock → Dispatch → CheckStock.
func TestPG_FullLifecycle(t *testing.T) {
	truncate(t)
	srv := newPGTestServer(t)

	// 1. Receive 60 units
	pgPost(t, srv, "/receive", domain.Item{ID: "BRG-PG-100", Name: "RAM 32GB", Quantity: 60, Location: "Rak-Z1"}).Body.Close()

	// 2. Verify via API
	r1 := pgGet(t, srv, "/check-stock?id=BRG-PG-100")
	var item domain.Item
	require.NoError(t, json.NewDecoder(r1.Body).Decode(&item))
	r1.Body.Close()
	assert.Equal(t, 60, item.Quantity)

	// 3. Dispatch 25 units
	pgPost(t, srv, "/dispatch", domain.StockRequest{ItemID: "BRG-PG-100", Quantity: 25}).Body.Close()

	// 4. Verify remaining stock via DB
	var qty int
	require.NoError(t, pgDB.QueryRow("SELECT quantity FROM items WHERE item_id = $1", "BRG-PG-100").Scan(&qty))
	assert.Equal(t, 35, qty)

	// 5. Receive 10 more
	pgPost(t, srv, "/receive", domain.Item{ID: "BRG-PG-100", Name: "RAM 32GB", Quantity: 10}).Body.Close()

	// 6. Final check: 35 + 10 = 45
	r2 := pgGet(t, srv, "/check-stock?id=BRG-PG-100")
	require.NoError(t, json.NewDecoder(r2.Body).Decode(&item))
	r2.Body.Close()
	assert.Equal(t, 45, item.Quantity)
}
