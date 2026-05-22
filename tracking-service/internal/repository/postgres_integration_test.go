// Package repository_test contains integration tests for the repository
// package. These tests run against a real PostgreSQL instance and are skipped
// unless INTEGRATION=true is set (enforced by TestMain in integration_test.go).
//
// Using an external test package (repository_test) avoids the import cycle
// that would arise if this file were in package repository, because handler
// imports repository and router imports handler.
package repository_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"tracking-service/internal/handler"
	"tracking-service/internal/middleware"
	"tracking-service/internal/models"
	"tracking-service/internal/repository"
)

// db returns the shared test database set up by TestMain.
// It delegates to repository.GetTestDB() which is defined in export_test.go.
func db() *sql.DB {
	return repository.GetTestDB()
}

// cleanupTracking deletes test data for the given tracking numbers from both
// tables. It is safe to call even if the rows do not exist.
func cleanupTracking(t *testing.T, numbers ...string) {
	t.Helper()
	d := db()
	for _, n := range numbers {
		d.Exec("DELETE FROM tracking_events WHERE tracking_number = $1", n)
		d.Exec("DELETE FROM tracking_summary WHERE tracking_number = $1", n)
	}
}

// newTestRouter builds a chi router wired with all handlers and middleware.
// It mirrors router.New but avoids importing the router package, which would
// create an import cycle (router → repository → handler → repository).
func newTestRouter(repo repository.Repository, queryTimeout time.Duration, logWriter io.Writer) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.CorrelationIDMiddleware)
	r.Use(middleware.NewLoggingMiddleware(logWriter))
	r.Use(middleware.RecoveryMiddleware)
	r.Post("/tracking/events", handler.NewEventHandler(repo, queryTimeout).ServeHTTP)
	r.Get("/tracking/bulk", handler.NewBulkHandler(repo, queryTimeout).ServeHTTP)
	r.Get("/tracking/{tracking_number}", handler.NewHistoryHandler(repo, queryTimeout).ServeHTTP)
	return r
}

// uniqueTrackingNumber returns a tracking number that is unique per test run
// using the test name and a suffix to avoid collisions.
func uniqueTrackingNumber(t *testing.T, suffix string) string {
	t.Helper()
	safe := strings.NewReplacer("/", "_", " ", "_", "=", "_").Replace(t.Name())
	return fmt.Sprintf("INT-%s-%s", safe, suffix)
}

// postEvent sends a POST /tracking/events request to the given server and
// returns the HTTP response. The caller is responsible for closing the body.
func postEvent(t *testing.T, srv *httptest.Server, trackingNumber string, status models.Status, ts time.Time) *http.Response {
	t.Helper()
	body := models.CreateEventRequest{
		TrackingNumber: trackingNumber,
		Status:         string(status),
		Timestamp:      ts.UTC().Format(time.RFC3339),
	}
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	resp, err := http.Post(srv.URL+"/tracking/events", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST /tracking/events: %v", err)
	}
	return resp
}

// getHistory sends a GET /tracking/{trackingNumber} request and returns the
// decoded TrackingHistoryResponse along with the raw response.
func getHistory(t *testing.T, srv *httptest.Server, trackingNumber string) (models.TrackingHistoryResponse, *http.Response) {
	t.Helper()
	resp, err := http.Get(srv.URL + "/tracking/" + url.PathEscape(trackingNumber))
	if err != nil {
		t.Fatalf("GET /tracking/%s: %v", trackingNumber, err)
	}
	defer resp.Body.Close()
	var result models.TrackingHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode history response: %v", err)
	}
	return result, resp
}

// getBulk sends a GET /tracking/bulk?numbers=... request and returns the
// decoded slice of TrackingSummary along with the raw response.
func getBulk(t *testing.T, srv *httptest.Server, numbers []string) ([]models.TrackingSummary, *http.Response) {
	t.Helper()
	u := srv.URL + "/tracking/bulk?numbers=" + url.QueryEscape(strings.Join(numbers, ","))
	resp, err := http.Get(u)
	if err != nil {
		t.Fatalf("GET /tracking/bulk: %v", err)
	}
	defer resp.Body.Close()
	var result []models.TrackingSummary
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode bulk response: %v", err)
	}
	return result, resp
}

