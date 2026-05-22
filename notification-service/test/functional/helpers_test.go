package functional_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/notification-service/internal/domain"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// truncateAll clears all tables before each test.
func truncateAll(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		"TRUNCATE TABLE notification_logs, notification_templates")
	require.NoError(t, err)
}

// resetStubs resets all stub call recorders and restores default (200) status codes.
func resetStubs() {
	firebaseStub.Reset()
	sendgridStub.Reset()
	whatsappStub.Reset()
}

// createTemplateHTTP posts to POST /notifications/templates with auth and returns the decoded response body.
// Asserts HTTP 201.
func createTemplateHTTP(t *testing.T, payload map[string]interface{}) map[string]interface{} {
	t.Helper()
	resp := doRequest(t, http.MethodPost, "/notifications/templates", payload)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result
}

// getNotifLogFromDB reads a notification_log record directly from the DB by notif_id.
func getNotifLogFromDB(t *testing.T, notifID string) *domain.NotificationLog {
	t.Helper()
	row := testDB.QueryRowContext(context.Background(),
		`SELECT notif_id, user_id, tracking_number, channel, template_id, message, status, sent_at
         FROM notification_logs WHERE notif_id = $1`, notifID)
	l := &domain.NotificationLog{}
	var channel, status string
	err := row.Scan(&l.NotifID, &l.UserID, &l.TrackingNumber, &channel,
		&l.TemplateID, &l.Message, &status, &l.SentAt)
	require.NoError(t, err)
	l.Channel = domain.Channel(channel)
	l.Status = domain.NotifStatus(status)
	return l
}

// doRequest sends an authenticated HTTP request to the testServer.
func doRequest(t *testing.T, method, path string, body interface{}) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, testServer.URL+path, bodyReader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// countNotifLogs returns the total number of rows in the notification_logs table.
func countNotifLogs(t *testing.T) int {
	t.Helper()
	var n int
	testDB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM notification_logs").Scan(&n) //nolint:errcheck
	return n
}

// genTemplatePayload generates a random valid CreateTemplate request payload.
func genTemplatePayload(rt *rapid.T) map[string]interface{} {
	channels := []string{"PUSH", "EMAIL", "WHATSAPP"}
	return map[string]interface{}{
		"event_type":    rapid.StringMatching(`[A-Z_]{4,20}`).Draw(rt, "event_type"),
		"channel":       rapid.SampledFrom(channels).Draw(rt, "channel"),
		"subject":       rapid.StringMatching(`[a-zA-Z0-9 .,!?-]{1,80}`).Draw(rt, "subject"),
		"body_template": rapid.StringMatching(`[a-zA-Z0-9 .,!?{}\[\]_-]{1,200}`).Draw(rt, "body_template"),
	}
}

// genTemplatePayloadWithPlaceholders generates a template body with {{var}} placeholders
// and a matching variables map. Variable values are constrained to printable ASCII to
// avoid null bytes and other characters that PostgreSQL rejects in text columns.
func genTemplatePayloadWithPlaceholders(rt *rapid.T) (bodyTemplate string, variables map[string]string) {
	keys := rapid.SliceOfN(rapid.StringMatching(`[a-z_]{2,10}`), 1, 4).Draw(rt, "keys")
	body := ""
	vars := map[string]string{}
	for _, k := range keys {
		body += "{{" + k + "}} "
		vars[k] = rapid.StringMatching(`[a-zA-Z0-9 .,!?@#$%^&*()\-_+=]{1,20}`).Draw(rt, "val_"+k)
	}
	return strings.TrimSpace(body), vars
}

// genSendPayload generates a random valid SendNotification request payload for a given templateID and channel.
func genSendPayload(rt *rapid.T, templateID string, channel string) map[string]interface{} {
	return map[string]interface{}{
		"user_id":     "user-" + rapid.StringMatching(`[a-z0-9]{4,16}`).Draw(rt, "user_id"),
		"channel":     channel,
		"template_id": templateID,
		"variables":   map[string]string{},
	}
}
