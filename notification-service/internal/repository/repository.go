package repository

import (
	"context"
	"time"

	"github.com/notification-service/internal/domain"
)

// NotificationRepository defines the interface for notification data operations.
type NotificationRepository interface {
	CreateNotificationLog(ctx context.Context, log *domain.NotificationLog) (string, error)
	GetNotificationLogByID(ctx context.Context, id string) (*domain.NotificationLog, error)
	UpdateNotificationLogStatus(ctx context.Context, id string, status domain.NotifStatus, sentAt time.Time) error
	ListTemplates(ctx context.Context) ([]*domain.NotificationTemplate, error)
	GetTemplateByID(ctx context.Context, id string) (*domain.NotificationTemplate, error)
	CreateTemplate(ctx context.Context, t *domain.NotificationTemplate) (string, error)
	UpdateTemplate(ctx context.Context, id, subject, bodyTemplate string) error
}