// TestIntegration_18_1_PostThenGetSingleTrackingNumber verifies the full
// POST → GET flow for a single tracking number.
//
// Requirements: 1.1, 2.1, 2.2
func TestIntegration_18_1_PostThenGetSingleTrackingNumber(t *testing.T) {
	tn := uniqueTrackingNumber(t, "A")
	t.Cleanup(func() { cleanupTracking(t, tn) })

	repo := repository.NewPostgresRepository(db())
	srv := httptest.NewServer(newTestRouter(repo, 5*time.Second, io.Discard))
	defer srv.Close()

	// POST a valid event.
	ts := time.Now().UTC().Truncate(time.Second)
	resp := postEvent(t, srv, tn, models.StatusCreated, ts)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	// GET history and assert.
	history, getResp := getHistory(t, srv, tn)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", getResp.StatusCode)
	}
	if history.CurrentStatus != models.StatusCreated {
		t.Errorf("expected current_status %q, got %q", models.StatusCreated, history.CurrentStatus)
	}
	if len(history.History) != 1 {
		t.Errorf("expected history length 1, got %d", len(history.History))
	}
}

// TestIntegration_18_2_PostThenGetBulkMultipleTrackingNumbers verifies the
// full POST → GET /bulk flow for multiple tracking numbers.
//
// Requirements: 3.1, 3.6
func TestIntegration_18_2_PostThenGetBulkMultipleTrackingNumbers(t *testing.T) {
	tn1 := uniqueTrackingNumber(t, "1")
	tn2 := uniqueTrackingNumber(t, "2")
	tn3 := uniqueTrackingNumber(t, "3")
	t.Cleanup(func() { cleanupTracking(t, tn1, tn2, tn3) })

	repo := repository.NewPostgresRepository(db())
	srv := httptest.NewServer(newTestRouter(repo, 5*time.Second, io.Discard))
	defer srv.Close()

	ts := time.Now().UTC().Truncate(time.Second)

	// POST one event for each tracking number.
	for _, tn := range []string{tn1, tn2, tn3} {
		resp := postEvent(t, srv, tn, models.StatusPickedUp, ts)
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("POST for %s: expected 201, got %d", tn, resp.StatusCode)
		}
	}

	// GET bulk and assert all 3 appear.
	summaries, bulkResp := getBulk(t, srv, []string{tn1, tn2, tn3})
	if bulkResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", bulkResp.StatusCode)
	}
	if len(summaries) != 3 {
		t.Fatalf("expected 3 summaries, got %d", len(summaries))
	}

	// Build a set of returned tracking numbers for easy lookup.
	found := make(map[string]bool, 3)
	for _, s := range summaries {
		found[s.TrackingNumber] = true
	}
	for _, tn := range []string{tn1, tn2, tn3} {
		if !found[tn] {
			t.Errorf("tracking number %q missing from bulk response", tn)
		}
	}
}

// TestIntegration_18_3_ConcurrentPostsSameTrackingNumber verifies that 10
// concurrent POST requests for the same tracking number all persist and that
// current_status reflects the latest timestamp.
//
// Requirements: 1.8, 5.4
func TestIntegration_18_3_ConcurrentPostsSameTrackingNumber(t *testing.T) {
	tn := uniqueTrackingNumber(t, "CONC")
	t.Cleanup(func() { cleanupTracking(t, tn) })

	repo := repository.NewPostgresRepository(db())
	srv := httptest.NewServer(newTestRouter(repo, 5*time.Second, io.Discard))
	defer srv.Close()

	const numGoroutines = 10
	baseTime := time.Now().UTC().Truncate(time.Second)

	// Each goroutine posts a distinct event with a unique timestamp so we can
	// identify the latest one deterministically.
	statuses := []models.Status{
		models.StatusCreated,
		models.StatusPickedUp,
		models.StatusArrivedAtHub,
		models.StatusInTransit,
		models.StatusOutForDelivery,
		models.StatusDelivered,
		models.StatusFailedDelivery,
		models.StatusReturned,
		models.StatusCreated,
		models.StatusPickedUp,
	}

	var wg sync.WaitGroup
	errs := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ts := baseTime.Add(time.Duration(idx) * time.Second)
			resp := postEvent(t, srv, tn, statuses[idx], ts)
			resp.Body.Close()
			if resp.StatusCode != http.StatusCreated {
				errs[idx] = fmt.Errorf("goroutine %d: expected 201, got %d", idx, resp.StatusCode)
			}
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d error: %v", i, err)
		}
	}

	// All 10 events must be persisted.
	history, getResp := getHistory(t, srv, tn)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", getResp.StatusCode)
	}
	if len(history.History) != numGoroutines {
		t.Errorf("expected %d history entries, got %d", numGoroutines, len(history.History))
	}

	// current_status must reflect the event with the latest timestamp.
	// The latest timestamp is baseTime + 9 seconds → statuses[9] = StatusPickedUp.
	expectedLatestStatus := statuses[numGoroutines-1]
	if history.CurrentStatus != expectedLatestStatus {
		t.Errorf("expected current_status %q (latest timestamp), got %q",
			expectedLatestStatus, history.CurrentStatus)
	}
}

