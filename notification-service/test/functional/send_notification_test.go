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

// TestSendNotification_RoundTrip verifies Property 1: SendNotification round-trip —
// stub called once, DB log persisted with correct fields.
//
// Feature: notification-service-functional-tests, Property 1: SendNotification round-trip — stub called once, DB log persisted with correct fields
//
// Validates: Requirements 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.12
func TestSendNotification_RoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Reset state for each iteration.
		truncateAll(t, testDB)
		resetStubs()

		// Generate a random channel.
		channels := []string{"PUSH", "EMAIL", "WHATSAPP"}
		channel := rapid.SampledFrom(channels).Draw(rt, "channel")

		// Create a template for the chosen channel.
		templatePayload := genTemplatePayload(rt)
		templatePayload["channel"] = channel
		tmplResp := createTemplateHTTP(t, templatePayload)
		templateID, ok := tmplResp["template_id"].(string)
		require.True(t, ok, "template_id should be a string in create response")
		require.NotEmpty(t, templateID, "template_id should not be empty")

		// Generate a random send payload for this template and channel.
		sendPayload := genSendPayload(rt, templateID, channel)
		userID, _ := sendPayload["user_id"].(string)

		// Send the notification.
		resp := doRequest(t, http.MethodPost, "/notifications/send", sendPayload)
		defer resp.Body.Close()

		// Assert HTTP 200.
		require.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 from send notification")

		// Decode the response body.
		var respBody map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))

		notificationID, ok := respBody["notification_id"].(string)
		require.True(t, ok, "notification_id should be a string in response")
		require.NotEmpty(t, notificationID, "notification_id should not be empty")

		// Assert response fields.
		assert.Equal(t, "SENT", respBody["status"], "response status should be SENT")
		assert.Equal(t, channel, respBody["channel"], "response channel should match request channel")

		// Query the DB log.
		log := getNotifLogFromDB(t, notificationID)

		// Assert notification_id round-trip: response ID matches DB record.
		assert.Equal(t, notificationID, log.NotifID, "notification_id in response should match notif_id in DB")

		// Assert DB record fields.
		assert.Equal(t, userID, log.UserID, "DB log user_id should match request user_id")
		assert.Equal(t, channel, string(log.Channel), "DB log channel should match request channel")
		assert.Equal(t, templateID, log.TemplateID, "DB log template_id should match request template_id")
		assert.Equal(t, "SENT", string(log.Status), "DB log status should be SENT")
		assert.NotEmpty(t, log.Message, "DB log message should not be empty")

		// Assert the correct provider stub received exactly one call,
		// and the other two received zero calls.
		switch channel {
		case "PUSH":
			fbCalls := firebaseStub.Calls()
			require.Len(t, fbCalls, 1, "Firebase stub should have received exactly one call for PUSH")
			assert.Equal(t, userID, fbCalls[0].UserID, "Firebase stub call user_id should match request user_id")
			assert.Equal(t, log.Message, fbCalls[0].Message, "Firebase stub call message should match DB log message")
			assert.Len(t, sendgridStub.Calls(), 0, "SendGrid stub should have received zero calls for PUSH")
			assert.Len(t, whatsappStub.Calls(), 0, "WhatsApp stub should have received zero calls for PUSH")

		case "EMAIL":
			sgCalls := sendgridStub.Calls()
			require.Len(t, sgCalls, 1, "SendGrid stub should have received exactly one call for EMAIL")
			assert.Equal(t, userID, sgCalls[0].Recipient, "SendGrid stub call recipient should match request user_id")
			assert.Equal(t, log.Message, sgCalls[0].Body, "SendGrid stub call body should match DB log message")
			assert.Len(t, firebaseStub.Calls(), 0, "Firebase stub should have received zero calls for EMAIL")
			assert.Len(t, whatsappStub.Calls(), 0, "WhatsApp stub should have received zero calls for EMAIL")

		case "WHATSAPP":
			waCalls := whatsappStub.Calls()
			require.Len(t, waCalls, 1, "WhatsApp stub should have received exactly one call for WHATSAPP")
			assert.Equal(t, userID, waCalls[0].Phone, "WhatsApp stub call phone should match request user_id")
			assert.Equal(t, log.Message, waCalls[0].Message, "WhatsApp stub call message should match DB log message")
			assert.Len(t, firebaseStub.Calls(), 0, "Firebase stub should have received zero calls for WHATSAPP")
			assert.Len(t, sendgridStub.Calls(), 0, "SendGrid stub should have received zero calls for WHATSAPP")
		}
	})
}

