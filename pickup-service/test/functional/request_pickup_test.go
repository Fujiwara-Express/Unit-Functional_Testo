package functional_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/pickup-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestRequestPickup_ValidInput verifies that a valid POST /pickups request
// returns HTTP 201 with all expected response fields.
func TestRequestPickup_ValidInput(t *testing.T) {
	truncateAll(t)

	payload := validPickupPayload()
	resp := createPickupHTTP(t, payload)

	// Assert required response fields are present
	pickupID, ok := resp["pickup_id"].(string)
	require.True(t, ok, "pickup_id should be a string")
	assert.NotEmpty(t, pickupID)

	orderID, ok := resp["order_id"].(string)
	require.True(t, ok, "order_id should be a string")
	assert.Equal(t, payload["order_id"], orderID)

	status, ok := resp["status"].(string)
	require.True(t, ok, "status should be a string")
	assert.Equal(t, "SCHEDULED", status)

	assert.NotEmpty(t, resp["estimated_pickup_time"], "estimated_pickup_time should be present")
	assert.NotEmpty(t, resp["created_at"], "created_at should be present")
}

// TestRequestPickup_MissingFields verifies that omitting any required field
// returns HTTP 400 and no pickup is created in the DB.
func TestRequestPickup_MissingFields(t *testing.T) {
	cases := []struct {
		name        string
		omitField   string
	}{
		{"missing order_id", "order_id"},
		{"missing user_id", "user_id"},
		{"missing pickup_address", "pickup_address"},
		{"missing pickup_city_code", "pickup_city_code"},
		{"missing contact_name", "contact_name"},
		{"missing contact_phone", "contact_phone"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			truncateAll(t)

			payload := validPickupPayload()
			delete(payload, tc.omitField)

			b, _ := json.Marshal(payload)
			req, _ := http.NewRequest(http.MethodPost, testServer.URL+"/pickups", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-token")
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assert.Equal(t, 0, countPickups(t), "no pickup should be created when %s is missing", tc.omitField)
		})
	}
}

// TestRequestPickup_Unauthorized verifies that a POST /pickups request without
// a Bearer token returns HTTP 401.
func TestRequestPickup_Unauthorized(t *testing.T) {
	truncateAll(t)

	payload := validPickupPayload()
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, testServer.URL+"/pickups", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	// Intentionally omit Authorization header

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestRequestPickup_RoundTrip is a property-based test verifying that for any
// valid RequestPickup payload, the pickup_id in the response matches the DB
// record and all input fields are stored correctly with status SCHEDULED.
//
// Feature: pickup-service-functional-tests, Property 1: For any valid RequestPickup payload,
// pickup_id in response matches DB and all input fields are stored correctly with status SCHEDULED.
//
// Validates: Requirements 2.2, 2.3, 2.6
func TestRequestPickup_RoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		truncateAll(t)
		notifStub.Reset()

		payload := genPickupRequest(rt)
		respBody := createPickupHTTP(t, payload)

		// Assert pickup_id is present in response
		pickupID, ok := respBody["pickup_id"].(string)
		require.True(t, ok, "pickup_id should be a string in response")
		require.NotEmpty(t, pickupID)

		// Assert DB record matches input
		dbRecord := getPickupFromDB(t, pickupID)
		assert.Equal(t, pickupID, dbRecord.PickupID)
		assert.Equal(t, payload["order_id"], dbRecord.OrderID)
		assert.Equal(t, payload["user_id"], dbRecord.UserID)
		assert.Equal(t, domain.StatusScheduled, dbRecord.Status)
		assert.Equal(t, payload["pickup_address"], dbRecord.PickupAddress)
		assert.Equal(t, payload["pickup_city_code"], dbRecord.PickupCityCode)
		assert.Equal(t, payload["contact_name"], dbRecord.ContactName)
		assert.Equal(t, payload["contact_phone"], dbRecord.ContactPhone)

		// Assert response body fields
		assert.Equal(t, payload["order_id"], respBody["order_id"])
		assert.Equal(t, "SCHEDULED", respBody["status"])
		assert.NotEmpty(t, respBody["estimated_pickup_time"])
		assert.NotEmpty(t, respBody["created_at"])
	})
}
