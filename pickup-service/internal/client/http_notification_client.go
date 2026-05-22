package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type httpNotificationClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPNotificationClient creates a new NotificationClient backed by HTTP.
func NewHTTPNotificationClient(baseURL string) NotificationClient {
	return &httpNotificationClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

type notifyEnRouteRequest struct {
	ContactName  string `json:"contact_name"`
	ContactPhone string `json:"contact_phone"`
	CourierID    string `json:"courier_id"`
}

// NotifyCourierEnRoute sends a notification that a courier is en route.
func (c *httpNotificationClient) NotifyCourierEnRoute(ctx context.Context, contactName, contactPhone, courierID string) error {
	payload := notifyEnRouteRequest{
		ContactName:  contactName,
		ContactPhone: contactPhone,
		CourierID:    courierID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/notifications/courier-en-route", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("notification service returned status: %d", resp.StatusCode)
	}
	return nil
}
