package functional_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/pickup-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"
)

// TestAssignCourier_ValidInput verifies that assigning a courier to a SCHEDULED
// pickup returns HTTP 200 with status ASSIGNED and the correct courier_id,
// and that the DB record is updated accordingly.
func TestAssignCourier_ValidInput(t *testing.T) {
	truncateAll(t)

	// Create a SCHEDULED pickup
	created := createPickupHTTP(t, validPickupPayload())
	pickupID, ok := created["pickup_id"].(string)
	require.True(t, ok, "pickup_id should be a string")
	require.NotEmpty(t, pickupID)

	// POST assign
	resp := doRequest(t, http.MethodPost, fmt.Sprintf("/pickups/%s/assign", pickupID),
		map[string]interface{}{"courier_id": "courier-001"})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Decode response body
	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	assert.Equal(t, "ASSIGNED", body["status"])
	assert.Equal(t, "courier-001", body["courier_id"])
	assert.Equal(t, pickupID, body["pickup_id"])

	// Assert DB record updated
	dbRecord := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusAssigned, dbRecord.Status)
	assert.Equal(t, "courier-001", dbRecord.CourierID)
}

// TestAssignCourier_NotFound verifies that assigning a courier to a non-existent
// pickup returns HTTP 404.
func TestAssignCourier_NotFound(t *testing.T) {
	truncateAll(t)

	resp := doRequest(t, http.MethodPost, "/pickups/nonexistent-id/assign",
		map[string]interface{}{"courier_id": "courier-001"})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestAssignCourier_InvalidStatus verifies that attempting to assign a courier
// to a pickup that is not in SCHEDULED status returns HTTP 400 and leaves the
// DB record unchanged.
func TestAssignCourier_InvalidStatus(t *testing.T) {
	cases := []struct {
		name string
	}{
		{name: "ASSIGNED"},
		{name: "CANCELLED"},
		{name: "PICKED_UP"},
		{name: "FAILED_ATTEMPT"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			truncateAll(t)

			// Create a SCHEDULED pickup
			created := createPickupHTTP(t, validPickupPayload())
			pickupID, ok := created["pickup_id"].(string)
			require.True(t, ok)
			require.NotEmpty(t, pickupID)

			// Use newTestEnv to transition the pickup to the desired status
			svc, tracker := newTestEnv(t)
			ctx := context.Background()

			switch tc.name {
			case "ASSIGNED":
				_, err := svc.AssignCourier(ctx, pickupID, "courier-setup")
				require.NoError(t, err)

			case "CANCELLED":
				_, err := svc.CancelPickup(ctx, pickupID)
				require.NoError(t, err)

			case "PICKED_UP":
				_, err := svc.AssignCourier(ctx, pickupID, "courier-setup")
				require.NoError(t, err)
				tracker.EXPECT().
					PublishPickedUpEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				_, err = svc.UpdatePickupStatus(ctx, pickupID, domain.StatusPickedUp)
				require.NoError(t, err)

			case "FAILED_ATTEMPT":
				_, err := svc.AssignCourier(ctx, pickupID, "courier-setup")
				require.NoError(t, err)
				_, err = svc.UpdatePickupStatus(ctx, pickupID, domain.StatusFailedAttempt)
				require.NoError(t, err)
			}

			// Record DB state before the invalid assign attempt
			dbBefore := getPickupFromDB(t, pickupID)

			// Attempt to assign courier — should fail with HTTP 400
			resp := doRequest(t, http.MethodPost, fmt.Sprintf("/pickups/%s/assign", pickupID),
				map[string]interface{}{"courier_id": "courier-new"})
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// Assert DB record is unchanged
			dbAfter := getPickupFromDB(t, pickupID)
			assert.Equal(t, dbBefore.Status, dbAfter.Status)
			assert.Equal(t, dbBefore.CourierID, dbAfter.CourierID)
		})
	}
}

// TestAssignCourier_PersistsCourierID is a property-based test verifying that
// for any SCHEDULED pickup and any courier ID string, AssignCourier results in
// the DB record having status ASSIGNED and courier_id equal to the value sent.
//
// Feature: pickup-service-functional-tests, Property 2: For any SCHEDULED pickup and any courier ID string,
// AssignCourier results in DB record having status ASSIGNED and courier_id equal to the value sent.
//
// Validates: Requirements 3.2
func TestAssignCourier_PersistsCourierID(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		truncateAll(t)
		notifStub.Reset()

		// Generate a random valid pickup payload and create it
		payload := genPickupRequest(rt)
		created := createPickupHTTP(t, payload)
		pickupID, ok := created["pickup_id"].(string)
		require.True(t, ok, "pickup_id should be a string")
		require.NotEmpty(t, pickupID)

		// Generate a random courier ID
		courierID := genCourierID(rt)

		// POST assign
		resp := doRequest(t, http.MethodPost, fmt.Sprintf("/pickups/%s/assign", pickupID),
			map[string]interface{}{"courier_id": courierID})
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Assert DB record has status ASSIGNED and correct courier_id
		dbRecord := getPickupFromDB(t, pickupID)
		assert.Equal(t, domain.StatusAssigned, dbRecord.Status)
		assert.Equal(t, courierID, dbRecord.CourierID)
	})
}
