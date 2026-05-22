package functional_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/notification-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestTemplateRendering_AllPlaceholdersSubstituted verifies that all {{var}} placeholders
// in a template body are replaced with the corresponding variable values.
//
// Validates: Requirements 3.1, 3.2
func TestTemplateRendering_AllPlaceholdersSubstituted(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	// Create a PUSH template with a {{tracking_number}} placeholder.
	tmplResp := createTemplateHTTP(t, map[string]interface{}{
		"event_type":    "OUT_FOR_DELIVERY",
		"channel":       "PUSH",
		"subject":       "Package Update",
		"body_template": "Your package {{tracking_number}} is on the way.",
	})
	templateID, ok := tmplResp["template_id"].(string)
	require.True(t, ok, "template_id should be a string")
	require.NotEmpty(t, templateID)

	// Send a notification with the variable substitution.
	sendPayload := map[string]interface{}{
		"user_id":     "user-abc",
		"channel":     "PUSH",
		"template_id": templateID,
		"variables":   map[string]string{"tracking_number": "TRK-999"},
	}
	resp := doRequest(t, http.MethodPost, "/notifications/send", sendPayload)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var respBody map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	notificationID, _ := respBody["notification_id"].(string)
	require.NotEmpty(t, notificationID)

	// Assert the message in the DB log has the placeholder substituted.
	log := getNotifLogFromDB(t, notificationID)
	assert.Contains(t, log.Message, "TRK-999", "rendered message should contain the substituted value")
	assert.NotContains(t, log.Message, "{{", "rendered message should not contain unresolved {{ markers")
	assert.NotContains(t, log.Message, "}}", "rendered message should not contain unresolved }} markers")
}

// TestTemplateRendering_UnmatchedPlaceholderPreserved verifies that placeholders with no
// matching variable are left unchanged in the rendered message.
//
// Validates: Requirements 3.2
func TestTemplateRendering_UnmatchedPlaceholderPreserved(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	// Create a PUSH template with an {{unmatched}} placeholder.
	tmplResp := createTemplateHTTP(t, map[string]interface{}{
		"event_type":    "UNMATCHED_EVENT",
		"channel":       "PUSH",
		"subject":       "Test",
		"body_template": "Hello {{unmatched}} world",
	})
	templateID, ok := tmplResp["template_id"].(string)
	require.True(t, ok, "template_id should be a string")
	require.NotEmpty(t, templateID)

	// Send a notification with an empty variables map.
	sendPayload := map[string]interface{}{
		"user_id":     "user-xyz",
		"channel":     "PUSH",
		"template_id": templateID,
		"variables":   map[string]string{},
	}
	resp := doRequest(t, http.MethodPost, "/notifications/send", sendPayload)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var respBody map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	notificationID, _ := respBody["notification_id"].(string)
	require.NotEmpty(t, notificationID)

	// Assert the unmatched placeholder is preserved in the DB log message.
	log := getNotifLogFromDB(t, notificationID)
	assert.Contains(t, log.Message, "{{unmatched}}", "unmatched placeholder should be preserved in rendered message")
}

// TestTemplateRendering_RenderCorrectness verifies Property 3: Template rendering correctness —
// rendered message equals RenderTemplate output.
//
// For any template body with {{var}} placeholders and any variables map, the message stored
// in the DB SHALL equal domain.RenderTemplate(body_template, variables).
//
// Feature: notification-service-functional-tests, Property 3: Template rendering correctness — rendered message equals RenderTemplate output
//
// Validates: Requirements 3.1, 3.2, 3.3
func TestTemplateRendering_RenderCorrectness(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Reset state for each iteration.
		truncateAll(t, testDB)
		resetStubs()

		// Generate a template body with {{var}} placeholders and a matching variables map.
		bodyTemplate, variables := genTemplatePayloadWithPlaceholders(rt)

		// Create a PUSH template with the generated body_template.
		tmplResp := createTemplateHTTP(t, map[string]interface{}{
			"event_type":    "RENDER_TEST_EVENT",
			"channel":       "PUSH",
			"subject":       "Render Test",
			"body_template": bodyTemplate,
		})
		templateID, ok := tmplResp["template_id"].(string)
		require.True(t, ok, "template_id should be a string in create response")
		require.NotEmpty(t, templateID, "template_id should not be empty")

		// Build the send payload, overriding variables with the generated ones.
		sendPayload := genSendPayload(rt, templateID, "PUSH")
		sendPayload["variables"] = variables

		// Send the notification.
		resp := doRequest(t, http.MethodPost, "/notifications/send", sendPayload)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 from send notification")

		// Decode the response to get the notification_id.
		var respBody map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
		notificationID, ok := respBody["notification_id"].(string)
		require.True(t, ok, "notification_id should be a string in response")
		require.NotEmpty(t, notificationID, "notification_id should not be empty")

		// Query the DB log.
		log := getNotifLogFromDB(t, notificationID)

		// Assert the stored message equals domain.RenderTemplate(bodyTemplate, variables).
		expected := domain.RenderTemplate(bodyTemplate, variables)
		assert.Equal(t, expected, log.Message,
			"DB log message should equal domain.RenderTemplate(bodyTemplate, variables); "+
				"bodyTemplate=%q variables=%v", bodyTemplate, variables)
	})
}
