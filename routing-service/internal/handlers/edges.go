package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"routing-service/internal/models"
	"routing-service/internal/repositories"
)

type edgeRepo interface {
	GetAllEdges(ctx context.Context, fromNodeID *string) ([]models.RouteEdge, error)
	CreateEdge(ctx context.Context, data models.CreateEdgeInput) (*models.RouteEdge, error)
	UpdateEdge(ctx context.Context, edgeID string, data models.UpdateEdgeInput) (*models.RouteEdge, error)
}

type cacheInvalidator interface {
	InvalidatePattern(ctx context.Context, pattern string) error
}

// EdgesHandler handles /routing/edges endpoints.
type EdgesHandler struct {
	repo  edgeRepo
	cache cacheInvalidator
}

// NewEdgesHandler creates a new EdgesHandler.
func NewEdgesHandler(repo edgeRepo, cache cacheInvalidator) *EdgesHandler {
	return &EdgesHandler{repo: repo, cache: cache}
}

// GetEdges handles GET /routing/edges — Requirements: 3.1
func (h *EdgesHandler) GetEdges(c *gin.Context) {
	var fromNodeID *string
	if v := c.Query("from_node_id"); v != "" {
		fromNodeID = &v
	}

	edges, err := h.repo.GetAllEdges(c.Request.Context(), fromNodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "INTERNAL_ERROR", "message": err.Error()})
		return
	}
	if edges == nil {
		edges = []models.RouteEdge{}
	}
	c.JSON(http.StatusOK, edges)
}

// CreateEdge handles POST /routing/edges — Requirements: 3.2, 3.5, 3.7
func (h *EdgesHandler) CreateEdge(c *gin.Context) {
	var input models.CreateEdgeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "VALIDATION_ERROR", "message": err.Error()})
		return
	}

	edge, err := h.repo.CreateEdge(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "NODE_NOT_FOUND", "message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "INTERNAL_ERROR", "message": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "CREATED", "edge_id": edge.EdgeID})
}

// UpdateEdge handles PATCH /routing/edges/:edge_id — Requirements: 3.3, 3.4, 3.6
func (h *EdgesHandler) UpdateEdge(c *gin.Context) {
	edgeID := c.Param("edge_id")

	var input models.UpdateEdgeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "VALIDATION_ERROR", "message": err.Error()})
		return
	}

	edge, err := h.repo.UpdateEdge(c.Request.Context(), edgeID, input)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "EDGE_NOT_FOUND", "message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "INTERNAL_ERROR", "message": err.Error()})
		return
	}

	// Invalidate all cached inter-hub routes — Requirements: 3.4
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = h.cache.InvalidatePattern(ctx, "route:*")

	c.JSON(http.StatusOK, gin.H{"status": "UPDATED", "edge_id": edge.EdgeID})
}
