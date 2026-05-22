package client

import "context"

// WhatsAppClient defines the interface for sending WHATSAPP notifications.
type WhatsAppClient interface {
	SendWhatsApp(ctx context.Context, phone, message string) error
}
