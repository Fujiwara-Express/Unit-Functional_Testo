CREATE TABLE tracking_events (
    event_id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tracking_number   TEXT        NOT NULL,
    status            TEXT        NOT NULL,
    location          TEXT,
    hub_id            TEXT,
    notes             TEXT,
    created_by_service TEXT       NOT NULL DEFAULT 'tracking-service',
    timestamp         TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_tracking_events_tracking_number
    ON tracking_events (tracking_number);

CREATE TABLE tracking_summary (
    tracking_number    TEXT        PRIMARY KEY,
    current_status     TEXT        NOT NULL,
    last_location      TEXT,
    estimated_delivery TIMESTAMPTZ,
    updated_at         TIMESTAMPTZ NOT NULL
);
