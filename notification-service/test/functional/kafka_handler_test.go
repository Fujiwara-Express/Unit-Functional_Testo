package functional_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestHandleTrackingStatusUpdated_Valid inserts a PUSH template whose template_id matches
// the event_type, calls HandleTrackingStatusUpdated with a valid JSON payload, and asserts:
//   - No error returned
//   - Firebase stub received exactly one call
//   - DB log has status SENT, channel PUSH, and correct user_id
//
// Validates: Requirements 7.1, 7.2
func TestHandleTrackingStatusUpdated_Valid(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	// The handler uses event_type as the template_id lookup key (TemplateID: event.EventType).
	// So we insert a template whose template_id equals the event_type we will send.
	eventType := "ORDER_SHIPPED"
	tmplResp := createTemplateHTTP(t, map[string]interface{}{
		"event_type":    eventType,
		"channel":       "PUSH",
		"subject":       "Order shipped",
		"body_template": "Your order {{tracking_number}} has shipped.",
	})
	templateID, ok := tmplResp["template_id"].(string)
	require.True(t, ok, "template_id should be a string")
	require.NotEmpty(t, templateID)

	// Build a valid event payload where event_type matches the template_id.
	userID := "user-kafka-001"
	trackingNumber := "TRK-12345"
	event := map[string]interface{}{
		"user_id":         userID,
		"tracking_number": trackingNumber,
		"event_type":      templateID, // handler uses event_type as template_id
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	// Call the Kafka handler directly.
	handlerErr := kafkaHandler.HandleTrackingStatusUpdated(payload)
	require.NoError(t, handlerErr, "HandleTrackingStatusUpdated should not return an error for a valid payload")

	// Assert Firebase stub received exactly one call.
	fbCalls := firebaseStub.Calls()
	require.Len(t, fbCalls, 1, "Firebase stub should have received exactly one call")
	assert.Equal(t, userID, fbCalls[0].UserID, "Firebase stub call user_id should match event user_id")

	// Assert SendGrid and WhatsApp stubs received zero calls.
	assert.Len(t, sendgridStub.Calls(), 0, "SendGrid stub should have received zero calls")
	assert.Len(t, whatsappStub.Calls(), 0, "WhatsApp stub should have received zero calls")

	// Assert DB log has status SENT, channel PUSH, and correct user_id.
	require.Equal(t, 1, countNotifLogs(t), "expected exactly one notification log")
	var notifID string
	err = testDB.QueryRowContext(t.Context(), "SELECT notif_id FROM notification_logs LIMIT 1").Scan(&notifID)
	require.NoError(t, err)
	log := getNotifLogFromDB(t, notifID)
	assert.Equal(t, "SENT", string(log.Status), "DB log status should be SENT")
	assert.Equal(t, "PUSH", string(log.Channel), "DB log channel should be PUSH")
	assert.Equal(t, userID, log.UserID, "DB log user_id should match event user_id")
}

// TestHandleTrackingStatusUpdated_MalformedJSON calls HandleTrackingStatusUpdated with
// malformed JSON and asserts an error is returned and no notification log is inserted.
//
// Validates: Requirements 7.3
func TestHandleTrackingStatusUpdated_MalformedJSON(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	err := kafkaHandler.HandleTrackingStatusUpdated([]byte("not-json"))
	require.Error(t, err, "HandleTrackingStatusUpdated should return an error for malformed JSON")
	assert.Equal(t, 0, countNotifLogs(t), "no notification log should be inserted for malformed JSON")
}

// TestHandleTrackingStatusUpdated_TemplateNotFound calls HandleTrackingStatusUpdated with a
// valid payload whose event_type does not match any template in the DB, and asserts an error
// is returned and no notification log is inserted.
//
// Validates: Requirements 7.4
func TestHandleTrackingStatusUpdated_TemplateNotFound(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	// Use an event_type that does not match any template in the DB.
	event := map[string]interface{}{
		"user_id":         "user-kafka-notfound",
		"tracking_number": "TRK-99999",
		"event_type":      "NONEXISTENT_EVENT_TYPE",
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	handlerErr := kafkaHandler.HandleTrackingStatusUpdated(payload)
	require.Error(t, handlerErr, "HandleTrackingStatusUpdated should return an error when template is not found")
	assert.Equal(t, 0, countNotifLogs(t), "no notification log should be inserted when template is not found")
}

// TestHandleTrackingStatusUpdated_RoundTrip verifies Property 7: Kafka handler round-trip —
// valid event dispatches to Firebase and persists log.
//
// For any valid TrackingStatusUpdatedEvent payload (any user_id, tracking_number, event_type
// matching an existing template), calling HandleTrackingStatusUpdated SHALL result in exactly
// one Firebase stub call and exactly one notification_logs record with status SENT, channel PUSH,
// and the correct user_id.
//
// Feature: notification-service-functional-tests, Property 7: Kafka handler round-trip — valid event dispatches to Firebase and persists log
//
// Validates: Requirements 7.1, 7.2, 7.5
func TestHandleTrackingStatusUpdated_RoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		truncateAll(t, testDB)
		resetStubs()

		// Generate a random event_type to use as the template_id.
		eventType := rapid.StringMatching(`[A-Z_]{4,20}`).Draw(rt, "event_type")

		// Insert a PUSH template whose template_id will equal the event_type.
		// We use createTemplateHTTP which returns the generated template_id.
		tmplResp := createTemplateHTTP(t, map[string]interface{}{
			"event_type":    eventType,
			"channel":       "PUSH",
			"subject":       "Tracking update",
			"body_template": "Update for {{tracking_number}}",
		})
		templateID, ok := tmplResp["template_id"].(string)
		require.True(t, ok, "template_id should be a string")
		require.NotEmpty(t, templateID)

		// Generate random user_id and tracking_number.
		userID := "user-" + rapid.StringMatching(`[a-z0-9]{4,16}`).Draw(rt, "user_id")
		trackingNumber := "TRK-" + rapid.StringMatching(`[A-Z0-9]{4,12}`).Draw(rt, "tracking_number")

		// Build the event payload using the template_id as event_type (handler uses event_type as template lookup key).
		event := map[string]interface{}{
			"user_id":         userID,
			"tracking_number": trackingNumber,
			"event_type":      templateID,
		}
		payload, err := json.Marshal(event)
		require.NoError(t, err)

		// Call the Kafka handler.
		handlerErr := kafkaHandler.HandleTrackingStatusUpdated(payload)
		require.NoError(t, handlerErr, "HandleTrackingStatusUpdated should not return an error for a valid payload")

		// Assert exactly one Firebase stub call.
		fbCalls := firebaseStub.Calls()
		require.Len(t, fbCalls, 1, "Firebase stub should have received exactly one call")
		assert.Equal(t, userID, fbCalls[0].UserID, "Firebase stub call user_id should match event user_id")

		// Assert exactly one notification_logs record with status SENT, channel PUSH, correct user_id.
		require.Equal(t, 1, countNotifLogs(t), "expected exactly one notification log")
		var notifID string
		scanErr := testDB.QueryRowContext(t.Context(), "SELECT notif_id FROM notification_logs LIMIT 1").Scan(&notifID)
		require.NoError(t, scanErr)
		log := getNotifLogFromDB(t, notifID)
		assert.Equal(t, "SENT", string(log.Status), "DB log status should be SENT")
		assert.Equal(t, "PUSH", string(log.Channel), "DB log channel should be PUSH")
		assert.Equal(t, userID, log.UserID, "DB log user_id should match event user_id")
	})
}

// TestHandleTrackingStatusUpdated_RejectsMalformed verifies Property 8: Kafka handler rejects
// malformed JSON — no log inserted, error returned.
//
// For any byte slice that is not valid JSON, calling HandleTrackingStatusUpdated SHALL return
// a non-nil error and SHALL NOT insert any record into notification_logs.
//
// Feature: notification-service-functional-tests, Property 8: Kafka handler rejects malformed JSON — no log inserted, error returned
//
// Validates: Requirements 7.3
func TestHandleTrackingStatusUpdated_RejectsMalformed(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		truncateAll(t, testDB)
		resetStubs()

		// Generate a byte slice that is guaranteed to not be valid JSON by prepending
		// a non-JSON character (e.g., 'x') to any random string.
		// This ensures json.Unmarshal will always fail.
		suffix := rapid.StringMatching(`[a-zA-Z0-9 .,!?]{0,50}`).Draw(rt, "suffix")
		invalidJSON := []byte("x" + suffix)

		err := kafkaHandler.HandleTrackingStatusUpdated(invalidJSON)
		require.Error(t, err, "HandleTrackingStatusUpdated should return an error for non-JSON input")
		assert.Equal(t, 0, countNotifLogs(t), "no notification log should be inserted for non-JSON input")
	})
}
