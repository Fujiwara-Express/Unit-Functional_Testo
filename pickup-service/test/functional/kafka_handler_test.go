package functional_test

import (
	"encoding/json"
	"testing"

	"github.com/pickup-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestHandleOrderCreatedEvent_Valid verifies that calling HandleOrderCreatedEvent
// with a valid JSON payload inserts a record with status SCHEDULED and all
// event fields correctly stored.
func TestHandleOrderCreatedEvent_Valid(t *testing.T) {
	truncateAll(t)

	payload := map[string]interface{}{
		"order_id":         "order-abc",
		"user_id":          "user-xyz",
		"pickup_address":   "123 Main St",
		"pickup_city_code": "JKT",
		"contact_name":     "John Doe",
		"contact_phone":    "+62812345678",
	}
	b, err := json.Marshal(payload)
	require.NoError(t, err)

	err = kafkaHandler.HandleOrderCreatedEvent(b)
	require.NoError(t, err)

	// Verify exactly one record was inserted
	count := countPickups(t)
	assert.Equal(t, 1, count)

	// Find the inserted record by querying all pickups and checking fields
	rows, err := testDB.Query(
		`SELECT pickup_id, order_id, user_id, status, pickup_address, pickup_city_code, contact_name, contact_phone
		 FROM pickups LIMIT 1`)
	require.NoError(t, err)
	defer rows.Close()

	require.True(t, rows.Next(), "expected at least one row")
	var pickupID, orderID, userID, status, pickupAddress, pickupCityCode, contactName, contactPhone string
	require.NoError(t, rows.Scan(&pickupID, &orderID, &userID, &status, &pickupAddress, &pickupCityCode, &contactName, &contactPhone))

	assert.NotEmpty(t, pickupID)
	assert.Equal(t, "order-abc", orderID)
	assert.Equal(t, "user-xyz", userID)
	assert.Equal(t, string(domain.StatusScheduled), status)
	assert.Equal(t, "123 Main St", pickupAddress)
	assert.Equal(t, "JKT", pickupCityCode)
	assert.Equal(t, "John Doe", contactName)
	assert.Equal(t, "+62812345678", contactPhone)
}

// TestHandleOrderCreatedEvent_MalformedJSON verifies that calling
// HandleOrderCreatedEvent with malformed JSON returns an error and inserts
// no record into the DB.
func TestHandleOrderCreatedEvent_MalformedJSON(t *testing.T) {
	truncateAll(t)

	err := kafkaHandler.HandleOrderCreatedEvent([]byte("not-json"))
	assert.Error(t, err)
	assert.Equal(t, 0, countPickups(t))
}

// TestHandleOrderCreatedEvent_MissingFields is a table-driven test verifying
// that calling HandleOrderCreatedEvent with a payload missing any required
// field returns an error and inserts no record into the DB.
func TestHandleOrderCreatedEvent_MissingFields(t *testing.T) {
	basePayload := map[string]interface{}{
		"order_id":         "order-abc",
		"user_id":          "user-xyz",
		"pickup_address":   "123 Main St",
		"pickup_city_code": "JKT",
		"contact_name":     "John Doe",
		"contact_phone":    "+62812345678",
	}

	cases := []struct {
		name         string
		omittedField string
	}{
		{name: "missing order_id", omittedField: "order_id"},
		{name: "missing user_id", omittedField: "user_id"},
		{name: "missing pickup_address", omittedField: "pickup_address"},
		{name: "missing pickup_city_code", omittedField: "pickup_city_code"},
		{name: "missing contact_name", omittedField: "contact_name"},
		{name: "missing contact_phone", omittedField: "contact_phone"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			truncateAll(t)

			// Build payload without the omitted field
			payload := make(map[string]interface{})
			for k, v := range basePayload {
				if k != tc.omittedField {
					payload[k] = v
				}
			}

			b, err := json.Marshal(payload)
			require.NoError(t, err)

			err = kafkaHandler.HandleOrderCreatedEvent(b)
			assert.Error(t, err, "expected error when %s is missing", tc.omittedField)
			assert.Equal(t, 0, countPickups(t), "expected no record inserted when %s is missing", tc.omittedField)
		})
	}
}

// TestHandleOrderCreatedEvent_RoundTrip is a property-based test verifying that
// for any valid OrderCreatedEvent payload, HandleOrderCreatedEvent inserts a
// record with status SCHEDULED and all event fields correctly stored.
//
// Feature: pickup-service-functional-tests, Property 9: For any valid OrderCreatedEvent payload, HandleOrderCreatedEvent inserts a record with status SCHEDULED and all event fields correctly stored.
//
// Validates: Requirements 8.1
func TestHandleOrderCreatedEvent_RoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		truncateAll(t)

		// Generate a random valid event payload using the existing generator
		fields := genPickupRequest(rt)

		payload := map[string]interface{}{
			"order_id":         fields["order_id"],
			"user_id":          fields["user_id"],
			"pickup_address":   fields["pickup_address"],
			"pickup_city_code": fields["pickup_city_code"],
			"contact_name":     fields["contact_name"],
			"contact_phone":    fields["contact_phone"],
		}

		b, err := json.Marshal(payload)
		require.NoError(t, err)

		err = kafkaHandler.HandleOrderCreatedEvent(b)
		require.NoError(t, err)

		// Verify exactly one record was inserted
		require.Equal(t, 1, countPickups(t))

		// Read the inserted record directly from DB
		row := testDB.QueryRow(
			`SELECT pickup_id, order_id, user_id, status, pickup_address, pickup_city_code, contact_name, contact_phone
			 FROM pickups LIMIT 1`)
		var pickupID, orderID, userID, status, pickupAddress, pickupCityCode, contactName, contactPhone string
		require.NoError(t, row.Scan(&pickupID, &orderID, &userID, &status, &pickupAddress, &pickupCityCode, &contactName, &contactPhone))

		assert.NotEmpty(t, pickupID)
		assert.Equal(t, fields["order_id"], orderID)
		assert.Equal(t, fields["user_id"], userID)
		assert.Equal(t, string(domain.StatusScheduled), status)
		assert.Equal(t, fields["pickup_address"], pickupAddress)
		assert.Equal(t, fields["pickup_city_code"], pickupCityCode)
		assert.Equal(t, fields["contact_name"], contactName)
		assert.Equal(t, fields["contact_phone"], contactPhone)
	})
}

// TestHandleOrderCreatedEvent_RejectsMalformed is a property-based test verifying
// that for any non-JSON byte slice, HandleOrderCreatedEvent returns an error and
// inserts no record into the DB.
//
// Feature: pickup-service-functional-tests, Property 10: For any non-JSON byte slice, HandleOrderCreatedEvent returns an error and inserts no record.
//
// Validates: Requirements 8.2
func TestHandleOrderCreatedEvent_RejectsMalformed(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		truncateAll(t)

		// Generate a random byte slice that is NOT valid JSON by prepending 0x01
		raw := rapid.SliceOf(rapid.Byte()).Draw(rt, "raw")
		invalid := append([]byte{0x01}, raw...)

		err := kafkaHandler.HandleOrderCreatedEvent(invalid)
		require.Error(t, err)
		assert.Equal(t, 0, countPickups(t))
	})
}
