package domain

import (
	"fmt"
	"strings"
	"time"
)

// Channel represents the delivery mechanism for a notification.
type Channel string

const (
	ChannelPush     Channel = "PUSH"
	ChannelEmail    Channel = "EMAIL"
	ChannelWhatsApp Channel = "WHATSAPP"
)

// Validate returns an error if the channel is not one of the known valid values.
func (c Channel) Validate() error {
	switch c {
	case ChannelPush, ChannelEmail, ChannelWhatsApp:
		return nil
	default:
		return fmt.Errorf("invalid channel: %s", c)
	}
}

// NotifStatus represents the delivery status of a notification attempt.
type NotifStatus string

const (
	NotifStatusSent    NotifStatus = "SENT"
	NotifStatusFailed  NotifStatus = "FAILED"
	NotifStatusPending NotifStatus = "PENDING"
)

// NotificationLog is a record of a notification attempt stored in notification_logs.
type NotificationLog struct {
	NotifID        string      `json:"notif_id"`
	UserID         string      `json:"user_id"`
	TrackingNumber string      `json:"tracking_number"`
	Channel        Channel     `json:"channel"`
	TemplateID     string      `json:"template_id"`
	Message        string      `json:"message"`
	Status         NotifStatus `json:"status"`
	SentAt         time.Time   `json:"sent_at"`
}

// NotificationTemplate is a reusable message template stored in notification_templates.
type NotificationTemplate struct {
	TemplateID   string  `json:"template_id"`
	EventType    string  `json:"event_type"`
	Channel      Channel `json:"channel"`
	Subject      string  `json:"subject"`
	BodyTemplate string  `json:"body_template"`
}

// RenderTemplate substitutes {{variable_name}} placeholders in bodyTemplate
// with values from the variables map. Unmatched placeholders are left unchanged.
func RenderTemplate(bodyTemplate string, variables map[string]string) string {
	result := bodyTemplate
	for k, v := range variables {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}
	return result
}
