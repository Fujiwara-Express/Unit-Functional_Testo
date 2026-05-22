package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/notification-service/internal/domain"
)

type notificationRepository struct {
	db *sql.DB
}

// NewNotificationRepository creates a new NotificationRepository backed by *sql.DB.
func NewNotificationRepository(db *sql.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) CreateNotificationLog(ctx context.Context, log *domain.NotificationLog) (string, error) {
	query := `INSERT INTO notification_logs (notif_id, user_id, tracking_number, channel, template_id, message, status, sent_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING notif_id`
	row := r.db.QueryRowContext(ctx, query,
		log.NotifID, log.UserID, log.TrackingNumber, string(log.Channel),
		log.TemplateID, log.Message, string(log.Status), log.SentAt,
	)
	var notifID string
	if err := row.Scan(&notifID); err != nil {
		return "", fmt.Errorf("CreateNotificationLog: %w", err)
	}
	return notifID, nil
}

func (r *notificationRepository) GetNotificationLogByID(ctx context.Context, id string) (*domain.NotificationLog, error) {
	query := `SELECT notif_id, user_id, tracking_number, channel, template_id, message, status, sent_at FROM notification_logs WHERE notif_id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	l := &domain.NotificationLog{}
	var channel, status string
	err := row.Scan(&l.NotifID, &l.UserID, &l.TrackingNumber, &channel, &l.TemplateID, &l.Message, &status, &l.SentAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: notification log %s", domain.ErrNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("GetNotificationLogByID: %w", err)
	}
	l.Channel = domain.Channel(channel)
	l.Status = domain.NotifStatus(status)
	return l, nil
}

func (r *notificationRepository) UpdateNotificationLogStatus(ctx context.Context, id string, status domain.NotifStatus, sentAt time.Time) error {
	query := `UPDATE notification_logs SET status = $1, sent_at = $2 WHERE notif_id = $3`
	_, err := r.db.ExecContext(ctx, query, string(status), sentAt, id)
	if err != nil {
		return fmt.Errorf("UpdateNotificationLogStatus: %w", err)
	}
	return nil
}

func (r *notificationRepository) ListTemplates(ctx context.Context) ([]*domain.NotificationTemplate, error) {
	query := `SELECT template_id, event_type, channel, subject, body_template FROM notification_templates`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ListTemplates: %w", err)
	}
	defer rows.Close()

	var templates []*domain.NotificationTemplate
	for rows.Next() {
		t := &domain.NotificationTemplate{}
		var channel string
		if err := rows.Scan(&t.TemplateID, &t.EventType, &channel, &t.Subject, &t.BodyTemplate); err != nil {
			return nil, fmt.Errorf("ListTemplates scan: %w", err)
		}
		t.Channel = domain.Channel(channel)
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

func (r *notificationRepository) GetTemplateByID(ctx context.Context, id string) (*domain.NotificationTemplate, error) {
	query := `SELECT template_id, event_type, channel, subject, body_template FROM notification_templates WHERE template_id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	t := &domain.NotificationTemplate{}
	var channel string
	err := row.Scan(&t.TemplateID, &t.EventType, &channel, &t.Subject, &t.BodyTemplate)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: template %s", domain.ErrNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("GetTemplateByID: %w", err)
	}
	t.Channel = domain.Channel(channel)
	return t, nil
}

func (r *notificationRepository) CreateTemplate(ctx context.Context, t *domain.NotificationTemplate) (string, error) {
	query := `INSERT INTO notification_templates (template_id, event_type, channel, subject, body_template) VALUES ($1, $2, $3, $4, $5) RETURNING template_id`
	row := r.db.QueryRowContext(ctx, query, t.TemplateID, t.EventType, string(t.Channel), t.Subject, t.BodyTemplate)
	var templateID string
	if err := row.Scan(&templateID); err != nil {
		return "", fmt.Errorf("CreateTemplate: %w", err)
	}
	return templateID, nil
}

func (r *notificationRepository) UpdateTemplate(ctx context.Context, id, subject, bodyTemplate string) error {
	query := `UPDATE notification_templates SET subject = $1, body_template = $2 WHERE template_id = $3`
	_, err := r.db.ExecContext(ctx, query, subject, bodyTemplate, id)
	if err != nil {
		return fmt.Errorf("UpdateTemplate: %w", err)
	}
	return nil
}
