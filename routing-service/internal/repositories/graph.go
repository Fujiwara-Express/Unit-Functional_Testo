package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"routing-service/internal/models"
)

// Sentinel errors
var (
	ErrDuplicate = errors.New("duplicate entry")
	ErrNotFound  = errors.New("not found")
)

// GraphRepository handles all PostgreSQL operations for route nodes and edges.
type GraphRepository struct {
	pool *pgxpool.Pool
}

// NewGraphRepository creates a new GraphRepository.
func NewGraphRepository(pool *pgxpool.Pool) *GraphRepository {
	return &GraphRepository{pool: pool}
}

// GetAllNodes returns all route nodes ordered by hub_id.
func (r *GraphRepository) GetAllNodes(ctx context.Context) ([]models.RouteNode, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT node_id, hub_id, city_code,
		        latitude::float8, longitude::float8
		 FROM route_nodes ORDER BY hub_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []models.RouteNode
	for rows.Next() {
		var n models.RouteNode
		if err := rows.Scan(&n.NodeID, &n.HubID, &n.CityCode, &n.Latitude, &n.Longitude); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

// CreateNode inserts a new route node and returns it.
// Returns ErrDuplicate if hub_id already exists.
func (r *GraphRepository) CreateNode(ctx context.Context, data models.CreateNodeInput) (*models.RouteNode, error) {
	nodeID := uuid.New().String()
	var n models.RouteNode
	err := r.pool.QueryRow(ctx,
		`INSERT INTO route_nodes (node_id, hub_id, city_code, latitude, longitude)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING node_id, hub_id, city_code, latitude::float8, longitude::float8`,
		nodeID, data.HubID, data.CityCode, data.Latitude, data.Longitude,
	).Scan(&n.NodeID, &n.HubID, &n.CityCode, &n.Latitude, &n.Longitude)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, fmt.Errorf("%w: hub_id '%s' already exists", ErrDuplicate, data.HubID)
		}
		return nil, err
	}
	return &n, nil
}

