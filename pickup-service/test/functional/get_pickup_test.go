package functional_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestGetPickup_Exists verifies that GET /pickups/{pickup_id} returns HTTP 200
// with all pickup fields matching the DB record.
func TestGetPickup_Exists(t *testing.T) {
	truncateAll(t)

	// Create a pickup via POST /pickups
	payload := validPickupPayload()
	created := createPickupHTTP(t, payload)

	pickupID, ok := created["pickup_id"].(string)
	require.True(t, ok, "pickup_id should be a string in create response")
	require.NotEmpty(t, pickupID)

	// GET /pickups/{pickup_id}
	resp := doRequest(t, http.MethodGet, "/pickups/"+pickupID, nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	// Read the DB record for comparison
	dbRecord := getPickupFromDB(t, pickupID)

	// Assert all fields match the DB record
	assert.Equal(t, dbRecord.PickupID, body["pickup_id"])
	assert.Equal(t, dbRecord.OrderID, body["order_id"])
	assert.Equal(t, dbRecord.UserID, body["user_id"])
	assert.Equal(t, string(dbRecord.Status), body["status"])
	assert.Equal(t, dbRecord.PickupAddress, body["pickup_address"])
	assert.Equal(t, dbRecord.PickupCityCode, body["pickup_city_code"])
	assert.Equal(t, dbRecord.ContactName, body["contact_name"])
	assert.Equal(t, dbRecord.ContactPhone, body["contact_phone"])
	assert.Equal(t, float64(dbRecord.AttemptCount), body["attempt_count"])
}

// TestGetPickup_NotFound verifies that GET /pickups/{pickup_id} returns HTTP 404
// when the pickup does not exist.
func TestGetPickup_NotFound(t *testing.T) {
	truncateAll(t)

	resp := doRequest(t, http.MethodGet, "/pickups/nonexistent-id", nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestGetPickup_RoundTrip is a property-based test verifying that for any pickup
// created via POST /pickups, a subsequent GET /pickups/{pickup_id} returns HTTP 200
// with response fields matching the DB record.
//
// Feature: pickup-service-functional-tests, Property 6: For any pickup created via POST /pickups,
// a subsequent GET /pickups/{pickup_id} returns HTTP 200 with response fields matching the DB record.
//
// Validates: Requirements 5.1, 5.3
func TestGetPickup_RoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		truncateAll(t)

		// Generate and create a pickup
		payload := genPickupRequest(rt)
		created := createPickupHTTP(t, payload)

		pickupID, ok := created["pickup_id"].(string)
		require.True(t, ok, "pickup_id should be a string in create response")
		require.NotEmpty(t, pickupID)

		// GET /pickups/{pickup_id}
		resp := doRequest(t, http.MethodGet, fmt.Sprintf("/pickups/%s", pickupID), nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

		// Read the DB record for comparison
		dbRecord := getPickupFromDB(t, pickupID)

		// Assert all response fields match the DB record
		assert.Equal(t, dbRecord.PickupID, body["pickup_id"])
		assert.Equal(t, dbRecord.OrderID, body["order_id"])
		assert.Equal(t, dbRecord.UserID, body["user_id"])
		assert.Equal(t, string(dbRecord.Status), body["status"])
		assert.Equal(t, dbRecord.PickupAddress, body["pickup_address"])
		assert.Equal(t, dbRecord.PickupCityCode, body["pickup_city_code"])
		assert.Equal(t, dbRecord.ContactName, body["contact_name"])
		assert.Equal(t, dbRecord.ContactPhone, body["contact_phone"])
		assert.Equal(t, float64(dbRecord.AttemptCount), body["attempt_count"])
	})
}
