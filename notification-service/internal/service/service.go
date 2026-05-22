package service

import (
	"context"

	"github.com/notification-service/internal/domain"
)

// SendNotificationInput contains the input for sending a notification.
type SendNotificationInput struct {
	UserID     string
	Channel    domain.Channel
	TemplateID string
	Variables  map[string]string
}

// SendNotificationOutput contains the result of sending a notification.
type SendNotificationOutput struct {
	NotificationID string
	Status         domain.NotifStatus
	Channel        domain.Channel
}

// CreateTemplateInput contains the input for creating a notification template.
type CreateTemplateInput struct {
	EventType    string
	Channel      domain.Channel
	Subject      string
	BodyTemplate string
}

// CreateTemplateOutput contains the result of creating a template.
type CreateTemplateOutput struct {
	TemplateID string
	Status     string
}

// UpdateTemplateInput contains the input for updating a notification template.
type UpdateTemplateInput struct {
	Subject      string
	BodyTemplate string
}

// UpdateTemplateOutput contains the result of updating a template.
type UpdateTemplateOutput struct {
	TemplateID string
	Status     string
}

// NotificationService defines the business logic interface for notification operations.
type NotificationService interface {
	SendNotification(ctx context.Context, req SendNotificationInput) (*SendNotificationOutput, error)
	ListTemplates(ctx context.Context) ([]*domain.NotificationTemplate, error)
	CreateTemplate(ctx context.Context, req CreateTemplateInput) (*CreateTemplateOutput, error)
	UpdateTemplate(ctx context.Context, id string, req UpdateTemplateInput) (*UpdateTemplateOutput, error)
}
