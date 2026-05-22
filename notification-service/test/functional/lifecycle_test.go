package functional_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLifecycle_HappyPathPush verifies the full lifecycle for PUSH:
// CreateTemplate(PUSH) → SendNotification(PUSH) → Firebase stub called once,
// SendGrid=0, WhatsApp=0, DB log status SENT.
//
// Validates: Requirements 9.1
func TestLifecycle_HappyPathPush(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	// Step 1: Create a PUSH template.
	tmplResp := createTemplateHTTP(t, map[string]interface{}{
		"event_type":    "LIFECYCLE_PUSH",
		"channel":       "PUSH",
		"subject":       "Push notification",
		"body_template": "Your order has been shipped!",
	})
	templateID, ok := tmplResp["template_id"].(string)
	require.True(t, ok, "template_id should be a string")
	require.NotEmpty(t, templateID)

	// Step 2: Send a PUSH notification.
	userID := "user-lifecycle-push-001"
	resp := doRequest(t, http.MethodPost, "/notifications/send", map[string]interface{}{
		"user_id":     userID,
		"channel":     "PUSH",
		"template_id": templateID,
		"variables":   map[string]string{},
	})
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var respBody map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	notificationID, ok := respBody["notification_id"].(string)
	require.True(t, ok, "notification_id should be a string")
	require.NotEmpty(t, notificationID)

	// Assert: Firebase stub called once, SendGrid=0, WhatsApp=0.
	assert.Len(t, firebaseStub.Calls(), 1, "Firebase stub should have received exactly one call")
	assert.Len(t, sendgridStub.Calls(), 0, "SendGrid stub should have received zero calls")
	assert.Len(t, whatsappStub.Calls(), 0, "WhatsApp stub should have received zero calls")

	// Assert: DB log status SENT.
	log := getNotifLogFromDB(t, notificationID)
	assert.Equal(t, "SENT", string(log.Status), "DB log status should be SENT")
	assert.Equal(t, "PUSH", string(log.Channel), "DB log channel should be PUSH")
}

// TestLifecycle_HappyPathEmail verifies the full lifecycle for EMAIL:
// CreateTemplate(EMAIL) → SendNotification(EMAIL) → SendGrid stub called once,
// Firebase=0, WhatsApp=0, DB log status SENT.
//
// Validates: Requirements 9.2
func TestLifecycle_HappyPathEmail(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	// Step 1: Create an EMAIL template.
	tmplResp := createTemplateHTTP(t, map[string]interface{}{
		"event_type":    "LIFECYCLE_EMAIL",
		"channel":       "EMAIL",
		"subject":       "Your order update",
		"body_template": "Your package has been delivered.",
	})
	templateID, ok := tmplResp["template_id"].(string)
	require.True(t, ok, "template_id should be a string")
	require.NotEmpty(t, templateID)

	// Step 2: Send an EMAIL notification.
	userID := "user-lifecycle-email-001"
	resp := doRequest(t, http.MethodPost, "/notifications/send", map[string]interface{}{
		"user_id":     userID,
		"channel":     "EMAIL",
		"template_id": templateID,
		"variables":   map[string]string{},
	})
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var respBody map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	notificationID, ok := respBody["notification_id"].(string)
	require.True(t, ok, "notification_id should be a string")
	require.NotEmpty(t, notificationID)

	// Assert: SendGrid stub called once, Firebase=0, WhatsApp=0.
	assert.Len(t, sendgridStub.Calls(), 1, "SendGrid stub should have received exactly one call")
	assert.Len(t, firebaseStub.Calls(), 0, "Firebase stub should have received zero calls")
	assert.Len(t, whatsappStub.Calls(), 0, "WhatsApp stub should have received zero calls")

	// Assert: DB log status SENT.
	log := getNotifLogFromDB(t, notificationID)
	assert.Equal(t, "SENT", string(log.Status), "DB log status should be SENT")
	assert.Equal(t, "EMAIL", string(log.Channel), "DB log channel should be EMAIL")
}

