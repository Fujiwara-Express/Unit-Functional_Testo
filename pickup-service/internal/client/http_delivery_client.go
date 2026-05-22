package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pickup-service/internal/domain"
)

type httpDeliveryClient struct {
	baseURL    string
	httpClient *http.Client
}

// New creates a new DeliveryClient backed by an HTTP implementation.
func New(baseURL string) DeliveryClient {
	return &httpDeliveryClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// GetAvailableCouriers returns couriers available in the given city.
func (c *httpDeliveryClient) GetAvailableCouriers(ctx context.Context, cityCode string) ([]Courier, error) {
	url := fmt.Sprintf("%s/couriers?city_code=%s", c.baseURL, cityCode)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusServiceUnavailable {
		return nil, domain.ErrServiceUnavailable
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var couriers []Courier
	if err := json.NewDecoder(resp.Body).Decode(&couriers); err != nil {
		return nil, err
	}
	return couriers, nil
}
