package functional_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestCreateTemplate_RoundTrip verifies Property 5: CreateTemplate round-trip —
// template_id in response matches DB record.
//
// For any valid CreateTemplate payload, the template_id returned in the HTTP 201 response
// SHALL match the template_id stored in the database, and the DB record SHALL contain all
// provided fields with their correct values.
//
// Feature: notification-service-functional-tests, Property 5: CreateTemplate round-trip — template_id in response matches DB record
//
// Validates: Requirements 5.2, 5.5
func TestCreateTemplate_RoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Reset state for each iteration.
		truncateAll(t, testDB)
		resetStubs()

		// Generate a random valid CreateTemplate payload.
		payload := genTemplatePayload(rt)

		// POST to /notifications/templates and assert HTTP 201.
		resp := doRequest(t, http.MethodPost, "/notifications/templates", payload)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode, "expected HTTP 201 from create template")

		// Decode the response body to get template_id.
		var respBody map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))

		templateID, ok := respBody["template_id"].(string)
		require.True(t, ok, "template_id should be a string in create response")
		require.NotEmpty(t, templateID, "template_id should not be empty in create response")

		// Query the DB directly to fetch the template by template_id.
		var dbTemplateID, dbEventType, dbChannel, dbSubject, dbBodyTemplate string
		row := testDB.QueryRowContext(
			context.Background(),
			`SELECT template_id, event_type, channel, subject, body_template
			 FROM notification_templates WHERE template_id = $1`,
			templateID,
		)
		err := row.Scan(&dbTemplateID, &dbEventType, &dbChannel, &dbSubject, &dbBodyTemplate)
		require.NoError(t, err, "DB record should exist for template_id %q returned in response", templateID)

		// Assert the template_id in the response matches the DB record.
		assert.Equal(t, templateID, dbTemplateID,
			"template_id in response should match template_id in DB")

		// Assert all fields match the original payload.
		assert.Equal(t, payload["event_type"].(string), dbEventType,
			"DB event_type should match the payload event_type")
		assert.Equal(t, payload["channel"].(string), dbChannel,
			"DB channel should match the payload channel")
		assert.Equal(t, payload["subject"].(string), dbSubject,
			"DB subject should match the payload subject")
		assert.Equal(t, payload["body_template"].(string), dbBodyTemplate,
			"DB body_template should match the payload body_template")
	})
}

// TestCreateTemplate_Valid verifies that POST /notifications/templates with a valid payload
// returns HTTP 201 with template_id and status: CREATED, and that the DB record exists
// with all provided fields correctly stored.
//
// Validates: Requirements 5.1, 5.2
func TestCreateTemplate_Valid(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	payload := map[string]interface{}{
		"event_type":    "OUT_FOR_DELIVERY",
		"channel":       "PUSH",
		"subject":       "Package Update",
		"body_template": "Your package {{tracking_number}} is on the way.",
	}

	resp := doRequest(t, http.MethodPost, "/notifications/templates", payload)
	defer resp.Body.Close()

	// Assert HTTP 201.
	require.Equal(t, http.StatusCreated, resp.StatusCode, "expected HTTP 201 from create template")

	// Decode response body.
	var respBody map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))

	templateID, ok := respBody["template_id"].(string)
	require.True(t, ok, "template_id should be a string in create response")
	require.NotEmpty(t, templateID, "template_id should not be empty")

	status, ok := respBody["status"].(string)
	require.True(t, ok, "status should be a string in create response")
	assert.Equal(t, "CREATED", status, "status should be CREATED")

	// Query the DB directly to verify the record was stored with all fields.
	var dbTemplateID, dbEventType, dbChannel, dbSubject, dbBodyTemplate string
	row := testDB.QueryRowContext(
		context.Background(),
		`SELECT template_id, event_type, channel, subject, body_template
		 FROM notification_templates WHERE template_id = $1`,
		templateID,
	)
	err := row.Scan(&dbTemplateID, &dbEventType, &dbChannel, &dbSubject, &dbBodyTemplate)
	require.NoError(t, err, "DB record should exist for template_id %q", templateID)

	assert.Equal(t, templateID, dbTemplateID, "DB template_id should match response template_id")
	assert.Equal(t, payload["event_type"].(string), dbEventType, "DB event_type should match payload")
	assert.Equal(t, payload["channel"].(string), dbChannel, "DB channel should match payload")
	assert.Equal(t, payload["subject"].(string), dbSubject, "DB subject should match payload")
	assert.Equal(t, payload["body_template"].(string), dbBodyTemplate, "DB body_template should match payload")
}

// TestCreateTemplate_MissingFields verifies that POST /notifications/templates with a missing
// required field returns HTTP 400 and no record is inserted into the DB.
//
// Validates: Requirements 5.3
func TestCreateTemplate_MissingFields(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	basePayload := map[string]interface{}{
		"event_type":    "OUT_FOR_DELIVERY",
		"channel":       "PUSH",
		"subject":       "Package Update",
		"body_template": "Your package is on the way.",
	}

	cases := []struct {
		name        string
		omitField   string
	}{
		{name: "missing event_type", omitField: "event_type"},
		{name: "missing channel", omitField: "channel"},
		{name: "missing subject", omitField: "subject"},
		{name: "missing body_template", omitField: "body_template"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			truncateAll(t, testDB)
			resetStubs()

			// Build payload with the specified field omitted.
			payload := make(map[string]interface{})
			for k, v := range basePayload {
				if k != tc.omitField {
					payload[k] = v
				}
			}

			resp := doRequest(t, http.MethodPost, "/notifications/templates", payload)
			defer resp.Body.Close()

			// Assert HTTP 400.
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
				"expected HTTP 400 when %s is missing", tc.omitField)

			// Assert no record was inserted.
			var count int
			err := testDB.QueryRowContext(context.Background(),
				"SELECT COUNT(*) FROM notification_templates").Scan(&count)
			require.NoError(t, err)
			assert.Equal(t, 0, count,
				"no template record should be inserted when %s is missing", tc.omitField)
		})
	}
}

// TestCreateTemplate_Unauthorized verifies that POST /notifications/templates without a
// Bearer token returns HTTP 401.
//
// Validates: Requirements 5.4
func TestCreateTemplate_Unauthorized(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	payload := map[string]interface{}{
		"event_type":    "OUT_FOR_DELIVERY",
		"channel":       "PUSH",
		"subject":       "Package Update",
		"body_template": "Your package is on the way.",
	}

	b, err := json.Marshal(payload)
	require.NoError(t, err)

	// Build request without Authorization header.
	req, err := http.NewRequest(http.MethodPost, testServer.URL+"/notifications/templates", bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	// Intentionally omit Authorization header.

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "expected HTTP 401 when no Bearer token is provided")
}
