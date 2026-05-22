package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/notification-service/internal/handler/http/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Validates: Requirements 7.1
func TestAuthMiddleware_ValidToken(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Auth(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.True(t, nextCalled, "next handler should have been called")
	assert.Equal(t, http.StatusOK, w.Code)
}

// Validates: Requirements 7.2
func TestAuthMiddleware_InvalidToken(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	handler := middleware.Auth(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "InvalidTokenFormat")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["code"])
	assert.NotEmpty(t, resp["message"])
	assert.Contains(t, resp, "request_id")
	assert.NotEmpty(t, resp["timestamp"])
}

// Validates: Requirements 7.2
func TestAuthMiddleware_MissingToken(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	handler := middleware.Auth(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["code"])
	assert.NotEmpty(t, resp["message"])
	assert.Contains(t, resp, "request_id")
	assert.NotEmpty(t, resp["timestamp"])
}

// Validates: Requirements 7.4
func TestRequestIDMiddleware_NoHeader(t *testing.T) {
	var capturedID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = middleware.GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.RequestID(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.NotEmpty(t, capturedID, "request_id should be injected into context when header is absent")
}

// Validates: Requirements 7.5
func TestRequestIDMiddleware_WithHeader(t *testing.T) {
	var capturedID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = middleware.GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.RequestID(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "my-request-id")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, "my-request-id", capturedID, "request_id in context should match the X-Request-ID header")
}

// Validates: Requirements 7.6
func TestErrorMiddleware_InternalError(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went terribly wrong internally")
	})

	handler := middleware.ErrorHandler(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["code"])
	assert.NotEmpty(t, resp["message"])
	assert.Contains(t, resp, "request_id")
	assert.NotEmpty(t, resp["timestamp"])

	body := w.Body.String()
	assert.NotContains(t, body, "something went terribly wrong internally",
		"internal error details should not be exposed in the response body")
}

// Feature: notification-service-unit-tests, Property 13: Request ID middleware injects non-empty request_id for any request
// Validates: Requirements 7.3, 7.4, 7.5
func TestRequestIDMiddleware_AlwaysInjectsNonEmptyID(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		var capturedID string
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedID = middleware.GetRequestID(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		handler := middleware.RequestID(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)

		includeHeader := rapid.Bool().Draw(rt, "include_header")
		if includeHeader {
			headerValue := rapid.StringN(1, 64, -1).Draw(rt, "header_value")
			req.Header.Set("X-Request-ID", headerValue)
		}

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.NotEmpty(t, capturedID, "request_id in context should always be non-empty")
	})
}
