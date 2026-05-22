// Package handler — functional tests.
//
// These tests exercise the full HTTP API end-to-end through the real router
// (all middleware + all routes wired together) using fixed, readable scenarios.
// They complement the property-based tests by providing clear, named examples
// that document expected behaviour and catch regressions.
//
// All tests use the in-memory MockRepository so no database is required.
package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"tracking-service/internal/middleware"
	"tracking-service/internal/models"
	"tracking-service/internal/repository"
)

// ---------------------------------------------------------------------------
// Test server helpers
// ---------------------------------------------------------------------------

// newFullRouter wires all handlers and middleware into a chi router, mirroring
// router.New but staying inside the handler package to avoid an import cycle.
func newFullRouter(repo *repository.MockRepository) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.CorrelationIDMiddleware)
	r.Use(middleware.NewLoggingMiddleware(io.Discard))
	r.Use(middleware.RecoveryMiddleware)
	r.Post("/tracking/events", NewEventHandler(repo, 5*time.Second).ServeHTTP)
	r.Get("/tracking/bulk", NewBulkHandler(repo, 5*time.Second).ServeHTTP)
	r.Get("/tracking/{tracking_number}", NewHistoryHandler(repo, 5*time.Second).ServeHTTP)
	r.Get("/health", NewHealthHandler(repo).ServeHTTP)
	return r
}

// newFunctionalServer returns an httptest.Server backed by the full router.
func newFunctionalServer(t *testing.T) (*httptest.Server, *repository.MockRepository) {
	t.Helper()
	repo := repository.NewMockRepository()
	srv := httptest.NewServer(newFullRouter(repo))
	t.Cleanup(srv.Close)
	return srv, repo
}

// doPost sends a POST /tracking/events request and returns the response.
func doPost(t *testing.T, srv *httptest.Server, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	resp, err := http.Post(srv.URL+"/tracking/events", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST /tracking/events: %v", err)
	}
	return resp
}

// doGet sends a GET request to the given path and returns the response.
func doGet(t *testing.T, srv *httptest.Server, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(srv.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

// decodeJSON decodes the response body into dst and closes the body.
func decodeJSON(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decode JSON response: %v", err)
	}
}

// assertStatus fails the test if the response status code does not match want.
func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected HTTP %d, got %d; body: %s", want, resp.StatusCode, body)
	}
}

