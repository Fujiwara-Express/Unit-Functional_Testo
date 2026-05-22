package functional_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetHubPerformanceReport_Valid(t *testing.T) {
	truncateAll(t, testDB)

	seedHubRow(t, "HUB_BDG", "WEEKLY", 8500, 8200, 8000, 6.2, 72.5)

	resp := doRequest(t, "/reports/hub-performance?hub_id=HUB_BDG&period=WEEKLY")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, "HUB_BDG", body["hub_id"])
	assert.Equal(t, "WEEKLY", body["period"])
	assert.Equal(t, float64(8500), body["total_inbound"])
	assert.Equal(t, float64(8200), body["total_outbound"])
	assert.Equal(t, float64(8000), body["total_dispatched"])
	dwellTime, ok := body["avg_dwell_time_hours"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 6.2, dwellTime, 0.01)
	utilPct, ok := body["capacity_utilization_pct"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 72.5, utilPct, 0.01)
}

func TestGetHubPerformanceReport_MultipleRowsAggregated(t *testing.T) {
	truncateAll(t, testDB)

	// Two rows for the same hub/period — should be summed/averaged
	seedHubRow(t, "HUB_BDG", "WEEKLY", 4000, 3800, 3700, 5.0, 60.0)
	seedHubRow(t, "HUB_BDG", "WEEKLY", 4500, 4400, 4300, 7.0, 80.0)

	resp := doRequest(t, "/reports/hub-performance?hub_id=HUB_BDG&period=WEEKLY")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, float64(8500), body["total_inbound"])  // 4000 + 4500
	assert.Equal(t, float64(8200), body["total_outbound"]) // 3800 + 4400
	dwellTime, ok := body["avg_dwell_time_hours"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 6.0, dwellTime, 0.01) // avg(5.0, 7.0)
}

func TestGetHubPerformanceReport_OnlyThisHub(t *testing.T) {
	truncateAll(t, testDB)

	seedHubRow(t, "HUB_BDG", "WEEKLY", 1000, 900, 800, 5.0, 50.0)
	seedHubRow(t, "HUB_JKT", "WEEKLY", 2000, 1800, 1700, 4.0, 60.0) // different hub

	resp := doRequest(t, "/reports/hub-performance?hub_id=HUB_BDG&period=WEEKLY")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, float64(1000), body["total_inbound"])
}

func TestGetHubPerformanceReport_EmptyResult(t *testing.T) {
	truncateAll(t, testDB)

	resp := doRequest(t, "/reports/hub-performance?hub_id=HUB_BDG&period=WEEKLY")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, float64(0), body["total_inbound"])
	assert.Equal(t, float64(0), body["capacity_utilization_pct"])
}

func TestGetHubPerformanceReport_MissingHubID(t *testing.T) {
	truncateAll(t, testDB)

	resp := doRequest(t, "/reports/hub-performance?period=WEEKLY")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, "VALIDATION_ERROR", body["code"])
}

func TestGetHubPerformanceReport_InvalidPeriod(t *testing.T) {
	truncateAll(t, testDB)

	resp := doRequest(t, "/reports/hub-performance?hub_id=HUB_BDG&period=INVALID")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, "VALIDATION_ERROR", body["code"])
}

func TestGetHubPerformanceReport_Unauthorized(t *testing.T) {
	resp := doRequestNoAuth(t, "/reports/hub-performance?hub_id=HUB_BDG&period=WEEKLY")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestGetHubPerformanceReport_AllPeriods(t *testing.T) {
	periods := []string{"DAILY", "WEEKLY", "MONTHLY"}
	for _, period := range periods {
		t.Run(period, func(t *testing.T) {
			truncateAll(t, testDB)
			seedHubRow(t, "HUB_BDG", period, 100, 90, 80, 4.0, 55.0)

			resp := doRequest(t, "/reports/hub-performance?hub_id=HUB_BDG&period="+period)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			body := decodeJSON(t, resp)
			assert.Equal(t, float64(100), body["total_inbound"])
		})
	}
}
