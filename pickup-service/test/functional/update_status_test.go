package functional_test

import (
	"context"
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

// TestUpdateStatus_PickedUp verifies that transitioning an ASSIGNED pickup to
// PICKED_UP calls PublishPickedUpEvent exactly once and updates the DB status.
func TestUpdateStatus_PickedUp(t *testing.T) {
	truncateAll(t)
	notifStub.Reset()

	svc, tracker := newTestEnv(t)
	ctx := context.Background()

	// Create and assign a pickup via service
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

	_, err = svc.AssignCourier(ctx, pickupID, "courier-001")
	require.NoError(t, err)

	// Expect exactly one tracking event call
	tracker.EXPECT().
		PublishPickedUpEvent(gomock.Any(), pickupID, out.OrderID, gomock.Any()).
		Return(nil).
		Times(1)

	// Transition to PICKED_UP
	_, err = svc.UpdatePickupStatus(ctx, pickupID, domain.StatusPickedUp)
	require.NoError(t, err)

	// Assert DB status is PICKED_UP
	dbRecord := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusPickedUp, dbRecord.Status)
}

// TestUpdateStatus_FailedAttempt verifies that transitioning an ASSIGNED pickup
// to FAILED_ATTEMPT increments attempt_count, updates DB status, and triggers
// a notification with the correct contact details.
func TestUpdateStatus_FailedAttempt(t *testing.T) {
	truncateAll(t)
	notifStub.Reset()

	svc, _ := newTestEnv(t)
	ctx := context.Background()

	payload := validPickupPayload()

	// Create and assign a pickup via service
	out, err := svc.RequestPickup(ctx, service.RequestPickupInput{
		OrderID:        payload["order_id"].(string),
		UserID:         payload["user_id"].(string),
		PickupAddress:  payload["pickup_address"].(string),
		PickupCityCode: payload["pickup_city_code"].(string),
		ContactName:    payload["contact_name"].(string),
		ContactPhone:   payload["contact_phone"].(string),
	})
	require.NoError(t, err)
	pickupID := out.PickupID

	courierID := "courier-001"
	_, err = svc.AssignCourier(ctx, pickupID, courierID)
	require.NoError(t, err)

	// Transition to FAILED_ATTEMPT
	_, err = svc.UpdatePickupStatus(ctx, pickupID, domain.StatusFailedAttempt)
	require.NoError(t, err)

	// Assert DB: attempt_count == 1 and status == FAILED_ATTEMPT
	dbRecord := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusFailedAttempt, dbRecord.Status)
	assert.Equal(t, 1, dbRecord.AttemptCount)

	// Assert notification stub received exactly one call with correct fields
	calls := notifStub.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, payload["contact_name"].(string), calls[0].ContactName)
	assert.Equal(t, payload["contact_phone"].(string), calls[0].ContactPhone)
	assert.Equal(t, courierID, calls[0].CourierID)
}

// TestUpdateStatus_FailedAttemptThenRescheduled verifies that after a
// FAILED_ATTEMPT, the pickup can be rescheduled to SCHEDULED, and that
// attempt_count is preserved.
func TestUpdateStatus_FailedAttemptThenRescheduled(t *testing.T) {
	truncateAll(t)
	notifStub.Reset()

	svc, _ := newTestEnv(t)
	ctx := context.Background()

	// Create and assign a pickup via service
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

	_, err = svc.AssignCourier(ctx, pickupID, "courier-001")
	require.NoError(t, err)

	// Transition to FAILED_ATTEMPT
	_, err = svc.UpdatePickupStatus(ctx, pickupID, domain.StatusFailedAttempt)
	require.NoError(t, err)

	dbAfterFailed := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusFailedAttempt, dbAfterFailed.Status)
	assert.Equal(t, 1, dbAfterFailed.AttemptCount)

	// Reschedule to SCHEDULED
	_, err = svc.UpdatePickupStatus(ctx, pickupID, domain.StatusScheduled)
	require.NoError(t, err)

	// Assert DB: status == SCHEDULED and attempt_count still 1
	dbAfterReschedule := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusScheduled, dbAfterReschedule.Status)
	assert.Equal(t, 1, dbAfterReschedule.AttemptCount)
}

