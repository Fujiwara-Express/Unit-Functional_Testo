package functional_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestListTemplates_Empty verifies that GET /notifications/templates returns HTTP 200
// with an empty JSON array when no templates exist in the DB.
//
// Validates: Requirements 4.1
func TestListTemplates_Empty(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	resp := doRequest(t, http.MethodGet, "/notifications/templates", nil)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 for empty template list")

	var result []map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Empty(t, result, "expected empty JSON array when no templates exist")
}

// TestListTemplates_ReturnsAll verifies that GET /notifications/templates returns HTTP 200
// with all inserted templates and their correct fields.
//
// Validates: Requirements 4.2
func TestListTemplates_ReturnsAll(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	// Insert multiple templates.
	payloads := []map[string]interface{}{
		{
			"event_type":    "OUT_FOR_DELIVERY",
			"channel":       "PUSH",
			"subject":       "Package Update",
			"body_template": "Your package is on the way.",
		},
		{
			"event_type":    "DELIVERED",
			"channel":       "EMAIL",
			"subject":       "Delivery Confirmation",
			"body_template": "Your package has been delivered.",
		},
		{
			"event_type":    "PICKUP_READY",
			"channel":       "WHATSAPP",
			"subject":       "Ready for Pickup",
			"body_template": "Your package is ready for pickup.",
		},
	}

	insertedIDs := make(map[string]bool)
	for _, payload := range payloads {
		tmplResp := createTemplateHTTP(t, payload)
		templateID, ok := tmplResp["template_id"].(string)
		require.True(t, ok, "template_id should be a string in create response")
		require.NotEmpty(t, templateID, "template_id should not be empty")
		insertedIDs[templateID] = true
	}

	// GET /notifications/templates.
	resp := doRequest(t, http.MethodGet, "/notifications/templates", nil)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 for list templates")

	var result []map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

	// Assert all inserted templates are returned.
	require.Len(t, result, len(payloads), "expected exactly %d templates in response", len(payloads))

	returnedIDs := make(map[string]bool)
	for _, tmpl := range result {
		templateID, ok := tmpl["template_id"].(string)
		require.True(t, ok, "template_id should be a string in list response")
		returnedIDs[templateID] = true

		// Assert all required fields are present and non-empty.
		assert.NotEmpty(t, tmpl["template_id"], "template_id should not be empty")
		assert.NotEmpty(t, tmpl["event_type"], "event_type should not be empty")
		assert.NotEmpty(t, tmpl["channel"], "channel should not be empty")
		assert.NotEmpty(t, tmpl["body_template"], "body_template should not be empty")
	}

	// Assert every inserted template_id appears in the response.
	for id := range insertedIDs {
		assert.True(t, returnedIDs[id], "inserted template_id %q should appear in list response", id)
	}
}

// TestListTemplates_Completeness verifies Property 4: ListTemplates completeness —
// response contains exactly the templates in the DB.
//
// For any set of templates inserted into the DB, GET /notifications/templates SHALL return
// HTTP 200 with a JSON array containing exactly those templates — no more, no fewer —
// with all fields correctly populated.
//
// Feature: notification-service-functional-tests, Property 4: ListTemplates completeness — response contains exactly the templates in the DB
//
// Validates: Requirements 4.2, 4.3
func TestListTemplates_Completeness(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Reset state for each iteration.
		truncateAll(t, testDB)
		resetStubs()

		// Generate a random number of templates (1–5).
		n := rapid.IntRange(1, 5).Draw(rt, "n")

		// Insert n templates and collect their returned template_ids and payloads.
		type insertedTemplate struct {
			templateID   string
			eventType    string
			channel      string
			subject      string
			bodyTemplate string
		}
		inserted := make([]insertedTemplate, 0, n)

		for i := 0; i < n; i++ {
			payload := genTemplatePayload(rt)
			tmplResp := createTemplateHTTP(t, payload)

			templateID, ok := tmplResp["template_id"].(string)
			require.True(t, ok, "template_id should be a string in create response")
			require.NotEmpty(t, templateID, "template_id should not be empty")

			inserted = append(inserted, insertedTemplate{
				templateID:   templateID,
				eventType:    payload["event_type"].(string),
				channel:      payload["channel"].(string),
				subject:      payload["subject"].(string),
				bodyTemplate: payload["body_template"].(string),
			})
		}

		// GET /notifications/templates.
		resp := doRequest(t, http.MethodGet, "/notifications/templates", nil)
		defer resp.Body.Close()

		// Assert HTTP 200.
		require.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 from list templates")

		// Decode the response body as a JSON array.
		var result []map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

		// Assert the response contains exactly the same number of templates as inserted.
		require.Len(t, result, n,
			"list response should contain exactly %d templates (the number inserted)", n)

		// Build a map from template_id to response object for O(1) lookup.
		responseByID := make(map[string]map[string]interface{}, len(result))
		for _, tmpl := range result {
			id, ok := tmpl["template_id"].(string)
			require.True(t, ok, "each template in list response should have a string template_id")
			responseByID[id] = tmpl
		}

		// Assert each inserted template_id appears in the response with all fields correctly populated.
		for _, ins := range inserted {
			tmpl, found := responseByID[ins.templateID]
			require.True(t, found,
				"inserted template_id %q should appear in list response", ins.templateID)

			assert.Equal(t, ins.templateID, tmpl["template_id"],
				"template_id in response should match inserted template_id")
			assert.Equal(t, ins.eventType, tmpl["event_type"],
				"event_type in response should match inserted event_type for template %q", ins.templateID)
			assert.Equal(t, ins.channel, tmpl["channel"],
				"channel in response should match inserted channel for template %q", ins.templateID)
			assert.Equal(t, ins.subject, tmpl["subject"],
				"subject in response should match inserted subject for template %q", ins.templateID)
			assert.Equal(t, ins.bodyTemplate, tmpl["body_template"],
				"body_template in response should match inserted body_template for template %q", ins.templateID)
		}
	})
}
