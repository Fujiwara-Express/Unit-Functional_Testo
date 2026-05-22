package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/pickup-service/internal/client"
	"github.com/pickup-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// TestDeliveryClient_GetAvailableCouriers_ValidResponse verifies that the client
// sends a GET request to the correct endpoint with city_code query param and
// correctly deserializes the JSON response into a courier list.
func TestDeliveryClient_GetAvailableCouriers_ValidResponse(t *testing.T) {
	couriers := []client.Courier{
		{CourierID: "courier-001", Name: "Jane Smith", CityCode: "JKT"},
		{CourierID: "courier-002", Name: "John Doe", CityCode: "JKT"},
	}

	var capturedPath string
	var capturedQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.Query().Get("city_code")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(couriers)
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	result, err := c.GetAvailableCouriers(context.Background(), "JKT")

	require.NoError(t, err)
	assert.Equal(t, "/couriers", capturedPath)
	assert.Equal(t, "JKT", capturedQuery)
	require.Len(t, result, 2)
	assert.Equal(t, "courier-001", result[0].CourierID)
	assert.Equal(t, "Jane Smith", result[0].Name)
	assert.Equal(t, "JKT", result[0].CityCode)
}

// TestDeliveryClient_GetAvailableCouriers_ServiceUnavailable verifies that
// an HTTP 503 response maps to domain.ErrServiceUnavailable.
func TestDeliveryClient_GetAvailableCouriers_ServiceUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	_, err := c.GetAvailableCouriers(context.Background(), "JKT")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrServiceUnavailable)
}

// Feature: pickup-service-unit-tests, Property 15: DeliveryClient sends GET with city_code for any city code
func TestDeliveryClient_GetAvailableCouriers_AnyCityCode(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cityCode := rapid.StringMatching(`[A-Z]{2,5}`).Draw(t, "city_code")

		var capturedCityCode string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedCityCode = r.URL.Query().Get("city_code")
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
		}))
		defer srv.Close()

		c := client.New(srv.URL)
		_, err := c.GetAvailableCouriers(context.Background(), cityCode)

		require.NoError(t, err)
		assert.Equal(t, cityCode, capturedCityCode)
	})
}

// Feature: pickup-service-unit-tests, Property 14: Courier response JSON round-trip
func TestCourierResponse_JSON_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original := client.Courier{
			CourierID: rapid.StringMatching(`[a-z0-9-]{4,20}`).Draw(t, "courier_id"),
			Name:      rapid.StringN(2, 50, -1).Draw(t, "name"),
			CityCode:  rapid.StringMatching(`[A-Z]{2,5}`).Draw(t, "city_code"),
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var got client.Courier
		require.NoError(t, json.Unmarshal(data, &got))

		assert.Equal(t, original, got)
	})
}

// TestDeliveryClient_GetAvailableCouriers_UnexpectedStatus verifies that a
// non-200, non-503 response returns a generic error.
func TestDeliveryClient_GetAvailableCouriers_UnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	_, err := c.GetAvailableCouriers(context.Background(), "JKT")

	require.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrServiceUnavailable)
}
