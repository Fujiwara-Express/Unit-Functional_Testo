package functional_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/pickup-service/internal/client"
	"github.com/pickup-service/internal/domain"
	"github.com/pickup-service/internal/repository"
	"github.com/pickup-service/internal/service"
	"github.com/pickup-service/test/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"
)

// truncateAll clears all tables before each test for isolation.
func truncateAll(t *testing.T) {
	t.Helper()
	_, err := testDB.ExecContext(context.Background(),
		"TRUNCATE TABLE pickup_attempts, pickups RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

// createPickupHTTP posts to POST /pickups with a Bearer token and asserts HTTP 201.
// Returns the decoded response body.
func createPickupHTTP(t *testing.T, payload map[string]interface{}) map[string]interface{} {
	t.Helper()
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, testServer.URL+"/pickups", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result
}

// getPickupFromDB reads a pickup record directly from the DB by pickup_id.
func getPickupFromDB(t *testing.T, pickupID string) *domain.Pickup {
	t.Helper()
	row := testDB.QueryRowContext(context.Background(),
		`SELECT pickup_id, order_id, user_id, courier_id, status,
		        pickup_address, pickup_city_code, contact_name, contact_phone,
		        attempt_count, estimated_pickup_time, created_at, updated_at
		 FROM pickups WHERE pickup_id = $1`, pickupID)
	p := &domain.Pickup{}
	var status string
	err := row.Scan(
		&p.PickupID, &p.OrderID, &p.UserID, &p.CourierID, &status,
		&p.PickupAddress, &p.PickupCityCode, &p.ContactName, &p.ContactPhone,
		&p.AttemptCount, &p.EstimatedPickupTime, &p.CreatedAt, &p.UpdatedAt,
	)
	require.NoError(t, err)
	p.Status = domain.Status(status)
	return p
}

// doRequest sends an authenticated HTTP request to the testServer.
func doRequest(t *testing.T, method, path string, body interface{}) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, testServer.URL+path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// newTestEnv creates a fresh PickupService wired with a new MockTrackingClient.
// Use this for tests that need to assert on TrackingClient calls.
func newTestEnv(t *testing.T) (service.PickupService, *mocks.MockTrackingClient) {
	t.Helper()
	ctrl := gomock.NewController(t)
	tracker := mocks.NewMockTrackingClient(ctrl)
	repo := repository.NewPickupRepository(testDB)
	svc := service.NewPickupService(
		repo,
		client.New(deliveryStub.URL()),
		tracker,
		client.NewHTTPNotificationClient(notifStub.URL()),
	)
	return svc, tracker
}

// countPickups returns the total number of rows in the pickups table.
func countPickups(t *testing.T) int {
	t.Helper()
	var n int
	_ = testDB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM pickups").Scan(&n)
	return n
}

// validPickupPayload returns a complete valid RequestPickup payload.
func validPickupPayload() map[string]interface{} {
	return map[string]interface{}{
		"order_id":         "order-001",
		"user_id":          "user-001",
		"pickup_address":   "123 Main St",
		"pickup_city_code": "JKT",
		"contact_name":     "John Doe",
		"contact_phone":    "+62812345678",
	}
}

// genPickupRequest generates a random valid RequestPickup payload using rapid.
// Uses safe alphanumeric characters to avoid PostgreSQL null-byte rejections.
func genPickupRequest(t *rapid.T) map[string]interface{} {
	return map[string]interface{}{
		"order_id":         "order-" + rapid.StringMatching(`[a-zA-Z0-9]{1,20}`).Draw(t, "order_id"),
		"user_id":          "user-" + rapid.StringMatching(`[a-zA-Z0-9]{1,20}`).Draw(t, "user_id"),
		"pickup_address":   rapid.StringMatching(`[a-zA-Z0-9 ]{5,50}`).Draw(t, "address"),
		"pickup_city_code": rapid.SampledFrom([]string{"JKT", "SBY", "BDG", "MDN"}).Draw(t, "city"),
		"contact_name":     rapid.StringMatching(`[a-zA-Z ]{2,30}`).Draw(t, "name"),
		"contact_phone":    "+628" + rapid.StringMatching(`[0-9]{8,11}`).Draw(t, "phone"),
	}
}

// genCourierID generates a random courier ID string using rapid.
// Uses safe alphanumeric characters to avoid PostgreSQL null-byte rejections.
func genCourierID(t *rapid.T) string {
	return "courier-" + rapid.StringMatching(`[a-zA-Z0-9]{1,20}`).Draw(t, "courier_id")
}