// TestSendNotification_ExclusiveRouting verifies Property 2: Exclusive provider routing —
// exactly one stub called per notification.
//
// Feature: notification-service-functional-tests, Property 2: Exclusive provider routing — exactly one stub called per notification
//
// Validates: Requirements 8.1, 8.2, 8.3, 8.4
func TestSendNotification_ExclusiveRouting(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Reset state for each iteration.
		truncateAll(t, testDB)
		resetStubs()

		// Generate a random channel.
		channels := []string{"PUSH", "EMAIL", "WHATSAPP"}
		channel := rapid.SampledFrom(channels).Draw(rt, "channel")

		// Create a template for the chosen channel.
		templatePayload := genTemplatePayload(rt)
		templatePayload["channel"] = channel
		tmplResp := createTemplateHTTP(t, templatePayload)
		templateID, ok := tmplResp["template_id"].(string)
		require.True(t, ok, "template_id should be a string in create response")
		require.NotEmpty(t, templateID, "template_id should not be empty")

		// Generate a random send payload for this template and channel.
		sendPayload := genSendPayload(rt, templateID, channel)

		// Send the notification.
		resp := doRequest(t, http.MethodPost, "/notifications/send", sendPayload)
		defer resp.Body.Close()

		// Assert HTTP 200.
		require.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 from send notification")

		// Assert exclusive routing: exactly one stub received a call, the other two received zero.
		fbCalls := len(firebaseStub.Calls())
		sgCalls := len(sendgridStub.Calls())
		waCalls := len(whatsappStub.Calls())

		switch channel {
		case "PUSH":
			// Firebase must receive exactly one call; SendGrid and WhatsApp must receive zero.
			assert.Equal(t, 1, fbCalls, "PUSH: Firebase stub should receive exactly one call")
			assert.Equal(t, 0, sgCalls, "PUSH: SendGrid stub should receive zero calls")
			assert.Equal(t, 0, waCalls, "PUSH: WhatsApp stub should receive zero calls")

		case "EMAIL":
			// SendGrid must receive exactly one call; Firebase and WhatsApp must receive zero.
			assert.Equal(t, 0, fbCalls, "EMAIL: Firebase stub should receive zero calls")
			assert.Equal(t, 1, sgCalls, "EMAIL: SendGrid stub should receive exactly one call")
			assert.Equal(t, 0, waCalls, "EMAIL: WhatsApp stub should receive zero calls")

		case "WHATSAPP":
			// WhatsApp must receive exactly one call; Firebase and SendGrid must receive zero.
			assert.Equal(t, 0, fbCalls, "WHATSAPP: Firebase stub should receive zero calls")
			assert.Equal(t, 0, sgCalls, "WHATSAPP: SendGrid stub should receive zero calls")
			assert.Equal(t, 1, waCalls, "WHATSAPP: WhatsApp stub should receive exactly one call")
		}
	})
}

// TestSendNotification_ValidPush inserts a PUSH template, sends a notification with channel PUSH,
// and asserts HTTP 200, correct response fields, Firebase stub called once, and DB log correct.
//
// Validates: Requirements 2.1, 2.2, 2.3, 2.4, 2.5
func TestSendNotification_ValidPush(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	// Insert a PUSH template.
	tmplResp := createTemplateHTTP(t, map[string]interface{}{
		"event_type":    "ORDER_SHIPPED",
		"channel":       "PUSH",
		"subject":       "Your order shipped",
		"body_template": "Hello, your order is on the way!",
	})
	templateID, ok := tmplResp["template_id"].(string)
	require.True(t, ok, "template_id should be a string")
	require.NotEmpty(t, templateID)

	// Send notification.
	userID := "user-push-001"
	resp := doRequest(t, http.MethodPost, "/notifications/send", map[string]interface{}{
		"user_id":     userID,
		"channel":     "PUSH",
		"template_id": templateID,
		"variables":   map[string]string{},
	})
	defer resp.Body.Close()

	// Assert HTTP 200.
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Decode response.
	var respBody map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))

	notificationID, ok := respBody["notification_id"].(string)
	require.True(t, ok, "notification_id should be a string")
	require.NotEmpty(t, notificationID)
	assert.Equal(t, "SENT", respBody["status"])
	assert.Equal(t, "PUSH", respBody["channel"])

	// Assert Firebase stub received exactly one call.
	fbCalls := firebaseStub.Calls()
	require.Len(t, fbCalls, 1, "Firebase stub should have received exactly one call")
	assert.Equal(t, userID, fbCalls[0].UserID)
	assert.Equal(t, "Hello, your order is on the way!", fbCalls[0].Message)

	// Assert SendGrid and WhatsApp stubs received zero calls.
	assert.Len(t, sendgridStub.Calls(), 0)
	assert.Len(t, whatsappStub.Calls(), 0)

	// Assert DB log.
	log := getNotifLogFromDB(t, notificationID)
	assert.Equal(t, "SENT", string(log.Status))
	assert.Equal(t, userID, log.UserID)
	assert.Equal(t, templateID, log.TemplateID)
	assert.Equal(t, "PUSH", string(log.Channel))
	assert.Equal(t, "Hello, your order is on the way!", log.Message)
}