// assertContentType fails if Content-Type does not contain "application/json".
func assertContentType(t *testing.T, resp *http.Response) {
	t.Helper()
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

// assertCorrelationID fails if X-Correlation-ID header is absent or empty.
func assertCorrelationID(t *testing.T, resp *http.Response) string {
	t.Helper()
	id := resp.Header.Get("X-Correlation-ID")
	if id == "" {
		t.Error("X-Correlation-ID response header is missing or empty")
	}
	return id
}

// ---------------------------------------------------------------------------
// POST /tracking/events — happy path
// ---------------------------------------------------------------------------

func TestFunctional_PostEvent_ValidRequest_Returns201(t *testing.T) {
	srv, _ := newFunctionalServer(t)

	req := models.CreateEventRequest{
		TrackingNumber: "TRK-001",
		Status:         "CREATED",
		Timestamp:      "2024-06-01T10:00:00Z",
		Location:       "Warsaw Warehouse",
		HubID:          "HUB-WA",
		Notes:          "Package received",
	}

	resp := doPost(t, srv, req)
	assertStatus(t, resp, http.StatusCreated)
	assertContentType(t, resp)
	assertCorrelationID(t, resp)

	var body models.CreateEventResponse
	decodeJSON(t, resp, &body)

	if body.EventID == "" {
		t.Error("event_id in response is empty")
	}
	if !isUUIDLike(body.EventID) {
		t.Errorf("event_id %q does not look like a UUID", body.EventID)
	}
}

func TestFunctional_PostEvent_SetsCreatedByService(t *testing.T) {
	srv, repo := newFunctionalServer(t)

	req := models.CreateEventRequest{
		TrackingNumber: "TRK-002",
		Status:         "PICKED_UP",
		Timestamp:      "2024-06-01T11:00:00Z",
	}

	resp := doPost(t, srv, req)
	assertStatus(t, resp, http.StatusCreated)

	var createResp models.CreateEventResponse
	decodeJSON(t, resp, &createResp)

	events, err := repo.GetEventsByTrackingNumber(t.Context(), "TRK-002")
	if err != nil {
		t.Fatalf("GetEventsByTrackingNumber: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 stored event, got %d", len(events))
	}
	if events[0].CreatedByService != "tracking-service" {
		t.Errorf("created_by_service: want %q, got %q", "tracking-service", events[0].CreatedByService)
	}
}

// ---------------------------------------------------------------------------
// POST /tracking/events — validation errors
// ---------------------------------------------------------------------------

func TestFunctional_PostEvent_MissingTrackingNumber_Returns400(t *testing.T) {
	srv, _ := newFunctionalServer(t)

	req := models.CreateEventRequest{
		Status:    "CREATED",
		Timestamp: "2024-06-01T10:00:00Z",
	}

	resp := doPost(t, srv, req)
	assertStatus(t, resp, http.StatusBadRequest)
	assertContentType(t, resp)

	var body models.ErrorResponse
	decodeJSON(t, resp, &body)

	if !strings.Contains(body.Error, "tracking_number") {
		t.Errorf("error message should mention 'tracking_number', got: %q", body.Error)
	}
	if body.CorrelationID == "" {
		t.Error("error response missing correlation_id")
	}
}

func TestFunctional_PostEvent_MissingStatus_Returns400(t *testing.T) {
	srv, _ := newFunctionalServer(t)

	req := models.CreateEventRequest{
		TrackingNumber: "TRK-003",
		Timestamp:      "2024-06-01T10:00:00Z",
	}

	resp := doPost(t, srv, req)
	assertStatus(t, resp, http.StatusBadRequest)

	var body models.ErrorResponse
	decodeJSON(t, resp, &body)

	if !strings.Contains(body.Error, "status") {
		t.Errorf("error message should mention 'status', got: %q", body.Error)
	}
}

func TestFunctional_PostEvent_MissingTimestamp_Returns400(t *testing.T) {
	srv, _ := newFunctionalServer(t)

	req := models.CreateEventRequest{
		TrackingNumber: "TRK-004",
		Status:         "CREATED",
	}

	resp := doPost(t, srv, req)
	assertStatus(t, resp, http.StatusBadRequest)

	var body models.ErrorResponse
	decodeJSON(t, resp, &body)

	if !strings.Contains(body.Error, "timestamp") {
		t.Errorf("error message should mention 'timestamp', got: %q", body.Error)
	}
}

func TestFunctional_PostEvent_AllRequiredFieldsMissing_Returns400WithAllFields(t *testing.T) {
	srv, _ := newFunctionalServer(t)

	resp := doPost(t, srv, models.CreateEventRequest{})
	assertStatus(t, resp, http.StatusBadRequest)

	var body models.ErrorResponse
	decodeJSON(t, resp, &body)

	for _, field := range []string{"tracking_number", "status", "timestamp"} {
		if !strings.Contains(body.Error, field) {
			t.Errorf("error message should mention %q; got: %q", field, body.Error)
		}
	}
}

func TestFunctional_PostEvent_InvalidStatus_Returns422(t *testing.T) {
	srv, _ := newFunctionalServer(t)

	req := models.CreateEventRequest{
		TrackingNumber: "TRK-005",
		Status:         "FLYING",
		Timestamp:      "2024-06-01T10:00:00Z",
	}

	resp := doPost(t, srv, req)
	assertStatus(t, resp, http.StatusUnprocessableEntity)
	assertContentType(t, resp)

	var body models.ErrorResponse
	decodeJSON(t, resp, &body)

	if !strings.Contains(body.Error, "FLYING") {
		t.Errorf("error message should mention the invalid value 'FLYING', got: %q", body.Error)
	}
}

func TestFunctional_PostEvent_InvalidTimestamp_Returns422(t *testing.T) {
	srv, _ := newFunctionalServer(t)

	req := models.CreateEventRequest{
		TrackingNumber: "TRK-006",
		Status:         "CREATED",
		Timestamp:      "01/06/2024 10:00",
	}

	resp := doPost(t, srv, req)
	assertStatus(t, resp, http.StatusUnprocessableEntity)

	var body models.ErrorResponse
	decodeJSON(t, resp, &body)

	if !strings.Contains(strings.ToLower(body.Error), "rfc 3339") {
		t.Errorf("error message should mention RFC 3339, got: %q", body.Error)
	}
}

func TestFunctional_PostEvent_MalformedJSON_Returns400(t *testing.T) {
	srv, _ := newFunctionalServer(t)

	resp, err := http.Post(srv.URL+"/tracking/events", "application/json",
		strings.NewReader(`{not valid json`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	assertStatus(t, resp, http.StatusBadRequest)
	assertContentType(t, resp)

	var body models.ErrorResponse
	decodeJSON(t, resp, &body)

	if !strings.Contains(strings.ToLower(body.Error), "malformed") {
		t.Errorf("error message should mention 'malformed', got: %q", body.Error)
	}
}

// ---------------------------------------------------------------------------
// POST /tracking/events — error paths
// ---------------------------------------------------------------------------

func TestFunctional_PostEvent_RepositoryError_Returns500(t *testing.T) {
	srv, repo := newFunctionalServer(t)
	repo.InjectError = errors.New("disk full")

	req := models.CreateEventRequest{
		TrackingNumber: "TRK-007",
		Status:         "CREATED",
		Timestamp:      "2024-06-01T10:00:00Z",
	}

	resp := doPost(t, srv, req)
	assertStatus(t, resp, http.StatusInternalServerError)

	var body models.ErrorResponse
	decodeJSON(t, resp, &body)

	if body.Error != "internal server error" {
		t.Errorf("expected generic error message, got: %q", body.Error)
	}
	if body.CorrelationID == "" {
		t.Error("error response missing correlation_id")
	}
}

func TestFunctional_PostEvent_PoolExhausted_Returns503(t *testing.T) {
	srv, repo := newFunctionalServer(t)
	repo.InjectError = repository.ErrPoolExhausted

	req := models.CreateEventRequest{
		TrackingNumber: "TRK-008",
		Status:         "CREATED",
		Timestamp:      "2024-06-01T10:00:00Z",
	}

	resp := doPost(t, srv, req)
	assertStatus(t, resp, http.StatusServiceUnavailable)
}

// ---------------------------------------------------------------------------
// GET /tracking/{tracking_number} — happy path
// ---------------------------------------------------------------------------

func TestFunctional_GetHistory_SingleEvent_Returns200(t *testing.T) {
	srv, _ := newFunctionalServer(t)

	// POST one event first.
	postResp := doPost(t, srv, models.CreateEventRequest{
		TrackingNumber: "TRK-100",
		Status:         "CREATED",
		Timestamp:      "2024-06-01T08:00:00Z",
		Location:       "Origin Hub",
	})
	assertStatus(t, postResp, http.StatusCreated)
	postResp.Body.Close()

	// GET history.
	resp := doGet(t, srv, "/tracking/TRK-100")
	assertStatus(t, resp, http.StatusOK)
	assertContentType(t, resp)
	assertCorrelationID(t, resp)

	var body models.TrackingHistoryResponse
	decodeJSON(t, resp, &body)

	if body.TrackingNumber != "TRK-100" {
		t.Errorf("tracking_number: want TRK-100, got %q", body.TrackingNumber)
	}
	if body.CurrentStatus != models.StatusCreated {
		t.Errorf("current_status: want CREATED, got %q", body.CurrentStatus)
	}
	if len(body.History) != 1 {
		t.Fatalf("history length: want 1, got %d", len(body.History))
	}
	if body.History[0].Status != models.StatusCreated {
		t.Errorf("history[0].status: want CREATED, got %q", body.History[0].Status)
	}
	if body.EstimatedDelivery != nil {
		t.Errorf("estimated_delivery: want null, got %v", body.EstimatedDelivery)
	}
}

func TestFunctional_GetHistory_MultipleEvents_SortedAscending(t *testing.T) {
	srv, _ := newFunctionalServer(t)

	events := []struct {
		status    string
		timestamp string
	}{
		{"IN_TRANSIT", "2024-06-01T12:00:00Z"},
		{"CREATED", "2024-06-01T08:00:00Z"},      // posted out of order
		{"ARRIVED_AT_HUB", "2024-06-01T10:00:00Z"},
	}

	for _, e := range events {
		r := doPost(t, srv, models.CreateEventRequest{
			TrackingNumber: "TRK-101",
			Status:         e.status,
			Timestamp:      e.timestamp,
		})
		assertStatus(t, r, http.StatusCreated)
		r.Body.Close()
	}

	resp := doGet(t, srv, "/tracking/TRK-101")
	assertStatus(t, resp, http.StatusOK)

	var body models.TrackingHistoryResponse
	decodeJSON(t, resp, &body)

	if len(body.History) != 3 {
		t.Fatalf("history length: want 3, got %d", len(body.History))
	}

	// History must be sorted ascending by timestamp.
	for i := 1; i < len(body.History); i++ {
		if body.History[i].Timestamp.Before(body.History[i-1].Timestamp) {
			t.Errorf("history not sorted ascending at index %d", i)
		}
	}

	// current_status must reflect the latest event (IN_TRANSIT at 12:00).
	if body.CurrentStatus != models.StatusInTransit {
		t.Errorf("current_status: want IN_TRANSIT, got %q", body.CurrentStatus)
	}
}

func TestFunctional_GetHistory_EstimatedDeliveryPresent(t *testing.T) {
	srv, repo := newFunctionalServer(t)

	// Insert an event so the tracking number exists.
	r := doPost(t, srv, models.CreateEventRequest{
		TrackingNumber: "TRK-102",
		Status:         "OUT_FOR_DELIVERY",
		Timestamp:      "2024-06-05T09:00:00Z",
	})
	assertStatus(t, r, http.StatusCreated)
	r.Body.Close()

	// Inject an estimated_delivery directly into the mock summary.
	delivery := time.Date(2024, 6, 6, 18, 0, 0, 0, time.UTC)
	repo.SetSummaryForTest(models.TrackingSummary{
		TrackingNumber:    "TRK-102",
		CurrentStatus:     models.StatusOutForDelivery,
		EstimatedDelivery: &delivery,
		UpdatedAt:         time.Date(2024, 6, 5, 9, 0, 0, 0, time.UTC),
	})

	resp := doGet(t, srv, "/tracking/TRK-102")
	assertStatus(t, resp, http.StatusOK)

	var body models.TrackingHistoryResponse
	decodeJSON(t, resp, &body)

	if body.EstimatedDelivery == nil {
		t.Fatal("estimated_delivery: want non-null, got null")
	}
	if !body.EstimatedDelivery.Equal(delivery) {
		t.Errorf("estimated_delivery: want %v, got %v", delivery, body.EstimatedDelivery)
	}
}

// ---------------------------------------------------------------------------
// GET /tracking/{tracking_number} — error paths
// ---------------------------------------------------------------------------

func TestFunctional_GetHistory_UnknownTrackingNumber_Returns404(t *testing.T) {
	srv, _ := newFunctionalServer(t)

	resp := doGet(t, srv, "/tracking/DOES-NOT-EXIST")
	assertStatus(t, resp, http.StatusNotFound)
	assertContentType(t, resp)

	var body models.ErrorResponse
	decodeJSON(t, resp, &body)

	if !strings.Contains(body.Error, "DOES-NOT-EXIST") {
		t.Errorf("error message should contain the tracking number, got: %q", body.Error)
	}
	if body.CorrelationID == "" {
		t.Error("error response missing correlation_id")
	}
}

// ---------------------------------------------------------------------------
// GET /tracking/bulk
// ---------------------------------------------------------------------------

func TestFunctional_GetBulk_ReturnsOnlyExistingNumbers(t *testing.T) {
	srv, _ := newFunctionalServer(t)

	// POST events for two tracking numbers.
	for _, tn := range []string{"BULK-A", "BULK-B"} {
		r := doPost(t, srv, models.CreateEventRequest{
			TrackingNumber: tn,
			Status:         "CREATED",
			Timestamp:      "2024-06-01T10:00:00Z",
		})
		assertStatus(t, r, http.StatusCreated)
		r.Body.Close()
	}

	// Query for two existing + one non-existing.
	resp := doGet(t, srv, "/tracking/bulk?numbers=BULK-A,BULK-B,BULK-MISSING")
	assertStatus(t, resp, http.StatusOK)
	assertContentType(t, resp)

	var summaries []models.TrackingSummary
	decodeJSON(t, resp, &summaries)

	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}
	found := map[string]bool{}
	for _, s := range summaries {
		found[s.TrackingNumber] = true
	}
	if !found["BULK-A"] || !found["BULK-B"] {
		t.Errorf("expected BULK-A and BULK-B in response, got: %v", found)
	}
	if found["BULK-MISSING"] {
		t.Error("BULK-MISSING should not appear in response")
	}
}

func TestFunctional_GetBulk_MissingNumbersParam_Returns400(t *testing.T) {
	srv, _ := newFunctionalServer(t)

	resp := doGet(t, srv, "/tracking/bulk")
	assertStatus(t, resp, http.StatusBadRequest)
	assertContentType(t, resp)

	var body models.ErrorResponse
	decodeJSON(t, resp, &body)

	if !strings.Contains(body.Error, "numbers") {
		t.Errorf("error message should mention 'numbers', got: %q", body.Error)
	}
}

func TestFunctional_GetBulk_TooManyNumbers_Returns422(t *testing.T) {
	srv, _ := newFunctionalServer(t)

	// Build a comma-separated list of 101 tracking numbers.
	nums := make([]string, 101)
	for i := range nums {
		nums[i] = "TRK-OVER"
	}
	path := "/tracking/bulk?numbers=" + strings.Join(nums, ",")

	resp := doGet(t, srv, path)
	assertStatus(t, resp, http.StatusUnprocessableEntity)

	var body models.ErrorResponse
	decodeJSON(t, resp, &body)

	if !strings.Contains(body.Error, "100") {
		t.Errorf("error message should mention the limit of 100, got: %q", body.Error)
	}
}

func TestFunctional_GetBulk_NoneExist_ReturnsEmptyArray(t *testing.T) {
	srv, _ := newFunctionalServer(t)

	resp := doGet(t, srv, "/tracking/bulk?numbers=GHOST-1,GHOST-2")
	assertStatus(t, resp, http.StatusOK)

	var summaries []models.TrackingSummary
	decodeJSON(t, resp, &summaries)

	if summaries == nil {
		t.Error("expected empty array, got null")
	}
	if len(summaries) != 0 {
		t.Errorf("expected 0 summaries, got %d", len(summaries))
	}
}

// ---------------------------------------------------------------------------
// GET /health
// ---------------------------------------------------------------------------

func TestFunctional_Health_HealthyDB_Returns200(t *testing.T) {
	srv, _ := newFunctionalServer(t)

	resp := doGet(t, srv, "/health")
	assertStatus(t, resp, http.StatusOK)
	assertContentType(t, resp)
	assertCorrelationID(t, resp)

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != `{"status":"ok"}` {
		t.Errorf("expected {\"status\":\"ok\"}, got %s", body)
	}
}

func TestFunctional_Health_UnhealthyDB_Returns503(t *testing.T) {
	srv, repo := newFunctionalServer(t)
	repo.InjectError = errors.New("connection refused")

	resp := doGet(t, srv, "/health")
	assertStatus(t, resp, http.StatusServiceUnavailable)
	assertContentType(t, resp)

	var body models.ErrorResponse
	decodeJSON(t, resp, &body)

	if body.Error != "database unavailable" {
		t.Errorf("expected 'database unavailable', got %q", body.Error)
	}
}

// ---------------------------------------------------------------------------
// Middleware — correlation ID propagation
// ---------------------------------------------------------------------------

func TestFunctional_CorrelationID_GeneratedWhenAbsent(t *testing.T) {
	srv, _ := newFunctionalServer(t)

	resp := doGet(t, srv, "/health")
	resp.Body.Close()

	id := resp.Header.Get("X-Correlation-ID")
	if id == "" {
		t.Error("X-Correlation-ID should be generated when not supplied")
	}
}

func TestFunctional_CorrelationID_EchoedWhenSupplied(t *testing.T) {
	srv, _ := newFunctionalServer(t)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/health", nil)
	req.Header.Set("X-Correlation-ID", "my-trace-id-123")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	resp.Body.Close()

	if got := resp.Header.Get("X-Correlation-ID"); got != "my-trace-id-123" {
		t.Errorf("expected echoed correlation ID 'my-trace-id-123', got %q", got)
	}
}

func TestFunctional_CorrelationID_PresentInErrorResponse(t *testing.T) {
	srv, _ := newFunctionalServer(t)

	resp := doGet(t, srv, "/tracking/NONEXISTENT")
	assertStatus(t, resp, http.StatusNotFound)

	headerID := resp.Header.Get("X-Correlation-ID")

	var body models.ErrorResponse
	decodeJSON(t, resp, &body)

	if body.CorrelationID == "" {
		t.Error("error response body missing correlation_id")
	}
	if body.CorrelationID != headerID {
		t.Errorf("body correlation_id %q does not match header %q", body.CorrelationID, headerID)
	}
}

// ---------------------------------------------------------------------------
// Middleware — panic recovery
// ---------------------------------------------------------------------------

func TestFunctional_PanicRecovery_Returns500(t *testing.T) {
	// Wire a handler that panics into the full middleware stack.
	r := chi.NewRouter()
	r.Use(middleware.CorrelationIDMiddleware)
	r.Use(middleware.NewLoggingMiddleware(io.Discard))
	r.Use(middleware.RecoveryMiddleware)
	r.Get("/panic", func(w http.ResponseWriter, _ *http.Request) {
		panic("something went very wrong")
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/panic")
	if err != nil {
		t.Fatalf("GET /panic: %v", err)
	}
	assertStatus(t, resp, http.StatusInternalServerError)
	assertContentType(t, resp)

	var body models.ErrorResponse
	decodeJSON(t, resp, &body)

	if body.Error != "internal server error" {
		t.Errorf("expected 'internal server error', got %q", body.Error)
	}
}
