-- migrations/001_init.sql
-- Initial schema for routing-service

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ── route_nodes ───────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS route_nodes (
    node_id   VARCHAR(36)   PRIMARY KEY DEFAULT gen_random_uuid()::text,
    hub_id    VARCHAR(50)   NOT NULL UNIQUE,
    city_code VARCHAR(10)   NOT NULL,
    latitude  DECIMAL(9,6)  NOT NULL,
    longitude DECIMAL(9,6)  NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ── route_edges ───────────────────────────────────────────────────────────────
CREATE TYPE transport_mode_enum AS ENUM ('DARAT', 'UDARA', 'LAUT');

CREATE TABLE IF NOT EXISTS route_edges (
    edge_id           VARCHAR(36)          PRIMARY KEY DEFAULT gen_random_uuid()::text,
    from_node_id      VARCHAR(36)          NOT NULL REFERENCES route_nodes(node_id),
    to_node_id        VARCHAR(36)          NOT NULL REFERENCES route_nodes(node_id),
    distance_km       DECIMAL(10,2)        NOT NULL CHECK (distance_km > 0),
    avg_transit_hours DECIMAL(6,2)         NOT NULL CHECK (avg_transit_hours > 0),
    transport_mode    transport_mode_enum  NOT NULL,
    is_active         BOOLEAN              NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ          NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ          NOT NULL DEFAULT NOW()
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_route_edges_from_node ON route_edges(from_node_id);
CREATE INDEX IF NOT EXISTS idx_route_edges_active    ON route_edges(is_active);

-- ── Seed data for local / functional tests ────────────────────────────────────
INSERT INTO route_nodes (node_id, hub_id, city_code, latitude, longitude) VALUES
    ('node-jkt', 'HUB_JKT', 'JKT', -6.200000,  106.816666),
    ('node-bdg', 'HUB_BDG', 'BDG', -6.917464,  107.619123),
    ('node-sby', 'HUB_SBY', 'SBY', -7.250445,  112.768845),
    ('node-mdn', 'HUB_MDN', 'MDN',  3.595196,   98.672226)
ON CONFLICT (hub_id) DO NOTHING;

INSERT INTO route_edges (edge_id, from_node_id, to_node_id, distance_km, avg_transit_hours, transport_mode, is_active) VALUES
    ('edge-jkt-bdg', 'node-jkt', 'node-bdg', 150.0,  3.0, 'DARAT', TRUE),
    ('edge-bdg-jkt', 'node-bdg', 'node-jkt', 150.0,  3.0, 'DARAT', TRUE),
    ('edge-jkt-sby', 'node-jkt', 'node-sby', 780.0, 12.0, 'DARAT', TRUE),
    ('edge-sby-jkt', 'node-sby', 'node-jkt', 780.0, 12.0, 'DARAT', TRUE),
    ('edge-bdg-sby', 'node-bdg', 'node-sby', 700.0, 11.0, 'DARAT', TRUE),
    ('edge-sby-bdg', 'node-sby', 'node-bdg', 700.0, 11.0, 'DARAT', TRUE),
    ('edge-jkt-mdn', 'node-jkt', 'node-mdn', 1400.0, 3.5, 'UDARA', TRUE),
    ('edge-mdn-jkt', 'node-mdn', 'node-jkt', 1400.0, 3.5, 'UDARA', TRUE)
ON CONFLICT (edge_id) DO NOTHING;