// TestSendNotification_ValidEmail inserts an EMAIL template, sends a notification with channel EMAIL,
// and asserts HTTP 200, SendGrid stub called once with correct fields, and DB log correct.
//
// Validates: Requirements 2.1, 2.2, 2.3, 2.6
func TestSendNotification_ValidEmail(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	// Insert an EMAIL template.
	subject := "Your shipment update"
	tmplResp := createTemplateHTTP(t, map[string]interface{}{
		"event_type":    "ORDER_DELIVERED",
		"channel":       "EMAIL",
		"subject":       subject,
		"body_template": "Your package has been delivered.",
	})
	templateID, ok := tmplResp["template_id"].(string)
	require.True(t, ok, "template_id should be a string")
	require.NotEmpty(t, templateID)

	// Send notification.
	userID := "user-email-001"
	resp := doRequest(t, http.MethodPost, "/notifications/send", map[string]interface{}{
		"user_id":     userID,
		"channel":     "EMAIL",
		"template_id": templateID,
		"variables":   map[string]string{},
	})
	defer resp.Body.Close()

	// Assert HTTP 200.
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Decode response.
	var respBody map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))

	notificationID, ok := respBody["notification_id"].(string)
	require.True(t, ok, "notification_id should be a string")
	require.NotEmpty(t, notificationID)

	// Assert SendGrid stub received exactly one call with correct fields.
	// recipient = user_id (SendEmail passes user_id as recipient)
	sgCalls := sendgridStub.Calls()
	require.Len(t, sgCalls, 1, "SendGrid stub should have received exactly one call")
	assert.Equal(t, userID, sgCalls[0].Recipient)
	assert.Equal(t, subject, sgCalls[0].Subject)
	assert.Equal(t, "Your package has been delivered.", sgCalls[0].Body)

	// Assert Firebase and WhatsApp stubs received zero calls.
	assert.Len(t, firebaseStub.Calls(), 0)
	assert.Len(t, whatsappStub.Calls(), 0)

	// Assert DB log.
	log := getNotifLogFromDB(t, notificationID)
	assert.Equal(t, "SENT", string(log.Status))
	assert.Equal(t, "EMAIL", string(log.Channel))
}

// TestSendNotification_ValidWhatsApp inserts a WHATSAPP template, sends a notification with channel WHATSAPP,
// and asserts HTTP 200, WhatsApp stub called once with correct fields, and DB log correct.
//
// Validates: Requirements 2.1, 2.2, 2.3, 2.7
func TestSendNotification_ValidWhatsApp(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	// Insert a WHATSAPP template.
	tmplResp := createTemplateHTTP(t, map[string]interface{}{
		"event_type":    "ORDER_OUT_FOR_DELIVERY",
		"channel":       "WHATSAPP",
		"subject":       "Delivery update",
		"body_template": "Your order is out for delivery!",
	})
	templateID, ok := tmplResp["template_id"].(string)
	require.True(t, ok, "template_id should be a string")
	require.NotEmpty(t, templateID)

	// Send notification.
	userID := "user-wa-001"
	resp := doRequest(t, http.MethodPost, "/notifications/send", map[string]interface{}{
		"user_id":     userID,
		"channel":     "WHATSAPP",
		"template_id": templateID,
		"variables":   map[string]string{},
	})
	defer resp.Body.Close()

	// Assert HTTP 200.
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Decode response.
	var respBody map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))

	notificationID, ok := respBody["notification_id"].(string)
	require.True(t, ok, "notification_id should be a string")
	require.NotEmpty(t, notificationID)

	// Assert WhatsApp stub received exactly one call with correct fields.
	// phone = user_id (SendWhatsApp passes user_id as phone)
	waCalls := whatsappStub.Calls()
	require.Len(t, waCalls, 1, "WhatsApp stub should have received exactly one call")
	assert.Equal(t, userID, waCalls[0].Phone)
	assert.Equal(t, "Your order is out for delivery!", waCalls[0].Message)

	// Assert Firebase and SendGrid stubs received zero calls.
	assert.Len(t, firebaseStub.Calls(), 0)
	assert.Len(t, sendgridStub.Calls(), 0)

	// Assert DB log.
	log := getNotifLogFromDB(t, notificationID)
	assert.Equal(t, "SENT", string(log.Status))
	assert.Equal(t, "WHATSAPP", string(log.Channel))
}

