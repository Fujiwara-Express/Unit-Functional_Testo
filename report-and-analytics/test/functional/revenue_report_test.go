package functional_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRevenueReport_Valid(t *testing.T) {
	truncateAll(t, testDB)

	seedOrderRow(t, "o1", "2026-03-01", "", "DELIVERED", "REG", "MONTHLY", 25000)
	seedOrderRow(t, "o2", "2026-03-02", "", "DELIVERED", "REG", "MONTHLY", 35000)
	seedOrderRow(t, "o3", "2026-03-03", "", "FAILED", "REG", "MONTHLY", 0)

	resp := doRequest(t, "/reports/revenue?period=MONTHLY&service_type=REG")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, "MONTHLY", body["period"])
	assert.Equal(t, "REG", body["service_type"])
	assert.Equal(t, float64(60000), body["total_revenue"])
	assert.Equal(t, float64(3), body["total_orders"])
	avgVal, ok := body["avg_order_value"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 20000.0, avgVal, 1.0) // 60000 / 3
}

func TestGetRevenueReport_NoServiceTypeFilter(t *testing.T) {
	truncateAll(t, testDB)

	seedOrderRow(t, "o1", "2026-03-01", "", "DELIVERED", "REG", "MONTHLY", 25000)
	seedOrderRow(t, "o2", "2026-03-02", "", "DELIVERED", "EXP", "MONTHLY", 50000)

	resp := doRequest(t, "/reports/revenue?period=MONTHLY")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, float64(75000), body["total_revenue"])
	assert.Equal(t, float64(2), body["total_orders"])
}

func TestGetRevenueReport_ServiceTypeFiltering(t *testing.T) {
	truncateAll(t, testDB)

	seedOrderRow(t, "o1", "2026-03-01", "", "DELIVERED", "REG", "MONTHLY", 25000)
	seedOrderRow(t, "o2", "2026-03-02", "", "DELIVERED", "EXP", "MONTHLY", 50000)

	resp := doRequest(t, "/reports/revenue?period=MONTHLY&service_type=EXP")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, float64(50000), body["total_revenue"])
	assert.Equal(t, float64(1), body["total_orders"])
}

func TestGetRevenueReport_EmptyResult(t *testing.T) {
	truncateAll(t, testDB)

	resp := doRequest(t, "/reports/revenue?period=MONTHLY")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, float64(0), body["total_revenue"])
	assert.Equal(t, float64(0), body["total_orders"])
	assert.Equal(t, float64(0), body["avg_order_value"])
}

func TestGetRevenueReport_InvalidPeriod(t *testing.T) {
	truncateAll(t, testDB)

	resp := doRequest(t, "/reports/revenue?period=QUARTERLY")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, "VALIDATION_ERROR", body["code"])
}

func TestGetRevenueReport_MissingPeriod(t *testing.T) {
	truncateAll(t, testDB)

	resp := doRequest(t, "/reports/revenue")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, "VALIDATION_ERROR", body["code"])
}

func TestGetRevenueReport_Unauthorized(t *testing.T) {
	resp := doRequestNoAuth(t, "/reports/revenue?period=MONTHLY")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}