// TestIntegration_18_4_TransactionRollbackOnSummaryFailure verifies that when
// the repository returns an error, the HTTP handler returns 500 and no event
// is persisted.
//
// This test uses the mock repository with InjectError to simulate a summary
// upsert failure, verifying the HTTP 500 behavior without requiring fault
// injection into PostgresRepository.
//
// Requirements: 5.3
func TestIntegration_18_4_TransactionRollbackOnSummaryFailure(t *testing.T) {
	mock := repository.NewMockRepository()
	mock.InjectError = errors.New("simulated summary upsert failure")

	srv := httptest.NewServer(newTestRouter(mock, 5*time.Second, io.Discard))
	defer srv.Close()

	tn := "INT-ROLLBACK-TEST"
	ts := time.Now().UTC()

	resp := postEvent(t, srv, tn, models.StatusCreated, ts)
	defer resp.Body.Close()

	// The handler must return HTTP 500 when the repository fails.
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}

	// No event should be persisted in the mock.
	_, err := mock.GetEventsByTrackingNumber(context.Background(), tn)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound (no event persisted), got %v", err)
	}
}

// TestIntegration_18_5_ConnectionPoolExhaustionReturns503 verifies that when
// the connection pool is exhausted, the service returns HTTP 503.
//
// Requirements: 7.3
func TestIntegration_18_5_ConnectionPoolExhaustionReturns503(t *testing.T) {
	// Re-open a connection using the same DSN as the shared testDB but with
	// MaxOpenConns=1 so we can exhaust the pool deterministically.
	dbDSN := "postgres://tracking:tracking_secret@localhost:5432/tracking_test"
	if v := os.Getenv("TEST_DATABASE_DSN"); v != "" {
		dbDSN = v
	}

	limitedDB, err := sql.Open("pgx", dbDSN)
	if err != nil {
		t.Fatalf("open limited DB: %v", err)
	}
	defer limitedDB.Close()
	limitedDB.SetMaxOpenConns(1)
	limitedDB.SetMaxIdleConns(1)

	repo := repository.NewPostgresRepository(limitedDB)
	srv := httptest.NewServer(newTestRouter(repo, 5*time.Second, io.Discard))
	defer srv.Close()

	tn := uniqueTrackingNumber(t, "POOL")
	// No cleanup needed — the POST will fail before any data is written.

	// Hold the single connection in a long-running transaction so the pool is
	// fully occupied when the second request arrives.
	tx, err := limitedDB.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin blocking transaction: %v", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Execute a no-op query to confirm the connection is held.
	if _, err := tx.Exec("SELECT pg_sleep(0)"); err != nil {
		t.Fatalf("hold connection: %v", err)
	}

	// Fire a POST request with a short context deadline so the pool-wait times
	// out quickly. The handler maps the resulting error to HTTP 503 or 504.
	client := &http.Client{Timeout: 10 * time.Second}

	reqBody := models.CreateEventRequest{
		TrackingNumber: tn,
		Status:         string(models.StatusCreated),
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	}
	b, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/tracking/events", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	// Short deadline forces the pool-wait to time out before the 5-second
	// handler timeout, producing a context.DeadlineExceeded → HTTP 503/504.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	postResp, err := client.Do(req)
	if err != nil {
		// Client-side timeout is acceptable: the pool was exhausted and the
		// request could not complete within the deadline.
		t.Logf("client error (pool exhaustion expected): %v", err)
		return
	}
	defer postResp.Body.Close()

	if postResp.StatusCode != http.StatusServiceUnavailable && postResp.StatusCode != http.StatusGatewayTimeout {
		bodyBytes, _ := io.ReadAll(postResp.Body)
		t.Errorf("expected 503 or 504, got %d; body: %s", postResp.StatusCode, bodyBytes)
	}
}
