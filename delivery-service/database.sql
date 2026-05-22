-- =============================================================
-- Delivery Service Database Schema
-- =============================================================

-- Create couriers table
CREATE TABLE IF NOT EXISTS couriers (
    courier_id   VARCHAR(64)     PRIMARY KEY,
    name         VARCHAR(255)    NOT NULL,
    phone        VARCHAR(32)     NOT NULL,
    hub_id       VARCHAR(64)     NOT NULL,
    vehicle_type VARCHAR(32)     NOT NULL,
    is_available BOOLEAN         NOT NULL DEFAULT TRUE,
    current_lat  DOUBLE PRECISION NOT NULL DEFAULT 0,
    current_lng  DOUBLE PRECISION NOT NULL DEFAULT 0
);

-- Create delivery_jobs table
CREATE TABLE IF NOT EXISTS delivery_jobs (
    job_id          VARCHAR(64)      PRIMARY KEY,
    tracking_number VARCHAR(64)      NOT NULL UNIQUE,
    courier_id      VARCHAR(64)      NOT NULL REFERENCES couriers(courier_id),
    hub_id          VARCHAR(64)      NOT NULL,
    status          VARCHAR(32)      NOT NULL CHECK (status IN ('ASSIGNED', 'OUT_FOR_DELIVERY', 'DELIVERED', 'FAILED', 'RETURNED')),
    attempt_count   INTEGER          NOT NULL DEFAULT 0,
    proof_photo_url TEXT             NOT NULL DEFAULT '',
    recipient_name  VARCHAR(255)     NOT NULL DEFAULT '',
    notes           TEXT             NOT NULL DEFAULT '',
    assigned_at     TIMESTAMPTZ      NOT NULL,
    completed_at    TIMESTAMPTZ      NULL
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_delivery_jobs_courier_id     ON delivery_jobs(courier_id);
CREATE INDEX IF NOT EXISTS idx_delivery_jobs_tracking_number ON delivery_jobs(tracking_number);
CREATE INDEX IF NOT EXISTS idx_couriers_hub_id              ON couriers(hub_id);

-- =============================================================
-- Test Database (used by functional tests)
-- =============================================================
-- Run the following to create the test database:
--
--   CREATE DATABASE delivery_test;
--
-- Then connect to delivery_test and run this entire file again.
-- =============================================================
