package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"routing-service/internal/calculators"
	"routing-service/internal/models"
	"routing-service/internal/services"
)

type graphRepo interface {
	GetActiveGraph(ctx context.Context) (*models.Graph, error)
}

type routeCache interface {
	Get(ctx context.Context, key string, dest interface{}) (bool, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

// RouteHandler handles GET /routing/route.
type RouteHandler struct {
	repo  graphRepo
	cache routeCache
}

// NewRouteHandler creates a new RouteHandler.
func NewRouteHandler(repo graphRepo, cache routeCache) *RouteHandler {
	return &RouteHandler{repo: repo, cache: cache}
}

// routeQuery holds the validated query parameters for GET /routing/route.
type routeQuery struct {
	Origin      string `form:"origin" binding:"required"`
	Destination string `form:"destination" binding:"required"`
	ServiceType string `form:"service_type" binding:"required"`
}

// GetRoute handles GET /routing/route — Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 6.2, 6.3
func (h *RouteHandler) GetRoute(c *gin.Context) {
	var q routeQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "VALIDATION_ERROR",
			"message": err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	cacheKey := services.InterHubKey(q.Origin, q.Destination, q.ServiceType)

	// Requirements 1.3: check cache first
	var cached models.InterHubRoute
	hit, err := h.cache.Get(ctx, cacheKey, &cached)
	if err != nil {
		log.Printf("[RouteHandler] cache get error: %v", err)
	}
	if hit {
		c.JSON(http.StatusOK, cached)
		return
	}

	// Cache miss — load active graph from DB
	graph, err := h.repo.GetActiveGraph(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "INTERNAL_ERROR",
			"message": err.Error(),
		})
		return
	}

	// Requirements 1.2: run Dijkstra
	route, err := calculators.Dijkstra(graph, q.Origin, q.Destination)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "VALIDATION_ERROR",
			"message": err.Error(),
		})
		return
	}

	// Requirements 1.5: no path found
	if route == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "NO_ROUTE_FOUND",
			"message": "no active route exists between the requested origin and destination",
		})
		return
	}

	// Requirements 1.3 / 1.4: store in cache with 24h TTL (best-effort, Redis may be unavailable)
	if setErr := h.cache.Set(ctx, cacheKey, route, services.InterHubTTL); setErr != nil {
		log.Printf("[RouteHandler] cache set error: %v", setErr)
	}

	c.JSON(http.StatusOK, route)
}
