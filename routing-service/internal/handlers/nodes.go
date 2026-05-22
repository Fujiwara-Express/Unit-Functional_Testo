package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"routing-service/internal/models"
	"routing-service/internal/repositories"
)

type nodeRepo interface {
	GetAllNodes(ctx context.Context) ([]models.RouteNode, error)
	CreateNode(ctx context.Context, data models.CreateNodeInput) (*models.RouteNode, error)
}

// NodesHandler handles /routing/nodes endpoints.
type NodesHandler struct {
	repo nodeRepo
}

// NewNodesHandler creates a new NodesHandler.
func NewNodesHandler(repo nodeRepo) *NodesHandler {
	return &NodesHandler{repo: repo}
}

// GetNodes handles GET /routing/nodes — Requirements: 2.1
func (h *NodesHandler) GetNodes(c *gin.Context) {
	nodes, err := h.repo.GetAllNodes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "INTERNAL_ERROR", "message": err.Error()})
		return
	}
	if nodes == nil {
		nodes = []models.RouteNode{}
	}
	c.JSON(http.StatusOK, nodes)
}

// CreateNode handles POST /routing/nodes — Requirements: 2.2, 2.3, 2.4
func (h *NodesHandler) CreateNode(c *gin.Context) {
	var input models.CreateNodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "VALIDATION_ERROR", "message": err.Error()})
		return
	}

	node, err := h.repo.CreateNode(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, repositories.ErrDuplicate) {
			c.JSON(http.StatusConflict, gin.H{"error": "DUPLICATE_HUB", "message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "INTERNAL_ERROR", "message": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":    "CREATED",
		"node_id":   node.NodeID,
		"hub_id":    node.HubID,
		"city_code": node.CityCode,
		"latitude":  node.Latitude,
		"longitude": node.Longitude,
	})
}
