package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"tracking-service/internal/models"
)

// testHandler returns a minimal HTTP handler for middleware testing:
//   - GET requests → 200 {"status":"ok"}
//   - POST requests → 400 ErrorResponse (to exercise error-response correlation ID)
func testHandler(correlationID func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
			return
		}
		// POST → 400 with ErrorResponse so we can verify correlation_id in body
		w.WriteHeader(http.StatusBadRequest)
		resp := models.ErrorResponse{
			Error:         "bad request",
			CorrelationID: correlationID(),
		}
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	})
}

// Feature: tracking-service, Property 13: Every request receives a unique correlation ID
// Validates: Requirements 8.2
func TestProperty13_UniqueCorrelationID(t *testing.T) {
	// Collect all correlation IDs across rapid iterations to assert global uniqueness.
	seenIDs := make(map[string]struct{})

	rapid.Check(t, func(t *rapid.T) {
		// Build the handler chain: CorrelationIDMiddleware wraps the test handler.
		// The test handler needs access to the correlation ID from context for the
		// error-response body, so we wire it through the middleware context.
		var capturedID string
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedID = GetCorrelationID(r.Context())
			w.Header().Set("Content-Type", "application/json")

			method := rapid.StringMatching(`GET|POST`).Draw(t, "method")
			if method == "GET" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
				return
			}
			// POST → 400 with ErrorResponse containing correlation_id
			w.WriteHeader(http.StatusBadRequest)
			resp := models.ErrorResponse{
				Error:         "bad request",
				CorrelationID: capturedID,
			}
			json.NewEncoder(w).Encode(resp) //nolint:errcheck
		})

		handler := CorrelationIDMiddleware(inner)
		server := httptest.NewServer(handler)
		defer server.Close()

		// Optionally supply an existing X-Correlation-ID header (50% of the time).
		suppliedID := ""
		if rapid.Bool().Draw(t, "supply_header") {
			suppliedID = rapid.StringMatching(`[a-zA-Z0-9\-]{8,36}`).Draw(t, "supplied_id")
		}

		// Pick a method to exercise both success and error paths.
		method := rapid.StringMatching(`GET|POST`).Draw(t, "req_method")

		req, err := http.NewRequest(method, server.URL+"/test", nil)
		if err != nil {
			t.Fatal(err)
		}
		if suppliedID != "" {
			req.Header.Set("X-Correlation-ID", suppliedID)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		// 1. Response header must carry a non-empty X-Correlation-ID.
		responseID := resp.Header.Get("X-Correlation-ID")
		if responseID == "" {
			t.Fatal("X-Correlation-ID response header is empty")
		}

		// 2. If we supplied a header, the same ID must be echoed back.
		if suppliedID != "" && responseID != suppliedID {
			t.Fatalf("expected echoed correlation ID %q, got %q", suppliedID, responseID)
		}

		// 3. For error responses (4xx), the body must contain correlation_id.
		if resp.StatusCode >= 400 {
			var errResp models.ErrorResponse
			if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
				t.Fatalf("failed to decode error response body: %v", err)
			}
			if errResp.CorrelationID == "" {
				t.Fatal("error response body missing correlation_id field")
			}
			if errResp.CorrelationID != responseID {
				t.Fatalf("body correlation_id %q does not match header %q",
					errResp.CorrelationID, responseID)
			}
		}

		// 4. Assert global uniqueness across all rapid iterations (only for
		//    server-generated IDs; supplied IDs are caller-controlled).
		if suppliedID == "" {
			if _, exists := seenIDs[responseID]; exists {
				t.Fatalf("duplicate correlation ID generated: %q", responseID)
			}
			seenIDs[responseID] = struct{}{}
		}
	})
}

// Feature: tracking-service, Property 14: Structured log entry emitted for every request
// Validates: Requirements 8.3
func TestProperty14_StructuredLogEntry(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Fresh buffer per iteration so we can count log lines precisely.
		var buf bytes.Buffer

		// Build the handler chain: CorrelationIDMiddleware → LoggingMiddleware → handler.
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			method := r.Method
			if method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
				return
			}
			// POST → 400 with ErrorResponse
			correlationID := GetCorrelationID(r.Context())
			w.WriteHeader(http.StatusBadRequest)
			resp := models.ErrorResponse{
				Error:         "bad request",
				CorrelationID: correlationID,
			}
			json.NewEncoder(w).Encode(resp) //nolint:errcheck
		})

		loggingMiddleware := NewLoggingMiddleware(&buf)
		handler := CorrelationIDMiddleware(loggingMiddleware(inner))
		server := httptest.NewServer(handler)
		defer server.Close()

		// Generate a random method and path.
		method := rapid.SampledFrom([]string{http.MethodGet, http.MethodPost}).Draw(t, "method")
		path := "/" + rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "path_segment")

		req, err := http.NewRequest(method, server.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()

		// Retrieve the correlation ID from the response header.
		correlationID := resp.Header.Get("X-Correlation-ID")
		if correlationID == "" {
			t.Fatal("X-Correlation-ID response header is empty")
		}

		// 1. Exactly one log line must have been written.
		logOutput := buf.String()
		lines := strings.Split(strings.TrimRight(logOutput, "\n"), "\n")
		if len(lines) != 1 || lines[0] == "" {
			t.Fatalf("expected exactly 1 log line, got %d: %q", len(lines), logOutput)
		}
		logLine := lines[0]

		// 2. Log line must contain the HTTP method.
		if !strings.Contains(logLine, "method="+method) {
			t.Fatalf("log line missing method=%s: %q", method, logLine)
		}

		// 3. Log line must contain the path.
		if !strings.Contains(logLine, "path="+path) {
			t.Fatalf("log line missing path=%s: %q", path, logLine)
		}

		// 4. Log line must contain a status code field.
		if !strings.Contains(logLine, "status=") {
			t.Fatalf("log line missing status field: %q", logLine)
		}

		// 5. Log line must contain a latency_ms field.
		if !strings.Contains(logLine, "latency_ms=") {
			t.Fatalf("log line missing latency_ms field: %q", logLine)
		}

		// 6. Log line must contain the correlation ID.
		if !strings.Contains(logLine, "correlation_id="+correlationID) {
			t.Fatalf("log line missing correlation_id=%s: %q", correlationID, logLine)
		}
	})
}
