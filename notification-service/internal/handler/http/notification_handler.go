package httphandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/notification-service/internal/domain"
	"github.com/notification-service/internal/handler/http/middleware"
	"github.com/notification-service/internal/service"
)

// NotificationHandler handles HTTP requests for notification operations.
type NotificationHandler struct {
	svc service.NotificationService
}

// NewNotificationHandler creates a new NotificationHandler.
func NewNotificationHandler(svc service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

// writeError writes a standard error response.
func writeError(w http.ResponseWriter, r *http.Request, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Code:      code,
		Message:   message,
		RequestID: middleware.GetRequestID(r.Context()),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// mapError maps a domain error to an HTTP status code and error code string.
func mapError(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "NOT_FOUND"
	case errors.Is(err, domain.ErrValidation):
		return http.StatusBadRequest, "VALIDATION_ERROR"
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, "CONFLICT"
	case errors.Is(err, domain.ErrServiceUnavailable):
		return http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR"
	}
}

// sendNotificationBody is the JSON body for POST /notifications/send.
type sendNotificationBody struct {
	UserID     string            `json:"user_id"`
	Channel    string            `json:"channel"`
	TemplateID string            `json:"template_id"`
	Variables  map[string]string `json:"variables"`
}

// SendNotification handles POST /notifications/send
func (h *NotificationHandler) SendNotification(w http.ResponseWriter, r *http.Request) {
	var body sendNotificationBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, "BAD_REQUEST", "invalid request body", http.StatusBadRequest)
		return
	}

	switch {
	case body.UserID == "":
		writeError(w, r, "VALIDATION_ERROR", "missing required field user_id", http.StatusBadRequest)
		return
	case body.Channel == "":
		writeError(w, r, "VALIDATION_ERROR", "missing required field channel", http.StatusBadRequest)
		return
	case body.TemplateID == "":
		writeError(w, r, "VALIDATION_ERROR", "missing required field template_id", http.StatusBadRequest)
		return
	}

	ch := domain.Channel(body.Channel)
	if err := ch.Validate(); err != nil {
		writeError(w, r, "VALIDATION_ERROR", err.Error(), http.StatusBadRequest)
		return
	}

	out, err := h.svc.SendNotification(r.Context(), service.SendNotificationInput{
		UserID:     body.UserID,
		Channel:    ch,
		TemplateID: body.TemplateID,
		Variables:  body.Variables,
	})
	if err != nil {
		status, code := mapError(err)
		writeError(w, r, code, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"notification_id": out.NotificationID,
		"status":          string(out.Status),
		"channel":         string(out.Channel),
	})
}

// ListTemplates handles GET /notifications/templates
func (h *NotificationHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.svc.ListTemplates(r.Context())
	if err != nil {
		status, code := mapError(err)
		writeError(w, r, code, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(templates)
}

// createTemplateBody is the JSON body for POST /notifications/templates.
type createTemplateBody struct {
	EventType    string `json:"event_type"`
	Channel      string `json:"channel"`
	Subject      string `json:"subject"`
	BodyTemplate string `json:"body_template"`
}

// CreateTemplate handles POST /notifications/templates
func (h *NotificationHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var body createTemplateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, "BAD_REQUEST", "invalid request body", http.StatusBadRequest)
		return
	}

	switch {
	case body.EventType == "":
		writeError(w, r, "VALIDATION_ERROR", "missing required field event_type", http.StatusBadRequest)
		return
	case body.Channel == "":
		writeError(w, r, "VALIDATION_ERROR", "missing required field channel", http.StatusBadRequest)
		return
	case body.Subject == "":
		writeError(w, r, "VALIDATION_ERROR", "missing required field subject", http.StatusBadRequest)
		return
	case body.BodyTemplate == "":
		writeError(w, r, "VALIDATION_ERROR", "missing required field body_template", http.StatusBadRequest)
		return
	}

	out, err := h.svc.CreateTemplate(r.Context(), service.CreateTemplateInput{
		EventType:    body.EventType,
		Channel:      domain.Channel(body.Channel),
		Subject:      body.Subject,
		BodyTemplate: body.BodyTemplate,
	})
	if err != nil {
		status, code := mapError(err)
		writeError(w, r, code, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"template_id": out.TemplateID,
		"status":      out.Status,
	})
}

// updateTemplateBody is the JSON body for PUT /notifications/templates/{template_id}.
type updateTemplateBody struct {
	Subject      string `json:"subject"`
	BodyTemplate string `json:"body_template"`
}

// UpdateTemplate handles PUT /notifications/templates/{template_id}
func (h *NotificationHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	templateID := r.PathValue("template_id")
	if templateID == "" {
		writeError(w, r, "BAD_REQUEST", "missing template_id", http.StatusBadRequest)
		return
	}

	var body updateTemplateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, "BAD_REQUEST", "invalid request body", http.StatusBadRequest)
		return
	}

	out, err := h.svc.UpdateTemplate(r.Context(), templateID, service.UpdateTemplateInput{
		Subject:      body.Subject,
		BodyTemplate: body.BodyTemplate,
	})
	if err != nil {
		status, code := mapError(err)
		writeError(w, r, code, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"template_id": out.TemplateID,
		"status":      out.Status,
	})
}
