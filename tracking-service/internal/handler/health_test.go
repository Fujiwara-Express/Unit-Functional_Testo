package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"tracking-service/internal/models"
	"tracking-service/internal/repository"
)

func TestHealthHandler_HealthyDB(t *testing.T) {
	repo := repository.NewMockRepository()
	handler := NewHealthHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if body != `{"status":"ok"}` {
		t.Fatalf("expected body {\"status\":\"ok\"}, got %s", body)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %s", ct)
	}
}

func TestHealthHandler_FailingDB(t *testing.T) {
	repo := repository.NewMockRepository()
	repo.InjectError = errors.New("connection refused")
	handler := NewHealthHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %s", ct)
	}

	var errResp models.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error != "database unavailable" {
		t.Fatalf("expected error 'database unavailable', got %q", errResp.Error)
	}
}
