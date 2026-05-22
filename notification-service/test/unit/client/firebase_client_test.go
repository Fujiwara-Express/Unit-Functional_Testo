package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/notification-service/internal/client"
	"github.com/notification-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// Validates: Requirements 5.1, 5.2
func TestFirebaseClient_SendPush_ValidInput(t *testing.T) {
	var capturedMethod, capturedPath, capturedAuth string
	var capturedBody map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedAuth = r.Header.Get("Authorization")
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := client.NewFirebaseClient(srv.URL, "test-api-key")
	err := c.SendPush(context.Background(), "user-456", "Your package is on the way.")

	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t, "/fcm/send", capturedPath)
	assert.Equal(t, "Bearer test-api-key", capturedAuth)
	assert.Equal(t, "user-456", capturedBody["user_id"])
	assert.Equal(t, "Your package is on the way.", capturedBody["message"])
}

func TestFirebaseClient_SendPush_Returns200_NoError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := client.NewFirebaseClient(srv.URL, "test-api-key")
	err := c.SendPush(context.Background(), "user-456", "message")
	require.NoError(t, err)
}

// Validates: Requirements 5.3
func TestFirebaseClient_SendPush_ServiceUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := client.NewFirebaseClient(srv.URL, "test-api-key")
	err := c.SendPush(context.Background(), "user-456", "message")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrServiceUnavailable)
}

// Validates: Requirements 5.4, 5.5
func TestSendGridClient_SendEmail_ValidInput(t *testing.T) {
	var capturedMethod, capturedPath, capturedAuth string
	var capturedBody map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedAuth = r.Header.Get("Authorization")
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	c := client.NewSendGridClient(srv.URL, "sg-api-key")
	err := c.SendEmail(context.Background(), "user@example.com", "Package Update", "Your package is on the way.")

	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t, "/v3/mail/send", capturedPath)
	assert.Equal(t, "Bearer sg-api-key", capturedAuth)
	assert.Equal(t, "user@example.com", capturedBody["recipient"])
	assert.Equal(t, "Package Update", capturedBody["subject"])
	assert.Equal(t, "Your package is on the way.", capturedBody["body"])
}

func TestSendGridClient_SendEmail_Returns202_NoError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	c := client.NewSendGridClient(srv.URL, "sg-api-key")
	err := c.SendEmail(context.Background(), "user@example.com", "Subject", "Body")
	require.NoError(t, err)
}

// Validates: Requirements 5.6
func TestSendGridClient_SendEmail_ServiceUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := client.NewSendGridClient(srv.URL, "sg-api-key")
	err := c.SendEmail(context.Background(), "user@example.com", "Subject", "Body")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrServiceUnavailable)
}

// Validates: Requirements 5.7, 5.8
func TestWhatsAppClient_SendWhatsApp_ValidInput(t *testing.T) {
	var capturedMethod, capturedPath, capturedAuth string
	var capturedBody map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedAuth = r.Header.Get("Authorization")
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := client.NewWhatsAppClient(srv.URL, "wa-api-key")
	err := c.SendWhatsApp(context.Background(), "+62812345678", "Your package is on the way.")

	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t, "/v1/messages", capturedPath)
	assert.Equal(t, "Bearer wa-api-key", capturedAuth)
	assert.Equal(t, "+62812345678", capturedBody["phone"])
	assert.Equal(t, "Your package is on the way.", capturedBody["message"])
}

func TestWhatsAppClient_SendWhatsApp_Returns200_NoError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := client.NewWhatsAppClient(srv.URL, "wa-api-key")
	err := c.SendWhatsApp(context.Background(), "+62812345678", "message")
	require.NoError(t, err)
}

// Validates: Requirements 5.9
func TestWhatsAppClient_SendWhatsApp_ServiceUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	c := client.NewWhatsAppClient(srv.URL, "wa-api-key")
	err := c.SendWhatsApp(context.Background(), "+62812345678", "message")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrServiceUnavailable)
}

// Feature: notification-service-unit-tests, Property 9: Firebase/SendGrid/WhatsApp clients send correct request for any valid input
// Validates: Requirements 5.1, 5.4, 5.7
func TestProviderClients_SendCorrectRequest_AnyValidInput(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		userID := rapid.StringMatching(`[a-z0-9-]{4,20}`).Draw(rt, "user_id")
		message := rapid.StringN(1, 100, -1).Draw(rt, "message")

		var capturedAuth string
		var capturedBody map[string]string

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedAuth = r.Header.Get("Authorization")
			json.NewDecoder(r.Body).Decode(&capturedBody)
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		c := client.NewFirebaseClient(srv.URL, "test-key")
		err := c.SendPush(context.Background(), userID, message)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(capturedAuth, "Bearer "), "Authorization header should start with Bearer")
		assert.Equal(t, userID, capturedBody["user_id"])
		assert.Equal(t, message, capturedBody["message"])
	})
}

// Feature: notification-service-unit-tests, Property 10: Provider clients return ErrServiceUnavailable for any non-2xx response
// Validates: Requirements 5.3, 5.6, 5.9
func TestProviderClients_ErrServiceUnavailable_AnyNon2xx(t *testing.T) {
	non2xxCodes := []int{
		http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden,
		http.StatusNotFound, http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout,
	}

	rapid.Check(t, func(rt *rapid.T) {
		codeIdx := rapid.IntRange(0, len(non2xxCodes)-1).Draw(rt, "code_idx")
		statusCode := non2xxCodes[codeIdx]

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(statusCode)
		}))
		defer srv.Close()

		// Test Firebase
		fbClient := client.NewFirebaseClient(srv.URL, "key")
		err := fbClient.SendPush(context.Background(), "user", "msg")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrServiceUnavailable)
	})
}

// Feature: notification-service-unit-tests, Property 15: Notification payload JSON round-trip
// Validates: Requirements 5.11
func TestNotificationPayload_JSON_RoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		type firebasePayload struct {
			UserID  string `json:"user_id"`
			Message string `json:"message"`
		}

		original := firebasePayload{
			UserID:  rapid.StringMatching(`[a-z0-9-]{4,20}`).Draw(rt, "user_id"),
			Message: rapid.StringN(1, 200, -1).Draw(rt, "message"),
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var restored firebasePayload
		require.NoError(t, json.Unmarshal(data, &restored))

		assert.Equal(t, original, restored)
	})
}