// TestLifecycle_HappyPathWhatsApp verifies the full lifecycle for WHATSAPP:
// CreateTemplate(WHATSAPP) → SendNotification(WHATSAPP) → WhatsApp stub called once,
// Firebase=0, SendGrid=0, DB log status SENT.
//
// Validates: Requirements 9.3
func TestLifecycle_HappyPathWhatsApp(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	// Step 1: Create a WHATSAPP template.
	tmplResp := createTemplateHTTP(t, map[string]interface{}{
		"event_type":    "LIFECYCLE_WHATSAPP",
		"channel":       "WHATSAPP",
		"subject":       "Delivery update",
		"body_template": "Your order is out for delivery!",
	})
	templateID, ok := tmplResp["template_id"].(string)
	require.True(t, ok, "template_id should be a string")
	require.NotEmpty(t, templateID)

	// Step 2: Send a WHATSAPP notification.
	userID := "user-lifecycle-wa-001"
	resp := doRequest(t, http.MethodPost, "/notifications/send", map[string]interface{}{
		"user_id":     userID,
		"channel":     "WHATSAPP",
		"template_id": templateID,
		"variables":   map[string]string{},
	})
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var respBody map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	notificationID, ok := respBody["notification_id"].(string)
	require.True(t, ok, "notification_id should be a string")
	require.NotEmpty(t, notificationID)

	// Assert: WhatsApp stub called once, Firebase=0, SendGrid=0.
	assert.Len(t, whatsappStub.Calls(), 1, "WhatsApp stub should have received exactly one call")
	assert.Len(t, firebaseStub.Calls(), 0, "Firebase stub should have received zero calls")
	assert.Len(t, sendgridStub.Calls(), 0, "SendGrid stub should have received zero calls")

	// Assert: DB log status SENT.
	log := getNotifLogFromDB(t, notificationID)
	assert.Equal(t, "SENT", string(log.Status), "DB log status should be SENT")
	assert.Equal(t, "WHATSAPP", string(log.Channel), "DB log channel should be WHATSAPP")
}

// TestLifecycle_ProviderFailure verifies the provider failure path:
// CreateTemplate(PUSH) → firebaseStub.SetStatus(500) → SendNotification(PUSH)
// → assert HTTP 503 and DB log status FAILED.
//
// Validates: Requirements 9.4
func TestLifecycle_ProviderFailure(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	// Step 1: Create a PUSH template.
	tmplResp := createTemplateHTTP(t, map[string]interface{}{
		"event_type":    "LIFECYCLE_FAILURE",
		"channel":       "PUSH",
		"subject":       "Delivery failed",
		"body_template": "We could not deliver your order.",
	})
	templateID, ok := tmplResp["template_id"].(string)
	require.True(t, ok, "template_id should be a string")
	require.NotEmpty(t, templateID)

	// Step 2: Configure Firebase stub to return 500.
	firebaseStub.SetStatus(500)

	// Step 3: Send a PUSH notification.
	resp := doRequest(t, http.MethodPost, "/notifications/send", map[string]interface{}{
		"user_id":     "user-lifecycle-fail-001",
		"channel":     "PUSH",
		"template_id": templateID,
		"variables":   map[string]string{},
	})
	defer resp.Body.Close()

	// Assert: HTTP 503.
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	// Assert: DB log status FAILED.
	// The service inserts a log with status FAILED before returning the provider error.
	var notifID string
	err := testDB.QueryRowContext(context.Background(),
		"SELECT notif_id FROM notification_logs LIMIT 1").Scan(&notifID)
	require.NoError(t, err, "expected a notification log to be inserted on provider failure")

	log := getNotifLogFromDB(t, notifID)
	assert.Equal(t, "FAILED", string(log.Status), "DB log status should be FAILED")
}

