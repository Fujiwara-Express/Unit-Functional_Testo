package functional_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// truncateAll clears all data warehouse tables before each test.
func truncateAll(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		"TRUNCATE TABLE fact_orders, fact_deliveries, fact_hub_performance")
	require.NoError(t, err)
}

// doRequest sends an authenticated GET request to the testServer.
func doRequest(t *testing.T, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, testServer.URL+path, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// doRequestNoAuth sends a GET request without an Authorization header.
func doRequestNoAuth(t *testing.T, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, testServer.URL+path, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// decodeJSON decodes the response body into a map.
func decodeJSON(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	defer resp.Body.Close()
	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result
}

// seedOrderRow inserts a single row into fact_orders.
func seedOrderRow(t *testing.T, orderID, date, hubID, status, serviceType, periodType string, revenue int64) {
	t.Helper()
	_, err := testDB.ExecContext(context.Background(),
		`INSERT INTO fact_orders (order_id, order_date, hub_id, status, service_type, period_type, revenue)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		orderID, date, hubID, status, serviceType, periodType, revenue,
	)
	require.NoError(t, err)
}

// seedDeliveryRow inserts a single row into fact_deliveries.
func seedDeliveryRow(t *testing.T, deliveryID, courierID, periodType, status string, deliveryTimeHours float64) {
	t.Helper()
	_, err := testDB.ExecContext(context.Background(),
		`INSERT INTO fact_deliveries (delivery_id, courier_id, period_type, status, delivery_time_hours)
		 VALUES ($1, $2, $3, $4, $5)`,
		deliveryID, courierID, periodType, status, deliveryTimeHours,
	)
	require.NoError(t, err)
}

// seedHubRow inserts a single row into fact_hub_performance.
func seedHubRow(t *testing.T, hubID, periodType string, inbound, outbound, dispatched int64, dwellHours, utilPct float64) {
	t.Helper()
	_, err := testDB.ExecContext(context.Background(),
		`INSERT INTO fact_hub_performance (hub_id, period_type, inbound_count, outbound_count, dispatched_count, avg_dwell_time_hours, capacity_utilization_pct)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		hubID, periodType, inbound, outbound, dispatched, dwellHours, utilPct,
	)
	require.NoError(t, err)
}