// TestSendNotification_TemplateNotFound sends a notification with a non-existent template_id
// and asserts HTTP 404 and no DB log inserted.
//
// Validates: Requirements 2.8
func TestSendNotification_TemplateNotFound(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	resp := doRequest(t, http.MethodPost, "/notifications/send", map[string]interface{}{
		"user_id":     "user-notfound-001",
		"channel":     "PUSH",
		"template_id": "nonexistent-template-id",
		"variables":   map[string]string{},
	})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, 0, countNotifLogs(t))
}

// TestSendNotification_MissingFields is a table-driven test that omits each required field
// and asserts HTTP 400 and no DB log inserted.
//
// Validates: Requirements 2.9, 2.10
func TestSendNotification_MissingFields(t *testing.T) {
	cases := []struct {
		name    string
		payload map[string]interface{}
	}{
		{
			name: "missing user_id",
			payload: map[string]interface{}{
				"channel":     "PUSH",
				"template_id": "some-template-id",
				"variables":   map[string]string{},
			},
		},
		{
			name: "missing channel",
			payload: map[string]interface{}{
				"user_id":     "user-missing-001",
				"template_id": "some-template-id",
				"variables":   map[string]string{},
			},
		},
		{
			name: "missing template_id",
			payload: map[string]interface{}{
				"user_id":   "user-missing-001",
				"channel":   "PUSH",
				"variables": map[string]string{},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			truncateAll(t, testDB)
			resetStubs()

			resp := doRequest(t, http.MethodPost, "/notifications/send", tc.payload)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "expected HTTP 400 for %s", tc.name)
			assert.Equal(t, 0, countNotifLogs(t), "expected no DB log for %s", tc.name)
		})
	}
}

// TestSendNotification_Unauthorized sends a notification without a Bearer token
// and asserts HTTP 401.
//
// Validates: Requirements 2.11
func TestSendNotification_Unauthorized(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	payload := map[string]interface{}{
		"user_id":     "user-unauth-001",
		"channel":     "PUSH",
		"template_id": "some-template-id",
		"variables":   map[string]string{},
	}
	b, err := json.Marshal(payload)
	require.NoError(t, err)

	// Send request without Authorization header.
	req, err := http.NewRequest(http.MethodPost, testServer.URL+"/notifications/send", bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	// Intentionally omit Authorization header.

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestSendNotification_ProviderFailure inserts a PUSH template, configures the Firebase stub to
// return 500, sends a notification, and asserts HTTP 503 and DB log has status FAILED.
//
// Validates: Requirements 2.4, 2.5
func TestSendNotification_ProviderFailure(t *testing.T) {
	truncateAll(t, testDB)
	resetStubs()

	// Insert a PUSH template.
	tmplResp := createTemplateHTTP(t, map[string]interface{}{
		"event_type":    "ORDER_FAILED",
		"channel":       "PUSH",
		"subject":       "Delivery failed",
		"body_template": "We could not deliver your order.",
	})
	templateID, ok := tmplResp["template_id"].(string)
	require.True(t, ok, "template_id should be a string")
	require.NotEmpty(t, templateID)

	// Configure Firebase stub to return 500.
	firebaseStub.SetStatus(500)

	// Send notification.
	resp := doRequest(t, http.MethodPost, "/notifications/send", map[string]interface{}{
		"user_id":     "user-fail-001",
		"channel":     "PUSH",
		"template_id": templateID,
		"variables":   map[string]string{},
	})
	defer resp.Body.Close()

	// Assert HTTP 503.
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	// Assert DB log has status FAILED.
	// The service saves the log before returning the provider error,
	// so we need to find the log by querying all logs.
	var notifID string
	err := testDB.QueryRowContext(context.Background(), "SELECT notif_id FROM notification_logs LIMIT 1").Scan(&notifID)
	require.NoError(t, err, "expected a notification log to be inserted on provider failure")

	log := getNotifLogFromDB(t, notifID)
	assert.Equal(t, "FAILED", string(log.Status))
}
