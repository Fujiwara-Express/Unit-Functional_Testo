package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	InterHubTTL     = 86400 * time.Second // 24 hours
	CourierRouteTTL = 600 * time.Second   // 10 minutes
)

// CacheService wraps a Redis client with JSON serialization and graceful fallback.
type CacheService struct {
	client *redis.Client
}

// NewCacheService creates a new CacheService.
func NewCacheService(client *redis.Client) *CacheService {
	return &CacheService{client: client}
}

// Get retrieves a value from cache and deserializes it into dest.
// Returns false (not found) if the key is missing or Redis is unavailable.
func (c *CacheService) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	raw, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		log.Printf("[CacheService] Redis unavailable on get: %v", err)
		return false, nil
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return false, err
	}
	return true, nil
}

// Set serializes value to JSON and stores it with the given TTL.
// Logs a warning and returns nil if Redis is unavailable.
func (c *CacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		log.Printf("[CacheService] Redis unavailable on set: %v", err)
	}
	return nil
}

// InvalidatePattern deletes all keys matching the given glob pattern.
// Logs a warning and returns nil if Redis is unavailable.
func (c *CacheService) InvalidatePattern(ctx context.Context, pattern string) error {
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		log.Printf("[CacheService] Redis unavailable on keys scan: %v", err)
		return nil
	}
	if len(keys) > 0 {
		if err := c.client.Del(ctx, keys...).Err(); err != nil {
			log.Printf("[CacheService] Redis unavailable on del: %v", err)
		}
	}
	return nil
}

// ── Cache key helpers ────────────────────────────────────────────────────────

// InterHubKey returns the cache key for an inter-hub route.
func InterHubKey(origin, destination, serviceType string) string {
	return fmt.Sprintf("route:%s:%s:%s", origin, destination, serviceType)
}

// CourierRouteKey returns the cache key for a courier's daily route.
func CourierRouteKey(courierID, date string) string {
	return fmt.Sprintf("courier_route:%s:%s", courierID, date)
}
