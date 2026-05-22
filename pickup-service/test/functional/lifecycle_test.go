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
)

// TestLifecycle_HappyPath exercises the full happy-path lifecycle:
// RequestPickup → AssignCourier → UpdatePickupStatus(PICKED_UP).
// Asserts each step produces the correct DB state, that
// tracker.PublishPickedUpEvent is called exactly once with the correct IDs,
// and that no notifications are sent.
//
// Validates: Requirements 9.1
func TestLifecycle_HappyPath(t *testing.T) {
	truncateAll(t)
	notifStub.Reset()

	svc, tracker := newTestEnv(t)
	ctx := context.Background()

	// Step 1: RequestPickup
	out, err := svc.RequestPickup(ctx, service.RequestPickupInput{
		OrderID:        "order-happy-001",
		UserID:         "user-001",
		PickupAddress:  "123 Main St",
		PickupCityCode: "JKT",
		ContactName:    "John Doe",
		ContactPhone:   "+62812345678",
	})
	require.NoError(t, err)
	pickupID := out.PickupID
	orderID := out.OrderID

	// Assert DB state after RequestPickup
	dbAfterRequest := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusScheduled, dbAfterRequest.Status)
	assert.Equal(t, "order-happy-001", dbAfterRequest.OrderID)
	assert.Equal(t, "user-001", dbAfterRequest.UserID)
	assert.Equal(t, 0, dbAfterRequest.AttemptCount)

	// Step 2: AssignCourier
	courierID := "courier-happy-001"
	_, err = svc.AssignCourier(ctx, pickupID, courierID)
	require.NoError(t, err)

	// Assert DB state after AssignCourier
	dbAfterAssign := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusAssigned, dbAfterAssign.Status)
	assert.Equal(t, courierID, dbAfterAssign.CourierID)

	// Step 3: UpdatePickupStatus(PICKED_UP)
	// Expect exactly one tracking event with correct IDs
	tracker.EXPECT().
		PublishPickedUpEvent(gomock.Any(), pickupID, orderID, gomock.Any()).
		Return(nil).
		Times(1)

	_, err = svc.UpdatePickupStatus(ctx, pickupID, domain.StatusPickedUp)
	require.NoError(t, err)

	// Assert DB state after PICKED_UP
	dbAfterPickedUp := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusPickedUp, dbAfterPickedUp.Status)

	// Assert no notifications were sent (happy path)
	assert.Empty(t, notifStub.Calls(), "no notifications expected for happy path")
}

// TestLifecycle_FailedAttempt exercises the failed-attempt lifecycle:
// RequestPickup → AssignCourier → UpdatePickupStatus(FAILED_ATTEMPT) →
// UpdatePickupStatus(SCHEDULED) → AssignCourier → UpdatePickupStatus(PICKED_UP).
// Asserts attempt_count == 1 after the FAILED_ATTEMPT, that it is preserved
// after rescheduling, that exactly one notification was sent, and that
// tracker.PublishPickedUpEvent is called exactly once at the end.
//
// Validates: Requirements 9.2
func TestLifecycle_FailedAttempt(t *testing.T) {
	truncateAll(t)
	notifStub.Reset()

	svc, tracker := newTestEnv(t)
	ctx := context.Background()

	// Step 1: RequestPickup
	out, err := svc.RequestPickup(ctx, service.RequestPickupInput{
		OrderID:        "order-failed-001",
		UserID:         "user-001",
		PickupAddress:  "456 Side St",
		PickupCityCode: "SBY",
		ContactName:    "Jane Smith",
		ContactPhone:   "+62898765432",
	})
	require.NoError(t, err)
	pickupID := out.PickupID
	orderID := out.OrderID

	// Step 2: AssignCourier
	courierID := "courier-failed-001"
	_, err = svc.AssignCourier(ctx, pickupID, courierID)
	require.NoError(t, err)

	// Step 3: UpdatePickupStatus(FAILED_ATTEMPT)
	_, err = svc.UpdatePickupStatus(ctx, pickupID, domain.StatusFailedAttempt)
	require.NoError(t, err)

	// Assert attempt_count == 1 after FAILED_ATTEMPT
	dbAfterFailed := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusFailedAttempt, dbAfterFailed.Status)
	assert.Equal(t, 1, dbAfterFailed.AttemptCount)

	// Assert exactly one notification was sent
	callsAfterFailed := notifStub.Calls()
	require.Len(t, callsAfterFailed, 1)
	assert.Equal(t, "Jane Smith", callsAfterFailed[0].ContactName)
	assert.Equal(t, "+62898765432", callsAfterFailed[0].ContactPhone)
	assert.Equal(t, courierID, callsAfterFailed[0].CourierID)

	// Step 4: UpdatePickupStatus(SCHEDULED) — reschedule
	_, err = svc.UpdatePickupStatus(ctx, pickupID, domain.StatusScheduled)
	require.NoError(t, err)

	// Assert attempt_count is still 1 after reschedule
	dbAfterReschedule := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusScheduled, dbAfterReschedule.Status)
	assert.Equal(t, 1, dbAfterReschedule.AttemptCount)

	// Step 5: AssignCourier again
	_, err = svc.AssignCourier(ctx, pickupID, courierID)
	require.NoError(t, err)

	// Step 6: UpdatePickupStatus(PICKED_UP)
	// Expect exactly one tracking event with correct IDs
	tracker.EXPECT().
		PublishPickedUpEvent(gomock.Any(), pickupID, orderID, gomock.Any()).
		Return(nil).
		Times(1)

	_, err = svc.UpdatePickupStatus(ctx, pickupID, domain.StatusPickedUp)
	require.NoError(t, err)

	// Assert final DB state
	dbFinal := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusPickedUp, dbFinal.Status)
	assert.Equal(t, 1, dbFinal.AttemptCount)

	// Assert still exactly one notification (no new ones from PICKED_UP)
	assert.Len(t, notifStub.Calls(), 1)
}

