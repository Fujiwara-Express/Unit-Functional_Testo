-- Payment Service Database Schema
-- Run this script to initialize the database before running functional tests.

CREATE DATABASE payment_test;

\c payment_test;

-- payments table
CREATE TABLE IF NOT EXISTS payments (
    payment_id   VARCHAR(255)  PRIMARY KEY,
    order_id     VARCHAR(255)  NOT NULL UNIQUE,
    user_id      VARCHAR(255)  NOT NULL,
    amount       NUMERIC(15,2) NOT NULL,
    method       VARCHAR(50)   NOT NULL CHECK (method IN ('TRANSFER', 'VIRTUAL_ACCOUNT', 'QRIS', 'COD')),
    status       VARCHAR(50)   NOT NULL CHECK (status IN ('PENDING', 'SUCCESS', 'FAILED', 'REFUNDED')),
    external_ref VARCHAR(255)  NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- cod_collections table
CREATE TABLE IF NOT EXISTS cod_collections (
    collection_id     VARCHAR(255)  PRIMARY KEY,
    order_id          VARCHAR(255)  NOT NULL,
    courier_id        VARCHAR(255)  NOT NULL,
    amount_collected  NUMERIC(15,2) NOT NULL,
    collected_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    remittance_status VARCHAR(50)   NOT NULL CHECK (remittance_status IN ('PENDING', 'REMITTED'))
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_payments_order_id      ON payments (order_id);
CREATE INDEX IF NOT EXISTS idx_payments_external_ref  ON payments (external_ref);
CREATE INDEX IF NOT EXISTS idx_cod_collections_order_id ON cod_collections (order_id);
