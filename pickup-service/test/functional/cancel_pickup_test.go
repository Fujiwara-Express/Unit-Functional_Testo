package functional_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/pickup-service/internal/domain"
	"github.com/pickup-service/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"
)

// TestCancelPickup_Scheduled verifies that cancelling a SCHEDULED pickup
// returns HTTP 200 with status CANCELLED and updates the DB record accordingly.
func TestCancelPickup_Scheduled(t *testing.T) {
	truncateAll(t)

	// Create a SCHEDULED pickup
	created := createPickupHTTP(t, validPickupPayload())
	pickupID, ok := created["pickup_id"].(string)
	require.True(t, ok, "pickup_id should be a string")
	require.NotEmpty(t, pickupID)

	// POST cancel
	resp := doRequest(t, http.MethodPost, fmt.Sprintf("/pickups/%s/cancel", pickupID), nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Decode response body
	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "CANCELLED", body["status"])

	// Assert DB record has status CANCELLED
	dbRecord := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusCancelled, dbRecord.Status)
}

// TestCancelPickup_Assigned verifies that attempting to cancel an ASSIGNED
// pickup returns HTTP 409 and leaves the DB record unchanged.
func TestCancelPickup_Assigned(t *testing.T) {
	truncateAll(t)

	// Create a SCHEDULED pickup
	created := createPickupHTTP(t, validPickupPayload())
	pickupID, ok := created["pickup_id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, pickupID)

	// Transition to ASSIGNED via service
	svc, _ := newTestEnv(t)
	ctx := context.Background()
	_, err := svc.AssignCourier(ctx, pickupID, "courier-001")
	require.NoError(t, err)

	// Record DB state before cancel attempt
	dbBefore := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusAssigned, dbBefore.Status)

	// POST cancel — should fail with HTTP 409
	resp := doRequest(t, http.MethodPost, fmt.Sprintf("/pickups/%s/cancel", pickupID), nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	// Assert DB record is unchanged (still ASSIGNED)
	dbAfter := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusAssigned, dbAfter.Status)
}

// TestCancelPickup_PickedUp verifies that attempting to cancel a PICKED_UP
// pickup returns HTTP 409 and leaves the DB record unchanged.
func TestCancelPickup_PickedUp(t *testing.T) {
	truncateAll(t)

	svc, tracker := newTestEnv(t)
	ctx := context.Background()

	// Create pickup via service
	out, err := svc.RequestPickup(ctx, service.RequestPickupInput{
		OrderID:        "order-001",
		UserID:         "user-001",
		PickupAddress:  "123 Main St",
		PickupCityCode: "JKT",
		ContactName:    "John Doe",
		ContactPhone:   "+62812345678",
	})
	require.NoError(t, err)
	pickupID := out.PickupID

	// Assign courier
	_, err = svc.AssignCourier(ctx, pickupID, "courier-001")
	require.NoError(t, err)

	// Transition to PICKED_UP
	tracker.EXPECT().
		PublishPickedUpEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)
	_, err = svc.UpdatePickupStatus(ctx, pickupID, domain.StatusPickedUp)
	require.NoError(t, err)

	// Record DB state before cancel attempt
	dbBefore := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusPickedUp, dbBefore.Status)

	// POST cancel — should fail with HTTP 409
	resp := doRequest(t, http.MethodPost, fmt.Sprintf("/pickups/%s/cancel", pickupID), nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	// Assert DB record is unchanged (still PICKED_UP)
	dbAfter := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusPickedUp, dbAfter.Status)
}

// TestCancelPickup_NotFound verifies that attempting to cancel a non-existent
// pickup returns HTTP 404.
func TestCancelPickup_NotFound(t *testing.T) {
	truncateAll(t)

	resp := doRequest(t, http.MethodPost, "/pickups/nonexistent-id/cancel", nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestCancelPickup_PersistsCancelled is a property-based test verifying that
// for any SCHEDULED pickup, CancelPickup results in the DB record having
// status CANCELLED.
//
// Feature: pickup-service-functional-tests, Property 8: For any SCHEDULED pickup, CancelPickup results in the DB record having status CANCELLED.
//
// Validates: Requirements 7.2
func TestCancelPickup_PersistsCancelled(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		truncateAll(t)

		// Generate a random valid pickup payload and create it
		payload := genPickupRequest(rt)
		created := createPickupHTTP(t, payload)
		pickupID, ok := created["pickup_id"].(string)
		require.True(t, ok, "pickup_id should be a string")
		require.NotEmpty(t, pickupID)

		// POST cancel
		resp := doRequest(t, http.MethodPost, fmt.Sprintf("/pickups/%s/cancel", pickupID), nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Assert DB record has status CANCELLED
		dbRecord := getPickupFromDB(t, pickupID)
		assert.Equal(t, domain.StatusCancelled, dbRecord.Status)
	})
}
