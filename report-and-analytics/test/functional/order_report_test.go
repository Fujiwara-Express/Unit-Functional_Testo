package functional_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrderReport_Valid(t *testing.T) {
	truncateAll(t, testDB)

	seedOrderRow(t, "o1", "2026-03-10", "HUB_BDG", "DELIVERED", "REG", "MONTHLY", 25000)
	seedOrderRow(t, "o2", "2026-03-11", "HUB_BDG", "DELIVERED", "REG", "MONTHLY", 30000)
	seedOrderRow(t, "o3", "2026-03-12", "HUB_BDG", "FAILED", "REG", "MONTHLY", 0)
	seedOrderRow(t, "o4", "2026-03-13", "HUB_BDG", "RETURNED", "REG", "MONTHLY", 0)

	resp := doRequest(t, "/reports/orders?date_from=2026-03-01&date_to=2026-03-31&hub_id=HUB_BDG")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, "2026-03", body["period"])
	assert.Equal(t, float64(4), body["total_orders"])
	assert.Equal(t, float64(2), body["delivered"])
	assert.Equal(t, float64(1), body["failed"])
	assert.Equal(t, float64(1), body["returned"])
	assert.Equal(t, float64(55000), body["total_revenue"])
	successRate, ok := body["success_rate"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 50.0, successRate, 0.01)
}

func TestGetOrderReport_NoHubFilter(t *testing.T) {
	truncateAll(t, testDB)

	seedOrderRow(t, "o1", "2026-03-10", "HUB_BDG", "DELIVERED", "REG", "MONTHLY", 10000)
	seedOrderRow(t, "o2", "2026-03-10", "HUB_JKT", "DELIVERED", "EXP", "MONTHLY", 20000)

	resp := doRequest(t, "/reports/orders?date_from=2026-03-01&date_to=2026-03-31")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, float64(2), body["total_orders"])
	assert.Equal(t, float64(30000), body["total_revenue"])
}

func TestGetOrderReport_EmptyResult(t *testing.T) {
	truncateAll(t, testDB)

	resp := doRequest(t, "/reports/orders?date_from=2026-03-01&date_to=2026-03-31")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, float64(0), body["total_orders"])
	assert.Equal(t, float64(0), body["total_revenue"])
	assert.Equal(t, float64(0), body["success_rate"])
}

func TestGetOrderReport_DateRangeFiltering(t *testing.T) {
	truncateAll(t, testDB)

	// Inside range
	seedOrderRow(t, "o1", "2026-03-15", "HUB_BDG", "DELIVERED", "REG", "MONTHLY", 10000)
	// Outside range
	seedOrderRow(t, "o2", "2026-04-01", "HUB_BDG", "DELIVERED", "REG", "MONTHLY", 10000)

	resp := doRequest(t, "/reports/orders?date_from=2026-03-01&date_to=2026-03-31")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, float64(1), body["total_orders"])
}

func TestGetOrderReport_Unauthorized(t *testing.T) {
	resp := doRequestNoAuth(t, "/reports/orders?date_from=2026-03-01&date_to=2026-03-31")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestGetOrderReport_MissingDates_ValidationError(t *testing.T) {
	truncateAll(t, testDB)

	resp := doRequest(t, "/reports/orders")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, "VALIDATION_ERROR", body["code"])
}
