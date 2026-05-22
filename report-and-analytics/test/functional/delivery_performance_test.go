package functional_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDeliveryPerformanceReport_Valid(t *testing.T) {
	truncateAll(t, testDB)

	seedDeliveryRow(t, "d1", "CR123", "WEEKLY", "DELIVERED", 3.0)
	seedDeliveryRow(t, "d2", "CR123", "WEEKLY", "DELIVERED", 4.0)
	seedDeliveryRow(t, "d3", "CR123", "WEEKLY", "FAILED", 0.0)
	seedDeliveryRow(t, "d4", "CR123", "WEEKLY", "RETURNED", 0.0)

	resp := doRequest(t, "/reports/delivery-performance?courier_id=CR123&period=WEEKLY")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, "CR123", body["courier_id"])
	assert.Equal(t, "WEEKLY", body["period"])
	assert.Equal(t, float64(4), body["total_jobs"])
	assert.Equal(t, float64(2), body["delivered"])
	assert.Equal(t, float64(1), body["failed"])
	assert.Equal(t, float64(1), body["returned"])
	successRate, ok := body["success_rate"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 50.0, successRate, 0.01)
	avgTime, ok := body["avg_delivery_time_hours"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 1.75, avgTime, 0.01) // avg of 3, 4, 0, 0
}

func TestGetDeliveryPerformanceReport_OnlyThisCourier(t *testing.T) {
	truncateAll(t, testDB)

	seedDeliveryRow(t, "d1", "CR123", "WEEKLY", "DELIVERED", 2.0)
	seedDeliveryRow(t, "d2", "CR999", "WEEKLY", "DELIVERED", 5.0) // different courier

	resp := doRequest(t, "/reports/delivery-performance?courier_id=CR123&period=WEEKLY")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, float64(1), body["total_jobs"])
}

func TestGetDeliveryPerformanceReport_EmptyResult(t *testing.T) {
	truncateAll(t, testDB)

	resp := doRequest(t, "/reports/delivery-performance?courier_id=CR123&period=WEEKLY")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, float64(0), body["total_jobs"])
	assert.Equal(t, float64(0), body["success_rate"])
}

func TestGetDeliveryPerformanceReport_MissingCourierID(t *testing.T) {
	truncateAll(t, testDB)

	resp := doRequest(t, "/reports/delivery-performance?period=WEEKLY")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, "VALIDATION_ERROR", body["code"])
}

func TestGetDeliveryPerformanceReport_InvalidPeriod(t *testing.T) {
	truncateAll(t, testDB)

	resp := doRequest(t, "/reports/delivery-performance?courier_id=CR123&period=YEARLY")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := decodeJSON(t, resp)
	assert.Equal(t, "VALIDATION_ERROR", body["code"])
}

func TestGetDeliveryPerformanceReport_Unauthorized(t *testing.T) {
	resp := doRequestNoAuth(t, "/reports/delivery-performance?courier_id=CR123&period=WEEKLY")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestGetDeliveryPerformanceReport_AllPeriods(t *testing.T) {
	periods := []string{"DAILY", "WEEKLY", "MONTHLY"}
	for _, period := range periods {
		t.Run(period, func(t *testing.T) {
			truncateAll(t, testDB)
			seedDeliveryRow(t, "d1", "CR123", period, "DELIVERED", 2.0)

			resp := doRequest(t, "/reports/delivery-performance?courier_id=CR123&period="+period)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			body := decodeJSON(t, resp)
			assert.Equal(t, float64(1), body["total_jobs"])
		})
	}
}