// GetAllEdges returns all route edges, optionally filtered by from_node_id.
func (r *GraphRepository) GetAllEdges(ctx context.Context, fromNodeID *string) ([]models.RouteEdge, error) {
	var rows pgx.Rows
	var err error

	if fromNodeID != nil {
		rows, err = r.pool.Query(ctx,
			`SELECT edge_id, from_node_id, to_node_id,
			        distance_km::float8, avg_transit_hours::float8,
			        transport_mode, is_active
			 FROM route_edges WHERE from_node_id = $1 ORDER BY edge_id`,
			*fromNodeID)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT edge_id, from_node_id, to_node_id,
			        distance_km::float8, avg_transit_hours::float8,
			        transport_mode, is_active
			 FROM route_edges ORDER BY edge_id`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []models.RouteEdge
	for rows.Next() {
		var e models.RouteEdge
		if err := rows.Scan(&e.EdgeID, &e.FromNodeID, &e.ToNodeID,
			&e.DistanceKm, &e.AvgTransitHours, &e.TransportMode, &e.IsActive); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	return edges, rows.Err()
}

// CreateEdge inserts a new route edge and returns it.
// Returns ErrNotFound if either referenced node does not exist.
func (r *GraphRepository) CreateEdge(ctx context.Context, data models.CreateEdgeInput) (*models.RouteEdge, error) {
	// Verify both nodes exist
	if err := r.assertNodeExists(ctx, data.FromNodeID); err != nil {
		return nil, err
	}
	if err := r.assertNodeExists(ctx, data.ToNodeID); err != nil {
		return nil, err
	}

	isActive := true
	if data.IsActive != nil {
		isActive = *data.IsActive
	}

	edgeID := uuid.New().String()
	var e models.RouteEdge
	err := r.pool.QueryRow(ctx,
		`INSERT INTO route_edges
		   (edge_id, from_node_id, to_node_id, distance_km, avg_transit_hours, transport_mode, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING edge_id, from_node_id, to_node_id,
		           distance_km::float8, avg_transit_hours::float8, transport_mode, is_active`,
		edgeID, data.FromNodeID, data.ToNodeID,
		data.DistanceKm, data.AvgTransitHours, data.TransportMode, isActive,
	).Scan(&e.EdgeID, &e.FromNodeID, &e.ToNodeID,
		&e.DistanceKm, &e.AvgTransitHours, &e.TransportMode, &e.IsActive)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// UpdateEdge applies a partial update to an existing edge and returns it.
// Returns ErrNotFound if the edge does not exist.
func (r *GraphRepository) UpdateEdge(ctx context.Context, edgeID string, data models.UpdateEdgeInput) (*models.RouteEdge, error) {
	// Build SET clause dynamically
	setClauses := []string{}
	args := []interface{}{}
	idx := 1

	if data.DistanceKm != nil {
		setClauses = append(setClauses, fmt.Sprintf("distance_km = $%d", idx))
		args = append(args, *data.DistanceKm)
		idx++
	}
	if data.AvgTransitHours != nil {
		setClauses = append(setClauses, fmt.Sprintf("avg_transit_hours = $%d", idx))
		args = append(args, *data.AvgTransitHours)
		idx++
	}
	if data.TransportMode != nil {
		setClauses = append(setClauses, fmt.Sprintf("transport_mode = $%d", idx))
		args = append(args, *data.TransportMode)
		idx++
	}
	if data.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", idx))
		args = append(args, *data.IsActive)
		idx++
	}

	if len(setClauses) == 0 {
		// Nothing to update — fetch current state
		return r.getEdgeByID(ctx, edgeID)
	}

	args = append(args, edgeID)
	query := fmt.Sprintf(
		`UPDATE route_edges SET %s WHERE edge_id = $%d
		 RETURNING edge_id, from_node_id, to_node_id,
		           distance_km::float8, avg_transit_hours::float8, transport_mode, is_active`,
		joinClauses(setClauses), idx,
	)

	var e models.RouteEdge
	err := r.pool.QueryRow(ctx, query, args...).
		Scan(&e.EdgeID, &e.FromNodeID, &e.ToNodeID,
			&e.DistanceKm, &e.AvgTransitHours, &e.TransportMode, &e.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: edge '%s'", ErrNotFound, edgeID)
	}
	return &e, err
}

// GetActiveGraph loads all nodes and active edges for use by Dijkstra.
func (r *GraphRepository) GetActiveGraph(ctx context.Context) (*models.Graph, error) {
	nodes, err := r.GetAllNodes(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT edge_id, from_node_id, to_node_id,
		        distance_km::float8, avg_transit_hours::float8,
		        transport_mode, is_active
		 FROM route_edges WHERE is_active = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []models.RouteEdge
	for rows.Next() {
		var e models.RouteEdge
		if err := rows.Scan(&e.EdgeID, &e.FromNodeID, &e.ToNodeID,
			&e.DistanceKm, &e.AvgTransitHours, &e.TransportMode, &e.IsActive); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	nodeMap := make(map[string]*models.RouteNode, len(nodes))
	for i := range nodes {
		nodeMap[nodes[i].HubID] = &nodes[i]
	}

	return &models.Graph{Nodes: nodeMap, Edges: edges}, nil
}

// ── private helpers ──────────────────────────────────────────────────────────

func (r *GraphRepository) assertNodeExists(ctx context.Context, nodeID string) error {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM route_nodes WHERE node_id = $1)`, nodeID,
	).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w: node '%s'", ErrNotFound, nodeID)
	}
	return nil
}

func (r *GraphRepository) getEdgeByID(ctx context.Context, edgeID string) (*models.RouteEdge, error) {
	var e models.RouteEdge
	err := r.pool.QueryRow(ctx,
		`SELECT edge_id, from_node_id, to_node_id,
		        distance_km::float8, avg_transit_hours::float8, transport_mode, is_active
		 FROM route_edges WHERE edge_id = $1`, edgeID,
	).Scan(&e.EdgeID, &e.FromNodeID, &e.ToNodeID,
		&e.DistanceKm, &e.AvgTransitHours, &e.TransportMode, &e.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: edge '%s'", ErrNotFound, edgeID)
	}
	return &e, err
}

func joinClauses(clauses []string) string {
	result := ""
	for i, c := range clauses {
		if i > 0 {
			result += ", "
		}
		result += c
	}
	return result
}