// TestUpdateStatus_InvalidStatus verifies that posting an invalid status value
// returns HTTP 400 and leaves the DB record unchanged.
func TestUpdateStatus_InvalidStatus(t *testing.T) {
	truncateAll(t)

	// Create a SCHEDULED pickup via HTTP
	created := createPickupHTTP(t, validPickupPayload())
	pickupID, ok := created["pickup_id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, pickupID)

	// Record DB state before the invalid request
	dbBefore := getPickupFromDB(t, pickupID)

	// POST with invalid status
	resp := doRequest(t, http.MethodPost, fmt.Sprintf("/pickups/%s/status", pickupID),
		map[string]interface{}{"status": "INVALID"})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Assert DB record is unchanged
	dbAfter := getPickupFromDB(t, pickupID)
	assert.Equal(t, dbBefore.Status, dbAfter.Status)
	assert.Equal(t, dbBefore.AttemptCount, dbAfter.AttemptCount)
}

// TestUpdateStatus_InvalidTransition verifies that attempting an invalid
// state transition (SCHEDULED → PICKED_UP directly) returns HTTP 400 and
// leaves the DB record unchanged.
func TestUpdateStatus_InvalidTransition(t *testing.T) {
	truncateAll(t)

	// Create a SCHEDULED pickup via HTTP (not assigned)
	created := createPickupHTTP(t, validPickupPayload())
	pickupID, ok := created["pickup_id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, pickupID)

	// Record DB state before the invalid transition attempt
	dbBefore := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusScheduled, dbBefore.Status)

	// Attempt SCHEDULED → PICKED_UP directly (invalid transition)
	resp := doRequest(t, http.MethodPost, fmt.Sprintf("/pickups/%s/status", pickupID),
		map[string]interface{}{"status": "PICKED_UP"})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Assert DB record is unchanged
	dbAfter := getPickupFromDB(t, pickupID)
	assert.Equal(t, dbBefore.Status, dbAfter.Status)
}

// TestUpdateStatus_TrackingEvent is a property-based test verifying that for
// any pickup, transitioning to PICKED_UP calls PublishPickedUpEvent exactly
// once with the correct pickup_id and order_id.
//
// Feature: pickup-service-functional-tests, Property 3: For any pickup, transitioning to PICKED_UP calls PublishPickedUpEvent exactly once with the correct pickup_id and order_id.
//
// Validates: Requirements 4.2
func TestUpdateStatus_TrackingEvent(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		truncateAll(t)
		notifStub.Reset()

		svc, tracker := newTestEnv(t)
		ctx := context.Background()

		payload := genPickupRequest(rt)
		courierID := genCourierID(rt)

		// Create pickup via service
		out, err := svc.RequestPickup(ctx, service.RequestPickupInput{
			OrderID:        payload["order_id"].(string),
			UserID:         payload["user_id"].(string),
			PickupAddress:  payload["pickup_address"].(string),
			PickupCityCode: payload["pickup_city_code"].(string),
			ContactName:    payload["contact_name"].(string),
			ContactPhone:   payload["contact_phone"].(string),
		})
		require.NoError(t, err)

		// Assign courier
		_, err = svc.AssignCourier(ctx, out.PickupID, courierID)
		require.NoError(t, err)

		// Expect exactly one tracking event with correct IDs
		tracker.EXPECT().
			PublishPickedUpEvent(gomock.Any(), out.PickupID, out.OrderID, gomock.Any()).
			Return(nil).
			Times(1)

		// Transition to PICKED_UP
		_, err = svc.UpdatePickupStatus(ctx, out.PickupID, domain.StatusPickedUp)
		require.NoError(t, err)

		// Assert DB status is PICKED_UP
		dbRecord := getPickupFromDB(t, out.PickupID)
		assert.Equal(t, domain.StatusPickedUp, dbRecord.Status)
	})
}

