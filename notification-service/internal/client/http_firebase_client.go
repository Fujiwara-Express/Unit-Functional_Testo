package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/notification-service/internal/domain"
)

type httpFirebaseClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewFirebaseClient creates a new FirebaseClient backed by HTTP.
func NewFirebaseClient(baseURL, apiKey string) FirebaseClient {
	return &httpFirebaseClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

type firebasePushRequest struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

// SendPush sends a push notification via Firebase Cloud Messaging.
func (c *httpFirebaseClient) SendPush(ctx context.Context, userID, message string) error {
	payload := firebasePushRequest{UserID: userID, Message: message}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/fcm/send", bytes.NewReader(body))
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
		return fmt.Errorf("%w: firebase returned status %d", domain.ErrServiceUnavailable, resp.StatusCode)
	}
	return nil
}
