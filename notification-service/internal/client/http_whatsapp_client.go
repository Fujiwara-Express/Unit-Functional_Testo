package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/notification-service/internal/domain"
)

type httpWhatsAppClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewWhatsAppClient creates a new WhatsAppClient backed by HTTP.
func NewWhatsAppClient(baseURL, apiKey string) WhatsAppClient {
	return &httpWhatsAppClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

type whatsAppRequest struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

// SendWhatsApp sends a WhatsApp notification via the WhatsApp Business API.
func (c *httpWhatsAppClient) SendWhatsApp(ctx context.Context, phone, message string) error {
	payload := whatsAppRequest{Phone: phone, Message: message}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(data))
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
		return fmt.Errorf("%w: whatsapp returned status %d", domain.ErrServiceUnavailable, resp.StatusCode)
	}
	return nil
}
