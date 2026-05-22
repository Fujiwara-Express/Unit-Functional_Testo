package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"pgregory.net/rapid"

	"tracking-service/internal/middleware"
	"tracking-service/internal/models"
	"tracking-service/internal/repository"
)

// newBulkTestServer creates an httptest.Server with a chi router that serves
// all three tracking routes:
//
//	POST /tracking/events
//	GET  /tracking/bulk
//	GET  /tracking/{tracking_number}
//
// The route order matters: /tracking/bulk must be registered before
// /tracking/{tracking_number} so that chi matches the literal path first.
func newBulkTestServer(repo *repository.MockRepository) *httptest.Server {
	r := chi.NewRouter()
	r.Use(middleware.CorrelationIDMiddleware)
	r.Post("/tracking/events", NewEventHandler(repo, 5*time.Second).ServeHTTP)
	r.Get("/tracking/bulk", NewBulkHandler(repo, 5*time.Second).ServeHTTP)
	r.Get("/tracking/{tracking_number}", NewHistoryHandler(repo, 5*time.Second).ServeHTTP)
	return httptest.NewServer(r)
}

// getBulk performs GET /tracking/bulk?numbers=... and returns the raw HTTP
// response and the decoded []TrackingSummary on HTTP 200.
func getBulk(serverURL string, numbers []string) (*http.Response, []models.TrackingSummary, error) {
	query := url.QueryEscape(strings.Join(numbers, ","))
	resp, err := http.Get(serverURL + "/tracking/bulk?numbers=" + query)
	if err != nil {
		return nil, nil, fmt.Errorf("GET bulk: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return resp, nil, nil
	}
	var summaries []models.TrackingSummary
	if err := json.NewDecoder(resp.Body).Decode(&summaries); err != nil {
		resp.Body.Close()
		return resp, nil, fmt.Errorf("decode bulk response: %w", err)
	}
	return resp, summaries, nil
}

// Feature: tracking-service, Property 8: Bulk query returns exactly the existing tracking numbers
// Validates: Requirements 3.1, 3.5
func TestProperty8_BulkReturnsOnlyExistingNumbers(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := repository.NewMockRepository()
		server := newBulkTestServer(repo)
		defer server.Close()

		// Draw 1–10 "existing" tracking numbers and POST events for each.
		existingCount := rapid.IntRange(1, 10).Draw(t, "existing_count")
		existingNumbers := make([]string, existingCount)
		existingSet := make(map[string]bool, existingCount)

		for i := 0; i < existingCount; i++ {
			// Generate a unique tracking number for each existing entry.
			var tn string
			for {
				tn = rapid.StringMatching(`[A-Za-z0-9]{4,20}`).Draw(t, fmt.Sprintf("existing_%d", i))
				if !existingSet[tn] {
					break
				}
			}
			existingNumbers[i] = tn
			existingSet[tn] = true

			// POST an event so the summary is created.
			req := models.CreateEventRequest{
				TrackingNumber: tn,
				Status:         rapid.SampledFrom(validStatusValues).Draw(t, fmt.Sprintf("status_%d", i)),
				Timestamp:      time.Date(2024, 1, 1, i, 0, 0, 0, time.UTC).Format(time.RFC3339),
			}
			resp, err := postEvent(server.URL, req)
			if err != nil {
				t.Fatalf("postEvent[%d] failed: %v", i, err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("postEvent[%d]: expected HTTP 201, got %d", i, resp.StatusCode)
			}
		}

		// Draw 0–10 "non-existing" tracking numbers (never POST events for these).
		nonExistingCount := rapid.IntRange(0, 10).Draw(t, "non_existing_count")
		nonExistingNumbers := make([]string, 0, nonExistingCount)
		nonExistingSet := make(map[string]bool, nonExistingCount)

		for i := 0; i < nonExistingCount; i++ {
			var tn string
			for {
				tn = rapid.StringMatching(`[A-Za-z0-9]{4,20}`).Draw(t, fmt.Sprintf("non_existing_%d", i))
				// Must not collide with existing numbers or previously drawn non-existing ones.
				if !existingSet[tn] && !nonExistingSet[tn] {
					break
				}
			}
			nonExistingNumbers = append(nonExistingNumbers, tn)
			nonExistingSet[tn] = true
		}

		// Combine all numbers into the bulk query (order is arbitrary).
		allNumbers := append(existingNumbers, nonExistingNumbers...)

		// Enforce the 100-number limit imposed by the validator.
		if len(allNumbers) > 100 {
			allNumbers = allNumbers[:100]
		}

		resp, summaries, err := getBulk(server.URL, allNumbers)
		if err != nil {
			t.Fatalf("getBulk failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
		}
		if summaries == nil {
			t.Fatal("summaries response is nil")
		}

		// Build a set of returned tracking numbers.
		returnedSet := make(map[string]bool, len(summaries))
		for _, s := range summaries {
			if returnedSet[s.TrackingNumber] {
				t.Fatalf("duplicate tracking number %q in bulk response", s.TrackingNumber)
			}
			returnedSet[s.TrackingNumber] = true
		}

		// Assert: every existing number appears exactly once.
		for _, tn := range existingNumbers {
			if !returnedSet[tn] {
				t.Fatalf("existing tracking number %q missing from bulk response", tn)
			}
		}

		// Assert: no non-existing number appears in the response.
		for _, tn := range nonExistingNumbers {
			if returnedSet[tn] {
				t.Fatalf("non-existing tracking number %q unexpectedly present in bulk response", tn)
			}
		}

		// Assert: response length equals the number of existing numbers that were
		// included in the query (after the 100-cap, some existing ones may have
		// been dropped — but we only query what's in allNumbers).
		queriedExistingCount := 0
		queriedSet := make(map[string]bool, len(allNumbers))
		for _, tn := range allNumbers {
			queriedSet[tn] = true
		}
		for _, tn := range existingNumbers {
			if queriedSet[tn] {
				queriedExistingCount++
			}
		}
		if len(summaries) != queriedExistingCount {
			t.Fatalf("expected %d summaries (existing numbers in query), got %d",
				queriedExistingCount, len(summaries))
		}
	})
}
