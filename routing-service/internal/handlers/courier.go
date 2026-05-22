package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"routing-service/internal/calculators"
	"routing-service/internal/clients"
	"routing-service/internal/models"
	"routing-service/internal/services"
)

// deliveryClient is the interface for fetching courier data from the Delivery Service.
type deliveryClient interface {
	GetCourierDeliveryPoints(ctx context.Context, courierID, date string) ([]models.DeliveryPoint, error)
	GetCourierHub(ctx context.Context, courierID string) (*models.HubOrigin, error)
}

// CourierHandler handles GET /routing/courier-route/:courier_id.
type CourierHandler struct {
	client deliveryClient
	cache  routeCache
}

// NewCourierHandler creates a new CourierHandler.
func NewCourierHandler(client deliveryClient, cache routeCache) *CourierHandler {
	return &CourierHandler{client: client, cache: cache}
}

// GetCourierRoute handles GET /routing/courier-route/:courier_id
// Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 4.7
func (h *CourierHandler) GetCourierRoute(c *gin.Context) {
	courierID := c.Param("courier_id")
	if courierID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "VALIDATION_ERROR",
			"message": "courier_id is required",
		})
		return
	}

	ctx := c.Request.Context()
	today := calculators.TodayDate()
	cacheKey := services.CourierRouteKey(courierID, today)

	// Requirements 4.4: check cache first
	var cached models.CourierRoute
	hit, err := h.cache.Get(ctx, cacheKey, &cached)
	if err != nil {
		log.Printf("[CourierHandler] cache get error: %v", err)
	}
	if hit {
		c.JSON(http.StatusOK, cached)
		return
	}

	// Cache miss — fetch from Delivery Service
	// Requirements 4.7: handle Delivery Service unavailable → 503
	points, err := h.client.GetCourierDeliveryPoints(ctx, courierID, today)
	if err != nil {
		var upstreamErr *clients.ErrUpstreamUnavailable
		if errors.As(err, &upstreamErr) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "UPSTREAM_UNAVAILABLE",
				"message": upstreamErr.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "INTERNAL_ERROR",
			"message": err.Error(),
		})
		return
	}

	hub, err := h.client.GetCourierHub(ctx, courierID)
	if err != nil {
		var upstreamErr *clients.ErrUpstreamUnavailable
		if errors.As(err, &upstreamErr) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "UPSTREAM_UNAVAILABLE",
				"message": upstreamErr.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "INTERNAL_ERROR",
			"message": err.Error(),
		})
		return
	}

	// Requirements 4.2, 4.3, 4.6: calculate optimized route
	route := calculators.CalculateCourierRoute(courierID, points, *hub, time.Now())

	// Requirements 4.4, 4.5: store in cache with 10-minute TTL (best-effort)
	if setErr := h.cache.Set(ctx, cacheKey, route, services.CourierRouteTTL); setErr != nil {
		log.Printf("[CourierHandler] cache set error: %v", setErr)
	}

	c.JSON(http.StatusOK, route)
}