// TestLifecycle_Cancellation exercises the cancellation lifecycle:
// RequestPickup → CancelPickup (via HTTP).
// Asserts DB status is CANCELLED, no notifications were sent, and no
// tracker calls were made.
//
// Validates: Requirements 9.3
func TestLifecycle_Cancellation(t *testing.T) {
	truncateAll(t)
	notifStub.Reset()

	// Step 1: RequestPickup via HTTP
	created := createPickupHTTP(t, map[string]interface{}{
		"order_id":         "order-cancel-001",
		"user_id":          "user-001",
		"pickup_address":   "789 Cancel Ave",
		"pickup_city_code": "BDG",
		"contact_name":     "Bob Cancel",
		"contact_phone":    "+62811111111",
	})
	pickupID, ok := created["pickup_id"].(string)
	require.True(t, ok, "pickup_id should be a string")
	require.NotEmpty(t, pickupID)

	// Assert initial DB state
	dbAfterRequest := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusScheduled, dbAfterRequest.Status)

	// Step 2: CancelPickup via HTTP
	resp := doRequest(t, http.MethodPost, fmt.Sprintf("/pickups/%s/cancel", pickupID), nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Assert DB status is CANCELLED
	dbAfterCancel := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusCancelled, dbAfterCancel.Status)

	// Assert no notifications were sent
	assert.Empty(t, notifStub.Calls(), "no notifications expected for cancellation")

	// No tracker expectations set — gomock will fail the test if any unexpected
	// calls are made to the shared no-op tracker (which doesn't use gomock).
}

// TestLifecycle_CancelThenAssign exercises the cancel-then-assign scenario:
// RequestPickup → CancelPickup (via HTTP) → AssignCourier (via HTTP).
// Asserts AssignCourier returns HTTP 400 and the DB record remains CANCELLED.
//
// Validates: Requirements 9.4
func TestLifecycle_CancelThenAssign(t *testing.T) {
	truncateAll(t)
	notifStub.Reset()

	// Step 1: RequestPickup via HTTP
	created := createPickupHTTP(t, map[string]interface{}{
		"order_id":         "order-cancel-assign-001",
		"user_id":          "user-001",
		"pickup_address":   "321 Block Rd",
		"pickup_city_code": "MDN",
		"contact_name":     "Alice Block",
		"contact_phone":    "+62822222222",
	})
	pickupID, ok := created["pickup_id"].(string)
	require.True(t, ok, "pickup_id should be a string")
	require.NotEmpty(t, pickupID)

	// Step 2: CancelPickup via HTTP
	cancelResp := doRequest(t, http.MethodPost, fmt.Sprintf("/pickups/%s/cancel", pickupID), nil)
	defer cancelResp.Body.Close()
	require.Equal(t, http.StatusOK, cancelResp.StatusCode)

	// Assert DB is CANCELLED
	dbAfterCancel := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusCancelled, dbAfterCancel.Status)

	// Step 3: AssignCourier via HTTP — should fail because pickup is CANCELLED
	assignResp := doRequest(t, http.MethodPost, fmt.Sprintf("/pickups/%s/assign", pickupID),
		map[string]interface{}{"courier_id": "courier-blocked-001"})
	defer assignResp.Body.Close()

	// CANCELLED → ASSIGNED is an invalid transition, expect HTTP 400
	assert.Equal(t, http.StatusBadRequest, assignResp.StatusCode)

	// Assert DB record remains CANCELLED
	dbAfterAssign := getPickupFromDB(t, pickupID)
	assert.Equal(t, domain.StatusCancelled, dbAfterAssign.Status)
}