// TestLifecycle_KafkaToDB verifies the Kafka-to-DB path:
// CreateTemplate(event_type) → kafkaHandler.HandleTrackingStatusUpdated(payload with event_type=template_id)
// → assert Firebase stub called once and DB log status SENT.
//
// Validates: Requirements 9.5
func TestLifecycle_KafkaToDB(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	// Step 1: Create a PUSH template with a specific event_type.
	eventType := "LIFECYCLE_KAFKA"
	tmplResp := createTemplateHTTP(t, map[string]interface{}{
		"event_type":    eventType,
		"channel":       "PUSH",
		"subject":       "Tracking update",
		"body_template": "Your order {{tracking_number}} has been updated.",
	})
	templateID, ok := tmplResp["template_id"].(string)
	require.True(t, ok, "template_id should be a string")
	require.NotEmpty(t, templateID)

	// Step 2: Call kafkaHandler.HandleTrackingStatusUpdated with event_type = template_id.
	// The Kafka handler uses event_type as the template_id lookup key.
	userID := "user-lifecycle-kafka-001"
	trackingNumber := "TRK-LIFECYCLE-001"
	event := map[string]interface{}{
		"user_id":         userID,
		"tracking_number": trackingNumber,
		"event_type":      templateID, // handler uses event_type as template_id
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	handlerErr := kafkaHandler.HandleTrackingStatusUpdated(payload)
	require.NoError(t, handlerErr, "HandleTrackingStatusUpdated should not return an error")

	// Assert: Firebase stub called once.
	fbCalls := firebaseStub.Calls()
	require.Len(t, fbCalls, 1, "Firebase stub should have received exactly one call")
	assert.Equal(t, userID, fbCalls[0].UserID, "Firebase stub call user_id should match event user_id")

	// Assert: DB log status SENT.
	require.Equal(t, 1, countNotifLogs(t), "expected exactly one notification log")
	var notifID string
	err = testDB.QueryRowContext(context.Background(),
		"SELECT notif_id FROM notification_logs LIMIT 1").Scan(&notifID)
	require.NoError(t, err)
	log := getNotifLogFromDB(t, notifID)
	assert.Equal(t, "SENT", string(log.Status), "DB log status should be SENT")
	assert.Equal(t, "PUSH", string(log.Channel), "DB log channel should be PUSH")
}

// TestLifecycle_UpdateThenSend verifies the update-then-send path:
// CreateTemplate → PUT /notifications/templates/{id} with new body_template
// → SendNotification → assert the message in the DB log uses the updated body_template.
//
// Validates: Requirements 9.6
func TestLifecycle_UpdateThenSend(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	// Step 1: Create a PUSH template with an initial body_template.
	tmplResp := createTemplateHTTP(t, map[string]interface{}{
		"event_type":    "LIFECYCLE_UPDATE",
		"channel":       "PUSH",
		"subject":       "Order update",
		"body_template": "Original message body.",
	})
	templateID, ok := tmplResp["template_id"].(string)
	require.True(t, ok, "template_id should be a string")
	require.NotEmpty(t, templateID)

	// Step 2: Update the template with a new body_template.
	updatedBodyTemplate := "Updated message body for lifecycle test."
	putResp := doRequest(t, http.MethodPut, "/notifications/templates/"+templateID, map[string]interface{}{
		"subject":       "Order update",
		"body_template": updatedBodyTemplate,
	})
	defer putResp.Body.Close()
	require.Equal(t, http.StatusOK, putResp.StatusCode, "expected HTTP 200 from update template")

	// Step 3: Send a notification using the same template.
	userID := "user-lifecycle-update-001"
	resp := doRequest(t, http.MethodPost, "/notifications/send", map[string]interface{}{
		"user_id":     userID,
		"channel":     "PUSH",
		"template_id": templateID,
		"variables":   map[string]string{},
	})
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var respBody map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	notificationID, ok := respBody["notification_id"].(string)
	require.True(t, ok, "notification_id should be a string")
	require.NotEmpty(t, notificationID)

	// Assert: the message in the DB log uses the updated body_template.
	log := getNotifLogFromDB(t, notificationID)
	assert.Equal(t, updatedBodyTemplate, log.Message,
		"DB log message should use the updated body_template")
	assert.Equal(t, "SENT", string(log.Status), "DB log status should be SENT")
}
