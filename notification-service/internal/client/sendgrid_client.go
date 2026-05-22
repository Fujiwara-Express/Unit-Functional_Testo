package client

import "context"

// SendGridClient defines the interface for sending EMAIL notifications via SendGrid.
type SendGridClient interface {
	SendEmail(ctx context.Context, recipient, subject, body string) error
}
