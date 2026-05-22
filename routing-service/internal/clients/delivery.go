package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"routing-service/internal/models"
)

// ErrUpstreamUnavailable is returned when the Delivery Service cannot be reached.
type ErrUpstreamUnavailable struct {
	Service string
	Cause   error
}

func (e *ErrUpstreamUnavailable) Error() string {
	return fmt.Sprintf("%s is unavailable: %v", e.Service, e.Cause)
}

// DeliveryServiceClient calls the external Delivery Service.
type DeliveryServiceClient struct {
	baseURL string
	http    *http.Client
}

// NewDeliveryServiceClient creates a new client with the given base URL and timeout.
func NewDeliveryServiceClient(baseURL string, timeout time.Duration) *DeliveryServiceClient {
	return &DeliveryServiceClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: timeout},
	}
}

// GetCourierDeliveryPoints fetches all active delivery points for a courier on a given date.
// Requirements: 4.1
func (c *DeliveryServiceClient) GetCourierDeliveryPoints(ctx context.Context, courierID, date string) ([]models.DeliveryPoint, error) {
	url := fmt.Sprintf("%s/couriers/%s/delivery-points?date=%s", c.baseURL, courierID, date)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, &ErrUpstreamUnavailable{Service: "Delivery Service", Cause: err}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &ErrUpstreamUnavailable{Service: "Delivery Service", Cause: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusServiceUnavailable {
		return nil, &ErrUpstreamUnavailable{Service: "Delivery Service", Cause: fmt.Errorf("HTTP 503")}
	}

	var points []models.DeliveryPoint
	if err := json.NewDecoder(resp.Body).Decode(&points); err != nil {
		return nil, err
	}
	return points, nil
}

// GetCourierHub fetches the hub assigned to a courier.
// Requirements: 4.1
func (c *DeliveryServiceClient) GetCourierHub(ctx context.Context, courierID string) (*models.HubOrigin, error) {
	url := fmt.Sprintf("%s/couriers/%s/hub", c.baseURL, courierID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, &ErrUpstreamUnavailable{Service: "Delivery Service", Cause: err}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &ErrUpstreamUnavailable{Service: "Delivery Service", Cause: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusServiceUnavailable {
		return nil, &ErrUpstreamUnavailable{Service: "Delivery Service", Cause: fmt.Errorf("HTTP 503")}
	}

	var hub models.HubOrigin
	if err := json.NewDecoder(resp.Body).Decode(&hub); err != nil {
		return nil, err
	}
	return &hub, nil
}
