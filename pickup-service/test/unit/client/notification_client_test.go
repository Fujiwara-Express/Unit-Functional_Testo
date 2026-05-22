package client_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pickup-service/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNotificationClient_NotifyCourierEnRoute_ValidPayload verifies that the
// notification client sends a request containing the sender's contact details
// (contact_name, contact_phone, courier_id).
func TestNotificationClient_NotifyCourierEnRoute_ValidPayload(t *testing.T) {
	var capturedBody map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	nc := client.NewHTTPNotificationClient(srv.URL)
	err := nc.NotifyCourierEnRoute(context.Background(), "John Doe", "+62812345678", "courier-001")

	require.NoError(t, err)
	assert.Equal(t, "John Doe", capturedBody["contact_name"])
	assert.Equal(t, "+62812345678", capturedBody["contact_phone"])
	assert.Equal(t, "courier-001", capturedBody["courier_id"])
}

// TestNotificationClient_NotifyCourierEnRoute_ErrorStatus verifies that a
// non-200/204 response returns an error.
func TestNotificationClient_NotifyCourierEnRoute_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	nc := client.NewHTTPNotificationClient(srv.URL)
	err := nc.NotifyCourierEnRoute(context.Background(), "John", "+628123", "courier-1")

	require.Error(t, err)
}

// TestNotificationClient_NotifyCourierEnRoute_NoContent verifies that a 204
// response is treated as success.
func TestNotificationClient_NotifyCourierEnRoute_NoContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	nc := client.NewHTTPNotificationClient(srv.URL)
	err := nc.NotifyCourierEnRoute(context.Background(), "John", "+628123", "courier-1")

	require.NoError(t, err)
}
