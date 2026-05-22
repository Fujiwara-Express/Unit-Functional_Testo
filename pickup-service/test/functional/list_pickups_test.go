package functional_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/pickup-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestListPickups_NoFilter verifies that GET /pickups with no filters returns
// HTTP 200 and all pickups in the database.
func TestListPickups_NoFilter(t *testing.T) {
	truncateAll(t)

	// Insert 3 pickups
	payloads := []map[string]interface{}{
		{
			"order_id": "order-001", "user_id": "user-001",
			"pickup_address": "123 Main St", "pickup_city_code": "JKT",
			"contact_name": "Alice", "contact_phone": "+62812345678",
		},
		{
			"order_id": "order-002", "user_id": "user-002",
			"pickup_address": "456 Oak Ave", "pickup_city_code": "SBY",
			"contact_name": "Bob", "contact_phone": "+62812345679",
		},
		{
			"order_id": "order-003", "user_id": "user-003",
			"pickup_address": "789 Pine Rd", "pickup_city_code": "BDG",
			"contact_name": "Carol", "contact_phone": "+62812345680",
		},
	}
	for _, p := range payloads {
		createPickupHTTP(t, p)
	}

	// GET /pickups with no filters
	resp := doRequest(t, http.MethodGet, "/pickups", nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body []map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	assert.Len(t, body, 3, "expected 3 pickups returned")
}

// TestListPickups_FilterByCourierID verifies that GET /pickups?courier_id={id}
// returns only pickups assigned to that courier.
func TestListPickups_FilterByCourierID(t *testing.T) {
	truncateAll(t)

	// Create 3 pickups
	p1 := createPickupHTTP(t, map[string]interface{}{
		"order_id": "order-101", "user_id": "user-001",
		"pickup_address": "1 Alpha St", "pickup_city_code": "JKT",
		"contact_name": "Alice", "contact_phone": "+62812345678",
	})
	p2 := createPickupHTTP(t, map[string]interface{}{
		"order_id": "order-102", "user_id": "user-002",
		"pickup_address": "2 Beta St", "pickup_city_code": "JKT",
		"contact_name": "Bob", "contact_phone": "+62812345679",
	})
	createPickupHTTP(t, map[string]interface{}{
		"order_id": "order-103", "user_id": "user-003",
		"pickup_address": "3 Gamma St", "pickup_city_code": "JKT",
		"contact_name": "Carol", "contact_phone": "+62812345680",
	})

	id1 := p1["pickup_id"].(string)
	id2 := p2["pickup_id"].(string)

	// Assign courier-A to pickup 1 and pickup 2, leave pickup 3 unassigned
	svc, _ := newTestEnv(t)
	ctx := context.Background()
	_, err := svc.AssignCourier(ctx, id1, "courier-A")
	require.NoError(t, err)
	_, err = svc.AssignCourier(ctx, id2, "courier-A")
	require.NoError(t, err)

	// GET /pickups?courier_id=courier-A
	resp := doRequest(t, http.MethodGet, "/pickups?courier_id=courier-A", nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body []map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	assert.Len(t, body, 2, "expected 2 pickups assigned to courier-A")
	for _, item := range body {
		assert.Equal(t, "courier-A", item["courier_id"])
	}
}

// TestListPickups_FilterByStatus verifies that GET /pickups?status=SCHEDULED
// returns only pickups with that status.
func TestListPickups_FilterByStatus(t *testing.T) {
	truncateAll(t)

	// Create 3 pickups: 2 will remain SCHEDULED, 1 will be ASSIGNED
	p1 := createPickupHTTP(t, map[string]interface{}{
		"order_id": "order-201", "user_id": "user-001",
		"pickup_address": "1 Alpha St", "pickup_city_code": "JKT",
		"contact_name": "Alice", "contact_phone": "+62812345678",
	})
	createPickupHTTP(t, map[string]interface{}{
		"order_id": "order-202", "user_id": "user-002",
		"pickup_address": "2 Beta St", "pickup_city_code": "JKT",
		"contact_name": "Bob", "contact_phone": "+62812345679",
	})
	p3 := createPickupHTTP(t, map[string]interface{}{
		"order_id": "order-203", "user_id": "user-003",
		"pickup_address": "3 Gamma St", "pickup_city_code": "JKT",
		"contact_name": "Carol", "contact_phone": "+62812345680",
	})

	// Assign courier to p1 and p3 to make them ASSIGNED
	svc, _ := newTestEnv(t)
	ctx := context.Background()
	_, err := svc.AssignCourier(ctx, p1["pickup_id"].(string), "courier-X")
	require.NoError(t, err)
	_, err = svc.AssignCourier(ctx, p3["pickup_id"].(string), "courier-Y")
	require.NoError(t, err)

	// GET /pickups?status=SCHEDULED — should return only pickup 2
	resp := doRequest(t, http.MethodGet, "/pickups?status=SCHEDULED", nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body []map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	assert.Len(t, body, 1, "expected 1 SCHEDULED pickup")
	assert.Equal(t, "SCHEDULED", body[0]["status"])

	// GET /pickups?status=ASSIGNED — should return 2 pickups
	resp2 := doRequest(t, http.MethodGet, "/pickups?status=ASSIGNED", nil)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var body2 []map[string]interface{}
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&body2))

	assert.Len(t, body2, 2, "expected 2 ASSIGNED pickups")
	for _, item := range body2 {
		assert.Equal(t, "ASSIGNED", item["status"])
	}
}

// TestListPickups_FilterByDate verifies that GET /pickups?date={today} returns
// only pickups created on that date.
func TestListPickups_FilterByDate(t *testing.T) {
	truncateAll(t)

	// Insert 2 pickups (both created today)
	createPickupHTTP(t, map[string]interface{}{
		"order_id": "order-301", "user_id": "user-001",
		"pickup_address": "1 Alpha St", "pickup_city_code": "JKT",
		"contact_name": "Alice", "contact_phone": "+62812345678",
	})
	createPickupHTTP(t, map[string]interface{}{
		"order_id": "order-302", "user_id": "user-002",
		"pickup_address": "2 Beta St", "pickup_city_code": "JKT",
		"contact_name": "Bob", "contact_phone": "+62812345679",
	})

	today := time.Now().UTC().Format("2006-01-02")

	// GET /pickups?date={today}
	resp := doRequest(t, http.MethodGet, "/pickups?date="+today, nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body []map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	assert.Len(t, body, 2, "expected 2 pickups created today")

	// GET /pickups?date=1970-01-01 — should return no pickups
	resp2 := doRequest(t, http.MethodGet, "/pickups?date=1970-01-01", nil)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	// The response may be null or [] for an empty result
	rawBody, err := readResponseBody(resp2)
	require.NoError(t, err)
	assertEmptyList(t, rawBody)
}

// TestListPickups_CombinedFilters verifies that GET /pickups?courier_id={id}&status=ASSIGNED
// returns only pickups matching both filters.
func TestListPickups_CombinedFilters(t *testing.T) {
	truncateAll(t)

	// Create 4 pickups
	p1 := createPickupHTTP(t, map[string]interface{}{
		"order_id": "order-401", "user_id": "user-001",
		"pickup_address": "1 Alpha St", "pickup_city_code": "JKT",
		"contact_name": "Alice", "contact_phone": "+62812345678",
	})
	p2 := createPickupHTTP(t, map[string]interface{}{
		"order_id": "order-402", "user_id": "user-002",
		"pickup_address": "2 Beta St", "pickup_city_code": "JKT",
		"contact_name": "Bob", "contact_phone": "+62812345679",
	})
	p3 := createPickupHTTP(t, map[string]interface{}{
		"order_id": "order-403", "user_id": "user-003",
		"pickup_address": "3 Gamma St", "pickup_city_code": "JKT",
		"contact_name": "Carol", "contact_phone": "+62812345680",
	})
	createPickupHTTP(t, map[string]interface{}{
		"order_id": "order-404", "user_id": "user-004",
		"pickup_address": "4 Delta St", "pickup_city_code": "JKT",
		"contact_name": "Dave", "contact_phone": "+62812345681",
	})

	svc, _ := newTestEnv(t)
	ctx := context.Background()

	// Assign courier-B to p1 and p2; assign courier-C to p3; leave p4 unassigned
	_, err := svc.AssignCourier(ctx, p1["pickup_id"].(string), "courier-B")
	require.NoError(t, err)
	_, err = svc.AssignCourier(ctx, p2["pickup_id"].(string), "courier-B")
	require.NoError(t, err)
	_, err = svc.AssignCourier(ctx, p3["pickup_id"].(string), "courier-C")
	require.NoError(t, err)

	// GET /pickups?courier_id=courier-B&status=ASSIGNED — should return p1 and p2
	resp := doRequest(t, http.MethodGet, "/pickups?courier_id=courier-B&status=ASSIGNED", nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body []map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	assert.Len(t, body, 2, "expected 2 pickups matching courier-B + ASSIGNED")
	for _, item := range body {
		assert.Equal(t, "courier-B", item["courier_id"])
		assert.Equal(t, "ASSIGNED", item["status"])
	}

	// GET /pickups?courier_id=courier-B&status=SCHEDULED — should return 0 (courier-B pickups are ASSIGNED)
	resp2 := doRequest(t, http.MethodGet, "/pickups?courier_id=courier-B&status=SCHEDULED", nil)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	rawBody2, err := readResponseBody(resp2)
	require.NoError(t, err)
	assertEmptyList(t, rawBody2)
}

// TestListPickups_EmptyResult verifies that GET /pickups?courier_id=nonexistent-courier
// returns HTTP 200 and an empty list (null or []).
func TestListPickups_EmptyResult(t *testing.T) {
	truncateAll(t)

	// Insert a pickup so the table is not empty, but with a different courier
	createPickupHTTP(t, validPickupPayload())

	resp := doRequest(t, http.MethodGet, "/pickups?courier_id=nonexistent-courier", nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	rawBody, err := readResponseBody(resp)
	require.NoError(t, err)
	assertEmptyList(t, rawBody)
}

// TestListPickups_FilterPredicate is a property-based test verifying that
// GET /pickups with a courier_id filter returns exactly the pickups assigned
// to that courier — no more, no fewer.
//
// Feature: pickup-service-functional-tests, Property 7: ListPickups filter predicate correctness —
// GET /pickups with filters returns exactly the pickups that satisfy all specified filter predicates.
//
// Validates: Requirements 6.1, 6.2, 6.3, 6.5
func TestListPickups_FilterPredicate(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		truncateAll(t)
		notifStub.Reset()

		// Generate N=2..5 pickups with random payloads and create them all
		n := rapid.IntRange(2, 5).Draw(rt, "n")
		pickupIDs := make([]string, n)
		for i := 0; i < n; i++ {
			payload := genPickupRequest(rt)
			created := createPickupHTTP(t, payload)
			id, ok := created["pickup_id"].(string)
			require.True(t, ok, "pickup_id should be a string")
			require.NotEmpty(t, id)
			pickupIDs[i] = id
		}

		// Randomly assign some pickups to couriers using newTestEnv
		svc, _ := newTestEnv(t)
		ctx := context.Background()

		// Track which pickup IDs are assigned to which courier
		courierAssignments := make(map[string]string) // pickupID -> courierID
		for _, pid := range pickupIDs {
			// Randomly decide whether to assign this pickup (50% chance)
			if rapid.Bool().Draw(rt, fmt.Sprintf("assign_%s", pid)) {
				cid := genCourierID(rt)
				_, err := svc.AssignCourier(ctx, pid, cid)
				require.NoError(t, err)
				courierAssignments[pid] = cid
			}
		}

		// Pick a random courier_id to filter by (from assigned couriers, or a fresh one)
		var filterCourierID string
		if len(courierAssignments) > 0 && rapid.Bool().Draw(rt, "use_existing_courier") {
			// Pick one of the assigned courier IDs
			for _, cid := range courierAssignments {
				filterCourierID = cid
				break
			}
		} else {
			filterCourierID = genCourierID(rt)
		}

		// Compute expected pickup IDs: those assigned to filterCourierID
		expectedIDs := make(map[string]bool)
		for pid, cid := range courierAssignments {
			if cid == filterCourierID {
				expectedIDs[pid] = true
			}
		}

		// GET /pickups?courier_id={filterCourierID}
		resp := doRequest(t, http.MethodGet, "/pickups?courier_id="+filterCourierID, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Decode response — may be null (empty) or an array
		bodyBytes, err := readResponseBody(resp)
		require.NoError(t, err)

		var returnedPickups []map[string]interface{}
		if string(bodyBytes) != "null\n" && string(bodyBytes) != "null" && len(bodyBytes) > 0 {
			require.NoError(t, json.Unmarshal(bodyBytes, &returnedPickups))
		}

		// Assert the response contains exactly the expected pickups
		assert.Len(t, returnedPickups, len(expectedIDs),
			"expected %d pickups for courier %s, got %d", len(expectedIDs), filterCourierID, len(returnedPickups))

		returnedIDs := make(map[string]bool)
		for _, item := range returnedPickups {
			pid, ok := item["pickup_id"].(string)
			require.True(t, ok, "pickup_id should be a string in list response")
			returnedIDs[pid] = true
			// Each returned pickup must have the correct courier_id
			assert.Equal(t, filterCourierID, item["courier_id"],
				"pickup %s should have courier_id %s", pid, filterCourierID)
		}

		// Every expected pickup must appear in the response
		for pid := range expectedIDs {
			assert.True(t, returnedIDs[pid],
				"pickup %s assigned to courier %s should appear in response", pid, filterCourierID)
		}
	})
}

// readResponseBody reads the full body of an HTTP response as bytes.
func readResponseBody(resp *http.Response) ([]byte, error) {
	var buf []byte
	tmp := make([]byte, 512)
	for {
		n, err := resp.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	return buf, nil
}

// assertEmptyList asserts that the response body represents an empty list
// (either JSON null or an empty JSON array []).
func assertEmptyList(t *testing.T, body []byte) {
	t.Helper()
	trimmed := string(body)
	// Strip trailing newline
	for len(trimmed) > 0 && (trimmed[len(trimmed)-1] == '\n' || trimmed[len(trimmed)-1] == '\r') {
		trimmed = trimmed[:len(trimmed)-1]
	}
	if trimmed == "null" || trimmed == "[]" {
		return
	}
	// Try to decode as array and check it's empty
	var arr []interface{}
	if err := json.Unmarshal([]byte(trimmed), &arr); err == nil {
		assert.Empty(t, arr, "expected empty list, got %s", trimmed)
		return
	}
	t.Errorf("expected empty list (null or []), got: %s", trimmed)
}

// Ensure domain package is used (for status constants referenced in test logic).
var _ = domain.StatusScheduled