// TestUpdateStatus_NotificationCall is a property-based test verifying that for
// any pickup, transitioning to FAILED_ATTEMPT triggers a notification with the
// correct contact details.
//
// Feature: pickup-service-functional-tests, Property 4: For any pickup, transitioning to FAILED_ATTEMPT triggers notification with correct contact details.
//
// Validates: Requirements 4.4
func TestUpdateStatus_NotificationCall(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		truncateAll(t)
		notifStub.Reset()

		svc, _ := newTestEnv(t)
		ctx := context.Background()

		payload := genPickupRequest(rt)
		courierID := genCourierID(rt)

		// Create pickup via service
		out, err := svc.RequestPickup(ctx, service.RequestPickupInput{
			OrderID:        payload["order_id"].(string),
			UserID:         payload["user_id"].(string),
			PickupAddress:  payload["pickup_address"].(string),
			PickupCityCode: payload["pickup_city_code"].(string),
			ContactName:    payload["contact_name"].(string),
			ContactPhone:   payload["contact_phone"].(string),
		})
		require.NoError(t, err)

		// Assign courier
		_, err = svc.AssignCourier(ctx, out.PickupID, courierID)
		require.NoError(t, err)

		// Transition to FAILED_ATTEMPT
		_, err = svc.UpdatePickupStatus(ctx, out.PickupID, domain.StatusFailedAttempt)
		require.NoError(t, err)

		// Assert DB: status FAILED_ATTEMPT and attempt_count == 1
		dbRecord := getPickupFromDB(t, out.PickupID)
		assert.Equal(t, domain.StatusFailedAttempt, dbRecord.Status)
		assert.Equal(t, 1, dbRecord.AttemptCount)

		// Assert notification stub received exactly one call with correct fields
		calls := notifStub.Calls()
		require.Len(t, calls, 1)
		assert.Equal(t, payload["contact_name"].(string), calls[0].ContactName)
		assert.Equal(t, payload["contact_phone"].(string), calls[0].ContactPhone)
		assert.Equal(t, courierID, calls[0].CourierID)
	})
}

// TestUpdateStatus_AttemptCountMonotonic is a property-based test verifying
// that attempt_count is monotonically non-decreasing across any sequence of
// FAILED_ATTEMPT transitions interspersed with SCHEDULED reschedules.
//
// Feature: pickup-service-functional-tests, Property 5: attempt_count is monotonically non-decreasing.
//
// Validates: Requirements 4.7
func TestUpdateStatus_AttemptCountMonotonic(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		truncateAll(t)
		notifStub.Reset()

		svc, _ := newTestEnv(t)
		ctx := context.Background()

		payload := genPickupRequest(rt)
		courierID := genCourierID(rt)

		// Create pickup via service
		out, err := svc.RequestPickup(ctx, service.RequestPickupInput{
			OrderID:        payload["order_id"].(string),
			UserID:         payload["user_id"].(string),
			PickupAddress:  payload["pickup_address"].(string),
			PickupCityCode: payload["pickup_city_code"].(string),
			ContactName:    payload["contact_name"].(string),
			ContactPhone:   payload["contact_phone"].(string),
		})
		require.NoError(t, err)

		// Assign courier
		_, err = svc.AssignCourier(ctx, out.PickupID, courierID)
		require.NoError(t, err)

		// Apply N FAILED_ATTEMPT transitions interspersed with SCHEDULED reschedules
		n := rapid.IntRange(1, 5).Draw(rt, "n")
		prevAttemptCount := 0

		for i := 0; i < n; i++ {
			// Transition to FAILED_ATTEMPT
			_, err = svc.UpdatePickupStatus(ctx, out.PickupID, domain.StatusFailedAttempt)
			require.NoError(t, err)

			dbRecord := getPickupFromDB(t, out.PickupID)
			assert.Equal(t, domain.StatusFailedAttempt, dbRecord.Status)
			// attempt_count must be non-decreasing
			assert.GreaterOrEqual(t, dbRecord.AttemptCount, prevAttemptCount)
			prevAttemptCount = dbRecord.AttemptCount

			// Reschedule (except after the last iteration)
			if i < n-1 {
				_, err = svc.UpdatePickupStatus(ctx, out.PickupID, domain.StatusScheduled)
				require.NoError(t, err)

				// Re-assign courier for next FAILED_ATTEMPT
				_, err = svc.AssignCourier(ctx, out.PickupID, courierID)
				require.NoError(t, err)

				dbRecord = getPickupFromDB(t, out.PickupID)
				assert.Equal(t, domain.StatusAssigned, dbRecord.Status)
				// attempt_count must not decrease after reschedule
				assert.Equal(t, prevAttemptCount, dbRecord.AttemptCount)
			}
		}

		// Final assertion: attempt_count == N
		dbFinal := getPickupFromDB(t, out.PickupID)
		assert.Equal(t, n, dbFinal.AttemptCount)
	})
}


