-- +goose Up
-- A webhook delivery is one event for one webhook. It is tried at once and,
-- when the endpoint fails, again later with a growing wait, up to a cap
-- (internal/services/webhook). Every try is a row in webhook_logs.
--
-- status: pending (waiting for its next try at next_attempt_at), sending (a
-- try is in flight; next_attempt_at is when that claim lapses), succeeded, or
-- failed (out of tries).
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id TEXT PRIMARY KEY,
    webhook_id TEXT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event TEXT NOT NULL,
    payload TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TEXT,
    last_status_code INTEGER,
    last_error TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_due ON webhook_deliveries(status, next_attempt_at);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook ON webhook_deliveries(webhook_id, created_at);

-- Each try names its delivery and its number. Test sends have no delivery.
ALTER TABLE webhook_logs ADD COLUMN delivery_id TEXT REFERENCES webhook_deliveries(id) ON DELETE CASCADE;
ALTER TABLE webhook_logs ADD COLUMN attempt INTEGER;
CREATE INDEX IF NOT EXISTS idx_webhook_logs_delivery_id ON webhook_logs(delivery_id);

-- +goose Down
-- SQLite can't drop a column that references another table, so
-- webhook_logs keeps delivery_id and attempt; nothing reads them after this.
DROP INDEX IF EXISTS idx_webhook_logs_delivery_id;
DROP INDEX IF EXISTS idx_webhook_deliveries_webhook;
DROP INDEX IF EXISTS idx_webhook_deliveries_due;
DROP TABLE IF EXISTS webhook_deliveries;
