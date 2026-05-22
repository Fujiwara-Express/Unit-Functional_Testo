package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/notification-service/internal/domain"
)

type httpSendGridClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewSendGridClient creates a new SendGridClient backed by HTTP.
func NewSendGridClient(baseURL, apiKey string) SendGridClient {
	return &httpSendGridClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

type sendGridEmailRequest struct {
	Recipient string `json:"recipient"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
}

// SendEmail sends an email notification via the SendGrid API.
func (c *httpSendGridClient) SendEmail(ctx context.Context, recipient, subject, body string) error {
	payload := sendGridEmailRequest{Recipient: recipient, Subject: subject, Body: body}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v3/mail/send", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: sendgrid returned status %d", domain.ErrServiceUnavailable, resp.StatusCode)
	}
	return nil
}
