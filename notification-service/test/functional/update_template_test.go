package functional_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestUpdateTemplate_Valid verifies that PUT /notifications/templates/{template_id} with a valid
// payload for an existing template returns HTTP 200 with template_id and status: UPDATED,
// and that the DB record is updated with the new subject and body_template values.
//
// Validates: Requirements 6.1, 6.2
func TestUpdateTemplate_Valid(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	// Create a template first.
	createPayload := map[string]interface{}{
		"event_type":    "OUT_FOR_DELIVERY",
		"channel":       "PUSH",
		"subject":       "Original Subject",
		"body_template": "Original body {{tracking_number}}.",
	}
	createResp := createTemplateHTTP(t, createPayload)
	templateID, ok := createResp["template_id"].(string)
	require.True(t, ok, "template_id should be a string in create response")
	require.NotEmpty(t, templateID, "template_id should not be empty")

	// PUT updated subject and body_template.
	updatePayload := map[string]interface{}{
		"subject":       "Updated Subject",
		"body_template": "Updated body {{tracking_number}} is now here.",
	}
	resp := doRequest(t, http.MethodPut, "/notifications/templates/"+templateID, updatePayload)
	defer resp.Body.Close()

	// Assert HTTP 200.
	require.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 from update template")

	// Decode response body.
	var respBody map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))

	respTemplateID, ok := respBody["template_id"].(string)
	require.True(t, ok, "template_id should be a string in update response")
	assert.Equal(t, templateID, respTemplateID, "template_id in response should match the updated template")

	status, ok := respBody["status"].(string)
	require.True(t, ok, "status should be a string in update response")
	assert.Equal(t, "UPDATED", status, "status should be UPDATED")

	// Query the DB directly to verify the record was updated.
	var dbSubject, dbBodyTemplate string
	row := testDB.QueryRowContext(
		context.Background(),
		`SELECT subject, body_template FROM notification_templates WHERE template_id = $1`,
		templateID,
	)
	err := row.Scan(&dbSubject, &dbBodyTemplate)
	require.NoError(t, err, "DB record should exist for template_id %q", templateID)

	assert.Equal(t, updatePayload["subject"].(string), dbSubject,
		"DB subject should match the updated subject")
	assert.Equal(t, updatePayload["body_template"].(string), dbBodyTemplate,
		"DB body_template should match the updated body_template")
}

// TestUpdateTemplate_NotFound verifies that PUT /notifications/templates/{template_id} for a
// non-existent template_id returns HTTP 404 and no DB record is modified.
//
// Validates: Requirements 6.3
func TestUpdateTemplate_NotFound(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	updatePayload := map[string]interface{}{
		"subject":       "Some Subject",
		"body_template": "Some body template.",
	}

	resp := doRequest(t, http.MethodPut, "/notifications/templates/nonexistent-template-id", updatePayload)
	defer resp.Body.Close()

	// Assert HTTP 404.
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "expected HTTP 404 for non-existent template_id")

	// Verify no DB record was modified (table should be empty).
	var count int
	err := testDB.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM notification_templates").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "no template record should exist after update on non-existent template_id")
}

// TestUpdateTemplate_Persistence verifies Property 6: UpdateTemplate persistence —
// updated fields reflected in subsequent list.
//
// For any existing template and any valid update payload, after a successful PUT, a subsequent
// GET /notifications/templates SHALL return the template with the updated subject and body_template.
//
// Feature: notification-service-functional-tests, Property 6: UpdateTemplate persistence — updated fields reflected in subsequent list
//
// Validates: Requirements 6.2, 6.4
func TestUpdateTemplate_Persistence(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Reset state for each iteration.
		truncateAll(t, testDB)
		resetStubs()

		// Generate a random template and create it.
		createPayload := genTemplatePayload(rt)
		createResp := createTemplateHTTP(t, createPayload)

		templateID, ok := createResp["template_id"].(string)
		require.True(t, ok, "template_id should be a string in create response")
		require.NotEmpty(t, templateID, "template_id should not be empty")

		// Generate random new subject and body_template values.
		newSubject := rapid.StringMatching(`[a-zA-Z0-9 .,!?-]{1,80}`).Draw(rt, "new_subject")
		newBodyTemplate := rapid.StringMatching(`[a-zA-Z0-9 .,!?{}\[\]_-]{1,200}`).Draw(rt, "new_body_template")

		updatePayload := map[string]interface{}{
			"subject":       newSubject,
			"body_template": newBodyTemplate,
		}

		// PUT the updated values.
		putResp := doRequest(t, http.MethodPut, "/notifications/templates/"+templateID, updatePayload)
		defer putResp.Body.Close()
		require.Equal(t, http.StatusOK, putResp.StatusCode, "expected HTTP 200 from update template")

		// GET /notifications/templates and verify the updated values appear.
		getResp := doRequest(t, http.MethodGet, "/notifications/templates", nil)
		defer getResp.Body.Close()
		require.Equal(t, http.StatusOK, getResp.StatusCode, "expected HTTP 200 from list templates")

		var templates []map[string]interface{}
		require.NoError(t, json.NewDecoder(getResp.Body).Decode(&templates))

		// Find the updated template in the list.
		var found bool
		for _, tmpl := range templates {
			id, ok := tmpl["template_id"].(string)
			if !ok || id != templateID {
				continue
			}
			found = true

			assert.Equal(t, newSubject, tmpl["subject"],
				"subject in list response should reflect the updated value for template %q", templateID)
			assert.Equal(t, newBodyTemplate, tmpl["body_template"],
				"body_template in list response should reflect the updated value for template %q", templateID)
			break
		}

		require.True(t, found, "updated template_id %q should appear in GET /notifications/templates response", templateID)
	})
}
